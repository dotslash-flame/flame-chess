# FlameChess — Barebones UX Tester (design)

**Date:** 2026-06-07
**Status:** approved, pre-implementation
**Goal:** A minimal, no-build web client to manually exercise the Phase 3 WebSocket flow (login → queue → play → game over) and watch the raw protocol. Built strictly against the contract in `API.md`.

## Scope

In: dev-login, queue join/leave, live board play over `/ws`, resign, draw offer/respond, online count, game over, raw event log.

Out (per API.md, do not build): ratings/`rating_delta`, challenges, rematch, chat, spectating, `/api/*`, Google OAuth, reconnect/persistence.

## Two-identity constraint

`/ws` authenticates **only** via the `fc_session` cookie (`internal/ws/ws.go` `identity()`), which is shared across tabs in one browser context. To run two distinct players we use **two cookie jars**: the tester is served same-origin from the Go server, the user opens Alice in a normal window and Bob in an incognito/second window. **No backend auth change.**

## Files

- `web/index.html` — the entire tester: markup + CSS + vanilla JS in one file, no build step.
- `internal/httpapi/router.go` — add a static route (`GET /`, serving `web/index.html`). `/healthz`, `/auth/dev-login`, `/ws` unchanged. The embed/serve path must not break existing router tests.

## UI layout (single column)

1. **Auth bar** — name input + `Login` → `POST /auth/dev-login` (form-encoded, `credentials: 'same-origin'`). Displays returned `{uid, name}`.
2. **Queue bar** — `base` (sec), `increment` (sec) inputs + `Join`/`Leave` → `queue.join`/`queue.leave`. Shows `online.count` and `queue.waiting`.
3. **Board** — 8×8 grid, Unicode pieces from authoritative FEN. Click source square → click destination = `move`. UCI text box kept as fallback. Orientation flips for Black.
4. **Game panel** — opponent name, both clocks, turn indicator, `Resign` / `Offer draw`. On `draw.offered` show `Accept`/`Decline` → `draw.respond`. On `game.over` show result + reason and re-enable `Join`.
5. **Event log** — raw JSON of every frame sent ↑ / received ↓.

## Behavior

- **Server authoritative.** Board/clocks only update from `game.start` / `game.state` / `game.over`. Never apply a move locally; an `error` frame leaves state untouched (shown + logged).
- **Clocks.** Display server `white_ms`/`black_ms`; tick the side-to-move down locally between frames for feel, resync on every `game.state`.
- **Promotion.** Pawn landing on rank 8/1 → `prompt()` for q/r/b/n (default `q`), append to UCI (e.g. `e7e8q`).
- **Single socket per user** (newest wins) is the server's behavior; the tester does not special-case it beyond logging a close.

## Error codes handled (display + log)

`bad_message`, `illegal_move`, `not_your_turn`, `not_in_game`, `game_not_active`, `unknown_game`.

## Testing / acceptance

Manual: `go run ./cmd/server`, open `localhost:8080` in two windows (normal + incognito), login as Alice/Bob, both join the same `(base, increment)` pool, confirm pairing, play moves both directions, exercise resign + draw-agree, watch clocks and the event log. Existing Go test suite still passes.
