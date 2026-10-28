# Data model: Text-to-speech tools

The server holds no state (Principle III). Every type below is either a tool input struct defined
here or a `hub` type returned as is; nothing is stored between calls.

## Tool inputs (defined in `internal/tools/tts.go`)

### `speakIn`

| Field | JSON | Type | Required | Constraint (set at registration) | Sent as |
|---|---|---|---|---|---|
| Text | `text` | string | yes | `minLength: 1` | `SpeakRequest.Text` |
| TargetName | `targetName` | string | yes | `minLength: 1` | `SpeakRequest.TargetName` |
| TargetType | `targetType` | string | yes | enum `SINGLE_OUTPUT`, `OUTPUT_GROUP` | `SpeakRequest.TargetType` |
| ProviderName | `providerName` | *string | no | — | `SpeakRequest.ProviderName` |
| Voice | `voice` | *string | no | — | `SpeakRequest.Voice` |
| Language | `language` | *string | no | — | `SpeakRequest.Language` |
| PlaybackMode | `playbackMode` | *string | no | none (the spec does not constrain it; the hub rejects other values) | `SpeakRequest.PlaybackMode` |
| RouteVolume | `routeVolume` | *int | no | 0–100 | `SpeakRequest.RouteVolume` |
| DuckingLevelDb | `duckingLevelDb` | *float64 | no | −60 to 0 | `SpeakRequest.DuckingLevelDb` |

Optional fields are pointers with `omitempty`; a nil field is not sent, so the hub's configured
default applies (FR-002).

### `listVoicesIn`

| Field | JSON | Type | Required | Constraint | Sent as |
|---|---|---|---|---|---|
| ProviderName | `providerName` | string | yes | `minLength: 1` (path parameter) | path |
| Language | `language` | *string | no | — | query, only when set |
| Engine | `engine` | *string | no | — | query, only when set |

### `clearCacheIn`

| Field | JSON | Type | Required | Sent as |
|---|---|---|---|---|
| ProviderName | `providerName` | *string | no | `providerName` query, only when set |

`getTtsCacheStats` takes `noInput`.

## Tool results

| Tool | Type | Fields |
|---|---|---|
| `speak` | `*hub.SpeakAccepted` | `announcementId`, `cacheHit`, `queueDepth` |
| `listTtsVoices` | `*hub.TTSVoiceList` | `providerName`, `voices[]` of `shortName`, `fullName`, `engine`, `language`, `gender` (each may be `null`) |
| `getTtsCacheStats` | `*hub.TTSCacheStats` (tagged since sonora-cli v0.3.4) | `totalEntries`, `totalSizeBytes`, `maxSizeBytes`, `entriesByProvider` |
| `clearTtsCache` | `cacheCleared` (local confirmation, the hub answers 204) | `cleared: true`, `providerName` when one was given |

## Error category `TtsUnavailable`

New constant `CategoryTtsUnavailable = "TtsUnavailable"` in `internal/tools/errors.go`.

| Source error | Category | Message |
|---|---|---|
| `*hub.TTSUnavailableError`, any diagnosis but `DiagnosisHubAddress` | `TtsUnavailable` | its `Error()`, e.g. "text-to-speech is not available on this hub: the TTS extension is installed but disabled in the hub's configuration" |
| `*hub.TTSUnavailableError` with `DiagnosisHubAddress` | `Network` | its `Error()`, without the usual `: <err>` suffix |
| `*hub.TTSNotOfferedError` that was not diagnosed (`ClassTTSUnavailable`) | `TtsUnavailable` | "text-to-speech is not available on this hub" |

The `TTSUnavailableError` check runs before the decode and timeout checks, so a failed inventory
call stays `TtsUnavailable` (research R4).

Diagnosis outcomes, all produced by `hub.DiagnoseTTSUnavailable` and passed through unchanged:
not installed (with whether extension loading is off), disabled, failed to load (with the hub's
reason), inactive, API version mismatch, unknown (inventory call failed), not a hub.
