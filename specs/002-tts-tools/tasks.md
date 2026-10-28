---

description: "Task list for 002-tts-tools"
---

# Tasks: Text-to-speech tools

**Input**: Design documents from `specs/002-tts-tools/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/tools.md](contracts/tools.md), [quickstart.md](quickstart.md)

**Tests**: Required. Constitution Principle V (test-first, NON-NEGOTIABLE): every test task comes
before its implementation and must be seen failing first.

**Organization**: one phase per user story, in spec priority order: US1 `speak` (P1), US2 playback
fields (P2), US3 `listTtsVoices` (P2), US4 cache tools (P4).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: the user story the task belongs to
- Every Go command runs with `GOWORK=off` when a `go.work` exists.

## Conventions used by every story

- Tests live in `internal/tools/` and drive tools through `newTestSession` against `newFakeHub`, using
  the harness helpers (`callTool`, `expectStructured`, `expectRequest`, `expectInvalidInput`,
  `expectErrorPrefix`, `expectAnnotations`, `hub.assertNoRequests`, `hub.requests()` for the
  two-request diagnosis cases).
- Inputs are structs with `json` and `jsonschema` (description-only) tags. Constraints are set at
  registration with `withMinLength`, `withEnum`, `withRange` (`internal/tools/schema.go`).
- Tools are registered with `add(s, toolSpec{Name, Description, Kind, Method, Path}, inputSchema[...](...), handler)`.
- Each TTS handler: `ctx, cancel := context.WithTimeout(ctx, c.Timeout)` (`c` = the tool's client),
  calls the `hub` function, and passes any error through `ttsError(ctx, client, hubURL, err)` (T005).

---

## Phase 1: Setup

- [X] T001 Bump the shared hub client: `go get github.com/Sonora-Multiroom/sonora-cli@v0.3.4` and `go mod tidy` in `go.mod`/`go.sum`. Run the gate (`gofmt -l .`, `go vet ./...`, `go test ./...`): all pass, 28 tools, `api.Spec` still 0.1.29.

---

## Phase 2: Foundational (blocks every story)

**Purpose**: the error category, the not-offered diagnosis, the speak client, the conformance
change and the new tool kind. All four tools depend on these.

- [X] T002 [P] Failing tests in `internal/tools/errors_test.go` (extend the existing table) for `classify`/`toolError`:
  - `*hub.TTSUnavailableError` with each of `DiagnosisNotInstalled` (with and without `LoadingDisabled`), `DiagnosisDisabled`, `DiagnosisRejected` (with `Reason`), `DiagnosisInert` and `DiagnosisVersionMismatch` gives `TtsUnavailable: <its Error()>`.
  - `DiagnosisUnknown` with `Cause: context.DeadlineExceeded`, and again with `Cause: &hub.DecodeError{...}`, gives `TtsUnavailable: text-to-speech is not available on this hub`, not `Timeout` or `MalformedResponse` (research R4).
  - `DiagnosisHubAddress` with `BaseURL` gives `Network: <its Error()>` exactly once, with no `: <err>` suffix repeating it.
  - A bare and a wrapped `&hub.TTSNotOfferedError{}` give `TtsUnavailable: text-to-speech is not available on this hub`.
  - A `*hub.TTSError` with `TARGET_NOT_FOUND` gives `NotFound`, with `PROVIDER_NOT_FOUND` `NotFound`, with `INVALID_REQUEST` (400) `Validation`, and with `PROVIDER_TIMEOUT` (503) `ServiceUnavailable`, each message containing the hub's message.
- [X] T003 Make T002 pass in `internal/tools/errors.go`:
  - add `CategoryTtsUnavailable = "TtsUnavailable"`;
  - in `classify`, check `*hub.TTSUnavailableError` before the decode and timeout checks: `DiagnosisHubAddress` gives `CategoryNetwork` with `Error()` as is, any other diagnosis gives `CategoryTtsUnavailable` with `Error()`;
  - map `hub.ClassTTSUnavailable` to `CategoryTtsUnavailable` in the switch.
- [X] T004 [P] Failing tests in `internal/tools/tts_unavailable_test.go` for the diagnosis helper `ttsError(ctx, client, hubURL, err)`, against a fake hub serving `GET /api/v2/extensions`:
  - With `&hub.TTSNotOfferedError{}`, the inventory `{"loadingEnabled":true,"extensions":[{"id":"tts","status":"DISABLED","rejectionReason":null}]}` gives a `*hub.TTSUnavailableError` with `DiagnosisDisabled`. One table row each for no `tts` extension (with `loadingEnabled:false` → `LoadingDisabled`), `REJECTED` with a reason, `INERT` and `ACTIVE`.
  - Inventory 404 gives `DiagnosisHubAddress`; inventory 500 gives `DiagnosisUnknown`.
  - Any other error (e.g. `&hub.TTSError{...}`) is returned unchanged with no inventory request (`hub.assertNoRequests`).
  - Add the inventory JSON constants to `internal/tools/fakehub_test.go`.
- [X] T005 Make T004 pass: create `internal/tools/tts.go` (package doc as in the other tool files) with `ttsError`, which returns `hub.DiagnoseTTSUnavailable(ctx, client, hubURL)` for an `errors.As` match on `*hub.TTSNotOfferedError` and `err` otherwise. Add `registerTTS(s, client, speakClient *http.Client, hubURL string) int`, returning 0 for now.
- [X] T006 Refactor, guarded by the existing suite (no behaviour changes, so no new failing test). Change the signature to `Register(s *mcp.Server, client, speakClient *http.Client, hubURL string) int` in `internal/tools/tools.go`. Call `registerTTS(s, client, speakClient, hubURL)` after the uniform `registrars` and add its count. Update every caller:
  - `cmd/sonora-mcp/main.go`: build `speakClient := hub.NewClientWithTimeout(hub.SpeakTimeout)` beside `hub.NewClient()` (research R6);
  - `internal/tools/harness_test.go` (`newTestSession` passes its client twice);
  - `internal/tools/conformance_test.go` (`TestRegisterCount` passes `nil, nil`);
  - `internal/server/server_test.go` (two calls) and `internal/server/shutdown_test.go`.

  The gate passes with 28 tools.
- [X] T007 [P] In `internal/tools/conformance_test.go`, make `successSchema` take the `application/json` content, or else `*/*` (TTS responses, research R7). Existing tools still pass.
- [X] T008 [P] Add `DestructiveIdempotent` to `Kind` in `internal/tools/register.go` (`DestructiveHint: &yes, IdempotentHint: true`), test-first in `internal/tools/register_test.go` (annotations of the new kind).

**Checkpoint**: the gate passes, 28 tools, nothing visible to agents has changed yet.

---

## Phase 3: User Story 1 - The agent says something in a room (P1) 🎯 MVP

**Goal**: `speak` with text, target and target type, plus optional provider, voice and language.

**Independent test**: against a fake hub, the hub receives exactly the given fields, and the tool
returns `announcementId`, `cacheHit` and `queueDepth`.

### Tests (write first, see them fail)

- [X] T009 [P] [US1] `internal/tools/tts_speak_test.go`, with `speakAcceptedJSON = {"announcementId":"6f1c…","cacheHit":false,"queueDepth":1}` in `fakehub_test.go`:
  - **Scenario 1**: `{text:"Dinner is ready", targetName:"kitchen", targetType:"SINGLE_OUTPUT"}` sends `POST /api/tts/speak` with exactly that body and returns `speakAcceptedJSON` structured.
  - **Scenario 2**: none of `providerName`, `voice` or `language` is in the body when omitted.
  - **Scenario 3**: `providerName:"piper-local"` and `voice` are passed unchanged.
  - **Scenario 4**: empty `text`, missing `targetName`, empty `targetName`, or `targetType:"ROOM"` each give `expectInvalidInput` and `assertNoRequests`.
  - **Scenario 5**: a 400 `{"code":"TARGET_NOT_FOUND","message":"… kitchn …"}` gives `NotFound:` with the target in the text.
  - **Scenario 6**: a 404 on `/api/tts/speak` and a `DISABLED` inventory give `TtsUnavailable: text-to-speech is not available on this hub: the TTS extension is installed but disabled in the hub's configuration`; a 404 and an inventory 404 give `Network:`.
  - **Scenario 7**: annotations are `StateChanging`; the description contains "queued", "targetId" and "billed".
  - **Speak client**: a session built with `Register(s, fast, slow, url)` (fast = 100 ms, slow = 2 s) and a speak response delayed 300 ms succeeds; with the two clients swapped it gives `Timeout:`.
  - **Diagnosis bound**: a 404 on `/api/tts/speak` answered at once, then an inventory response delayed past the bound, gives `TtsUnavailable: text-to-speech is not available on this hub` (not `Timeout`), within the bound.
  - **Conformance**: add `"speak"` to `contractTools` in `internal/tools/conformance_test.go` and set the expected count in `cmd/sonora-mcp/main_test.go` to 29 (see the count note in Dependencies). `TestConformance/speak` and the count fail until T010.

### Implementation

- [X] T010 [US1] In `internal/tools/tts.go`, add `speakIn` with `Text string` (`text`), `TargetName string` (`targetName`), `TargetType string` (`targetType`), `ProviderName *string` (`providerName,omitempty`), `Voice *string` (`voice,omitempty`) and `Language *string` (`language,omitempty`), each with a description. Register `speak` with:
  - `Kind: StateChanging, Method: "POST", Path: "/api/tts/speak"`;
  - `withMinLength("text", 1)`, `withMinLength("targetName", 1)`, `withEnum("targetType", targetTypes...)`;
  - the description per [contracts/tools.md](contracts/tools.md) "`speak`";
  - a handler that uses `speakClient` and returns `*hub.SpeakAccepted`, with errors through `ttsError` using `client`.
- [X] T011 [US1] Run `go test ./...`: the T009 tests, `TestConformance/speak` (inputs and output both) and the count of 29 now pass.

**Checkpoint**: an agent can speak; MVP.

---

## Phase 4: User Story 2 - The agent controls how an announcement plays (P2)

**Goal**: optional `playbackMode`, `routeVolume` and `duckingLevelDb` on `speak`.

**Independent test**: the three fields reach the hub unchanged when given, and are absent when not.

- [X] T012 [P] [US2] Extend `internal/tools/tts_speak_test.go`:
  - **Scenario 1**: `playbackMode:"duck-others"`, `routeVolume:40` and `duckingLevelDb:-20` are sent unchanged.
  - Scenario 2 from US1 now also checks that the three fields are absent.
  - **Scenario 2**: `routeVolume` 101 or -1 and `duckingLevelDb` -61 or 0.5 each give `expectInvalidInput` and no request.
  - **Scenario 3**: a 400 `{"code":"INVALID_REQUEST","message":"…mix…"}` gives `Validation:` with the hub's message.
  - **Scenario 4**: the description contains "duck-others", "mix" and that no output's volume changes.
- [X] T013 [US2] In `internal/tools/tts.go`, add to `speakIn`:
  - `PlaybackMode *string` (`playbackMode,omitempty`), with no enum (data-model: "none (the spec does not constrain it; the hub rejects other values)");
  - `RouteVolume *int` (`routeVolume,omitempty`) with `withRange("routeVolume", 0, 100)`;
  - `DuckingLevelDb *float64` (`duckingLevelDb,omitempty`) with `withRange("duckingLevelDb", -60, 0)`.

  Map all three onto `hub.SpeakRequest`, and extend the description per the contract. `TestConformance/speak` still passes.

**Checkpoint**: US1 and US2 work; the playback fields are optional.

---

## Phase 5: User Story 3 - The agent picks a valid voice (P2)

**Goal**: `listTtsVoices` for a provider, with optional language and engine filters.

**Independent test**: the request path and query are as given, and the voices come back unchanged.

- [X] T014 [P] [US3] `internal/tools/tts_voices_test.go`, with `voiceListJSON` in `fakehub_test.go` (`providerName` and two voices, one without `gender`):
  - **Scenario 1**: `{providerName:"piper-local"}` sends `GET /api/tts/providers/piper-local/voices` with no query and returns the list. The missing gender comes back as `null`.
  - **Scenario 2**: `language:"uk-UA"` sends the query `language=uk-UA`; `engine` sends `engine=…`.
  - **Scenario 3**: a 400 `PROVIDER_NOT_FOUND` gives `NotFound:`.
  - **Scenario 4**: `{"providerName":"piper-local","voices":[]}` is a success with an empty list.
  - **Scenario 5**: annotations are `ReadOnly`.
  - An empty `providerName` gives `expectInvalidInput`.
  - A 404 with a `REJECTED` inventory gives `TtsUnavailable: …failed to load: <reason>`.
  - **Bound**: a response delayed past the shared client's timeout gives `Timeout:` (the 5 s bound, not the speak bound).
  - **Conformance**: add `"listTtsVoices"` to `contractTools` and set the expected count in `main_test.go` to the number registered so far (30 when done in task order). They fail until T015.
- [X] T015 [US3] In `internal/tools/tts.go`, add `listVoicesIn` with `ProviderName string` (`providerName`), `Language *string` (`language,omitempty`) and `Engine *string` (`engine,omitempty`). Register `listTtsVoices` with:
  - `Kind: ReadOnly, Method: "GET", Path: "/api/tts/providers/{providerName}/voices"`;
  - `withMinLength("providerName", 1)`;
  - the description per the contract;
  - a handler that calls `hub.ListTTSVoices` and returns `*hub.TTSVoiceList`.
- [X] T016 [US3] Run `go test ./...`: the T014 tests, `TestConformance/listTtsVoices` and the count now pass.

**Checkpoint**: US1–US3 work.

---

## Phase 6: User Story 4 - The agent inspects and clears the TTS cache (P4)

**Goal**: `getTtsCacheStats` and `clearTtsCache`.

**Independent test**: the four figures come back under their spec names, and clearing sends the
right request and confirms it.

- [X] T017 [P] [US4] `internal/tools/tts_cache_test.go`, with `cacheStatsJSON = {"totalEntries":42,"totalSizeBytes":15728640,"maxSizeBytes":524288000,"entriesByProvider":{"piper-local":7}}` in `fakehub_test.go`:
  - **Scenario 1**: `getTtsCacheStats` sends `GET /api/tts/cache/stats` and returns `cacheStatsJSON` structured, with camelCase names (sonora-cli v0.3.4).
  - **Scenario 2**: `clearTtsCache` with `{}` sends `DELETE /api/tts/cache` (204, no query) and returns `{"cleared":true}`.
  - **Scenario 3**: with `providerName:"piper-local"` it sends the query `providerName=piper-local` and returns `{"cleared":true,"providerName":"piper-local"}`.
  - An unknown provider (400 `PROVIDER_NOT_FOUND`) gives `NotFound:`.
  - **Scenario 4**: `getTtsCacheStats` is `ReadOnly` and `clearTtsCache` is `DestructiveIdempotent`.
  - A 404 on the stats path with an `INERT` inventory gives `TtsUnavailable: …loaded but inactive`.
  - **Conformance**: add `"getTtsCacheStats"` and `"clearTtsCache"` to `contractTools`, set the expected count in `main_test.go` to the number registered so far (32 when done in task order) and update the "28 tool names" comment in `conformance_test.go`. They fail until T018.
- [X] T018 [US4] In `internal/tools/tts.go`:
  - register `getTtsCacheStats` (`noInput`, `ReadOnly`, `GET /api/tts/cache/stats`, returns `*hub.TTSCacheStats`);
  - add `clearCacheIn` with `ProviderName *string` (`providerName,omitempty`), and the confirmation type `cacheCleared` (`Cleared bool` `json:"cleared"`, `ProviderName *string` `json:"providerName,omitempty"`, with descriptions, like `inputDeleted`);
  - register `clearTtsCache` (`DestructiveIdempotent`, `DELETE /api/tts/cache`). Its description says that the next announcement of a cleared phrase is synthesised again, and a billed provider charges again.
- [X] T019 [US4] Run `go test ./...`: the T017 tests, `TestConformance` for both tools and the count now pass.

**Checkpoint**: all four tools work; 32 tools.

---

## Phase 7: Polish & cross-cutting

- [X] T020 [P] `README.md`:
  - change the tool count in the intro and in "Tools (28)" to 32;
  - list the four tools with one line each;
  - add a `TtsUnavailable` row to the error category table;
  - note that omitting `providerName` uses the hub's configured provider, which may be billed.
- [X] T021 [P] `AGENTS.md`:
  - "Active Technologies": sonora-cli v0.3.4;
  - "Project Structure": the 32 tools;
  - add a "Recent Changes" line for 002-tts-tools.
- [X] T022 [P] `docs/backlog/`: delete `tts-speak.md` and `tts-cache.md`, and remove their rows from `docs/backlog/INDEX.md`. `list-extensions.md` stays as rewritten (FR-017).
- [X] T023 Full gate with `GOWORK=off`: `gofmt -l .` (empty), `go vet ./...`, `go test ./...`. Then `go test -race ./...` in Docker (command in `AGENTS.md`).
- [X] T024 Read-only live checks from [quickstart.md](quickstart.md) §2 against `http://multiroom.lan:8080`: `listTtsVoices` for piper-local, an unknown provider, `getTtsCacheStats`.
- [X] T025 Audible check from [quickstart.md](quickstart.md) §3, **only after asking the user**:
  - record the output's volume, mute and enabled state and the master mute;
  - `speak` with `providerName: piper-local` and `routeVolume: 15`;
  - run the ALSA and route-metric checks, and confirm nothing changed.
- [X] T026 Create `specs/002-tts-tools/verification.md` with the commands, date, counts and results of T023–T025.

The master spec's tool inventory and changelog are updated by `/speckit-archive-run` after merge.
The release is MINOR (v1.3.0): bump `RELEASE_TAG` in `deploy/pi/install.sh` in the commit that gets
tagged, when the user asks for the release.

---

## Dependencies & execution order

- **Setup (T001)** comes before everything.
- **Foundational (T002–T008)**:
  - T002 → T003;
  - T004 → T005;
  - T006 depends on T005 (`registerTTS` exists);
  - T007 and T008 are independent.

  Every story needs all of Phase 2.
- **US1 (T009–T011)** comes after Phase 2.
- **US2 (T012–T013)** comes after US1: it extends `speakIn` and the same test file.
- **US3 (T014–T016)** and **US4 (T017–T019)** come after Phase 2 only, and are independent of US1 and of each other. Their implementation tasks all edit `tts.go`, so they are serialised when done by one person. Each story's test task edits `contractTools` and the count in `main_test.go`; set the count to the number of tools registered once that story is done (29, 30, 32 in task order; adjust if US4 is done before US3).
- **Polish (T020–T026)** comes after the stories that ship.

## Parallel opportunities

- Phase 2: T002, T004, T007 and T008 (different files).
- After Phase 2: the test tasks T009, T014 and T017 can be written together (different files, with fixtures added to `fakehub_test.go` one after another).
- Polish: T020, T021 and T022.

## Implementation strategy

1. **MVP**: Setup, then Foundational, then US1 (`speak` with the hub's defaults). That covers the main reason for the feature.
2. US2 (playback mode, announcement volume, ducking) on the same tool.
3. US3 (voices), then US4 (cache), each a self-contained addition.
4. Polish, live checks and `verification.md`; then commit and PR (MINOR release).
