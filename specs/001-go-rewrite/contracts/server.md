# Contract: Command Line, HTTP Endpoints, Pi Service

## Command line

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

Startup log (stderr): version, hub URL, listen address, number of tools registered.

## HTTP endpoints

### `POST /mcp`

MCP Streamable HTTP, stateless (no `Mcp-Session-Id`). `GET` and `DELETE` on `/mcp` return 405.
Server info: `name: "sonora-mcp"`, `version: <build version>`. Capabilities: tools only.
Tool contract: [tools.md](tools.md).

Requests that arrive through a loopback address with a non-localhost `Host` header are rejected with
403 (DNS-rebinding protection). No CORS headers are sent.

### `GET /health`

Always `200 OK` while the server runs; `Content-Type: application/json`.

```json
{ "status": "ok", "server": "sonora-mcp", "version": "1.1.0", "hub": "reachable" }
```

`hub` is `"reachable"` or `"unreachable"`, from one read-only hub request bounded by 2 seconds.

### Anything else

`404 Not Found`.

## Build

```
go build -ldflags "-X github.com/tiger-seo/sonora-mcp/internal/version.Version=<v>" ./cmd/sonora-mcp
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "..." -o dist/pi/sonora-mcp-linux-arm64 ./cmd/sonora-mcp
gh release create <tag> dist/pi/sonora-mcp-linux-arm64 --title <tag>   # once per release
```

Without `-ldflags` the version is `dev`. Releases are tagged on `main` only, after the merge gate
passes with a tagged sonora-cli. `deploy/pi/install.sh`'s `RELEASE_TAG` is bumped to `<tag>` in
the commit that gets tagged, so the script always matches the release it ships with (T072).
Pre-merge builds are published as GitHub prereleases (`<tag>-rc.N`, `--prerelease`) and installed
with `--version`; `RELEASE_TAG` never points at one (T060a). `install.sh` is the single file copied
to the Pi.

## Pi service (`deploy/pi/`)

### `install.sh`

The only file copied to the Pi. The systemd unit is embedded as a heredoc; the built
`linux/arm64` binary is **not** embedded — the script downloads it from the GitHub Release whose
tag is baked in as `RELEASE_TAG` (see Build above). The repository is public, so no credentials
are needed. Run on the Pi with sudo, from wherever it was copied to:

```
sudo ./install.sh --hub-url <url> [--port <port>] [--host <address>] [--version <tag>]
```

| Effect | Detail |
|---|---|
| Binary | Downloaded with `curl -fsSL` from `https://github.com/tiger-seo/sonora-mcp/releases/download/<tag>/sonora-mcp-linux-arm64` (`<tag>` = `--version` if given, else the script's baked-in `RELEASE_TAG`) to `/usr/local/bin/sonora-mcp` (mode 0755) |
| Config | `/etc/default/sonora-mcp` with `SONORA_HUB_URL`, `SONORA_PORT`, `SONORA_HOST_ARG` |
| Unit | Written from the script's embedded heredoc to `/etc/systemd/system/sonora-mcp.service` |
| Service | `daemon-reload`; enabled at boot; started, or restarted if already running |
| Re-run | Overwrites binary, config and unit; restarts; never duplicates |
| Exit | 0 on success with `systemctl status` summary; non-zero with a message if not root, `--hub-url` missing, systemd or `curl` absent, or the download fails (network error or non-2xx, e.g. unknown tag) |

The embedded systemd unit runs `sonora-mcp` with the configured flags as a dynamic unprivileged
user, after the network is up. Restarts on failure after 2 seconds. Stopped with SIGTERM, which
triggers graceful shutdown.

Installation needs outbound internet access to GitHub; the running server does not.
