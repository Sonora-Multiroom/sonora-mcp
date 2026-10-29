# List the hub's extensions

- **Status:** Proposed, not started. Written 2026-10-26 from a comparison of sonora-cli's `hub` package
  with the tools; listed as out of scope (AS-007) in the master spec.

- **Origin:** `GET /api/v2/extensions` (`listExtensions`) in `api.Spec` 0.1.29; sonora-cli uses
  `hub.ListExtensions` to explain why a TTS call answered "not offered".

- **Depends on:** nothing.

- **Effort:** Small (one read-only tool).

## Problem

The hub's extension inventory says which extensions are loaded and why one was rejected, but no tool
reads it. It was first meant to explain TTS failures; since `specs/002-tts-tools`, the
TTS-unavailable error carries that reason itself (diagnosed by sonora-cli's `hub`), so what remains
is general diagnostics: "which extensions does the hub run?", or why an extension other than TTS
did not load.

## Proposal

A new read-only tool **`listExtensions`** (no arguments) wrapping `hub.ListExtensions`. It returns the
`ExtensionInventory` as `hub` decodes it: `loadingEnabled`, and `extensions` with each one's `id`,
`status` and `rejectionReason`. It is already an object, so it needs no wrapping. Pass `status`
through as a plain string, so a value the hub adds later is not rejected.

`hub` leaves out the spec's other fields on purpose, decoding only what a TTS diagnosis needs. A
version mismatch is a `REJECTED` status with its `rejectionReason`, and `connectionState` is always
`NOT_APPLICABLE` for TTS, which contacts its providers per request.

## Open questions

- Worth a tool only once someone asks an assistant about the hub's extensions; TTS failures no
  longer need it.
