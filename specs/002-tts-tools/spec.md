# Feature Specification: Text-to-speech tools

**Feature Branch**: `002-tts-tools`

**Created**: 2026-10-27

**Status**: Draft

**Input**: User description: "@docs/backlog/tts-speak.md @docs/backlog/tts-cache.md
@docs/backlog/list-extensions.md": speak announcements through the hub's TTS extension, list a
provider's voices, read and clear the TTS cache, and list the hub's extensions so an agent can
explain why text-to-speech is unavailable. Revised 2026-10-28: the reason now comes in the
TTS-unavailable error itself, diagnosed by the shared hub client, so `listExtensions` is no longer
part of this feature.

## User Scenarios & Testing *(mandatory)*

Actors are those of the master spec: the **Agent** (an AI assistant calling tools), the
**Operator** (who runs the server and the hub) and the **Maintainer**. The person at home who asks
the assistant to say something is the **User**.

### User Story 1 - The agent says something in a room (Priority: P1)

The user asks the assistant to announce "dinner is ready" in the kitchen, or "the washing machine has
finished" downstairs. The agent calls one tool with the text and the output or group, and the
announcement is spoken there by the hub's TTS extension, using the provider, voice and playback mode
the hub is configured with unless the agent names others.

**Why this priority**: spoken announcements are one of the main reasons people connect an assistant
to a home audio system. The hub already does it; no tool exposes it. Nothing else in this feature is
useful without it.

**Independent Test**: against a fake hub, call `speak` with text, a target and a target type, and
check that the hub receives the announcement request with exactly those fields and that the tool
returns the hub's acceptance (announcement ID, whether the audio came from the cache, queue depth).

**Acceptance Scenarios**:

1. **Given** output `kitchen` exists, **When** the agent calls `speak` with text "Dinner is ready",
   target `kitchen` and target type `SINGLE_OUTPUT`, **Then** the hub receives that announcement and
   the tool returns its announcement ID, cache-hit flag and queue depth.
2. **Given** the agent omits provider, voice and language, **When** it calls `speak`, **Then** none of
   them is sent and the hub's configured defaults apply.
3. **Given** the agent passes provider `piper-local` and a voice, **When** it calls `speak`, **Then**
   the hub receives both unchanged.
4. **Given** empty text, a missing target, or a target type other than `SINGLE_OUTPUT` or
   `OUTPUT_GROUP`, **When** the agent calls `speak`, **Then** it gets an input error and the hub
   receives no request.
5. **Given** the target is not a known output or group ID, **When** the agent calls `speak`, **Then**
   it gets a not-found error naming the target.
6. **Given** the hub has no TTS extension serving announcements, **When** the agent calls `speak`,
   **Then** it gets an error in a dedicated TTS-unavailable category, not a network error, saying
   text-to-speech is not available on this hub and why: the extension is not installed, is disabled
   in the hub's configuration, failed to load (with the hub's reason), is loaded but inactive, or
   does not match this server's hub API version.
7. **Given** the agent reads the `speak` tool's metadata, **Then** it is marked as changing state and
   not idempotent, and its description says the call returns once the announcement is queued, not
   once it has been spoken, and that omitting the provider uses the hub's configured default, which
   may be a billed service.

---

### User Story 2 - The agent controls how an announcement plays (Priority: P2)

The user is listening to music in the living room. The agent announces something so that the music
keeps playing quietly underneath (ducked) or plays on unchanged (mixed), at a volume chosen for the
announcement alone, without touching the room's volume.

**Why this priority**: without it every announcement uses the extension's configured playback mode
and the room's volume, which is acceptable but not what the user may want. It needs the hub client
bumped to sonora-cli v0.3.4 (see Assumptions).

**Independent Test**: against a fake hub, call `speak` with playback mode, announcement volume and
ducking level, and check that the hub receives them unchanged; call it without them and check that
none is sent.

**Acceptance Scenarios**:

1. **Given** music plays on `livingroom`, **When** the agent calls `speak` with playback mode
   `duck-others`, announcement volume 40 and ducking level -20 dB, **Then** the hub receives all three
   unchanged.
2. **Given** an announcement volume outside 0–100 or a ducking level outside -60 to 0 dB, **When** the
   agent calls `speak`, **Then** it gets an input error and the hub receives no request.
3. **Given** a playback mode the hub does not accept, or a ducking level together with playback mode
   `mix`, **When** the agent calls `speak`, **Then** the hub's rejection reaches the agent as a
   validation error with the hub's explanation.
4. **Given** the agent reads the `speak` description, **Then** it learns that the hub accepts
   `duck-others` and `mix`, that `duck-others` keeps other audio playing quieter underneath, and that
   the announcement volume changes no output's volume.

---

### User Story 3 - The agent picks a valid voice (Priority: P2)

The user asks for an announcement "in a male voice" or "in Ukrainian". The agent lists the voices a
provider offers, optionally filtered by language and engine, and passes one of them to `speak`
instead of guessing a name.

**Why this priority**: a guessed voice name fails; listing removes the guess. It is read-only and
small, but only matters once `speak` exists.

**Independent Test**: against a fake hub, call `listTtsVoices` for a provider with and without a
language filter and check the request and that the result lists each voice's short name, full name,
engine, language and gender.

**Acceptance Scenarios**:

1. **Given** provider `piper-local` has voices, **When** the agent calls `listTtsVoices` for it,
   **Then** it gets the provider name and the voice list as the hub returns them.
2. **Given** a language filter, **When** the agent calls `listTtsVoices`, **Then** the hub receives
   the filter and only matching voices come back.
3. **Given** a provider the hub does not know, **When** the agent calls `listTtsVoices`, **Then** it
   gets a not-found error naming the provider.
4. **Given** a provider with no voices for the filter, **When** the agent calls `listTtsVoices`,
   **Then** it gets an empty list, not an error.
5. **Given** the agent reads the tool's metadata, **Then** it is marked read-only.

---

### User Story 4 - The agent inspects and clears the TTS cache (Priority: P4)

After the operator changes the configured voice, old phrases keep playing in the old voice because
the hub cached them. On request, the agent reads how large the cache is and clears it, for all
providers or for one.

**Why this priority**: maintenance an assistant is rarely asked for; kept in this feature only
because it shares the TTS error handling and test setup with the stories above.

**Independent Test**: against a fake hub, call `getTtsCacheStats` and check the four figures; call
`clearTtsCache` without and with a provider and check the request and the confirmation.

**Acceptance Scenarios**:

1. **Given** the cache holds entries, **When** the agent calls `getTtsCacheStats`, **Then** it gets
   the total entries, total size, maximum size and entries per provider.
2. **Given** no provider is named, **When** the agent calls `clearTtsCache`, **Then** the whole cache
   is cleared and the tool confirms it.
3. **Given** provider `piper-local` is named, **When** the agent calls `clearTtsCache`, **Then** only
   that provider's entries are cleared and the confirmation names the provider.
4. **Given** the agent reads the metadata, **Then** `getTtsCacheStats` is read-only and
   `clearTtsCache` is destructive and idempotent.

---

### Edge Cases

- **TTS extension missing, disabled, rejected or inactive**: every TTS tool answers in the
  TTS-unavailable category with the reason, never as a network error. Finding the reason costs one
  extra hub call (the extension inventory), made only after a TTS call has failed this way and within
  the same time bound.
- **Reason cannot be found**: if the inventory call itself fails, the error is still TTS-unavailable,
  without a reason. If the hub does not serve the inventory either, the address is not a Multiroom
  hub, and the error says so in the network category.
- **TTS provider down, slow or rate-limited**: the hub's own failure reaches the agent in the
  service-unavailable category with the hub's explanation, so the agent can retry later.
- **Slow announcement acceptance**: a cold cloud provider can take longer than an ordinary hub call
  to accept an announcement. `speak` waits for the longer bound the shared hub client sets for
  announcements (15 seconds); every other TTS tool keeps the ordinary bound (5 seconds).
- **Target disabled or not available**: whatever the hub answers reaches the agent with its category
  and explanation; the server does not check the target itself.
- **Target given by display name** ("Kitchen speaker") instead of ID: the hub answers not-found; the
  `speak` description says the target takes the same IDs as the other tools' `targetId`.
- **Announcement queued behind others**: `speak` succeeds and reports the queue depth; it does not
  wait for earlier announcements.
- **Unknown provider for clearing the cache**: the hub answers `PROVIDER_NOT_FOUND`, which reaches
  the agent as a not-found error naming the provider.
- **Hub restarted mid-announcement**: queued announcements are lost with the hub's other state; the
  server keeps no record of them.

## Requirements *(mandatory)*

### Functional Requirements

**Speaking**

- **FR-001**: The server MUST offer a `speak` tool that asks the hub to speak a text on one output or
  output group. Text, target ID (`targetName`) and target type (`SINGLE_OUTPUT` or `OUTPUT_GROUP`)
  are required; provider, voice and language are optional.
- **FR-002**: `speak` MUST send only the optional fields the agent gave, so the hub's configured
  defaults apply to the rest. It MUST NOT choose a provider, voice, language or playback mode itself.
- **FR-003**: `speak` MUST return the hub's acceptance unchanged: announcement ID, cache-hit flag and
  queue depth.
- **FR-004**: The `speak` description MUST say that the call returns once the announcement is queued,
  not once it has been spoken; that the target takes the same IDs as `targetId` elsewhere, not display
  names; and that omitting the provider uses the hub's configured default, which may be a billed
  cloud service.
- **FR-005**: `speak` MUST be marked as changing state, not destructive and not idempotent (each call
  queues another announcement).
- **FR-006**: `speak` MUST accept optional playback mode (`playbackMode`),
  announcement volume (`routeVolume`, 0–100) and ducking level (`duckingLevelDb`, -60 to 0 dB), reject out-of-range numbers before
  contacting the hub, send each only when given, and describe the two playback modes the hub accepts
  (`duck-others`, `mix`) and that the announcement volume changes no output's volume.
- **FR-007**: `speak` MUST NOT expose engine, pitch, speaking rate or style prompt in this feature.

**Voices**

- **FR-008**: The server MUST offer a read-only `listTtsVoices` tool taking a required provider name
  and optional language and engine filters, returning the provider name and its voices (short name,
  full name, engine, language, gender) as the hub reports them. An empty list is a success.

**Cache**

- **FR-009**: The server MUST offer a read-only `getTtsCacheStats` tool, without arguments, returning
  total entries, total size in bytes, maximum size in bytes and entries per provider.
- **FR-010**: The server MUST offer a `clearTtsCache` tool, marked destructive and idempotent, with an
  optional provider name. Without one it clears the whole cache; with one, only that provider's
  entries. It MUST confirm the clearing and, when a provider was given, name it.

**Errors**

- **FR-011**: When the hub does not serve a TTS operation, the tool MUST ask the shared hub client
  for the reason and return an error whose category is a new `TtsUnavailable`, saying that
  text-to-speech is not available on this hub and why (not installed, with whether extension loading
  is switched off; disabled; failed to load, with the hub's reason; inactive; or a hub API version
  mismatch). It MUST NOT be reported as a network failure, except when the hub address turns out
  not to serve the hub API at all.
- **FR-012**: The reason MUST come from the shared hub client's diagnosis, used as is; the server
  MUST NOT interpret the extension inventory itself.
- **FR-013**: Other TTS failures MUST keep the categories the shared hub client assigns: an unknown
  target as not-found, an unknown provider as not-found, a rejected request as validation, and a
  provider failure, timeout or rate limit as service-unavailable, each with the hub's explanation.
- **FR-014**: `speak` MUST be bounded by the shared hub client's announcement time bound; every other
  tool in this feature by the ordinary bound. The diagnosis after a failure runs within the same
  bound. A hub call that exceeds its bound MUST return a timeout error. When the hub call itself
  answered "not offered" in time but the diagnosis exceeds the bound, the error is
  `TtsUnavailable` without a reason, not a timeout (see Edge Cases, "Reason cannot be found").

**Surface and documentation**

- **FR-015**: Every new tool's inputs and outputs MUST match the hub API specification shipped with
  the pinned hub client, as checked for every existing tool; field names follow the hub's, including
  `targetName`.
- **FR-016**: The new tools are additions (28 → 32 tools); no existing tool changes, so the release is
  MINOR. The README tool list, the agent instructions file and the master spec's tool inventory MUST
  list them, and the master spec's out-of-scope note MUST no longer name them. The master spec
  (`.specify/memory/`) is updated by `/speckit-archive-run` after merge, as for every feature; the
  README and agent instructions are updated in this feature.
- **FR-017**: The backlog items this feature ships MUST be deleted, or reduced to what remains, and
  the backlog index kept in sync. The `listExtensions` item stays, rewritten as general diagnostics
  now that TTS errors carry their reason.

### Key Entities

- **Announcement request**: text, target ID and type, and optionally provider, voice, language,
  playback mode, announcement volume and ducking level. Sent once; the server keeps nothing.
- **Announcement acceptance**: announcement ID, whether the audio came from the cache, and how many
  announcements are queued.
- **Voice**: short name (the value `speak` takes), full name, engine, language, gender; listed per
  provider.
- **Cache statistics**: total entries, total size, maximum size, entries per provider.
- **TtsUnavailable**: a new error category for "the hub does not offer text-to-speech right now",
  with the reason the shared hub client diagnosed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An agent can make the hub speak a sentence in a named room with one tool call, and the
  call returns within 15 seconds even when the hub never answers.
- **SC-002**: 100% of the hub's documented announcement, voice and cache responses reach the agent
  with every field the hub client provides, unchanged.
- **SC-003**: 100% of invalid inputs the hub API specification can describe (missing text or target,
  wrong target type, out-of-range volume or ducking level) are rejected without any request reaching
  the hub.
- **SC-004**: When text-to-speech is unavailable, 100% of TTS tool calls return the TTS-unavailable
  category with the reason, and 0% are reported as network failures.
- **SC-005**: In one live check on the production hub with the free local provider, at low volume,
  an announcement is heard on the chosen output, and afterwards nothing the test started is left
  playing.
- **SC-006**: The server lists 32 tools, the existing 28 unchanged, and the full quality gate passes.

## Assumptions

- **Shared hub client: sonora-cli v0.3.4** (released 2026-10-28; pinned today: v0.3.1). v0.3.2
  (2026-10-27) speaks, lists voices and reads and clears the cache, and over v0.3.1 adds the
  announcement request's playback mode, announcement volume, ducking level, engine, pitch, speaking
  rate and style prompt, each sent only when set; that covers User Story 2. v0.3.3 (2026-10-28) adds
  the "not offered" diagnosis below. v0.3.4 gives the cache statistics type its JSON field names,
  which it lacks so far (plan research R2). The bump is additive. Its hub API specification is still
  0.1.29.
- **"Not offered" diagnosis** (sonora-cli #31, `hub.DiagnoseTTSUnavailable`): up to v0.3.2 the
  shared hub client reports a TTS endpoint the hub does not serve as "not offered", classifies it as
  a network failure, and keeps the code that finds the reason inside the `sonora` CLI. v0.3.3 moves
  that diagnosis into the client (one extension-inventory lookup, returning a TTS-unavailable error
  with the reason) and classifies an undiagnosed "not offered" as TTS-unavailable. This feature
  uses that rather than copying the diagnosis; the extra hub call happens only after a
  failure and does not make the server more than a thin proxy, since it looks up the hub's own
  report.
- **`listExtensions` left out**: its purpose in the backlog was explaining TTS failures, which the
  error now does. It stays in the backlog as general diagnostics.
- **Announcement time bound**: the shared client's announcement bound (15 seconds) is used as is,
  which the constitution allows for TTS; a cold cloud provider that needs longer is the hub's
  timeout to report.
- **Playback mode values**: the hub API specification declares playback mode as plain text; the hub
  accepts only `duck-others` and `mix`. The tool documents both and leaves the rejection of other
  values to the hub, as the specification does not constrain them.
- **No confirmation argument**: MCP clients already confirm state-changing and destructive tools;
  `speak` and `clearTtsCache` take no extra confirmation field.
- **Expressive voice fields** (engine, pitch, speaking rate, style prompt) can be sent by v0.3.2 but
  depend on the provider; they are left out until someone needs them, as sonora-cli's own command
  does.
- **Live testing** follows the production rules: ask first, use the free local provider, keep the
  volume low, record and restore what changes.
