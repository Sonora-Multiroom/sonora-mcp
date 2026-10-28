# Server instructions and a fuller server identity

- **Status:** Proposed, not started. Written 2026-10-27 after checking what the server returns from
  `server/discover` (SEP-2575, protocol 2026-07-28).

- **Origin:** go-sdk v1.8.0 answers `server/discover` (and the legacy `initialize`) from
  `ServerOptions` and `mcp.Implementation`. Production answers with `name`, `version`, `capabilities:
  {"tools": {}}` and the supported versions, and no `instructions`.

- **Depends on:** nothing.

- **Effort:** Small (static text and three identity fields in `cmd/sonora-mcp/main.go`, plus a test).

## Problem

Agents learn how to use the server only from each tool's description. Guidance that spans tools has
no place. Examples: confirm before anything audible or speaker-changing, prefer `routeVolume` over the
room's volume for an announcement, and look up IDs with the list tools instead of guessing. It is
either repeated in many descriptions or missing.

The server also identifies itself only as `sonora-mcp` plus a version. A client listing several
servers shows no title, description or link.

## Proposal

- **`ServerOptions.Instructions`:** a short static text, a few lines, covering:
  - the server controls a real home audio system, so ask before playing, stopping or changing
    speakers;
  - IDs come from the list tools (`listOutputs`, `listGroups`, `listInputs`, `listRoutes`);
  - `routeVolume` sets one playback's volume and leaves the room's volume alone;
  - errors start with a category (`NotFound`, `Conflict`, ...), and the text after it says what to
    do.

  Once the join-mode, bulk-stop and TTS items ship, each adds its own line (for example: announce
  with `DUCK_OTHERS`; if `speak` fails as TtsUnavailable, `listExtensions` says why).
- **`mcp.Implementation`:** add `Title` ("Sonora Multiroom"), a one-sentence `Description`, and
  `WebsiteURL` (the repository).
- A test checks that `server/discover` returns the instructions and the identity fields, so a later
  SDK upgrade cannot drop them silently.

Keep it static. Hub state, such as which hub extensions are loaded, does not belong in discovery:
the answer may be cached, it is built before any hub call, and the hub is unreachable for a few
seconds after every boot (Principle III: no mirrored hub state). Tools such as `listExtensions` give
the live state.

## Open questions

- How much hub behaviour can the text describe without becoming a second copy of the hub's rules
  (Principle III)? Recommendation: say how to use the tools, not what the hub does; link the
  behaviour to the tool that has it.
- Whether clients show `instructions` to the model varies. Check one or two clients (Claude Code,
  the MCP inspector) before relying on it for anything safety-relevant. The per-tool descriptions
  must stay complete on their own.
- The text is part of the agent-facing surface (Principle II). Decide whether changing it needs a
  version bump: a minor version, like a reworded tool description, is the suggestion.
