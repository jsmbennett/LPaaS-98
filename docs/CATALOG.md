# Catalog Format

LPaaS 98 servers install games by downloading prebuilt WASM archives from a remote catalog. This document defines the catalog format, the relationship between the server and the catalog repo, and the conventions for archives.

## Repositories

- **`lpaas-98`** — the server. Knows how to read catalogs and install from them.
- **`lpaas-98-games`** — the default catalog. Hosts `catalog.json` and the archives as GitHub release assets.

Anyone can fork `lpaas-98-games` and run their own catalog. Server operators point at an alternative with `--catalog-url`.

## Two Kinds of Catalog Entries

The catalog contains two kinds of entries:

1. **Games** (`kind: "game"`) — anything you `install`. Each game declares a `pattern`: `"A"` (self-contained pure-WASM, e.g. OpenArena), `"B"` (engine-only pure-WASM with separate assets, e.g. Quakespasm), or `"C"` (native server subprocess + WASM client + UDP proxy, e.g. Zandronum).
2. **Asset bundles** (`kind: "asset-bundle"`) — opt-in free game assets (Freedoom, LibreQuake). Installed via the explicit `install-free-assets` command into `./assets/<game-id>/`.

A game manifest's `requires_assets[]` entry may name a `free_alternative` referencing an asset bundle's `id`. Games with empty `requires_assets` (only Pattern A in the launch lineup) have no `install-free-assets` flow.

## `catalog.json` (Launch Lineup)

```json
{
  "schema_version": 1,
  "updated": "2026-04-12T18:00:00Z",
  "entries": [
    {
      "kind": "game",
      "id": "openarena",
      "name": "OpenArena",
      "description": "Free Quake III-style arena FPS. Self-contained.",
      "categories": ["fps", "arena", "self-contained"],
      "versions": [
        {
          "version": "0.8.8",
          "released": "2026-04-01",
          "pattern": "A",
          "url": "https://github.com/<owner>/lpaas-98-games/releases/download/openarena-0.8.8/openarena-0.8.8.tar.gz",
          "sha256": "...",
          "size_bytes": 320000000,
          "license": "GPL-2.0",
          "source_url": "https://github.com/OpenArena/engine",
          "min_server_version": "0.2.0",
          "requires_assets": []
        }
      ]
    },
    {
      "kind": "game",
      "id": "zandronum",
      "name": "DOOM (Zandronum)",
      "description": "Modern multiplayer Doom port. Native server subprocess with WASM client.",
      "categories": ["fps", "classic", "multiplayer-focused"],
      "versions": [
        {
          "version": "3.2-alpha",
          "released": "2026-03-20",
          "pattern": "C",
          "url": "https://github.com/<owner>/lpaas-98-games/releases/download/zandronum-3.2-alpha/zandronum-3.2-alpha.tar.gz",
          "sha256": "...",
          "size_bytes": 120000000,
          "license": "GPL-3.0",
          "source_url": "https://zandronum.com/",
          "min_server_version": "0.4.0",
          "supported_platforms": [
            "linux-amd64", "linux-arm64",
            "darwin-amd64", "darwin-arm64",
            "windows-amd64"
          ],
          "requires_assets": [
            {
              "filename": "doom.wad",
              "description": "DOOM IWAD",
              "free_alternative": "freedoom-phase1"
            }
          ]
        }
      ]
    },
    {
      "kind": "asset-bundle",
      "id": "freedoom-phase1",
      "name": "Freedoom Phase 1",
      "description": "Free Doom-compatible game data under the BSD license",
      "target_games": ["zandronum"],
      "versions": [
        {
          "version": "0.13.0",
          "released": "2025-09-01",
          "url": "https://github.com/<owner>/lpaas-98-games/releases/download/freedoom-phase1-0.13.0/freedoom-phase1-0.13.0.tar.gz",
          "sha256": "...",
          "size_bytes": 25600000,
          "license": "BSD-3-Clause",
          "source_url": "https://freedoom.github.io/",
          "installs_to": {
            "zandronum": [
              { "from": "freedoom1.wad", "to": "doom.wad" }
            ]
          }
        }
      ]
    },
    {
      "kind": "game",
      "id": "quakespasm",
      "name": "Quake (Quakespasm)",
      "description": "Modern Quake source port. Requires Quake PAK files.",
      "categories": ["fps", "classic"],
      "versions": [
        {
          "version": "0.96.2",
          "released": "2026-03-20",
          "pattern": "B",
          "url": "https://github.com/<owner>/lpaas-98-games/releases/download/quakespasm-0.96.2/quakespasm-0.96.2.tar.gz",
          "sha256": "...",
          "size_bytes": 5100000,
          "license": "GPL-2.0",
          "source_url": "https://quakespasm.sourceforge.net/",
          "min_server_version": "0.2.0",
          "requires_assets": [
            {
              "filename": "id1/pak0.pak",
              "description": "Quake shareware or full PAK0",
              "free_alternative": "librequake"
            }
          ]
        }
      ]
    },
    {
      "kind": "asset-bundle",
      "id": "librequake",
      "name": "LibreQuake",
      "description": "Free Quake-compatible game assets under CC-BY-SA",
      "target_games": ["quakespasm"],
      "versions": [
        {
          "version": "0.10.0",
          "released": "2025-10-15",
          "url": "https://github.com/<owner>/lpaas-98-games/releases/download/librequake-0.10.0/librequake-0.10.0.tar.gz",
          "sha256": "...",
          "size_bytes": 180000000,
          "license": "CC-BY-SA-4.0",
          "source_url": "https://librequake.org/",
          "installs_to": {
            "quakespasm": [
              { "from": "lq1/pak0.pak", "to": "id1/pak0.pak" }
            ]
          }
        }
      ]
    }
  ]
}
```

### Game entry fields
- `kind`: `"game"`
- `id`, `name`, `description`, `categories`
- `versions[]`: with `version`, `released`, `pattern` (`"A"`, `"B"`, or `"C"`), `url`, `sha256`, `size_bytes`, `license`, `source_url`, `min_server_version`, `requires_assets[]`, and (for Pattern C) `supported_platforms[]`
- `pattern`: required. `A` = self-contained pure-WASM, `B` = engine-only pure-WASM with separate assets, `C` = native server subprocess + WASM client + UDP proxy.
- `requires_assets`: array, possibly empty. Empty is only valid for Pattern A; for Pattern B and C, must list the assets the game expects.
- `requires_assets[].free_alternative`: optional; ID of an asset bundle that satisfies this requirement
- `supported_platforms[]` (Pattern C only): list of `<os>-<arch>` strings (e.g. `linux-amd64`, `darwin-arm64`) for which the archive includes a server binary. The server refuses to install a Pattern C game whose `supported_platforms` doesn't include the host's platform.

### Asset-bundle entry fields
- `kind`: `"asset-bundle"`
- `id`, `name`, `description`
- `target_games[]`: which game IDs this bundle is compatible with
- `versions[]`: with `url`, `sha256`, `size_bytes`, `license`, `source_url`, `installs_to`
- `installs_to`: map from game `id` → list of `{from, to}` pairs describing how archive files become asset files

## Game Archive Format

A `.tar.gz` containing one game's files. Root contains, at minimum:

```
manifest.json
<game>.wasm
loader.js
icon_16.png
icon_32.png
```

**For Pattern A games** (self-contained, e.g. OpenArena), the archive also contains the game's own data directories:

```
manifest.json
openarena.wasm
loader.js
icon_16.png
icon_32.png
baseoa/
  pak0.pk3
  pak1-maps.pk3
```

**For Pattern B games** (engine-only, e.g. Quakespasm), no asset data is included; the host or `install-free-assets` provides it.

**For Pattern C games** (subprocess server + WASM client, e.g. Zandronum), the archive additionally contains per-platform native server binaries:

```
manifest.json
zandronum-client.wasm
loader.js
icon_16.png
icon_32.png
server/
  linux-amd64/zandronum-server
  linux-arm64/zandronum-server
  darwin-amd64/zandronum-server
  darwin-arm64/zandronum-server
  windows-amd64/zandronum-server.exe
```

Server binaries must be statically linked where possible to avoid host-side dependency issues. On macOS, binaries should be signed and notarized (handled by the games-repo release pipeline). The executable bit on Pattern C server binaries is the **only** exception to the "no executable bits in archives" rule; the installer chmods them to 0755 on extraction.

**Archives must not contain:**
- Commercial assets the project doesn't have rights to redistribute
- Absolute paths or directory escapes (`../`)
- Executable bits, except for the `server/<platform>/<binary>` files in Pattern C archives

**Naming convention:** `<id>-<version>.tar.gz` — e.g. `openarena-0.8.8.tar.gz`, `zandronum-3.2-alpha.tar.gz`.

**Size guidance:** Pattern A archives that include data are large (OpenArena ~300 MB). Pattern B archives are a few MB. Pattern C archives are moderate (Zandronum ~120 MB with all five server binaries plus the WASM client). The catalog's `size_bytes` field is informational only — the installer streams the download and reports progress.

## Asset-Bundle Archive Format

A `.tar.gz` containing free assets. Layout is bundle-specific but typically mirrors what the engine expects:

```
# freedoom-phase1-0.13.0.tar.gz
freedoom1.wad
LICENSE
README.md
```

```
# librequake-0.10.0.tar.gz
lq1/pak0.pak
lq1/pak1.pak
LICENSE
README.md
```

The `installs_to` map in the catalog translates archive paths to `assets/<game-id>/` paths.

`LICENSE` and `README.md` are **required** in asset bundles and are preserved at `assets/<game-id>/_bundle-info/` after install.

## Install Flow (Games)

1. Read cached `catalog.json` (fetching if `--catalog-update-on-install`).
2. Resolve `<game-id>` or `<game-id>@<version>` to a specific version of `kind: game`.
3. Check `min_server_version`. Abort if incompatible.
4. Download archive to a temp file with progress.
5. Verify SHA256. Abort on mismatch.
6. Unpack to a temp directory, validate `manifest.json` matches catalog metadata.
7. Atomic rename to `./games/<game-id>/`, replacing any prior install.
8. Registry picks up the new game on next scan / SIGHUP.

Self-contained games (empty `requires_assets`) are immediately launchable after this. Engine-only games show as "needs assets" until either the host drops files into `./assets/<game-id>/` or runs `install-free-assets`.

## Install Flow (Free Assets)

1. Resolve `<game-id>` to its game entry. If `requires_assets` is empty, print "this game is self-contained; nothing to install" and exit 0.
2. Find each `requires_assets[].free_alternative`; collect the asset bundles to install.
3. **Show the host:** bundle name, version, license, source URL, total download size. Prompt for confirmation (unless `--yes`).
4. If `assets/<game-id>/.source` exists and says `host-supplied`, refuse without `--force`.
5. Download, verify SHA256, unpack to temp.
6. Apply the `installs_to[<game-id>]` mapping: copy each `{from, to}` from the archive into `assets/<game-id>/<to>`.
7. Write `assets/<game-id>/.source` = `<bundle-id> <bundle-version>`.
8. Copy bundle's `LICENSE` and `README.md` to `assets/<game-id>/_bundle-info/`.

## Uninstall

- `lpaas98 uninstall <game-id>`: removes `./games/<game-id>/`. Leaves `./assets/<game-id>/` alone.
- `lpaas98 uninstall-assets <game-id>`: removes `./assets/<game-id>/`. Refuses without `--force` if `.source` says `host-supplied`.

## Update

`lpaas98 update` walks installed games, fetches catalog, and updates any with newer versions in the catalog. Free-asset bundles are updated alongside if newer versions exist.

Pinned versions are stored in `./games/<game-id>/.pinned` and skipped.

## Forking and Custom Catalogs

To run a private catalog:

1. Fork `lpaas-98-games`.
2. Edit `catalog.json`, upload archives as releases.
3. Point your server at it: `lpaas98 --catalog-url https://raw.githubusercontent.com/<you>/<fork>/main/catalog.json install <game>`.

## Signature Verification (Future)

v1 trusts the SHA256 in `catalog.json`, which means trusting the catalog URL. A future schema version will add a detached signature over `catalog.json` (minisign or cosign), with the public key compiled into the server binary. The `schema_version` field is the migration lever.

## Schema Evolution

- Additive changes (new optional fields) don't bump `schema_version`. Clients ignore unknown fields.
- Breaking changes bump it. Old clients refuse catalogs with unknown schema versions.
- `min_server_version` per-version lets new game versions safely coexist with older servers.
