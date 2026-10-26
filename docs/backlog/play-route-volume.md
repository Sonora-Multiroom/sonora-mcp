# Route volume and ducking level

- **Status:** Proposed, not started. Written 2026-10-24, updated 2026-10-25 now that sonora-cli has
  adopted hub 034.

- **Origin:** hub `specs/034-per-route-volume-ducking` (API 0.1.29); sonora-cli PR #28, released in
  `v0.3.0`.

- **Depends on:** sonora-cli v0.3.1, shipped
  ([specs/tiny/adopt-sonora-cli-0.3.md](../../specs/tiny/adopt-sonora-cli-0.3.md)) and, for the
  ducking level, [join-modes.md](join-modes.md).

- **Effort:** Small.

## Problem

The play tool can set a target's volume, which changes the room for good. An assistant that plays an
alert or an announcement has no way to give that playback its own volume, nor to change it once it
plays. A ducking playback also lowers the music by the hub's one system-wide level (−15 dB in
production), which 023's listening test found too little for speech over dense music.

## Proposal

- **`routeVolume`** (0 to 100, optional) on `playback` and `createRoute`, sent as
  `hub.PlaybackRequest.RouteVolume` / `hub.CreateRouteRequest.RouteVolume`. Describe it as "the volume
  of this playback only; the room's volume is not changed and its volume control does not reach this
  playback; use it for announcements and alerts". Mute, master mute and each output's gain still apply.
- `playback`'s existing `volume` is described as setting the room's volume, which stays after playback
  ends, and is sent as `OutputVolume` (done in the sonora-cli bump).
- **`duckingLevelDb`** (−60.0 to 0.0, optional) on `playback` and `createRoute`: how far a `DUCK_OTHERS`
  playback lowers the others. The hub answers 400 when the effective join mode is not `DUCK_OTHERS`;
  leave that check to the hub (Principle III) and say so in the description.
- **`defaultRouteVolume`** and **`defaultDuckingLevelDb`** (optional) on `createInput`, sent as the
  `hub.CreateInputRequest` fields of the same names.
- **New idempotent tool `setRouteVolume`** (`routeId`, `volume` 0 to 100), wrapping `hub.SetRouteVolume`
  (`PUT /api/v2/routes/{routeId}/volume`): changes a playing route's own volume, also on a route started
  without one, and no output's volume. The description must say a route volume cannot be cleared once
  set, only changed. Returns the route; an unknown route is NotFound.

Every new field is a pointer omitted when nil, so the hub's defaults apply (the input's default, else
the outputs' volume or the configured ducking level).

## Open questions

- `duckingLevelDb` is a float with a decimal range. Check that `withRange` and the conformance test
  handle `number` bounds, not only integers, before relying on them.
- The hub's TTS `speak` request gained `routeVolume` and `duckingLevelDb` too, but sonora-mcp has no TTS
  tool and `hub.SpeakRequest` lags that schema. Out of scope here.
