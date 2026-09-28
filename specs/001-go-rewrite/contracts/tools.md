# Contract: MCP Tools

The 24 tools the server exposes over `POST /mcp`. Names and inputs are the compatibility contract with
the Node.js server (spec FR-001) and must not change without a MAJOR version (Principle II). Result and
error shapes: [data-model.md](../data-model.md) (ToolResult, ErrorResult).

Conventions: `?` = optional. `TargetType` = `SINGLE_OUTPUT` | `OUTPUT_GROUP`. `Volume` = integer 0–100.
`RouteStatus` = `STARTING` | `ACTIVE` | `STOPPING` | `STOPPED` | `FAILED`.

## Inputs

| Tool | Hub operation | Inputs | Kind | Structured result |
|---|---|---|---|---|
| `listInputs` | `GET /api/v2/inputs` | `includeDisabled?: bool` | read-only | `{"inputs": [Input]}` |
| `getInput` | `GET /api/v2/inputs/{inputId}` | `inputId` | read-only | `Input` |
| `createInput` | `POST /api/v2/inputs` | `inputId` (pattern `^[a-zA-Z0-9\-_]{1,255}$`), `displayName`, `uri`, `enabled?: bool`, `autoRemove?: bool` | state-changing | `Input` |
| `deleteInput` | `DELETE /api/v2/inputs/{inputId}` | `inputId` | destructive | `{"deleted": true, "inputId"}` |
| `setInputEnabled` | `PUT /api/v2/inputs/{inputId}/enabled` | `inputId`, `enabled: bool` | idempotent | `Input` |

## Outputs

| Tool | Hub operation | Inputs | Kind | Structured result |
|---|---|---|---|---|
| `listOutputs` | `GET /api/v2/outputs` | `includeDisabled?: bool` | read-only | `{"outputs": [Output]}` |
| `getOutput` | `GET /api/v2/outputs/{outputId}` | `outputId` | read-only | `Output` |
| `setOutputVolume` | `PUT /api/v2/outputs/{outputId}/volume` | `outputId`, `volume: Volume` | idempotent | `OutputVolume` |
| `setOutputMute` | `PUT /api/v2/outputs/{outputId}/mute` | `outputId`, `muted: bool` | idempotent | `Output` |
| `setOutputEnabled` | `PUT /api/v2/outputs/{outputId}/enabled` | `outputId`, `enabled: bool` | idempotent | `Output` |

## Groups

| Tool | Hub operation | Inputs | Kind | Structured result |
|---|---|---|---|---|
| `listGroups` | `GET /api/v2/groups` | `includeDisabled?: bool` | read-only | `{"groups": [Group]}` |
| `getGroup` | `GET /api/v2/groups/{groupId}` | `groupId` | read-only | `Group` |
| `setGroupVolume` | `PUT /api/v2/groups/{groupId}/volume` | `groupId`, `volume: Volume` | idempotent | `GroupVolume` |
| `setGroupMute` | `PUT /api/v2/groups/{groupId}/mute` | `groupId`, `muted: bool` | idempotent | `Group` |
| `setGroupEnabled` | `PUT /api/v2/groups/{groupId}/enabled` | `groupId`, `enabled: bool` | idempotent | `Group` |

## Routes

| Tool | Hub operation | Inputs | Kind | Structured result |
|---|---|---|---|---|
| `listRoutes` | `GET /api/v2/routes` | `status?: RouteStatus`, `inputId?`, `targetId?` | read-only | `{"routes": [Route]}` |
| `getRoute` | `GET /api/v2/routes/{routeId}` | `routeId` | read-only | `Route` |
| `createRoute` | `POST /api/v2/routes` | `inputId`, `targetId`, `targetType: TargetType` | state-changing | `Route` |
| `deleteRoute` | `DELETE /api/v2/routes/{routeId}` | `routeId` | destructive | `{"deleted": true, "routeId"}` |
| `transferRoute` | `POST /api/v2/routes/{routeId}/transfer` | `routeId`, `targetId`, `targetType: TargetType` | state-changing | `Route` |
| `setRoutePause` | `PUT /api/v2/routes/{routeId}/pause` | `routeId`, `paused: bool` | idempotent | `Route` |

## Playback and master mute

| Tool | Hub operation | Inputs | Kind | Structured result |
|---|---|---|---|---|
| `playback` | `POST /api/v2/play` | `uri`, `targetId`, `targetType: TargetType`, `displayName?`, `volume?: Volume` | state-changing | `PlaybackResponse` (`inputId`, `route`, `message`) |
| `getMasterMute` | `GET /api/v2/master-mute` | — | read-only | `MasterMute` |
| `setMasterMute` | `PUT /api/v2/master-mute` | `muted: bool` | idempotent | `MasterMute` |

Resource shapes (`Input`, `Output`, `OutputVolume`, `Group`, `GroupVolume`, `Route`,
`PlaybackResponse`, `MasterMute`) are the `hub` package types, which carry every field of the
corresponding spec response schema.

Hub operation paths were checked against `api.Spec` (hub API 0.1.18) on 2026-09-28. The conformance
test, not this table, is authoritative when the spec changes.

## Error categories

Every failed call returns `isError: true` with text `"<Category>: <message>"`. Exception: input that
fails the tool's schema is rejected by the MCP SDK with `validating "arguments": <details>` (no hub call
made).

| Category | When |
|---|---|
| `NotFound` | The input/output/group/route (or a target) doesn't exist |
| `Validation` | The hub rejected the request as invalid (includes the hub's explanation) |
| `RouteFailed` | The hub could not create the route (422) |
| `SourceUnreachable` | The hub could not reach the audio source (502) |
| `ServiceUnavailable` | The hub's service is temporarily unavailable (503) |
| `Timeout` | The hub did not respond within the time limit |
| `Network` | The hub could not be reached |
| `MalformedResponse` | The hub's response could not be read as the expected data |
| `HubError` | Any other hub error status |
| `Internal` | Unexpected fault in the server while handling the call |
