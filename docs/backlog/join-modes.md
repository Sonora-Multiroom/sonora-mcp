# Join modes, and what an output is playing

- **Status:** Proposed, not started. Written 2026-10-25.

- **Origin:** hub `023-multi-route-output-mixing` (API 0.1.21); sonora-cli PR #27, released in `v0.2.2`.

- **Depends on:** sonora-cli v0.3.1, shipped ([specs/tiny/adopt-sonora-cli-0.3.md](../../specs/tiny/adopt-sonora-cli-0.3.md)).

- **Effort:** Small to medium (three optional fields and one new tool).

## Problem

Since hub 0.1.21 an output can play several routes at once, and each route says how it joins its
target: `REPLACE` stops what is playing there (except ducking routes), `MIX` plays alongside it, and
`DUCK_OTHERS` plays alongside it and lowers it while it plays. The tools cannot choose a join mode, so
every `playback` and `createRoute` replaces what is playing: an assistant asked to announce something
stops the user's music instead of talking over it.

An agent also cannot see what an output plays. `listRoutes` filtered by `targetId` misses group routes
that play on the output, and nothing shows which route is lowered under a ducking one.

## Proposal

- **`joinMode`** (enum `REPLACE`, `MIX`, `DUCK_OTHERS`, optional) on `playback` and `createRoute`, sent
  as `hub.PlaybackRequest.JoinMode` / `hub.CreateRouteRequest.JoinMode`. Omitted means the hub's default:
  `REPLACE` for play; the input's default join mode, else `REPLACE`, for a route. Describe each mode in
  one clause and suggest `DUCK_OTHERS` for announcements.
- **`defaultJoinMode`** (same enum, optional) on `createInput`, sent as
  `hub.CreateInputRequest.DefaultJoinMode`.
- **New read-only tool `listOutputRoutes`** (`outputId`), wrapping `hub.GetRoutesForOutput`
  (`GET /api/v2/outputs/{outputId}/routes`): every route on the output, group routes included, in join
  order, each with its `joinMode`, `duckState` (`FULL` or `LOWERED`) and whether it is the output's
  `main` route. Wrap the list as `{"routes": [...]}`, as the other list tools do.
- Mention in `transferRoute`'s and `createRoute`'s descriptions that an output may play several routes,
  and that `outputs` on a route lists where it plays right now.

The enum values come from the spec; per its contract an unknown join mode or duck state read back from
the hub is passed through unchanged, which the inferred output schema must allow (plain string, no
enum on the response side).

## Open questions

- Name: `listOutputRoutes` or `getOutputRoutes`? The first matches `listRoutes`; the hub's operation is
  `getRoutesForOutput`. Either is a new name, so either is a minor release.
- Should `getOutput` gain an `includeRoutes` flag instead, like `sonora get outputs/<id>
  --include-routes`? A separate tool keeps each tool one hub operation (the inventory rule in the master
  spec), so the recommendation is the separate tool.
