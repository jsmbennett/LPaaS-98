# Decisions

A log of non-obvious decisions made during planning and development. New entries go at the top.

Format:
```
## YYYY-MM-DD: Title

**Decision:** What we chose.
**Alternatives:** What we considered.
**Reason:** Why.
**Revisit when:** What would make us reconsider.
```

---

## 2026-05-25 (superseded 2026-05-25, see "Chocolate Doom dropped" below): Internet-play performance depends on game's network model

**Decision:** Acknowledge that the launch lineup's network models perform very differently over the internet, and that modern client/server Doom ports (Odamex, Zandronum, ZDaemon) would give better internet performance than Chocolate Doom — but at the cost of harder WASM compilation.

**Originally:** Deferred to a Phase 7 spike: attempt Odamex-to-WASM; if feasible, swap Odamex in for Chocolate Doom; if not, ship Chocolate Doom and accept LAN-only as the realistic use case.

**Superseded by:** Discovery of dorch (gib.gg)'s subprocess-server pattern. Zandronum is now directly in the launch lineup via a different architectural pattern (native server subprocess + WASM client + UDP proxy), not via direct WASM compilation. Chocolate Doom is dropped from the launch lineup entirely. See the "Subprocess game servers" and "Chocolate Doom dropped" entries below.

---

## 2026-05-25: vLAN transport is pluggable; v1 ships WebSocket, future may add WebTransport

**Decision:** The `internal/vlan` package is designed around a transport-agnostic frame interface ("send opaque bytes to peer X," "receive opaque bytes from peer Y"), with WebSocket as the only v1 implementation. WebTransport (HTTP/3 / QUIC, providing both reliable and unreliable streams) is identified as the right transport for the eventual VPS / internet-host mode, but is not implemented in v1.

**Alternatives:** (1) Hard-code WebSocket throughout the relay code. (2) Implement WebTransport now. (3) Use raw TCP/UDP with no transport abstraction (won't work in browsers).

**Reason:** WebSocket is universally supported and is the right choice for LAN play where TCP semantics don't bite. WebTransport gives unreliable datagrams (matching what vintage games actually expect) and avoids TCP head-of-line blocking, which dominates internet-play latency for these games — but Safari support is incomplete and corporate networks sometimes block QUIC. Building both now doubles the surface; building one with a clean abstraction lets us add the other when the post-v1 internet-hosting mode actually exists. The cost of "design the interface to be pluggable" today is one extra Go interface declaration; the cost of retrofitting later would be a rewrite of the relay.

**Revisit when:** When VPS / internet-host mode (post-v1) is built. WebTransport browser support, especially in Safari, is also a trigger to revisit.

---

## 2026-05-25: Chocolate Doom dropped from launch lineup

**Decision:** Launch lineup is OpenArena, Zandronum (with Freedoom), and Quakespasm (with LibreQuake). Chocolate Doom is not in v1. The `broadcast` network model is defined in the protocol for future use but not implemented in v1.

**Alternatives:** (1) Keep Chocolate Doom as a third game alongside the others. (2) Include only OpenArena and Zandronum. (3) Original plan: Chocolate Doom, Quakespasm, and OpenArena.

**Reason:** Zandronum is strictly better than Chocolate Doom for multiplayer LAN play — more players, RUDP networking that handles real-world latency, join-in-progress, modern menus, mod support. Including Chocolate Doom alongside Zandronum would be shipping the worse Doom experience to demo a different technology stack. The new three-game lineup covers three distinct architectural patterns, each load-bearing: self-contained pure-WASM (OpenArena), subprocess native server + WASM client + UDP proxy (Zandronum), and engine-only pure-WASM with host-supplied or free assets (Quakespasm). The `broadcast` model stays in the protocol spec because future games (Heretic, Hexen, eDuke32) will likely need it.

**Revisit when:** When adding Heretic, Hexen, or other Doom-engine games post-v1, the `broadcast` model gets implemented and Chocolate Doom could be added back as the reference implementation. Not on the v1 critical path.

---

## 2026-05-25: Subprocess game servers promoted to v1 architecture for Zandronum

**Decision:** LPaaS 98 supports a fourth game-architecture pattern: a native game-server binary running as a subprocess of the LPaaS 98 server, with the LPaaS 98 server acting as a transport proxy between browser-side WASM clients and the subprocess via UDP. Used for the launch lineup's Zandronum integration. Inspired directly by the dorch / gib.gg project (https://github.com/beebs-dev/dorch).

**Alternatives:** (1) Skip Zandronum entirely. (2) WASM-compile both Zandronum client and server, host server in browser, use vLAN relay (impractical — server needs filesystem access, threading, lots of native state). (3) Build on top of dorch's full Kubernetes stack (massive overkill for LAN-party use).

**Reason:** Zandronum is the multiplayer-first Doom port. Running its actual upstream server binary in a subprocess is dramatically simpler than WASM-compiling the engine and faking a network — spawn the process, allocate a UDP port, route browser WebSocket traffic to that UDP port. The WASM Zandronum *client* talks Zandronum's native wire protocol through the proxy. dorch demonstrates this in production; we adapt the pattern to our scale (no Kubernetes, no Postgres, no identity stack).

This adds a new architectural pattern but doesn't displace existing ones — OpenArena and Quakespasm stay pure-WASM-in-browser. Three patterns in v1, each demonstrated by one game.

**Revisit when:** If the Phase 7c spike reveals the WASM Zandronum client cannot be built cleanly, fall back to (a) dropping Zandronum or (b) substituting Odamex via the same subprocess pattern. If subprocess management proves too platform-divergent (Windows process handling differs significantly from Unix), this might need OS-specific code paths.

---

## 2026-05-25: vLAN protocol uses JSON control + binary data on one WebSocket

**Decision:** The vLAN relay carries two kinds of messages on each WebSocket: JSON text frames for control (join/leave/errors), raw binary frames for game packets. Broadcast model has zero envelope on the data plane; host-client uses a one-byte addressing prefix. Spec lives in `docs/PROTOCOL.md`.

**Alternatives:** (1) All JSON with base64-encoded payloads. (2) All binary with a custom framing format. (3) MessagePack throughout. (4) Protocol Buffers.

**Reason:** Game packets at 30-60 Hz with multiple peers can be thousands of packets/second. Per-packet overhead matters. JSON+base64 adds ~33% to the data plane *and* per-packet CPU; binary frames add zero (broadcast) or one byte (host-client). Meanwhile the control plane is low frequency and benefits hugely from JSON's readability for debugging — you can watch traffic in browser devtools and `wscat`. WebSocket supports text and binary opcodes on the same socket natively, so this is free architecturally. Avoids any non-stdlib dependency on either side.

**Revisit when:** If profiling shows JSON parsing on the control plane is non-trivial (very unlikely given low frequency), or if a future game needs structured payloads on the data plane (move that game to `network_model: custom` rather than changing the broadcast/host-client framing).

---

## 2026-05-25: Default port is 9898

**Decision:** The server listens on `0.0.0.0:9898` by default. Overridable via `--addr host:port`.

**Alternatives:** 8080 (the original default); a port in the IANA private range 49152+; letting the OS pick a port.

**Reason:** 8080 collides with countless other services (Jenkins, Tomcat, dev servers). 9898 is in the user-ports range, not assigned by IANA to anything common, memorable, reinforces the project's "98" branding, and is visually distinctive in netstat output. The private range 49152+ would be safest from collision but loses the brand value and looks weird in URLs. OS-picked ports are wrong here because the URL needs to be readable aloud at a LAN party.

**Revisit when:** If a host reports a real-world collision (some Monit/monitoring tools have historically used ports in the 989x range), revisit. Otherwise stable.

---

## 2026-05-24: OpenArena added as launch game, placed first in build order

**Decision:** v1 ships with three games — OpenArena, Chocolate Doom, Quakespasm. OpenArena is built first (Phase 7), Doom second (Phase 8), Quake third (Phase 9).

**Alternatives:** (1) Just Doom + Quake. (2) Same three games but Doom first.

**Reason:** OpenArena is self-contained (no asset model), so it isolates the WASM-shim-meets-vLAN integration risk from the asset-bundle risk. Building it first means the first real-game phase tackles one unknown rather than two simultaneously. It's also the friendliest possible first-run experience (`install openarena` and you're playing), which makes a stronger initial demo.

**Revisit when:** If a half-day Emscripten spike on ioquake3 reveals WASM compilation is harder than for Chocolate Doom, swap the order — Doom first as a known path, OpenArena second once shim patterns are established.

---

## 2026-05-24: Asset taxonomy has five states, not four

**Decision:** Asset status is one of `not_required`, `missing`, `present_unverified`, `verified_commercial`, `verified_free`.

**Alternatives:** Four states (treating `not_required` as a special case of `verified_*`); or two states (`ok` / `missing`).

**Reason:** `not_required` is the canonical signal for self-contained games (empty `requires_assets`). Treating it as a first-class state means OpenArena isn't an edge case, future self-contained games drop in cleanly, and the UI can render distinct affordances (no "install free assets" hint for self-contained).

**Revisit when:** If we ever need to distinguish "verified specific commercial version" from "verified some commercial version," the taxonomy may grow. Not now.

---

## 2026-05-24: Single `kind: "game"` instead of separate `engine` vs. `self-contained`

**Decision:** Catalog entries are either `kind: "game"` or `kind: "asset-bundle"`. Self-contained vs engine-only is determined by whether the manifest's `requires_assets` is empty.

**Alternatives:** Three kinds (`engine`, `self-contained-game`, `asset-bundle`).

**Reason:** The distinction lives more naturally in the manifest (where the consumer of the information is) than in the catalog entry type. The installer doesn't need to do anything different at install time — the difference only matters to the registry when reporting asset status.

**Revisit when:** If asset bundles ever start needing the same "may or may not have asset deps" treatment, this might need rethinking. Unlikely.

---

## 2026-05-24: Project name is "LPaaS 98"

**Decision:** LPaaS 98 — LAN Party as a Service, with "98" anchoring the Win98 UI theme.

**Alternatives considered:** Many. The shortlist was Network Neighborhood (rejected over trademark concerns with Microsoft), AutoRun, Hub98, Sneakernet, LANhood, PartyOS, just plain "LPaaS."

**Reason:** "LPaaS" reads as a real enterprise acronym (parallel to SaaS/PaaS/IaaS) which sets up the deadpan corporate joke. "98" makes the Win98-themed UI feel inevitable rather than decorative. Doesn't invoke any specific Microsoft product/feature name (so lower trademark risk than "Network Neighborhood").

**Revisit when:** If the Win98 UI direction is abandoned, "98" loses its grounding and just "LPaaS" might be cleaner.

---

## 2026-05-24: Two-repository structure

**Decision:** Server (`lpaas-98`) and catalog+games (`lpaas-98-games`) live in separate repos.

**Alternatives:** Monorepo with everything; server fetches games from arbitrary URLs with no central catalog.

**Reason:** Lets game builds update independently of the server. Keeps GPL game-port redistribution cleanly isolated from the Apache-2.0 server. Lets people fork the catalog (corporate LAN parties, custom packs) without forking the server. The default catalog URL is hardcoded but `--catalog-url` overrides.

**Revisit when:** If the catalog ever needs server-side computation (signing, dynamic content), the games repo becomes more than static GitHub releases and the relationship may need to change.

---

## 2026-05-24: HTTP only on LAN; no TLS in v1

**Decision:** Plain HTTP. No TLS. The host's LAN IP goes directly into the browser address bar.

**Alternatives:** Self-signed certs with click-through; mkcert-style local CA; HTTPS-only.

**Reason:** Self-signed certs trigger browser warnings that frustrate users at a LAN party. mkcert requires installing a local CA on every guest's laptop, which is hostile. Plain HTTP works everywhere with no setup. The threat model for a LAN at a friend's house is low. HTTPS becomes relevant only for the post-v1 VPS hosting mode.

**Revisit when:** If browsers start refusing more APIs over HTTP on local IPs (the trend is in that direction), or when VPS mode is implemented.

---

## 2026-05-24: Go, not Rust

**Decision:** Server is written in Go.

**Alternatives:** Rust.

**Reason:** Cross-compilation is much easier in Go (single env var per target, static binaries, no toolchain juggling for arm64). WebSocket libraries are more mature. The project's needs (HTTP server, WebSocket relay, file I/O, subprocess management) play to Go's strengths. Rust's advantages (memory safety beyond what GC provides, performance ceiling) aren't decisive here.

**Revisit when:** Not foreseeing a reason to revisit. If performance ever becomes a bottleneck — unlikely for a small LAN relay — profile first.

---

## 2026-05-24: Host supplies commercial game assets; free alternatives are opt-in via separate command

**Decision:** The installer never bundles or auto-installs commercial assets. For games with free alternatives (Freedoom for Doom, LibreQuake for Quake), the user must explicitly run `install-free-assets <game-id>`. The installer refuses to overwrite `host-supplied` assets without `--force`.

**Alternatives:** Auto-install free assets when the engine is installed; bundle Freedoom/LibreQuake into the engine archives.

**Reason:** Conflating "I installed the Doom engine" with "I want Freedoom's specific game data" is bad UX — Freedoom is a different game from Doom, with different levels and aesthetics. The host should make an informed choice. The `.source` marker mechanism also protects hosts who legitimately own and prefer their own WADs from having them replaced by `update`.

**Revisit when:** If users overwhelmingly want the simpler "auto-install everything" path and find the current flow confusing. The first-run wizard in Phase 10 might paper over this without changing the underlying model.
