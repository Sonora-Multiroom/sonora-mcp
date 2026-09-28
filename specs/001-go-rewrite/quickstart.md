# Quickstart: Validate the Go Rewrite

How to prove the feature works end to end. Contracts: [contracts/tools.md](contracts/tools.md),
[contracts/server.md](contracts/server.md).

## Prerequisites

- Go 1.27 (`go version`).
- sonora-cli checked out next to this repo on a branch that contains the public `hub/` package **and**
  the `CreateInputRequest` optional-boolean fix ([research.md](research.md) R7).
- For the live checks: a running hub (e.g. `http://multiroom.lan:8080`), Node.js for
  `npx @modelcontextprotocol/inspector`, and a Raspberry Pi (ARM64) with systemd for the Pi checks.

## 1. Workspace (development only)

```powershell
cd D:\projects-sonora
go work init ./sonora-cli ./sonora-mcp   # once; never commit go.work
```

## 2. Merge gate (offline)

```powershell
cd D:\projects-sonora\sonora-mcp
gofmt -l .          # expect no output
go vet ./...
go build ./...
go test -race ./... # expect PASS with the network disabled (SC-006); includes the T068a
                     # architecture check (FR-006/FR-008: no direct net/http use or cached
                     # hub state in internal/tools)
```

Before merging, repeat with the workspace off and a tagged sonora-cli:

```powershell
$env:GOWORK="off"; go build ./...; go test ./...; Remove-Item Env:GOWORK
```

**Drift check (SC-006)**: temporarily change the volume maximum of `setOutputVolume` to 150 and run
`go test ./internal/tools/`. Expect a failure naming `setOutputVolume`, `volume`, `maximum`, 150 vs 100.
Revert.

## 3. Run locally

```powershell
go build -ldflags "-X github.com/tiger-seo/sonora-mcp/internal/version.Version=1.1.0" -o sonora-mcp.exe ./cmd/sonora-mcp
.\sonora-mcp.exe                                   # expect usage on stderr, exit code 2
.\sonora-mcp.exe --help                            # expect usage, exit code 0
.\sonora-mcp.exe --multiroom-url http://multiroom.lan:8080/
```

In another terminal:

```powershell
curl http://localhost:3001/health
# expect {"status":"ok","server":"sonora-mcp","version":"1.1.0","hub":"reachable"}
```

Start it with `--multiroom-url http://127.0.0.1:9` (nothing listening) and repeat: expect
`"hub":"unreachable"` with status 200 in about 2 s or less.

## 4. Agent parity (SC-001, SC-002)

1. `npx @modelcontextprotocol/inspector`, connect to `http://localhost:3001/mcp` (Streamable HTTP).
   Expect 24 tools with the names in [contracts/tools.md](contracts/tools.md). List/get tools are
   marked read-only; `deleteInput` and `deleteRoute` are marked destructive.
2. Against the real hub, call: `listOutputs`, `getOutput`, `setOutputVolume` (35), `setOutputMute`,
   `setGroupVolume`, `listRoutes`, `playback` (a stream URI to one output), `transferRoute` (to a group),
   `setRoutePause` (true, then false), `deleteRoute`, `createInput` without `enabled` (expect
   `enabled: true` in the result), `deleteInput`, `setMasterMute` (true, then false).
   Each result has `structuredContent` and a text copy with the same data.
3. VS Code: with the existing `.vscode/mcp.json` (unchanged), the server shows up and tools run.

## 5. Errors (SC-003, SC-004)

- `getOutput` with `garage` → text starts `NotFound:`.
- `setOutputVolume` with 150 → `validating "arguments": ...`; the hub log shows no request; the server
  log shows `outcome=InvalidInput`.
- Stop the hub (or point at a non-routable address) and call `listOutputs` → `Network:` or `Timeout:`
  within 10 s; `/health` still returns 200.

## 6. `--host` (FR-013a)

- Start with `--host 127.0.0.1`: `curl http://localhost:3001/health` works; from another LAN machine
  `curl http://<pc-ip>:3001/health` is refused.
- Start without `--host`: the LAN request works.

## 7. Graceful shutdown (FR-017a)

Start the server, begin a slow call (point it at a hub that delays responses, or call during a hub
restart), then press Ctrl+C. Expect the call to complete or time out, then exit code 0.

## 8. Raspberry Pi (SC-005, SC-008)

On the development machine:

```powershell
$env:GOOS="linux"; $env:GOARCH="arm64"; $env:CGO_ENABLED="0"
go build -ldflags "-X github.com/tiger-seo/sonora-mcp/internal/version.Version=1.1.0" -o dist/pi/sonora-mcp-linux-arm64 ./cmd/sonora-mcp
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED
gh release create v1.1.0 dist/pi/sonora-mcp-linux-arm64 --title v1.1.0   # once per release (T060a)
scp deploy/pi/install.sh pi@multiroom.lan:~/install.sh
```

`install.sh` is the only file that needs to reach the Pi (SC-005, SC-008); it downloads the
binary itself from the GitHub Release tagged in its `RELEASE_TAG`, so the Pi needs outbound
internet access to `github.com` during install (not afterward).

On the Pi:

```bash
sudo ~/install.sh --hub-url http://localhost:8080
systemctl status sonora-mcp            # active (running), enabled
curl localhost:3001/health             # hub: reachable, within 2 s of start
sudo ~/install.sh --hub-url http://localhost:8080 --port 3002   # re-run: reconfigures, no duplicate unit
sudo systemctl kill -s KILL sonora-mcp; sleep 3; systemctl is-active sonora-mcp   # active (restarted)
sudo reboot   # after boot: systemctl is-active sonora-mcp → active
```

Then point an assistant on the development machine at `http://multiroom.lan:3002/mcp` and repeat a
few calls from step 4.

## 9. Node removal (SC-007)

```powershell
git ls-files | Select-String -Pattern 'package(-lock)?\.json|tsconfig|\.ts$|update-openapi|openapi\.json'
# expect no output
```
