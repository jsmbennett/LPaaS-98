# Protocol

The vLAN relay's WebSocket frame format. Phase 6 of `PLAN.md` implements this; everything that follows depends on it being stable.

## Goals

- Carry opaque game packets between browsers in a room with minimal overhead
- Support the three network models (`broadcast`, `host-client`, `custom`) without the relay knowing anything about game protocols
- Be debuggable enough that we can diagnose problems with browser devtools and `wscat`
- Be evolvable — add new control messages without breaking old clients
- Avoid dependencies on either side (no protobuf, no MessagePack, no custom binary framing libraries)

## Two channels on one socket

Each room WebSocket carries two kinds of messages, distinguished by WebSocket's native opcode:

- **Text frames (opcode 0x1) carry JSON control messages.** Join, leave, errors, room state changes. Low frequency, readable, easy to extend.
- **Binary frames (opcode 0x2) carry opaque game data.** High frequency, zero envelope overhead, the relay forwards the bytes unmodified.

Splitting them this way means the data path has *no* per-packet parsing cost on the server — the relay reads a binary frame, looks up the room's network model, and forwards bytes to the right peers. JSON is only parsed on the much rarer control path.

## Transport abstraction (forward-looking)

WebSocket is the only transport implemented in v1, but the relay is designed against an interface that hides it. Concretely, `internal/vlan` exposes something like:

```go
// Transport is the surface the relay sees. v1 has one implementation
// (WebSocket); future versions may add WebTransport.
type Transport interface {
    // SendControl writes a JSON control message to the peer.
    SendControl(msg ControlMessage) error
    // SendData writes opaque game-packet bytes to the peer. May be
    // unreliable depending on transport; relay code treats it as
    // best-effort regardless.
    SendData(data []byte) error
    // Recv returns the next message from the peer, blocking until one
    // arrives or the transport closes.
    Recv() (Message, error)
    // Close terminates the transport.
    Close() error
}
```

Why this matters: WebSocket runs over TCP, which means delayed packets cause head-of-line blocking — a problem for vintage FPS games over real internet links, even though it's invisible on LAN. The eventual VPS-hosting mode (post-v1) will want **WebTransport** (HTTP/3 over QUIC), which gives both reliable and unreliable streams natively in modern browsers. Vintage game protocols expect unreliable UDP, and WebTransport's `datagrams` API matches that shape exactly.

Don't build WebTransport now. Do build the interface so it can slot in later. Per-game shims on the browser side should also be written against a small adapter ("here's how to send a packet, here's how to receive") so swapping transports doesn't require changes to every game's shim.

## Connection lifecycle

```
Client                                          Server
  │                                                │
  │  HTTP GET /ws/room/:id?nickname=alice          │
  │  Upgrade: websocket                            │
  ├───────────────────────────────────────────────►│
  │                                                │
  │       101 Switching Protocols                  │
  │◄───────────────────────────────────────────────┤
  │                                                │
  │       {"type":"hello","you":"alice",           │
  │        "peer_id":3,"is_host":false,            │
  │        "room":{...}}                           │
  │◄───────────────────────────────────────────────┤  (JSON)
  │                                                │
  │       {"type":"peer_joined","peer_id":3,       │
  │        "nickname":"alice"}                     │
  │     (sent to other peers in the room)          │
  │                                                │
  │  <binary game packet bytes>                    │
  ├───────────────────────────────────────────────►│  (binary)
  │                                                │
  │  <forwarded binary game packet bytes>          │
  │◄───────────────────────────────────────────────┤  (binary)
  │                                                │
  │       {"type":"peer_left","peer_id":2}         │
  │◄───────────────────────────────────────────────┤  (JSON, broadcast)
  │                                                │
  │  (close)                                       │
  ├───────────────────────────────────────────────►│
```

The server sends a `hello` message immediately on connect with the client's assigned `peer_id` and the current room state. After that, control traffic is asynchronous: either side sends control messages whenever it has something to say.

## Peer IDs

The relay assigns each connected client a `peer_id`: a small non-negative integer, unique within the room, allocated in join order. `peer_id` 0 is reserved (unassigned). The first joiner gets `peer_id` 1, the next gets 2, and so on. IDs are not reused within a room's lifetime — if peer 2 disconnects, the next joiner gets 5, not 2.

`peer_id` exists so binary game packets can be addressed to specific peers without re-encoding nicknames or browser session tokens into the high-frequency path.

In `host-client` rooms, the host's peer ID is whichever ID was assigned to the first joiner. It does not change if other peers come and go. If the host disconnects, the room ends (v1 behavior — host migration is post-v1).

## Control messages (JSON)

All control messages are JSON objects with a `type` field. Unknown `type` values are ignored (forward-compatibility). Unknown fields within a known type are ignored. Required fields below; optional fields may be added in later versions.

### Server → client

**`hello`** — sent once, immediately after upgrade.
```json
{
  "type": "hello",
  "you": "alice",
  "peer_id": 3,
  "is_host": false,
  "room": {
    "id": "r-1a2b3c",
    "game_id": "quakespasm",
    "network_model": "broadcast",
    "peers": [
      { "peer_id": 1, "nickname": "bob", "is_host": true },
      { "peer_id": 2, "nickname": "carol", "is_host": false }
    ],
    "max_players": 4
  }
}
```

**`peer_joined`** — broadcast to existing peers when someone new joins.
```json
{ "type": "peer_joined", "peer_id": 4, "nickname": "dave", "is_host": false }
```

**`peer_left`** — broadcast when someone disconnects.
```json
{ "type": "peer_left", "peer_id": 4 }
```

**`room_closing`** — sent to all peers before the server tears down the room (e.g. host left, server shutting down). Followed by the WebSocket close.
```json
{ "type": "room_closing", "reason": "host_left" }
```

Defined reasons: `host_left`, `idle_timeout`, `server_shutdown`, `kicked`. Unknown reasons should be treated as generic disconnect.

**`error`** — sent on a recoverable protocol or relay error. Non-recoverable errors close the socket with a WebSocket close code (see below).
```json
{ "type": "error", "code": "rate_limit", "message": "sending too fast" }
```

### Client → server

**`ready`** — optional. The shim sends this when its WASM game has finished loading and is ready to exchange game packets. The server uses it to gate room-start in games that need a synchronized start.
```json
{ "type": "ready" }
```

**`ping`** — keepalive. The server echoes back a `pong` with the same `id`. Used to measure RTT; not required (WebSocket has its own ping/pong frames at the protocol layer).
```json
{ "type": "ping", "id": 42 }
```

### Server → client (responses)

**`pong`**
```json
{ "type": "pong", "id": 42 }
```

## Data messages (binary)

Binary WebSocket frames carry opaque game packets. The relay's behavior depends on the room's network model.

### `broadcast` model

The simplest case. No envelope at all — the bytes of the WebSocket binary frame *are* the game packet. The server forwards each received frame, unmodified, to every other peer in the room. The sender never receives its own packets back.

This is what Doom expects: IPX-style "every node sees every packet."

### `host-client` model

Packets are addressed. There is a one-byte prefix indicating destination:

| First byte | Meaning |
|---|---|
| `0x00` | To host (only meaningful from a non-host) |
| `0x01`–`0xFE` | To peer with that `peer_id` |
| `0xFF` | Broadcast to all peers except sender |

The remaining bytes of the frame are the game packet itself, forwarded unmodified.

The host is the only peer allowed to address specific clients (`0x01`–`0xFE`) or to broadcast (`0xFF`). Non-host clients that send anything other than `0x00`-prefixed frames are sent an `error` control message and have their data frame dropped. Repeated violations cause disconnection.

This caps `peer_id` at 254 in `host-client` rooms — far above any realistic LAN party size.

### `custom` model

The per-game adapter decides. The Go adapter receives raw binary frames with the sender's `peer_id` and decides where they go. v1 doesn't ship any custom-model games, but the hook exists so a future game (e.g. OpenTTD with its tick-locked client-server model) can be added without changing the relay protocol.

## Why the data plane has minimal envelope

Game packets are sent at 30-60 Hz per peer, sometimes with 4-16 peers in a room. That's potentially thousands of packets per second through the relay. The fewer bytes we add per packet, the better — especially over Wi-Fi at a LAN party where the AP is the bottleneck.

- Broadcast model: zero envelope. WebSocket frame bytes == game packet bytes.
- Host-client model: one byte of envelope. Worst case 6.7% overhead on a minimum 14-byte UDP-equivalent packet, but typically <1% on real game packets.

JSON-wrapped data packets would have run 30-50% overhead from base64 encoding plus envelope. That's the whole reason for the text/binary split.

## Error handling

### Recoverable errors (control plane)
Sent as `{ "type": "error", "code": "...", "message": "..." }`. The client should display or log the error but the connection continues. Codes:

- `rate_limit` — peer is sending too many bytes/packets per second. The relay throttles or temporarily blocks the offending peer.
- `bad_address` — host-client model, peer sent a malformed addressed packet.
- `unauthorized_broadcast` — host-client model, non-host attempted to broadcast.
- `unknown_control_type` — a control message with an unrecognized `type` was received. Informational only; receivers should ignore unknown types but this code lets clients debug their own bugs.

### Non-recoverable errors (connection close)
The server closes the WebSocket with a close code. Standard WebSocket close codes are used where applicable; custom codes are in the 4000-4999 range as specified by RFC 6455.

| Code | Meaning |
|---|---|
| 1000 | Normal closure (client or server initiated cleanly) |
| 1011 | Server internal error |
| 4000 | Room full |
| 4001 | Room not found |
| 4002 | Nickname rejected (collision could not be resolved, or invalid characters) |
| 4003 | Game not installed on this server |
| 4004 | Game has missing required assets — not launchable |
| 4005 | Banned for repeated protocol violations |

Close codes include a human-readable reason string in the close frame's payload.

## Rate limits and backpressure

Each peer is limited to:
- **Data plane:** 256 KB/sec inbound, 1024 packets/sec inbound. Configurable per-game in the adapter (Quake at 60 Hz wants more headroom than Doom at 35 Hz).
- **Control plane:** 100 messages/sec. Bursts allowed, sustained traffic throttled.

Outbound, the server uses WebSocket-level backpressure: if a peer's send buffer exceeds a threshold (default 1 MB), the relay drops further packets to that peer until it drains. The peer receives an `error` with code `rate_limit` (slow consumer). If the buffer stays saturated for >5 seconds, the server closes the connection with 1011.

This is the only way to keep one slow client from delaying the whole room.

## Frame size limits

- Maximum WebSocket frame size: 64 KiB. Vintage game packets are far smaller (Doom: ~14-128 bytes; Quake: usually <1500 bytes), so this is generous.
- Maximum control message size: 16 KiB. JSON shouldn't approach this; if it does, something's wrong.

Frames exceeding the limit are dropped and the peer receives an `error` (data plane) or are kicked with close 4005 (control plane, since exceeding control-message limits implies bug or attack).

## Sequence numbers and reliability

The protocol does not provide sequence numbers, reliability, or ordering beyond what the transport already guarantees. Vintage games handle their own packet loss and ordering at the application layer (Doom's networking expects unreliable transport and reorders/drops freely; Quake's NetQuake protocol has reliable + unreliable channels internally).

The relay's contract is: bytes that arrive are forwarded to each destination. Bytes may not arrive (peer disconnected mid-stream, or unreliable transport dropped them) — game code handles that. Bytes are not duplicated. Bytes from the same sender are not reordered relative to each other on a single reliable channel; the relay does not impose ordering across channels or across senders.

**Caveat about TCP head-of-line blocking** (WebSocket-specific): TCP retransmits lost packets in order, which means a delayed packet on a high-latency connection freezes the room until the retransmission arrives. This is invisible on LAN but painful over real internet. Games designed for UDP (Doom's broadcast, Quake's NetQuake) expect to drop and move on, not to wait. WebTransport will fix this when added; for v1 / LAN-only use, accept it.

## Versioning

The protocol version is implicit in the server version. Breaking changes bump the server's major version. Within a major version, additions are backward-compatible: new control message types, new fields in existing messages, new close codes, new error codes.

Clients indicate their understanding via the WebSocket upgrade query string:

```
GET /ws/room/r-1a2b3c?nickname=alice&shim_version=2
```

The `shim_version` is the integer protocol version the JS shim was built against. The server can use this to maintain compatibility shims if the protocol ever evolves incompatibly within a server major version (last-resort escape hatch — prefer additive changes).

If `shim_version` is missing, the server assumes 1.

## Reference: minimal client pseudocode

```js
const ws = new WebSocket("ws://192.168.1.47:9898/ws/room/r-1a2b3c?nickname=alice&shim_version=1");
ws.binaryType = "arraybuffer";

ws.onmessage = (e) => {
  if (typeof e.data === "string") {
    const msg = JSON.parse(e.data);
    handleControl(msg);
  } else {
    // Raw game packet — hand to the shim's fake network layer.
    deliverToGame(new Uint8Array(e.data));
  }
};

function sendGamePacket(bytes, opts) {
  // For broadcast model: just send the bytes.
  // For host-client model: prepend addressing byte.
  if (networkModel === "host-client") {
    const framed = new Uint8Array(bytes.length + 1);
    framed[0] = opts?.toHost ? 0x00 : opts.peerId;
    framed.set(bytes, 1);
    ws.send(framed);
  } else {
    ws.send(bytes);
  }
}
```

## Open questions

These should be settled by the time Phase 6 ships:

1. **JSON parser** — stdlib `encoding/json` is fine for the control plane. No need for jsoniter or sonic unless profiling proves otherwise.
2. **Buffer pool for the data plane** — small per-room sync.Pool of `[]byte` to avoid GC pressure from high-frequency forwarding. Build the simple version first, profile, decide.
3. **Whether to expose `shim_version` to the per-game adapter** — could let an adapter behave differently for old shims, but adds complexity. Probably defer until needed.
