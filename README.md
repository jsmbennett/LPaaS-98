# LPaaS 98

**LAN Party as a Service.**

A self-hosted server for running vintage LAN parties through the web browser. Drop the binary on a laptop, pick which games you want from the catalog, and your friends connect over the local network to play classics together — no installs, no port forwarding, no driver hell.

> Enterprise-grade infrastructure for non-enterprise activities.

## Launch Lineup

v1 ships with three games, each demonstrating a different architectural pattern:

- **OpenArena** (ioquake3 engine, GPL) — a free Quake III-style arena shooter. **No assets required.** Pure WASM-in-browser; one command, instant deathmatch.
- **Zandronum** (Doom client/server port, GPL) — modern multiplayer Doom with RUDP networking, join-in-progress, and proper deathmatch/CTF/Survival modes. Bring your own `doom.wad`, or install **Freedoom** (free BSD-licensed game compatible with the Doom engine). Runs as a native server subprocess with a WASM client; LPaaS 98 acts as the UDP proxy. Inspired by [dorch / gib.gg](https://github.com/beebs-dev/dorch).
- **Quakespasm** (Quake engine, GPL) — bring your own `pak0.pak`, or install **LibreQuake** (free CC-BY-SA assets compatible with the Quake engine). Pure WASM-in-browser.

The lineup is designed to exercise the architecture:

- **OpenArena** = self-contained, pure-WASM, no assets, host-client networking
- **Zandronum** = subprocess server + WASM client + UDP proxy, host-supplied or free-bundle assets
- **Quakespasm** = engine-only pure-WASM, host-supplied or free-bundle assets, host-client networking

If everything works for these three, the system generalizes cleanly to almost any vintage open-source-engine game.

## Status

Pre-alpha. See [`PLAN.md`](./PLAN.md) for the build roadmap and [`ARCHITECTURE.md`](./ARCHITECTURE.md) for the system design.

## What it is

- A single static Go binary (Linux / macOS / Windows on amd64 and arm64)
- An HTTP server that hosts a Win98-styled web desktop where players join, see installed games, and launch them
- A virtual LAN relay that lets WebAssembly-compiled vintage games talk to each other as if they were on a real local network
- A catalog system that downloads precompiled WASM game builds from a separate GitHub-hosted repository, verified by SHA256
- Optional opt-in installer for free game assets (Freedoom, LibreQuake) for hosts who don't own the commercial originals
- No accounts, no TLS, no telemetry, no internet required after install

## What it is not (yet)

- Pretty
- TLS-enabled (HTTP only on LAN for v1; HTTPS comes later for VPS hosting)
- A redistributor of commercial game assets — host always supplies their own, or explicitly opts into a free alternative
- Stable

## Quick Start (eventual)

```
$ ./lpaas98 install openarena
$ ./lpaas98 server
LPaaS 98 listening on:
  http://192.168.1.47:9898
  http://10.0.0.12:9898
Tell your friends to open that in their browser.
```

That's it. The other two games take an extra step — either drop your own WAD/PAK into the assets directory, or run `lpaas98 install-free-assets zandronum` (or `quakespasm`) to grab the free alternatives.

## License

Apache-2.0 for the server. Game engine builds in the catalog retain their own licenses (mostly GPL). Free assets retain theirs (Freedoom: BSD-3-Clause; LibreQuake: CC-BY-SA-4.0). OpenArena ships under GPL with its own assets included.
