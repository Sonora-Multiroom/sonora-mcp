# Adopt RADIO-002: search and browse radio stations

- **Status:** Proposed, not started. Written 2026-11-21.

- **Origin:** CLI-014 (sonora-cli, radio station search and browse). The hub's radio extension 0.2.0
  (RADIO-002) serves a station catalogue, and sonora-cli adds it to the `hub` package and as the
  `sonora search radio-stations` command.

- **Depends on:** a sonora-cli release that contains CLI-014 (a minor tag after this repository's
  current v0.3.4), and the hub's radio extension 0.2.0 or later on the Pi.

- **Effort:** Small–Medium (five read-only tools and one error mapping).

## Problem

An assistant can play a `radiobrowser://<uuid>` input but cannot find a station's UUID, so it cannot
answer "play Kiss FM" without the user looking the station up by hand.

## Proposal

- Bump the client: `go get github.com/Sonora-Multiroom/sonora-cli@vX.Y.Z`.
- Add read-only tools over `hub.SearchStations`, `hub.Station`, `hub.Countries`, `hub.Languages` and
  `hub.Tags`, using `hub.NewClientWithTimeout(hub.RadioTimeout)` (the hub's catalogue budget is 5 s, so the
  standard 5 s client would race it). Optional filters are pointers sent only when set. Page tools return
  `nextOffset` so an assistant can ask for the next page.
- Map the new errors onto the categories `internal/tools/errors.go` already uses for TTS:
  `hub.ClassRadioUnavailable` and the `*hub.RadioUnavailableError` reasons (the radio extension is not
  installed, disabled, failed to load, inactive, or too old), with `hub.DiagnoseRadioUnavailable` run once
  after a `*hub.RadioNotOfferedError`. A radio `*hub.APIError` carries `Type`, `Parameter` and `RetryAfter`:
  pass the retry delay on for a 503.
- Station names, tags and addresses are community text. Return them as data and never as instructions to
  the assistant.
- Use `hub.ExtensionDiagnosis` wherever a diagnosis type is named, never `hub.TTSDiagnosis`, which is
  deprecated and will be removed. On 2026-11-21 this repository names only the `Diagnosis*` constants and
  the `Diagnosis` field, so for existing code this is a check, and the rule applies to new code.

## Open questions

- One `searchRadioStations` tool with every filter, or a separate tool for the catalogue lists
  (countries, languages, tags)? An assistant needs the lists to fill `tag` and `language`, which match a
  whole value.
- Should `playback` accept a station UUID directly, or should the assistant build
  `radiobrowser://<uuid>` itself from the search result's `uri`?
