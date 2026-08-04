# Project: Flame Chess UI Redesign

## Architecture
- Single Go binary backend (`cmd/server`) serving REST API, WebSockets (`/ws`), and static frontend assets embedded via `web/web.go` (`//go:embed index.html` or compiled static bundle).
- Backend tech stack (UNTOUCHED): Go 1.22, gorilla/websocket, pgx, sqlc, PostgreSQL.
- Frontend tech stack: Modernized responsive single-page application with CSS custom properties (light & dark mode design tokens), Lucide icons, chess.js, interactive chessboard, Web Workers (Stockfish 10 asm.js), and Playwright verification test suite.
- Key constraint: The Go backend (`cmd/`, `internal/`) MUST NOT BE MODIFIED. The frontend must integrate seamlessly with existing HTTP endpoints and WebSocket protocol.

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| 1 | Exploration & Technical Design | Backend API & frontend structure & mockup analysis | None | DONE |
| 2 | E2E Test Suite & Harness | Playwright screenshot framework & test suite (Tiers 1-4) | M1 | DONE |
| 3 | Design System & UI Shell | CSS tokens, Theme Toggle (Light/Dark), Navbar, Sidebar | M1 | DONE |
| 4 | Login & Auth UI | Modern login screen (Light & Dark mode variants) | M3 | DONE |
| 5 | Dashboard & Lobby UI | Dashboard, quick match, challenge list, leaderboards | M3 | DONE |
| 6 | Gameplay UI | Interactive chess board, clocks, move history, turn state | M3 | DONE |
| 7 | Game Over Modal & Social | Game over modal (Light & Dark mode), chat, spectating | M6 | DONE |
| 8 | E2E Test Pass & Audit | Launch dev server, Playwright screenshots, mockup audit | M2, M4, M5, M6, M7 | IN_PROGRESS |

## Interface Contracts
### Backend REST API (`internal/httpapi`)
- `GET /{$}` -> Serves `web/index.html`
- `GET /healthz` -> `{ "status": "ok" }`
- `POST /auth/dev-login` -> `{ "identity": { "id": "...", "name": "...", "email": "..." } }`
- `GET /auth/google/login` & `GET /auth/google/callback` -> OAuth flow
- `POST /auth/logout` -> Reset session cookie (`fc_session`)
- `GET /api/me` -> User profile & rating map
- `PATCH /api/me` -> Update display name
- `GET /api/leaderboard` -> Leaderboard list (`category`, `limit`)
- `GET /api/games` -> User game history
- `GET /api/games/{id}` -> Game details & chat history
- `GET /api/games/live` -> Live games list
- `POST /api/challenges` -> Challenge link creation

### Backend WebSocket Protocol (`internal/ws`)
- `WS /ws` -> Session-cookie authenticated WebSocket protocol supporting `queue.join`, `move`, `chat.send`, `spectate.join`, `rematch.offer`, etc.

## Code Layout
- Backend (UNTOUCHED): `cmd/server/main.go`, `internal/`
- Frontend: `web/index.html`, `web/web.go`
- Test & Verification: `e2e/`, `e2e/screenshots/`
