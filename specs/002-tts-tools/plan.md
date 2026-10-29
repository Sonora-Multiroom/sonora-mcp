# Implementation Plan: Text-to-speech tools

**Branch**: `002-tts-tools` | **Date**: 2026-10-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-tts-tools/spec.md`

## Summary

Add four tools, `speak`, `listTtsVoices`, `getTtsCacheStats` and `clearTtsCache`. Each wraps one
TTS operation of sonora-cli's `hub` package. When the hub does not serve a TTS operation, the
tool asks `hub.DiagnoseTTSUnavailable` for the reason and returns it under a new error category,
`TtsUnavailable`. `speak` gets the 15 s announcement client; the other tools use the shared 5 s
client. The work needs sonora-cli v0.3.4, which tags `hub.TTSCacheStats` for JSON
([research](research.md) R1, R2).

## Technical Context

**Language/Version**: Go 1.27 (`CGO_ENABLED=0`)

**Primary Dependencies**:
- `github.com/modelcontextprotocol/go-sdk` v1.8.0 and `github.com/google/jsonschema-go`, both
  unchanged.
- `github.com/Sonora-Multiroom/sonora-cli` v0.3.1 → v0.3.4 (released 2026-10-28, sonora-cli #32,
  with the cache stats tags).

**Storage**: N/A (stateless proxy)

**Testing**:
- `go test ./...` with the in-memory MCP client against the fake hub (`fakehub_test.go`).
- The conformance test against `api.Spec` 0.1.29.

**Target Platform**: Linux arm64 (Raspberry Pi, systemd service), Windows amd64 for development

**Project Type**: MCP server (single Go module)

**Performance Goals**: none new. Every TTS call is one hub request, plus one inventory request only
after a "not offered" answer.

**Constraints**:
- `speak` is bounded at 15 s and the other tools at 5 s, the diagnosis included.
- Tests run offline.

**Scale/Scope**: +4 tools (28 → 32), one new error category; about 150 lines of code plus tests.

## Constitution Check

*Constitution 1.0.1. Checked before research, and again after design.*

| Principle | Status | Notes |
|---|---|---|
| I. API contract fidelity | Pass | All calls go through `hub`. The `TTSCacheStats` tag bug was fixed in sonora-cli (v0.3.4), not worked around with a local type (R2). Conformance covers the new tools once `successSchema` reads `*/*` responses (R7). No local `openapi.json`. |
| II. Agent-oriented tool design | Pass | camelCase verb+noun names. Descriptions and annotations per [contracts/tools.md](contracts/tools.md). Constraints are set at registration (`withMinLength`, `withEnum`, `withRange`). The release is MINOR (additions only). |
| III. Thin, stateless proxy | Pass | The diagnosis after a failure is one read of the hub's own extension report, and the mapping lives in `hub` (spec Assumptions; R3). No defaults are filled in: optional fields are sent only when set. |
| IV. Resilient, transparent errors | Pass | `speak` uses `hub.NewClientWithTimeout(hub.SpeakTimeout)` (R6); handlers pass the request context. Errors keep the `"<Category>: <message>"` contract, with `TtsUnavailable` added (R4). Panics are recovered by `add`. |
| V. Test-first | Pass (process) | `tasks.md` orders a failing test before each tool, the error mapping and the conformance change. |
| VI. Minimal dependencies | Pass | No new dependency. |
| Merge gate, versioning | Pass | `gofmt`, `go vet`, `go test` with `GOWORK=off`. `go.mod` pins a tagged release. |

Post-design re-check: no change. No violations, so Complexity Tracking stays empty.

## Project Structure

### Documentation (this feature)

```text
specs/002-tts-tools/
├── plan.md              # this file
├── research.md          # R1–R9
├── data-model.md        # inputs, results, TtsUnavailable mapping
├── quickstart.md        # offline gate, read-only and audible checks
├── contracts/tools.md   # the four tools and the new category
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
cmd/sonora-mcp/main.go           # builds speakClient (hub.SpeakTimeout), passes it to tools.Register
internal/tools/
├── tools.go                     # Register(s, client, speakClient, hubURL); calls registerTTS
├── tts.go                       # NEW: input types, the not-offered diagnosis helper, the 4 tools
├── errors.go                    # CategoryTtsUnavailable; TTSUnavailableError checked first
├── conformance_test.go          # successSchema reads */*; contractTools +4
├── harness_test.go              # newTestSession passes the test client as both clients
├── fakehub_test.go              # TTS and /api/v2/extensions fixtures, if helpers are needed
├── tts_speak_test.go            # NEW
├── tts_voices_test.go           # NEW
├── tts_cache_test.go            # NEW
└── tts_unavailable_test.go      # NEW: every diagnosis outcome, inventory 404 and failure
README.md, AGENTS.md             # tool list and count (28 → 32), TtsUnavailable
docs/backlog/                    # delete tts-speak.md and tts-cache.md; update INDEX.md
```

**Structure decision**: one new file, `internal/tools/tts.go`, for the TTS family, like `routes.go`
and `playback.go`. `registerTTS` takes both clients, so it is called separately from the uniform
`registrars` list.

## Design notes

- **`Register` signature**: `Register(s, client, speakClient, hubURL)`. `main` passes
  `hub.NewClient()` and `hub.NewClientWithTimeout(hub.SpeakTimeout)`. The test harness passes its one
  client twice, and `TestRegisterCount` passes `nil, nil`.
- **The diagnosis helper** (R3): `ttsError(ctx, client, hubURL, err) error` returns
  `hub.DiagnoseTTSUnavailable(ctx, client, hubURL)` for `*hub.TTSNotOfferedError`, and `err`
  otherwise. Each TTS handler wraps its context with `context.WithTimeout(ctx, c.Timeout)` (`c`
  being the tool's client), so the call and the diagnosis share one bound.
- **`classify`** (R4): `*hub.TTSUnavailableError` is checked first. `DiagnosisHubAddress` →
  `Network` with the message as is; any other diagnosis → `TtsUnavailable`. `hub.ClassTTSUnavailable`
  → `TtsUnavailable`.
- **A new `Kind`, `DestructiveIdempotent`** (`register.go`): `destructiveHint: true` and
  `idempotentHint: true`, for `clearTtsCache`. No existing kind gives both. The `stop*Routes` tools
  call themselves idempotent but are marked `Destructive` only; changing them is out of scope.
- **Spec correction** (R5): an unknown provider for `clearTtsCache` is `NotFound`, not
  `Validation`. Fixed in spec.md's edge cases.
- **Out of scope**: `engine`, `pitch`, `speakingRate` and `stylePrompt` on `speak` (FR-007), and
  `listExtensions` (it stays in the backlog).

## Prerequisite outside this repo

Done: sonora-cli v0.3.4 (#32) gives `hub.TTSCacheStats` its `json` tags, with a marshal test.

## Complexity Tracking

None.
