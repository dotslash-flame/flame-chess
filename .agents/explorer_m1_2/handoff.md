# Handoff Report - Explorer 2 (Frontend Architecture & Existing Web App Analyst)

**Working Directory**: `/Users/hriday/Documents/Flame Chess/.agents/explorer_m1_2`  
**Date**: 2026-08-04  
**Handoff Type**: Hard (Task complete)  

---

## 1. Observation

- **Backend Embedding**: `web/web.go` lines 1–7 embeds `web/index.html` using `//go:embed index.html` into `var Index []byte`.
- **Frontend File**: `web/index.html` is a 3,658-line monolithic single-page application (404,345 bytes).
- **Embedded Dependencies**:
  - **`chess.js`**: Inline TypeScript/JS compilation output of `chess.js` bundled from lines 265 to 2268, exposed globally as `window.ChessLib`.
  - **Stockfish 10**: Loaded via dynamic Web Worker Blob shim targeting `https://cdn.jsdelivr.net/npm/stockfish@10.0.2/src/stockfish.asm.js` at line 2870.
  - **Piece Symbols**: Inline SVG symbols defined for 12 piece variants (`p-wP`, `p-wN`, `p-wB`, `p-wR`, `p-wQ`, `p-wK`, `p-bP`, `p-bN`, `p-bB`, `p-bR`, `p-bQ`, `p-bK`).
  - **Audio SFX**: Base64 encoded MP3 strings in `window.SFX` (`boom`, `shotgun`, `cap`, `move`, `check`, `low`, `start`, `offer`).
- **DOM Layout & Views**:
  - Top navigation bar (`.topbar`) with brand and view routing buttons.
  - Views (`#view-landing`, `#view-lobby`, `#view-game`, `#view-leaderboard`, `#view-history`, `#view-admin`) managed via `showView(name)` at line 2300.
- **State Management**: Global `state` object at lines 2277–2296 tracking `me`, `ws`, `view`, `online`, `game`, `spectating`, `chat`, `rematch`, `liveGames`, `review`, `premove`, `reviewEval`, `histGame`.
- **Chessboard & Movement**:
  - Pure DOM 8x8 grid inside `#board` container rendered by `renderBoard()` (line 2423).
  - Drag-and-drop pointer tracking in `onSquareDown()` (line 2516).
  - Chained premove queueing (`state.premove`) validated via `projectedBoard()` and executed via `maybeRunPremove()`.
- **Clock Timer**: 100ms interval countdown (`startClock()`, line 3204) tracking `whiteMs` and `blackMs` with active turn glow and low-time audio alert.
- **WebSocket Protocol**: `/ws` socket managed via `connectWS()` (line 2564) and `handleWS()` (line 2594), implementing exponential backoff reconnection and session replacement handling.
- **HTTP APIs**: Uses `api(path, opts)` wrapper to call `/api/me`, `/auth/google/login`, `/auth/dev-login`, `/api/challenges`, `/api/leaderboard`, `/api/games`, `/api/games/live`, `/api/games/:id`, `/api/admin/users`, `/api/admin/games`, etc.

---

## 2. Logic Chain

1. **Observation**: `web/web.go` embeds `web/index.html`, and `web/index.html` contains all CSS, HTML, vendored JS libraries (`chess.js`), Stockfish 10 integration, audio SFX, WebSocket logic, board drag-and-drop, and view routing in a single file.
   - **Reasoning**: The current frontend architecture is a fully self-contained monolithic SPA embedded directly into the Go binary. No external build step (Vite, Webpack, npm) is used in the repository.

2. **Observation**: State management relies on a single global `state` object, and view rendering relies on string interpolation (`innerHTML = ...`) coupled with query selectors (`$()`, `vq()`).
   - **Reasoning**: While functional and zero-dependency, this architecture leads to complete DOM teardown and recreation during view transitions or full board re-renders. Component state and DOM elements are tightly coupled.

3. **Observation**: Real-time communication is handled entirely via WebSocket (`/ws`) messages, while authentication, challenge creation, leaderboards, and game history rely on standard REST endpoints (`/api/...`).
   - **Reasoning**: The interface contract between the frontend and backend is cleanly divided: real-time gameplay and interactive events flow through `/ws`, while persistent data queries and auth flow through `/api/...` and `/auth/...`.

---

## 3. Caveats

- **No Caveats**: All frontend code and embedded resources within `web/index.html` were completely inspected and documented. No external or hidden frontend assets exist outside `web/index.html` and `web/web.go`.

---

## 4. Conclusion

The existing Flame Chess web application is a feature-complete single-page application contained entirely within `web/index.html`. It implements a full suite of chess features including real-time play, spectating, premoving, clock timers, game review with Stockfish 10, chat, rematching, leaderboards, game history, and admin management.

For the Flame Chess UI redesign, the primary architectural goals will be:
1. Modularizing `web/index.html` into structured component files and stylesheets.
2. Abstracting state management away from global mutable variables into a clean reactive state store or component hierarchy.
3. Preserving full compatibility with all existing `/ws` WebSocket protocol messages and `/api/...` REST endpoints.

---

## 5. Verification Method

To verify these findings independently:
1. Inspect `web/web.go` to confirm the `//go:embed index.html` directive.
2. Inspect `web/index.html` at the following key line ranges:
   - **Styles**: Lines 1–198
   - **Vendored `chess.js`**: Lines 265–2268
   - **State & Router**: Lines 2277–2324
   - **Board Render & Drag**: Lines 2423–2561
   - **WebSocket Handlers**: Lines 2564–2620
   - **Lobby & Matchmaking**: Lines 2660–2768
   - **Stockfish Game Review**: Lines 2868–2984
   - **Clock & Sound**: Lines 3203–3222
   - **History & Replay**: Lines 3402–3482
   - **Admin Dashboard**: Lines 3485–3563
3. Read the detailed breakdown report in `.agents/explorer_m1_2/frontend_analysis.md`.
