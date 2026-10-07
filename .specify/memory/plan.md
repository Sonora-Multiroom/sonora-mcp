# Main Implementation Plan

> **Revision**: 2026-09-28 — Bootstrapped and populated by archiving `specs/001-go-rewrite`: the Go
> server on sonora-cli's `hub` package, its interfaces, Pi deployment, release pipeline, test
> approach and design decisions.

## Summary

sonora-mcp is a Go MCP server that exposes 24 tools and delegates every hub call to sonora-cli's
public `hub` package. It adds an error category on every failure, returns results as structured
content plus JSON text, has a `--host` flag and a hub reachability field in `/health`, shuts down
cleanly, and ships a systemd unit inside an install script for the Pi. Tools use the MCP Go SDK's
typed `mcp.AddTool` (the SDK validates input and builds structured results), with a panic-recovery
wrapper and a logging middleware. The error category is a text prefix only. A conformance test
checks every tool against the spec embedded in sonora-cli (`api.Spec`).
[Source: specs/001-go-rewrite/plan.md -> "Summary"]

## Technical Context

**Language/Version**: Go 1.27 (same as sonora-cli `go.mod`)
[Source: specs/001-go-rewrite/plan.md -> "Language/Version"]

**Primary Dependencies**: `github.com/modelcontextprotocol/go-sdk` v1.8.0 (MCP server, Streamable
HTTP, JSON Schema via `github.com/google/jsonschema-go`); `github.com/Sonora-Multiroom/sonora-cli`
v0.1.1 (`hub`, `api` packages); standard library otherwise (`net/http`, `flag`, `log/slog`,
`os/signal`) [Source: specs/001-go-rewrite/plan.md -> "Primary Dependencies"]
[Source: specs/001-go-rewrite/tasks.md -> T073]

**Storage**: N/A (stateless proxy) [Source: specs/001-go-rewrite/plan.md -> "Storage"]

**Testing**: `go test` with `net/http/httptest` fake hub; MCP client over
`mcp.NewInMemoryTransports` for tool tests; `mcp.StreamableClientTransport` for one end-to-end HTTP
test [Source: specs/001-go-rewrite/plan.md -> "Testing"]

**Target Platform**: Windows amd64 (development), Linux arm64 on Raspberry Pi with systemd
(production); static binary, `CGO_ENABLED=0` [Source: specs/001-go-rewrite/plan.md -> "Target Platform"]

**Project Type**: single-binary network service (MCP server over HTTP)
[Source: specs/001-go-rewrite/plan.md -> "Project Type"]

**Performance Goals**: hub latency dominates; server overhead per call negligible. `/health`
answers within 2 s even with the hub down; every tool returns within the 5 s hub timeout (SC-003
bound 10 s) [Source: specs/001-go-rewrite/plan.md -> "Performance Goals"]

**Constraints**: stateless (no sessions, no cached hub state); every hub call through `hub`, with
the request context and a timeout; LAN-only, unauthenticated; tool names and inputs unchanged
[Source: specs/001-go-rewrite/plan.md -> "Constraints"]

**Scale/Scope**: 24 tools, one household, a handful of concurrent agents; ~1–1.5k lines of Go plus
tests, one systemd unit, one install script [Source: specs/001-go-rewrite/plan.md -> "Scale/Scope"]

## Project Structure

### Source Code (repository root)

[Source: specs/001-go-rewrite/plan.md -> "Source Code (repository root)"]

```text
go.mod                         # module github.com/Sonora-Multiroom/sonora-mcp; go 1.27
go.sum
cmd/sonora-mcp/
├── main.go                    # parse config, build server, run until signal, exit codes
└── main_test.go               # run(ctx, args, stdout, stderr) exit codes; single version source
internal/
├── config/
│   ├── config.go              # flag parsing + validation (--multiroom-url, --port, --host), usage
│   └── config_test.go
├── version/
│   └── version.go             # Version = "dev", set by -ldflags
├── tools/
│   ├── register.go            # add[In, Out]: explicit input schema + mcp.AddTool + panic recovery
│   ├── logging.go             # receiving middleware: one log line per tools/call
│   ├── errors.go              # hub error → category, error result builder
│   ├── schema.go              # constraint helpers (range, enum, minLength, pattern)
│   ├── inputs.go              # listInputs, getInput, createInput, deleteInput, setInputEnabled
│   ├── outputs.go             # listOutputs, getOutput, setOutputVolume, setOutputMute, setOutputEnabled
│   ├── groups.go              # listGroups, getGroup, setGroupVolume, setGroupMute, setGroupEnabled
│   ├── routes.go              # listRoutes, getRoute, createRoute, deleteRoute, transferRoute, setRoutePause
│   ├── playback.go            # playback, getMasterMute, setMasterMute
│   ├── tools.go               # Register(server, hubClient, hubURL): all 24 (logging is middleware)
│   ├── fakehub_test.go        # httptest fake hub recording requests, canned responses
│   ├── harness_test.go        # in-memory MCP client session + callTool/toolByName helpers
│   ├── *_test.go              # per-file tool tests through an in-memory MCP client
│   ├── conformance_test.go    # tools vs api.Spec (inputs and responses); tool-name inventory
│   └── architecture_test.go   # no direct net/http hub calls, no package-level hub state
└── server/
    ├── server.go              # Handler(s, client, hubURL): /mcp (stateless), /health, 404; Run(ctx, srv) graceful shutdown
    ├── health.go              # /health with 2 s hub check
    ├── server_test.go         # end-to-end over httptest with StreamableClientTransport
    ├── health_test.go         # /health reachable / unreachable within 2 s
    └── shutdown_test.go       # in-flight call drains, Run returns nil
deploy/pi/
└── install.sh                 # idempotent installer (bash): embeds the systemd unit as a
                                # heredoc and a pinned release tag; downloads the matching
                                # linux/arm64 archive from GitHub Releases at install time and
                                # checks it against checksums.txt, so this is the only file
                                # copied to the Pi
.goreleaser.yaml               # release builds and archives, as in sonora-cli without Scoop
.github/workflows/
├── release.yml                # on v* tags: GoReleaser publishes the GitHub Release
└── test.yml                   # on PRs: gofmt, go vet, go test -race
release.sh                     # tags main and pushes the tag (copied from sonora-cli)
README.md                      # build, cross-build, run, Pi install, client config
.gitignore                     # Go/binary ignores, go.work*, Spec Kit rules
```

**Structure Decision**: single Go module with one binary under `cmd/sonora-mcp` and private
packages under `internal/`, the same layout sonora-cli uses. `internal/tools` holds all
MCP-specific mapping so it can be tested in isolation from HTTP; `internal/server` owns HTTP
concerns only. [Source: specs/001-go-rewrite/plan.md -> "Structure Decision"]

## Interfaces & Routing

### Command line

[Source: specs/001-go-rewrite/contracts/server.md -> "Command line"]

```
sonora-mcp --multiroom-url <url> [--port <port>] [--host <address>]
sonora-mcp -h | --help
```

| Flag | Required | Default | Rules |
|---|---|---|---|
| `--multiroom-url` | yes | — | `http`/`https` URL with a host; trailing `/` allowed |
| `--port` | no | `3001` | Integer 1–65535 |
| `--host` | no | all addresses | IP literal (e.g. `127.0.0.1`, `::1`) or `localhost` |
| `-h`, `--help` | no | — | Print usage |

| Situation | Output | Exit status |
|---|---|---|
| `--help` | Usage on stdout | 0 |
| Missing or invalid flag value | Error + usage on stderr | 2 |
| Cannot listen (port in use, address not local) | Error on stderr | 1 |
| Stop signal (Ctrl+C, SIGTERM) after draining | Shutdown log line | 0 |

A listen failure exits 1 without usage text: the flags parsed and were valid, the environment
refused them (port taken, address not on this machine), so repeating the usage would not help. It
is a startup failure, not a flag error, and still stops the server before it serves (Principle IV).

Startup log (stderr): version, hub URL, listen address, number of tools registered.

### HTTP endpoints

[Source: specs/001-go-rewrite/contracts/server.md -> "HTTP endpoints"]

- **`POST /mcp`**: MCP Streamable HTTP, stateless (no `Mcp-Session-Id`). `GET` and `DELETE` on
  `/mcp` return 405. Server info: `name: "sonora-mcp"`, `version: <build version>`. Capabilities:
  tools only. Requests that arrive through a loopback address with a non-localhost `Host` header
  are rejected with 403 (DNS-rebinding protection). No CORS headers are sent.
- **`GET /health`**: always `200 OK` while the server runs; `Content-Type: application/json`;
  body `{ "status": "ok", "server": "sonora-mcp", "version": "1.1.0", "hub": "reachable" }`.
  `hub` is `"reachable"` or `"unreachable"`, from one read-only hub request bounded by 2 seconds.
- **Anything else**: `404 Not Found`.

### MCP tools

[Source: specs/001-go-rewrite/contracts/tools.md -> "Inputs"]
[Source: specs/001-go-rewrite/contracts/tools.md -> "Outputs"]
[Source: specs/001-go-rewrite/contracts/tools.md -> "Groups"]
[Source: specs/001-go-rewrite/contracts/tools.md -> "Routes"]
[Source: specs/001-go-rewrite/contracts/tools.md -> "Playback and master mute"]

Names and inputs are the compatibility contract with the old Node.js server (FR-001) and must not
change without a MAJOR version (Principle II). `?` = optional. `TargetType` = `SINGLE_OUTPUT` |
`OUTPUT_GROUP`. `Volume` = integer 0–100. `RouteStatus` = `STARTING` | `ACTIVE` | `STOPPING` |
`STOPPED` | `FAILED`.

| Tool | Hub operation | Inputs | Kind | Structured result |
|---|---|---|---|---|
| `listInputs` | `GET /api/v2/inputs` | `includeDisabled?: bool` | read-only | `{"inputs": [Input]}` |
| `getInput` | `GET /api/v2/inputs/{inputId}` | `inputId` | read-only | `Input` |
| `createInput` | `POST /api/v2/inputs` | `inputId` (pattern `^[a-zA-Z0-9\-_]{1,255}$`), `displayName`, `uri`, `enabled?: bool`, `autoRemove?: bool` | state-changing | `Input` |
| `deleteInput` | `DELETE /api/v2/inputs/{inputId}` | `inputId` | destructive | `{"deleted": true, "inputId"}` |
| `setInputEnabled` | `PUT /api/v2/inputs/{inputId}/enabled` | `inputId`, `enabled: bool` | idempotent | `Input` |
| `listOutputs` | `GET /api/v2/outputs` | `includeDisabled?: bool` | read-only | `{"outputs": [Output]}` |
| `getOutput` | `GET /api/v2/outputs/{outputId}` | `outputId` | read-only | `Output` |
| `setOutputVolume` | `PUT /api/v2/outputs/{outputId}/volume` | `outputId`, `volume: Volume` | idempotent | `OutputVolume` |
| `setOutputMute` | `PUT /api/v2/outputs/{outputId}/mute` | `outputId`, `muted: bool` | idempotent | `OutputMute` |
| `setOutputEnabled` | `PUT /api/v2/outputs/{outputId}/enabled` | `outputId`, `enabled: bool` | idempotent | `Output` |
| `listGroups` | `GET /api/v2/groups` | `includeDisabled?: bool` | read-only | `{"groups": [Group]}` |
| `getGroup` | `GET /api/v2/groups/{groupId}` | `groupId` | read-only | `Group` |
| `setGroupVolume` | `PUT /api/v2/groups/{groupId}/volume` | `groupId`, `volume: Volume` | idempotent | `GroupVolume` |
| `setGroupMute` | `PUT /api/v2/groups/{groupId}/mute` | `groupId`, `muted: bool` | idempotent | `GroupMute` |
| `setGroupEnabled` | `PUT /api/v2/groups/{groupId}/enabled` | `groupId`, `enabled: bool` | idempotent | `Group` |
| `listRoutes` | `GET /api/v2/routes` | `status?: RouteStatus`, `inputId?`, `targetId?` | read-only | `{"routes": [Route]}` |
| `getRoute` | `GET /api/v2/routes/{routeId}` | `routeId` | read-only | `Route` |
| `createRoute` | `POST /api/v2/routes` | `inputId`, `targetId`, `targetType: TargetType` | state-changing | `Route` |
| `deleteRoute` | `DELETE /api/v2/routes/{routeId}` | `routeId` | destructive | `{"deleted": true, "routeId"}` |
| `transferRoute` | `POST /api/v2/routes/{routeId}/transfer` | `routeId`, `targetId`, `targetType: TargetType` | state-changing | `Route` |
| `setRoutePause` | `PUT /api/v2/routes/{routeId}/pause` | `routeId`, `paused: bool` | idempotent | `Route` |
| `playback` | `POST /api/v2/play` | `uri`, `targetId`, `targetType: TargetType`, `displayName?`, `volume?: Volume` | state-changing | `PlaybackResponse` (`inputId`, `route`, `message`) |
| `getMasterMute` | `GET /api/v2/master-mute` | — | read-only | `MasterMute` |
| `setMasterMute` | `PUT /api/v2/master-mute` | `muted: bool` | idempotent | `MasterMute` |

Resource shapes (`Input`, `Output`, `OutputVolume`, `OutputMute`, `Group`, `GroupVolume`,
`GroupMute`, `Route`, `PlaybackResponse`, `MasterMute`) are the `hub` package types, which carry
every field of the corresponding spec response schema. Hub operation paths were checked against
`api.Spec` (hub API 0.1.18) on 2026-09-28; the conformance test, not this table, is authoritative
when the spec changes.

### Error categories

[Source: specs/001-go-rewrite/contracts/tools.md -> "Error categories"]

Every failed call returns `isError: true` with text `"<Category>: <message>"`. Exception: input that
fails the tool's schema is rejected by the MCP SDK with `validating "arguments": <details>` (no hub
call made).

| Category | When |
|---|---|
| `NotFound` | The input/output/group/route (or a target) doesn't exist |
| `Validation` | The hub rejected the request as invalid (includes the hub's explanation) |
| `Conflict` | The hub refused the request in its current state (409); the hub's `reason`, if any, follows in parentheses |
| `RouteFailed` | The hub could not create the route (422) |
| `SourceUnreachable` | The hub could not reach the audio source (502) |
| `ServiceUnavailable` | The hub's service is temporarily unavailable (503) |
| `Timeout` | The hub did not respond within the time limit |
| `Network` | The hub could not be reached |
| `MalformedResponse` | The hub's response could not be read as the expected data |
| `HubError` | Any other hub error status |
| `Internal` | Unexpected fault in the server while handling the call |

## Configuration

- Server flags: `--multiroom-url` (required), `--port` (default 3001), `--host` (default: all
  addresses); see Interfaces & Routing. [Source: specs/001-go-rewrite/contracts/server.md -> "Command line"]
- Version: `internal/version.Version` (default `dev`), set at build time with
  `-ldflags "-X github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=<v>"`.
  [Source: specs/001-go-rewrite/research.md -> R11]
- Pi service config: `/etc/default/sonora-mcp` with `SONORA_HUB_URL`, `SONORA_PORT`,
  `SONORA_HOST_ARG`, written by `install.sh`. [Source: specs/001-go-rewrite/contracts/server.md -> "Config"]
- Local development: a `go.work` in the parent folder (`./sonora-cli`, `./sonora-mcp`), never
  committed; the gate and release builds run with `GOWORK=off`.
  [Source: specs/001-go-rewrite/research.md -> R15]

## Build & Release

[Source: specs/001-go-rewrite/contracts/server.md -> "Build"]

```
go build -ldflags "-X github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=<v>" ./cmd/sonora-mcp
```

Without `-ldflags` the version is `dev`. Releases are built by GoReleaser (`.goreleaser.yaml`, as in
sonora-cli, without Scoop). Pushing a `v*` tag runs `.github/workflows/release.yml`, which builds
linux, darwin and windows for amd64 and arm64 (`CGO_ENABLED=0`, `-s -w`, version from the tag) and
publishes a GitHub Release with `sonora-mcp_<version>_<os>_<arch>.tar.gz` (`.zip` on windows) and
`checksums.txt`. Tags with a pre-release suffix (`v1.1.0-rc.1`) are published as prereleases.
`release.sh` tags `main` and pushes the tag. `.github/workflows/test.yml` runs `gofmt`, `go vet` and
`go test -race` on pull requests.

Releases are tagged on `main` only, after the merge gate passes with a tagged sonora-cli.
`deploy/pi/install.sh`'s `RELEASE_TAG` is bumped to `<tag>` in the commit that gets tagged, so the
script always matches the release it ships with. Pre-merge builds are published as prereleases
(`<tag>-rc.N`, tagged on the feature branch) and installed with `--version`; `RELEASE_TAG` never
points at one.

## Deployment (Raspberry Pi)

[Source: specs/001-go-rewrite/contracts/server.md -> "Pi service (`deploy/pi/`)"]

`deploy/pi/install.sh` is the only file the Pi needs. The systemd unit is embedded as a heredoc; the
built `linux/arm64` binary is not embedded — the script downloads it from the GitHub Release whose
tag is baked in as `RELEASE_TAG`. The repository is public, so no credentials are needed. Install in
one command, piping the script from `main`:

```
curl -fsSL https://raw.githubusercontent.com/Sonora-Multiroom/sonora-mcp/main/deploy/pi/install.sh | sudo bash -s -- --hub-url <url> [--port <port>] [--host <address>] [--version <tag>]
```

or run a copy on the Pi with `sudo ./install.sh --hub-url <url> [...]`. The script body runs from a
`main` function called on its last line, so a download cut off part way through executes nothing.

| Effect | Detail |
|---|---|
| Binary | Downloads `sonora-mcp_<version>_linux_arm64.tar.gz` and `checksums.txt` with `curl -fsSL` from `https://github.com/Sonora-Multiroom/sonora-mcp/releases/download/<tag>/` (`<tag>` = `--version` if given, else `RELEASE_TAG`; `<version>` = `<tag>` without the leading `v`), checks the SHA-256 against `checksums.txt`, extracts `sonora-mcp`, copies it to a temp file in `/usr/local/bin`, sets mode 0755, then renames it (`mv -f`) onto `/usr/local/bin/sonora-mcp`. The rename works while the old binary is running, and a failed download or checksum leaves it untouched |
| Config | `/etc/default/sonora-mcp` with `SONORA_HUB_URL`, `SONORA_PORT`, `SONORA_HOST_ARG` |
| Unit | Written from the embedded heredoc to `/etc/systemd/system/sonora-mcp.service` |
| Service | `daemon-reload`; enabled at boot; started, or restarted if already running |
| Re-run | Overwrites binary, config and unit; restarts; never duplicates |
| Exit | 0 on success with `systemctl status` summary; non-zero with a message if not root, `--hub-url` missing, systemd or `curl` absent, the OS is not 64-bit ARM (`uname -m` ≠ `aarch64`), the download fails (network error or non-2xx, e.g. unknown tag), or the checksum does not match |

The unit runs `sonora-mcp` with the configured flags as a dynamic unprivileged user
(`DynamicUser=yes`, `EnvironmentFile=/etc/default/sonora-mcp`), after `network-online.target`, with
`Restart=on-failure`, `RestartSec=2` and basic hardening (`NoNewPrivileges`, `ProtectSystem=strict`,
`ProtectHome=yes`). It is stopped with SIGTERM, which triggers graceful shutdown. Installation needs
outbound internet access to GitHub; the running server does not.
[Source: specs/001-go-rewrite/research.md -> R13]

## Testing Strategy

[Source: specs/001-go-rewrite/research.md -> R14]

- Tool tests drive tools through a real MCP client over `mcp.NewInMemoryTransports()` against a fake
  hub (`httptest.Server`) that records requests and serves canned spec-shaped responses.
- One end-to-end test runs the full HTTP handler (`/mcp`, `/health`) with
  `mcp.StreamableClientTransport` over `httptest`.
- Flag parsing, category mapping and version are unit-tested.
- Everything runs offline on Windows and Linux; hub contract tests stay in sonora-cli.
- `internal/tools/conformance_test.go` checks every tool against `api.Spec`: the operation exists;
  every input field maps to a path/query parameter or request-body property; `required`, `enum`,
  `minimum`, `maximum`, `minLength`, `pattern` match (path parameters compared as `minLength: 1`);
  every property of the success-response schema is in the tool's `outputSchema` (list envelopes
  unwrapped, 204 operations skipped); and the registered tool names equal the 24-name inventory.
  [Source: specs/001-go-rewrite/research.md -> R4]
- Cancellation is covered twice: over in-memory transports (tool handler → hub call), and over the
  real `/mcp` handler with a Streamable HTTP client on protocol ≥ 2026-07-28, which fails if
  `PropagateRequestCancellation` is dropped. [Source: specs/001-go-rewrite/research.md -> R6]
- `internal/tools/architecture_test.go`: no direct `net/http` hub calls, no package-level hub
  state. [Source: specs/001-go-rewrite/plan.md -> "Source Code (repository root)"]
- `install.sh` is validated on the Pi per the quickstart; `shellcheck` if available.
  [Source: specs/001-go-rewrite/research.md -> R13]

## Design Decisions

- **MCP SDK**: `github.com/modelcontextprotocol/go-sdk` v1.8.0 (official, named by the
  constitution; v1.x API stable). Rejected: `mark3labs/mcp-go` (not sanctioned by Principle VI),
  pre-releases. The SDK pulls transitive modules (`google/jsonschema-go`, `golang-jwt/jwt`,
  `golang.org/x/oauth2`, `x/tools`, `segmentio/encoding`, …); only imported packages are linked.
  [Source: specs/001-go-rewrite/research.md -> R1]
- **Tool registration**: typed `mcp.AddTool[In, Out]`, input schema passed explicitly (inferred +
  constraints), output schema inferred from `Out`. Handlers return `(nil, out, nil)` or
  `(nil, zero, err)` with `err` text `"<Category>: <message>"`. The SDK validates arguments before
  the handler, fills `structuredContent` and a JSON text copy, and turns a returned error into
  `isError: true`. We add a `recoverPanics` wrapper (panic → `Internal: unexpected server error`,
  logged with stack) and a `tools/call` receiving middleware that logs every call, including calls
  rejected during input validation. Upgrade path if a structured category is ever needed: the
  low-level `Server.AddTool` behind the same per-tool functions
  (`_meta["sonora/errorCategory"]`). [Source: specs/001-go-rewrite/research.md -> R2]
- **Input schemas**: one Go input struct per tool (`json` tags for names, `jsonschema` tag for the
  description only); `minimum`/`maximum`, `enum`, `minLength`/`pattern` applied in code on the
  inferred schema, exactly where the spec has them plus the path-parameter rule. Optional fields
  are pointer types with `omitempty`. Rejected: generating schemas from `api.Spec` at startup.
  [Source: specs/001-go-rewrite/research.md -> R3]
- **Result and error shape**: success = `structuredContent` (hub data, lists wrapped, deletes
  `{"deleted": true, "<id>": "..."}`) plus one text block with the same JSON; error = `isError:
  true`, text `"<Category>: <message>"`, no `structuredContent` or `_meta`. Category mapping:
  `*hub.DecodeError` → `MalformedResponse`; `context.DeadlineExceeded` or a timing-out `net.Error`
  → `Timeout`; then `ClassNotFound`/`ClassInputNotFound`/`ClassTargetNotFound` → `NotFound`,
  `ClassValidation` → `Validation`, `ClassConflict` → `Conflict`, `ClassRouteFailed` → `RouteFailed`, `ClassSourceUnreachable`
  → `SourceUnreachable`, `ClassServiceUnavailable` → `ServiceUnavailable`, `ClassNetwork` →
  `Network`, `ClassHub` → `HubError`; panics → `Internal`; schema failures logged as
  `InvalidInput`. `ClassifyError`'s message carries the hub's problem `detail`/`title` for
  400/409/422 (a 409's `reason` is appended as ` (<REASON>)`), and the typed error's text otherwise. [Source: specs/001-go-rewrite/research.md -> R5]
- **Transport**: `mcp.NewStreamableHTTPHandler` with `Stateless: true` and
  `PropagateRequestCancellation: true` at `/mcp`, one shared `*mcp.Server`. SDK localhost
  protection kept on. No CORS headers (browser clients are not a target).
  [Source: specs/001-go-rewrite/research.md -> R6]
- **Hub client**: one `*http.Client` from `hub.NewClient()` (5 s timeout, shared transport) for all
  tools. Optional hub request fields are fixed in sonora-cli, not filled in here (Principle III):
  `CreateInputRequest.Enabled`/`AutoRemove` became `*bool` + `omitempty` in sonora-cli `3ee7369`.
  [Source: specs/001-go-rewrite/research.md -> R7]
- **`/health`**: hub check is `hub.GetMasterMute` under a 2 s context deadline.
  [Source: specs/001-go-rewrite/research.md -> R8]
- **Flags**: stdlib `flag` (accepts `-x` and `--x`); trailing slashes in the hub URL are trimmed
  by `hub`; the URL must be `http`/`https` with a host. Rejected: cobra/pflag.
  [Source: specs/001-go-rewrite/research.md -> R9]
- **Logging**: `log/slog` text handler to stderr (journald on the Pi); one line per tool call
  (`tool`, `args`, `outcome`, `duration`); startup line with version, hub URL and listen address;
  SDK logger left nil. [Source: specs/001-go-rewrite/research.md -> R10]
- **Versioning**: first Go release 1.1.0 (no tool renamed or input changed, so not MAJOR; the new
  flag and health field are additive). A test asserts server info, `/health` and the startup log
  read the same variable. [Source: specs/001-go-rewrite/research.md -> R11]
- **Graceful shutdown**: `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`; on signal,
  `http.Server.Shutdown` with a 6 s deadline (hub timeout + 1 s), then exit 0.
  [Source: specs/001-go-rewrite/research.md -> R12]
- **Pi install**: download a pinned release instead of embedding the binary in `install.sh`
  (rejected: base64 blob per release) or fetching "latest" (rejected: version drift);
  `DynamicUser` avoids creating a system user; the env file makes reconfiguration a re-run.
  [Source: specs/001-go-rewrite/research.md -> R13]

## Complexity Tracking

No constitution violations to justify. [Source: specs/001-go-rewrite/plan.md -> "Complexity Tracking"]
