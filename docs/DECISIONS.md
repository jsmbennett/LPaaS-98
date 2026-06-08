# Decisions Log

## 2026-06-08: OpenArena WASM Approach — Use jdarpinian/ioq3 Fork

**Decision:** Use jdarpinian/ioq3 (Emscripten port) as the starting point for OpenArena WASM integration. Do not build from scratch.

**Alternatives:** 
1. Build ioquake3 from source with Emscripten (3-4 weeks, not recommended)
2. Use OpenArena Live's WebRTC implementation (simpler peer-to-peer, but incompatible with our relay architecture)
3. Use older klaussilveira/ioquake3.js (works but less maintained)

**Reason:** 
- jdarpinian/ioq3 is actively maintained (June 2024 commits)
- Already proven in production (thelongestyard.link is live)
- WebRTC support demonstrates understanding of modern browser constraints
- Clear path to adapt for LPaaS 98: fork, remove WebRTC code, add WebSocket shim

**Revisit when:** 
- jdarpinian/ioq3 is abandoned (no commits >6 months)
- Our integration hits unsolvable blockers in the existing codebase
- Performance requirements change (e.g., mobile support needed)

**Phase 7 Effort Estimate:** 10-15 days (2-3 weeks)
- Get existing binary running single-player: 2-3 days
- JS shim for WebSocket networking: 3-5 days  
- Multi-player testing & bugfixes: 5-7 days

**Key Technical Notes:**
- Network layer: Emscripten socket emulation + JS shim → WebSocket → our vLAN relay
- WASM binary size: 8-15 MB; game assets: 60-100 MB (total archive ~70-120 MB)
- Performance: 30-60 FPS typical; WebSocket adds ~5-10ms vs. raw UDP (acceptable for LAN)
- Dependencies (SDL2, OpenGL ES, OpenAL) all Emscripten-compatible

**Action Items Before Phase 7:**
1. Clone jdarpinian/ioq3, build locally, verify it runs in browser
2. Read their JS networking shim; understand UDP→WebSocket translation
3. Design our JS layer to convert those packets to PROTOCOL.md frame format
4. Prototype single-player with our relay before debugging multiplayer
