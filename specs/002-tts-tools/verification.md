# Verification: Text-to-speech tools

Branch `002-tts-tools`, 2026-10-28, Windows 10 / Git Bash, Go 1.27.0, `GOWORK=off` (no `go.work`).

## T023: offline gate

| Command | Result |
|---|---|
| `gofmt -l .` | prints nothing |
| `go vet ./...` | exit 0 |
| `go build ./...` | exit 0 |
| `go test -count=1 ./...` | `cmd/sonora-mcp`, `internal/config`, `internal/server`, `internal/tools` ok; 0 failures |
| `docker run … golang:1.27 go test -race -count=1 ./...` (command from `AGENTS.md`) | exit 0, same four packages ok |

The race run started before the last edit, a sentence added to the `listTtsVoices` description and its
test; the plain `go test` was re-run after it and passes.

- Tools registered: 32 (`TestToolInventory`, `TestRegisterCount`, `cmd/sonora-mcp` count).
- `TestConformance` passes for `speak`, `listTtsVoices`, `getTtsCacheStats` and `clearTtsCache`,
  inputs and outputs (`api.Spec` of hub API 0.1.29).
- sonora-cli v0.3.4 pinned in `go.mod`.
- `git ls-files --eol`: no changed or new file has `w/crlf`.

## T024: read-only checks against `http://multiroom.lan:8080`

Built binary run locally on `127.0.0.1:3055`, called over `POST /mcp`; stopped afterwards. `/health`
reported the hub reachable.

| Call | Result |
|---|---|
| `getTtsCacheStats` | `{"entriesByProvider":{"gemini":9,"google":8,"piper-local":10},"maxSizeBytes":524288000,"totalEntries":27,"totalSizeBytes":22959788}`, camelCase names |
| `listTtsVoices` `nope` | `NotFound: PROVIDER_NOT_FOUND: No provider configured with name 'nope'` |
| `listTtsVoices` `google`, `uk-UA` | voices with engine, full name, gender, language and short name |
| `listTtsVoices` `gemini`, `uk-UA` | voices with `language: null` (the provider does not say) |
| `listTtsVoices` `piper-local` | `Validation: INVALID_REQUEST: Provider 'piper-local' of type PIPER does not support voice listing` |

**Finding:** the quickstart expected `piper-local` to list voices. The real hub answers that PIPER
providers cannot, so `quickstart.md` §2 is out of date and the `listTtsVoices` description now says
that some providers, such as `piper-local`, cannot list voices and answer Validation.

## T025: audible check (user approved, output `office`)

Recorded before: `office` volume 14, not muted, enabled, available; master mute off; no routes.
sonora-mcp built from this branch and run on `127.0.0.1:3055`; stopped afterwards.

| Step | Result |
|---|---|
| `speak` "Test", `piper-local`, `routeVolume: 15`, first call (not cached) | `Timeout: the hub did not respond in time` after 15.001 s. The hub had accepted it and the announcement played and completed (hub log: `TTS_REQUEST_RECEIVED`, `TTS_PLAYBACK_STARTED`, `TTS_PLAYBACK_COMPLETED`). |
| Direct `POST /api/tts/speak` "Test two" (not cached), `curl -w` | 202 after 14.5 s (`starttransfer=14.498 s`): the hub answers an uncached `piper-local` announcement only after synthesis. |
| `speak` "Test two" through the tool (cached) | `{"announcementId":"44247c7f-…","cacheHit":true,"queueDepth":1}` in 41 ms |

After: `office` volume 14, not muted, enabled; master mute off; `GET /api/v2/routes` is `[]`;
`audio.output.routes{output:office}` is 0.0; the ALSA check
(`grep -h "^state" /proc/asound/card*/pcm*p/sub*/status`) printed no line. Nothing was left open or
changed. The announcement cache gained two entries ("Test", "Test two" for `piper-local`); the cache
was not cleared.

**Finding:** uncached synthesis with `piper-local` on the Pi takes about 14–15 s (its configured
`timeout-seconds` is 30), which is the same as `speak`'s 15 s bound (`hub.SpeakTimeout`). A first,
uncached announcement can therefore time out in the tool although the hub speaks it; a repeat is
served from the cache. The spec's assumption that the shared client's bound is used as is still
holds, but the bound is too tight for this provider. Raising `SpeakTimeout` is a sonora-cli change
(Principle I); it is not done here.
