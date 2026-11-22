# Data model: Radio station search and browse

All result types are the shared client's (`hub.Radio*`); this repository defines only the two wrapper
results and the input structs. Field names are the JSON names an agent sees. A field marked `null` is always
present, with `null` when the catalogue does not know the value ([research](research.md) R4).

## Inputs

| Tool | Field | Type | Required | Constraint |
|---|---|---|---|---|
| `searchRadioStations` | `name` | string | no | maxLength 100 |
| | `tag` | string | no | maxLength 100 |
| | `countryCode` | string | no | pattern `^[A-Za-z]{2}$` |
| | `language` | string | no | maxLength 100 |
| | `codec` | string | no | maxLength 32 |
| | `bitrateMin` | integer | no | 0 to 1000000 |
| | `order` | string | no | `name`, `votes`, `clickCount`, `bitrate` |
| | `direction` | string | no | `asc`, `desc` |
| | `limit` | integer | no | 1 to 100 |
| | `offset` | integer | no | 0 to 10000 |
| `getRadioStation` | `uuid` | string | yes | minLength 1 |
| `listRadioCountries` | none | | | |
| `listRadioLanguages` | none | | | |
| `listRadioTags` | `name` | string | no | maxLength 100 |
| | `limit` | integer | no | 1 to 100 |
| | `offset` | integer | no | 0 to 10000 |

Optional fields are pointers with `omitempty`: an omitted field is not sent, so the hub's defaults apply (20
stations or tags, offset 0, order `votes`). The server fills in no default.

## Results

### Radio station (`hub.RadioStation`)

| Field | Type | Meaning |
|---|---|---|
| `uuid` | string | the station's identifier, lower case |
| `uri` | string | `radiobrowser://<uuid>`, to pass unchanged to `playback` as the input address |
| `name` | string or null | display name (untrusted text) |
| `favicon` | string or null | icon address, not fetched (untrusted) |
| `homepage` | string or null | web page (untrusted) |
| `countryCode` | string or null | ISO 3166-1 alpha-2 |
| `languages` | string array | at most 10 language names (untrusted) |
| `tags` | string array | at most 30 catalogue tags (untrusted) |
| `codec` | string or null | catalogue label, such as `MP3` |
| `bitrate` | integer or null | kbit/s |
| `hls` | boolean | the stream is an HLS playlist |
| `lastCheckOk` | boolean | the catalogue's last check of the stream passed |
| `votes` | integer | votes received |
| `clickCount` | integer | plays in the last 24 hours |

### Station page (`hub.RadioStationPage`)

`stations`: array of radio station, possibly empty. `nextOffset`: integer, or null on the last page. Searching
again with `offset` set to it returns the following page. The server keeps no cursor.

### Country list (`radioCountries`, new wrapper)

`countries`: array of `{code, name, stationCount}` (`hub.RadioCountry`), in the hub's order, largest first.
`code` is upper-case ISO 3166-1 alpha-2, `name` is English, `stationCount` is an integer. The code is the value
for `countryCode`.

### Language list (`radioLanguages`, new wrapper)

`languages`: array of `{name, code, stationCount}` (`hub.RadioLanguage`), largest first. `code` is the ISO 639
code or null. `name` is the value for `language`, which matches one whole name.

### Tag page (`hub.RadioTagPage`)

`tags`: array of `{name, stationCount}`, possibly empty. `nextOffset`: integer or null. `name` is the value for
`tag`, which matches one whole tag.

## Error mapping

| Hub answer | Client error | Category | Message adds |
|---|---|---|---|
| 404 on a radio path, not "station not found" | `*hub.RadioNotOfferedError`, then diagnosed to `*hub.RadioUnavailableError` | `RadioUnavailable` | the reason: not installed (loading switched off or not), disabled, failed to load (hub's reason), inactive, or too old (needs 0.2.0) |
| the same, and the diagnosis finds the hub address wrong | `*hub.RadioUnavailableError` with `DiagnosisHubAddress` | `Network` | the address message |
| the same, and the inventory cannot be read | `*hub.RadioUnavailableError` with `DiagnosisUnknown` | `RadioUnavailable` | "radio is not available on this hub" only |
| 404 "station not found" | `*hub.NotFoundError` | `NotFound` | the UUID |
| 400 `invalid-parameter` | `*hub.APIError` | `Validation` | the hub's detail and `(parameter <name>)` |
| 503 `catalogue-unavailable` | `*hub.APIError` | `ServiceUnavailable` | "retry in N s" when `Retry-After` was sent |
| 502 `catalogue-bad-answer` | `*hub.APIError` | `SourceUnreachable` | the hub's detail |
| other radio problem | `*hub.APIError` | `HubError` | the hub's detail |
| deadline passed | context error | `Timeout` | |
| body not the declared shape | `*hub.DecodeError` | `MalformedResponse` | |

`RadioUnavailable` is the one new category. Every row except it and the `(parameter ...)` text is the existing
mapping.

## State

None. Every call is one hub request (plus one inventory request after a "not offered" answer).
