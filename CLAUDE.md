# CLAUDE.md

Context for Claude (or any AI coding assistant) working on this repository. Read this at the start of every session.

## What this project is

**LPaaS 98 — LAN Party as a Service.** A self-hosted Go server that runs vintage LAN parties through the web browser. A host runs the binary on their laptop; guests connect via local IP, see a Win98-styled desktop, pick a game, play. Vintage games are integrated via one of three architectural patterns (see ARCHITECTURE.md): pure-WASM with a vLAN relay (OpenArena, Quakespasm), or native subprocess server + WASM client + UDP proxy (Zandronum).

The full design lives in `docs/ARCHITECTURE.md`. The catalog/installer system lives in `docs/CATALOG.md`. The build plan lives in `PLAN.md`. Read the relevant doc before answering substantive questions; don't infer from filenames.

## The two repos

- **`lpaas-98`** — this repo. The server.
- **`lpaas-98-games`** — the catalog and prebuilt WASM game archives.

When the user mentions "the catalog" or "the games repo," they mean the second one. The server fetches `catalog.json` from a stable URL there and downloads archives from its GitHub releases.

## Launch games (v1)

Three games, deliberately chosen to exercise the architecture:
1. **OpenArena** — Pattern A: self-contained pure-WASM (no host-supplied assets), host-client networking. Launch hero because `install openarena` and you're playing.
2. **Zandronum + Freedoom** — Pattern C: native server subprocess + WASM client + UDP proxy. Inspired by dorch/gib.gg. Best Doom multiplayer experience.
3. **Quakespasm + LibreQuake** — Pattern B: engine-only pure-WASM with host-supplied or free-bundle assets, host-client networking.

If something works for these three, it generalizes. If a proposed change breaks one of these three, it's the wrong change.

## Conventions (please follow)

### No magic
Prefer boring, readable code over clever code. Stdlib first. Hand-written code is fine even when a library exists, if the library adds dependency weight disproportionate to the benefit. Specifically:
- `net/http` for the HTTP server, not a framework.
- `log/slog` for logging, not zap or zerolog.
- Standard `encoding/json` for parsing, not faster alternatives.
- Hand-rolled subcommand dispatch is fine; cobra/urfave-cli are overkill for the CLI surface we have.
- No ORMs. No dependency injection frameworks. No code generation unless it's `go generate` for something genuinely repetitive.

If you're tempted to reach for a library, ask first.

### No premature abstraction
We have three games at launch and a small handful of post-v1 candidates. Don't design for fifty games. The per-game adapter pattern is the *only* abstraction we've committed to. Don't introduce others (plugin systems, event buses, generic "Manager" types) without an explicit discussion.

### Small surface area
Every public function, every API endpoint, every CLI flag is a maintenance commitment. Default to keeping things unexported / private / undocumented until they need to be otherwise.

### Static binary, always
`CGO_ENABLED=0`. No cgo dependencies, ever. This is a hard constraint — the whole point is a single binary that runs on any laptop without setup. If a proposed dependency requires cgo, find another way or push back to the user.

### Tests live alongside code
For any non-trivial function (parsing, hashing, file manipulation, HTTP handlers, WebSocket relay logic), write a test. Co-located `_test.go` files. Table-driven where it makes sense. Use the stdlib `testing` package; no testify, no ginkgo.

### Error handling
Wrap errors with context using `fmt.Errorf("doing X: %w", err)`. Don't swallow errors. Don't `log.Fatal` from library code — return errors, let the caller decide.

## What NOT to propose

Things that have come up and been decided against, or that conflict with the project's goals:

- **No TLS in v1.** HTTP only on the LAN. Don't suggest HTTPS, Let's Encrypt, or self-signed certs. It's in the post-v1 list.
- **No user accounts.** Nickname-only, client-side localStorage. Don't propose auth, sessions beyond ephemeral lobby state, or persistent identity.
- **No telemetry.** The server makes one kind of outbound request: fetching the catalog. Nothing else. No metrics endpoints, no error reporting services, no usage analytics.
- **No bundling commercial assets.** Ever. The host always supplies their own (`doom.wad`, `pak0.pak`) or explicitly opts into a free alternative (Freedoom, LibreQuake).
- **No silently substituting free assets for host-supplied ones.** The `.source` marker mechanism exists to prevent this. Respect it.
- **No databases.** State is in-memory; restart wipes lobbies and rooms. Catalog cache is a single JSON file. Don't propose SQLite, BoltDB, etc.
- **No web frameworks** (React, Vue, Svelte) in v1. The frontend is plain HTML + vanilla JS until Phase 11 (Win98 UI), and even then `98.css` + minimal JS, not a framework.
- **No protobuf / gRPC.** WebSocket frame format is being decided in Phase 6; the options on the table are JSON, MessagePack, or hand-rolled binary. Don't introduce protobuf unless explicitly asked.
- **No dynamic plugin loading.** Per-game Go adapters are compiled in. WASM/JS shims live in the catalog archives, not loaded as Go plugins.

## What to do when you're uncertain

- **Ask before adding a dependency.** Even a small one.
- **Ask before changing the manifest or catalog schema.** Other docs reference these; changes ripple.
- **Ask before changing CLI surface.** Every flag and subcommand is documented in `ARCHITECTURE.md`.
- **Surface tradeoffs explicitly.** Don't pick silently between two reasonable options; lay them out and let the user decide.
- **Don't refactor unprompted.** If you see something you'd structure differently, mention it but don't change it as a side effect of an unrelated task.

## Decision log

When we make a non-obvious decision in a session, write it to `docs/DECISIONS.md` as a short entry: date, decision, alternatives considered, reason. This is the single most important thing for long-horizon continuity — without it, future sessions will propose changes that contradict earlier choices.

Format:
```
## YYYY-MM-DD: Short title

**Decision:** What we chose.
**Alternatives:** What we considered.
**Reason:** Why we chose this.
**Revisit when:** What would make us reconsider.
```

## Session start

At the beginning of any session, read these files in this order:
1. This file (`CLAUDE.md`) — conventions and guardrails
2. `DECISIONS.md` — the log of non-obvious choices we've made and why
3. The relevant doc in `docs/` for whatever task is in front of you:
   - Architecture / system-wide changes → `docs/ARCHITECTURE.md`
   - Catalog format, manifest schema, installer behavior → `docs/CATALOG.md`
   - WebSocket protocol, vLAN frame format → `docs/PROTOCOL.md`
4. `PLAN.md` — to confirm which phase the work belongs to and what's expected of it

This is the single most important habit. Skipping it leads to proposals that contradict earlier decisions or duplicate work already planned.

## Workflow (Claude Code specifics)

Claude Code can read, edit, and run commands. With that power:

- **Run tests after every non-trivial change.** `go test ./...` is fast; use it.
- **Run `go vet` and `gofmt`** before considering work done.
- **Don't `git commit` unprompted.** The user reviews diffs before commits. Stage changes if helpful (`git add -p`), but commits are the user's call.
- **Don't push to remote unprompted.** Same reason.
- **When editing multiple files**, do them in one logical batch with a clear summary at the end, not as a stream of one-file diffs.
- **If a command fails**, report the actual error before proposing a fix. Don't silently retry with adjustments.

## Workflow expectations

- **Small diffs.** If a proposed change is over ~200 lines, break it into reviewable chunks.
- **Tests in the same change.** Code without tests in the same diff is incomplete.
- **No drive-by changes.** If you're asked to fix X, fix X. Don't reformat unrelated files or "improve" things while you're there.
- **Read before writing.** When asked to modify a file, view it first. When asked to add a feature, view the adjacent code first.
- **Explain surprises.** If your proposed code does something subtle, say so in the response, not just in a code comment.

## What the user is good at, and where they want your help

The user is comfortable with Go and the architectural level of the project. Where Claude is most useful:
- Boilerplate generation (HTTP handlers, parsers, installers, CLI plumbing)
- Cross-file consistency checks
- Test writing
- Documentation polishing
- Working through subtle design tradeoffs

Where Claude is *not* useful and shouldn't pretend to be:
- The novel parts of Phases 7-9 (WASM compilation, JS shim writing, browser-side game integration). These require reading specific game source, iterating against a real browser, and understanding undocumented quirks. Claude can help reason about *what* the shim needs to do, but the actual integration loop is the user's.
- Anything that requires knowing the current state of upstream projects (which Doom port is most maintained, whether someone has already published a usable ioquake3 WASM build). Claude doesn't have current information; the user should check upstream directly before committing to specific dependencies.

## Tone

Direct. Honest about uncertainty. Push back when something seems wrong rather than going along with it. The user wants a collaborator, not a yes-machine.
