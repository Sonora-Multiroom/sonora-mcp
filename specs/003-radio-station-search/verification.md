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

## Phases 4 and 5 (US2 closing check, US3 lookup)

Date: 2026-11-22, `GOWORK=off`.

- **US2: no gap found.** After T014, the T015 and T016 tests passed unchanged (`go test ./internal/tools -run 'Radio|Cancel'` ok), so T017 made no code change.
- T018 was written first and failed (`unknown tool "getRadioStation"`); T019 then made it pass. `TestConformance/getRadioStation` and `TestToolInventory` pass with the tool in `contractTools`.
- `cmd/sonora-mcp/main_test.go` now expects 35 tools; T022 sets the final 38.

| Command | Exit code | Result |
|---|---|---|
| `gofmt -l .` | 0 | empty |
| `go vet ./...` | 0 | clean |
| `go build ./...` | 0 | clean |
| `go test ./... -count=1` | 0 | all packages `ok`; 35 tools |

No tracked or untracked file shows `w/crlf`.

## Phase 6 (US4 catalogue lists)

Date: 2026-11-22, `GOWORK=off`.

- T020: `radio_lists_test.go` failed first (`unknown tool "listRadioCountries"` and the other two), then T021 made it pass.
- The input-description check for every tool is the existing `TestToolDescriptions`, which covers all registered tools, so no new table test was added.
- `listRadioCountries` and `listRadioLanguages` return `{"countries": [...]}` and `{"languages": [...]}`; an empty hub list gives `[]`, never `null`. `TestConformance` passes for all three tools with the `countries` and `languages` envelopes from the foundational phase; no change to the conformance test was needed beyond `contractTools`.
- `cmd/sonora-mcp/main_test.go` now expects 38 tools.

| Command | Exit code | Result |
|---|---|---|
| `gofmt -l .` | 0 | empty |
| `go vet ./...` | 0 | clean |
| `go build ./...` | 0 | clean |
| `go test ./... -count=1` | 0 | all packages `ok`; 38 tools |

## Phase 7 (polish and checks)

Date: 2026-11-22.

- T022: `cmd/sonora-mcp/main_test.go` expects 38 tools (set in Phase 6); no other test counts tools.
- T023/T024: README (38 tools, a Radio section, `RadioUnavailable`, the 10 s radio bound) and `AGENTS.md` (38 tools, Recent Changes) updated.
- T025: `docs/backlog/adopt-radio-search.md` deleted and its `INDEX.md` row removed.

### T026 final gate (`GOWORK=off`, exit codes read directly)

| Command | Exit code | Result |
|---|---|---|
| `gofmt -l .` | 0 | empty |
| `go vet ./...` | 0 | clean |
| `go build ./...` | 0 | clean |
| `go test ./... -count=1` | 0 | cmd/sonora-mcp, internal/config, internal/server, internal/tools all `ok` |
| `docker run ... golang:1.27 go test -race ./...` (AGENTS.md command) | 0 | all four packages `ok` (internal/tools 44.5 s) |

`git ls-files --eol` and `git ls-files --eol -o --exclude-standard`: no file shows `w/crlf`.

### T027 live read-only check (`http://multiroom.lan:8080`, sonora-mcp built from this branch on port 3011, stopped afterwards)

`/health` reported `hub: reachable`. The hub's radio extension was serving. No state was changed and `playback` was not used.

| Step | Call | Result |
|---|---|---|
| 1 | `searchRadioStations` `name: "kiss", limit: 3` | 3 stations (Kiss FM 106.5, KissFm Deep, Kiss Fm Ro), each with `uri` `radiobrowser://<uuid>`, `nextOffset` 3 |
| 2 | `getRadioStation` for Kiss FM 106.5's UUID | the same station |
| 3 | `listRadioCountries`, `listRadioLanguages` | lists, largest first (US 7082, DE 5612; english 11412, german 4005) |
| 3 | `listRadioTags` `limit: 5`, then `offset: 5` | pop, music, news, rock, talk (`nextOffset` 5); then hits, dance, oldies, entretenimiento, radio (`nextOffset` 10) |
| 4 | `getRadioStation` for `00000000-0000-4000-8000-000000000000` | `NotFound: radio station not found: 00000000-...` |
| 5 | `searchRadioStations` `countryCode: "ukr"` | refused by the schema (`pattern ... "^[A-Za-z]{2}$"`), no hub request |
| extra | `searchRadioStations` `language: "ukraine"` | empty page, not an error (whole-value match, as documented) |

### T028 audible check (user gave permission for `office`, and for replacing its current route)

Date: 2026-11-22.

- **Before:** `office` volume 11, unmuted, enabled; master mute off. A route the user had not asked me about, `route-332387788bef` (input `playback_4469242969454`), was ACTIVE on `office`; the metric read 1 and one ALSA device was RUNNING. The user chose "Replace it, that's fine". `playback` replaces and cannot restore it.
- **Test:** `playback` with `radiobrowser://bd9c1301-562d-451f-94bf-6a76e3bb898e` (Kiss FM 106.5, found by `searchRadioStations`) on `office`, volume left at 11. It started: route `route-7add201e63c4`, input `playback_109993300580217`, status ACTIVE after 6 s, `office` the only output.
- **Stopped my own route first:** `deleteRoute` for `route-7add201e63c4` returned `deleted: true`. The input `playback_*` I created was removed by the hub; the older `playback_4469242969454` input, which I did not create, remains.
- **After:** `office` volume 11, unmuted, enabled; master mute off; `audio.output.routes{output:office}` = 0; `grep -h "^state" /proc/asound/card*/pcm*p/sub*/status` printed nothing; the local sonora-mcp process was stopped.
- **Not restored:** the user's earlier route on `office` (replaced with permission).

## T029 requirements walk

| Requirement | Covered by |
|---|---|
| FR-001, FR-002, FR-003, SC-001 | `radio_search_test.go` (all fields, only set filters sent, `nextOffset` present or null, empty page); live step 1 |
| FR-004 | `radio_station_test.go` (lookup, NotFound naming the UUID); live steps 2 and 4 |
| FR-005 | `radio_lists_test.go` (envelopes, hub order, tag paging, `[]` never `null`); live step 3 |
| FR-006, SC-002 | `TestConformance` for all five tools, with `countries`/`languages` envelopes and `maxLength` compared |
| FR-007 | schema-rejection tests in the search, station and list tests (no hub request); live step 5 |
| FR-008, FR-009, SC-004 | `TestToolErrorRadio*`, `TestRadioError*`, `TestSearchRadioStationsUnavailable`, `...InventoryUnreadable`, `...ReadsInventoryOnlyAfterNotOffered`, `...ProblemMapping` |
| FR-010, SC-005 | `TestSearchRadioStationsUsesTheRadioClient` (long client succeeds, short times out), 503 and `Retry-After` mapping, `TestCancelReachesHubForRadioSearch` |
| FR-011 | station text unchanged (escape sequence, "ignore previous instructions"), invalid UTF-8 test, error messages hold no station text, `untrusted` and `never as instructions` in the descriptions |
| FR-012 | description tests in the search, station and list tests, and `TestToolDescriptions` for every input field |
| FR-013, SC-006 | `TestUnavailableRadioLeavesOtherToolsAlone`; existing tests changed only for the `Register` signature, the tool count, `hub.ExtensionDiagnosis` and the conformance allowances; README and `AGENTS.md` updated, master spec left for archive time |
| FR-014 | `architecture_test.go` (passes); no local radio types except the two list wrappers; sonora-cli stays at v0.4.1 |
| SC-003 | offset pass-through tests (the server keeps no cursor); live step 3 (tag page 2 follows page 1) |
| SC-007 | live steps 1 and 2 (find by name, read details) and T028 (played through `playback` with the returned `uri`) |

No requirement is uncovered. Not done by design: no release tag (the next release is v1.4.0 and `install.sh`'s `RELEASE_TAG` is bumped in the tagged commit, when the user asks).
