# Agent Instructions

## Product naming

- The product is **Sonora Multiroom**. This repo is **sonora-mcp**, the MCP server for it.
- The backend it talks to is the **Multiroom Audio Hub API**. Its OpenAPI spec is published by
  sonora-cli as `api.Spec`; there is no copy in this repo.

## Git / PR conventions

- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).
- Never add a `Co-Authored-By` trailer (or similar attribution) to commit messages, and never add
  "Generated with Claude Code" (or similar) to PR descriptions.
- Module path is `github.com/Sonora-Multiroom/sonora-mcp`. The version ldflag must use it exactly
  (`-X github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=<v>`): Go silently ignores
  `-X` for a package that isn't in the build, and the binary then reports `dev`.

## Project context

- Governance and engineering principles live in `.specify/memory/constitution.md`; read it before
  planning or implementing. Feature work follows the Spec Kit workflow
  (`specs/<NNN-feature>/spec.md` → `plan.md` → `tasks.md`), driven by the `/speckit-*` skills.
- All hub access goes through sonora-cli's `hub` package (`architecture_test.go` enforces it).
  When `hub` is wrong, fix it in sonora-cli first (PR, patch tag), then bump here
  (`go get github.com/Sonora-Multiroom/sonora-cli@vX.Y.Z`). Never work around it locally.
- `conformance_test.go` compares every tool's input and output schema with `api.Spec`. When it
  fails, suspect the code, not the test: it has caught real `hub` bugs (e.g. mute endpoints
  decoded as the wrong response type). Don't loosen it to make a change pass.

## Building and testing

- The gate: `gofmt -l .` (must be empty), `go vet ./...`, `go test ./...`. If you use a `go.work`
  (git-ignored) to develop against a local sonora-cli checkout, run the gate and release builds
  with `GOWORK=off`, so they use the sonora-cli version pinned in `go.mod`.
- Tests are offline: tools are driven through an in-memory MCP client against a fake hub
  (`internal/tools/fakehub_test.go`, `harness_test.go`).
- `go test -race` needs cgo, so run it on Linux (CI does this on every PR). Locally on Windows:
  `docker run --rm --network none -e GOWORK=off -e GOPROXY=off -e GOFLAGS=-mod=mod -v <repo>:/src
  -v <GOMODCACHE>:/go/pkg/mod -w /src golang:1.27 go test -race ./...`.

## Testing against the real hub

- It is a real home audio system (hub at `http://multiroom.lan:8080`, sonora-mcp as a service on
  the same Raspberry Pi). Read-only tools and requests that return 404 are fine to run.
- State-changing tools make sound or change speakers. Ask first, record the current state, and
  restore it right after (volume, mute, enabled, master mute; delete any routes and inputs you
  created). Don't touch routes you didn't create.
- Rebooting the Pi restarts the hub, which drops active routes. Right after boot `/health` reports
  `"hub":"unreachable"` for a few seconds; that's expected.

## Releases and the Pi

- Pushing a `vX.Y.Z` tag runs GoReleaser (`.goreleaser.yaml`, `.github/workflows/release.yml`).
  `-rc.N` tags are published as prereleases. `./release.sh` tags `main`; tag release candidates on
  a branch by hand.
- `deploy/pi/install.sh` is run as `curl … | sudo bash -s -- --hub-url <url>`. Keep all of its
  logic inside functions and `main "$@"` as the last line, so a cut-off download runs nothing.
  Bump its `RELEASE_TAG` in the commit that gets tagged. It downloads the release archive and
  verifies it against `checksums.txt`, so the GoReleaser archive naming
  (`sonora-mcp_<version>_linux_arm64.tar.gz`) is part of its contract.
