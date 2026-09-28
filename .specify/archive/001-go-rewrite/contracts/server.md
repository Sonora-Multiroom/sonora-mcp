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

A listen failure exits 1 without usage text: the flags parsed and were valid, the environment
refused them (port taken, address not on this machine), so repeating the usage would not help. It is
a startup failure, not a flag error, and still stops the server before it serves (Principle IV).

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
go build -ldflags "-X github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=<v>" ./cmd/sonora-mcp
```

Without `-ldflags` the version is `dev`.

Releases are built by GoReleaser (`.goreleaser.yaml`, as in sonora-cli, without Scoop). Pushing a
`v*` tag runs `.github/workflows/release.yml`, which builds linux, darwin and windows for amd64 and
arm64 (`CGO_ENABLED=0`, `-s -w`, version from the tag) and publishes a GitHub Release with
`sonora-mcp_<version>_<os>_<arch>.tar.gz` (`.zip` on windows) and `checksums.txt`. Tags with a
pre-release suffix (`v1.1.0-rc.1`) are published as prereleases. `release.sh` tags `main` and pushes
the tag. `.github/workflows/test.yml` runs `gofmt`, `go vet` and `go test -race` on pull requests.

Releases are tagged on `main` only, after the merge gate passes with a tagged sonora-cli.
`deploy/pi/install.sh`'s `RELEASE_TAG` is bumped to `<tag>` in the commit that gets tagged, so the
script always matches the release it ships with (T072). Pre-merge builds are published as
prereleases (`<tag>-rc.N`, tagged on the feature branch) and installed with `--version`;
`RELEASE_TAG` never points at one (T060a). `install.sh` is the single file copied to the Pi.

## Pi service (`deploy/pi/`)

### `install.sh`

The only file the Pi needs. The systemd unit is embedded as a heredoc; the built
`linux/arm64` binary is **not** embedded — the script downloads it from the GitHub Release whose
tag is baked in as `RELEASE_TAG` (see Build above). The repository is public, so no credentials
are needed. Install in one command, piping the script from `main` (whose `RELEASE_TAG` is the
latest release):

```
curl -fsSL https://raw.githubusercontent.com/Sonora-Multiroom/sonora-mcp/main/deploy/pi/install.sh | sudo bash -s -- --hub-url <url> [--port <port>] [--host <address>] [--version <tag>]
```

or run a copy on the Pi with sudo:

```
sudo ./install.sh --hub-url <url> [--port <port>] [--host <address>] [--version <tag>]
```

The script body runs from a `main` function called on its last line, so a download cut off part
way through executes nothing.

| Effect | Detail |
|---|---|
| Binary | Downloads `sonora-mcp_<version>_linux_arm64.tar.gz` and `checksums.txt` with `curl -fsSL` from `https://github.com/Sonora-Multiroom/sonora-mcp/releases/download/<tag>/` (`<tag>` = `--version` if given, else the script's baked-in `RELEASE_TAG`; `<version>` = `<tag>` without the leading `v`), checks the archive's SHA-256 against `checksums.txt`, extracts `sonora-mcp`, copies it to a temp file in `/usr/local/bin`, sets mode 0755, then renames it (`mv -f`) onto `/usr/local/bin/sonora-mcp`. The rename works while the old binary is running, and a failed download or checksum leaves it untouched |
| Config | `/etc/default/sonora-mcp` with `SONORA_HUB_URL`, `SONORA_PORT`, `SONORA_HOST_ARG` |
| Unit | Written from the script's embedded heredoc to `/etc/systemd/system/sonora-mcp.service` |
| Service | `daemon-reload`; enabled at boot; started, or restarted if already running |
| Re-run | Overwrites binary, config and unit; restarts; never duplicates |
| Exit | 0 on success with `systemctl status` summary; non-zero with a message if not root, `--hub-url` missing, systemd or `curl` absent, the OS is not 64-bit ARM (`uname -m` ≠ `aarch64`), the download fails (network error or non-2xx, e.g. unknown tag), or the checksum does not match |

The embedded systemd unit runs `sonora-mcp` with the configured flags as a dynamic unprivileged
user, after the network is up. Restarts on failure after 2 seconds. Stopped with SIGTERM, which
triggers graceful shutdown.

Installation needs outbound internet access to GitHub; the running server does not.
