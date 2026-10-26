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
| P2 | [Join modes and an output's routes](join-modes.md) | Feature | Small–Medium | Every `playback` replaces what is playing, so an announcement stops the user's music instead of ducking it |
| P3 | [Route volume and ducking level](play-route-volume.md) | Feature | Small | Lets an assistant play an alert at its own volume without changing the room, and change it while it plays |
| P3 | [Rename an input](rename-input.md) | Feature | Small | The hub can rename inputs without interrupting them; no tool exposes it |
