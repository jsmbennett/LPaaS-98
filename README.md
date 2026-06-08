# LPaaS 98

**LAN Party as a Service.**

A self-hosted server for running vintage LAN parties through the web browser. Drop the binary on a laptop, pick which games you want from the catalog, and your friends connect over the local network to play classics together — no installs, no port forwarding, no driver hell.

> Enterprise-grade infrastructure for non-enterprise activities.

## Launch Game

v1 launches with **OpenArena**, a free Quake III-style arena shooter:

- **OpenArena** (ioquake3 engine, GPL) — **No assets required.** Pure WASM-in-browser; one command, instant deathmatch.

The initial implementation focuses on this single game to validate the WASM-to-vLAN-relay architecture end-to-end before expanding to additional titles.

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

That's it. OpenArena ships with everything included — no additional assets needed.

## License

Apache-2.0 for the server. OpenArena and ioquake3 are GPL-licensed.
