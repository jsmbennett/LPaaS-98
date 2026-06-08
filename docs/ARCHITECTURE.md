# Architecture

## Goals & Non-Goals

**Goals (v1)**
- Single statically-linked Go binary, runs on a laptop at a LAN party
- Cross-platform: Linux, macOS, Windows on amd64 and arm64
- Host installs games via a built-in catalog command; precompiled WASM builds are downloaded from a separate GitHub-hosted games repository, verified by SHA256
- Supports three game-architecture patterns cleanly: self-contained pure-WASM games, engine-only pure-WASM games with host-supplied or free-bundle assets, and native-server-subprocess games with WASM clients
- Ships with three launch games: OpenArena (pure-WASM), Zandronum (subprocess server + WASM client), and Quakespasm (pure-WASM)
- Browser-to-browser game traffic flows through a WebSocket vLAN relay; browser-to-subprocess-server traffic flows through a WebSocket-to-UDP proxy
- Guests open a browser, type the host's local IP, pick a nickname, see installed games on a Win98-styled desktop, join a room, play
- A "virtual LAN" relay so vintage games that expect broadcast/UDP networking can talk to each other through the server
- Works offline after games are installed

**Non-goals (v1)**
- TLS / HTTPS — plain HTTP on the LAN only
- User accounts, persistence, leaderboards
- Bundling or silently substituting commercial assets — every asset source is an explicit choice the host makes
- Polished UI (Win98 styling is in scope; the CRT-glow polish is post-v1)
- VPS / public-internet hosting (designed for, not built for)

## Two-Repository Structure

LPaaS 98 lives in two repos:

1. **`lpaas-98`** — the server itself (this repo). Go code, frontend, docs.
2. **`lpaas-98-games`** — a separate repo that hosts the catalog and the precompiled WASM game builds as GitHub release artifacts. Maintained by the project author for trust/control; forkable by anyone who wants to run their own catalog.

The server has a default catalog URL pointing at `lpaas-98-games`, overridable with `--catalog-url`. This separation lets the server stay small, lets game builds be updated independently, and keeps licensing concerns (GPL engines, free asset bundles, OpenArena's GPL game data) cleanly isolated in the games repo.

## Three Game-Architecture Patterns

Games in the catalog fall into one of three architectural patterns. The server treats all three as first-class; each launch game demonstrates one.

**Pattern A — Pure-WASM, self-contained** (e.g. OpenArena)
- The game engine is compiled to WASM. The game data ships in the catalog archive alongside the engine.
- Multiple browser clients connect to the LPaaS 98 server via WebSocket.
- The server's **vLAN relay** forwards game packets between clients (per the game's `network_model`: `broadcast`, `host-client`, or `custom`).
- The game inside each browser thinks it's on a local network. The browser-side JS shim does the translation.
- No native server process. No host-supplied assets.

**Pattern B — Pure-WASM, engine-only** (e.g. Quakespasm)
- The engine is compiled to WASM. Game data lives separately in `./assets/<game-id>/`, supplied by the host or installed via `install-free-assets`.
- Otherwise identical to Pattern A: vLAN relay between browsers, JS shim translates between game's network calls and WebSocket frames.

**Pattern C — Native server subprocess + WASM client + UDP proxy** (e.g. Zandronum)
- The catalog archive contains both a WASM build of the game *client* AND a per-platform native game *server* binary (Linux/macOS/Windows × amd64/arm64).
- When a room is created, LPaaS 98 spawns the native server as a subprocess, binding it to a local UDP port.
- Browser clients connect to LPaaS 98 via WebSocket. The server acts as a **UDP proxy**: bytes from the browser WebSocket go out as UDP to the subprocess; UDP responses from the subprocess go back over the WebSocket.
- The native game server thinks it's talking to native clients on localhost. The WASM client thinks it's talking to a normal game server. Neither knows about LPaaS 98.
- Host-supplied or free-bundle assets work the same way as Pattern B.
- Inspired by [dorch / gib.gg](https://github.com/beebs-dev/dorch).

The vLAN relay (Patterns A and B) and the UDP proxy (Pattern C) are separate code paths in `internal/`. They share the lobby/room concept but have different wire protocols and different per-room state.

## Three Asset Models

Orthogonal to architecture pattern, games have one of three asset models. The server treats all three as first-class.

**Self-contained** (OpenArena)
- The catalog archive contains everything. Manifest has empty `requires_assets`. Registry reports `not_required`. UI shows "Ready."

**Host-supplied or free-bundle** (Zandronum, Quakespasm)
- Archive contains the engine/client; assets live separately in `./assets/<game-id>/`.
- Manifest's `requires_assets` lists filenames, optional known-hashes, and an optional `free_alternative` pointing to an asset bundle.
- Host drops their own files into the assets directory or runs `install-free-assets`.
- Registry reports `missing`, `present_unverified`, `verified_commercial`, or `verified_free`.

**Host-supplied only** (no free bundle exists; e.g. eventual Heretic, Duke3D)
- Same as above but `free_alternative` is null. UI says "needs `<filename>`" without an install-free-assets suggestion.

The taxonomy is small enough to keep in your head and covers every catalog game we can foresee.

## System Overview

```
                  GitHub: lpaas-98-games repo
                  ┌─────────────────────────────────┐
                  │  catalog.json                   │
                  │  Releases:                      │
                  │   - WASM engine builds          │
                  │   - self-contained game bundles │
                  │   - free asset bundles          │
                  └─────────────────┬───────────────┘
                                    │ HTTPS, on-demand
                                    │ (install/update only)
                                    ▼
        ┌─────────────────────────────────────────────────┐
        │  Host laptop (running ./lpaas98 server)         │
        │                                                 │
        │   ┌────────────┐  ┌──────────────┐ ┌──────────┐ │
        │   │ HTTP       │  │ vLAN relay   │ │ UDP      │ │
        │   │ server     │  │ (Pattern A/B)│ │ proxy    │ │
        │   │            │  │              │ │(Pattern C)│ │
        │   └────────────┘  └──────────────┘ └────┬─────┘ │
        │         ▲                ▲              │       │
        │         │                │              ▼       │
        │   ┌─────┴────┐  ┌────────┴───┐  ┌───────────┐  │
        │   │ Game     │  │ Lobby /    │  │ Subprocess│  │
        │   │ registry │  │ session    │  │ manager   │  │
        │   └──────────┘  │ manager    │  │           │  │
        │         ▲       └────────────┘  └─────┬─────┘  │
        │   ┌─────┴──────┐                      │        │
        │   │ Installer  │                      ▼        │
        │   └────────────┘             ┌────────────┐    │
        │         ▲                    │ Zandronum  │    │
        │   ./games/                   │ server     │    │
        │   ./assets/                  │ (UDP, local)│   │
        │                              └────────────┘    │
        └──────────┼──────────────────────────────────────┘
                   │
        Local Wi-Fi / Ethernet (HTTP, no TLS)
                   │
        ┌──────────┴─────────┬─────────────────┬──────────┐
        ▼                    ▼                 ▼          ▼
    Guest browser 1     Guest browser 2     Guest 3    Guest 4
    (Win98 desktop UI + WASM game + WebSocket to relay or proxy)
```

## Components

### HTTP Server
Serves the SPA, game manifests, WASM blobs, and game assets. Plain `net/http`. Listens on `0.0.0.0:9898` by default, configurable via `--addr`. Logs all LAN-reachable URLs on startup so the host can read one out.

### Catalog & Installer
Fetches the remote `catalog.json` on demand (`lpaas98 catalog update`), and installs:
- **Game entries** (`lpaas98 install <game-id>`) into `./games/<game-id>/`. For Pattern A games (OpenArena), the archive includes engine WASM + game data. For Pattern B (Quakespasm), only the engine WASM. For Pattern C (Zandronum), the archive includes WASM client + native server binaries for each supported OS/arch.
- **Free asset bundles** (`lpaas98 install-free-assets <game-id>`) into `./assets/<game-id>/`, only when explicitly invoked.

Both flows download from URLs in `catalog.json`, verify SHA256, and unpack atomically.

See `CATALOG.md` for the catalog format.

### Game Registry
On startup (and on `SIGHUP`) scans the `games/` directory. Each subdirectory must contain a `manifest.json`. Invalid manifests are logged and skipped.

For each game, the registry derives an **asset status**:
- `not_required` — manifest's `requires_assets` is empty (e.g. OpenArena). Game is launchable.
- `missing` — assets required, none present. Game shown but not launchable.
- `present_unverified` — files exist but don't match any known SHA256. Allowed; game is launchable.
- `verified_commercial` — files match a known commercial hash.
- `verified_free` — files match the free-asset bundle hash.

For Pattern C games, the registry also verifies that the host's OS/arch is among the bundled server binaries; if not, the game shows as "unavailable on this platform."

The UI uses these to label games and show appropriate install hints.

### Lobby / Session Manager
Tracks connected clients, nicknames, and rooms. A **room** is one instance of one game. Rooms have a max player count from the game manifest. All state is in-memory; restart wipes it.

Room creation behaves differently per pattern:
- **Pattern A / B:** room is purely logical — just a record in the lobby that browsers can join. The vLAN relay attaches WebSocket connections to the room.
- **Pattern C:** room creation spawns a subprocess (via the Subprocess Manager) bound to an allocated UDP port. The room record stores the subprocess PID and port. The UDP proxy routes per-room traffic to/from that port.

### Virtual LAN Relay (vLAN) — Pattern A / B
Each room is a virtual network segment. Clients connect via WebSocket and the server forwards messages between them per the game's network model:
- **`broadcast`** (IPX-style): server fans out every message to every other client in the room. Defined for future games (Heretic, Hexen, eDuke32, etc.); not used by any launch game.
- **`host-client`** (used by Quake-family games and ioquake3-family games): server picks the first joiner as host, routes others' traffic to them, host's traffic back to specified clients.
- **`custom`**: per-game adapter decides.

The relay doesn't understand game protocols. It moves opaque bytes between WebSocket connections within a room. The per-game WASM shim is responsible for translating between the game's expected networking and our WebSocket frames.

### UDP Proxy — Pattern C
For native-subprocess games, the proxy bridges browser WebSocket connections to UDP packets the subprocess server expects.

For each browser client in a Pattern C room:
- The proxy maintains a per-client UDP socket bound to an ephemeral local port. Bytes received over the client's WebSocket are sent as a UDP packet from that socket to the subprocess's port. From the subprocess's perspective, each browser client looks like a distinct UDP client at a unique localhost:port.
- UDP packets *from* the subprocess back to that ephemeral port are forwarded out over the client's WebSocket.
- This per-client socket mapping is what lets the subprocess (which has no concept of WebSockets) treat each browser as a distinct UDP peer.

The proxy is unaware of game protocols. It moves opaque bytes between WebSocket and UDP socket per room.

### Subprocess Manager — Pattern C
Spawns and supervises the native game-server binaries from `games/<game-id>/server/<os>-<arch>/`. Per-room responsibilities:
- Allocate a free UDP port (from a configurable range, default `10666-10765`).
- Launch the subprocess with appropriate command-line flags (port, game-mode, WAD, max players — all derived from the manifest and room settings).
- Capture stdout/stderr to a per-room log buffer for diagnostics.
- Monitor the process; if it exits, mark the room as ended and notify connected clients.
- Enforce a per-host limit on concurrent subprocesses (default 4) to protect the host's CPU and ports.

When the room closes (all clients disconnected, or explicitly), the manager sends SIGTERM to the subprocess, waits a few seconds for clean exit, then SIGKILL if it hasn't terminated. Allocated UDP port is returned to the pool.

Subprocess binaries are trusted (they're shipped via the catalog and SHA256-verified) but still run as the LPaaS 98 user, not as root. No privilege escalation, no special capabilities.

### Per-Game Adapter (host-side, compiled in)
A small Go file per supported game declaring its pattern (A, B, or C), network model (for A/B), and any pre/post hooks. For Pattern C games, the adapter also declares the subprocess launch command template and any required environment. Ships compiled into the server binary. Adding a new game adapter requires a server release.

### Per-Game Shim (client-side, in catalog archive)
Each game's WASM bundle is paired with a small JS adapter that:
1. Opens the WebSocket to the room
2. Hands the game a fake network interface it recognizes (BSD sockets, NetQuake sockets, ioquake3's netchan, Zandronum's wire protocol, etc.)
3. Translates the game's network calls into WebSocket frames

For Pattern A/B games, the WebSocket terminates at the vLAN relay; for Pattern C, the WebSocket terminates at the UDP proxy. The shim doesn't need to know which — both look identical from the WASM side.

This is per-game work and the main source of effort for adding new games. Ships in the catalog archive.

## Game Manifest Format

```json
{
  "id": "openarena",
  "name": "OpenArena",
  "version": "0.8.8",
  "description": "Free Quake III-style arena FPS, GPL game and assets",
  "min_players": 2,
  "max_players": 16,
  "network_model": "host-client",
  "wasm": "openarena.wasm",
  "js_loader": "loader.js",
  "requires_assets": [],
  "icons": {
    "16": "icon_16.png",
    "32": "icon_32.png",
    "48": "icon_48.png"
  },
  "shortcut_name": "OpenArena",
  "license": "GPL-2.0",
  "source_url": "https://github.com/OpenArena/engine"
}
```

For Pattern B games (engine-only with assets), `requires_assets` is populated:

```json
{
  "id": "quakespasm",
  "pattern": "B",
  "network_model": "host-client",
  "wasm": "quakespasm.wasm",
  "js_loader": "loader.js",
  "requires_assets": [
    {
      "filename": "id1/pak0.pak",
      "description": "Quake PAK0",
      "known_hashes": {
        "verified_commercial": ["sha256:..."],
        "verified_free": ["sha256:..."]
      },
      "free_alternative": "librequake"
    }
  ],
  ...
}
```

For Pattern C games (subprocess server + WASM client), the manifest also declares the server binaries and launch parameters:

```json
{
  "id": "zandronum",
  "pattern": "C",
  "wasm_client": "zandronum-client.wasm",
  "js_loader": "loader.js",
  "server": {
    "binaries": {
      "linux-amd64": "server/linux-amd64/zandronum-server",
      "linux-arm64": "server/linux-arm64/zandronum-server",
      "darwin-amd64": "server/darwin-amd64/zandronum-server",
      "darwin-arm64": "server/darwin-arm64/zandronum-server",
      "windows-amd64": "server/windows-amd64/zandronum-server.exe"
    },
    "launch": {
      "args": ["-host", "{max_players}", "-port", "{udp_port}", "-iwad", "{asset_path}/doom.wad"],
      "graceful_shutdown_seconds": 5
    }
  },
  "requires_assets": [
    {
      "filename": "doom.wad",
      "description": "DOOM IWAD",
      "known_hashes": { "verified_commercial": ["..."], "verified_free": ["..."] },
      "free_alternative": "freedoom-phase1"
    }
  ],
  ...
}
```

The `{max_players}`, `{udp_port}`, and `{asset_path}` placeholders are substituted by the Subprocess Manager at launch.

Empty `requires_assets` is the canonical signal for "self-contained game, no assets to set up."

## Directory Layout (on the host, at runtime)

```
lpaas98                       # the server binary
games/                        # engine builds + self-contained game data + native server binaries
  openarena/                  # Pattern A
    manifest.json
    openarena.wasm
    loader.js
    icon_16.png
    icon_32.png
    baseoa/                   # OpenArena's own game data, shipped in archive
      pak0.pk3
      pak1-maps.pk3
  quakespasm/                 # Pattern B
    manifest.json
    quakespasm.wasm
    loader.js
    icon_16.png
    icon_32.png
  zandronum/                  # Pattern C
    manifest.json
    zandronum-client.wasm     # WASM client
    loader.js
    icon_16.png
    icon_32.png
    server/                   # native server binaries, per-platform
      linux-amd64/zandronum-server
      linux-arm64/zandronum-server
      darwin-amd64/zandronum-server
      darwin-arm64/zandronum-server
      windows-amd64/zandronum-server.exe
assets/                       # host-supplied OR install-free-assets
                              # OpenArena has no entry here (not_required)
  zandronum/
    doom.wad                  # shared with any future Doom-engine games
    .source                   # "freedoom-phase1 0.13.0" or "host-supplied"
    _bundle-info/
      LICENSE
      README.md
  quakespasm/
    id1/
      pak0.pak
    .source
```

The installer writes to `games/` (any install) and `assets/` (only on explicit `install-free-assets`). It never overwrites a `.source: host-supplied` directory without `--force`.

## CLI Surface

```
lpaas98 server                                # start the HTTP server
lpaas98 catalog list                          # list available games + asset bundles
lpaas98 catalog update                        # refresh local catalog cache
lpaas98 install <game-id>                     # download + verify + unpack
lpaas98 install <game-id>@<version>           # pin a specific version
lpaas98 install-free-assets <game-id>         # opt-in free assets (no-op for self-contained games)
lpaas98 uninstall <game-id>                   # remove game (assets preserved)
lpaas98 uninstall-assets <game-id>            # remove assets only
lpaas98 update                                # update all installed games
lpaas98 list                                  # show installed games + asset status
lpaas98 version
```

`install-free-assets` on a self-contained game like OpenArena prints a friendly message ("OpenArena ships with its own assets — nothing to install") and exits 0.

Flags (selected):
- `--addr` (default `0.0.0.0:9898`)
- `--games-dir` (default `./games`)
- `--assets-dir` (default `./assets`)
- `--catalog-url` (default points at `lpaas-98-games`)
- `--config` (path to optional TOML config)
- `--yes` (skip confirmation prompts)

## Discovery on the LAN

For v1: server prints all non-loopback IPv4 addresses on startup, in a friendly format the host can read out. mDNS / Bonjour is post-v1.

## Identity

Nickname-only. Stored client-side (localStorage). No accounts. Duplicate nicknames get suffixed (`bob`, `bob (2)`).

## Frontend (deferred but stub it)

The eventual UI is a Win98-styled web desktop. Installed games appear as desktop icons. Double-click launches a room. Active rooms appear in a taskbar.

The asset-status taxonomy maps to UI states:
- `not_required` / `verified_*` / `present_unverified` → game icon is bright, double-click works
- `missing` → game icon is greyed, double-click shows a dialog explaining what's needed (and offering `install-free-assets` if applicable)

For v1, only a minimal functional frontend is needed. The Win98 polish is its own milestone (see `PLAN.md`).

## Repo Layout

```
/cmd/lpaas98/             # main package, CLI entrypoint
/internal/
  /http/                  # HTTP server, static handlers, API
  /lobby/                 # sessions, rooms, nicknames
  /vlan/                  # WebSocket relay for Pattern A/B games
  /udpproxy/              # WebSocket↔UDP proxy for Pattern C games
  /subprocess/            # native game-server subprocess manager (Pattern C)
  /registry/              # games/ scanning, manifest parsing, asset state
  /catalog/               # remote catalog fetch + parse
  /installer/             # download, verify, unpack (engines + free assets)
  /games/                 # per-game Go adapters (compiled in)
    /openarena/
    /zandronum/
    /quakespasm/
/web/                     # frontend
/docs/
  ARCHITECTURE.md
  CATALOG.md
  ADDING_A_GAME.md
  PROTOCOL.md
/.github/workflows/
go.mod
README.md
LICENSE
```

## Cross-Compilation Targets

Release pipeline produces:
- `linux/amd64`, `linux/arm64`
- `darwin/amd64`, `darwin/arm64`
- `windows/amd64`, `windows/arm64`

Plus optionally `linux/arm` (Raspberry Pi 32-bit) and `freebsd/amd64`.

CI: GitHub Actions matrix build, attach binaries to releases. `CGO_ENABLED=0` keeps it pure-Go and trivially static.

## Future (post-v1)
- HTTPS flag and Let's Encrypt integration for VPS mode
- mDNS advertisement of `lpaas98.local`
- Catalog signature verification (minisign/cosign)
- More games: Heretic, Hexen, eDuke32, Freeciv-web, OpenTTD, OpenRA, OpenRCT2
- Win98 desktop UI polish
- Voice chat relay
- Spectator mode
