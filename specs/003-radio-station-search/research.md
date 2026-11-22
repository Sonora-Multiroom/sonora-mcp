# Research: Radio station search and browse

Decisions for [plan.md](plan.md). Each one was checked against the pinned sonora-cli v0.4.1 source and its
`api.Spec` (hub API 0.1.35) on 2026-11-22.

## R1: No sonora-cli bump is needed

**Decision**: keep `github.com/Sonora-Multiroom/sonora-cli v0.4.1`.

**Rationale**: the backlog item was written before the route-chime change moved this repository from v0.3.4
to v0.4.1. `hub/radio.go` (CLI-014, commit 7e7cb6b) is in the tags v0.4.0 and v0.4.1, and v0.4.1's embedded
spec already has the five `/api/radio/*` paths. The module cache holds v0.4.1, so the work builds offline.
`hub.RadioUnavailableError`, `hub.DiagnoseRadioUnavailable`, `hub.ClassRadioUnavailable` and
`hub.ExtensionDiagnosis` are all there.

**Alternatives considered**: waiting for a newer tag. Nothing in it is needed, and a bump would only add
unrelated change.

**Consequence**: the spec's Assumptions and FR-014 said a later tag was needed; both are corrected.

## R2: Error mapping

**Decision**: add the category `RadioUnavailable` (named like `TtsUnavailable`), and extend `classify` in
`internal/tools/errors.go`:

1. `*hub.RadioUnavailableError` is checked first, next to the TTS check. `DiagnosisHubAddress` gives `Network`
   with the message as is; any other diagnosis gives `RadioUnavailable`.
2. `hub.ClassRadioUnavailable` gives `RadioUnavailable` (the generic "radio is not available on this hub" for a
   `*hub.RadioNotOfferedError` that was not diagnosed).
3. Radio problem types arrive classified by `hub.ClassifyError`: `invalid-parameter` is `Validation`,
   `catalogue-unavailable` is `ServiceUnavailable` ("retry in N s" when the hub sent `Retry-After`),
   `catalogue-bad-answer` is `SourceUnreachable`, an unknown type is `HubError`. A station that is not in the
   catalogue is `NotFound` ("station not found: <uuid>").
4. For a radio `*hub.APIError` with a `Parameter`, the message gets ` (parameter <name>)` appended, so the
   agent knows which input to correct (spec FR-009). This follows how `classify` already appends `Reason`.

**Rationale**: the shared client already holds the mapping, so this layer only names the category and adds the
parameter (Principle I and III). A diagnosed "not offered" answer must come first because its `Cause` may be
the inventory's own timeout, which would otherwise hide that radio is unavailable (the TTS comment in
`errors.go` says the same).

**Alternatives considered**: reusing `TtsUnavailable` for both (wrong text for an agent), or a general
`ExtensionUnavailable` (renames an existing category, which is a breaking change under Principle II).

## R3: The diagnosis helper and the time bound

**Decision**: `radioError(ctx, client, hubURL, err)` mirrors `ttsError`: for a `*hub.RadioNotOfferedError` it
returns `hub.DiagnoseRadioUnavailable(ctx, client, hubURL)`, otherwise `err`. It runs once, only after that
failure. Each radio handler wraps its context with the radio client's timeout (`withTimeout`), so the call and
the diagnosis share one bound; the diagnosis request itself uses the standard 5 s client.

**Rationale**: `hub.RadioTimeout` is 10 s, longer than the hub's own 5 s catalogue budget, so the hub's
"catalogue unavailable" answer arrives before the client gives up (spec FR-010). The existing `withTimeout`
already takes the client whose timeout applies.

**Alternatives considered**: using the standard 5 s client, which would race the hub's own 5 s budget and
report a client `Timeout` where the hub would have said "catalogue unavailable, retry in N s".

## R4: Result shapes

**Decision**:
- `searchRadioStations` returns `hub.RadioStationPage` as is (`stations`, `nextOffset`).
- `listRadioTags` returns `hub.RadioTagPage` as is (`tags`, `nextOffset`).
- `getRadioStation` returns `hub.RadioStation` as is.
- `listRadioCountries` and `listRadioLanguages` wrap the hub's bare arrays: `{"countries": [...]}` and
  `{"languages": [...]}` (a bare array is not valid structured content; see AGENTS.md).

**Rationale**: the hub types already carry every field with `json` tags, `null` for an unknown optional value,
and `nextOffset` as null on the last page, which is the spec's wording. Wrapping adds only the envelope.

**Conformance impact**: `listEnvelopes` in `conformance_test.go` gains `countries` and `languages`, so the
test looks inside the wrapper at the item schema. The page results are objects in the spec, so they need no
unwrapping.

## R5: Input constraints

**Decision**: set every constraint the hub spec declares for a parameter, at registration:

| Tool input | Constraint (from `api.Spec`) |
|---|---|
| `name`, `tag`, `language` (search); `name` (tags) | maxLength 100 |
| `countryCode` | pattern `^[A-Za-z]{2}$` |
| `codec` | maxLength 32 |
| `bitrateMin` | 0 to 1000000 |
| `order` | enum `name`, `votes`, `clickCount`, `bitrate` |
| `direction` | enum `asc`, `desc` |
| `limit` | 1 to 100 |
| `offset` | 0 to 10000 |
| `uuid` (path) | minLength 1 (the path-ID rule in AGENTS.md); the hub checks the UUID format and answers 400 |

**Rationale**: spec FR-007 asks for a tool error and no hub request on a schema failure. A new `withMaxLength`
helper joins the existing ones in `schema.go`, and `maxLength` is added to the conformance test's
`constraintKeys` so the tool and the spec are compared on it. No existing tool's spec parameter has a
`maxLength`, so the existing tools are unaffected.

**Alternatives considered**: leaving `maxLength` to the hub (it answers 400). An agent would then pay a request
for what the schema can say, and the spec's constraint would be unverified here.

## R6: Untrusted text and UTF-8

**Decision**: do nothing to station text in code (clarification, 2026-11-22). Go's JSON encoding already
replaces any invalid UTF-8 with U+FFFD, and the shared client decodes the hub's JSON strings, so the result is
always valid UTF-8. The guard is in the descriptions of `searchRadioStations` and `getRadioStation` (and the
list tools): the text fields are untrusted community text, to be treated as data and never as instructions.

**Testable parts**:
- A fake hub station whose name holds an escape sequence and "ignore previous instructions" comes back
  byte-for-byte in `structuredContent`.
- Error messages never contain station text: the only text from the catalogue that reaches a message is the
  UUID the agent sent (`station not found: <uuid>`) and the hub's own problem detail.
- A test asserts the word "untrusted" in the four station-text tools' descriptions, so the warning cannot be
  dropped unnoticed.

**Alternatives considered**: stripping control characters and capping length (options B and C of the
clarification). Rejected there.

## R7: How the tools are registered

**Decision**: one new file, `internal/tools/radio.go`, like `tts.go`. `Register` gains a `radioClient`
parameter (`Register(s, client, speakClient, radioClient, hubURL)`); `main` passes
`hub.NewClientWithTimeout(hub.RadioTimeout)`. `registerRadio` returns 5. The test harness gets a variant of
`newTestSessionClients` that sets the radio client; the existing helper keeps passing one client for all.

**Rationale**: the same pattern 002 used for `speakClient`. A struct of clients would be an abstraction for a
second use only.

## R8: `/health` and other callers

**Decision**: no change. `/health` and the hub check use the standard client. Only `newApp` and the tests call
`Register`.

## R9: Test cleanups that come with the change

- `tts_unavailable_test.go` names the deprecated `hub.TTSDiagnosis`, which the backlog says will be removed.
  It becomes `hub.ExtensionDiagnosis` in this change; it is the one place in the repository.
- The radio and TTS diagnosis tests share the inventory fixtures (`inventoryJSON`, `ttsDisabledInventoryJSON`).
  The radio test needs a disabled-radio fixture; it uses `inventoryJSON` with a radio extension entry.

## R10: Release and documentation

- New tools only, so the next release is MINOR (v1.4.0 after v1.3.0). `deploy/pi/install.sh`'s `RELEASE_TAG`
  is bumped in the commit that gets tagged, as AGENTS.md says. Tagging is the user's decision.
- README (tool list, count 33 → 38, the new error category), AGENTS.md (tool count, Recent Changes, the sonora-cli
  note) and the contracts doc are updated in the change. The master `spec.md`, `plan.md` and `changelog.md` in
  `.specify/memory/` are updated when the feature is archived, as for earlier features.
- `docs/backlog/adopt-radio-search.md` is deleted and its row removed from `INDEX.md` when the feature ships.

## R11: Live check

The real hub at `multiroom.lan` has the radio extension (RADIO-002 was verified there on 2026-11-15, per
CLI-014). The read-only tools may be run against it without asking; `playback` of a found station is audible and
needs the user's permission and the rules in AGENTS.md (`office` output, low volume, stop the route first).
