# TinySpec: Report the hub's 409 refusals as a Conflict category

**Branch**: conflict-category
**Date**: 2026-10-07
**Status**: done
**Complexity**: small

## What

sonora-cli v0.1.2 decodes the hub's 409 answers (route refusals and duplicate input IDs) into
`hub.ClassConflict` with the hub's `detail`, `Reason` and `OutputID`. sonora-mcp is on v0.1.1 and maps
any unknown class to `HubError`, so playing to a disabled speaker reaches the agent as
"HubError: hub reported an error (HTTP 409)". Bump to v0.1.2 and give 409 its own category, so an
agent knows to fix the state (enable the output, stop a route) and retry.
Origin: `sonora-cli/docs/backlog/decode-route-refusals-409.md` (last bullet of the proposal).

## Context

| File | Role |
|------|------|
| `go.mod`, `go.sum`, `AGENTS.md` | Modified: `sonora-cli` v0.1.1 → v0.1.3 (v0.1.3 adds `hub.Input.DefaultJoinMode`, sonora-cli PR #22, which the conformance test required) |
| `internal/tools/errors.go` | Modified: `CategoryConflict`, map `hub.ClassConflict`, append the reason |
| `internal/tools/routes.go`, `playback.go`, `inputs.go` | Modified: tool descriptions name `Conflict`; route results list join mode and current outputs |
| `internal/tools/errors_test.go`, `errors_scenarios_test.go` | Modified: 409 cases |
| `internal/tools/fakehub_test.go`, `routes_write_test.go`, `inputs_write_test.go` | Modified: route fixtures gain `joinMode`/`outputs`, input fixtures `defaultJoinMode` |
| `README.md`, `.specify/memory/spec.md`, `plan.md` | Modified: the category tables and the closed set |
| `internal/tools/conformance_test.go` | Context: must pass unchanged against the v0.1.3 `api.Spec` |

## Requirements

1. A 409 from any tool returns `Conflict: <hub detail>`; a 409 without a decodable body returns
   `Conflict: ` followed by sonora-cli's generic conflict message.
2. When the hub gives a `reason`, the message ends with it in parentheses, e.g.
   `Conflict: Output 'bathroom' is disabled (OUTPUT_DISABLED)`. An unknown reason is shown as is;
   no reason, nothing appended. `outputId` is not added (the hub's detail already names the output).
3. `createInput` with a taken ID returns `Conflict: Input ID already exists` (was `HubError`).
4. `createRoute`, `transferRoute` and `getRoute`/`listRoutes` results include `joinMode` and
   `outputs`; `conformance_test.go` passes without changes to the test.
5. Descriptions of `createRoute`, `transferRoute`, `playback` and `createInput` say when `Conflict`
   is returned; `transferRoute` no longer implies a disabled target is a `Validation` failure.
6. Logging records `Conflict` as the outcome (it uses the category, so no logging change expected).

## Plan

1. `GOWORK=off go get github.com/Sonora-Multiroom/sonora-cli@v0.1.3 && go mod tidy`.
2. `errors.go`: add `CategoryConflict = "Conflict"`; case `hub.ClassConflict`; after the existing
   detail-append, append ` (<Reason>)` when `apiErr.Reason != ""`.
3. Update the four tool descriptions and the route field lists.
4. Tests: table cases in `errors_test.go` (409 with reason, without reason, unknown reason, bare
   `StatusError{409}`); a scenario in `errors_scenarios_test.go` driving `createRoute` against a fake
   hub 409 body with `reason`/`outputId`; update `routeJSON` and any route-result assertions.
5. Docs: add `Conflict` to the README table, master `spec.md` closed set and `plan.md` mapping; add a
   `changelog.md` entry.
6. Gate with `GOWORK=off`; delete the sonora-mcp bullet from the sonora-cli backlog item (or the item
   itself if nothing else remains) and keep its `INDEX.md` in sync.

## Tasks

- [x] Bump sonora-cli to v0.1.3 (after fixing `hub.Input` there)
- [x] Add `CategoryConflict` and the reason suffix in `classify`
- [x] Update tool descriptions (routes, playback, createInput)
- [x] Update `routeJSON` in the fake hub for the new route fields
- [x] Add unit and scenario tests for 409 (reason, no reason, unknown reason, bare status, duplicate input)
- [x] Update README, `.specify/memory/spec.md`, `plan.md`, `changelog.md`
- [x] Update the sonora-cli backlog item and its index
- [x] Gate: `gofmt -l .`, `go vet ./...`, `go test ./...` (all with `GOWORK=off`)

## Done When

- [x] All tasks checked off
- [x] Gate passes; `conformance_test.go` and `architecture_test.go` unchanged and green
- [x] No 409 path in the tests yields `HubError`

## Verification (2026-10-07)

- `gofmt -l .` empty; `GOWORK=off go vet ./...` clean; `GOWORK=off go test ./...` all packages ok,
  including the unchanged `TestConformance` and architecture test.
- The v0.1.2 bump alone failed `TestConformance` on 4 input tools (`defaultJoinMode` missing from
  `hub.Input`); fixed upstream in sonora-cli PR #22 (CI green), tagged v0.1.3.
- Not run: `go test -race` (CI on the PR), any call against the real hub.
