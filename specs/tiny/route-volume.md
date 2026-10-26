# TinySpec: Route volume

**Branch**: route-volume
**Date**: 2026-10-26
**Status**: done
**Complexity**: small

## What

Let an agent give a playback its own volume without changing the room's, and change that volume while
it plays: `routeVolume` on `playback` and `createRoute`, `defaultRouteVolume` on `createInput`, and a
new idempotent tool `setRouteVolume` (24 → 25 tools, a MINOR release). The ducking level
(`duckingLevelDb`, `defaultDuckingLevelDb`) moves to [join-modes.md](../../docs/backlog/join-modes.md):
it only works with `DUCK_OTHERS`, and no tool can choose a join mode yet.
Origin: [docs/backlog/play-route-volume.md](../../docs/backlog/play-route-volume.md).

## Context

| File | Role |
|------|------|
| `internal/tools/playback.go` | Modified: `routeVolume` → `hub.PlaybackRequest.RouteVolume` |
| `internal/tools/routes.go` | Modified: `routeVolume` on `createRoute`; new `setRouteVolume` → `hub.SetRouteVolume`; count 6 → 7 |
| `internal/tools/inputs.go` | Modified: `defaultRouteVolume` → `hub.CreateInputRequest.DefaultRouteVolume` |
| `internal/tools/playback_test.go`, `routes_write_test.go`, `inputs_write_test.go` | Modified: request bodies with and without the new fields; `setRouteVolume` success and NotFound |
| `internal/tools/conformance_test.go`, `cmd/sonora-mcp/main_test.go` | Modified: `contractTools` gains `setRouteVolume`; the count is 25 |
| `internal/tools/schema.go` | Context: `withRange` takes `float64` bounds |
| `README.md`, `AGENTS.md`, `.specify/memory/spec.md`, `plan.md`, `changelog.md` | Modified: 25 tools, the inventory row, a changelog entry |
| `docs/backlog/play-route-volume.md`, `join-modes.md`, `INDEX.md` | Deleted / gains the ducking level / synced |

## Requirements

1. `playback` and `createRoute` accept an optional `routeVolume` (integer 0–100) and send it as
   `routeVolume`. When it is omitted, no `routeVolume` key goes to the hub.
2. The `routeVolume` description says: the volume of this playback only; the room's volume is not
   changed and its volume control does not reach this playback; mute, master mute and output gain still
   apply; use it for announcements and alerts. `playback`'s `volume` keeps its room-volume wording.
3. `createInput` accepts an optional `defaultRouteVolume` (integer 0–100), the route volume for routes
   from this input that don't set one. It is omitted when not given.
4. `setRouteVolume` (`routeId` minLength 1, `volume` integer 0–100, both required, idempotent) calls
   `PUT /api/v2/routes/{routeId}/volume` with `{"volume": n}` and returns the route. An unknown route
   is NotFound. The description says it changes only this route, not any output's volume, works on a
   route started without one, and that a route volume can be changed but never cleared.
5. Out-of-range values are rejected by the schema. The tool fills in no hub defaults (Principle III).
6. `conformance_test.go` passes with only the tool-name addition. Its comparison logic is not changed.

## Plan

1. Tests first (Principle V): bodies for `routeVolume` / `defaultRouteVolume` present and absent;
   `setRouteVolume` success and 404 against the fake hub; add `setRouteVolume` to `contractTools` and
   set the count to 25. They fail.
2. Add the fields and `withRange(..., 0, 100)` in `playback.go`, `routes.go` and `inputs.go`. Register
   `setRouteVolume` in `routes.go` next to `setRoutePause`.
3. Gate with `GOWORK=off`. If conformance fails, suspect the code or `hub`, not the test.
4. Docs: README tool table and count, AGENTS.md, master spec inventory and counts, plan, changelog.
5. Backlog: delete `play-route-volume.md`, add the ducking level to `join-modes.md`, sync `INDEX.md`.

## Tasks

- [x] Write the failing tests for `routeVolume`, `defaultRouteVolume` and `setRouteVolume`, and the 25-tool inventory
- [x] Add `routeVolume` to `playback` and `createRoute`
- [x] Add `defaultRouteVolume` to `createInput`
- [x] Add the `setRouteVolume` tool
- [x] Gate: `gofmt -l .`, `go vet ./...`, `go test ./...` (all with `GOWORK=off`)
- [x] Update README, AGENTS.md, `.specify/memory/spec.md`, `plan.md`, `changelog.md`
- [x] Move the ducking level into `join-modes.md`, delete this backlog item, sync `INDEX.md`

## Done When

- [x] All tasks checked off
- [x] The gate passes, and `architecture_test.go` and the conformance logic are unchanged
- [x] `tools/list` shows 25 tools, and no existing input schema lost a field or got stricter (MINOR)
