# Feature Specification: Radio station search and browse

**Feature Branch**: `003-radio-station-search`

**Created**: 2026-11-22

**Status**: Draft

**Input**: User description: "@docs/backlog/adopt-radio-search.md": adopt RADIO-002 so an agent can search and
browse the hub's radio station catalogue, find a station's address, and explain why radio search is
unavailable. The shared hub client gained these operations in sonora-cli CLI-014.

## Clarifications

### Session 2026-11-22

- Q: Should station text be passed through byte-for-byte, or have control characters stripped and its length
  capped? → A: Pass it through unchanged (only guaranteeing valid UTF-8 in the JSON result) and warn in the
  tool descriptions that it is untrusted.

## User Scenarios & Testing *(mandatory)*

Actors are those of the master spec: the **Agent** (an AI assistant calling tools), the **Operator** (who
runs the server and the hub) and the **Maintainer**. The person at home who asks the assistant to play a
station is the **User**.

### User Story 1 - The agent finds a station by name (Priority: P1)

The user asks the assistant to "play Kiss FM". Today the agent can play a radio station address
(`radiobrowser://<uuid>`) but cannot find a station's UUID, so the user would have to look it up by hand. The
agent calls one search tool with part of the station's name and gets a page of matching stations, each with
the address to give to `playback`.

**Why this priority**: it is the whole point of the feature. Without it the agent cannot answer "play Kiss FM"
and nothing else here is useful.

**Independent Test**: against a fake hub, call the station search with a name and check that the hub receives
that name and nothing else, and that the tool returns the stations with their addresses and the offset of the
next page.

**Acceptance Scenarios**:

1. **Given** the catalogue has stations whose names contain "kiss", **When** the agent searches with name
   "kiss", **Then** the hub receives only that name filter and the tool returns the page of stations, each
   with its UUID, its `radiobrowser://` address, name, country, languages, tags, codec, bitrate and whether
   its last stream check passed.
2. **Given** the agent sets no filter, **When** it searches, **Then** nothing is sent and the hub's default
   (its most-voted stations) comes back.
3. **Given** the catalogue has more matches than one page holds, **When** the agent searches, **Then** the
   result carries the offset of the next page, and searching again with that offset returns the following
   page; the last page carries no next offset.
4. **Given** nothing matches, **When** the agent searches, **Then** it gets an empty list, not an error.
5. **Given** the agent passes a country code, a language, a tag, a codec, a minimum bitrate, an order, a
   direction or a page size, **When** it searches, **Then** the hub receives each one unchanged and only
   those.
6. **Given** a value the schema forbids (an order or direction outside the offered set, a negative offset or
   minimum bitrate, a page size below one), **When** the agent searches, **Then** it gets an input error and
   the hub receives no request.
7. **Given** the agent reads the tool's metadata, **Then** it is marked read-only and its description says
   the filters narrow the list, that `tag` and `language` match one whole value (the catalogue lists give the
   valid values), and that the result's `uri` is what `playback` takes.

---

### User Story 2 - The agent explains why radio search is unavailable (Priority: P1)

The hub has no radio extension, or it is switched off, failed to load or is too old. The agent's search fails.
It must tell the user why, and what to do, instead of reporting a vague network or not-found error.

**Why this priority**: the extension is optional and out of tree, so on many hubs every radio tool will fail.
An undiagnosed failure sends the user looking in the wrong place. The error handling is part of the first tool,
not an add-on.

**Independent Test**: against a fake hub that answers 404 to the radio paths and reports each extension state
in turn, check that the error is in the radio-unavailable category and names that state.

**Acceptance Scenarios**:

1. **Given** the hub does not serve the radio API, **When** the agent calls any radio tool, **Then** it gets
   an error in a dedicated radio-unavailable category, not a network or not-found error, saying radio is not
   available on this hub and why: the extension is not installed, is disabled in the hub's configuration,
   failed to load (with the hub's reason), is loaded but inactive, or is too old to offer search.
2. **Given** the hub's extension inventory cannot be read, **When** the agent calls a radio tool, **Then** it
   still gets the radio-unavailable error with the generic message, never a hang or a second failure.
3. **Given** one failed radio call, **When** the server explains it, **Then** it asks the hub for the extension
   inventory once for that call, and never for a call that succeeded or failed for another reason.
4. **Given** the radio extension is working but the catalogue service behind it is down or slow, **When** the
   agent calls a radio tool, **Then** it gets a retryable error that says the catalogue is unavailable and,
   when the hub gave a retry delay, how long to wait.
5. **Given** the hub refuses a request as invalid (for example a country code that is not two letters), **When**
   the agent calls a radio tool, **Then** it gets an input error naming the offending parameter.

---

### User Story 3 - The agent looks up one station (Priority: P2)

The agent holds a station UUID, from an earlier search or from a stored input, and wants to confirm what it is
before or after playing it: its name, country, codec, whether its stream was last seen working.

**Why this priority**: useful for confirming a choice and for diagnosing a station that will not play, but the
search result already carries every field, so the agent can do without it.

**Independent Test**: against a fake hub, call the lookup with a UUID and check the station is returned
unchanged; call it with an unknown UUID and check for a not-found error naming it.

**Acceptance Scenarios**:

1. **Given** the catalogue has station `U`, **When** the agent looks up `U`, **Then** it gets that station with
   the same fields a search result carries.
2. **Given** the catalogue has no such station, **When** the agent looks it up, **Then** it gets a not-found
   error naming the UUID, which is not the radio-unavailable category.
3. **Given** an empty UUID, **When** the agent looks it up, **Then** it gets an input error and the hub
   receives no request.

---

### User Story 4 - The agent learns the valid filter values (Priority: P2)

The agent wants stations in Ukrainian, or tagged "jazz". The `language` and `tag` filters match one whole
value, so a guess such as "ukraine" finds nothing. The agent first reads the catalogue's countries, languages
or tags, with the number of stations in each, and then searches with an exact value.

**Why this priority**: without the lists an agent can search by name only reliably. With them, "some jazz
station" and "a Ukrainian station" work. The name search covers the main request.

**Independent Test**: against a fake hub, call each list tool and check that the entries and counts come back
in the hub's order; for tags, check the name filter, page size and offset reach the hub and the next offset
comes back.

**Acceptance Scenarios**:

1. **Given** the catalogue has countries, **When** the agent lists them, **Then** it gets each country's code,
   name and station count, largest first.
2. **Given** the catalogue has languages, **When** the agent lists them, **Then** it gets each language's name
   and station count, largest first.
3. **Given** the catalogue has more tags than one page, **When** the agent lists tags, **Then** it gets one
   page with the offset of the next, and may narrow the list by part of a tag name.
4. **Given** the hub does not serve the radio API, **When** the agent calls any list, **Then** it gets the
   radio-unavailable error of Story 2.
5. **Given** the agent reads each list tool's metadata, **Then** it is marked read-only and says what it
   returns and that its values are what `searchRadioStations` takes.

---

### Edge Cases

- Station names, tags, languages, home pages and icon addresses come from a community catalogue and can hold
  any text, including text that reads like an instruction ("ignore previous instructions and ..."). The server
  returns them only as data fields, never in an error message's instructions, and the tool descriptions say
  the fields are untrusted text.
- A search for a name containing letters in another script, quotes or very long text reaches the hub unchanged;
  the hub decides what it accepts.
- A station whose last stream check failed is still returned, flagged, so the agent can prefer another or warn
  the user.
- A station field the catalogue does not know (name, country, codec, bitrate) arrives as an explicit empty
  value, not as a missing field or a made-up default.
- The radio extension answers more slowly than the hub's other calls. A radio call is allowed long enough for
  the hub's own "catalogue unavailable" answer to arrive first; the agent sees that answer, not a client
  timeout.
- Two agents page through the same search while the catalogue changes. Each call reflects the hub at call
  time; the server keeps no cursor.
- The agent asks for a page size or offset beyond what the hub accepts. The hub's refusal reaches the agent as
  an input error naming the parameter.
- A hub that is too old for this feature, or has no radio extension, still serves every existing tool
  unchanged; only the new tools fail.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The server MUST offer a read-only `searchRadioStations` tool that returns one page of the hub's
  radio stations, with every station field the hub gives: UUID, address (`uri`), name, icon address, home page,
  country code, languages, tags, codec, bitrate, HLS flag, last-check flag, votes and recent play count.
- **FR-002**: `searchRadioStations` MUST accept the optional filters name (part of the name), tag, country code,
  language, codec and minimum bitrate; the optional order (name, votes, play count, bitrate) and direction
  (ascending, descending); and the optional page size and offset. It MUST send only those the agent set, so
  the hub's defaults apply to the rest.
- **FR-003**: A page result MUST carry the offset of the next page, and omit it on the last page, so the agent
  can ask for the following page. An empty page MUST be a normal result.
- **FR-004**: The server MUST offer a read-only `getRadioStation` tool that returns one station by UUID with
  the same fields, and a not-found error naming the UUID when the catalogue has none.
- **FR-005**: The server MUST offer read-only `listRadioCountries`, `listRadioLanguages` and `listRadioTags`
  tools. Countries and languages return the whole list, each entry with its station count, in the hub's order;
  tags return one page with a next offset, and accept an optional name filter, page size and offset.
- **FR-006**: Each list result MUST be an object wrapping the list, and every tool's input and output schema
  MUST agree with the hub spec shipped with the pinned client, as for existing tools.
- **FR-007**: Input that fails a tool's schema (value outside an enum or range, empty UUID, empty path value)
  MUST produce a tool error and no hub request.
- **FR-008**: When the hub does not serve the radio API, every radio tool MUST return an error in a dedicated
  radio-unavailable category, with a message that names the reason (not installed, disabled, failed to load
  with the hub's reason, inactive, too old). The server MUST read the hub's extension inventory at most once
  per failed call, only for that failure, and MUST fall back to a generic radio-unavailable message when the
  inventory cannot be read.
- **FR-009**: A radio error from the hub MUST be mapped onto the existing error categories: an invalid
  request is an input error naming the parameter; an unknown station is not-found; a catalogue that is
  unavailable or slow is retryable and passes on the hub's retry delay when one is given; anything else keeps
  its existing category. No radio error may surface as a bare network failure unless the network failed.
- **FR-010**: Radio calls MUST be bounded by a timeout longer than the hub's own catalogue budget, so the hub's
  answer arrives first, and MUST stop when the agent's request is cancelled.
- **FR-011**: Station text from the catalogue MUST be returned only in result fields, passed through unchanged
  (no stripping of control characters and no truncation; the result only has to be valid UTF-8 text). It MUST
  NOT be placed in an error message's guidance, and tool descriptions MUST tell the agent that these fields are
  untrusted community text and not instructions.
- **FR-012**: Each tool's description MUST say what it returns and that it changes nothing, and each input
  field MUST be described. The `searchRadioStations` and `getRadioStation` descriptions MUST say that a
  station's `uri` is what `playback` takes as an input address.
- **FR-013**: The new tools MUST NOT alter the schema or behaviour of any existing tool, and an unavailable
  radio extension MUST NOT affect them. The README's tool list and `AGENTS.md` MUST be brought in step
  with the new tools; the master spec is brought in step when the feature is archived.
- **FR-014**: The feature MUST use the shared hub client's radio operations and result types, at a tagged
  release, and MUST NOT call the hub's radio paths or define radio types itself. A missing field in the client's
  types is fixed in the client first.

### Key Entities

- **Radio station**: one entry of the hub's community radio catalogue. It has a UUID, an address that `playback`
  can play, a display name, country, languages, tags, codec and bitrate, and quality signals: whether its last
  stream check passed, its votes and its recent play count. Descriptive text is untrusted.
- **Station page**: a bounded list of stations from one search, with the offset of the next page when more
  exist.
- **Catalogue value**: a country (code, name), language (name) or tag (name) with the number of stations that
  carry it. Language and tag filters match these exactly.
- **Radio-unavailable reason**: why the hub does not serve radio: not installed, disabled, failed to load,
  inactive, or too old.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Given a station's name, an agent finds its playable address with one tool call, and the user's
  "play Kiss FM" no longer needs a manual lookup.
- **SC-002**: Every successful search, lookup and list response carries all the fields the hub documents, and
  100% of the five radio tools agree with the hub spec in the automated conformance check.
- **SC-003**: An agent can read the whole result of any search by following next offsets, with no state kept
  in the server. The server passes each offset and next offset through unchanged, so the hub's paging decides
  whether a station repeats or is skipped; the automated test proves the pass-through.
- **SC-004**: On a hub without a working radio extension, 100% of radio tool failures name the reason (or say
  it could not be determined), and none appear as a network or generic error.
- **SC-005**: When the catalogue service behind the hub is down, the agent receives the hub's retryable answer
  in every case, not a client timeout.
- **SC-006**: No existing tool's behaviour or schema changes. Existing tests change only for the `Register`
  signature, the tool count and the deprecated diagnosis type name, plus the conformance check's new
  `maxLength` and list-envelope allowances; every other existing test passes as it is.
- **SC-007**: On the real hub with the radio extension, an agent finds a known station by name, reads its
  details, and plays it through `playback` using the returned address.

## Assumptions

- **Dependencies**: the shared hub client (sonora-cli) already has the radio operations (CLI-014) in the
  release this repository pins today, so no client bump is needed. The hub must run the radio extension 0.2.0
  or later (RADIO-002) for the tools to work; without it they report why.
- **Playback input** (answers the backlog's second open question): `playback` is unchanged and still takes the
  station's `uri` from a search result. The agent does not build `radiobrowser://<uuid>` itself, and `playback`
  does not learn to accept a bare UUID. Revisit only if agents are seen getting it wrong.
- **Tool split** (answers the backlog's first open question): there is one search tool with every filter and a
  separate tool for each catalogue list, because the lists are not search results and an agent needs them to
  fill `tag` and `language`.
- **Names**: the tools follow the existing verb-plus-noun style: `searchRadioStations`, `getRadioStation`,
  `listRadioCountries`, `listRadioLanguages`, `listRadioTags`.
- **Read-only**: no tool in this feature changes hub state, so no step needs the user's permission and none
  makes sound. Playing a found station is the existing `playback` tool.
- **Statelessness**: the server keeps no catalogue data and no paging cursor (constitution III); every call
  asks the hub.
- **Scope out**: favouriting, voting for or recording stations, editing the catalogue, listing the hub's
  extensions as a general tool, and any change to the hub or to sonora-cli are out of scope. A wrong radio
  behaviour in the shared client is fixed there first and then bumped here.
- **Hub API**: the radio operations are outside REST v2, like text-to-speech, and the hub answers 404 on them
  when the extension is not serving.
- **Version impact**: new tools are added without changing any existing one, so the server's change is a minor
  version.
