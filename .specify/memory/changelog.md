# Changelog

## Merged Features Log

### Stop routes in bulk — 2026-10-27
**Branch:** stop-routes
**Spec:** [specs/tiny/stop-routes.md](../../specs/tiny/stop-routes.md)

- Three new destructive, idempotent tools, each returning the hub's `BulkStopResponse`
  (`stoppedCount`, `stoppedRoutes`) unchanged: `stopRoutesForOutput` (also stops group routes whose
  group includes the output), `stopRoutesForGroup` and `stopAllRoutes`. 25 → 28 tools, a MINOR
  release. They use the hub's operation names and take no confirmation argument.

### Route volume — 2026-10-26
**Branch:** route-volume
**Spec:** [specs/tiny/route-volume.md](../../specs/tiny/route-volume.md)

- `routeVolume` (0–100, optional) on `playback` and `createRoute`: the volume of that playback
  only, without changing the room's; `defaultRouteVolume` on `createInput` for its routes.
- New idempotent tool `setRouteVolume` (`PUT /api/v2/routes/{routeId}/volume`), 24 → 25 tools.
  A route volume can be changed but never cleared. MINOR release: nothing existing got stricter.
- The ducking level (`duckingLevelDb`, `defaultDuckingLevelDb`) moved to the join-modes backlog
  item, since it only works with `DUCK_OTHERS` and no tool chooses a join mode yet.

### Move to sonora-cli v0.3.1 (hub API 0.1.29) — 2026-10-26
**Branch:** adopt-sonora-cli-0.3
**Spec:** [specs/tiny/adopt-sonora-cli-0.3.md](../../specs/tiny/adopt-sonora-cli-0.3.md)

- sonora-cli v0.1.3 → v0.3.1: conformance now checks against hub API 0.1.29 instead of 0.1.22.
- `playback.volume` keeps its name but reaches the hub as `outputVolume` (sonora-cli v0.3.0 renamed
  `PlaybackRequest.Volume`); the input schemas are unchanged, so this is a MINOR release.
- Descriptions say what disabling and re-enabling do to running routes (`setInputEnabled`,
  `setOutputEnabled`, `setGroupEnabled`), that a `ROUTE_CHANGING_OUTPUTS` Conflict on
  `transferRoute` can be retried, and that `playback` returns `displayName`.
- Output schemas gain the hub's additive fields: `routeVolume` and `duckingLevelDb` on routes,
  `defaultRouteVolume` and `defaultDuckingLevelDb` on inputs, `displayName` on the playback result.

### Conflict category for the hub's 409 refusals — 2026-10-07
**Branch:** conflict-category
**Spec:** [specs/tiny/conflict-category.md](../../specs/tiny/conflict-category.md)

- sonora-cli v0.1.1 → v0.1.3: 409s on `createRoute`, `transferRoute`, `playback` and `createInput`
  are decoded with the hub's detail and reason (v0.1.2), and `hub.Input` gains `defaultJoinMode`
  (v0.1.3, PR #22, found by the conformance test).
- New error category `Conflict`, e.g. `Conflict: Output 'bathroom' is disabled (OUTPUT_DISABLED)`;
  a duplicate input ID is now `Conflict` instead of `HubError`.
- Route results include `joinMode` and `outputs`, input results `defaultJoinMode`; transfer keeps
  the route ID (hub 0.1.22).

### Rewrite sonora-mcp on the Shared Hub Client — archived 2026-09-28
**Branch:** 001-go-rewrite
**Spec:** [specs/001-go-rewrite/spec.md](../archive/001-go-rewrite/spec.md) (folder moved to `.specify/archive/001-go-rewrite/`)

**What was added:**
- The Node.js/TypeScript server is replaced by a Go server on sonora-cli's public `hub` package,
  with the same 24 tools, names and inputs (US1–US3), released as v1.1.0 (PR #1).
- Structured results (`structuredContent` plus JSON text, declared output schemas) and tool
  annotations; category-prefixed errors (`NotFound:`, `Validation:`, `Timeout:` …) with the hub's
  explanation; 5 s bound on every hub call and cancellation (US4).
- `--host` flag, `/health` with hub reachability, graceful shutdown on SIGINT/SIGTERM, static
  builds for Windows and linux/arm64 (US5).
- One-command Raspberry Pi install (`deploy/pi/install.sh`) of a hardened systemd service from a
  checksum-verified GitHub Release; GoReleaser release pipeline and PR test workflow (US5).
- Conformance test against `api.Spec`, a fixed 24-tool inventory test, and removal of all Node.js
  sources, npm tooling and the local `openapi.json` (US6).

**New Components:**
- `cmd/sonora-mcp`, `internal/config`, `internal/version`, `internal/tools`, `internal/server`
- `deploy/pi/install.sh`, `.goreleaser.yaml`, `.github/workflows/release.yml`,
  `.github/workflows/test.yml`, `release.sh`
- Upstream in sonora-cli: `CreateInputRequest` optional booleans (`3ee7369`), a mute fix, and
  404 attribution on route calls (PR #20, v0.1.1)

**Tasks Completed:** 78/78 tasks
