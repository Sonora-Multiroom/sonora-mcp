# TinySpec: Move to sonora-cli v0.3.1 (hub API 0.1.29)

**Branch**: adopt-sonora-cli-0.3
**Date**: 2026-10-26
**Status**: done
**Complexity**: small

## What

`go.mod` pins sonora-cli v0.1.3, whose `api.Spec` is hub API 0.1.22. Production runs 0.1.29. Bump to
v0.3.1 so conformance checks the current contract, and make the enable/disable, `transferRoute` and
`playback` descriptions say what the hub now does. No new tools, and the input schemas don't change.
Origin: [docs/backlog/adopt-sonora-cli-0.3.md](../../docs/backlog/adopt-sonora-cli-0.3.md).

## Context

| File | Role |
|------|------|
| `go.mod`, `go.sum` | Modified: sonora-cli v0.1.3 → v0.3.1 |
| `internal/tools/playback.go` | Modified: `Volume` → `hub.PlaybackRequest.OutputVolume` (MCP field stays `volume`); describe it as the room volume that persists; result lists `displayName` |
| `internal/tools/inputs.go`, `outputs.go`, `groups.go` | Modified: `setInputEnabled`, `setOutputEnabled`, `setGroupEnabled` say what disabling and re-enabling do to running routes |
| `internal/tools/routes.go` | Modified: `transferRoute` names `ROUTE_CHANGING_OUTPUTS` as the Conflict to retry unchanged |
| `internal/tools/playback_test.go` | Modified: the request body sent to the hub now carries `outputVolume` |
| `internal/tools/fakehub_test.go` and route/input fixtures | Modified only if conformance or decoding needs the new fields |
| `internal/tools/conformance_test.go`, `architecture_test.go` | Context: must pass unchanged |
| `AGENTS.md`, `.specify/memory/plan.md`, `spec.md`, `changelog.md` | Modified: the sonora-cli version, the tool descriptions and a changelog entry |
| `docs/backlog/adopt-sonora-cli-0.3.md`, `INDEX.md` | Deleted / updated once this ships |

## Requirements

1. `go.mod` requires sonora-cli v0.3.1, and the gate passes with `GOWORK=off`, with
   `conformance_test.go` unchanged.
2. `playback` still accepts `volume` (0–100, optional). The hub receives it as `outputVolume`, and
   omitting it sends nothing. The description says it sets the target's own volume, which stays after
   playback ends.
3. The `playback` description lists `displayName` in the result (the name sent, the resolved title of
   a SoundCloud or YouTube link, or a generated placeholder).
4. `setInputEnabled`: disabling stops every route playing the input, and an auto-remove input is then
   removed. `setGroupEnabled`: disabling stops routes addressed to the group, and direct routes on its
   members keep playing. `setOutputEnabled`: disabling takes the output out of the routes playing on it,
   and re-enabling puts it back into the live group routes on its group under the same route ID.
5. The `transferRoute` description says a Conflict with `ROUTE_CHANGING_OUTPUTS` clears within moments
   and the same request can be retried. The tool doesn't retry by itself (Principle III).
6. Output schemas gain the additive fields that come with the bump (`routeVolume`, `duckingLevelDb`,
   `defaultRouteVolume`, `defaultDuckingLevelDb`, `displayName`). No code is written for them.

## Plan

1. `GOWORK=off go get github.com/Sonora-Multiroom/sonora-cli@v0.3.1 && GOWORK=off go mod tidy`.
2. Test first: change `playback_test.go` to expect `"outputVolume":0` in the hub body. It fails
   until step 3 is done.
3. `playback.go`: map `in.Volume` to `OutputVolume`, and reword the `volume` tag and the description.
4. Reword the descriptions in `inputs.go`, `outputs.go`, `groups.go` and `routes.go`.
5. Run the gate. If conformance fails, a fix in `hub` (sonora-cli) may be needed. Stop and re-scope;
   don't change the test.
6. Docs: `AGENTS.md` (Active Technologies), master `plan.md`, the descriptions in `spec.md`, and a
   `changelog.md` entry. Delete the backlog item and its `INDEX.md` row. The other items' `Depends on`
   pointers stay valid against the shipped version.

## Tasks

- [x] Bump sonora-cli to v0.3.1
- [x] Update `playback_test.go` for `outputVolume` (red)
- [x] Map `volume` → `OutputVolume`, and reword `playback` (volume, `displayName`) (green)
- [x] Reword `setInputEnabled`, `setOutputEnabled`, `setGroupEnabled`
- [x] Reword `transferRoute` for `ROUTE_CHANGING_OUTPUTS`
- [x] Gate: `gofmt -l .`, `go vet ./...`, `go test ./...` (all with `GOWORK=off`)
- [x] Update `AGENTS.md`, `.specify/memory/plan.md`, `spec.md`, `changelog.md`
- [x] Remove the backlog item, and sync `docs/backlog/INDEX.md` and its dependents

## Done When

- [x] All tasks checked off
- [x] The gate passes, and `conformance_test.go` and `architecture_test.go` are unchanged and green
- [x] The MCP input schemas are unchanged, so the release is MINOR, not MAJOR
