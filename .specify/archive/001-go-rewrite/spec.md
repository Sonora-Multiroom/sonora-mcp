# Feature Specification: Rewrite sonora-mcp on the Shared Hub Client

**Feature Branch**: `001-go-rewrite`

**Created**: 2026-09-27

**Status**: Completed

**Input**: User description: "@docs/future/go-rewrite-shared-hub-client.md" — phase 3.2 of that plan:
replace the Node.js/TypeScript MCP server with a Go implementation built on sonora-cli's public
`hub` package, keeping the same tools, flags and endpoints, then remove the Node.js code.

## Clarifications

### Session 2026-09-27

- Q: Which network addresses should the server accept connections on by default? → A: All
  addresses by default (as today), plus an optional `--host` flag to restrict it (e.g.
  `--host 127.0.0.1`).
- Q: Should tool results give agents the hub data only as JSON text, as today, or also as
  structured data with a declared format? → A: Both: JSON text plus structured data with a
  declared format; errors carry the category in the message and in a structured field
  (structured field later dropped, see the planning revisit below).
- Q: Should `/health` report only that the MCP server is running, as today, or also whether the
  hub can be reached? → A: Always 200 with status, server and version, plus a `hub` field
  (`reachable`/`unreachable`) from one quick, time-limited hub request.
- Q: Should this feature also set the server up to run on the Pi as a background service that
  starts at boot and restarts after a crash, or only deliver the executable? → A: Executable,
  clean shutdown on a stop signal, a service definition, and an install script that sets up and
  enables the service on the Pi.
- Q: Should the install script run on the Pi itself after you copy the files there, or on your
  Windows machine and deploy to the Pi over SSH? → A: On the Pi: copy the install script over,
  then run it there with the hub URL; it downloads the matching pre-built executable from the
  project's GitHub Releases itself (the service definition is embedded in the script).
- Q: (revisited during planning, 2026-09-28) Must the error category also be in a structured field,
  given the SDK's simple tool API can't attach one? → A: No, the text prefix is enough for now;
  revisit with the SDK's low-level API if a client needs a structured field.

## User Scenarios & Testing *(mandatory)*

Actors:

- **Agent**: an AI assistant (VS Code Copilot, Claude, MCP Inspector) connected to the server,
  choosing and calling tools.
- **Operator**: the person who runs the server, on a dev machine or on the Raspberry Pi next to
  the hub, and configures assistants to connect to it.
- **Maintainer**: the person who changes the server when the hub API evolves.

### User Story 1 - Agents discover and inspect the audio system unchanged (Priority: P1)

An agent that was connected to the old server connects to the new one at the same address and
sees the same 24 tools under the same names. It lists and fetches inputs, outputs, groups,
routes and the master-mute state, and gets the same information it got before.

**Why this priority**: read-only discovery is what every agent does first and is the smallest
slice that proves the new server works. Nothing else is usable without it.

**Independent Test**: start the new server against a hub (or a fake hub), connect MCP Inspector
to the same URL as before, check that the tool list matches the old one exactly, and call each
list/get tool.

**Acceptance Scenarios**:

1. **Given** the new server is running with the same flags as the old one, **When** an agent
   lists tools, **Then** it sees exactly the 24 tool names in the Tool Inventory below, each
   with a description and described inputs.
2. **Given** the hub has outputs A and B (B disabled), **When** the agent calls `listOutputs`
   without `includeDisabled`, **Then** the result contains only A, with every field the hub
   API defines for an output.
3. **Given** a route exists, **When** the agent calls `getRoute` with its ID, **Then** the
   result contains every field the hub API defines for a route.
4. **Given** the agent calls a read-only tool, **When** it inspects the tool's metadata,
   **Then** the tool is marked read-only.

---

### User Story 2 - Agents control volume, mute and availability (Priority: P1)

An agent changes the volume of an output or group, mutes or unmutes outputs, groups or the whole
system, and enables or disables inputs, outputs and groups, exactly as with the old server.

**Why this priority**: these are the everyday actions people ask an assistant to perform
("turn the kitchen down", "mute everything").

**Independent Test**: against a fake hub, call each control tool with valid input and verify the
hub receives the matching request and the tool returns the hub's resulting state.

**Acceptance Scenarios**:

1. **Given** output `kitchen` exists, **When** the agent calls `setOutputVolume` with volume
   35, **Then** the hub receives a volume change to 35 for `kitchen` and the tool returns the
   new volume state.
2. **Given** any volume tool, **When** the agent passes volume 150 or -1, **Then** the tool
   returns an input error and the hub receives no request.
3. **Given** the system is unmuted, **When** the agent calls `setMasterMute` with muted=true,
   **Then** the hub's master mute is set and the tool returns the new state.
4. **Given** a state-changing tool, **When** the agent inspects its metadata, **Then** it is
   marked as not read-only, and as idempotent where repeating the same call has no further
   effect (set volume, set mute, set enabled).

---

### User Story 3 - Agents route and play audio (Priority: P1)

An agent plays a URI on an output or group, creates routes from inputs to targets, transfers,
pauses, resumes and deletes routes, and creates or deletes ephemeral inputs.

**Why this priority**: playback and routing are the core value of a multiroom system; they are
the flows the constitution names as must-test.

**Independent Test**: against a fake hub, run playback, then create → transfer → pause → resume →
delete a route, and create → delete an input; verify each hub request and each tool result.

**Acceptance Scenarios**:

1. **Given** output `livingroom` exists, **When** the agent calls `playback` with a URI,
   target `livingroom` and target type `SINGLE_OUTPUT`, **Then** the hub starts playback and
   the tool returns the hub's playback response (including the created input and route).
2. **Given** an active route, **When** the agent calls `transferRoute` to group `downstairs`
   (`OUTPUT_GROUP`), **Then** the route moves and the tool returns the updated route.
3. **Given** an active route, **When** the agent calls `deleteRoute`, **Then** the route is
   stopped and the tool returns a success confirmation (the hub returns no body).
4. **Given** a target type other than `SINGLE_OUTPUT` or `OUTPUT_GROUP`, **When** any routing
   or playback tool is called, **Then** it returns an input error and the hub receives no
   request.
5. **Given** a tool that removes or stops something (`deleteInput`, `deleteRoute`), **When**
   the agent inspects its metadata, **Then** it is marked destructive.

---

### User Story 4 - Agents get actionable errors (Priority: P2)

When something goes wrong (the hub is down, slow, rejects the request, or the ID doesn't exist),
the agent receives an error result that says what happened and what kind of failure it was,
and the server keeps serving other requests.

**Why this priority**: agents decide whether to retry, fix their input or tell the user based on
the error. The old server had no timeout, so a hung hub made tools hang.

**Independent Test**: against a fake hub that returns 404, 400 with problem details, 503, a
malformed body, and one that never answers, call tools and inspect each error result; confirm a
concurrent call to a healthy endpoint still succeeds.

**Acceptance Scenarios**:

1. **Given** no output `garage` exists, **When** the agent calls `getOutput` with `garage`,
   **Then** it receives an error result naming the not-found category and the missing output.
2. **Given** the hub rejects a request with problem details, **When** the tool returns,
   **Then** the error result carries the hub's explanation and a validation category.
3. **Given** the hub does not respond, **When** a tool is called, **Then** it returns a
   timeout error within the time bound and does not hang.
4. **Given** the hub address is unreachable, **When** a tool is called, **Then** it returns a
   network-category error, and the server stays up.
5. **Given** the agent cancels a call in progress, **When** cancellation arrives, **Then** the
   pending hub request is abandoned.

---

### User Story 5 - Operator runs it anywhere with the same setup (Priority: P2)

The operator starts the server with the same flags as before, points assistants at the same
URL, and checks health the same way, on Windows or on the Raspberry Pi, from a single
self-contained executable with no language runtime to install.

**Why this priority**: existing client configurations (VS Code, Claude) must keep working, and
removing the runtime requirement is a main reason for the rewrite.

**Independent Test**: build the executable for Windows and for 64-bit ARM Linux, copy the ARM
build to the Pi, start it with `--multiroom-url`, connect an assistant with the existing config,
and request `/health`.

**Acceptance Scenarios**:

1. **Given** the server is started without `--multiroom-url`, **Then** it prints usage and
   exits with a non-zero status without starting.
2. **Given** `--help` or `-h`, **Then** it prints usage and exits with status 0.
3. **Given** no `--port`, **Then** it listens on 3001; **given** `--port 4000`, it listens on
   4000.
4. **Given** no `--host`, **Then** assistants on other LAN machines can connect; **given**
   `--host 127.0.0.1`, **Then** only connections from the same machine are accepted.
5. **Given** the server is running, **When** the operator requests `/health`, **Then** the
   response reports status ok, the server name `sonora-mcp`, the build's version, and
   `hub: reachable`.
6. **Given** the hub is down, **When** the operator requests `/health`, **Then** the response
   is still successful (200) and reports `hub: unreachable` within 2 seconds.
7. **Given** a release build, **Then** the version in `/health`, the MCP server information and
   the startup log is the same value.
8. **Given** a trailing slash in `--multiroom-url`, **Then** hub calls still resolve correctly.
9. **Given** the server is handling a tool call, **When** it receives a stop signal, **Then** it
   stops accepting new connections, lets the call finish (bounded by the hub time limit), and
   exits with status 0.
10. **Given** a Pi with only the install script copied to it, **When** the operator runs it
    with the hub URL, **Then** the server runs as a service, starts at boot, restarts
    after a crash, and running the script again updates the existing installation instead of
    duplicating it.

---

### User Story 6 - Maintainer has one client and no Node.js left (Priority: P3)

The maintainer finds a single implementation: no Node.js sources, build files or local copy of
the hub spec remain, the README describes only the new build, and an automated test catches any
tool that drifts from the hub spec shipped with the shared client.

**Why this priority**: this delivers the long-term saving (one client to maintain), but only
after the server is proven equivalent.

**Independent Test**: inspect the repository for Node.js artifacts; change a tool's volume range
locally and confirm the conformance test fails.

**Acceptance Scenarios**:

1. **Given** the feature is complete, **Then** the repository contains no `package.json`,
   `package-lock.json`, `tsconfig.json`, `src/*.ts`, `scripts/update-openapi.mjs` or local
   `openapi.json`, and README contains no `npm` instructions.
2. **Given** a tool whose input constraint differs from the hub spec, **When** the test suite
   runs, **Then** it fails and names the tool and the mismatch.

---

### Edge Cases

- An ID containing characters that need escaping in a URL path (spaces, `/`, `#`) is passed to
  a get/set tool: the hub receives the ID intact, not a different path.
- `listRoutes` is called with none, some, or all of its optional filters (`status`,
  `inputId`, `targetId`): only the given filters are sent.
- `deleteInput` targets a static (configured) input: the hub's rejection comes back as a
  validation error, not a generic failure.
- `playback` is called with a URI the hub cannot reach: the error says the audio source could
  not be reached.
- Optional fields (`createInput.enabled`, `createInput.autoRemove`, `playback.displayName`,
  `playback.volume`) are omitted: the hub receives no value for them and applies its defaults,
  rather than receiving `false` or `0`.
- The hub returns a body that cannot be read as the expected data (not JSON, wrong shape):
  the tool returns a malformed-response error rather than partial or invented data.
- A handler fails unexpectedly: that call returns an error result; the server and other calls
  continue.
- Several agents call tools at the same time: each call is independent (no shared session).

## Requirements *(mandatory)*

### Functional Requirements

**Tool surface**

- **FR-001**: The server MUST expose exactly the 24 tools in the Tool Inventory, with the same
  names and the same input fields (names, types, required/optional) as the current server.
- **FR-002**: Each tool's input constraints (required fields, enumerations such as target type,
  ranges such as volume 0–100) MUST match the hub API specification, and input that violates
  them MUST be rejected with an input error before any hub request is made.
- **FR-003**: Every tool MUST have a description stating what it does, its side effects and
  what it returns; every input field MUST have a description.
- **FR-004**: Every tool MUST be marked read-only or state-changing; state-changing tools MUST
  additionally be marked idempotent or destructive where that applies (see Tool Inventory).
- **FR-005**: A successful tool result MUST contain the hub's response data for that operation
  with every field the hub API specification defines for it. Operations for which the hub
  returns no body MUST return an explicit success confirmation.
- **FR-005a**: Every successful result MUST carry the data twice: as JSON text (so clients that
  read only text keep working) and as structured data. Each tool MUST declare the format of its
  structured data, and the structured data MUST conform to it.

**Hub interaction**

- **FR-006**: Every hub interaction MUST use the shared hub client maintained in sonora-cli; the
  server MUST NOT call hub endpoints any other way.
- **FR-007**: Every hub call MUST be bounded by a time limit and MUST be abandoned when the
  agent's request is cancelled.
- **FR-008**: The server MUST hold no hub state between calls and MUST keep no per-client
  session; each tool call reflects the hub's state at that moment.

**Errors & logging**

- **FR-009**: A failed hub call MUST produce an error result (not a protocol failure) containing
  a human-readable message and a stable error category name (at least: not found, validation,
  network/timeout, hub error, service unavailable, source unreachable, malformed response).
  The category MUST appear at the start of the message text (e.g.
  `NotFound: output garage not found`).
- **FR-010**: Where the hub supplies problem details, the error message MUST include the hub's
  explanation.
- **FR-011**: A failure or unexpected fault in one tool call MUST NOT stop the server or affect
  other calls.
- **FR-012**: Each tool call MUST be logged with tool name, arguments, outcome (success or error
  category) and duration.

**Operation**

- **FR-013**: The server MUST accept `--multiroom-url <url>` (required), `--port <port>`
  (default 3001) and `-h`/`--help`, with the same meaning as today; `--help` MUST exit with
  status 0, and a missing or invalid flag value MUST stop startup with usage text and exit status
  2 (Go's usage-error convention; the old server exited with 1, which no known client or service
  definition depends on).
- **FR-013a**: The server MUST accept connections on all network addresses by default (as
  today), and MUST accept an optional `--host <address>` flag that restricts the listening
  address (e.g. `127.0.0.1` for local-only use). An invalid address MUST stop startup with
  usage text and exit status 2 (as for any invalid flag value, FR-013).
- **FR-014**: The server MUST serve the MCP protocol over streamable HTTP at `/mcp`, without
  sessions, so existing client configurations work unchanged.
- **FR-015**: The server MUST serve `GET /health` returning status, server name, version and a
  `hub` field (`reachable` or `unreachable`) determined by one time-limited, read-only hub
  request per health check. The response MUST be successful (200) whenever the server itself is
  running, regardless of the hub's state.
- **FR-016**: The version MUST be set once per build and reported identically by `/health`, the
  MCP server information and the startup log.
- **FR-017**: The server MUST build as a single self-contained executable for Windows and for
  64-bit ARM Linux, requiring no separately installed language runtime.
- **FR-017a**: On a stop signal (interrupt or terminate), the server MUST stop accepting new
  connections, let in-flight calls finish within the hub time limit, and exit with status 0.
- **FR-017b**: The repository MUST provide a service definition for Linux (the Pi) that starts
  the server at boot with the configured hub URL, port and host, and restarts it after a crash.
- **FR-017c**: The repository MUST provide a single install script — embedding the service
  definition and a pinned release version — that is copied to the Pi as the only file and run
  there. It takes the hub URL (and optionally port, host and a version override) as input,
  downloads the matching pre-built executable from the project's GitHub Releases, installs the
  executable and the service definition, enables and starts the service, and is safe to re-run
  to upgrade or reconfigure. It MUST NOT be required for building or for running on Windows.
  Remote deployment from the development machine is out of scope.

**Maintenance**

- **FR-018**: An automated test MUST verify every tool against the hub API specification shipped
  with the shared client version in use, and fail with the tool name and mismatch on drift.
- **FR-019**: The automated test suite MUST run without a real hub or network access, on Windows
  and Linux, and MUST cover the core flows (playback; route create, transfer, pause, delete;
  volume and mute) and error translation.
- **FR-020**: All Node.js sources, build configuration, dependency manifests, the local hub
  specification copy and its update script MUST be removed; README MUST document only the new
  build, cross-build for the Pi, service installation, run and client-configuration steps.

### Tool Inventory

The contract to preserve. Inputs marked `?` are optional. Target type is `SINGLE_OUTPUT` or
`OUTPUT_GROUP`; volume is an integer 0–100.

| Tool | Inputs | Kind |
|---|---|---|
| `listInputs` | `includeDisabled?` | read-only |
| `getInput` | `inputId` | read-only |
| `createInput` | `inputId`, `displayName`, `uri`, `enabled?`, `autoRemove?` | state-changing |
| `deleteInput` | `inputId` | destructive |
| `setInputEnabled` | `inputId`, `enabled` | idempotent |
| `listOutputs` | `includeDisabled?` | read-only |
| `getOutput` | `outputId` | read-only |
| `setOutputVolume` | `outputId`, `volume` | idempotent |
| `setOutputMute` | `outputId`, `muted` | idempotent |
| `setOutputEnabled` | `outputId`, `enabled` | idempotent |
| `listGroups` | `includeDisabled?` | read-only |
| `getGroup` | `groupId` | read-only |
| `setGroupVolume` | `groupId`, `volume` | idempotent |
| `setGroupMute` | `groupId`, `muted` | idempotent |
| `setGroupEnabled` | `groupId`, `enabled` | idempotent |
| `listRoutes` | `status?`, `inputId?`, `targetId?` | read-only |
| `getRoute` | `routeId` | read-only |
| `createRoute` | `inputId`, `targetId`, `targetType` | state-changing |
| `deleteRoute` | `routeId` | destructive |
| `transferRoute` | `routeId`, `targetId`, `targetType` | state-changing |
| `setRoutePause` | `routeId`, `paused` | idempotent |
| `playback` | `uri`, `targetId`, `targetType`, `displayName?`, `volume?` | state-changing |
| `getMasterMute` | — | read-only |
| `setMasterMute` | `muted` | idempotent |

### Key Entities

- **Tool**: a named action an agent can call; has a description, an input schema with
  constraints, a kind (read-only / state-changing / idempotent / destructive) and maps to one hub operation.
- **Hub resource**: input, output, group, route, master mute, playback result; the data a tool
  returns, as defined by the hub API specification.
- **Tool result**: the hub data for one call, delivered as JSON text and as structured data
  in the tool's declared format.
- **Error result**: message prefixed with the error category, returned to the agent instead of
  data.
- **Build version**: one identifier per build, reported by health, server info and logs.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An assistant configured for the old server connects to the new one with zero
  configuration changes and lists exactly the same 24 tool names.
- **SC-002**: All 24 tools, exercised against a real hub (list, get, volume/mute, enable, route
  create/transfer/pause/delete, playback, master mute), return results containing the same
  information as the old server for the same calls.
- **SC-003**: With the hub unresponsive, every tool returns an error within 10 seconds; none
  hang.
- **SC-004**: 100% of failed hub calls return an error whose text starts with a category name
  (e.g. `NotFound:`, `Validation:`, `Network:`), and input rejected by a tool's schema returns
  the MCP SDK's validation error (`validating "arguments": …`) with no hub request; an agent
  can distinguish "not found", "bad input" and "hub unreachable" from the first token of the
  error text.
- **SC-005**: The Pi deployment needs one file copied and no runtime installed; the server
  starts and, with the hub up, answers `/health` with `hub: reachable` within 2 seconds.
- **SC-006**: The automated test suite passes with the network disabled, and introducing a
  deliberate tool/spec mismatch makes it fail.
- **SC-007**: Zero Node.js files remain in the repository after the feature is merged.
- **SC-008**: On a fresh Pi with internet access, the server is running as a service within 5
  minutes of copying the install script to it, and is running again within 1 minute of a reboot
  or a crash.

## Assumptions

- The implementation language is Go and the hub client is sonora-cli's public `hub` package
  with its embedded API specification, as decided in
  `docs/future/go-rewrite-shared-hub-client.md` and required by the constitution (Principle I).
- sonora-cli's `hub` package (PR #19, branch `010-public-hub-package`) already covers all 24
  operations and its response types carry every field the specification defines for them (checked for
  outputs and routes on 2026-09-27; the conformance test covers the rest). Development uses a
  local workspace against that branch; merging this feature requires a tagged sonora-cli
  release (constitution: dependency pinning).
- Tool results may differ from the old server in formatting (field order, whitespace) but not in
  content. Error message wording may differ; error content and categories are new.
- List results are an object wrapping the array (`{"inputs": [...]}`, `{"outputs": [...]}`,
  `{"groups": [...]}`, `{"routes": [...]}`) in both the structured data and the JSON text, because
  MCP clients require structured data to be an object. The old server returned the bare array as
  text; the items are unchanged. Agents read the text as JSON, so this is treated as a formatting
  change.
- The old server returned `{"success": true, "status": 204}` for no-body responses; an
  equivalent explicit confirmation is sufficient, exact wording is not preserved.
- Tool descriptions may be richer than the old ones (which were one-line summaries); names and
  inputs are what must not change.
- New tools for operations the shared client already supports (speak, voices, extensions,
  stop-all, stop by output/group) are out of scope and follow as a separate feature.
- Authentication, remote exposure, MCP resources, prompts and push notifications are out of
  scope; the server stays on a trusted home LAN (constitution: Security).
- The Pi runs a Linux distribution with a standard service manager (Raspberry Pi OS / systemd),
  as the hub's Pi does; the operator can run the install script with administrator rights.
- The time limit per hub call is the shared client's default (5 seconds), from one client shared
  by all 24 tools; none of them needs a longer bound. A future tool that does (e.g. text-to-speech)
  must also revisit the shutdown drain and SC-003.
- The Pi has outbound internet access to GitHub (`github.com`, `objects.githubusercontent.com`)
  at install time only, to download the release binary the install script fetches; no inbound
  exposure changes and no internet access is needed once the server is running.
