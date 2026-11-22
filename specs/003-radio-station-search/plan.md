# Implementation Plan: Radio station search and browse

**Branch**: `003-radio-station-search` | **Date**: 2026-11-22 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-radio-station-search/spec.md`

## Summary

Add five read-only tools, `searchRadioStations`, `getRadioStation`, `listRadioCountries`, `listRadioLanguages`
and `listRadioTags`. Each wraps one radio operation of sonora-cli's `hub` package, which v0.4.1 (already
pinned) has ([research](research.md) R1). A radio call gets its own 10 s client (`hub.RadioTimeout`), because
the hub's catalogue budget is 5 s. When the hub does not serve a radio operation, the tool asks
`hub.DiagnoseRadioUnavailable` for the reason and returns it under a new error category, `RadioUnavailable`.
Radio problems keep their existing categories, with the offending parameter named. Station text passes through
unchanged and the descriptions warn that it is untrusted (R6).

The hub's result types already carry every field of the spec's schemas (checked field by field on 2026-11-22:
no field is missing or extra), so no change in sonora-cli is needed.

## Technical Context

**Language/Version**: Go 1.27 (`CGO_ENABLED=0`)

**Primary Dependencies**:
- `github.com/modelcontextprotocol/go-sdk` v1.8.0 and `github.com/google/jsonschema-go`, unchanged.
- `github.com/Sonora-Multiroom/sonora-cli` v0.4.1, unchanged (`api.Spec` of hub API 0.1.35).

**Storage**: N/A (stateless proxy)

**Testing**:
- `go test ./...` with the in-memory MCP client against the fake hub (`fakehub_test.go`).
- The conformance test against `api.Spec` 0.1.35, now also comparing `maxLength`.

**Target Platform**: Linux arm64 (Raspberry Pi, systemd service), Windows amd64 for development

**Project Type**: MCP server (single Go module)

**Performance Goals**: none new. Every radio call is one hub request, plus one inventory request only after a
"not offered" answer.

**Constraints**:
- Radio calls, and a diagnosis that follows, are bounded at 10 s; every other tool keeps 5 s (15 s for `speak`).
- Tests run offline.

**Scale/Scope**: +5 tools (33 → 38), one new error category, one new schema helper; about 150 lines of code plus
tests.

## Constitution Check

*Constitution 1.0.1. Checked before research, and again after design.*

| Principle | Status | Notes |
|---|---|---|
| I. API contract fidelity | Pass | All calls go through `hub`; no local radio types except two list wrappers (the hub returns bare arrays). Conformance covers the five tools, and `maxLength` is now compared (R4, R5). No `openapi.json` copy. |
| II. Agent-oriented tool design | Pass | camelCase verb+noun names; descriptions and annotations per [contracts/tools.md](contracts/tools.md); every spec constraint is set at registration (R5); all five are `readOnlyHint`. Additions only, so MINOR. |
| III. Thin, stateless proxy | Pass | No cache, no cursor (`nextOffset` is the hub's). The diagnosis is one read of the hub's own extension report; the mapping lives in `hub`. No defaults filled in: optional fields are pointers (R5). |
| IV. Resilient, transparent errors | Pass | `hub.NewClientWithTimeout(hub.RadioTimeout)` (R3), request context passed on; errors keep the `"<Category>: <message>"` form with one new category (R2). |
| V. Test-first | Pass (process) | `tasks.md` orders a failing test before each tool, the error mapping and the conformance change. |
| VI. Minimal dependencies | Pass | No new dependency. |
| Merge gate, versioning | Pass | `gofmt`, `go vet`, `go test` with `GOWORK=off`; `go.mod` pins a tagged release. |

Post-design re-check: no change. No violations, so Complexity Tracking stays empty.

## Project Structure

### Documentation (this feature)

```text
specs/003-radio-station-search/
├── plan.md              # this file
├── research.md          # R1–R11
├── data-model.md        # inputs, results, error mapping
├── quickstart.md        # offline gate, read-only and audible checks
├── contracts/tools.md   # the five tools and the new category
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
cmd/sonora-mcp/main.go           # builds radioClient (hub.RadioTimeout), passes it to tools.Register
internal/tools/
├── tools.go                     # Register(s, client, speakClient, radioClient, hubURL); calls registerRadio
├── radio.go                     # NEW: input types, the not-offered diagnosis helper, list wrappers, the 5 tools
├── errors.go                    # CategoryRadioUnavailable; RadioUnavailableError checked first; parameter in message
├── schema.go                    # withMaxLength
├── conformance_test.go          # contractTools +5; listEnvelopes +countries, languages; constraintKeys +maxLength
├── harness_test.go              # a session helper that sets the radio client
├── fakehub_test.go              # radio fixtures (station, page, problem bodies), if helpers are needed
├── radio_search_test.go         # NEW
├── radio_station_test.go        # NEW
├── radio_lists_test.go          # NEW: countries, languages, tags
├── radio_unavailable_test.go    # NEW: every diagnosis outcome, inventory 404 and failure, problem mapping,
│                                #      and the radio timeout (a call outlasting 5 s still succeeds)
└── tts_unavailable_test.go      # hub.TTSDiagnosis → hub.ExtensionDiagnosis (R9)
README.md, AGENTS.md             # tool list and count (33 → 38), RadioUnavailable, Recent Changes
docs/backlog/                    # delete adopt-radio-search.md; update INDEX.md
```

**Structure decision**: one new file, `internal/tools/radio.go`, for the radio family, like `tts.go` and
`chimes.go`. `registerRadio` takes both clients (the radio client for the call, the standard client for the
diagnosis), so it is called separately from the uniform `registrars` list, as `registerTTS` is.

## Design notes

- **`Register` signature** (R7): `Register(s, client, speakClient, radioClient, hubURL)`. `main` passes
  `hub.NewClient()`, `hub.NewClientWithTimeout(hub.SpeakTimeout)` and `hub.NewClientWithTimeout(hub.RadioTimeout)`.
  The existing harness helpers pass their one client for all three, and `TestRegisterCount` passes `nil, nil, nil`.
- **The diagnosis helper** (R3): `radioError(ctx, client, hubURL, err) error` returns
  `hub.DiagnoseRadioUnavailable(ctx, client, hubURL)` for `*hub.RadioNotOfferedError`, and `err` otherwise. Each
  radio handler wraps its context with `withTimeout(ctx, radioClient)`.
- **`classify`** (R2): `*hub.RadioUnavailableError` is checked first. `DiagnosisHubAddress` → `Network`; any
  other diagnosis → `RadioUnavailable`. `hub.ClassRadioUnavailable` → `RadioUnavailable`. A radio `*hub.APIError`
  with a `Parameter` gets ` (parameter <name>)` appended to its message.
- **List wrappers** (R4): `radioCountries{Countries []hub.RadioCountry}` and
  `radioLanguages{Languages []hub.RadioLanguage}`. The page types are returned as the hub gives them.
- **Not changed**: `playback` and every other existing tool.
- **Out of scope**: voting, favouriting and recording stations, `listExtensions` (stays in the backlog), any
  sonora-cli change.

## Prerequisite outside this repo

None. sonora-cli v0.4.1 has the radio operations and result types.

## Complexity Tracking

None.
