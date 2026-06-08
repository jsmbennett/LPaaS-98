# Build Plan

A phased plan for getting LPaaS 98 from empty repos to a launch-ready release with one playable game (OpenArena).

Each phase ends in something demonstrable. Two repos are involved: **`lpaas-98`** (the server) and **`lpaas-98-games`** (the catalog + archives).

## Phase 0 — Repo Foundation (1 evening)

**Server repo (`lpaas-98`):**
- `go mod init github.com/<you>/lpaas-98`
- Apache-2.0 license
- `README.md` (drafted)
- `docs/ARCHITECTURE.md`, `docs/CATALOG.md`, `PLAN.md`
- GitHub Actions skeleton: `go vet`, `go test ./...` on push
- `.editorconfig`, `.gitignore`

**Games repo (`lpaas-98-games`):**
- Empty `catalog.json` with `schema_version: 1` and `entries: []`
- `README.md` explaining it's the default catalog
- Same license

**Deliverable:** both repos exist, CI green, docs readable.

## Phase 1 — HTTP Server + Game Registry (1-2 evenings)

- `cmd/lpaas98/main.go` with subcommand parsing (`server`, `version`, stubs for the rest)
- `internal/http`: serves a static `index.html` ("Hello, LPaaS 98") and `GET /api/games` returning the registry's list with asset status
- `internal/registry`: scans `--games-dir`, parses `manifest.json` files, derives the five asset statuses (`not_required`, `missing`, `present_unverified`, `verified_commercial`, `verified_free`)
- Log all LAN-reachable IPv4 addresses on startup
- Hand-create two test game directories in `games/`: one self-contained, one with `requires_assets`, both with stub WASM. Exercise both paths in the registry.

**Deliverable:** run the binary, hit `http://<lan-ip>:9898/api/games` from another laptop, see both test games with correct asset statuses. Manifest format and asset taxonomy are settled.

## Phase 2 — Cross-Compile Release Pipeline + Docker (1-2 evenings)

**Binaries:**
- GitHub Actions release workflow: on tag push, build all target platforms, attach to release
- `CGO_ENABLED=0`, embed version via `-ldflags`
- Verify by tagging v0.1.0 and downloading the macOS arm64 binary on a real Mac

**Docker:**
- Multi-stage `Dockerfile` at repo root: build stage on `golang:alpine`, runtime stage on `scratch`. Final image ~10 MB.
- Multi-arch build via `docker buildx`: `linux/amd64` and `linux/arm64` from a single tag. This is the performance-critical piece — it ensures Docker Desktop on Apple Silicon and arm64 Linux servers (Raspberry Pi, Graviton, etc.) run native binaries, not QEMU-emulated amd64.
- GitHub Actions workflow pushes images to GitHub Container Registry (`ghcr.io/<owner>/lpaas-98:vX.Y.Z` and `:latest`) on tag push.
- `docker-compose.yml` example in the repo for the common "host wants to run it as a container" case, with `./games` and `./assets` mounted as volumes and port 9898 published.

**Deliverable:** anyone can either grab a binary from Releases or `docker run ghcr.io/<owner>/lpaas-98:latest`. Both paths produce native performance on amd64 and arm64.

## Phase 3 — Catalog & Installer (Games) (2-3 evenings)

- `internal/catalog`: fetch + parse `catalog.json`
- `internal/installer`: `install`, `uninstall`, `update`, `list` subcommands for `kind: game` entries
- SHA256 verification, safe tar extraction, atomic install
- Catalog caching to `./catalog.cache.json`
- Streaming download with progress reporting (matters for big archives like OpenArena)
- In `lpaas-98-games`: publish one **dummy game** archive (self-contained, ~1 MB) with a minimal manifest and stub WASM

**Deliverable:** `lpaas98 install example` downloads from `lpaas-98-games`, verifies, unpacks; registry picks it up. End-to-end catalog flow works against a real GitHub release.

## Phase 4 — Lobby + Nicknames + Rooms (2-3 evenings)

- Minimal frontend: nickname prompt, game list, "create room" / "join room" buttons. Plain HTML+JS. Win98 styling deferred to Phase 8.
- `internal/lobby`: in-memory rooms with game ID, host, members, max size
- HTTP endpoints: `POST /api/rooms`, `GET /api/rooms`, `POST /api/rooms/:id/join`
- WebSocket `/ws/lobby` for live room list updates

**Deliverable:** two laptops can both join, see each other in a room, watch the list update live.

## Phase 5 — Virtual LAN Relay (2-4 evenings)

- `internal/vlan`: WebSocket endpoint `/ws/room/:id`, frame format in `docs/PROTOCOL.md`
- Implement `host-client` model for OpenArena
- Backpressure — drop or disconnect slow clients
- Test harness: a Go program that pretends to be N clients and exchanges messages through a room

**Deliverable:** automated relay tests pass; manual browser-tab test works.

## Phase 6 — Upstream Spike (3-5 days, fixed timebox)

A research phase before committing to specific implementation paths. Goal: validate that OpenArena can actually be built with our architectural pattern, before sinking weeks into engine-side work.

**Spike 6a — OpenArena / ioquake3 WASM (3-5 days):**
- Look for existing WASM builds (`ioq3-emscripten`, ioquake3 forks). Does any run today in a current browser?
- If yes: how modular is the JS-side networking? Could we replace it without rebuilding the WASM?
- If no working build exists: estimate effort to build from source with Emscripten. ioquake3 uses SDL2, OpenAL, WebGL-compatible OpenGL — all targetable.
- Output: YES/NO on "OpenArena is feasible for Phase 7" plus an effort estimate and a concrete plan.

**Deliverable:** spike report with findings, a finalized feasibility decision, and any necessary updates to `ARCHITECTURE.md` and the catalog manifests.

## Phase 7 — OpenArena End-to-End (2-4 weeks)

The first real game. Self-contained pure-WASM; building it validates the WASM-shim-to-vLAN-relay path end to end.

Sub-phases:
1. Get OpenArena/ioquake3 WASM running single-player (or bot match) in the browser, loaded from the catalog flow
2. Per-game Go adapter, `network_model: host-client`
3. JS shim translating ioquake3's netchan calls to our WebSocket frames
4. Two browsers in a deathmatch on localhost
5. Two laptops over real Wi-Fi
6. Package as `openarena-0.8.8.tar.gz` (engine + game data), publish to `lpaas-98-games`, add to `catalog.json`
7. Document the integration in `docs/ADDING_A_GAME.md`

**Deliverable:** `lpaas98 install openarena && lpaas98 server` produces a playable OpenArena LAN session. Tag server v0.2.0.

## Phase 8 — Hardening (1-2 weeks)

- Reconnection on dropped WebSockets
- Room cleanup when empty
- Per-room rate limits to protect the host's CPU
- Graceful shutdown
- Structured logging (`log/slog`)
- TOML config file support in addition to flags
- A `--help` that's actually helpful
- Better installer error messages
- "First-run wizard" mode that recommends `install openarena` for instant gratification

**Deliverable:** server v0.3.0, "stable enough to use at a real party."

## Phase 9 — Win98 Desktop UI (1-2 weeks)

This is when LPaaS 98 *looks* like LPaaS 98.

- Integrate `98.css` or hand-roll equivalent
- Boot screen with the LPaaS 98 wordmark
- Teal desktop with installed games as draggable icons (16/32 px from manifests)
- Double-click to launch a room
- Active rooms appear in the taskbar
- "Start" menu: About, Settings, Shutdown (= disconnect)
- "My Network" icon showing connected players
- Optional: modem handshake SFX on first connect

**Deliverable:** server v0.4.0. Screenshots that make people want to try it.

## Phase 10 — Launch

- Public release announcement
- Contributor guide
- Documentation for adding new games
- Discussion forum or Discord

## Effort Estimate

Solo, evenings and weekends:
- Phases 0-5 (foundation + plumbing): ~4-5 weeks
- Phase 6 (upstream spike, fixed timebox): 3-5 days
- Phase 7 (OpenArena): ~2-4 weeks
- Phase 8 (hardening): ~1-2 weeks
- Phase 9 (Win98 UI): ~1-2 weeks
- Phase 10 (launch): ~1 week

**~2-3 months to v0.4.0 with one playable game and a Win98 desktop UI.**

Phase 6's spike is the single most valuable sprint in the plan — it converts "can we actually build ioquake3 WASM?" into known constraints before weeks of integration work.

## Open Questions to Resolve

**Before Phase 6 (the spike):**
- Which existing WASM builds for ioquake3 are maintained and runnable today?
- Is the JS-side networking shim replaceable without rebuilding the WASM, or do we need to fork and patch the engine?

**Before Phase 7 (OpenArena integration):**
- Output of Phase 6 spike: existing WASM build vs. self-built; if self-built, which Emscripten flags?
- Max room size policy — hard cap from manifest, or host-override permitted?

**Before Phase 10 (launch):**
- Catalog signing: minisign or cosign for the v2 schema?
- Community forum: Discord vs GitHub Discussions?

None of these block earlier phases.
