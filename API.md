AI generated API contracts

INSERT GARBAGE BELOW 


# FlameChess — API Contract

**Status:** Phase 4 (auth). Still in-memory; no persistence/ratings yet.
**Source of truth:** generated from `internal/wire/wire.go` + `internal/httpapi/router.go` + `internal/httpapi/auth.go` + `internal/ws/ws.go`.

This is the contract a frontend / test client builds against. Sections marked **[live]** are implemented and stable now. Sections marked **[deferred]** are from the design spec (§6) but **not** served yet — do not build against them.

---

## Auth & sessions

Identity is carried by an HMAC-signed cookie named **`fc_session`** (`HttpOnly`, `Path=/`, `SameSite=Lax`, `Secure` in production). The `/ws` upgrade and `GET /api/me` read it; there is no bearer-token / header auth.

The signed payload (after verify) is an `Identity`:

```json
{ "uid": "u-1a2b3c4d5e6f7a8b", "email": "alice@flame.edu.in", "name": "Alice" }
```

- `uid` — stable per identity. From Google login it derives from the Google `sub` (`UserIDForSub`); from dev-login it derives from the display name (`UserIDForName`).
- `email` — present for Google logins; omitted for dev-login cookies.
- `name` — display name.

### Google OAuth **[live]**

Production login. Gated to verified Flame University accounts: the email must be verified **and** end with `@<ALLOWED_EMAIL_SUFFIX>` (default `flame.edu.in`). These routes are registered only when `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`/`GOOGLE_REDIRECT_URL` are configured.

- **`GET /auth/google/login`** — sets a short-lived `fc_oauth_state` cookie and `302`-redirects to Google's consent screen.
- **`GET /auth/google/callback`** — Google redirects here with `code` + `state`. The server validates `state` against the cookie (mismatch/missing → `400`), exchanges the code, and applies the suffix gate:
  - **allowed** → sets the `fc_session` cookie and `302`-redirects to `APP_REDIRECT_URL` (default `/`).
  - **rejected** (unverified or non-Flame email) → `403` with a minimal "Flame accounts only" HTML page.

### `POST /auth/logout` **[live]**

Clears the `fc_session` cookie (expired `Set-Cookie`). Returns `204`. Afterwards `GET /api/me` is `401` and the `/ws` upgrade is `401`.

### `POST /auth/dev-login` **[live, dev only]**

Mints a session cookie for a display name **without Google**. Gated by `DEV_LOGIN` (default `true`); production runs `DEV_LOGIN=false`, which removes this route (`404`) and requires the Google flow. This is the path the local frontend uses while developing.

- **Body:** form-encoded, field `name` (defaults to `anon` if omitted).
- **Response:** sets the `fc_session` cookie and returns the `Identity` JSON above.

```bash
curl -i -X POST -d 'name=Alice' http://localhost:8080/auth/dev-login
# Set-Cookie: fc_session=<payload>.<hmac>; Path=/; HttpOnly; SameSite=Lax
```

Browser flow: `POST /auth/dev-login` (same origin so the cookie sticks) → open the WebSocket; the cookie rides along automatically.

---

## REST

### `GET /healthz` **[live]**

Liveness. `200` with `{ "status": "ok" }`.

### `GET /api/me` **[live]**

Returns the identity carried by the `fc_session` cookie. No body required.

- **`200`** with JSON `{ "uid": "...", "email": "...", "display_name": "..." }` (`email` is empty for dev-login sessions).
- **`401`** if the cookie is missing or fails verification.

```bash
curl -i --cookie 'fc_session=<payload>.<hmac>' http://localhost:8080/api/me
```

> **[deferred]** `PATCH /api/me` (durable display-name change — Phase 5), `GET /api/leaderboard`, `GET /api/games`, `GET /api/games/{id}`, `POST /api/challenges` — Phases 5–6.

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

## Not yet live (don't build against these yet)

- `rating_delta` on `game.over` (Phase 5).
- `PATCH /api/me` durable display-name change, ratings on `/api/me` (Phase 5).
- Challenge-by-link / direct challenge: `challenge.accept`, `challenge.create_direct`, `challenge.incoming` (Phase 6).
- `rematch.offer` / `rematch.respond` / `rematch.offered`, in-game `chat`, spectating, reconnect-resume (Phase 7).
- `GET /api/leaderboard`, `GET /api/games`, `GET /api/games/{id}` (Phases 5–6).

---

## Frontend handoff notes

- **Local dev login:** the frontend authenticates via `POST /auth/dev-login?name=X` (no Google needed) while `DEV_LOGIN=true`. **Production** uses the Google flow (`GET /auth/google/login`); dev-login is `404` there.
- **Cross-origin (CORS):** set `CORS_ALLOWED_ORIGINS` to a comma-separated allowlist of frontend origins (e.g. `https://app.flame.edu.in,http://localhost:5173`). When set, the server returns credentialed CORS headers (`Access-Control-Allow-Origin` echoed per-request, `Access-Control-Allow-Credentials: true`, preflight `OPTIONS` → `204`) and switches auth cookies to **`SameSite=None; Secure`** so the browser sends them on cross-site requests. From the frontend, **all** REST/auth calls must use `credentials: "include"` (e.g. `fetch(url, { credentials: "include" })`).
- **HTTPS required for cross-origin:** `SameSite=None` cookies must be `Secure`, so a cross-origin frontend must talk to the server over **HTTPS** (browsers reject `SameSite=None` non-Secure cookies). For plain `http://localhost` development, prefer same-origin or serve over TLS.
- **Same origin (no CORS):** if `CORS_ALLOWED_ORIGINS` is empty, no CORS headers are sent and cookies stay `SameSite=Lax` — serve the frontend from the **same origin** as the Go server. (The `/ws` upgrade does not check Origin, so the socket itself works cross-port regardless — but `GET /api/me` and the auth cookie follow the policy above.)
- **Who am I:** call `GET /api/me` on load; `401` means "not logged in" → send the user to dev-login (local) or `GET /auth/google/login` (prod).
