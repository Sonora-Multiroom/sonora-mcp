# Quickstart: validating the radio tools

## 1. Offline gate (no hub, no network)

```bash
GOWORK=off gofmt -l .        # prints nothing
GOWORK=off go vet ./...
GOWORK=off go test ./...
```

Expected: all pass. The new tests cover the five tools through the in-memory MCP client against the fake hub;
the conformance test checks the five tools' inputs and outputs against `api.Spec` 0.1.35; the inventory test
checks `TestToolInventory` lists 38 tools. Run `go test -race ./...` on Linux (CI does) or through the Docker
command in AGENTS.md.

Look for these cases in the output of `go test ./internal/tools -run Radio -v`:

- search sends only the filters the agent set, and none when it set none
- next offset present on a page, absent on the last page, and an empty page is a result
- every schema violation (bad order, `countryCode` of three letters, `limit` 0, negative `offset`) is a tool
  error and the fake hub saw no request
- each radio-unavailable diagnosis, an unreadable inventory (generic message), a wrong hub address (`Network`)
- one inventory request after a "not offered" answer, and none after success or another error
- 503 with `Retry-After` gives `ServiceUnavailable` with the delay; 400 gives `Validation` with the parameter
- station text with an escape sequence and "ignore previous instructions" returns unchanged
- a slow catalogue answer (longer than 5 s, shorter than 10 s) still arrives, because the radio client allows
  10 s

## 2. Read-only check on the real hub (allowed without asking)

Start the server against the hub and call the tools with the MCP inspector:

```bash
go run ./cmd/sonora-mcp --multiroom-url http://multiroom.lan:8080 --port 3001
npx @modelcontextprotocol/inspector     # connect to http://localhost:3001/mcp
```

1. `searchRadioStations` with `name: "kiss"`: stations whose names contain it, each with a `uri` of the form
   `radiobrowser://<uuid>`.
2. `getRadioStation` with a `uuid` from step 1: the same station.
3. `listRadioCountries`, `listRadioLanguages`, `listRadioTags` (with `limit: 5`, then `offset` set to
   `nextOffset`): lists in the hub's order, the second tag page following the first.
4. `getRadioStation` with a well-formed UUID the catalogue does not hold (`00000000-0000-4000-8000-000000000000`):
   `NotFound`.
5. `searchRadioStations` with `countryCode: "ukr"`: refused by the schema, no request to the hub.

Expected outcome: every call succeeds or fails as listed. If the hub's radio extension is down, step 1 fails
with `RadioUnavailable` and a reason, which is also a pass for that case.

## 3. Audible check (needs the user's permission each time)

Per AGENTS.md: ask first, read the `office` output's volume, mute and enabled state, keep the volume low, then:

1. Pick a station from step 2 whose `lastCheckOk` is true (KissFM is known to start reliably).
2. Call `playback` with that station's `uri` on `office`.
3. Stop the route you created (`stopRoutesForOutput` only if no other route is the user's; otherwise `deleteRoute`
   with the ID you were given), restore the output's volume and enabled state, and delete any input
   `playback` created that remains.
4. Check nothing was left open: no line from `grep -h "^state" /proc/asound/card*/pcm*p/sub*/status` over SSH, and
   `/actuator/metrics/audio.output.routes?tag=output:office` reads 0.

Expected outcome: the station plays on `office`, and afterwards no route or ALSA device is left open.
