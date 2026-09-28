# Plan: Rewrite sonora-mcp in Go on a shared, public hub client

- **Status**: Proposed (not started)
- **Date**: 2026-09-27
- **Repos affected**: `sonora-cli` (phases 1–2), `sonora-mcp` (phase 3)
- **Decision**: Option 2a. Promote `sonora-cli/internal/hub` to a public package inside the
  sonora-cli module, and rewrite sonora-mcp as a separate Go module that imports it.

## Why

sonora-mcp is a ~900-line Node.js/TypeScript proxy. Its 24 hub calls are hand-written and have
no timeout or tests, and they cover 24 of the 49 operations in the hub spec. sonora-cli already
has a tested Go client for the same API (`internal/hub`, ~2,150 lines, standard library only),
with typed errors, request timeouts, and contract tests. It also covers TTS, extensions, and bulk
stop. Keeping two clients in two languages in step with every `openapi.json` change costs more
than a one-time rewrite. A Go binary also deploys to the Raspberry Pi without a Node runtime.

Options considered and rejected:

- **Stay on Node.js**: this leaves two clients in two languages to keep in sync with the spec.
- **Option 1, a second binary inside sonora-cli**: less work, but sonora-mcp would lose its own
  repo, README, releases, and constitution.
- **Option 2b, a separate `sonora-hub-go` repo**: a third repo and a three-step release for each
  API change. 2a can be split out into 2b later without changing the package API.

## Target layout

```
sonora-cli/                 module github.com/Sonora-Multiroom/sonora-cli   (public repo)
  hub/                      ← moved from internal/hub; public, semver-versioned
  api/openapi.json          the single copy of the hub spec
  api/spec.go               (recommended) //go:embed openapi.json → exported Spec []byte
  internal/cli/...          CLI keeps its commands; imports .../sonora-cli/hub
  internal/cli/exitcode/    ← new: ErrorClass → exit code mapping (moved out of hub)
  cmd/sonora/

sonora-mcp/                 module github.com/Sonora-Multiroom/sonora-mcp
  go.mod                    require github.com/Sonora-Multiroom/sonora-cli v0.x.y
  cmd/sonora-mcp/main.go    flags, HTTP server, /mcp and /health
  internal/tools/           one MCP tool per hub operation, each calling the hub package
```

---

## Phase 1: Amend the sonora-cli constitution

Run in sonora-cli: `/speckit-constitution` with the intent below. This is a MINOR bump
(1.1.3 → 1.2.0): a section is added and nothing is relaxed.

Proposed content (new section, or a new principle after II):

> **Public Hub Client Package**
> - `hub/` is a public Go package consumed by other Sonora projects (at minimum sonora-mcp).
>   Its exported API is a contract and follows the module's semver tags. While the module is
>   `v0.x`, breaking changes are allowed but MUST be called out in the release notes.
> - `hub/` MUST contain only hub protocol concerns: request/response types, HTTP calls, error
>   types, and error classification. CLI concerns (exit codes, rendering, flags, config
>   discovery) MUST live under `internal/`.
> - `hub/` MUST keep its standard-library-only dependency footprint (Principle III applies to
>   consumers too).
> - `hub/` exported identifiers MUST have godoc comments that make sense without sonora-cli's
>   specs. References such as "research.md §6" belong in internal code or commit history,
>   not in public docs.
> - Contract tests for `hub/` stay in sonora-cli and remain the conformance gate for
>   Principle II. Consumers test only their own mapping onto `hub/`.

Also check **Principle II** (API Contract Fidelity): it should name `hub/` as the one place
where the spec is turned into Go types for all Sonora Go consumers.

## Phase 2: Refactor sonora-cli

Run in sonora-cli: `/speckit-specify` (suggested name `010-public-hub-package`). It's a pure
refactor with no CLI behavior change. Everything below was checked against the repo as of
2026-09-27.

### 2.1 Rename the module path

- `go.mod`: `module sonora-cli` → `module github.com/Sonora-Multiroom/sonora-cli`.
- Rewrite every `"sonora-cli/...` import (111 files) to the new path.
- **Gotcha**: `Makefile` (2 places) and `build.sh` inject the version with
  `-ldflags "-X sonora-cli/internal/version.Version=..."`. If these aren't updated, the build
  still succeeds but the version stays at its default. Update them and add a test (or a
  release-script check) that `sonora --version` isn't the default.
- Check `release.sh` and `scripts/` for the old path too.

### 2.2 Move `internal/hub` → `hub/`

- `git mv internal/hub hub` (keeps history), then update the 77 imports of
  `.../internal/hub`, including `tests/contract/*`.

### 2.3 Move CLI concerns out of `hub`

- **`ErrorClass.ExitCode()`**: exit codes are CLI-only (it's called from 34 files under
  `internal/cli`). Move the mapping to a new `internal/cli/exitcode` package as
  `exitcode.For(hub.ErrorClass) int` and update call sites. `ErrorClass` and `ClassifyError`
  stay in `hub`: sonora-mcp needs the classification to build `isError` tool results.
- **`ClassifyError` messages**: its messages ("hub did not respond in time", "the audio source
  could not be reached") don't mention flags or exit codes, so they're fine for an AI assistant
  too. Keep them in `hub`.
- **`SingleLine`**: `hub/tts.go` uses it to build error messages, and one CLI file uses it too.
  Keep it exported in `hub`.
- **Godoc cleanup**: rewrite comments that cite "constitution Principle X", `research.md`, or
  `data-model.md` so they explain the behavior directly.

### 2.4 (Recommended) Export the spec

Add `api/spec.go` with `//go:embed openapi.json` and `var Spec []byte`. sonora-mcp can then run a
conformance test against the same spec version as the client it imports, instead of keeping its
own `openapi.json` copy.

### 2.5 Done when

- `go build ./...`, `go vet ./...`, and the full test suite pass on Windows and Linux.
- No CLI behavior change: same commands, output, and exit codes (the existing contract and unit
  tests prove it).
- `go doc github.com/Sonora-Multiroom/sonora-cli/hub` reads as a self-contained client API.
- A release is tagged (e.g. `v0.1.0`, or the next `v0.0.x`). The repo is public, so the Go
  module proxy caches tags permanently: **never move or re-create a pushed tag**; publish a
  new one instead.

## Phase 3: Rewrite sonora-mcp in Go

Back in this repo, after a sonora-cli release that contains `hub/`.

### 3.1 Constitution

Run `/speckit-constitution` here. The Node.js constitution (1.0.0, ratified 2026-09-27) gets
rewritten for Go. If 1.0.0 was committed, this is 2.0.0 (principles redefined); if it wasn't,
re-ratify it as 1.0.0.

- **Carries over unchanged in intent**: I API Contract Fidelity, II Agent-Oriented Tool Design,
  III Thin Stateless Proxy, IV Resilient Error Handling, V Test-First.
- **Changes**:
  - All hub calls MUST go through `github.com/Sonora-Multiroom/sonora-cli/hub`. Calling hub
    endpoints directly with `net/http` is prohibited. A missing operation gets added to `hub/`
    in sonora-cli first.
  - Spec fidelity is enforced through the imported client plus a tool-to-spec conformance
    test (using `api.Spec` if 2.4 was done). sonora-mcp no longer keeps its own
    `openapi.json`, and `npm run openapi:update` is retired.
  - Test-first scope becomes the MCP layer: tool input validation, mapping arguments to `hub`
    calls, turning `hub` errors into `isError` results, and flag parsing. Tests use a fake hub
    built with `httptest.Server`. Hub request and response contract tests live in sonora-cli.
  - Stack: Go (same version as sonora-cli's `go.mod`), the official MCP Go SDK
    (`github.com/modelcontextprotocol/go-sdk`), standard library otherwise. Tooling is
    `gofmt`, `go vet`, and `go test`.
  - Merge gate: `go build ./...`, `go vet ./...`, `go test ./...`.
  - Dependency rule: updating the sonora-cli dependency is a normal change, and the
    conformance test must pass.

### 3.2 Implementation

Run `/speckit-specify` (suggested name `001-go-rewrite`). Scope:

- **Keep the interface**: `--multiroom-url` (required), `--port` (default 3001), Streamable HTTP
  at `/mcp` (stateless), `GET /health` returning `{status, server, version}`. The VS Code and
  Claude client configs in README keep working unchanged.
- **Keep the 24 tool names** (`listInputs` … `setMasterMute`) so existing prompts and habits
  still work. Adding tools for operations `hub/` already has (`speak`, `listVoices`,
  `listExtensions`, `stopAllRoutes`, `stopRoutesForOutput/Group`, …) is a separate follow-up
  feature, not part of the rewrite.
- **Tool inputs**: Go structs with `json`/`jsonschema` tags, with constraints matching the spec
  (e.g. volume 0–100). Set MCP tool annotations (`readOnlyHint`, `destructiveHint`,
  `idempotentHint`).
- **Errors**: `hub.ClassifyError(err)` → an `isError: true` result carrying the friendly
  message, with the class name included so the agent can react to it (e.g. `NotFound` vs
  `Network`).
- **Logging**: one line per tool call: name, arguments, outcome, duration.
- **Version**: set by `-ldflags` at build time and reported in the MCP server info, `/health`,
  and the startup log.
- **Remove Node**: `package.json`, `package-lock.json`, `tsconfig.json`, `src/`, `dist/`,
  `scripts/update-openapi.mjs`, `openapi.json` (if 2.4 was done), and the `npm` lines in
  `run.sh`/`run.dev.sh`. Update `.gitignore` and README (build, cross-compile for
  `GOOS=linux GOARCH=arm64`, tech stack).
- **Check the MCP Go SDK API** (server constructor, typed tool registration, Streamable HTTP
  handler) against its current docs when planning. Don't rely on memory of its API.

### 3.3 Local development across both repos

Before a sonora-cli release exists, or while changing `hub/` and the tools together, use a Go
workspace from the parent folder (don't commit it):

```
cd D:\projects-sonora
go work init ./sonora-cli ./sonora-mcp
```

Then release sonora-cli, run `go get github.com/Sonora-Multiroom/sonora-cli@<tag>` here, and
make sure the build passes with `GOWORK=off` before merging.

### 3.4 Done when

- All 24 tools behave the same as the Node version against a real hub, checked with
  MCP Inspector: list, get, volume/mute, route create/transfer/pause/delete, playback,
  master mute.
- `go test ./...` passes with no network access, including the conformance test.
- No Node artifacts remain. README describes only the Go build.
- A linux/arm64 binary runs on the Pi next to the hub.

---

## Risks and open questions

- **Go MCP SDK lag**: new MCP spec features reach the TypeScript SDK first. This is acceptable
  while sonora-mcp only uses tools. Revisit if resources, prompts, or OAuth are needed.
- **Coupled release cadence (the cost of 2a)**: a hub fix needs a sonora-cli tag before
  sonora-mcp can use it. If this becomes painful, split `hub/` and `api/` into their own repo
  (option 2b). Import paths change but the API doesn't.
- **Repo owner**: sonora-mcp has moved into the `Sonora-Multiroom` org next to sonora-cli, and
  its module path is `github.com/Sonora-Multiroom/sonora-mcp`.
- **Hub URL configuration**: sonora-cli has `internal/config` (hub URL discovery). The MCP
  server keeps the explicit required flag (Principle III). Don't make `config` public unless a
  concrete need appears.

## Checklist

- [ ] Phase 1: sonora-cli constitution amended (1.2.0)
- [ ] Phase 2: `010-public-hub-package` specified, implemented, merged
- [ ] Phase 2: sonora-cli release tagged with public `hub/` (and `api.Spec`)
- [ ] Phase 3: sonora-mcp constitution rewritten for Go
- [ ] Phase 3: `001-go-rewrite` specified, implemented, verified on the Pi
- [ ] Phase 3: Node.js sources and tooling removed
