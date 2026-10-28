# Backlog index

Every item in `docs/backlog/`, highest priority first. Each one is a proposal, not a requirement. When
you add, ship or delete an item, update this list.

| Priority | Meaning |
|---|---|
| **P1** | A crash, data loss or other user-visible fault. Do it next. |
| **P2** | Wrong or confusing behaviour, or something users need now. |
| **P3** | A worthwhile feature or cleanup with no pressure on it. |
| **P4** | Only worth doing once a concrete need appears. |

| Priority | Item | Kind | Effort | Why this priority |
|---|---|---|---|---|
| P2 | [Join modes, ducking level and an output's routes](join-modes.md) | Feature | Small–Medium | Every `playback` replaces what is playing, so an announcement stops the user's music instead of ducking it |
| P3 | [Speak announcements](tts-speak.md) | Feature | Small–Medium | The hub can speak text but no tool exposes it; a first version works on sonora-cli v0.3.1, ducking needs `playbackMode` added to sonora-cli's `SpeakRequest` |
| P3 | [Server instructions and a fuller server identity](server-instructions.md) | Feature | Small | Cross-tool guidance (confirm before audible actions, use `routeVolume` for announcements) has no place; discovery returns no instructions |
| P3 | [Rename an input](rename-input.md) | Feature | Small | The hub can rename inputs without interrupting them; no tool exposes it |
| P4 | [List the hub's extensions](list-extensions.md) | Feature | Small | General diagnostics only; TTS errors now carry their own reason |
| P4 | [TTS cache statistics and clearing](tts-cache.md) | Feature | Small | Maintenance an assistant is rarely asked for |
