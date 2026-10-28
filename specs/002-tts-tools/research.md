# Research: Text-to-speech tools

Facts checked on 2026-10-28 against sonora-cli v0.3.3, then v0.3.4 for R2 (`hub`, `api.Spec` of hub API 0.1.29) and the
`sonora-tts` sources.

## R1. Shared hub client version

- **Decision**: pin sonora-cli **v0.3.4** (released 2026-10-28, #32), the patch that adds `json`
  tags to `hub.TTSCacheStats` (see R2).
- **Rationale**: v0.3.3 has every TTS operation (`Speak`, `ListTTSVoices`, `GetTTSCacheStats`,
  `ClearTTSCache`), `SpeakRequest`'s playback fields, `SpeakTimeout`, and the "not offered" diagnosis
  (`DiagnoseTTSUnavailable`, `ClassifyError` → `ClassTTSUnavailable`). Its `api.Spec` is still 0.1.29,
  so the existing tools are unaffected.
- **Alternatives**: stay on v0.3.3 and wrap the cache stats in a local type. Rejected: a local type
  for a hub resource breaks Principle I, and the bug affects every JSON consumer of `hub`.

## R2. `hub.TTSCacheStats` has no `json` tags

- **Finding**: `TTSCacheStats` has four exported fields without tags (`TotalEntries`,
  `TotalSizeBytes`, `MaxSizeBytes`, `EntriesByProvider`); only its private wire type is tagged. The
  `sonora` CLI never notices, because it renders the stats through its own view type
  (`internal/render/tts.go`). Returned from a tool, it would serialise and be described as
  `TotalEntries` and so on, and the conformance test would report `totalEntries` missing.
- **Decision**: fixed in sonora-cli (tags matching `#/components/schemas/CacheStats`, #32, v0.3.4),
  pinned here. The CLI's view type stays; CLI output is unchanged.
- **Alternatives**: none acceptable here (see R1).

## R3. Diagnosing "not offered"

- **Decision**: every TTS handler passes the hub call's error through one helper. If the error is
  `*hub.TTSNotOfferedError`, the helper returns `hub.DiagnoseTTSUnavailable(ctx, client, hubURL)`,
  using the ordinary 5 s client; any other error goes through unchanged. Each TTS handler runs under
  `context.WithTimeout(ctx, <its client's Timeout>)`, so the hub call and the diagnosis together stay
  within the tool's bound (spec: the diagnosis runs within the same bound).
- **Rationale**: the diagnosis is a single read of the hub's own report and runs only after a
  failure (Principle III allows it; the spec's Assumptions record why). The mapping lives in `hub`,
  so this repo interprets nothing.
- **Alternatives**: rely only on `ClassifyError`'s new `ClassTTSUnavailable` for an undiagnosed
  error. Kept as the fallback, but the agent would get no reason.

## R4. Classifying the diagnosed error

- **Finding**: `TTSUnavailableError` unwraps to its `Cause`. When the inventory call itself failed,
  `Cause` may be a timeout or a `*hub.DecodeError`, and `classify` checks those first, so the error
  would come out as `Timeout` or `MalformedResponse`, not `TtsUnavailable`. For
  `DiagnosisHubAddress`, `ClassifyError` returns `ClassNetwork`, and `classify` appends `: <err>` to
  Network messages, which would repeat the message.
- **Decision**: in `classify`, check `*hub.TTSUnavailableError` before every other branch.
  `DiagnosisHubAddress` → `Network` with the message as is; anything else → `TtsUnavailable` with
  `Error()`. Map `hub.ClassTTSUnavailable` to `TtsUnavailable` as well, for an undiagnosed "not
  offered".
- **Alternatives**: unwrap-aware ordering inside `hub.ClassifyError` only. Not enough: the
  decode/timeout checks that win are this repo's own.

## R5. TTS error codes

- **Finding** (`sonora-tts` `TtsExceptionHandler`, `TtsCacheController`, `ProviderRegistry`): an
  unknown target answers 400 `TARGET_NOT_FOUND`; an unknown provider, for `speak`, voices and cache
  clearing alike, 400 `PROVIDER_NOT_FOUND`; a bad playback mode, a ducking level with `mix` or a
  malformed filter, 400 `INVALID_REQUEST`; provider failures, 503 with `PROVIDER_*` codes.
  `hub.ClassifyError` maps these to TargetNotFound, NotFound, Validation and ServiceUnavailable, and
  `classify` already maps those classes to `NotFound`, `Validation` and `ServiceUnavailable`.
- **Decision**: no new mapping beyond R4. Correct the spec's edge case: an unknown provider for
  clearing the cache is **NotFound**, not Validation.

## R6. The announcement time bound

- **Decision**: `Register` gets a second client, `speakClient`, built in `main` with
  `hub.NewClientWithTimeout(hub.SpeakTimeout)` (15 s); `speak` uses it, and the other TTS tools use
  the shared client (5 s). The test harness passes the same client for both, so timeout tests stay
  fast.
- **Rationale**: Principle IV names `NewClientWithTimeout` for TTS; passing the client in keeps tests
  in control of the bound, as they are for every other tool.
- **Alternatives**: a `context.WithTimeout(15 s)` over the 5 s client. Rejected: the client's own
  5 s bound would still cut `speak` short.

## R7. Conformance for TTS operations

- **Finding**: TTS responses are published as `*/*`, not `application/json`. `successSchema` reads
  only `application/json`, so it would treat every TTS response as a 204 and skip the output check.
- **Decision**: `successSchema` accepts `application/json`, then `*/*`. The input checks already
  cover TTS: `text` and `targetName` `minLength: 1`, `targetType` enum, `routeVolume` 0–100,
  `duckingLevelDb` −60 to 0, `providerName` as a path parameter (`minLength: 1`), and optional
  query filters. `contractTools` grows by the four new tools.

## R8. Result shapes

- **Decision**: `speak` returns `*hub.SpeakAccepted`; `listTtsVoices` returns `*hub.TTSVoiceList`
  (already an object, `voices` never nil); `getTtsCacheStats` returns `*hub.TTSCacheStats` (after R2).
  `clearTtsCache` answers 204, so it returns a confirmation like `deleteInput`'s:
  `{"cleared": true, "providerName"?}`.
- **Note**: `TTSVoice`'s fields are pointers without `omitempty`, so a missing field comes back as
  `null`, which is how `hub` keeps "absent" apart from "empty". The output schema allows it.

## R9. Inputs left out

- `engine`, `pitch`, `speakingRate` and `stylePrompt` on `speak` (spec: not exposed in this
  feature). The conformance test only checks that the tool's inputs exist in the spec and that
  required ones are present, so leaving optional fields out passes.
