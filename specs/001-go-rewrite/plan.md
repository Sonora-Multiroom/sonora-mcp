# Implementation Plan: Rewrite sonora-mcp on the Shared Hub Client

**Branch**: `001-go-rewrite` | **Date**: 2026-09-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-go-rewrite/spec.md`

## Summary

Replace the Node.js/TypeScript MCP server with a Go server that keeps the same 24 tools, flags and
endpoints and delegates every hub call to sonora-cli's public `hub` package. The Go server adds an
error category on every failure, returns results as structured content plus JSON text, adds a `--host`
flag and a hub reachability field in `/health`, shuts down cleanly, and ships a systemd unit with an
install script for the Pi. Tools use the MCP Go SDK's typed `mcp.AddTool` (the SDK validates input and
builds structured results); we add a panic-recovery wrapper and a logging middleware. The error category
is a text prefix only ([research.md](research.md) R2). A conformance test checks every tool against the spec embedded in
sonora-cli (`api.Spec`). All Node.js files are removed at the end.

## Technical Context

**Language/Version**: Go 1.27 (same as sonora-cli `go.mod`)

**Primary Dependencies**: `github.com/modelcontextprotocol/go-sdk` v1.8.0 (MCP server, Streamable HTTP,
JSON Schema via `github.com/google/jsonschema-go`); `github.com/Sonora-Multiroom/sonora-cli` (`hub`,
`api` packages); standard library otherwise (`net/http`, `flag`, `log/slog`, `os/signal`)

**Storage**: N/A (stateless proxy)

**Testing**: `go test` with `net/http/httptest` fake hub; MCP client over `mcp.NewInMemoryTransports`
for tool tests; `mcp.StreamableClientTransport` for one end-to-end HTTP test

**Target Platform**: Windows amd64 (development), Linux arm64 on Raspberry Pi with systemd (production);
static binary, `CGO_ENABLED=0`

**Project Type**: single-binary network service (MCP server over HTTP)

**Performance Goals**: hub latency dominates; server overhead per call negligible. `/health` answers
within 2 s even with the hub down; every tool returns within the 5 s hub timeout (SC-003 bound 10 s)

**Constraints**: stateless (no sessions, no cached hub state); every hub call through `hub`, with the
request context and a timeout; LAN-only, unauthenticated; tool names and inputs unchanged

**Scale/Scope**: 24 tools, one household, a handful of concurrent agents; ~1–1.5k lines of Go plus tests,
one systemd unit, one install script

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / rule | How the design complies | Status |
|---|---|---|
| I. Every hub call through `hub`; no local hub types | Tool handlers call `hub.*` functions only; results are `hub` types (lists wrapped in an envelope struct, not redefined) | Pass |
| I. Missing hub behaviour added to `hub` first | `CreateInputRequest` optional booleans fixed in sonora-cli before tagging (R7), not patched here | Pass (prerequisite task) |
| I. Conformance test against `api.Spec`; no local `openapi.json` | `internal/tools/conformance_test.go` (R4); `openapi.json` and `update-openapi.mjs` removed | Pass |
| II. Stable names, descriptions, schema validation, annotations | 24 names from the inventory; descriptions per tool and field; schemas validated before handler; annotations per Kind ([data-model.md](data-model.md)) | Pass |
| II. MAJOR bump on tool-surface breaks | No rename, removal or incompatible input change; release 1.1.0 (R11) | Pass |
| III. No cached state; stateless transport; explicit hub URL | Shared `*mcp.Server`, `Stateless: true`; `--multiroom-url` required; health check does one live call, stores nothing | Pass |
| IV. Timeout via `hub.NewClient`; request context passed | One `hub.NewClient()` client; handlers receive the MCP request context; `PropagateRequestCancellation` (R6) | Pass (older-protocol clients: timeout bound only, R6) |
| IV. Errors as `isError` with category; no crash; per-call log | Handlers map `hub.ClassifyError` + pre-checks to a category-prefixed error; a wrapper recovers panics; a middleware logs every call (R2, R5, R10) | Pass |
| V. Test-first; MCP-layer scope; fake hub; offline | Tests written before each tool/component; `httptest` fake hub; conformance test in `go test ./...` (R14) | Pass |
| VI. Sanctioned dependencies only | Stdlib + MCP Go SDK + sonora-cli. SDK's transitive modules noted with binary size in the PR (R1) | Pass |
| Constraints: Go 1.27, module path, ldflags version, platforms, `CGO_ENABLED=0` | As in Technical Context; version single-sourced (R11) | Pass |
| Constraints: pinning (tagged sonora-cli on `main`), `go.work` uncommitted, gate with `GOWORK=off` | Final task pins the tag; `.gitignore` adds `go.work*` (R15) | Pass (merge blocked until sonora-cli is tagged, by design) |
| Constraints: shell scripts only as optional conveniences | `install.sh` is Pi deployment only; build and Windows run need none (FR-017c) | Pass |
| Constraints: LAN only; v2 only; legacy Node frozen | No auth or exposure beyond `--host`; v2 operations only; Node removed, not modified | Pass |
| Workflow: gofmt/vet/build/test gate; Conventional Commits; README sync | Gate in [quickstart.md](quickstart.md) §2; README rewritten for Go | Pass |

**Post-design re-check (after Phase 1)**: no new violations. Two recorded limitations, neither a
violation: (1) cancellation reaches the hub call only for clients on MCP protocol ≥ 2026-07-28; older
clients rely on the 5 s timeout; (2) `main` cannot be merged until sonora-cli is tagged with the
`hub/` package and the R7 fix.

## Project Structure

### Documentation (this feature)

```text
specs/001-go-rewrite/
├── plan.md              # This file
├── research.md          # Phase 0: SDK/hub findings and decisions
├── data-model.md        # Phase 1: tool, result, error, health, config entities
├── quickstart.md        # Phase 1: validation guide
├── contracts/
│   ├── tools.md         # 24 tools: operation, inputs, kind, result; error categories
│   └── server.md        # flags, exit codes, /mcp, /health, build, Pi service
├── checklists/requirements.md   # spec quality checklist (git-ignored)
└── tasks.md             # Phase 2 (/speckit-tasks)
```

### Source Code (repository root)

```text
go.mod                         # module github.com/tiger-seo/sonora-mcp; go 1.27
go.sum
cmd/sonora-mcp/
├── main.go                    # parse config, build server, run until signal, exit codes
└── main_test.go               # run(args, stdout, stderr) exit codes; single version source
internal/
├── config/
│   ├── config.go              # flag parsing + validation (--multiroom-url, --port, --host), usage
│   └── config_test.go
├── version/
│   └── version.go             # Version = "dev", set by -ldflags
├── tools/
│   ├── register.go            # add[In, Out]: explicit input schema + mcp.AddTool + panic recovery
│   ├── logging.go             # receiving middleware: one log line per tools/call
│   ├── errors.go              # hub error → category, error result builder
│   ├── schema.go              # constraint helpers (range, enum, minLength, pattern)
│   ├── inputs.go              # listInputs, getInput, createInput, deleteInput, setInputEnabled
│   ├── outputs.go             # listOutputs, getOutput, setOutputVolume, setOutputMute, setOutputEnabled
│   ├── groups.go              # listGroups, getGroup, setGroupVolume, setGroupMute, setGroupEnabled
│   ├── routes.go              # listRoutes, getRoute, createRoute, deleteRoute, transferRoute, setRoutePause
│   ├── playback.go            # playback, getMasterMute, setMasterMute
│   ├── tools.go               # Register(server, hubClient, hubURL): all 24 (logging is middleware)
│   ├── fakehub_test.go        # httptest fake hub recording requests, canned responses
│   ├── harness_test.go        # in-memory MCP client session + callTool/toolByName helpers
│   ├── *_test.go              # per-file tool tests through an in-memory MCP client
│   ├── conformance_test.go    # tools vs api.Spec (inputs and responses); tool-name inventory
│   └── architecture_test.go   # no direct net/http hub calls, no package-level hub state
└── server/
    ├── server.go              # Handler(s, client, hubURL): /mcp (stateless), /health, 404; Run(ctx, srv) graceful shutdown
    ├── health.go              # /health with 2 s hub check
    ├── server_test.go         # end-to-end over httptest with StreamableClientTransport
    ├── health_test.go         # /health reachable / unreachable within 2 s
    └── shutdown_test.go       # in-flight call drains, Run returns nil
deploy/pi/
└── install.sh                 # idempotent installer (bash): embeds the systemd unit as a
                                # heredoc and a pinned release tag; downloads the matching
                                # linux/arm64 binary from GitHub Releases at install time, so
                                # this is the only file copied to the Pi
README.md                      # rewritten for Go: build, cross-build, run, Pi install, client config
.gitignore                     # Go/binary ignores, go.work*, Spec Kit rules; Node rules removed
```

Removed: `src/`, `dist/`, `package.json`, `package-lock.json`, `tsconfig.json`, `openapi.json`,
`scripts/update-openapi.mjs`, `specs/sonora-mcp-plan.md` (Node-era plan), `run.sh`/`run.dev.sh`
(untracked one-line `npm` launchers, not replaced), `node_modules/` (untracked).

**Structure Decision**: single Go module with one binary under `cmd/sonora-mcp` and private packages
under `internal/`, the same layout sonora-cli uses. `internal/tools` holds all MCP-specific mapping so it
can be tested in isolation from HTTP; `internal/server` owns HTTP concerns only.

## Implementation Order (input for /speckit-tasks)

1. ~~**Prerequisite in sonora-cli (PR #19)**: `CreateInputRequest.Enabled`/`AutoRemove` → `*bool` +
   `omitempty`~~ — done in sonora-cli `3ee7369`.
2. **Scaffold**: `go.mod` (placeholder sonora-cli requirement), local `go.work`, `internal/version`,
   `.gitignore` update.
3. **Config**: tests then flags/validation/usage.
4. **Tool plumbing**: tests then `register.go`, `errors.go`, `schema.go`, `logging.go` (constraints,
   categories, panic recovery, logging).
5. **Tools by user story**: read-only tools (US1) → control tools (US2) → routing, playback, inputs
   (US3); each tool test-first against the fake hub.
6. **Conformance test** (US6) as soon as the first tool exists; it grows with each tool.
7. **Server**: `/mcp`, `/health`, graceful shutdown, `main.go` (US4/US5).
8. **Pi deployment**: `install.sh` with the embedded unit; publish the cross-built binary as a
   GitHub **prerelease** (`v1.1.0-rc.1`) and validate on the Pi with `--version` (quickstart §8).
9. **Remove Node and update README** (US6).
10. **Pin**: after sonora-cli is tagged, `go get …@<tag>`, gate with `GOWORK=off` on Windows and
    Linux.
11. **Release**: bake `RELEASE_TAG="v1.1.0"` into `install.sh` in the PR; after merge, tag `v1.1.0`
    on `main` and attach the cross-built binary to that release.

## Complexity Tracking

No constitution violations to justify.
