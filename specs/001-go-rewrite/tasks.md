---

description: "Task list for 001-go-rewrite"
---

# Tasks: Rewrite sonora-mcp on the Shared Hub Client

**Input**: Design documents from `specs/001-go-rewrite/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/tools.md](contracts/tools.md),
[contracts/server.md](contracts/server.md), [quickstart.md](quickstart.md)

**Tests**: REQUIRED. Constitution Principle V (Test-First, NON-NEGOTIABLE): every test task is
written first and must fail before its implementation task starts. All tests run offline with
`go test ./...`.

**Organization**: Tasks are grouped by user story so each story can be built and verified on its own.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: User story from spec.md (US1–US6)

## Conventions for every task

- Module `github.com/Sonora-Multiroom/sonora-mcp`; Go 1.27; `gofmt`-formatted; godoc on exported identifiers;
  errors wrapped with `%w`.
- Hub calls ONLY through `github.com/Sonora-Multiroom/sonora-cli/hub` (imported as `hub`), using the
  request `context.Context` and the `*http.Client` passed into `tools.Register`. No local hub types.
- Tools are registered with the `add` helper from T012 (typed `mcp.AddTool`, explicit input schema,
  panic recovery). Each tool also appends its `toolSpec{Name, Description, Kind, Method, Path}` to the
  package registry used by the conformance test (T015).
- Tool tests live next to the code in `internal/tools/*_test.go`, drive tools through an MCP client
  (harness T005) against the fake hub (T004), and assert: the hub request (method, path, query, body),
  `structuredContent`, the JSON text copy, and the tool's annotations.
- Error text on failure is `"<Category>: <message>"` (research R5). Input rejected by the schema returns
  the SDK's `validating "arguments": …` text and makes **no** hub request.
- Result shapes: single resources = the `hub` type; lists wrapped as `{"inputs": [...]}`,
  `{"outputs": [...]}`, `{"groups": [...]}`, `{"routes": [...]}`; deletes =
  `{"deleted": true, "inputId": "<id>"}` / `{"deleted": true, "routeId": "<id>"}`.
- Constraints (quote from data-model.md): path IDs (`inputId`, `outputId`, `groupId`, `routeId` in
  the URL path) are "string, `minLength: 1`" (implicit in the spec, research R4); body IDs
  (`createRoute.inputId`, `targetId`) are "string, `minLength: 1`" (explicit in the spec); the
  `listRoutes` filters `inputId`/`targetId` are plain optional strings (no `minLength`);
  `createInput.inputId` "additionally matches `^[a-zA-Z0-9\-_]{1,255}$`"; `volume` is "integer,
  0–100"; `targetType` is "enum `SINGLE_OUTPUT` | `OUTPUT_GROUP`"; `status` is "enum `STARTING` |
  `ACTIVE` | `STOPPING` | `STOPPED` | `FAILED`, optional"; `uri` and `createInput.displayName` are
  `minLength: 1`; `playback.displayName` is a plain optional string (no `minLength`).

---

## Phase 1: Setup

**Purpose**: Go module and repository basics. Node.js files stay untouched until Phase 8.

- [X] T001 Create `go.mod` with `module github.com/Sonora-Multiroom/sonora-mcp` and `go 1.27`; run
  `go get github.com/modelcontextprotocol/go-sdk@v1.8.0` and
  `go get github.com/Sonora-Multiroom/sonora-cli@010-public-hub-package` (pseudo-version of PR #19,
  allowed on this feature branch only; replaced by a tag in T069); commit `go.mod`/`go.sum`
- [X] T002 [P] Add Go ignores to `.gitignore`: `/sonora-mcp`, `/sonora-mcp.exe`, `/dist/`, `go.work`,
  `go.work.sum`; keep the existing Node and Spec Kit rules for now
- [X] T003 [P] Create `internal/version/version.go`: package `version` with
  `var Version = "dev"` and a godoc saying it is set via
  `-ldflags "-X github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=<v>"`
- [X] T003a [P] Commit the design note `docs/future/go-rewrite-shared-hub-client.md` on its own
  (`docs:` commit) so the references to it from spec.md, the constitution's Sync Impact Report and
  plan decisions resolve after merge
- [X] T003b [P] Commit `.vscode/mcp.json` on its own (`chore:` commit; it holds only
  `http://localhost:3001/mcp`, no secrets) so the reference client configuration used by T070 and
  SC-001 is in the repository; leave `.vscode/settings.json` untracked

---

## Phase 2: Foundational (blocking)

**Purpose**: test harness, tool plumbing, conformance check, minimal config/server/main so the binary
starts and lists tools. No tool exists yet at the end of this phase.

**⚠️ No user story work can begin until this phase is complete.**

### Test infrastructure

- [X] T004 [P] Create fake hub in `internal/tools/fakehub_test.go`: `newFakeHub(t)` wrapping
  `httptest.Server`; `handle(method, path string, status int, body string)` to register canned
  responses (unregistered routes → 404 with RFC 7807 body); records every request (method, path, raw
  query, body) for assertions; optional per-route delay to simulate a slow hub; fixture constants for a
  spec-shaped Output, Group, Input, Route, MasterMute, PlaybackResponse (all fields from the hub types)
- [X] T005 [P] Create MCP test harness in `internal/tools/harness_test.go`: `newTestSession(t, hubURL,
  client *http.Client)` builds `mcp.NewServer(&mcp.Implementation{Name: "sonora-mcp", Version: "test"},
  nil)`, calls `Register`, connects an `mcp.Client` over `mcp.NewInMemoryTransports()`, returns the
  `*mcp.ClientSession`; helpers `callTool(t, s, name, args)` → `*mcp.CallToolResult`,
  `toolByName(t, s, name)` → `*mcp.Tool` (from `ListTools`), `errorText(res)`

### Tool plumbing (tests first)

- [X] T006 [P] Write `internal/tools/schema_test.go`: `inputSchema[T]` infers an object schema from a
  struct and applies `withRange("volume", 0, 100)`, `withEnum("targetType", "SINGLE_OUTPUT",
  "OUTPUT_GROUP")`, `withMinLength("outputId", 1)`, `withPattern("inputId", …)`; unknown property
  name panics at startup (programming error); descriptions come from `jsonschema` tags
- [X] T007 Implement `internal/tools/schema.go` to pass T006 (uses `jsonschema.For[T](nil)` from
  `github.com/google/jsonschema-go/jsonschema`; constraints set on the inferred `*jsonschema.Schema`,
  because the `jsonschema` tag only carries the description — research R3)
- [X] T008 [P] Write `internal/tools/errors_test.go` table test for `toolError(err error) error` →
  message `"<Category>: <message>"`: `*hub.DecodeError` → `MalformedResponse`;
  `context.DeadlineExceeded` or a `net.Error` with `Timeout()` → `Timeout`; `hub.ClassNotFound`,
  `ClassInputNotFound`, `ClassTargetNotFound` → `NotFound`; `ClassValidation` → `Validation`;
  `ClassRouteFailed` → `RouteFailed`; `ClassSourceUnreachable` → `SourceUnreachable`;
  `ClassServiceUnavailable` → `ServiceUnavailable`; `ClassNetwork` → `Network`; `ClassHub` →
  `HubError`; a `*hub.APIError` with `Detail` keeps the hub's detail in the message (FR-010); a
  `*hub.StatusError` message includes the HTTP status
- [X] T009 Implement `internal/tools/errors.go` to pass T008 (pre-checks, then `hub.ClassifyError`;
  export the category names as constants)
- [X] T010 [P] Write `internal/tools/register_test.go`: a throwaway test tool registered via `add`
  appears in `ListTools` with the given description, the explicit input schema, the output schema
  inferred from `Out`, and annotations per Kind (read-only → `readOnlyHint: true`; idempotent →
  `readOnlyHint: false, destructiveHint: false, idempotentHint: true`; state-changing → both false;
  destructive → `destructiveHint: true`); invalid arguments return a `CallToolResult` with
  `isError: true` whose text starts `validating "arguments"` (a tool error, not a JSON-RPC error —
  Principle II, SC-004) and never invoke the handler; a handler that
  panics returns `isError` with text `Internal: unexpected server error` and a following call still
  succeeds; the registry records the tool's `toolSpec`
- [X] T011 [P] Write `internal/tools/logging_test.go`: middleware from `LogToolCalls(logger)` writes
  exactly one record per `tools/call` with `tool`, `args` (JSON), `outcome` (`ok`, the category
  prefix of an error result, or `InvalidInput` for SDK validation errors), `duration`; other methods
  (`tools/list`) are not logged
- [X] T012 Implement `internal/tools/register.go` to pass T010: `type Kind int` (`ReadOnly`,
  `Idempotent`, `StateChanging`, `Destructive`), `type toolSpec struct{Name, Description string;
  Kind Kind; Method, Path string}`, package registry `var registered []toolSpec`, and
  `func add[In, Out any](s *mcp.Server, spec toolSpec, in *jsonschema.Schema, h func(context.Context,
  In) (Out, error))` that builds `mcp.Tool{Name, Description, InputSchema: in, Annotations}`,
  wraps `h` with panic recovery (log stack via slog) and calls `mcp.AddTool`
- [X] T013 Implement `internal/tools/logging.go` to pass T011 (`mcp.Middleware` for
  `Server.AddReceivingMiddleware`, slog text records)
- [X] T014 Create `internal/tools/tools.go`: `func Register(s *mcp.Server, client *http.Client,
  hubURL string)` that calls one `registerX` function per file (inputs, outputs, groups, routes,
  playback); start with empty stubs so the package compiles
- [X] T015 Write `internal/tools/conformance_test.go` (research R4): load `api.Spec` from
  `github.com/Sonora-Multiroom/sonora-cli/api`; for every entry in `registered`: the `Method`+`Path`
  operation exists in the spec; every input property maps to a path/query parameter or a
  request-body property of that operation; `required`, `enum`, `minimum`, `maximum`, `minLength`,
  `pattern` equal the spec's; failures name tool, field, tool value and spec value. Spec
  `["boolean","null"]`-style types count as "optional" for `required`. Path parameters are
  compared as if the spec declared `minLength: 1` (an empty path segment changes the route,
  research R4); query parameters and body properties get no such allowance. Also (FR-005), for
  every operation with a JSON success response: every property of the spec's response schema is
  a property of the tool's `outputSchema` (unwrap the list envelope — `inputs`, `outputs`,
  `groups`, `routes` — to its item schema first; skip 204 operations, whose result is the
  `deleted` confirmation); failures name the tool and the missing field. Also (FR-003), via the
  T005 harness's `ListTools`: every tool has a non-empty description and every input-schema
  property has a non-empty `description`; failures name the tool and field
- [X] T015a [P] Add `internal/tools/architecture_test.go` (FR-006, FR-008), a guard in place before
  any tool or server code is written: using `go/parser`, walk every non-test `.go` file under
  `internal/` and `cmd/` (from the module root, `../..`; this covers `internal/server/health.go` and
  `main.go` too) and fail, naming file and line, if it calls `http.Get`, `http.Post`, `http.Head`,
  `http.NewRequest` or references `http.DefaultClient` (all hub access MUST go through `hub.*`,
  Principle I), or declares a package-level `var` holding a `hub` resource type (Principle III — no
  cached hub state); it runs as part of `go test ./...`, so it needs no separate merge-gate step

### Minimal config, server and entry point (tests first)

- [X] T016 [P] Write `internal/config/config_test.go` for `Parse(args []string) (Config, error)`:
  missing `--multiroom-url` → usage error; URL must be `http`/`https` with a host (trailing `/`
  accepted); `--port` default 3001, must be integer 1–65535; `-h`/`--help` → `ErrHelp`; unknown flag →
  usage error
- [X] T017 Implement `internal/config/config.go` to pass T016 (stdlib `flag.FlagSet` with
  `ContinueOnError`; `Usage()` text matching contracts/server.md)
- [X] T018 [P] Write `internal/server/server_test.go` end-to-end: `httptest.NewServer(server.Handler(
  mcpServer, hubClient, hubURL))` + `mcp.StreamableClientTransport` → initialize and `ListTools`
  succeed without a session ID; `GET /mcp` → 405; `GET /nope` → 404
- [X] T019 Implement `internal/server/server.go` to pass T018: `Handler(s *mcp.Server, client
  *http.Client, hubURL string) http.Handler` (client and URL are used by `/health`, T055) with a mux
  mounting `mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s },
  &mcp.StreamableHTTPOptions{Stateless: true, PropagateRequestCancellation: true})` at `/mcp`
  (research R6; no CORS headers; SDK localhost protection left on) and 404 for everything else
- [X] T019a [P] Write `cmd/sonora-mcp/main_test.go` for `run(ctx context.Context, args []string,
  stdout, stderr io.Writer) int` (spec US5 AS1–2, FR-013; `ctx` lets T053 stand in for a stop signal): `-h` and `--help` → usage on stdout, returns 0;
  no `--multiroom-url` → usage on stderr, returns 2; invalid `--port` → message + usage on
  stderr, returns 2; `--port` already bound by the test (listen error) → message on stderr,
  returns 1
- [X] T020 Implement `cmd/sonora-mcp/main.go` to pass T019a: `main` only calls
  `os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))`; `run` does `config.Parse(args)` (help → usage
  to stdout, return 0; error → message + usage to stderr, return 2); `hub.NewClient()`; `mcp.NewServer` with
  `Implementation{Name: "sonora-mcp", Version: version.Version}`; `tools.Register`;
  `AddReceivingMiddleware(tools.LogToolCalls(logger))`; slog text logger on stderr; startup line
  with version, hub URL, listen address, tool count; `http.Server.ListenAndServe` on `:<port>`
  (listen error → stderr, return 1)

**Checkpoint**: `go test ./...` passes; `go run ./cmd/sonora-mcp --multiroom-url http://localhost:8080`
starts and MCP Inspector connects (0 tools).

---

## Phase 3: User Story 1 — Agents discover and inspect the audio system (P1) 🎯 MVP

**Goal**: the 10 read-only tools behave as before, with structured results.

**Independent Test**: MCP Inspector against a hub (or the fake hub in tests) lists the read-only tools
and each returns every field the spec defines (spec US1).

### Tests (write first, must fail)

- [X] T021 [P] [US1] Write read-tool tests in `internal/tools/inputs_read_test.go`: `listInputs`
  (no args → `GET /api/v2/inputs` without `includeDisabled`; `includeDisabled: true` →
  `?includeDisabled=true`; result `{"inputs": [...]}` with all Input fields) and `getInput`
  (`GET /api/v2/inputs/{inputId}`; an ID with a space, a `/` and a `#` reaches the hub
  path-escaped; result is the Input); both annotated read-only; plus one `listInputs` call through a
  harness built with a trailing-slash hub URL (`fakeHub.URL + "/"`) reaching the hub at exactly
  `/api/v2/inputs` (spec US5 AS8)
- [X] T022 [P] [US1] Write `internal/tools/outputs_read_test.go`: `listOutputs`, `getOutput`, same
  pattern as T021 (`GET /api/v2/outputs`, `GET /api/v2/outputs/{outputId}`, wrapper `outputs`); the
  default call sends no `includeDisabled` query parameter and `includeDisabled: true` sends
  `?includeDisabled=true`; the fake hub returns only output A for the default call and A plus the
  disabled B for `?includeDisabled=true`, and each result contains exactly the outputs the hub
  returned (filtering is the hub's job; spec US1 AS2)
- [X] T023 [P] [US1] Write `internal/tools/groups_read_test.go`: `listGroups`, `getGroup`
  (`GET /api/v2/groups`, `GET /api/v2/groups/{groupId}`, wrapper `groups`)
- [X] T024 [P] [US1] Write `internal/tools/routes_read_test.go`: `listRoutes` sends only the filters
  given (none; `status` only; all three `status`, `inputId`, `targetId`); `status` outside
  "enum `STARTING` | `ACTIVE` | `STOPPING` | `STOPPED` | `FAILED`" is rejected with no hub request;
  wrapper `routes`; `getRoute` (`GET /api/v2/routes/{routeId}`) returns every Route field including
  `startedAt: null`
- [X] T025 [P] [US1] Write `internal/tools/mastermute_read_test.go`: `getMasterMute` (no inputs,
  `GET /api/v2/master-mute`, result `{"muted": false}`), read-only

### Implementation

- [X] T026 [P] [US1] Implement `listInputs`, `getInput` in `internal/tools/inputs.go` (inputs:
  `includeDisabled?: bool`; `inputId` "string, `minLength: 1`"; `hub.ListInputs`, `hub.GetInput`);
  descriptions per Principle II
- [X] T027 [P] [US1] Implement `listOutputs`, `getOutput` in `internal/tools/outputs.go`
  (`hub.ListOutputs`, `hub.GetOutput`)
- [X] T028 [P] [US1] Implement `listGroups`, `getGroup` in `internal/tools/groups.go`
  (`hub.ListGroups`, `hub.GetGroup`)
- [X] T029 [P] [US1] Implement `listRoutes`, `getRoute` in `internal/tools/routes.go`
  (`hub.ListRoutes(ctx, c, url, status, inputID, targetID)`, `hub.GetRoute`)
- [X] T030 [P] [US1] Implement `getMasterMute` in `internal/tools/playback.go` (`hub.GetMasterMute`)
- [X] T031 [US1] Wire the five `registerX` functions into `Register` in `internal/tools/tools.go`;
  run `go test ./internal/tools/` including the conformance test for the 10 tools

**Checkpoint**: the MVP. 10 read-only tools pass their tests and the conformance check.

---

## Phase 4: User Story 2 — Agents control volume, mute and availability (P1)

**Goal**: the 8 idempotent control tools.

**Independent Test**: against the fake hub, each control tool sends the matching request and returns
the hub's new state; out-of-range volume makes no request (spec US2).

### Tests (write first, must fail)

- [X] T032 [P] [US2] Write `internal/tools/outputs_control_test.go`: `setOutputVolume`
  (`PUT /api/v2/outputs/{outputId}/volume` body `{"volume":35}`, returns OutputVolume; `volume` 150
  and -1 and 35.5 rejected with no hub request — "integer, 0–100"), `setOutputMute`
  (`PUT …/mute` `{"muted":true}`, then `{"muted":false}` — the body must contain the `false` key,
  not drop it), `setOutputEnabled` (`PUT …/enabled` `{"enabled":false}`); all annotated idempotent
- [X] T033 [P] [US2] Write `internal/tools/groups_control_test.go`: `setGroupVolume`,
  `setGroupMute`, `setGroupEnabled` (paths `/api/v2/groups/{groupId}/volume|mute|enabled`), same
  assertions as T032, including `{"muted":false}` for `setGroupMute`
- [X] T034 [P] [US2] Write `internal/tools/inputs_enabled_test.go`: `setInputEnabled`
  (`PUT /api/v2/inputs/{inputId}/enabled`), idempotent
- [X] T035 [P] [US2] Write `internal/tools/mastermute_set_test.go`: `setMasterMute`
  (`PUT /api/v2/master-mute` `{"muted":true}`, then `{"muted":false}` with the `false` key present),
  idempotent

### Implementation

- [X] T036 [P] [US2] Implement `setOutputVolume`, `setOutputMute`, `setOutputEnabled` in
  `internal/tools/outputs.go` (`withRange("volume", 0, 100)`; `hub.SetOutputVolume`,
  `hub.SetOutputMuted`, `hub.SetOutputEnabled`)
- [X] T037 [P] [US2] Implement `setGroupVolume`, `setGroupMute`, `setGroupEnabled` in
  `internal/tools/groups.go` (`hub.SetGroupVolume`, `hub.SetGroupMuted`, `hub.SetGroupEnabled`)
- [X] T038 [P] [US2] Implement `setInputEnabled` in `internal/tools/inputs.go`
  (`hub.SetInputEnabled`)
- [X] T039 [P] [US2] Implement `setMasterMute` in `internal/tools/playback.go`
  (`hub.SetMasterMute`)

**Checkpoint**: 18 tools; conformance test green.

---

## Phase 5: User Story 3 — Agents route and play audio (P1)

**Goal**: the 6 routing, playback and input-lifecycle tools.

**Independent Test**: playback, then create → transfer → pause → resume → delete a route, and
create → delete an input, against the fake hub (spec US3).

### Tests (write first, must fail)

- [X] T040 [P] [US3] Write `internal/tools/routes_write_test.go`: `createRoute` (`POST /api/v2/routes`
  body `{"inputId","targetId","targetType"}`, state-changing), `transferRoute`
  (`POST /api/v2/routes/{routeId}/transfer` body `{"targetId","targetType"}`, state-changing),
  `setRoutePause` (`PUT /api/v2/routes/{routeId}/pause` `{"paused":true}` to pause, then
  `{"paused":false}` to resume with the `false` key present, idempotent), `deleteRoute`
  (`DELETE /api/v2/routes/{routeId}`, hub 204 → `{"deleted": true, "routeId": "<id>"}`,
  destructive); `targetType` `"SPEAKER"` rejected with no hub request
- [X] T041 [P] [US3] Write `internal/tools/inputs_write_test.go`: `createInput` (`POST /api/v2/inputs`;
  with `enabled`/`autoRemove` omitted the request body has **no** `enabled`/`autoRemove` keys; with
  them given they are sent; `inputId` `"bad id!"` rejected — pattern `^[a-zA-Z0-9\-_]{1,255}$`),
  `deleteInput` (`DELETE /api/v2/inputs/{inputId}`, 204 → `{"deleted": true, "inputId": "<id>"}`,
  destructive; hub 400 for a static input → text starts `Validation:`)
- [X] T042 [P] [US3] Write `internal/tools/playback_test.go`: `playback` (`POST /api/v2/play` body with
  `uri`, `targetId`, `targetType`; `displayName`/`volume` omitted when not given, sent when given;
  `volume` 101 rejected; result is PlaybackResponse with `inputId`, `route`, `message`;
  state-changing; hub 502 → text starts `SourceUnreachable:`)

### Implementation

- [X] T043 [P] [US3] Implement `createRoute`, `transferRoute`, `setRoutePause`, `deleteRoute` in
  `internal/tools/routes.go` (`hub.CreateRoute`, `hub.TransferRoute`, `hub.SetPauseState`,
  `hub.DeleteRoute`)
- [X] T044 [P] [US3] Implement `createInput`, `deleteInput` in `internal/tools/inputs.go`
  (`hub.CreateInput` with `Enabled`/`AutoRemove` as `*bool` passed through; `hub.DeleteInput`)
- [X] T045 [P] [US3] Implement `playback` in `internal/tools/playback.go` (`hub.Playback`;
  `DisplayName`/`Volume` pointers passed through)

**Checkpoint**: all 24 tools; conformance test green for all.

---

## Phase 6: User Story 4 — Agents get actionable errors (P2)

**Goal**: every failure returns a categorized, specific error within the SC-003 bound (10s); the
server stays up.

**Independent Test**: fake hub returning 404, 400 with problem details, 503, a malformed body, and
never answering; plus a concurrent healthy call (spec US4).

### Tests (write first, must fail if behaviour is missing)

- [X] T046 [P] [US4] Write `internal/tools/errors_scenarios_test.go`: `getOutput garage` with hub 404
  → `NotFound:` and the message names `garage`; `setOutputVolume` with hub 400 problem
  `{"detail":"output is disabled"}` → `Validation: output is disabled`; `playback` with 503 →
  `ServiceUnavailable:`; `listOutputs` with body `not json` → `MalformedResponse:`; `getOutput`
  with a wrong-shape JSON body (`[]`, and `{"volume":"loud"}`) → `MalformedResponse:` with no
  partial data in the result (spec Edge Cases); `listOutputs` against a closed port → `Network:`
- [X] T047 [P] [US4] Write `internal/tools/timeout_test.go`: with the harness given an
  `http.Client{Timeout: 200ms}` and a fake-hub delay of 2 s, `listOutputs` returns `Timeout:` in under
  1 s; meanwhile a concurrent `getMasterMute` on a non-delayed route succeeds (FR-011, SC-003)
- [X] T048 [P] [US4] Write `internal/tools/cancel_test.go`: cancelling the client call's context while
  the fake hub delays causes the fake hub to observe its request context cancelled (FR-007)
- [X] T048a [P] [US4] Extend `internal/server/server_test.go` (from T018) with cancellation over
  the real `/mcp` handler (FR-007, US4 AS5, research R6): build the server with `tools.Register`
  against an `httptest` hub handler that blocks until its request context is done (a local helper;
  the `internal/tools` fake hub is test-only and not importable); connect with
  `mcp.StreamableClientTransport`, assert the negotiated protocol version is ≥ `2026-07-28`, call
  `getMasterMute`, cancel the call's context, and assert the hub handler observes cancellation well
  before the 5 s client timeout. Fails if `PropagateRequestCancellation` is dropped from T019

### Implementation

- [X] T049 [US4] Make T046–T048a pass, changing only `internal/tools/errors.go`,
  `internal/tools/register.go` and, for T048a, `internal/server/server.go` (keep error text `"<Category>: <message>"`; never return protocol
  errors for hub failures); if they already pass, record that and change nothing

**Checkpoint**: error behaviour verified across categories, timeout, cancellation and concurrency.

---

## Phase 7: User Story 5 — Operator runs it anywhere with the same setup (P2)

**Goal**: `--host`, `/health` with hub reachability, one version everywhere, clean shutdown, Pi
service.

**Independent Test**: build for Windows and linux/arm64, run on the Pi via the install script, connect
an assistant with the existing config, request `/health` (spec US5, quickstart §3, §6–§8).

### Tests (write first, must fail)

- [X] T050 [P] [US5] Extend `internal/config/config_test.go`: `--host` default empty (all addresses);
  accepts `127.0.0.1`, `::1`, `localhost`; rejects `not a host!` with a usage error;
  `Config.ListenAddr()` = `:3001`, `127.0.0.1:3001`, `[::1]:3001`
- [X] T051 [P] [US5] Write `internal/server/health_test.go`: `GET /health` → 200,
  `Content-Type: application/json`, body `{"status":"ok","server":"sonora-mcp","version":<v>,
  "hub":"reachable"}` when `hub.GetMasterMute` succeeds; `"hub":"unreachable"` (still 200) when the
  fake hub is down or delays 5 s, answered in ≤ 2.5 s (the 2 s bound of spec US5 AS6 / SC-005 is
  enforced by the `context.WithTimeout` in T055; the extra 0.5 s only absorbs test-scheduling
  jitter)
- [X] T052 [P] [US5] Write `internal/server/shutdown_test.go`: `Run(ctx, srv)` serving on a free port;
  start a tool call against a hub delayed 300 ms, cancel `ctx` (simulated signal); the call completes
  successfully, new connections are refused, `Run` returns nil within 6 s
- [X] T053 [P] [US5] Extend `cmd/sonora-mcp/main_test.go` (from T019a): the version string used by the MCP
  `Implementation`, `/health` and the startup log line all come from `version.Version` (build a server
  via the same constructor `main` uses and compare); the `*http.Client` that constructor passes to
  the tools and `/health` has `0 < Timeout ≤ 5s` (it comes from `hub.NewClient()`; keeps every tool
  under SC-003's 10 s and inside the 6 s shutdown drain); `--host "not a host!"` → message + usage on
  stderr, returns 2 (FR-013a); stop-signal exit status (FR-017a, US5 AS9): `run` started with a
  cancellable `ctx` and a free `--port` serves `/health`, and after `ctx` is cancelled it returns
  0 within 6 s

### Implementation

- [X] T054 [US5] Add `--host` and `ListenAddr()` to `internal/config/config.go` (pass T050; update
  `Usage()`)
- [X] T055 [US5] Implement `internal/server/health.go` and mount `/health` in `server.go` (pass T051;
  `context.WithTimeout(r.Context(), 2*time.Second)` around `hub.GetMasterMute`; nothing cached)
- [X] T056 [US5] Implement `Run(ctx context.Context, srv *http.Server) error` in
  `internal/server/server.go` (pass T052: `ListenAndServe` in a goroutine; on `ctx.Done()`
  `srv.Shutdown` with a 6 s deadline; `http.ErrServerClosed` is not an error)
- [X] T057 [US5] Update `cmd/sonora-mcp/main.go`: listen on `cfg.ListenAddr()`;
  `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` on the `ctx` passed to `run`;
  `server.Run`; `run` returns 0 after drain (pass T053); log
  shutdown; factor server construction into a function used by T053
- [X] T058 [US5] Write the embedded systemd unit inside `deploy/pi/install.sh` per
  contracts/server.md and research R13 (a heredoc, not a separate file):
  `After=network-online.target`, `Wants=network-online.target`, `DynamicUser=yes`,
  `EnvironmentFile=/etc/default/sonora-mcp`, `ExecStart=/usr/local/bin/sonora-mcp --multiroom-url
  ${SONORA_HUB_URL} --port ${SONORA_PORT} $SONORA_HOST_ARG`, `Restart=on-failure`, `RestartSec=2`,
  `NoNewPrivileges=yes`, `ProtectSystem=strict`, `ProtectHome=yes`, `WantedBy=multi-user.target`
- [X] T059 [US5] (after T058, same file) Finish `deploy/pi/install.sh` (bash, `set -euo pipefail`)
  per contracts/server.md: a `RELEASE_TAG="dev"` placeholder near the top (set to the
  release tag only on `main`, by T072); flags `--hub-url` (required), `--port` (default 3001), `--host` (optional →
  `SONORA_HOST_ARG=--host <addr>`), `--version` (overrides `RELEASE_TAG`); checks root, systemd
  and `curl` present, and that `uname -m` is `aarch64` (else a clear "64-bit ARM OS required"
  message and non-zero exit, before downloading anything); downloads the binary with `curl -fsSL`
  from `https://github.com/Sonora-Multiroom/sonora-mcp/releases/download/<tag>/sonora-mcp-linux-arm64`
  (`<tag>` = `--version` or `RELEASE_TAG`) to a temp file in `/usr/local/bin` (`mktemp`, removed by
  an `EXIT` trap on failure), `chmod 0755`, then `mv -f` onto `/usr/local/bin/sonora-mcp` — an
  atomic rename, so re-running while the service is up doesn't fail with "Text file busy" and a
  failed download leaves the installed binary intact; fails with a clear message on a network error
  or non-2xx response; writes the T058 unit to `/etc/systemd/system/sonora-mcp.service` and the
  env file to `/etc/default/sonora-mcp`; `systemctl daemon-reload`; `enable` + `restart`
  (idempotent on re-run); prints `systemctl --no-pager status sonora-mcp`; run `shellcheck` on the
  file if available
- [X] T060 [US5] Verify cross-build with the workspace off, so the binary is built from the pushed
  sonora-cli pseudo-version in `go.mod` and not from a local checkout (re-run
  `go get github.com/Sonora-Multiroom/sonora-cli@010-public-hub-package` first if the branch has
  moved): `GOWORK=off GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-X
  github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=1.1.0-rc.1" -o dist/pi/sonora-mcp-linux-arm64
  ./cmd/sonora-mcp` succeeds; record the binary size for the PR description (Principle VI)
- [X] T060b [US5] Release with GoReleaser as in sonora-cli, without Scoop: `.goreleaser.yaml`
  (linux/darwin/windows × amd64/arm64, `CGO_ENABLED=0`, `-s -w`, version ldflag, tar.gz/zip
  archives, `checksums.txt`, `prerelease: auto`), `.github/workflows/release.yml` (on `v*` tags),
  `.github/workflows/test.yml` (gofmt, vet, `go test -race` on PRs) and `release.sh`; switch
  `deploy/pi/install.sh` to download `sonora-mcp_<version>_linux_arm64.tar.gz`, verify it against
  `checksums.txt` and extract the binary. Validated with `goreleaser check`, a snapshot build, and
  `install.sh` run in an arm64 container against the snapshot (install, re-run, unknown tag,
  tampered archive)
- [X] T060a [US5] Publish a **prerelease** for Pi validation (FR-017c), not the final release:
  `v1.1.0` is only cut from `main` (T072). Tag the current feature-branch commit `v1.1.0-rc.1` and
  push the tag; the release workflow (T060b) publishes it as a prerelease. Leave `RELEASE_TAG` in
  `install.sh` unchanged; confirm `curl -fsSL -o /dev/null -w '%{http_code}'
  https://github.com/Sonora-Multiroom/sonora-mcp/releases/download/v1.1.0-rc.1/sonora-mcp_1.1.0-rc.1_linux_arm64.tar.gz`
  returns `200`
- [X] T061 [US5] Validate on the Pi per quickstart.md §8 with `--version v1.1.0-rc.1` (install,
  `/health`, re-run with another port, kill → restart, reboot → running); time the install (SC-008:
  ≤ 5 min) and the restart after kill and after reboot (≤ 1 min); note results in the PR description
  — **Result (2026-09-28, Raspberry Pi, v1.1.0-rc.1, installed with the one-command `curl … |
  sudo bash -s --` form from the branch)**: install completed in seconds (SC-008 ≤ 5 min ✓);
  `/health` answered `1.1.0-rc.1`, hub reachable; re-run with `--port 3002` reconfigured the
  single unit; `systemctl kill -s KILL` → restarted by systemd (`NRestarts=1`, `RestartSec=2`) ✓;
  after reboot the service was active 18.3 s after boot ✓. Right after boot `/health` reported
  `hub: unreachable` for a few seconds until the hub came up, then `reachable` (no restart needed)

**Checkpoint**: operator stories verified locally and on the Pi.

---

## Phase 8: User Story 6 — One client and no Node.js left (P3)

**Goal**: remove Node.js, document the Go build, lock the tool inventory.

**Independent Test**: no Node artifacts tracked; a deliberate constraint change fails the tests
(spec US6, quickstart §2, §9).

- [X] T062 [US6] Add an inventory test to `internal/tools/conformance_test.go`: the registered tool
  names equal exactly the 24 names in contracts/tools.md (no more, no fewer)
- [X] T063 [US6] Drift check (manual, not committed): set `setOutputVolume`'s maximum to 150, confirm
  `go test ./internal/tools/` fails naming `setOutputVolume`, `volume`, `maximum`, 150 vs 100; revert
- [ ] T064 [US6] Delete Node.js sources and tooling. First run `git status --short`: tracked Node
  files must have no local changes, or `git rm` refuses them. The known case is an uncommitted,
  never-shipped `openapi:update` script in `package.json` and its "Updating the API Spec" section
  in `README.md` (constitution: legacy freeze); discard both with
  `git restore package.json README.md` (the script and README section are removed here and in
  T067 anyway). Then `git rm -r src/ package.json package-lock.json
  tsconfig.json openapi.json specs/sonora-mcp-plan.md` (the last is the Node-era plan, superseded
  by this feature's spec and plan; tracked files only — `git rm` aborts without removing anything if a
  path is untracked); then delete the untracked `scripts/update-openapi.mjs` (and `scripts/` if it
  is left empty), `run.sh` and `run.dev.sh` (one-line `npm` launchers, not replaced: README documents
  `go run`/`go build`), `dist/` and `node_modules/` from disk with plain `rm`; confirm with the
  quickstart §9 `git ls-files` check and that `scripts/update-openapi.mjs`, `run.sh` and `run.dev.sh`
  no longer exist
- [ ] T066 [P] [US6] Remove Node rules from `.gitignore` (`node_modules/`, `*.js`, `*.d.ts`,
  `*.d.ts.map`, `*.js.map`, `!.gitignore` block) keeping Go and Spec Kit rules
- [ ] T067 [US6] Rewrite `README.md` for Go: architecture diagram (unchanged), build with version
  ldflags, cross-build for linux/arm64, run with `--multiroom-url/--port/--host`, `/health` example
  with `hub`, client configs (VS Code, Claude Desktop — unchanged URLs), the 24-tool list, error
  categories, Pi install via `deploy/pi/install.sh` as one command (`curl -fsSL
  https://raw.githubusercontent.com/Sonora-Multiroom/sonora-mcp/main/deploy/pi/install.sh | sudo bash -s -- --hub-url <url>`, plus the options and the copy-and-run
  alternative), tech stack (Go 1.27, MCP Go SDK v1.8.0,
  sonora-cli `hub`); remove all `npm` and `openapi.json` sections

**Checkpoint**: repository is Go-only; `git ls-files` check in quickstart §9 is empty.

---

## Phase 9: Polish & merge readiness

- [X] T068 Run the merge gate from quickstart.md §2 on Windows with `GOWORK=off` (constitution
  Workflow §4): `gofmt -l .` empty, `go vet ./...`, `go build ./...`, `go test ./...` (with the
  network disabled once, SC-006; includes the T015a architecture check); also build the Windows
  executable (FR-017): `go build -ldflags "-X
  github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=1.1.0-rc.1" -o sonora-mcp.exe
  ./cmd/sonora-mcp` and check `sonora-mcp.exe --help` exits 0. Then run `go vet ./...` and `go test -race ./...` on Linux (WSL or a
  `golang:1.27` container with the workspace mounted) to meet FR-019 / Principle V ("pass on
  Windows and Linux"); `-race` runs only there because it needs cgo and a C toolchain, which the
  Windows setup does not have. Record both results in the PR description
- [X] T069 After sonora-cli PR #19 is merged and tagged: `go get
  github.com/Sonora-Multiroom/sonora-cli@<tag>`, `go mod tidy`, re-run the T068 gate (Windows and Linux) with `GOWORK=off`;
  confirm `go.mod` has no pseudo-version or `replace` for sonora-cli (constitution: dependency pinning)
- [ ] T070 Run the full quickstart.md validation (§3–§7) against the real hub and MCP Inspector,
  including VS Code with the existing `.vscode/mcp.json`; record results in the PR description
- [ ] T072 Cut the `v1.1.0` release from `main` (FR-017c; constitution: dependency pinning). Before
  merge, as the last commit of the PR: set `RELEASE_TAG="v1.1.0"` in `deploy/pi/install.sh`. After
  the PR is merged: tag `main` `v1.1.0` with `release.sh` (the release workflow builds and
  publishes it); confirm the `sonora-mcp_1.1.0_linux_arm64.tar.gz` download URL returns `200`; re-run `install.sh` on the Pi **without** `--version` and check `/health` reports
  `1.1.0`; delete the `v1.1.0-rc.1` prerelease and tag

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (1)** → **Foundational (2)** → user stories.
- **US1 (3)** first (MVP). **US2 (4)** and **US3 (5)** depend only on Foundational but touch the same
  files as US1 (`inputs.go`, `outputs.go`, `groups.go`, `routes.go`, `playback.go`), so run them after
  US1 to avoid edit conflicts.
- **US4 (6)** needs at least the tools it exercises (US1–US3).
- **US5 (7)** depends on Foundational, plus US1 for T052 (its in-flight call uses `getMasterMute`);
  T055/T051 use `hub.GetMasterMute` directly, not the tool.
- **US6 (8)**: T062 after US3; T064, T066, T067 last, after the Go server is verified (US5
  checkpoint).
- **Polish (9)**: T069 is blocked on the sonora-cli tag. T072 is the last task: after T069–T070 and
  the merge to `main`. T060a publishes only a prerelease; no final release is tagged off `main`.

### Within each phase

- Test tasks before their implementation tasks; see them fail first.
- T007 before T012 (the helper uses schema helpers); T009 before any tool task; T019a before T020.

### Parallel opportunities

- Setup: T002, T003, T003a, T003b.
- Foundational: T004, T005, T006, T008, T010, T011, T015a, T016, T018, T019a (all separate files).
- US1: T021–T025 together; then T026–T030 together (separate files).
- US2: T032–T035; then T036–T039.
- US3: T040–T042; then T043–T045.
- US4: T046–T048a.
- US5: T050–T053; T058 then T059 (same file, `deploy/pi/install.sh`), anytime in the phase.
- US5 can run alongside US2–US4 once US1 is done (different packages).
- T065 and T071 were removed (the untracked `run*.sh` launchers are deleted in T064; the `docs/future/`
  design note is committed by T003a instead); the IDs are not reused.

---

## Parallel Example: User Story 1

```text
Task: "T021 Write read-tool tests in internal/tools/inputs_read_test.go"
Task: "T022 Write internal/tools/outputs_read_test.go"
Task: "T023 Write internal/tools/groups_read_test.go"
Task: "T024 Write internal/tools/routes_read_test.go"
Task: "T025 Write internal/tools/mastermute_read_test.go"

# then
Task: "T026 Implement listInputs, getInput in internal/tools/inputs.go"
Task: "T027 Implement listOutputs, getOutput in internal/tools/outputs.go"
Task: "T028 Implement listGroups, getGroup in internal/tools/groups.go"
Task: "T029 Implement listRoutes, getRoute in internal/tools/routes.go"
Task: "T030 Implement getMasterMute in internal/tools/playback.go"
```

---

## Implementation Strategy

### MVP first (US1)

1. Phase 1 + Phase 2 → binary starts, MCP Inspector connects.
2. Phase 3 → 10 read-only tools. **Stop and validate** with MCP Inspector against the real hub.

### Incremental delivery

1. US2 → control tools (everyday use).
2. US3 → routing and playback: full tool parity; the Go server can replace the Node one locally.
3. US4 → error behaviour hardened.
4. US5 → Pi deployment; the Go server replaces the Node one in production.
5. US6 → Node removed, README rewritten.
6. Polish → pin sonora-cli tag, gate, merge.

---

## Notes

- Commit after each task or logical group, Conventional Commits.
- Keep the Node server runnable until T064 as the behaviour reference (constitution: legacy freeze —
  no changes to it).
- If a task reveals missing behaviour in `hub`, stop: fix it in sonora-cli first (Principle I).
