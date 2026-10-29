# Quickstart: validating the TTS tools

## Prerequisites

- `go.mod` pins sonora-cli v0.3.4 (research R1). No `go.work`, or run with `GOWORK=off`.

## 1. Offline gate (no hub needed)

```sh
gofmt -l .           # prints nothing
go vet ./...
go test ./...
```

Expected:

- `TestToolInventory` lists 32 tools, including `speak`, `listTtsVoices`, `getTtsCacheStats` and
  `clearTtsCache`.
- `TestConformance` passes for the four new tools, including their output schemas (TTS responses are
  `*/*` in the spec, research R7).
- The TTS tests against the fake hub cover:
  - a 404 on a TTS path followed by each inventory state, giving `TtsUnavailable: …` with the reason;
  - the inventory itself answering 404, giving `Network: … is not serving the Multiroom Audio Hub API …`;
  - the inventory failing otherwise, giving `TtsUnavailable: text-to-speech is not available on this hub`.

`go test -race` runs on Linux (CI, or the Docker command in `AGENTS.md`).

## 2. Read-only checks against the real hub

These are allowed without asking. Run the server locally against production:

```sh
go run ./cmd/sonora-mcp --multiroom-url http://multiroom.lan:8080
npx @modelcontextprotocol/inspector   # connect to http://localhost:3001/mcp
```

- `listTtsVoices` with `providerName: google` and `language: uk-UA` lists voices; `piper-local` cannot list voices and answers `Validation: INVALID_REQUEST: … does not support voice listing`.
- `listTtsVoices` with `providerName: nope` gives `NotFound: PROVIDER_NOT_FOUND: …`.
- `getTtsCacheStats` gives the four figures, with `totalEntries` and the other names in camelCase.

## 3. Audible check (ask the user first)

Follow the production rules in `AGENTS.md`: ask, record the target output's volume, mute, enabled
state and the master mute, and keep the volume low.

- Call `speak` with `text: "Test"`, `targetName: <output>`, `targetType: SINGLE_OUTPUT`,
  `providerName: piper-local` and `routeVolume: 15`. It answers with an `announcementId`, and the
  announcement is heard.
- Afterwards:
  - the ALSA check (`grep -h "^state" /proc/asound/card*/pcm*p/sub*/status` over SSH) prints nothing;
  - `/actuator/metrics/audio.output.routes?tag=output:<output>` reads 0;
  - the recorded state is unchanged (`routeVolume` touches no output volume).

`clearTtsCache` is not run on production unless the user asks: it drops cached audio, and a billed
provider would be charged again to rebuild it.
