# Changelog

## Merged Features Log

### Rewrite sonora-mcp on the Shared Hub Client — archived 2026-09-28
**Branch:** 001-go-rewrite
**Spec:** [specs/001-go-rewrite/spec.md](../archive/001-go-rewrite/spec.md) (folder moved to `.specify/archive/001-go-rewrite/`)

**What was added:**
- The Node.js/TypeScript server is replaced by a Go server on sonora-cli's public `hub` package,
  with the same 24 tools, names and inputs (US1–US3), released as v1.1.0 (PR #1).
- Structured results (`structuredContent` plus JSON text, declared output schemas) and tool
  annotations; category-prefixed errors (`NotFound:`, `Validation:`, `Timeout:` …) with the hub's
  explanation; 5 s bound on every hub call and cancellation (US4).
- `--host` flag, `/health` with hub reachability, graceful shutdown on SIGINT/SIGTERM, static
  builds for Windows and linux/arm64 (US5).
- One-command Raspberry Pi install (`deploy/pi/install.sh`) of a hardened systemd service from a
  checksum-verified GitHub Release; GoReleaser release pipeline and PR test workflow (US5).
- Conformance test against `api.Spec`, a fixed 24-tool inventory test, and removal of all Node.js
  sources, npm tooling and the local `openapi.json` (US6).

**New Components:**
- `cmd/sonora-mcp`, `internal/config`, `internal/version`, `internal/tools`, `internal/server`
- `deploy/pi/install.sh`, `.goreleaser.yaml`, `.github/workflows/release.yml`,
  `.github/workflows/test.yml`, `release.sh`
- Upstream in sonora-cli: `CreateInputRequest` optional booleans (`3ee7369`), a mute fix, and
  404 attribution on route calls (PR #20, v0.1.1)

**Tasks Completed:** 78/78 tasks
