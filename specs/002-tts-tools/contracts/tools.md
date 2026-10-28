# Contract: TTS tools

Four tools added to the 28 of the master plan (28 → 32). Additions only, so the release is MINOR
(Principle II). Conventions as in the master plan: `?` = optional, `TargetType` = `SINGLE_OUTPUT` |
`OUTPUT_GROUP`, `Volume` = integer 0–100. Input and result fields: [data-model.md](../data-model.md).

| Tool | Hub operation | Inputs | Kind | Bound | Structured result |
|---|---|---|---|---|---|
| `speak` | `POST /api/tts/speak` | `text` (minLength 1), `targetName` (minLength 1), `targetType: TargetType`, `providerName?`, `voice?`, `language?`, `playbackMode?`, `routeVolume?: Volume`, `duckingLevelDb?` (−60 to 0) | state-changing | 15 s | `SpeakAccepted` (`announcementId`, `cacheHit`, `queueDepth`) |
| `listTtsVoices` | `GET /api/tts/providers/{providerName}/voices` | `providerName` (minLength 1), `language?`, `engine?` | read-only | 5 s | `TTSVoiceList` (`providerName`, `voices`) |
| `getTtsCacheStats` | `GET /api/tts/cache/stats` | — | read-only | 5 s | `TTSCacheStats` (`totalEntries`, `totalSizeBytes`, `maxSizeBytes`, `entriesByProvider`) |
| `clearTtsCache` | `DELETE /api/tts/cache` | `providerName?` | destructive, idempotent | 5 s | `{"cleared": true, "providerName"?}` |

The bound includes the diagnosis after a "not offered" answer.

## Descriptions must say

- **`speak`**: it returns once the announcement is queued, not once it has been spoken; each call
  queues another announcement. `targetName` takes the same IDs as `targetId` elsewhere, not display
  names. Omitting `providerName` uses the hub's configured default, which may be a billed cloud
  service. `playbackMode` is `duck-others` (other audio keeps playing quieter underneath) or `mix`
  (it plays on unchanged); `duckingLevelDb` applies only with `duck-others`. `routeVolume` sets this
  announcement's volume only and changes no output's volume. Failures: NotFound (unknown target or
  provider), Validation, ServiceUnavailable (provider down, slow or rate-limited), TtsUnavailable.
- **`listTtsVoices`**: pass a voice's `shortName` to `speak`'s `voice`; an empty list is not an
  error; NotFound for an unknown provider.
- **`getTtsCacheStats`**: sizes are in bytes.
- **`clearTtsCache`**: clears every provider's cached audio, or one provider's; the next
  announcement of a cleared phrase is synthesised again (a billed provider is charged again).

## Error categories

One category is added to the master plan's table:

| Category | When |
|---|---|
| `TtsUnavailable` | The hub does not serve the TTS operation; the message names why (not installed, disabled, failed to load with the hub's reason, inactive, API version mismatch), or gives no reason when the hub's extension list could not be read |

If the address turns out not to serve the hub API at all, the error is `Network`. Every other TTS
failure uses the existing categories: `NotFound` (`TARGET_NOT_FOUND`, `PROVIDER_NOT_FOUND`),
`Validation` (`INVALID_REQUEST`, other 400s), `ServiceUnavailable` (provider failures, 503),
`Timeout`, `MalformedResponse`, `HubError`.
