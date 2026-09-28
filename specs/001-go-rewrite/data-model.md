# Data Model: Rewrite sonora-mcp on the Shared Hub Client

**Feature**: [spec.md](spec.md) | **Research**: [research.md](research.md)

The server stores nothing (Principle III). The entities below are in-memory definitions built at
startup and the per-call values that flow through them. Hub resource shapes are owned by
sonora-cli's `hub` package and the hub spec; they are referenced here, not redefined.

## ToolDefinition (startup, immutable)

One per tool; 24 in total ([contracts/tools.md](contracts/tools.md)).

| Field | Description | Rules |
|---|---|---|
| `Name` | Tool name, e.g. `setOutputVolume` | Unique; exactly the 24 inventory names; camelCase; no `_1` suffixes |
| `Description` | What it does, side effects, what it returns | Non-empty; every input field also has a description |
| `Operation` | Hub operation wrapped: HTTP method + spec path, e.g. `PUT /api/v2/outputs/{outputId}/volume` | Must exist in `api.Spec` (conformance test) |
| `Kind` | `read-only`, `idempotent`, `state-changing`, `destructive` | Maps to annotations (below) |
| `InputSchema` | JSON Schema inferred from the input struct + applied constraints | Constraints equal the spec's (conformance test) |
| `OutputSchema` | JSON Schema inferred from the result type | Always type `object` (lists wrapped) |
| `Handler` | Calls exactly one `hub` function with the request context | No direct HTTP; no state kept |

**Kind → annotations**

| Kind | `readOnlyHint` | `destructiveHint` | `idempotentHint` |
|---|---|---|---|
| read-only | true | — | — |
| idempotent | false | false | true |
| state-changing | false | false | false |
| destructive | false | true | false |

## ToolInput (per call)

Arguments object sent by the agent, validated against `InputSchema` before the handler runs.

- Identifier fields are strings; their constraints follow the spec exactly (R4):
  - Path IDs (`inputId`, `outputId`, `groupId`, `routeId` where they fill a URL path segment):
    `minLength: 1`. The spec declares no `minLength` on path parameters, but an empty segment would
    change the route (`GET /api/v2/outputs/` is the list), so the conformance test treats path
    parameters as `minLength: 1`. Passed to `hub`, which path-escapes them.
  - Body IDs (`createRoute.inputId`, `targetId` in `createRoute`/`transferRoute`/`playback`):
    `minLength: 1`, as in the spec. `createInput.inputId` additionally matches
    `^[a-zA-Z0-9\-_]{1,255}$`.
  - `listRoutes` filters `inputId`, `targetId`: optional, no `minLength` (none in the spec).
- Other strings: `uri` and `createInput.displayName` are `minLength: 1`; `playback.displayName` is
  optional with no `minLength` (spec type `["string","null"]`).
- `volume`: integer, 0–100 (required for set-volume tools, optional for `playback`).
- `targetType`: enum `SINGLE_OUTPUT` | `OUTPUT_GROUP`.
- `status` (`listRoutes`): enum `STARTING` | `ACTIVE` | `STOPPING` | `STOPPED` | `FAILED`, optional.
- Booleans (`enabled`, `muted`, `paused`, `includeDisabled`, `autoRemove`): required where the hub
  requires them; optional ones are omitted from the hub request when absent (prerequisite R7 for
  `createInput`).
- Validation failure → error result with the SDK's `validating "arguments": ...` text (logged as
  `InvalidInput`); no hub request.

## ToolResult (per call, success)

| Field | Content |
|---|---|
| `structuredContent` | Hub data as an object: single resources as returned by `hub`; lists wrapped (`{"inputs": [...]}`, `{"outputs": [...]}`, `{"groups": [...]}`, `{"routes": [...]}`); deletes return `{"deleted": true, "inputId"/"routeId": "<id>"}` |
| `content` | One text block containing the same JSON |

Every field the spec defines for the resource is present (hub types verified complete for outputs and
routes; for every tool the conformance test checks that each property of the spec's success-response
schema is a property of the tool's `outputSchema`, list envelopes unwrapped, 204 operations skipped).

## ErrorResult (per call, failure)

| Field | Content |
|---|---|
| `isError` | `true` |
| `content` | One text block: `"<Category>: <message>"` |

**Category** (closed set): `NotFound`, `Validation`, `RouteFailed`, `SourceUnreachable`,
`ServiceUnavailable`, `Network`, `Timeout`, `MalformedResponse`, `HubError`, `Internal`
(`InvalidInput` appears only in logs, see ToolInput). The mapping from `hub` errors is in [research.md](research.md) R5.

## HealthStatus (per `/health` request)

| Field | Value |
|---|---|
| `status` | `"ok"` (always, while the process serves requests) |
| `server` | `"sonora-mcp"` |
| `version` | `BuildVersion` |
| `hub` | `"reachable"` if `GetMasterMute` succeeds within 2 s, else `"unreachable"` |

## ServerConfig (startup)

| Field | Source | Rules |
|---|---|---|
| `HubURL` | `--multiroom-url` | Required; `http`/`https` URL with host |
| `Port` | `--port` | Default 3001; 1–65535 |
| `Host` | `--host` | Default empty (all addresses); IP literal or `localhost` |

Invalid config → usage on stderr, exit 2. `--help` → usage, exit 0.

## BuildVersion

Single string set at build time (default `dev`); read by MCP server info, `/health` and the startup
log.

## CallLogEntry (per call)

`tool`, `args` (JSON), `outcome` (`ok` or Category), `duration`. Emitted once per call, including
input-validation failures and panics.

## Lifecycle

```
start → parse config ─invalid→ exit 2
          │
          ▼
       register 24 tools → listen ─bind error→ exit 1
          │
          ▼
       serving ──SIGINT/SIGTERM──▶ draining (no new conns; in-flight ≤ 6 s) ──▶ exit 0
```
