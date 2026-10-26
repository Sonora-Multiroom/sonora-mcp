# Rename an input

- **Status:** Proposed, not started. Written 2026-10-25.

- **Origin:** hub `030-input-rename-api` (API 0.1.27); sonora-cli PR #26, released in `v0.2.1`.

- **Depends on:** sonora-cli v0.3.1, shipped ([specs/tiny/adopt-sonora-cli-0.3.md](../../specs/tiny/adopt-sonora-cli-0.3.md)).

- **Effort:** Small (one new tool).

## Problem

The hub can rename any input, configured or runtime, without interrupting its playback, but no tool
exposes it. An agent asked to give a station a friendlier name, or to fix the name a played link got,
cannot.

## Proposal

New idempotent tool **`setInputDisplayName`** (`inputId`, `displayName`), wrapping
`hub.SetInputDisplayName` (`PUT /api/v2/inputs/{inputId}/display-name`). It returns the input as it now
reads. The description says:

- playback continues and the input's ID, enabled state and routes are unchanged;
- the hub trims the name and allows at most 64 characters with no control characters (400, Validation);
- a rename lasts until a hub restart for a configured input and until removal for a runtime input;
- an empty or null name restores the input's original name.

Constraints come from the spec; whatever it does not state is left to the hub's 400.

## Open questions

- How to express "restore the original name": an optional `displayName` (omitted sends null), or
  require it and document that an empty string restores? The hub treats both alike; an optional field
  is clearer to an agent, but a required one makes an accidental reset less likely. Recommendation:
  required, with "send an empty string to restore the original name".
