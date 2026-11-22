# TinySpec: Route chime

**Branch**: route-chime
**Date**: 2026-11-22
**Status**: done
**Complexity**: small

## What

Let an agent announce a route with a short sound and see why a route started with one: `chime` on
`playback` and `createRoute`, `defaultChime` on `createInput`, a new read-only tool `listChimes`
(32 → 33 tools, a MINOR release), and `chime` / `defaultChime` in the output, group and input results.
HUB-040 (API 0.1.34). Origin: [docs/backlog/adopt-route-chime.md](../../docs/backlog/adopt-route-chime.md).

**Dependency:** sonora-cli v0.3.7 (2026-11-20) is the first release with the chime (`hub/chimes.go`,
`DefaultChime`), but it and v0.4.0 lack HUB-041's input fields (see Known gap). This change pins
**v0.4.1** (sonora-cli #37), which has both; this repository was on v0.3.4. Never work around a
missing `hub` field here.

## Context

| File | Role |
|------|------|
| `go.mod`, `go.sum` | Modified: sonora-cli bump |
| `internal/tools/playback.go` | Modified: `chime` → `hub.PlaybackRequest.Chime` |
| `internal/tools/routes.go` | Modified: `chime` on `createRoute` → `hub.CreateRouteRequest.Chime` |
| `internal/tools/inputs.go` | Modified: `defaultChime` → `hub.CreateInputRequest.DefaultChime`; `hub.Input` already carries `DefaultChime` |
| `internal/tools/chimes.go` | New: `listChimes` → `hub.ListChimes`, wrapped as `{"chimes": [...]}`, registered from `tools.go` |
| `internal/tools/outputs.go`, `groups.go` | Context: results are `hub.Output` / `hub.Group`, so `chime` passes through the bump |
| `internal/tools/fakehub_test.go` | Modified: `GET /api/v2/chimes`; chime fields on fake outputs, groups, inputs |
| `internal/tools/playback_test.go`, `routes_write_test.go`, `inputs_write_test.go`, `*_read_test.go` | Modified: bodies with and without the new fields; pass-through in results |
| new `internal/tools/chimes_test.go` | New: `listChimes` result and 5 s timeout/error behaviour like the other list tools |
| `internal/tools/conformance_test.go`, `register_test.go`, `cmd/sonora-mcp/main_test.go` | Modified: `contractTools` gains `listChimes`; count 33 |
| `README.md`, `AGENTS.md`, `.specify/memory/spec.md`, `plan.md`, `changelog.md` | Modified: 33 tools, inventory row, sonora-cli version, changelog entry |
| `docs/backlog/adopt-route-chime.md`, `INDEX.md` | Deleted / synced |

## Requirements

1. `playback` and `createRoute` accept an optional string `chime` (minLength 1) and send it as `chime`.
   When omitted, no `chime` key goes to the hub. An unknown id is left to the hub's 400.
2. The `chime` description says: omitted means the hub decides (the input's default chime, then the
   target's); `none` means no chime whatever those say; ids come from `listChimes`; a chime is a
   request, not a guarantee, as the hub skips it when every target output is already playing. It does
   not list ids.
3. `createInput` accepts an optional string `defaultChime` (minLength 1), omitted when not given.
4. `listChimes` takes no arguments, is read-only, and returns `{"chimes": [{"id", "durationMillis"}]}`.
   `none` is never listed. Its description mentions `none`.
5. `getOutput`/`listOutputs`, `getGroup`/`listGroups` and `getInput`/`listInputs` show `chime` /
   `defaultChime` when set (`none` is a value) and omit them when not. Their descriptions say so.
6. The tools fill in no hub defaults (Principle III). `conformance_test.go` passes with only the
   tool-name addition; its comparison logic is not loosened.

## Plan

1. Bump sonora-cli; run the gate to see what the new spec changes (`GOWORK=off`).
2. Tests first (Principle V): request bodies with and without `chime` / `defaultChime`; `listChimes`
   against the fake hub; chime fields in read results; `contractTools` and the count 33. They fail.
3. Add the fields in `playback.go`, `routes.go`, `inputs.go` (pointers, `omitempty`, `withMinLength`);
   add `chimes.go` and register it in `tools.go`.
4. Gate. If conformance fails, suspect the code or `hub`, not the test.
5. Docs and backlog: README tool table and count, AGENTS.md (count, sonora-cli version, spec API
   version), master spec, plan, changelog; delete the backlog item and sync `INDEX.md`.

## Tasks

- [x] Bump sonora-cli to v0.4.1 (`go get ...@v0.4.1`): HUB-040 plus the `hub.Input` fields (v0.3.7 had only the chime)
- [x] Write the failing tests for `chime`, `defaultChime`, `listChimes`, read pass-through and the 33-tool inventory
- [x] Add `chime` to `playback` and `createRoute`
- [x] Add `defaultChime` to `createInput`
- [x] Add the `listChimes` tool
- [x] Update descriptions of the output, group and input read tools
- [x] Gate: `gofmt -l .`, `go vet ./...`, `go test ./...` (all with `GOWORK=off`), 2026-11-22: all pass
- [x] Update README, AGENTS.md, `.specify/memory/spec.md`, `plan.md`, `changelog.md`
- [x] Delete the backlog item, sync `INDEX.md`

## Done When

- [x] All tasks checked off
- [x] The gate passes; `architecture_test.go` and the conformance logic are unchanged
- [x] `tools/list` shows 33 tools and no existing input schema lost a field or got stricter (MINOR)

## Known gap (closed)

sonora-cli v0.3.7 and v0.4.0 pin `api.Spec` 0.1.35, but `hub.Input` lacked HUB-041's `origin`,
`playable` and `unplayableReason`, so `TestConformance` failed for `listInputs`, `getInput`,
`setInputEnabled` and `createInput`. Fixed in sonora-cli first (#37, v0.4.1) and bumped here; the
conformance logic was not loosened. The input fixtures gained the three fields.
