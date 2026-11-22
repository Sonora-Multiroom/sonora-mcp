# Verification: Radio station search and browse

## Baseline

Date: 2026-11-22, branch `003-radio-station-search`, before any code change, `GOWORK=off` (no `go.work` exists).

| Command | Exit code | Result |
|---|---|---|
| `gofmt -l .` | 0 | empty |
| `go vet ./...` | 0 | clean |
| `go test ./...` | 0 | all packages `ok` (cmd/sonora-mcp, internal/config, internal/server, internal/tools) |

`go.mod` pins `github.com/Sonora-Multiroom/sonora-cli v0.4.1`. No tracked file shows `w/crlf`.

## Phases 1 and 2 (setup and foundational)

Date: 2026-11-22, `GOWORK=off`.

- T002/T003: `TestToolErrorRadio` and `TestToolErrorRadioProblems` failed to compile (no `CategoryRadioUnavailable`), then passed.
- T004/T005: `TestRadioError*` failed to compile (`radioError` undefined), then passed.
- T006/T007: `TestInputSchemaMaxLength*` failed to compile (`withMaxLength` undefined), then passed.
- T008: `Register` takes `radioClient`; every caller updated; `registerRadio` is a stub returning 0.
- T009: `listEnvelopes` gained `countries`, `languages`; `constraintKeys` gained `maxLength`.
- T010: radio fixtures added to `fakehub_test.go`; T011: `hub.TTSDiagnosis` replaced by `hub.ExtensionDiagnosis`.

| Command | Exit code | Result |
|---|---|---|
| `gofmt -l .` | 0 | empty |
| `go vet ./...` | 0 | clean |
| `go build ./...` | 0 | clean |
| `go test ./...` | 0 | all packages `ok`; `TestRegisterCount` still reports 33 tools |

No tracked or untracked file shows `w/crlf`. The race run and the live checks belong to T026 and T027.

## Phase 3 (US1 search, with the US2 error tests)

Date: 2026-11-22, `GOWORK=off`.

- T013, T015, T016 were written first and failed (`unknown tool "searchRadioStations"`; the cancel test saw no request). T014 then made them pass.
- `TestConformance/searchRadioStations` and `TestToolInventory` pass with the tool added to `contractTools`.
- `cmd/sonora-mcp/main_test.go` now expects 34 tools; T022 sets the final 38.
- `TestRunStopsCleanlyOnSignal` failed once under a full parallel run (did not return within 6 s) and passed in three isolated reruns (0.2 to 6.1 s). It is timing-sensitive and unrelated to this change.

| Command | Exit code | Result |
|---|---|---|
| `gofmt -l .` | 0 | empty |
| `go vet ./...` | 0 | clean |
| `go build ./...` | 0 | clean |
| `go test ./... -count=1` | 0 | all packages `ok`; 34 tools |
