# Agent Instructions

## Product naming

- The product is **Sonora Multiroom**. This repo is **sonora-mcp**, the MCP server for it.
- The backend it talks to is the **Multiroom Audio Hub API**. Its OpenAPI spec is published by
  sonora-cli as `api.Spec`; there is no copy in this repo.

## Git / PR conventions

- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).
- Never add a `Co-Authored-By` trailer (or similar attribution) to commit messages, and never add
  "Generated with Claude Code" (or similar) to PR descriptions.
- Module path is `github.com/Sonora-Multiroom/sonora-mcp`. The version ldflag must use it exactly
  (`-X github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=<v>`): Go silently ignores
  `-X` for a package that isn't in the build, and the binary then reports `dev`.

## Project context

- Governance and engineering principles live in `.specify/memory/constitution.md`; read it before
  planning or implementing. Feature work follows the Spec Kit workflow
  (`specs/<NNN-feature>/spec.md` → `plan.md` → `tasks.md`), driven by the `/speckit-*` skills.
- All hub access goes through sonora-cli's `hub` package (`architecture_test.go` enforces it).
  When `hub` is wrong, fix it in sonora-cli first (PR, patch tag), then bump here
  (`go get github.com/Sonora-Multiroom/sonora-cli@vX.Y.Z`). Never work around it locally.
- `conformance_test.go` compares every tool's input and output schema with `api.Spec`. When it
  fails, suspect the code, not the test: it has caught real `hub` bugs (e.g. mute endpoints
  decoded as the wrong response type). Don't loosen it to make a change pass.

## Building and testing

- The gate: `gofmt -l .` (must be empty), `go vet ./...`, `go test ./...`. If you use a `go.work`
  (git-ignored) to develop against a local sonora-cli checkout, run the gate and release builds
  with `GOWORK=off`, so they use the sonora-cli version pinned in `go.mod`.
- Tests are offline: tools are driven through an in-memory MCP client against a fake hub
  (`internal/tools/fakehub_test.go`, `harness_test.go`).
- `go test -race` needs cgo, so run it on Linux (CI does this on every PR). Locally on Windows:
  `docker run --rm --network none -e GOWORK=off -e GOPROXY=off -e GOFLAGS=-mod=mod -v <repo>:/src
  -v <GOMODCACHE>:/go/pkg/mod -w /src golang:1.27 go test -race ./...`.

## Testing against the real hub

- It is a real home audio system (hub at `http://multiroom.lan:8080`, sonora-mcp as a service on
  the same Raspberry Pi). Read-only tools and requests that return 404 are fine to run.
- State-changing tools make sound or change speakers. Ask first, record the current state, and
  restore it right after (volume, mute, enabled, master mute; delete any routes and inputs you
  created). Don't touch routes you didn't create.
- Rebooting the Pi restarts the hub, which drops active routes. Right after boot `/health` reports
  `"hub":"unreachable"` for a few seconds; that's expected.

## Releases and the Pi

- Pushing a `vX.Y.Z` tag runs GoReleaser (`.goreleaser.yaml`, `.github/workflows/release.yml`).
  `-rc.N` tags are published as prereleases. `./release.sh` tags `main`; tag release candidates on
  a branch by hand.
- `deploy/pi/install.sh` is run as `curl … | sudo bash -s -- --hub-url <url>`. Keep all of its
  logic inside functions and `main "$@"` as the last line, so a cut-off download runs nothing.
  Bump its `RELEASE_TAG` in the commit that gets tagged. It downloads the release archive and
  verifies it against `checksums.txt`, so the GoReleaser archive naming
  (`sonora-mcp_<version>_linux_arm64.tar.gz`) is part of its contract.

## Active Technologies

- Go 1.27, `CGO_ENABLED=0` static builds (Windows amd64 for development, linux/arm64 on the Pi).
- `github.com/modelcontextprotocol/go-sdk` v1.8.0 (typed `mcp.AddTool`, stateless Streamable
  HTTP), with `github.com/google/jsonschema-go` for schemas.
- `github.com/Sonora-Multiroom/sonora-cli` v0.1.1 (`hub` client, `api.Spec`).
- Standard library otherwise: `net/http`, `flag`, `log/slog`, `os/signal`.
- GoReleaser and GitHub Actions for releases and PR checks; systemd on the Pi.

## Project Structure

- `cmd/sonora-mcp`: entry point (config, server, signal handling, exit codes).
- `internal/config`, `internal/version`: flags and the build version.
- `internal/tools`: the 24 tools, schemas, error categories, logging middleware, and the fake
  hub, conformance and architecture tests.
- `internal/server`: `/mcp`, `/health`, graceful shutdown.
- `deploy/pi/install.sh`: the Pi installer with the embedded systemd unit.
- `.specify/memory/`: the constitution plus the master `spec.md`, `plan.md` and `changelog.md`,
  which describe the current system. Merged features are archived under `.specify/archive/`;
  `specs/` holds only work in progress.

## Commands

- Gate: `gofmt -l .`, `go vet ./...`, `go test ./...` (with `GOWORK=off` when a `go.work` exists).
- Run: `go run ./cmd/sonora-mcp --multiroom-url http://multiroom.lan:8080 [--port 3001] [--host 127.0.0.1]`.
- Health: `curl http://localhost:3001/health`.
- Inspect tools: `npx @modelcontextprotocol/inspector`, then connect to `http://localhost:3001/mcp`.

## Recent Changes

- specs/001-go-rewrite: Go rewrite on sonora-cli's `hub` package, released as v1.1.0. Same 24
  tools with structured results and category-prefixed errors; `--host`, `/health` hub check,
  graceful shutdown; one-command Pi install; GoReleaser releases; Node.js server removed.

## Known Issues & Gotchas

### ⚠️ `jsonschema` struct tags carry only descriptions
**Issue:** Constraints such as ranges, enums, lengths and patterns can't be written as struct tags.
**Root Cause:** In `google/jsonschema-go` the `jsonschema` tag is only the description; tags
starting `WORD=` are rejected.
**Prevention Rule:** Set constraints on the inferred schema at registration (`internal/tools/schema.go`).

### ⚠️ The error category is only a text prefix
**Issue:** Errors can't carry the category in a structured field or `_meta`, and schema failures
use the SDK's `validating "arguments": ...` text without a prefix.
**Root Cause:** The typed `mcp.AddTool` discards any result a handler builds when it returns an
error, and the SDK validates input before the handler runs.
**Prevention Rule:** Keep `"<Category>: <message>"` as the contract; switch to the low-level
`Server.AddTool` only if a client needs a structured category.

### ⚠️ The SDK doesn't recover panics
**Issue:** A panic in a tool handler would crash the whole server.
**Root Cause:** The MCP Go SDK has no `recover()` around handlers.
**Prevention Rule:** Register every tool through the helper that wraps it in `recoverPanics`.

### ⚠️ Cancellation reaches the hub only for newer clients
**Issue:** For MCP protocol versions before 2026-07-28, a client disconnect doesn't cancel the
hub call.
**Root Cause:** `PropagateRequestCancellation` ties the handler context to the HTTP request only
for protocol ≥ 2026-07-28.
**Prevention Rule:** Keep the 5 s hub timeout as the bound, and keep the HTTP cancellation test
that fails if `PropagateRequestCancellation` is dropped.

### ⚠️ Loopback requests with a foreign `Host` get 403
**Issue:** Requests through 127.0.0.1 with a non-localhost `Host` header are rejected.
**Root Cause:** The SDK's DNS-rebinding protection is on by default.
**Prevention Rule:** Keep it on; LAN clients connecting via the Pi's LAN address are unaffected.

### ⚠️ Structured results must be objects
**Issue:** Returning a bare array as `structuredContent` breaks clients.
**Root Cause:** MCP clients before SEP-2106 require `structuredContent` to be an object.
**Prevention Rule:** Wrap lists (`{"outputs": [...]}`) and return `{"deleted": true, "<id>": ...}`
for no-body operations.

### ⚠️ Path IDs need `minLength: 1` although the spec doesn't say so
**Issue:** An empty path ID changes the route (`GET /api/v2/outputs/` is the list endpoint).
**Root Cause:** The hub spec types path parameters as plain strings.
**Prevention Rule:** Give path IDs `minLength: 1`; the conformance test compares path parameters
as if the spec declared it. Query and body parameters get no such allowance.

### ⚠️ Optional hub fields must be omitted, not zeroed
**Issue:** An agent omitting `createInput.enabled` would have created a disabled input.
**Root Cause:** `hub.CreateInputRequest.Enabled`/`AutoRemove` were plain `bool` without
`omitempty`, so `false` was sent instead of letting the hub apply its defaults.
**Prevention Rule:** Optional hub request fields are pointers with `omitempty`, fixed in
sonora-cli; never fill in hub defaults here (Principle III).

### ⚠️ Overwriting the running binary fails on the Pi
**Issue:** Copying over `/usr/local/bin/sonora-mcp` while the service runs fails with "Text file
busy".
**Root Cause:** Linux refuses writes to an executing file, but allows renaming over it.
**Prevention Rule:** Write to a temp file in `/usr/local/bin`, `chmod 0755`, then `mv -f` onto the
target.
