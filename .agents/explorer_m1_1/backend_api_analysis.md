# Flame Chess — Backend API & Integration Analysis

**Author:** Explorer 1 (Backend API & Integration Analyst)  
**Date:** 2026-08-03  
**Target Repository:** `/Users/hriday/Documents/Flame Chess`  
**Go Version:** 1.26.3 (`go.mod`)  

---

## 1. Executive Summary & Architecture Overview

Flame Chess is a real-time, multiplayer-only online chess application tailored for Flame University. The backend is built as a single Go binary with PostgreSQL persistence, actor-per-game concurrency, and real-time state synchronization over WebSockets.

### Core Architectural Layers & Packages:
- **`cmd/server/main.go`**: Server entrypoint. Loads configuration, connects to PostgreSQL pool, initializes matchmaking `Hub`, sets up HTTP router, and handles graceful shutdown.
- **`internal/config`**: Environment variable configuration parser (`Config` struct) with default fallback values and validator logic.
- **`internal/auth`**: Session HMAC signing/verification (`session.go`), Google OAuth 2.0 integration, display name sanitization, and email domain suffix validation (`google.go`).
- **`internal/httpapi`**: Standard HTTP router (`net/http.ServeMux`), REST handler implementations (`auth.go`, `api.go`, `games.go`, `challenges.go`, `admin.go`), and CORS middleware (`cors.go`).
- **`internal/ws`**: WebSocket transport layer (`ws.go`) built on `github.com/coder/websocket`. Translates JSON text frames into `Hub` commands and handles connection pumps.
- **`internal/wire`**: Wire protocol definitions (`wire.go`), JSON payload structs, message type constants, and error codes.
- **`internal/hub`**: Matchmaking engine, presence tracking, player-to-player challenges, live games lobby registry, and rematch tracking (`hub.go`).
- **`internal/game`**: Per-game concurrency actor (`actor.go`), core chess engine adapter (`game.go`), clock management (`clock.go`), and game categories (`category.go`).
- **`internal/store`**: PostgreSQL persistence via `pgx/v5` and `sqlc`-generated queries (`store.go`, `db/`).
- **`internal/recorder`**: Seam implementations for recording finished game results, applying Elo rating changes, and persisting in-game chat messages.
- **`web/web.go`**: Static asset embedding using Go `//go:embed index.html`.

---

## 2. Authentication & Session Management

### 2.1 Session Cookie Protocol (`fc_session`)
- Identity is carried by an HTTP-only cookie named **`fc_session`**.
- The cookie value consists of a base64url-encoded JSON payload concatenated with a hex-encoded SHA-256 HMAC signature separated by a period:  
  `<base64url(Identity)>.<hex(HMAC-SHA256(payload, SESSION_HMAC_SECRET))>`
- **`Identity` Schema (`internal/auth/session.go`):**
  ```json
  {
    "uid": "1f2e3d4c-5b6a-7f8e-9d0c-1b2a3f4e5d6c",
    "name": "Alice",
    "email": "alice@flame.edu.in"
  }
  ```
  - `uid`: User UUID in PostgreSQL (`users.id`).
  - `name`: Current display name stored in the database.
  - `email`: User email address (`<name>@dev.local` for dev login).

### 2.2 Cookie Policy & CORS Behavior (`internal/httpapi/router.go`, `cors.go`)
- **Same-Origin Mode (default when `CORS_ALLOWED_ORIGINS` is empty):**
  - Cookie attributes: `Path=/`, `HttpOnly`, `SameSite=Lax`, `Secure` (true if `DEV_LOGIN=false`).
- **Cross-Origin Mode (when `CORS_ALLOWED_ORIGINS` contains domain origins):**
  - Cookie attributes: `Path=/`, `HttpOnly`, `SameSite=None`, `Secure` (always true).
  - CORS Headers returned:
    - `Access-Control-Allow-Origin: <Request Origin>` (echoed if in allowlist)
    - `Access-Control-Allow-Credentials: true`
    - `Access-Control-Allow-Methods: GET, POST, PATCH, OPTIONS`
    - `Access-Control-Allow-Headers: Content-Type` (or requested headers)
    - `Vary: Origin`
  - Preflight `OPTIONS` requests return `204 No Content`.
  - **Frontend Requirement:** Frontend must pass `{ credentials: "include" }` on all `fetch` requests.

### 2.3 Authentication Workflows & Endpoints

#### 1. Dev Bypass Login (`POST /auth/dev-login`)
- **Availability:** Active only when `DEV_LOGIN=true` (default: `true`). In production (`DEV_LOGIN=false`), this route is omitted (returns `404`).
- **Request:** `application/x-www-form-urlencoded`
  - Field `name` (optional string, defaults to `"anon"`).
- **Behavior:**
  1. Upserts a user with synthetic Google Sub `dev:<name>`, email `<name>@dev.local`, and display name `<name>`.
  2. Ensures default rating records (bullet, blitz, rapid) exist in `user_ratings`.
  3. Signs and sets the `fc_session` cookie.
- **Response:** `200 OK` with JSON `Identity` object.

#### 2. Google OAuth 2.0 Login (`GET /auth/google/login`)
- **Availability:** Registered when `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and `GOOGLE_REDIRECT_URL` are configured.
- **Behavior:**
  1. Generates a 32-byte secure random state token.
  2. Sets a short-lived state cookie `fc_oauth_state` (`MaxAge: 300s`, `HttpOnly`, `Path=/`).
  3. Returns `302 Found` redirecting to Google's consent screen (`https://accounts.google.com/o/oauth2/v2/auth`) asking for scopes `openid`, `email`, `profile`.

#### 3. Google OAuth Callback (`GET /auth/google/callback`)
- **Query Parameters:** `code` (string), `state` (string).
- **Validation:**
  1. Validates `state` against `fc_oauth_state` cookie (`400 Bad Request` if mismatch/missing).
  2. Exchanges `code` for Google OAuth token and fetches OpenID user info.
  3. Applies domain suffix gate (`AllowedEmail`): checks `email_verified == true` and `email` ends with `@<ALLOWED_EMAIL_SUFFIX>` (default `flame.edu.in`).
- **Result:**
  - **Allowed:** Upserts user in PostgreSQL (`users`), sets `fc_session` cookie, and returns `302 Found` redirecting to `APP_REDIRECT_URL` (default `/`).
  - **Rejected:** Returns `403 Forbidden` with HTML page `"Flame accounts only"`.

#### 4. Logout (`POST /auth/logout`)
- **Behavior:** Overwrites `fc_session` cookie with empty value and `MaxAge: -1`.
- **Response:** `204 No Content`.

---

## 3. Static File Serving Mechanism

Static files are served directly from the Go binary via Go's standard `embed` feature (`web/web.go`):

```go
package web

import _ "embed"

//go:embed index.html
var Index []byte
```

### HTTP Route Handling (`internal/httpapi/router.go`):
- **Route:** `GET /{$}` (matches root path `/` specifically in Go 1.22+ `net/http` router).
- **Handler:**
  ```go
  mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
      w.Header().Set("Content-Type", "text/html; charset=utf-8")
      _, _ = w.Write(web.Index)
  })
  ```
- **Asset Bundle:** `web/index.html` is a single self-contained Single Page Application (SPA) (~404 KB) bundling the entire HTML structure, CSS styles, and client JS.

---

## 4. HTTP REST API Specification

All `/api/*` endpoints require cookie authentication via `fc_session` (returns `401 Unauthorized` if cookie is missing/invalid or user UUID does not exist).

### 4.1 Liveness Health Check
- **`GET /healthz`**
  - **Auth Required:** No.
  - **Response `200 OK`:**
    ```json
    { "status": "ok" }
    ```

### 4.2 User Profile & Account Management
- **`GET /api/me`**
  - **Auth Required:** Yes.
  - **Response `200 OK`:**
    ```json
    {
      "uid": "1f2e3d4c-5b6a-7f8e-9d0c-1b2a3f4e5d6c",
      "email": "alice@flame.edu.in",
      "display_name": "Alice",
      "avatar_url": "",
      "is_admin": false,
      "ratings": {
        "bullet": { "rating": 800, "games_played": 0 },
        "blitz":  { "rating": 816, "games_played": 3 },
        "rapid":  { "rating": 800, "games_played": 0 }
      }
    }
    ```

- **`PATCH /api/me`**
  - **Auth Required:** Yes.
  - **Request Body:**
    ```json
    { "display_name": "Alice_V2" }
    ```
  - **Validation Rules:** Trimmed length 1–30 characters; allowed characters: letters, digits, spaces, `_`, `-`, `.`.
  - **Side Effect:** Updates display name in DB, re-signs and updates `fc_session` cookie on response.
  - **Response `200 OK`:** Returns updated profile object (same as `GET /api/me`).
  - **Error Responses:**
    - `400 Bad Request`: Invalid length or forbidden characters.
    - `409 Conflict`: Display name is already taken (`display_name already taken`).

### 4.3 Leaderboards & Game History
- **`GET /api/leaderboard`**
  - **Query Parameters:**
    - `category` (optional string, `bullet` | `blitz` | `rapid`, default: `blitz`).
    - `limit` (optional int, 1–200, default: `50`).
  - **Response `200 OK`:** Array of top players ordered by rating descending:
    ```json
    [
      {
        "rank": 1,
        "display_name": "Bob",
        "rating": 1240,
        "games_played": 15
      }
    ]
    ```

- **`GET /api/games`**
  - **Query Parameters:**
    - `limit` (optional int, 1–200, default: `50`).
  - **Response `200 OK`:** Array of caller's finished games (newest first):
    ```json
    [
      {
        "id": "game-uuid",
        "opponent": "Bob",
        "color": "white",
        "result": "1-0",
        "reason": "checkmate",
        "category": "blitz",
        "rating_before": 800,
        "rating_after": 816,
        "ended_at": "2026-08-03T18:30:00Z"
      }
    ]
    ```

- **`GET /api/games/{id}`**
  - **Path Parameters:** `id` (game UUID string).
  - **Response `200 OK`:** Game metadata and full persisted chat history:
    ```json
    {
      "game": {
        "id": "game-uuid",
        "white_id": "user-uuid-1",
        "black_id": "user-uuid-2",
        "category": "blitz",
        "status": "finished",
        "result": "1-0",
        "reason": "checkmate",
        "pgn": "1. e4 e5 2. Nf3 Nc6 ...",
        "started_at": "2026-08-03T18:20:00Z",
        "ended_at": "2026-08-03T18:25:00Z"
      },
      "messages": [
        {
          "sender_id": "user-uuid-1",
          "sender_name": "Alice",
          "body": "Good game!",
          "ts": 1770143100000
        }
      ]
    }
    ```
  - **Error Responses:** `404 Not Found` if game ID does not exist.

- **`GET /api/games/live`**
  - **Response `200 OK`:** List of currently active games:
    ```json
    {
      "games": [
        {
          "game_id": "game-uuid",
          "white": "Alice",
          "black": "Bob",
          "category": "blitz",
          "base": 300,
          "increment": 0
        }
      ]
    }
    ```

### 4.4 Challenge Links
- **`POST /api/challenges`**
  - **Request Body:**
    ```json
    {
      "base": 300,
      "increment": 0
    }
    ```
  - **Response `200 OK`:** Returns generated single-use challenge token and invite URL:
    ```json
    {
      "token": "a1b2c3d4e5f6",
      "url": "http://localhost:8080/?c=a1b2c3d4e5f6"
    }
    ```

### 4.5 Admin REST API
Requires caller's email to be listed in `ADMIN_EMAILS` config (returns `403 Forbidden` otherwise).

- **`GET /api/admin/users`**: List users & ratings (Query: `limit`, default 200, max 500).
- **`PATCH /api/admin/users/{id}`**: Admin rename user (Body: `{"display_name":"..."}`).
- **`PUT /api/admin/users/{id}/ratings/{category}`**: Manually override rating/games played (Body: `{"rating":int, "games_played":int}`). Returns `204`.
- **`GET /api/admin/games`**: List games (Query: `limit`, `user`).
- **`POST /api/admin/games/{id}/void`**: Toggle void status of game (Body: `{"voided":bool}`). Returns `204`.
- **`GET /api/admin/games/{id}/messages`**: View game chat including hidden messages.
- **`POST /api/admin/messages/{id}/hide`**: Hide/unhide chat message (Body: `{"hidden":bool}`). Returns `204`.

---

## 5. WebSocket Protocol Specification

### 5.1 Connection Lifecycle
- **Endpoint:** `GET /ws`
- **Authentication:** Handled during HTTP upgrade request via `fc_session` cookie (`401 Unauthorized` if invalid).
- **Origin Handling:** `InsecureSkipVerify: true` (Origin header is not restricted for WS connections).
- **Single Connection Invariant:** Connecting a new socket for an existing `uid` automatically closes the older socket (sending `session.replaced` event to the old socket).

### 5.2 Frame Format
All WebSocket frames are text JSON with a `"type"` string discriminator:
```json
{
  "type": "<event_name>",
  ...fields
}
```

### 5.3 Client → Server Messages

| Event (`type`) | Payload Fields | Description & Rules |
|---|---|---|
| `queue.join` | `category` (string), `base` (int sec), `increment` (int sec) | Join matchmaking pool. Category derives from `base` (e.g. `< 180s`: bullet, `< 600s`: blitz, `>= 600s`: rapid). Pool matching requires exact `(base, increment)` match. |
| `queue.leave` | — | Cancel queue searching. |
| `move` | `game_id` (string), `uci` (string, e.g. `"e2e4"`, `"e7e8q"`) | Propose move in UCI notation. |
| `resign` | `game_id` (string) | Resign active game. |
| `draw.offer` | `game_id` (string) | Send draw offer to opponent. |
| `draw.respond` | `game_id` (string), `accept` (bool) | Respond to pending draw offer. `accept: true` ends game in `1/2-1/2` `draw_agreed`. |
| `ping` | — | Heartbeat. Server replies immediately with `pong`. |
| `challenge.create_direct` | `opponent_id` (string uid), `base` (int), `increment` (int) | Challenge a specific connected user. |
| `challenge.accept` | `token` (string) | Accept direct or link challenge. Triggers `game.start` for both. |
| `challenge.decline` | `token` (string) | Decline direct challenge. Creator receives `challenge.declined`. |
| `challenge.cancel` | `token` (string) | Cancel created challenge. Target receives `challenge.gone`. |
| `rematch.offer` | `game_id` (string) | Offer rematch after game finish. Valid for 60s (`REMATCH_TTL_SECONDS`). |
| `rematch.respond` | `game_id` (string), `accept` (bool) | Accept/Decline rematch. Accept starts game with swapped colors. |
| `chat.send` | `game_id` (string), `text` (string) | Send chat message (max 500 runes, truncated). |
| `spectate.join` | `game_id` (string) | Join as spectator for live game. Receives snapshot then live stream. |
| `spectate.leave` | — | Stop spectating. |

### 5.4 Server → Client Messages

| Event (`type`) | Payload Fields | Trigger / Context |
|---|---|---|
| `online.count` | `n` (int) | Broadcast on connect/disconnect. |
| `online.list` | `users` (`[{"uid":"...","name":"..."}]`) | Broadcast on connect/disconnect. |
| `queue.waiting` | — | Sent to client when queued without immediate match. |
| `game.start` | `game_id` (str), `color` (`"white"|"black"|"spectator"`), `opponent` (str), `clocks` (`{white_ms, black_ms}`), `fen` (str), `white` (str, spectator only), `black` (str, spectator only) | Match created or spectator/reconnect snapshot. |
| `game.state` | `game_id` (str), `fen` (str), `last_move` (str), `white_ms` (int64), `black_ms` (int64), `turn` (`"white"|"black"`) | Broadcast after every accepted move. |
| `game.over` | `game_id` (str), `result` (`"1-0"|"0-1"|"1/2-1/2"`), `reason` (str), `ratings` (optional `{white:{before,after,delta}, black:{...}}`) | Game termination event. Reasons: `checkmate`, `stalemate`, `insufficient`, `threefold`, `fifty_move`, `resign`, `draw_agreed`, `timeout`, `abandoned`. |
| `draw.offered` | `game_id` (str), `from` (uid str) | Delivered to opponent when draw offered. |
| `challenge.incoming` | `token` (str), `from` (str), `from_name` (str), `base` (int), `increment` (int), `category` (str) | Received direct challenge. |
| `challenge.created` | `token` (str), `url` (str) | Challenge registration ack. |
| `challenge.declined` | `token` (str) | Direct challenge declined. |
| `challenge.gone` | `token` (str) | Received challenge withdrawn/cancelled. |
| `opponent.disconnected` | `color` (`"white"|"black"`), `grace_seconds` (int) | Opponent socket closed mid-game. |
| `opponent.reconnected` | `color` (`"white"|"black"`) | Opponent socket reconnected mid-game within grace window. |
| `rematch.offered` | `game_id` (str), `from` (str), `from_name` (str) | Opponent offered rematch. |
| `rematch.declined` | `game_id` (str) | Rematch offer declined or expired. |
| `chat.msg` | `game_id` (str), `from` (str), `from_name` (str), `text` (str), `ts` (int64 unix millis) | Relayed in-game chat message. |
| `games.live` | `games` (`[{"game_id":"...","white":"...","black":"...","category":"...","base":int,"increment":int}]`) | Live games list update broadcast. |
| `pong` | — | Response to `ping`. |
| `session.replaced` | — | Sent to old socket when new socket connects for same user. |
| `error` | `code` (str), `msg` (str) | Action rejection notice. |

### 5.5 Error Codes
- `bad_message`: Malformed frame or JSON unmarshal error.
- `illegal_move`: Invalid move in current position.
- `not_your_turn`: Attempted move out of turn.
- `not_in_game`: In-game command sent when not in active game.
- `game_not_active`: Action performed on finished game.
- `unknown_game`: Game ID mismatch or game not found.
- `unknown_challenge`: Challenge token missing, expired, or invalid.
- `challenge_self`: Self-challenge forbidden.
- `busy`: One or both players are already in a game.
- `opponent_offline`: Target user disconnected.
- `rematch_unavailable`: Rematch offer expired (60s limit) or unavailable.

---

## 6. Local Backend Setup & Execution Guide

### 6.1 Configuration Environment Variables
Defined and managed in `internal/config/config.go` and `.env.example`:

| Environment Variable | Required? | Default Value | Description |
|---|---|---|---|
| `PORT` | No | `8080` | HTTP & WebSocket server port. |
| `DATABASE_URL` | **Yes** | — | PostgreSQL connection string (e.g. `postgres://flame:flame@localhost:5432/flamechess?sslmode=disable`). |
| `SESSION_HMAC_SECRET` | **Yes** | — | HMAC signing secret for session cookies. |
| `DEV_LOGIN` | No | `true` | Enables `POST /auth/dev-login` bypass route. Must be `false` in prod. |
| `ALLOWED_EMAIL_SUFFIX` | No | `flame.edu.in` | Required email domain suffix for Google OAuth gate. |
| `APP_REDIRECT_URL` | No | `/` | Post-OAuth login redirect target. |
| `CORS_ALLOWED_ORIGINS` | No | `""` | Comma-separated allowlist of frontend origins (e.g. `http://localhost:5173`). |
| `RECONNECT_GRACE_SECONDS`| No | `30` | Grace period (seconds) before disconnect turns into game abandonment. |
| `REMATCH_TTL_SECONDS` | No | `60` | Rematch offer validity duration in seconds. |
| `STARTING_RATING` | No | `800` | Initial rating assigned to new players across all categories. |
| `GOOGLE_CLIENT_ID` | Conditional | — | Google OAuth Client ID (Required if `DEV_LOGIN=false`). |
| `GOOGLE_CLIENT_SECRET` | Conditional | — | Google OAuth Client Secret (Required if `DEV_LOGIN=false`). |
| `GOOGLE_REDIRECT_URL` | Conditional | `http://localhost:8080/auth/google/callback` | OAuth callback URL (Required if `DEV_LOGIN=false`). |
| `ADMIN_EMAILS` | No | `""` | Comma-separated list of admin email addresses. |

### 6.2 Database Dependency & Docker Setup
The backend requires a PostgreSQL database (version 16+). Database schema migrations are located under `migrations/` and use `goose`.

#### Method 1: Using Docker Compose (Recommended for local dev)
1. Start PostgreSQL container:
   ```bash
   docker compose up -d db
   ```
2. Run database migrations (or start whole docker stack):
   ```bash
   docker compose up --build
   ```

#### Method 2: Running Go Server directly on Host
1. Ensure PostgreSQL is running on `localhost:5432` with database `flamechess`, user `flame`, password `flame`.
2. Apply migrations in `migrations/` using `goose` or direct SQL script.
3. Export environment variables or create a `.env` file:
   ```bash
   export PORT=8080
   export DATABASE_URL="postgres://flame:flame@localhost:5432/flamechess?sslmode=disable"
   export SESSION_HMAC_SECRET="local-dev-secret-key"
   export DEV_LOGIN="true"
   ```
4. Build and start server:
   ```bash
   go run ./cmd/server
   ```
5. Test liveness and local auth:
   ```bash
   curl http://localhost:8080/healthz
   curl -i -X POST -d 'name=Alice' http://localhost:8080/auth/dev-login
   ```
