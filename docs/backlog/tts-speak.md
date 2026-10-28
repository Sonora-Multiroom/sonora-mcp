# Speak announcements through the TTS extension

- **Status:** Proposed, not started. Written 2026-10-26 from a comparison of sonora-cli's `hub` package
  with the tools; listed as out of scope (AS-007) in the master spec.

- **Origin:** the out-of-tree `sonora-tts` hub extension (`Extension-Id: tts`), `POST /api/tts/speak`
  and `GET /api/tts/providers/{providerName}/voices` in `api.Spec` 0.1.29; sonora-cli `sonora speak`
  and `sonora tts voices`.

- **Depends on:** nothing for a first version: sonora-cli v0.3.1 already has `hub.Speak` (text,
  target, provider, voice, language) and `hub.ListTTSVoices`. Only `playbackMode`, `routeVolume` and
  `duckingLevelDb` need a sonora-cli release whose `hub.SpeakRequest` carries them (sonora-cli backlog
  `speak-request-fields.md`; see below); without
  them every announcement uses the extension's configured playback mode and the room's volume.

- **Effort:** Small to medium here (two new tools, one new error category); plus a small sonora-cli
  change for the missing fields.

## Problem

An assistant can play a URI, but it cannot say something. Text announcements ("dinner is ready",
"the washing machine has finished") are the reason many users connect an assistant to a home audio
system, and the hub already does it through the TTS extension. No tool exposes it.

The client is also behind the spec: `SpeakRequest` in `api.Spec` has 13 properties, and
`hub.SpeakRequest` sends only `text`, `targetName`, `targetType`, `providerName`, `voice` and
`language`. It cannot send `playbackMode` (`duck-others` or `mix`), `routeVolume` or `duckingLevelDb`,
so an announcement cannot be told to duck the music or play at its own volume. It also lacks `engine`,
`pitch`, `speakingRate` and `stylePrompt`.

## Proposal

1. **sonora-cli (for the playback fields):** add the missing optional fields to `hub.SpeakRequest` as
   pointers with `omitempty`, release a patch tag, and bump it here. That change is sonora-cli's work,
   not this repository's. The tools below can ship before it, and gain the fields with the bump.
2. **New tool `speak`** wrapping `hub.Speak`: `text`, `targetName`, `targetType` (`SINGLE_OUTPUT` |
   `OUTPUT_GROUP`) required. `providerName`, `voice`, `language`, `playbackMode`, `routeVolume` and
   `duckingLevelDb` are optional, with constraints taken from the spec. It returns the hub's 202 body
   (`announcementId`, `cacheHit`, `queueDepth`). The description says the call returns once the
   announcement is queued, not once it has been spoken, and that `duck-others` keeps the music playing
   underneath.
3. **New read-only tool `listTtsVoices`** (`providerName`, optional `language`, `engine`) wrapping
   `hub.ListTTSVoices`, so an agent can pick a valid `voice` instead of guessing.
4. **Error category:** map `hub.ClassTTSUnavailable` (`*TTSUnavailableError`: the extension is not
   loaded or its provider is down) to a new `TtsUnavailable` prefix; today it falls through to the
   generic category in `internal/tools/errors.go`. `*TTSError` needs nothing new: `hub.ClassifyError`
   already maps its codes to TargetNotFound, NotFound, Validation and ServiceUnavailable.

The default provider may be a billed cloud service. Leave the choice to the hub's configuration
(Principle III) and say in the description that omitting `providerName` uses the configured default.
Do not hard-code `piper-local`.

## Open questions

- `targetName` here, `targetId` everywhere else: keep the hub's field name (the conformance test
  compares names with the spec), and explain in the description that it takes the same IDs as
  `targetId`.
- Should the expressive fields (`engine`, `pitch`, `speakingRate`, `stylePrompt`) be exposed at all?
  They depend on the provider, and an agent guessing them gets 400s. Recommendation: expose
  `playbackMode`, `routeVolume` and `duckingLevelDb` now and add the rest when someone needs them.
- The hub's 5 s client timeout bounds the call. Speak answers 202 after queueing, so it fits, but a
  cold cloud provider might not. Check against production with `piper-local` before choosing a longer
  timeout for this tool alone.
