# TTS cache statistics and clearing

- **Status:** Proposed, not started. Written 2026-10-26 from a comparison of sonora-cli's `hub` package
  with the tools.

- **Origin:** `GET /api/tts/cache/stats` and `DELETE /api/tts/cache` in `api.Spec` 0.1.29 (TTS
  extension); sonora-cli `sonora tts cache`.

- **Depends on:** nothing; sonora-cli v0.3.1 already has `hub.GetTTSCacheStats` and
  `hub.ClearTTSCache`. It shares the `TtsUnavailable` error category and the fake hub's `/api/tts`
  endpoints with [Speak announcements](tts-speak.md); whichever ships first adds them. Speak first is
  the natural order, since an agent rarely needs the cache before it can speak.

- **Effort:** Small (two new tools).

## Problem

The TTS extension caches synthesised audio on the Pi. An agent cannot see how large the cache is or
clear it, for example after a voice change, when old phrases would keep playing in the old voice.

## Proposal

- **`getTtsCacheStats`** (read-only, no arguments) wrapping `hub.GetTTSCacheStats`: returns
  `totalEntries`, `totalSizeBytes`, `maxSizeBytes` and `entriesByProvider`.
- **`clearTtsCache`** (destructive, idempotent; optional `providerName`) wrapping `hub.ClearTTSCache`:
  clears the whole cache, or one provider's entries. It returns `{"cleared": true}`, plus
  `providerName` when one was given, following the no-body result convention.

## Open questions

- Is this an agent's job at all? It is maintenance the user would rarely ask an assistant for. Keep it
  at P4 until someone does.
