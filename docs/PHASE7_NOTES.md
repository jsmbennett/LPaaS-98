# Phase 7 Notes: OpenArena End-to-End

## Status

**Infrastructure: ✅ Complete**
**WASM Build: ✅ Complete**
**Engine Initialization: ✅ Complete**
**Game Rendering & Input: 🔲 Pending**
**Multiplayer Testing: 🔲 Pending**

## What's Been Built

### Server-side
- Game adapter framework (`internal/games/openarena.go`)
- vLAN relay integrated into HTTP server
- Game page handler (`/game.html?room=<id>&game=<id>`)
- JavaScript loader handler (`/loader.js`)
- WebSocket relay endpoint (`/ws/room/:id?nickname=<name>`)

### Client-side (Browser)
- HTML5 game container with HUD (fullscreen support)
- GameLoader class that:
  - Connects to WebSocket relay
  - Handles relay protocol (hello, peer_joined, peer_left)
  - Implements host-client packet routing (0x00 to host, 0xFF broadcast)
  - Updates HUD with connection status

### Network Model
- Host-client routing implemented in relay
- Packets routed by first byte: 0x00 (to host), 0x01-0xFE (to specific peer), 0xFF (broadcast)
- Non-host clients can only send 0x00 (to host)
- Host can send any routing byte

## What's Been Completed (as of 2026-06-08)

1. ✅ **WASM Binary** — OpenArena built with Emscripten, 2.2MB
2. ✅ **Game Archive** — Packaged as openarena-0.8.8.tar.gz with manifest
3. ✅ **Server File Serving** — `/game/{gameID}/{file}` route for assets
4. ✅ **Game Page Loader** — HTML page that loads game-specific WASM module
5. ✅ **WASM Initialization** — ioquake3 engine starts, VFS initializes
6. ✅ **vLAN Relay Integration** — WebSocket connection established and stable
7. ✅ **Peer Status** — Client joins as peer 1 with host role assigned

Console output confirms:
- "Connected to relay"
- "Joined as peer 1 (host: true)"
- "Game module initialized"
- ioquake3 startup messages (ioq3 1.36_GIT_e4ce732d-2026-04-12)

## What's Left to Playable Game

### 1. **Rendering to Canvas**
The WASM engine is running but output isn't appearing. Need to:
- Verify WebGL context initialization in Emscripten
- Ensure the canvas element passed to the module is properly configured
- Check for any WebGL errors in the console
- May need to add requestAnimationFrame loop for rendering

### 2. **Input Handling**
Wire up keyboard, mouse, and gamepad events to the WASM engine:
- Key down/up events → engine's input system
- Mouse movement → engine's look/aim
- Mouse click → weapon fire / UI interaction
- Gamepad support (optional but nice for LAN parties)

### 3. **Network Shim (Most Complex)**
Intercept WASM socket calls and translate to WebSocket relay protocol:
- Emscripten provides socket emulation; hook into it or wrap at JS boundary
- Game sends UDP packets → translate to WebSocket frames
- Incoming WebSocket frames → translate back to fake UDP packets for game
- Implement host-client routing (0x00 to host, 0xFF broadcast, etc.)

This requires understanding:
- ioquake3's netchan packet format
- Emscripten's socket emulation API
- How to intercept/proxy socket operations from WASM

### 4. **Game Data / Config**
Currently the game complains about missing `default.cfg` and `baseq3/` directory. Options:
- Bundle full game assets (~100MB) — increases archive size significantly
- Generate minimal config that gets engine running
- Create a virtual filesystem stub that makes the engine think files exist
- Use data in VFS (virtual file system) that Emscripten provides

Current effort: ~1-2 days to get basic config working

## Effort Remaining to Playable

- **Rendering**: 1-2 days (likely WebGL setup issue)
- **Input**: 1-2 days (event wiring)
- **Network Shim**: 3-5 days (most complex, requires understanding ioquake3 netchan)
- **Game Config**: 1-2 days (minimal default.cfg or VFS stub)
- **Testing**: 1-2 days (LAN multiplayer validation)

**Total: ~7-13 days to fully playable OpenArena deathmatch**

Current blockers: None. Path forward is clear.

## Known Gotchas

1. **Memory Growth**: Ensure WASM module has `-sALLOW_MEMORY_GROWTH=1` flag
2. **Mouse Lock**: Browser Pointer Lock requires user gesture; ensure click-to-play
3. **Asset Size**: Game assets ~60-100 MB; download will take time (expected)
4. **Cross-Domain**: If catalog is on different domain, ensure CORS headers
5. **WebSocket Reliability**: Unlike raw UDP, WebSocket is ordered; this is good for LAN

## Timeline

- Get jdarpinian/ioq3 compiling: 1-2 days
- Create/adapt JS shim: 2-3 days  
- Integration testing: 2-3 days
- **Total: 5-8 days** (shorter than original estimate because jdarpinian fork is already proven)

## Files to Update After WASM Build

- `lpaas-98-games/catalog.json` — add real OpenArena entry
- `lpaas-98-games/ADDING_A_GAME.md` — add OpenArena worked example
- Create GitHub release in lpaas-98-games with tar.gz archive

## Deliverable

`lpaas98 install openarena && lpaas98 server` → server running  
Browser: Create room → 2+ browsers join → game packets exchange → READY

## Resources

- jdarpinian/ioq3: https://github.com/jdarpinian/ioq3
- The Longest Yard demo: https://thelongestyard.link
- Emscripten docs: https://emscripten.org/docs/
- Protocol specification: `docs/PROTOCOL.md`
