<!--
Sync Impact Report
==================
Version change: 1.0.0 (Node.js draft, never committed) → 1.0.0 (Go, re-ratified 2026-09-27)
Why not 2.0.0: the Node.js draft was never committed or used to govern a feature, so this is
  treated as the first adopted version (per docs/future/go-rewrite-shared-hub-client.md §3.1).
Modified principles (intent carried over, obligations rewritten for Go):
  - I. API Contract Fidelity (OpenAPI-Driven) → I. API Contract Fidelity via the Shared Hub
    Client: all hub calls go through sonora-cli's public `hub` package; conformance is checked
    against `api.Spec`; the local `openapi.json` copy and `npm run openapi:update` are retired.
  - II. Agent-Oriented Tool Design: Zod schemas → Go input structs with JSON Schema tags;
    otherwise unchanged.
  - III. Thin, Stateless Proxy: unchanged in intent.
  - IV. Resilient, Transparent Error Handling: timeouts come from `hub.NewClient`; errors are
    translated through `hub.ClassifyError`.
  - V. Test-First Development (NON-NEGOTIABLE): scope narrowed to the MCP layer; hub
    contract tests live in sonora-cli (its Principle VII).
  - VI. Minimal Dependencies & Simplicity: Node built-ins → Go standard library; sanctioned
    dependencies are the MCP Go SDK and the sonora-cli module.
Added sections: none new versus the draft; "Technology & Operational Constraints" rewritten
  for Go (toolchain, dependency pinning, go.work rule, cross-compilation, legacy Node freeze).
Removed sections: none.
Aligned with: sonora-cli constitution 1.2.0 (Principle II hub-package bullet, Principle VII
  Public Hub Client Package).
Deferred / TODO items: none.
Known gaps between the repository and this constitution (resolved by feature 001-go-rewrite):
  - The server is still the Node.js implementation (src/, package.json, openapi.json,
    scripts/update-openapi.mjs); it is frozen by the "Legacy Node.js implementation" rule.
  - No go.mod exists yet; sonora-cli's public `hub/` is on PR #19 (branch
    010-public-hub-package), not yet tagged.
Templates requiring follow-up: none. plan/spec/tasks templates read this file at runtime.
-->

# Sonora MCP Constitution

## Core Principles

### I. API Contract Fidelity via the Shared Hub Client

The Multiroom Audio Hub OpenAPI document is the single source of truth for the hub API, and
sonora-cli's public `hub` package (`github.com/Sonora-Multiroom/sonora-cli/hub`) is the one
place where that spec becomes Go types and HTTP calls.

- Every hub call MUST go through the `hub` package. Calling hub endpoints directly with
  `net/http`, or defining local request/response types for hub resources, is prohibited.
- An operation missing from `hub` MUST be added there first (in sonora-cli, under its
  constitution), released, and then consumed here.
- Every MCP tool that wraps a hub operation MUST be verified by an automated conformance test
  against the spec embedded in the same sonora-cli version (`api.Spec`): the operation exists,
  and the tool's input constraints (enums, ranges such as volume 0–100, required fields)
  match the spec's.
- This repository MUST NOT keep its own copy of `openapi.json`. The spec version is whatever
  the pinned sonora-cli version embeds.
- The server MUST NOT invent behavior the spec does not describe (undocumented endpoints,
  guessed fields).

Rationale: one tested client, shared by every Sonora Go project, can't drift from itself.
Checking the tools against the spec that ships with that client keeps the MCP surface in
step with it.

### II. Agent-Oriented Tool Design

The consumers of this server are LLM agents, so tool surface quality is a correctness concern.

- Tool names MUST be stable, unique, camelCase verb+noun names (e.g. `setOutputVolume`),
  consistent across resource families (inputs, outputs, groups, routes). Hub `operationId`
  suffixes such as `_1` MUST NOT leak into tool names.
- Every tool MUST have a description stating what it does, its side effects, and what it
  returns, in terms an agent can act on without reading the hub's source. Every input field
  MUST carry a description.
- Tool inputs MUST be defined as Go structs whose JSON Schema (via struct tags) carries the
  spec's constraints. Input that fails the schema MUST produce a tool error and no hub request.
- Tools that change state MUST be distinguishable from read-only ones through MCP tool
  annotations (`readOnlyHint`, `destructiveHint`, `idempotentHint`).
- Renaming or removing a tool, or changing its input schema incompatibly, is a breaking change
  for connected agents and MUST bump the server's MAJOR version.

Rationale: an agent picks and calls tools from their names, descriptions, and schemas alone.
Ambiguity there shows up as wrong actions on a live audio system.

### III. Thin, Stateless Proxy

The server translates MCP tool calls into `hub` calls and back. It holds no domain logic.

- The server MUST NOT cache, mirror, or persist hub state; every tool call reflects the hub's
  state at call time.
- The MCP transport MUST remain stateless (no session IDs) unless a feature spec justifies
  session state and this constitution is amended.
- Business rules (routing, volume semantics, input lifecycle) MUST live in the hub. A
  composite tool that orchestrates several `hub` calls MAY be added only when a spec shows no
  single hub operation covers the workflow, and it MUST NOT reimplement validation the hub
  already performs.
- The hub base URL MUST come from explicit configuration (`--multiroom-url`); the server MUST
  refuse to start without it rather than guessing or discovering a default.

Rationale: two sources of truth for audio state diverge. Keeping this layer thin keeps it
correct as the hub changes, and keeps failures attributable to one place.

### IV. Resilient, Transparent Error Handling

- Every hub call MUST be bounded by a timeout, using an `*http.Client` from `hub.NewClient`
  (or `hub.NewClientWithTimeout` where a longer bound is justified, e.g. TTS). Hub calls MUST
  also receive the request's `context.Context` so a cancelled MCP request stops its hub call.
- A failed hub call MUST be returned as an MCP tool result with `isError: true`, built from
  `hub.ClassifyError`: the friendly message plus a stable name for the error class (e.g.
  `NotFound`, `Network`, `Validation`) so an agent can decide whether to retry, correct its
  input, or report.
- A tool failure or panic in a handler MUST NOT crash the process or affect other in-flight
  requests. Only startup configuration errors MAY terminate the server, with a non-zero exit
  code and usage text.
- Logs MUST record each tool call (name, arguments, outcome/error class, duration) to
  stdout/stderr, with enough detail to diagnose a failed call without a debugger.

Rationale: agents retry or change strategy based on error text. An opaque or hanging failure
on a home LAN is worse than a fast, specific one.

### V. Test-First Development (NON-NEGOTIABLE)

Tests MUST be written before implementation for every new tool, behavior change, and bugfix:
write the test, watch it fail, implement the minimum to pass, then refactor.

- Scope is the MCP layer: tool input schemas and validation, mapping of arguments onto `hub`
  calls, translation of `hub` errors into tool results, flag parsing, and the `/health`
  endpoint. Hub request/response contract tests belong to sonora-cli and MUST NOT be
  duplicated here.
- Tool tests MUST exercise tools through the MCP layer against a fake hub
  (`net/http/httptest.Server`), covering at least the core flows: playback, route
  create/delete/transfer/pause, and volume/mute.
- The spec conformance test required by Principle I MUST run as part of `go test ./...`.
- Tests MUST NOT require a real hub or network access, and MUST pass on Windows and Linux.
- Implementation submitted without a preceding failing test for its behavior MUST be rejected.

Rationale: this server's whole value is faithful translation between MCP and the hub client.
Tests are the only practical way to prove that translation stays correct as sonora-cli and
the MCP SDK move.

### VI. Minimal Dependencies & Simplicity

- The sanctioned direct dependencies are the Go standard library, the official MCP Go SDK
  (`github.com/modelcontextprotocol/go-sdk`), and the sonora-cli module. Any other dependency
  MUST be justified in the change that adds it (what it provides that these cannot), with its
  effect on binary size noted.
- Use the standard library for HTTP serving, flag parsing, logging (`log/slog`), and testing.
- Start with the simplest design that satisfies the spec (YAGNI). Abstractions MUST be
  motivated by a current requirement, not an anticipated one.

Rationale: every dependency is upgrade churn and attack surface for a small, long-running LAN
service. A small codebase stays auditable by one person.

## Technology & Operational Constraints

- **Language & toolchain**: Go, at the same version as sonora-cli's `go.mod` (currently
  1.27). Module path `github.com/tiger-seo/sonora-mcp`.
- **MCP**: official MCP Go SDK; Streamable HTTP transport served at `/mcp`, plus a
  `GET /health` endpoint that reports server name and version.
- **Dependency pinning**: `go.mod` on `main` MUST require a tagged sonora-cli release. Pseudo-
  versions of unmerged branches and `replace` directives MUST NOT be merged to `main`.
  Updating the sonora-cli version is a normal change and MUST pass the full merge gate,
  including the conformance test.
- **Local cross-repo development**: a `go.work` workspace in the parent folder MAY point at a
  local sonora-cli checkout. `go.work` and `go.work.sum` MUST NOT be committed, and the merge
  gate MUST pass with `GOWORK=off`.
- **Versioning**: the server version MUST be injected once at build time (`-ldflags -X`) and
  reported identically by the MCP server info, `/health`, and the startup log. Semantic
  versioning applies; tool-surface breaking changes are MAJOR (Principle II).
- **Platforms**: the server MUST build and run on Windows (development) and Linux, including
  linux/arm64 (Raspberry Pi, alongside the hub), as a static binary (`CGO_ENABLED=0`).
  File paths MUST be built with `path/filepath`. Shell scripts MAY exist only as optional
  conveniences; the documented build is plain `go build`.
- **Security**: the server and hub are unauthenticated and intended for a trusted home LAN
  only. The server MUST NOT be exposed to the public internet. Adding authentication or remote
  exposure requires a feature spec and a constitution amendment.
- **Scope**: the hub's v2 API only; legacy v1 endpoints MUST NOT be wrapped.
- **Legacy Node.js implementation**: the existing TypeScript server (`src/`, `package.json`,
  `openapi.json`, `scripts/update-openapi.mjs`) is frozen. It MUST NOT receive new features
  and is removed by the Go rewrite; until then it is only a behavior reference.

## Development Workflow & Quality Gates

1. **Spec first**: features start with `/speckit-specify` under `specs/[###-feature-name]/`,
   then `/speckit-plan` and `/speckit-tasks`, whose Constitution Check MUST pass or record a
   justified exception.
2. **Test first**: per Principle V.
3. **Style**: code MUST be `gofmt`-formatted and match the conventions of the surrounding
   code and of sonora-cli (package layout under `cmd/` and `internal/`, error wrapping with
   `%w`, godoc on exported identifiers).
4. **Merge gate**: with `GOWORK=off`, `gofmt -l .` MUST report nothing, and `go vet ./...`,
   `go build ./...`, and `go test ./...` MUST pass. Until a CI host exists, run the gate
   locally and record the result in the commit or PR description. When CI is introduced, it
   MUST run the gate on Windows and Linux.
5. **Hub changes**: a change that needs a new or modified `hub` operation MUST land in
   sonora-cli first (Principle I); the sonora-mcp change then bumps the pinned version.
6. **Commits**: Conventional Commits (`type(scope): description`) explaining why; each commit
   one logical change. Work on feature branches named `[###-feature-name]` when following a
   Spec Kit feature.
7. **Documentation**: README MUST stay in sync with the tool list, CLI flags, build and
   cross-compile instructions, and client configuration examples whenever those change.

## Governance

This constitution supersedes conflicting informal practice in this repository. It is the
runtime guidance for `/speckit-plan`, `/speckit-tasks`, `/speckit-analyze`, and
`/speckit-implement`.

**Amendments** are made by editing this file via `/speckit-constitution`, with a Sync Impact
Report describing the change. Principles shared with sibling projects (sonora-cli,
multiroom-ai) SHOULD stay consistent in intent; divergence MUST be deliberate and stated in
the report. A sonora-cli amendment that changes its Principle II or VII MUST trigger a review
of Principle I here.

**Versioning policy** (semantic versioning of this document):

- **MAJOR**: a principle is removed or redefined, or a MUST is relaxed or dropped.
- **MINOR**: a principle or section is added, or guidance is materially expanded.
- **PATCH**: clarifications, wording, examples, and typo fixes with no change in obligation.

**Compliance review**: every spec, plan, and change MUST be checked against these principles
before merge. A violation MUST either be fixed or be justified explicitly (in the plan's
Complexity Tracking or the change description); a recurring justified violation signals that
this constitution needs amending.

**Version**: 1.0.0 | **Ratified**: 2026-09-27 | **Last Amended**: 2026-09-27
