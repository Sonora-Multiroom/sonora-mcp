# Research: Rewrite sonora-mcp on the Shared Hub Client

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-09-28

The MCP Go SDK facts below were checked against the SDK source (v1.8.0 in the local module cache) rather
than recalled. The hub facts were checked against sonora-cli branch `010-public-hub-package` (PR #19).

## R1. MCP SDK and version

- **Decision**: `github.com/modelcontextprotocol/go-sdk` **v1.8.0** (latest stable; module requires
  go 1.25, we build with 1.27).
- **Rationale**: official SDK named by the constitution; v1.x API is stable. It provides
  `mcp.NewServer`, `Server.AddTool`, `mcp.NewStreamableHTTPHandler` with a `Stateless` option, tool
  annotations, structured content, and in-memory / streamable client transports for tests.
- **Alternatives**: `mark3labs/mcp-go` (community, not sanctioned by Principle VI); v1.8.0-pre.x (no reason
  to run a pre-release).
- **Cost noted (Principle VI)**: the SDK pulls transitive modules (`google/jsonschema-go`,
  `golang-jwt/jwt`, `golang.org/x/oauth2`, `x/tools`, `segmentio/encoding`, …). Only packages we import
  are linked; the binary size is recorded in the PR that introduces the dependency.

## R2. Tool registration: typed `mcp.AddTool`

- **Decision** (revised 2026-09-28; the maintainer chose the simpler option): register every tool with
  the SDK's typed `mcp.AddTool[In, Out]`. The input schema is passed explicitly (inferred + constraints,
  R3); the output schema is inferred from `Out`. Handlers return `(nil, out, nil)` on success or
  `(nil, zero, err)` on failure, where `err`'s text is `"<Category>: <message>"` (R5).
- **What the SDK does for us** (from `mcp/server.go`, `toolForErr`): validates arguments against the
  input schema before the handler runs (an invalid call never reaches the hub); fills
  `structuredContent` and a JSON text copy from `Out`; turns a returned error into `isError: true` with
  the error text.
- **What we add**:
  - A generic `recoverPanics` wrapper around each handler: the SDK has no `recover()` in `mcp/`, and
    FR-011 requires one call's fault not to stop the server. A panic becomes the error
    `Internal: unexpected server error` and is logged with its stack.
  - Logging through `Server.AddReceivingMiddleware`: one middleware for `tools/call` logs tool name,
    arguments, outcome and duration for **every** call, including calls the SDK rejects during input
    validation (which never reach our handler).
- **Known limit, accepted**: the SDK discards any result a handler builds when it returns an error, so
  the category cannot also go into a structured field (`_meta`); it is only the text prefix. Input
  validation errors carry the SDK's own text (`validating "arguments": ...`), without our prefix.
- **Upgrade path if needed**: switch to the low-level `Server.AddTool` behind the same per-tool
  functions (own validation, result building with `_meta["sonora/errorCategory"]`). Tool definitions
  and tests stay; only the registration helper changes.
- **Alternatives**: low-level `Server.AddTool` with our own helper now (more code for a structured
  category no client uses yet).

## R3. Input schemas and constraints

- **Decision**: one Go input struct per tool (`json` tags for names, `jsonschema` tag for the field
  description). Constraints the tag cannot express are applied in code after inference: `minimum` /
  `maximum` (volume 0–100), `enum` (target type, route status), `minLength` / `pattern` (path and
  body IDs, `uri`, `createInput.displayName`, `inputId` pattern `^[a-zA-Z0-9\-_]{1,255}$`), exactly
  where the spec has them plus the path-parameter rule in R4 (no `minLength` on the `listRoutes`
  filters or `playback.displayName`). Optional fields are pointer types with `omitempty`.
- **Rationale**: in `google/jsonschema-go` v0.4.3 the `jsonschema` struct tag is **only** the
  description (tags starting `WORD=` are rejected), so constraints must be set on the inferred
  `*jsonschema.Schema`. Explicit per-tool code keeps each tool readable.
- **Alternatives**: generate tool input schemas from `api.Spec` at startup (zero drift, but couples tool
  surface to OpenAPI parameter layout and makes descriptions hard to tailor for agents; the conformance
  test in R4 gives the same safety).

## R4. Spec conformance test (Principle I, FR-018)

- **Decision**: `internal/tools/conformance_test.go` parses `api.Spec` (from the pinned sonora-cli
  version). Each tool declares the hub operation it wraps (`method`, `path`). The test checks, for every
  tool: the operation exists in the spec; every tool input field maps to a path/query parameter or a
  request-body property of that operation; and `required`, `enum`, `minimum`, `maximum`, `minLength`,
  `pattern` match. Path parameters are compared as if the spec declared `minLength: 1`: the hub spec
  (0.1.18) gives path IDs only `{"type":"string"}`, yet an empty segment would change the route
  (`GET /api/v2/outputs/` is the list endpoint), so a non-empty path ID is what the spec's path
  means, not a guessed constraint. Query and body parameters get no such allowance. The test also
  checks responses (FR-005): every property of the operation's success-response schema is a
  property of the tool's `outputSchema` (list envelopes unwrapped to their item schema; 204
  operations skipped). Mismatches fail with tool name, field and both values. A second check
  asserts the registered tool names equal the 24-name inventory.
- **Rationale**: this is the automated drift detector the constitution requires, using the same spec
  version the client was tested against.
- **Alternatives**: keep a local `openapi.json` (forbidden by Principle I).

## R5. Result and error shape (FR-005, FR-005a, FR-009, FR-010)

- **Decision**:
  - **Success**: `structuredContent` = the hub data; `content` = one text block with the same JSON.
    `outputSchema` inferred from the result type. MCP clients before SEP-2106 require
    `structuredContent` to be an object, so **list results are wrapped** (`{"outputs": [...]}`,
    `{"routes": [...]}`, …) and no-body operations return `{"deleted": true, "<id>": "..."}` (deletes).
  - **Error**: `isError: true`; text `"<Category>: <message>"`. No `structuredContent` and no
    `_meta` category (R2 limit).
  - **Categories** (from `hub.ClassifyError`, with two pre-checks):
    `*hub.DecodeError` → `MalformedResponse`; `context.DeadlineExceeded` or a `net.Error` whose
    `Timeout()` is true (the `http.Client` timeout) → `Timeout`; then
    `ClassNotFound`/`ClassInputNotFound`/`ClassTargetNotFound` → `NotFound`, `ClassValidation` →
    `Validation`, `ClassRouteFailed` → `RouteFailed`, `ClassSourceUnreachable` → `SourceUnreachable`,
    `ClassServiceUnavailable` → `ServiceUnavailable`, `ClassNetwork` → `Network`, `ClassHub` →
    `HubError`. MCP-side: `Internal` (panic). Input that fails the schema is rejected by the SDK with
    its own `validating "arguments": ...` text; the log middleware records it as `InvalidInput`.
  - **Hub explanation**: `ClassifyError`'s message already uses the hub's problem `detail`/`title`
    for 400/422; for other statuses the message includes the typed error's text so the hub's
    explanation is not lost (FR-010).
- **Rationale**: the text prefix is what the model reads, so it is enough for agents. Wrapping lists is required for object-only clients and the assumption in the spec
  allows formatting changes.
- **Alternatives**: bare arrays in `structuredContent` (breaks older clients); error details in
  `structuredContent` (conflicts with `outputSchema`); category in `_meta` (needs the low-level API,
  R2).

## R6. Transport, statelessness, cancellation (FR-007, FR-008, FR-014)

- **Decision**: `mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, &mcp.StreamableHTTPOptions{Stateless: true, PropagateRequestCancellation: true})`
  mounted at `/mcp`. One `*mcp.Server` shared by all requests.
- **Findings**: in stateless mode GET/DELETE on `/mcp` return 405 (clients POST only; matches the Node
  server's stateless setup). `PropagateRequestCancellation` ties the handler context to the HTTP
  request only for protocol ≥ 2026-07-28; for older protocol versions a disconnect is not propagated,
  and the per-call hub timeout (5 s) is the bound. The cancellation test uses the new protocol path.
- **Localhost protection**: enabled by default in the SDK (requests via 127.0.0.1 with a non-localhost
  `Host` are rejected with 403). Keep it; LAN clients connecting via the Pi's LAN address are
  unaffected.
- **CORS** (the spec's outstanding item): the Node server allowed any origin. Browser-based clients are
  not a target (VS Code, Claude and MCP Inspector's proxy are not subject to CORS). **Decision**: no
  CORS headers. Revisit only with a concrete browser client.

## R7. Hub client usage and a prerequisite fix in sonora-cli

- **Decision**: one `*http.Client` from `hub.NewClient()` (5 s timeout, shared transport) for all tools.
  All 24 operations exist in `hub` (`ListInputs` … `SetMasterMute`, `ListRoutes(status, inputID,
  targetID)`, `SetPauseState`, `Playback`).
- **Gap found**: `hub.CreateInputRequest.Enabled` and `.AutoRemove` are plain `bool` without
  `omitempty`, while the spec types them `["boolean","null"]` with defaults true/false. An agent that
  omits `enabled` would create a **disabled** input (Node sent nothing). **Decision**: change both to
  `*bool` with `omitempty` in sonora-cli **before** PR #19 is tagged (the module is still untagged, so
  this breaks nobody), with a sonora-cli contract test. This is a prerequisite task, per Principle I
  ("missing operations/behaviour go to `hub` first"). **Done** on 2026-09-28 in sonora-cli commit
  `3ee7369` on PR #19 (`*bool` + `omitempty`, contract test, CLI call site updated; tests pass).
- **Alternatives**: sonora-mcp fills in the spec defaults (true/false) itself: works, but duplicates
  hub defaults in the proxy (Principle III).

## R8. `/health` (FR-015)

- **Decision**: `GET /health` → 200 always while the server runs, body
  `{"status":"ok","server":"sonora-mcp","version":"<v>","hub":"reachable|unreachable"}`. Hub check:
  `hub.GetMasterMute` (single cheap read) under a 2 s context deadline, so health answers within ~2 s
  even when the hub hangs (SC-005).
- **Alternatives**: `ListOutputs` (larger response); no hub check (rejected in clarification).

## R9. Flags and startup (FR-013, FR-013a)

- **Decision**: stdlib `flag` with `--multiroom-url` (required), `--port` (default 3001, 1–65535),
  `--host` (default empty = all addresses; accepts an IP literal or `localhost`), `-h/--help`. Go's
  `flag` accepts both `-x` and `--x`. Invalid or missing values → usage to stderr, exit 2; `--help` →
  usage, exit 0. Trailing slashes in the hub URL are harmless (`hub` trims them); the URL must parse as
  `http`/`https` with a host.
- **Alternatives**: cobra/pflag (unneeded dependency, Principle VI).

## R10. Logging (FR-012)

- **Decision**: `log/slog` text handler to stderr (journald captures it on the Pi). One line per tool
  call: `tool`, `args` (JSON), `outcome` (`ok` or category), `duration`. Startup line with version, hub
  URL and listen address. SDK internal logger left nil.
- **Alternatives**: JSON logs (harder to read in a terminal; no log pipeline exists).

## R11. Version injection (FR-016)

- **Decision**: `internal/version.Version` (default `dev`) set with
  `-ldflags "-X github.com/tiger-seo/sonora-mcp/internal/version.Version=<v>"`; used by
  `mcp.Implementation{Name: "sonora-mcp", Version}`, `/health` and the startup log. A test asserts all
  three read the same variable. First Go release: **1.1.0** (no tool renamed or input changed, so not
  MAJOR per Principle II; new flag and health field are additive).

## R12. Graceful shutdown (FR-017a)

- **Decision**: `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`; on signal,
  `http.Server.Shutdown` with a 6 s deadline (hub timeout + 1 s), then exit 0. Works on Windows
  (Ctrl+C) and systemd (SIGTERM).

## R13. Pi service and install script (FR-017b, FR-017c)

- **Decision**: a single `install.sh` is the only file copied to the Pi. The systemd unit is a
  heredoc inside the script (not a separate tracked file), written at install time to
  `/etc/systemd/system/sonora-mcp.service`: `DynamicUser=yes`,
  `EnvironmentFile=/etc/default/sonora-mcp`, `ExecStart=/usr/local/bin/sonora-mcp --multiroom-url
  ${SONORA_HUB_URL} --port ${SONORA_PORT} $SONORA_HOST_ARG`, `Restart=on-failure`, `RestartSec=2`,
  `After=network-online.target`, basic hardening (`NoNewPrivileges`, `ProtectSystem=strict`,
  `ProtectHome=yes`). The binary is not embedded: `deploy/pi/install.sh` has a `RELEASE_TAG`
  variable baked in at release-cut time on `main` (T072; pre-merge validation uses a prerelease via
  `--version`, T060a), matching the GitHub Release the cross-built
  `linux/arm64` binary (`sonora-mcp-linux-arm64`) was uploaded to. At install time the script
  downloads that asset with `curl -fsSL` from
  `https://github.com/tiger-seo/sonora-mcp/releases/download/${RELEASE_TAG}/sonora-mcp-linux-arm64`
  to `/usr/local/bin/sonora-mcp` (0755; fails with a clear message on a non-2xx response); it
  takes flags `--hub-url` (required), `--port`, `--host`, `--version` (overrides the baked-in
  `RELEASE_TAG` to install a different release); writes the env file and the unit; runs
  `daemon-reload`; `enable --now` or `restart` if already installed; idempotent; prints status.
  The repo is public (`tiger-seo/sonora-mcp` on GitHub), so the download needs no credentials.
- **Rationale**: `DynamicUser` avoids creating a system user; env file makes reconfiguration a
  re-run; downloading from a pinned release keeps `install.sh` small and readable (no binary blob
  in git history or in the copied file) while still needing only one file copied to the Pi
  (SC-005, SC-008); pinning to a baked-in tag (rather than "latest") keeps the version installed
  in step with the docs and script the operator is looking at (FR-016), with `--version` as an
  explicit escape hatch.
- **Alternatives**: base64-embed the binary in `install.sh` (works fully offline, but bloats the
  script and git history with a binary blob per release; rejected once GitHub Releases were
  available as the distribution point); always fetch "latest" (rejected: version drift, surprise
  upgrades on re-run).
- **New constraint**: the Pi needs outbound internet access to GitHub at install time (not
  needed once the server is running); this is an added Assumption in spec.md.
- **Testing**: validated on the Pi per [quickstart.md](quickstart.md); `shellcheck` on
  `install.sh` if available.

## R14. Test approach (Principle V)

- **Decision**:
  - Tool tests drive tools through a real MCP client over `mcp.NewInMemoryTransports()` against a fake hub
    (`httptest.Server`) that records requests and serves canned spec-shaped responses.
  - One end-to-end test runs the full HTTP handler (`/mcp`, `/health`) with
    `mcp.StreamableClientTransport` over `httptest`.
  - Flag parsing, category mapping and version are unit-tested.
  - Everything runs offline on Windows and Linux.
- **Rationale**: exercises schemas, validation and result shape exactly as an agent sees them, without
  duplicating hub contract tests (those stay in sonora-cli).

## R15. Local development and pinning

- **Decision**: `D:\projects-sonora\go.work` (uncommitted) with `./sonora-cli` and `./sonora-mcp`
  during development. `go.mod` gets a placeholder requirement until sonora-cli is tagged; the last task
  runs `go get github.com/Sonora-Multiroom/sonora-cli@<tag>` and the gate with `GOWORK=off`.
