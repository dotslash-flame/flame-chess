AI generated API contracts

INSERT GARBAGE BELOW 


# FlameChess — API Contract

**Status:** Phase 3 (first playable). In-memory only; no persistence/ratings yet.
**Source of truth:** generated from `internal/wire/wire.go` + `internal/httpapi/router.go` + `internal/ws/ws.go`.

This is the contract a frontend / test client builds against. Sections marked **[Phase 3 — live]** are implemented and stable now. Sections marked **[deferred]** are from the design spec (§6) but **not** served yet — do not build against them.

---

## Auth & sessions

Identity is carried by an HMAC-signed cookie named **`fc_session`** (`HttpOnly`, `Path=/`). The `/ws` upgrade reads it; there is no bearer-token / header auth.

The signed payload (after verify) is an `Identity`:

```json
{ "uid": "u-1a2b3c4d5e6f7a8b", "name": "Alice" }
```

- `uid` — stable per display name (`UserIDForName`): same name → same id.
- `name` — display name.

### `POST /auth/dev-login` **[Phase 3 — live, dev only]**

Mints a session cookie for a display name. Gated by `DEV_LOGIN` (default `true`; Phase 4 turns it off and replaces it with Google OAuth behind the **same** cookie).

- **Body:** form-encoded, field `name` (defaults to `anon` if omitted).
- **Response:** sets the `fc_session` cookie and returns the `Identity` JSON above.

```bash
curl -i -X POST -d 'name=Alice' http://localhost:8080/auth/dev-login
# Set-Cookie: fc_session=<payload>.<hmac>; Path=/; HttpOnly
```

Browser flow: `POST /auth/dev-login` (same origin so the cookie sticks) → open the WebSocket; the cookie rides along automatically.

> **[deferred]** `GET /auth/google/login`, `GET /auth/google/callback`, `POST /auth/logout` — Phase 4.

---

## REST

### `GET /healthz` **[Phase 3 — live]**

Liveness. `200` with `{ "status": "ok" }`.

> **[deferred]** `GET /api/me`, `PATCH /api/me`, `GET /api/leaderboard`, `GET /api/games`, `GET /api/games/{id}`, `POST /api/challenges` — Phases 4–6.

---

## WebSocket — `GET /ws` **[Phase 3 — live]**

Cookie-authenticated on upgrade. Missing or invalid `fc_session` → **`401 Unauthorized`** (no upgrade). Origin is not checked in dev (`InsecureSkipVerify`), so a client on a different port can connect.

All frames are **text** JSON with a `type` discriminator: `{ "type": "...", ...fields }`. Decode `type` first, then the typed struct.

### Server invariant

The server only ever emits authoritative state (real FEN + real clock ms). The client *proposes* a `move` and receives either a new `game.state` or an `error` — never a locally-applied result. Desync and protocol cheating are impossible by construction.

---

### Client → server

| `type` | Fields | Notes |
|---|---|---|
| `queue.join` | `category` (string), `base` (int, seconds), `increment` (int, seconds) | Join the quick-match pool. Pairing is FIFO within an **exact `(base, increment)`** pool. **`category` is currently informational** — the server derives the real category from `base` and does not pool on it. Joining while already queued or in a game is ignored. |
| `queue.leave` | — | Leave the pool. |
| `move` | `game_id` (string), `uci` (string, e.g. `"e2e4"`, `"e7e8q"`) | Propose a move. `game_id` must match your active game. |
| `resign` | `game_id` (string) | Resign the active game. |
| `draw.offer` | `game_id` (string) | Offer a draw to the opponent. |
| `draw.respond` | `game_id` (string), `accept` (bool) | Respond to a pending offer. `accept:true` ends the game `1/2-1/2`; `false` clears the offer and play continues. |
| `ping` | — | Liveness; server replies `pong`. |

Example:

```json
{ "type": "queue.join", "category": "blitz", "base": 300, "increment": 0 }
{ "type": "move", "game_id": "g1", "uci": "e2e4" }
{ "type": "draw.respond", "game_id": "g1", "accept": true }
```

---

### Server → client

| `type` | Fields | When |
|---|---|---|
| `online.count` | `n` (int) | Broadcast to everyone on every register/unregister. |
| `queue.waiting` | — | You joined a pool with no waiting opponent; you're parked. |
| `game.start` | `game_id`, `color` (`"white"`/`"black"`), `opponent` (display name), `clocks` (`{white_ms,black_ms}`), `fen` | A match was made. Colors are random; both players receive opposite colors and the same `game_id`. |
| `game.state` | `game_id`, `fen`, `last_move` (UCI), `white_ms`, `black_ms`, `turn` (`"white"`/`"black"`) | After every accepted move. Authoritative board + clocks. |
| `game.over` | `game_id`, `result` (`"1-0"`/`"0-1"`/`"1/2-1/2"`), `reason` | Terminal transition. **No `rating_delta` in Phase 3** (added in Phase 5). |
| `draw.offered` | `game_id`, `from` (offerer's `uid`) | Opponent offered a draw. |
| `pong` | — | Reply to `ping`. |
| `error` | `code`, `msg` | A request was rejected; game state is untouched. |

`game.over.reason` is one of:
`checkmate`, `stalemate`, `insufficient`, `threefold`, `fifty_move`, `resign`, `draw_agreed`, `timeout`.

Example:

```json
{ "type": "game.start", "game_id": "g1", "color": "white",
  "opponent": "Bob", "clocks": { "white_ms": 300000, "black_ms": 300000 },
  "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" }

{ "type": "game.state", "game_id": "g1",
  "fen": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1",
  "last_move": "e2e4", "white_ms": 298731, "black_ms": 300000, "turn": "black" }

{ "type": "game.over", "game_id": "g1", "result": "0-1", "reason": "checkmate" }
```

---

### Error codes

| `code` | Meaning |
|---|---|
| `bad_message` | Malformed frame, unknown `type`, or undecodable fields. The read pump survives — the connection stays open. |
| `illegal_move` | Move is not legal in the current position. |
| `not_your_turn` | It's the opponent's turn. |
| `not_in_game` | You sent an in-game action but aren't in a game. |
| `game_not_active` | Action against a finished game. |
| `unknown_game` | `game_id` doesn't match your active game. |

---

## Behavior the client must handle (Phase 3)

- **Single active socket per user.** A second `/ws` connection for the same `uid` closes the older one (newest wins). `online.count` is unchanged across the swap.
- **Disconnect mid-game does NOT end the game.** The clock keeps running; you lose on the flag (`game.over` with `reason: "timeout"`). No reconnect grace yet (Phase 7).
- **Clocks are wall-clock authoritative.** Trust `white_ms`/`black_ms` from `game.state`; don't compute results client-side.
- **One game per user at a time** in the quick-match flow.

---

## Not in Phase 3 (don't build against these yet)

- `rating_delta` on `game.over` (Phase 5).
- Challenge-by-link / direct challenge: `challenge.accept`, `challenge.create_direct`, `challenge.incoming` (Phase 6).
- `rematch.offer` / `rematch.respond` / `rematch.offered`, in-game `chat`, spectating, reconnect-resume (Phase 7).
- Any `/api/*` REST endpoints and Google OAuth (Phases 4–6).
