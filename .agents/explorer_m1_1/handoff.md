# Handoff Report — Explorer 1 (Backend API & Integration Analyst)

**Working Directory:** `/Users/hriday/Documents/Flame Chess/.agents/explorer_m1_1`  
**Target Repository:** `/Users/hriday/Documents/Flame Chess`  
**Date:** 2026-08-03  
**Status:** Complete (Hard Handoff)  

---

## 1. Observation

Direct code inspection of the Go backend architecture yielded the following key observations:

1. **Static Asset Embedding:**
   - File: `web/web.go`, lines 1–7:
     ```go
     package web

     import _ "embed"

     //go:embed index.html
     var Index []byte
     ```
   - File: `internal/httpapi/router.go`, lines 23–26:
     ```go
     mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
         w.Header().Set("Content-Type", "text/html; charset=utf-8")
         _, _ = w.Write(web.Index)
     })
     ```
   - Single-page application bundle `web/index.html` is embedded directly into the Go binary and served at `GET /`.

2. **Authentication & Session Cookies:**
   - File: `internal/auth/session.go`, lines 14–27, 29–50:
     - Defines `Identity` struct (`uid`, `name`, `email`).
     - Uses `HMAC-SHA256` signature: `payload.mac(payload, secret)`.
   - File: `internal/httpapi/auth.go`, lines 66–80, 178–232:
     - `POST /auth/dev-login` (lines 66–80): Gated by `DEV_LOGIN=true` (config line 51 in `internal/config/config.go`). Accepts `name` form field, calls `persistLogin(..., "dev:"+name, name+"@dev.local", name, "")`, sets `fc_session` cookie, and returns `Identity` JSON.
     - `GET /auth/google/login` (lines 178–196): Sets `fc_oauth_state` state cookie and redirects 302 to Google consent screen.
     - `GET /auth/google/callback` (lines 198–232): Validates OAuth state, exchanges code, enforces `AllowedEmail` domain suffix gate (`@<ALLOWED_EMAIL_SUFFIX>`, default `flame.edu.in`), sets `fc_session` cookie, and redirects to `APP_REDIRECT_URL`.
     - `POST /auth/logout` (lines 163–176): Sets `fc_session` cookie with `MaxAge: -1` and returns `204 No Content`.

3. **HTTP REST API Endpoints:**
   - File: `internal/httpapi/router.go`, lines 22–57:
     - `GET /{$}`: Serves embedded `index.html`.
     - `GET /healthz`: Returns `{"status":"ok"}` (`200 OK`).
     - `POST /auth/dev-login`: Dev authentication (`DEV_LOGIN=true`).
     - `GET /auth/google/login` & `GET /auth/google/callback`: Google OAuth 2.0 flow.
     - `POST /auth/logout`: Logout session reset (`204`).
     - `GET /api/me`: Returns profile and rating map (`200 OK`).
     - `PATCH /api/me`: Renames user (`display_name` body), re-signs `fc_session` cookie, returns updated profile (`200 OK`).
     - `GET /api/leaderboard`: Leaderboard query (`category`, `limit`).
     - `GET /api/games`: User game history (`limit`).
     - `GET /api/games/{id}`: Single game details and chat history.
     - `GET /api/games/live`: Live games list.
     - `POST /api/challenges`: Challenge link creation (`base`, `increment`).
     - `GET /api/admin/users`, `PATCH /api/admin/users/{id}`, `PUT /api/admin/users/{id}/ratings/{category}`, `GET /api/admin/games`, `POST /api/admin/games/{id}/void`, `GET /api/admin/games/{id}/messages`, `POST /api/admin/messages/{id}/hide`: Admin management endpoints (gated by `ADMIN_EMAILS`).

4. **WebSocket Protocol & Events (`GET /ws`):**
   - File: `internal/ws/ws.go`, lines 214–232:
     - Upgrades HTTP to WebSocket (checks `fc_session` cookie, skips origin verify in dev).
   - File: `internal/wire/wire.go`, lines 7–46, 83–384:
     - Defines Client -> Server events: `queue.join`, `queue.leave`, `move`, `resign`, `draw.offer`, `draw.respond`, `ping`, `challenge.create_direct`, `challenge.accept`, `challenge.decline`, `challenge.cancel`, `rematch.offer`, `rematch.respond`, `chat.send`, `spectate.join`, `spectate.leave`.
     - Defines Server -> Client events: `online.count`, `online.list`, `queue.waiting`, `game.start`, `game.state`, `game.over`, `draw.offered`, `challenge.incoming`, `challenge.created`, `challenge.declined`, `challenge.gone`, `opponent.disconnected`, `opponent.reconnected`, `rematch.offered`, `rematch.declined`, `chat.msg`, `games.live`, `pong`, `session.replaced`, `error`.

5. **Local Environment & Database Setup:**
   - File: `internal/config/config.go`, lines 40–85:
     - Requires `DATABASE_URL` and `SESSION_HMAC_SECRET`.
     - Optional variables: `PORT` (8080), `DEV_LOGIN` (true), `ALLOWED_EMAIL_SUFFIX` (flame.edu.in), `APP_REDIRECT_URL` (/), `CORS_ALLOWED_ORIGINS`, `RECONNECT_GRACE_SECONDS` (30), `REMATCH_TTL_SECONDS` (60), `STARTING_RATING` (800), `ADMIN_EMAILS`.
   - File: `docker-compose.yml`, lines 1–35:
     - Defines `db` service running `postgres:16-alpine` on port `5432:5432` with user `flame`, database `flamechess`, password `flame`.

---

## 2. Logic Chain

1. **Authentication Verification:**
   - *Observation:* `internal/auth/session.go` signs `Identity` into `fc_session` cookie; `internal/httpapi/router.go` attaches `meHandler` and `ws.Handler` which both invoke `identityFrom(r, secret)`.
   - *Reasoning:* Authentication is strictly cookie-based (`fc_session`). There are no Bearer token headers.
   - *Deduction:* Any frontend client (SPA or test client) must issue credentialed HTTP requests (`credentials: "include"`) and allow cookie storage across WebSocket upgrades.

2. **Static Asset Integration:**
   - *Observation:* `web/web.go` embeds `index.html` via `//go:embed index.html`, served at `GET /{$}`.
   - *Reasoning:* The Go server operates as a standalone single binary hosting both API backend and frontend SPA assets.
   - *Deduction:* UI redesign deliverables can either compile directly into `web/index.html` or run via a separate Vite/React dev server configured with CORS (`CORS_ALLOWED_ORIGOrigins`).

3. **REST & Real-Time Protocol Division:**
   - *Observation:* Game history, profile management, leaderboards, admin functions, and challenge link generation use HTTP REST. Matchmaking, live move propagation, clock updates, chat relay, presence, challenges, rematches, and spectating use WebSocket (`/ws`).
   - *Reasoning:* REST handles persistent query/mutation operations; WS handles low-latency state synchronization.
   - *Deduction:* Real-time UI components (chess board, clocks, move list, active chat, online player list) must bind directly to WS event streams.

4. **Local Development Setup:**
   - *Observation:* `DEV_LOGIN` defaults to `true`, creating `/auth/dev-login`. `docker-compose.yml` provides PostgreSQL on `localhost:5432`.
   - *Reasoning:* Developers can run `docker compose up -d db` and `go run ./cmd/server` to develop locally without requiring live Google OAuth credentials or cloud infrastructure.

---

## 3. Caveats

- **No bearer token / header auth:** Token-based header auth (`Authorization: Bearer ...`) is not implemented; all client calls rely on cookie transport.
- **Single connection rule per user:** Connecting a new WebSocket for the same `uid` closes the previous socket. Multi-tab browsing on the same user account will trigger `session.replaced` and disconnect earlier tabs.
- **CORS & HTTPS requirement:** If cross-origin frontend development is used with `CORS_ALLOWED_ORIGINS`, cookies switch to `SameSite=None; Secure`, requiring TLS/HTTPS in non-localhost browser environments.

---

## 4. Conclusion

The Go backend architecture for Flame Chess is robust, self-contained, and fully specified:
- **Authentication:** HMAC-signed `fc_session` cookie with Google OAuth 2.0 (`@flame.edu.in`) for production and `/auth/dev-login` bypass for local development.
- **Static Asset Serving:** Embedded single-page bundle (`web/index.html`) served at `GET /`.
- **API & Protocol:** 13 user REST endpoints, 7 admin REST endpoints, and 34 WebSocket message types covering matchmaking, gameplay, challenges, rematches, chat, spectating, and presence.
- **Local Dev:** Quick startup using `docker compose up -d db`, `export SESSION_HMAC_SECRET=dev-secret DATABASE_URL=postgres://flame:flame@localhost:5432/flamechess?sslmode=disable`, and `go run ./cmd/server`.

All findings are documented in detail in `backend_api_analysis.md`.

---

## 5. Verification Method

1. **Inspect Analysis Files:**
   - Verify `backend_api_analysis.md` exists in `/Users/hriday/Documents/Flame Chess/.agents/explorer_m1_1/backend_api_analysis.md`.
2. **Verify Code Locations:**
   - Inspect `cmd/server/main.go`, `internal/config/config.go`, `internal/auth/session.go`, `internal/auth/google.go`, `internal/httpapi/router.go`, `internal/ws/ws.go`, `internal/wire/wire.go`, `internal/hub/hub.go`, `internal/game/actor.go`, `web/web.go`.
3. **Verify Local Build Command:**
   - Run `go check` / build check:
     ```bash
     go test ./...
     ```
