# Phase 7 Notes: OpenArena End-to-End

## Status

**Infrastructure: Complete**
**WASM Build: Pending**
**Integration Testing: Pending**

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

## What's Left

### 1. Build WASM Binary
- Clone https://github.com/jdarpinian/ioq3
- Configure Emscripten build flags
- Compile ioquake3 to WASM
- Output: `openarena.wasm`

Typical Emscripten flags:
```bash
emconfigure ./configure --enable-wasm
emmake make
```

Binary will be ~8-15 MB depending on optimizations.

### 2. Create JavaScript Shim
- Intercept ioquake3's UDP socket calls
- Translate to GameLoader.sendGamePacket()
- Translate incoming relay packets back to what ioquake3 expects

This is the most game-specific work. Expect to read ioquake3's networking code and understand:
- Packet format (likely raw UDP datagrams)
- Socket emulation layer (Emscripten provides one)
- Address family usage (AF_INET, UDP)

Reference: jdarpinian/ioq3's existing JS shim (if they have one for WebRTC)

### 3. Bundle Game Data
- Copy OpenArena game data into `games/openarena/`
- Include: manifest.json, openarena.wasm, loader.js, icon files, game assets (baseoa/)
- Create tar.gz: `openarena-0.8.8.tar.gz`

File structure:
```
openarena-0.8.8/
  manifest.json
  openarena.wasm
  loader.js
  icon_16.png
  icon_32.png
  icon_48.png
  baseoa/
    pak0.pk3
    pak1-maps.pk3
    ...
```

### 4. Update Catalog
- Calculate SHA256 of tar.gz
- Add entry to `lpaas-98-games/catalog.json`
- Update download_url and sha256 fields

### 5. Test End-to-End
1. Build server: `go build ./cmd/lpaas98`
2. Install game: `./lpaas98 install openarena`
3. Start server: `./lpaas98 server`
4. Open browser: `http://localhost:9898`
5. Create room, join from another browser
6. Verify game loads and packet exchange works

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
