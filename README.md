# sonora-mcp

MCP server for the Sonora Multiroom audio system. It exposes the Multiroom Audio Hub API as 24 MCP
tools, so any AI assistant that supports MCP can list and control your speakers, groups, inputs
and routes over a URL.

## Architecture

```
AI Assistant (Copilot, Claude, etc.)
       │
       ▼  MCP (Streamable HTTP, stateless) at /mcp
┌──────────────────────────┐
│  sonora-mcp              │  port 3001
│  (Go, one static binary) │
└────────────┬─────────────┘
             │  HTTP, via the sonora-cli hub package
             ▼
┌──────────────────────────┐
│  Multiroom Audio Hub API │  e.g. port 8080
└──────────────────────────┘
```

sonora-mcp keeps no state of its own: every tool call is one request to the hub.

## Install on a Raspberry Pi

One command on the Pi (64-bit Raspberry Pi OS) installs sonora-mcp as a systemd service:

```bash
curl -fsSL https://raw.githubusercontent.com/Sonora-Multiroom/sonora-mcp/main/deploy/pi/install.sh | sudo bash -s -- --hub-url http://localhost:8080
```

Options go after `--hub-url`:

| Option | Default | Description |
|---|---|---|
| `--hub-url <url>` | — (required) | Base URL of the Multiroom Audio Hub |
| `--port <port>` | `3001` | Port sonora-mcp listens on |
| `--host <address>` | all addresses | Address to listen on, e.g. `127.0.0.1` |
| `--version <tag>` | the release the script is pinned to (its `RELEASE_TAG`) | Release to install, e.g. `v1.1.0` |

The script downloads the release archive, checks it against the release's `checksums.txt`,
installs `/usr/local/bin/sonora-mcp`, and writes `/etc/default/sonora-mcp` and the
`sonora-mcp.service` unit. The service starts at boot and restarts on failure. Re-run the same
command with different options to reconfigure or upgrade. You can also copy
[deploy/pi/install.sh](deploy/pi/install.sh) to the Pi and run `sudo ./install.sh --hub-url <url>`.

```bash
systemctl status sonora-mcp
curl localhost:3001/health
```

Other platforms: download the archive for your OS from
[Releases](https://github.com/Sonora-Multiroom/sonora-mcp/releases) and run the binary.

## Run

```bash
sonora-mcp --multiroom-url http://multiroom.lan:8080
```

| Flag | Required | Default | Description |
|---|---|---|---|
| `--multiroom-url <url>` | **yes** | — | Base URL of the Multiroom Audio Hub (`http` or `https`) |
| `--port <port>` | no | `3001` | Port to listen on (1–65535) |
| `--host <address>` | no | all addresses | Address to listen on: an IP such as `127.0.0.1` or `::1`, or `localhost` |
| `-h`, `--help` | no | — | Show usage |

The server does not start without `--multiroom-url`. A missing or invalid flag prints the error and
usage and exits with status 2; a port that can't be opened exits with status 1. Ctrl+C or SIGTERM
stops accepting connections, lets in-flight calls finish (up to 6 s) and exits 0. Each tool call
is logged to stderr.

### Health check

```bash
curl http://localhost:3001/health
# {"status":"ok","server":"sonora-mcp","version":"1.1.0","hub":"reachable"}
```

`/health` always answers `200` while the server runs. `hub` is `reachable` or `unreachable`,
from one read-only hub request bounded by 2 seconds.

## Connect an AI assistant

The MCP endpoint is `http://<host>:3001/mcp`.

### VS Code (GitHub Copilot)

`.vscode/mcp.json`:

```json
{
  "servers": {
    "sonora-mcp": {
      "type": "http",
      "url": "http://localhost:3001/mcp"
    }
  }
}
```

### Claude Desktop

`claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "sonora-mcp": {
      "type": "streamableHttp",
      "url": "http://localhost:3001/mcp"
    }
  }
}
```

### MCP Inspector (testing)

```bash
npx @modelcontextprotocol/inspector
```

Connect to `http://localhost:3001/mcp` with the Streamable HTTP transport.

## Tools (24)

Every tool returns structured content (and the same JSON as text). Lists are wrapped in an object,
such as `{"outputs": [...]}`.

### Inputs

| Tool | Description |
|---|---|
| `listInputs` | List audio inputs (`includeDisabled` to include disabled ones) |
| `getInput` | Get one input by ID |
| `createInput` | Register a new ephemeral input from a stream or file URI |
| `deleteInput` | Delete an ephemeral input (static inputs can't be deleted) |
| `setInputEnabled` | Enable or disable an input |

### Outputs

| Tool | Description |
|---|---|
| `listOutputs` | List outputs (speakers or zones; `includeDisabled` to include disabled ones) |
| `getOutput` | Get one output by ID |
| `setOutputVolume` | Set an output's volume (0–100) |
| `setOutputMute` | Mute or unmute an output |
| `setOutputEnabled` | Enable or disable an output |

### Groups

| Tool | Description |
|---|---|
| `listGroups` | List output groups (`includeDisabled` to include disabled ones) |
| `getGroup` | Get one group by ID |
| `setGroupVolume` | Set the volume of every output in a group (0–100) |
| `setGroupMute` | Mute or unmute a group |
| `setGroupEnabled` | Enable or disable a group |

### Routes

| Tool | Description |
|---|---|
| `listRoutes` | List routes, optionally filtered by `status`, `inputId` and `targetId` |
| `getRoute` | Get one route by ID |
| `createRoute` | Play an input to an output or group |
| `deleteRoute` | Stop and delete a route |
| `transferRoute` | Move a route to another output or group |
| `setRoutePause` | Pause or resume a route (pauseable inputs only) |

### Playback and master mute

| Tool | Description |
|---|---|
| `playback` | Play a URI on an output or group in one step (creates the input and route) |
| `getMasterMute` | Get the system-wide master mute state |
| `setMasterMute` | Turn the system-wide master mute on or off |

### Errors

A failed call returns an error result whose text starts with a category:

| Category | Meaning |
|---|---|
| `NotFound` | The input, output, group, route or target doesn't exist |
| `Validation` | The hub rejected the request (the hub's explanation is included) |
| `Conflict` | The hub refused the request in its current state, for example a disabled speaker; the hub's reason follows in parentheses. Change that state and retry |
| `RouteFailed` | The hub could not create the route |
| `SourceUnreachable` | The hub could not reach the audio source |
| `ServiceUnavailable` | The hub is temporarily unavailable |
| `Timeout` | The hub did not answer in time (5 s) |
| `Network` | The hub could not be reached |
| `MalformedResponse` | The hub's response didn't match the API |
| `HubError` | Any other hub error |
| `Internal` | Unexpected fault in sonora-mcp |

Arguments that don't match a tool's schema (for example a volume of 150) are rejected with
`validating "arguments": …` before anything is sent to the hub.

## Build from source

Requires Go 1.27.

```bash
go build -ldflags "-X github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=1.1.0" ./cmd/sonora-mcp
```

Without `-ldflags` the version is `dev`. Cross-build for a Raspberry Pi:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-X github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=1.1.0" -o sonora-mcp ./cmd/sonora-mcp
```

Before sending a change:

```bash
gofmt -l .      # must print nothing
go vet ./...
go test ./...   # offline; the hub is faked
```

The tests also check every tool's inputs and outputs against the hub's OpenAPI spec, published by
sonora-cli as `api.Spec`.

## Releases

Push a `vX.Y.Z` tag on `main` (`./release.sh` does this). The release workflow runs
[GoReleaser](.goreleaser.yaml), which builds linux, macOS and Windows for amd64 and arm64 and
publishes the archives and `checksums.txt` to a GitHub Release. Tags such as `v1.2.0-rc.1` are
published as prereleases. Pull requests run gofmt, vet and `go test -race`.

## Tech stack

- Go 1.27, standard library HTTP server
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) v1.8.0, Streamable HTTP (stateless)
- [sonora-cli](https://github.com/Sonora-Multiroom/sonora-cli) `hub` package for all hub requests
- GoReleaser for releases

## License

[GNU Affero General Public License v3.0](LICENSE)
