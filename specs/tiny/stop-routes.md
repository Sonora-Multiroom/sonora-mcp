# TinySpec: Stop routes in bulk

**Branch**: stop-routes
**Date**: 2026-10-27
**Status**: done
**Complexity**: small

## What

Let an agent stop what is playing on an output, in a group, or everywhere in one call, instead of
listing routes and deleting each one (which misses group routes that play on the output). Three new
destructive, idempotent tools: `stopRoutesForOutput`, `stopRoutesForGroup`, `stopAllRoutes` (25 → 28
tools, a MINOR release). Origin: [docs/backlog/stop-routes.md](../../docs/backlog/stop-routes.md).

## Context

| File | Role |
|------|------|
| `internal/tools/outputs.go` | Modified: `stopRoutesForOutput` → `hub.StopRoutesForOutput`; count 5 → 6 |
| `internal/tools/groups.go` | Modified: `stopRoutesForGroup` → `hub.StopRoutesForGroup`; count 5 → 6 |
| `internal/tools/routes.go` | Modified: `stopAllRoutes` → `hub.StopAllRoutes`; count 7 → 8 |
| `internal/tools/outputs_control_test.go`, `groups_control_test.go`, `routes_write_test.go` | Modified: success, empty result, NotFound, empty ID, annotations |
| `internal/tools/conformance_test.go`, `cmd/sonora-mcp/main_test.go` | Modified: `contractTools` gains the three names; the count is 28 |
| `internal/tools/schema.go`, `tools.go` | Context: `withMinLength`, `Destructive` kind, `add` |
| `README.md`, `AGENTS.md`, `.specify/memory/spec.md`, `plan.md`, `changelog.md` | Modified: 28 tools, inventory rows, AS-007 loses stop-all and stop by output/group, changelog entry |
| `docs/backlog/stop-routes.md`, `INDEX.md` | Deleted / synced |

## Requirements

1. `stopRoutesForOutput` (`outputId`, minLength 1) calls `DELETE /api/v2/outputs/{outputId}/routes`;
   `stopRoutesForGroup` (`groupId`, minLength 1) calls `DELETE /api/v2/groups/{groupId}/routes`;
   `stopAllRoutes` (no arguments) calls `DELETE /api/v2/routes`. All are annotated destructive and
   idempotent, as `deleteRoute` is.
2. Each returns the hub's `BulkStopResponse` unchanged (`stoppedCount`, `stoppedRoutes`); no wrapping.
   Zero matching routes is a success with `stoppedCount: 0`. An unknown output or group is NotFound.
3. The `stopRoutesForOutput` description says it stops routes addressed to the output and every group
   route whose group includes it, so the group route stops on the other members too.
   `stopRoutesForGroup` says it stops routes addressed to the group and routes addressed to any of its
   members. `stopAllRoutes` says it stops everything on every speaker and is for when asked to stop
   everything.
4. `stopAllRoutes` takes no confirmation argument; MCP clients already confirm destructive tools.
5. Hub names are used (they match `stopAllRoutes`); `conformance_test.go` passes with only the
   tool-name additions, its comparison logic unchanged. No hub defaults are filled in (Principle III).

## Plan

1. Tests first (Principle V) against the fake hub: path and method, `BulkStopResponse` passed through,
   `stoppedCount: 0`, 404, empty ID rejected, destructive annotations; add the names to
   `contractTools`; set the count to 28. They fail.
2. Register each tool next to its sibling (`setOutputEnabled`, `setGroupEnabled`, `deleteRoute`) with
   `Kind: Destructive`, and raise the three `register*` counts.
3. Gate with `GOWORK=off`. If conformance fails, suspect the code or `hub`, not the test.
4. Docs: README table and count, AGENTS.md, master spec (inventory, counts, AS-007), plan, changelog.
5. Backlog: delete `stop-routes.md`, sync `INDEX.md`.

## Tasks

- [x] Write the failing tests for the three tools and the 28-tool inventory
- [x] Add `stopRoutesForOutput` to `outputs.go`
- [x] Add `stopRoutesForGroup` to `groups.go`
- [x] Add `stopAllRoutes` to `routes.go`
- [x] Gate: `gofmt -l .`, `go vet ./...`, `go test ./...` (all with `GOWORK=off`)
- [x] Update README, AGENTS.md, `.specify/memory/spec.md`, `plan.md`, `changelog.md`
- [x] Delete the backlog item and sync `INDEX.md`

## Done When

- [x] All tasks checked off
- [x] The gate passes, and `architecture_test.go` and the conformance logic are unchanged
- [x] `tools/list` shows 28 tools, and no existing input schema changed (MINOR)

## Verification

2026-10-27, the branch build run locally against the hub at `multiroom.lan` (hub API 0.1.29), with
`routeVolume` 1 on office and kitchen only; no room volume was changed.

- `stopRoutesForOutput` on office with nothing playing: `stoppedCount` 0.
- A `livingroom` group route, then `stopRoutesForOutput` on kitchen: stopped it (`GROUP_MEMBERSHIP`).
- Direct routes to office and kitchen, then `stopRoutesForGroup` on `livingroom`: `stoppedCount` 2; an
  unknown group answered `NotFound`.
- `stopAllRoutes` with only the test route playing: `stoppedCount` 1.
- After: no routes, `audio.output.routes` 0 for office and kitchen, no ALSA device open, volumes
  (office 11, kitchen 23, bedroom 50) and master mute unchanged.
