# Frontend Architecture & Existing Web App Analysis - Flame Chess

**Date**: 2026-08-04  
**Analyst**: Explorer 2 (Frontend Architecture & Existing Web App Analyst)  
**Target Files**: `web/index.html`, `web/web.go`  

---

## 1. Executive Summary & Architecture Overview

The existing frontend of **Flame Chess** is a single-page web application (SPA) delivered as a single embedded HTML file (`web/index.html`, ~404 KB) compiled into the Go backend binary via `//go:embed index.html` in `web/web.go`.

The application features a complete real-time multiplayer chess platform with:
- **Authentication**: Google OAuth2 (`/auth/google/login`) and fallback dev login (`/auth/dev-login`).
- **Real-Time Multiplayer**: WebSocket connection (`/ws`) for matchmaking, moves, clock sync, chat, draw/resign, rematch, and spectating.
- **Chess Engine & Rules**: Vendored `chess.js` (TypeScript/JS compilation output) embedded directly in `<script>` tags for rule validation, move generation, FEN handling, and SAN/PGN parsing.
- **Game Review & Stockfish**: Client-side Stockfish 10 (asm.js loaded via CDN Web Worker Blob shim) for move classification (`brilliant`, `great`, `best`, `excellent`, `good`, `inaccuracy`, `mistake`, `blunder`) and accuracy % calculation.
- **Interactive Board**: Pure DOM 8x8 grid with SVG piece definitions (`#p-wP`, `#p-bK`, etc.), drag-and-drop pointer handlers, legal move destination dots, rank/file coordinate labels, and chained premove queueing.
- **Clocks & Audio**: 100ms interval client-side clock countdown with active clock highlighting and embedded base64 audio SFX for move/capture/check/low-time alerts.

---

## 2. File Structure & Delivery

- **`web/web.go`**: Go source file containing `//go:embed index.html` and exposing `var Index []byte`.
- **`web/index.html`** (3,658 lines, 404,345 bytes): Monolithic single file comprising:
  1. **Lines 1–198**: `<style>` block containing root CSS variables, layout, panel styling, board grid styles, clock badges, and matrix rain loading screen CSS.
  2. **Lines 200–264**: HTML markup for `#dlLoader` (Dotslash matrix rain loading animation doors).
  3. **Lines 265–2268**: Vendored `chess.js` library bundled into `window.ChessLib`.
  4. **Lines 2271–3658**: Primary application JavaScript containing state, view router, auth, board rendering, drag-and-drop, WebSocket protocol handlers, API client functions, Stockfish analysis engine, chat, admin dashboard, audio SFX, and initialization logic.

---

## 3. Third-Party Libraries & Dependencies

| Library / Resource | Version / Source | Delivery Method | Usage in Application |
|---|---|---|---|
| **Google Fonts** | `Inter`, `JetBrains Mono` | External `<link rel="stylesheet">` CDN (`fonts.googleapis.com`) | Typography across UI, clocks, coordinates, ratings |
| **`chess.js`** | 0.13.4+ (Jeff Hlywa) | Embedded inline in `index.html` (`window.ChessLib`) | Rule validation, move generation, FEN string management, SAN/LAN conversion, PGN parsing |
| **Stockfish 10** | Stockfish 10.0.2 (`stockfish.asm.js`) | Loaded dynamically via CDN (`cdn.jsdelivr.net`) inside Web Worker Blob shim | Client-side game review engine evaluation, MultiPV score, move classification |
| **Chess Piece SVG Assets** | Custom SVG vectors | Embedded inline inside `<svg>` symbol definitions | SVG `<use>` references for 12 piece types (`p-wP`, `p-wN`, `p-wB`, `p-wR`, `p-wQ`, `p-wK`, `p-bP`, `p-bN`, `p-bB`, `p-bR`, `p-bQ`, `p-bK`) |
| **Audio SFX** | Base64 encoded MP3 data URIs | Embedded inline in JS (`window.SFX`) | Move sound (`SFX.move`), capture (`SFX.cap`), check (`SFX.check`), low-time warning (`SFX.low`), game start (`SFX.start`), offer (`SFX.offer`), game end (`SFX.boom`) |

---

## 4. DOM Structure & View Router

### Top Navigation Bar (`.topbar` / `#topbar`)
- Brand header (`FlameChess` gradient text)
- Navigation links (`.navlink[data-view]`):
  - `Lobby` (`data-view="lobby"`)
  - `Leaderboard` (`data-view="leaderboard"`)
  - `History` (`data-view="history"`)
  - `Admin` (`data-view="admin"`, visible only if `state.me.is_admin === true`)
- User Profile Dropdown (`#profileToggle`, `#profileDropdown`): User avatar, display name, logout button.

### Dynamic View Container (`#view-landing`, `#view-lobby`, `#view-game`, `#view-leaderboard`, `#view-history`, `#view-admin`)
The SPA uses a minimal client-side view router `showView(name)`:
```javascript
function showView(name) {
  state.view = name;
  ['landing','lobby','game','leaderboard','history','admin'].forEach(v =>
    $('#view-'+v).classList.toggle('hidden', v!==name));
  document.querySelectorAll('.navlink[data-view]').forEach(n =>
    n.classList.toggle('active', n.dataset.view===name));
  if(name==='landing') renderLanding();
  if(name==='lobby') renderLobby();
  if(name==='leaderboard') renderLeaderboard();
  if(name==='history') renderHistory();
  if(name==='admin') renderAdmin();
}
```

#### Detailed Breakdown of Views:

1. **Landing View (`#view-landing`)**:
   - Welcome card, "Login with Google" button.
   - Dev login block (`#devBlock`, auto-detected via `/auth/dev-login` GET endpoint).

2. **Lobby View (`#view-lobby`)**:
   - Time Control Selection Grid: Bullet (1+0), Blitz (3+2), Blitz (5+0), Rapid (10+0), plus custom base/increment inputs.
   - Action Buttons: `Find match` (`queue.join`), `Play a friend` (creates challenge link via `POST /api/challenges`).
   - Side Panels: User rating card, `ONLINE` users list (with direct challenge buttons), `TOP` leaderboard preview, `WATCH LIVE GAMES` list, `RECENT` game history preview.

3. **Game View (`#view-game`)**:
   - Top player card (opponent/spectator view) with avatar, name, rating chip, captured piece tray, material advantage indicator, and clock timer (`#clk-black` / `#clk-white`).
   - Board Wrapper (`.boardWrap` -> `#board` 8x8 CSS grid).
   - Bottom player card (user view).
   - Side Panel: Move notation history list (`#moveList`), last move label, move stepper navigation bar (`⏮ ◀ ▶ ⏭`), game review summary box (`#reviewSummary`), game control buttons (`Resign`, `Offer draw`), and game chat panel (`#chatPanel`).

4. **Leaderboard View (`#view-leaderboard`)**:
   - Category filter tabs (`Bullet`, `Blitz`, `Rapid`).
   - Ranked players table (#, Player, Rating, Games).

5. **History View (`#view-history`)**:
   - History list: List of past 50 games (result W/L/D, opponent name, rating delta `1500->1516`, category).
   - Game Review / Replay detail mode: Interactive board stepper (`openGameReview()`), game outcome header, move notation list, chat history replay, and "⚡ Review game" Stockfish analysis trigger.

6. **Admin Dashboard (`#view-admin`)**:
   - Authorized only for users with `is_admin: true`.
   - `Users & ratings` Tab: Editable table of display names and rating values across categories with inline save (`PATCH /api/admin/users/:id`, `PUT /api/admin/users/:id/ratings/:category`).
   - `Games & chat` Tab: List of past 100 games, inline game review button, void/un-void game button (`POST /api/admin/games/:id/void`), and message expander/hider (`POST /api/admin/messages/:id/hide`).

---

## 5. CSS Styling & Theme Architecture

- **Color Palette & CSS Variables**:
  ```css
  :root {
    --bg: #161513; --bg2: #211f1c; --panel: #262420; --panel2: #302d28;
    --txt: #ece8e1; --muted: #9d968a; --line: #ffffff14;
    --accent: #ff7a2f; --accent2: #ffb056; --ok: #86d98f; --bad: #ff8a8a; --warn: #e8c35b;
    --light: #e9edcf; --dark: #6d8c4a; --sel: #fbe06b4d; --lastmv: #ff7a2f40; --check: #ff4d4dcc;
    --radius: 14px; --shadow: 0 10px 30px #000a; --glow: 0 0 24px #ff7a2f33;
  }
  ```
- **Board Coordinates & Styling**:
  - Chess squares use `.sq.l` (light: `--light`) and `.sq.d` (dark: `--dark`).
  - Edge coordinates are co-located in square elements: `.rkco` (rank 1-8 top-left) and `.flco` (file a-h bottom-right).
  - Selected square: `.sq.sel::before` (`#fbe06b4d`).
  - Last move square: `.sq.last::before` (`#ff7a2f40`).
  - Premove square: `.sq.premove::before` (`#3aa0e055`).
  - King in check square: `.sq.check::after` (radial red glow `--check`).
- **Responsive Layout**:
  - `@media(max-width:760px)` collapses lobby grid into single column, scales `.boardWrap` to `92vw`, wraps topbar navigation.
- **Loader Animation (`#dlLoader`)**:
  - Matrix rain doors with canvas animation, pulsing heart logo, and splitting door transition.

---

## 6. JavaScript State Management

The application maintains state in a single global mutable object `state`:

```javascript
const state = {
  me: null,               // { uid, name, email, avatar, ratings, is_admin }
  ws: null,               // WebSocket instance
  view: 'landing',        // active view name
  online: 0,              // connected users count
  lbCategory: 'blitz',    // active leaderboard category
  game: { 
    id: null, color: null, fen: null, turn: 'white', 
    whiteMs: null, blackMs: null, opp: null, 
    lastMove: null, moves: [], over: false, drawFrom: null 
  },
  lastDelta: null,        // rating change after last game
  onlineUsers: [],        // array of { uid, name }
  incoming: {},           // incoming challenges by token
  pendingOut: null,       // outgoing challenge metadata
  linkToken: null,        // challenge link token from URL ?c=
  spectating: false,      // true if watching a game
  chat: [],               // array of { from_name, text, ts, mine }
  rematch: null,          // { game_id, from_name, sent: true/false }
  liveGames: [],          // list of active live games from server
  watchToken: null,       // game_id from URL ?watch=
  reconnecting: false,    // true during WS backoff retry
  deliberateClose: false, // true during logout
  review: {               // board-stepping state
    active: false, ply: 0, plies: [], moves: [], lastMoves: [] 
  },
  premove: [],            // chained array of { from, to, promo? }
  reviewEval: null,       // Stockfish eval/classification results
  histGame: null          // metadata for game opened in History
};
```

---

## 7. Chess Board Rendering & Move Input

### Board Rendering Logic (`renderBoard()`)
1. Reads `boardFen()`, which returns either the reviewed historical ply (`state.review.plies[state.review.ply]`) or the live position (`state.game.fen`).
2. If premoves are queued (`state.premove.length > 0`) during non-review mode, computes `projectedBoard()` to display projected piece positions.
3. Flips board perspective if `state.game.color === 'black'`.
4. Renders an 8x8 grid of square `div`s with CSS classes `.sq.l` or `.sq.d`.
5. Appends SVG piece element (`pieceEl(ch)`) referencing `<use href="#p-wP">`.
6. Computes legal destination squares for the selected square using `legalDestsFrom(sel)` (invoking `window.ChessLib.Chess(fen).moves({square, verbose: true})`) and appends `.dot` or `.dot.cap` indicators.
7. Displays rank numbers (`.rkco`) on the left column (`fi === 0`) and file letters (`.flco`) on the bottom row (`ri === 7`).

### Move Input & Drag-and-Drop (`onSquareDown()`)
- Unified `pointerdown`/`pointermove`/`pointerup` event handler attached to each square.
- **Drag-to-Move**: Dragging a piece >4px slides the piece smoothly across the DOM; on pointer release, computes target square `to` and triggers `doMove(from+to)` or `setPremove(from,to)`.
- **Tap-to-Move**: Tapping a piece selects it (`sel = sq`); tapping a legal target square executes the move.
- **Promotion Handling**: When a pawn reaches rank 8 (or rank 1), opens promotion overlay modal (`promptPromotion()`) for piece selection (`q`, `r`, `b`, `n`).
- **Premove Queueing**: While `state.game.turn !== state.game.color`, user inputs are appended to `state.premove` queue (max 6 steps). When `game.state` arrives from WebSocket and it becomes user's turn, `maybeRunPremove()` validates and executes the first queued premove automatically.
- **Right-Click & Escape**: Right-clicking the board or pressing `Escape` clears the premove queue (`clearPremove()`).

---

## 8. Clock Timer Implementation

- **Interval Loop (`startClock()`)**:
  - Runs a 100ms `setInterval` (`clockTimer`).
  - Measures elapsed time using `performance.now()`.
  - Decrements `state.game.whiteMs` or `state.game.blackMs` depending on `state.game.turn`.
  - Updates clock elements (`#clk-white`, `#clk-black`) formatted as `M:SS`.
- **Visual & Audio Alerts**:
  - Active turn clock gets class `.active` (orange gradient glow).
  - Time < 10 seconds gets class `.low` (red text) and triggers `playSound('low')` once per side.

---

## 9. WebSocket Integration (`/ws`)

### Protocol Connection
- Endpoint: `ws://<host>/ws` (or `wss://`)
- Connection initiated via `connectWS()` upon user login.
- **Reconnection Strategy**: Capped exponential backoff with random jitter (0.5s -> 1s -> 2s -> 5s max cap). Displays top banner `"Reconnecting…"`.
- **Single Session Enforcement**: Handles `session.replaced` WS event when user opens game in another tab, showing `"Game is open in another tab — tap to use it here"`.

### WebSocket Events Catalog

| Direction | Event `type` | Payload | Purpose / Frontend Action |
|---|---|---|---|
| **Client -> Server** | `queue.join` | `{ category, base, increment }` | Enter matchmaking queue |
| | `queue.leave` | `{}` | Exit matchmaking queue |
| | `move` | `{ game_id, uci }` | Submit chess move (e.g. `"e2e4"`) |
| | `resign` | `{ game_id }` | Resign current game |
| | `draw.offer` | `{ game_id }` | Offer draw to opponent |
| | `draw.respond` | `{ game_id, accept }` | Accept or decline draw offer |
| | `challenge.create_direct` | `{ opponent_id, base, increment }` | Send direct challenge to online user |
| | `challenge.accept` | `{ token }` | Accept incoming or link challenge |
| | `challenge.decline` | `{ token }` | Decline incoming challenge |
| | `spectate.join` | `{ game_id }` | Start spectating a game |
| | `spectate.leave` | `{}` | Stop spectating |
| | `chat.send` | `{ game_id, text }` | Send chat message in game |
| | `rematch.offer` | `{ game_id }` | Offer rematch after game over |
| | `rematch.respond` | `{ game_id, accept }` | Accept or decline rematch |
| **Server -> Client** | `online.count` | `{ n }` | Update online user count badge |
| | `online.list` | `{ users: [{ uid, name }] }` | Update online users list in lobby |
| | `queue.waiting` | `{}` | Show "searching for an opponent..." |
| | `game.start` | `{ game_id, color, fen, clocks, opponent, white, black }` | Switch view to `#view-game`, reset board, start clocks |
| | `game.state` | `{ fen, turn, white_ms, black_ms, last_move }` | Update board position, play move sound, execute premove |
| | `game.over` | `{ result, reason, rating_before, rating_after }` | Stop clock, play sound, open game over result overlay |
| | `draw.offered` | `{ from }` | Display draw offer banner/popup |
| | `challenge.incoming` | `{ token, from_name, base, increment, category }` | Display challenge toast notification |
| | `challenge.created` | `{ token }` | Save challenge token for pending toast |
| | `challenge.declined` | `{}` | Flash notice "They declined" |
| | `challenge.gone` | `{ token }` | Remove challenge toast |
| | `opponent.disconnected` | `{ grace_seconds }` | Show top banner "Opponent disconnected — Xs to reconnect" |
| | `opponent.reconnected` | `{}` | Clear opponent disconnect banner |
| | `rematch.offered` | `{ game_id, from_name }` | Show rematch offer prompt |
| | `rematch.declined` | `{}` | Flash notice "Opponent declined rematch" |
| | `chat.msg` | `{ from, from_name, text, ts }` | Append message to game chat list |
| | `games.live` | `{ games: [...] }` | Update live games watch panel in lobby |
| | `session.replaced` | `{}` | Close socket, show single-session banner |
| | `error` | `{ code, msg }` | Display error flash notice |

---

## 10. HTTP API Interactions

The client uses `api(path, opts)` wrapper around `fetch` with `credentials: 'include'`:

| Method | Endpoint Path | Description / Usage |
|---|---|---|
| `GET` | `/api/me` | Check session authentication, returns user profile & ratings |
| `GET` | `/auth/google/login` | Redirects to Google OAuth2 login flow |
| `GET` | `/auth/dev-login` | Detects if dev-login is enabled (returns 200 or 404) |
| `POST` | `/auth/dev-login` | Form data `name=Alice` to log in under dev mode |
| `POST` | `/auth/logout` | Terminates active session cookie |
| `POST` | `/api/challenges` | Body `{ base, increment }`, creates sharable challenge URL (`/index.html?c=<token>`) |
| `GET` | `/api/leaderboard` | Params `category` (`bullet`/`blitz`/`rapid`), `limit`, returns top ranked players |
| `GET` | `/api/games` | Params `limit`, returns user's recent game history list |
| `GET` | `/api/games/live` | Returns array of active live games for spectating |
| `GET` | `/api/games/:id` | Returns game detail (metadata, PGN, move timestamps, chat messages) |
| `GET` | `/api/admin/users` | (Admin) Returns all registered users and ratings |
| `PATCH` | `/api/admin/users/:id` | (Admin) Updates display name for user |
| `PUT` | `/api/admin/users/:id/ratings/:category` | (Admin) Body `{ rating, games_played }`, updates player rating |
| `GET` | `/api/admin/games` | (Admin) Returns all past games |
| `POST` | `/api/admin/games/:id/void` | (Admin) Voids a game and reverts rating changes |
| `GET` | `/api/admin/games/:id/messages` | (Admin) Returns chat messages for a game |
| `POST` | `/api/admin/messages/:id/hide` | (Admin) Toggles visibility of a chat message |

---

## 11. Additional Features

1. **Client-Side Game Review (Stockfish 10)**:
   - Initializes Stockfish 10 asm.js Web Worker via Blob shim.
   - Replays game plies, evaluates position scores (CP/mate), calculates centipawn loss (CPL), and derives Win-Probability-based accuracy % per side.
   - Assigns move quality badges: `brilliant` (`!!`), `great` (`!`), `best` (`★`), `excellent` (`✓`), `good` (`·`), `inaccuracy` (`?!`), `mistake` (`?`), `blunder` (`??`).

2. **Spectator Mode**:
   - Joined via `spectate.join` WS event or URL parameter `?watch=<game_id>`.
   - Read-only game board view (white perspective), live clocks, read-only chat list.

3. **In-Game Chat**:
   - Character limit 500.
   - Cleared on game start, supports historical replay when opening past games.

4. **Rematch System**:
   - One-tap rematch request and accept/decline workflow after game completion.

---

## 12. Key Recommendations & Architectural Considerations for UI Redesign

1. **Deconstruct Monolithic `index.html`**:
   - Extract vendored `chess.js` to an external script file or module import.
   - Separate CSS styles into modular stylesheets or Tailwind CSS classes.
   - Split JavaScript code into modular ES6 modules (`auth.js`, `board.js`, `websocket.js`, `clock.js`, `stockfish_review.js`, `views/`).

2. **UI Framework & State Management**:
   - The current UI relies on string template concatenation (`innerHTML = ...`) and manual DOM queries (`vq('#board')`).
   - Transitioning to a modern component-based framework (React, Vue, or Svelte) or a clean structured component system will prevent DOM re-render glitches (such as losing focus or re-animating DOM nodes).

3. **Chessboard UI & Touch Accessibility**:
   - The custom DOM-grid chessboard currently uses pointer events; adding smooth WebGL or CSS transform piece movement, piece animations, SVG piece themes, and mobile touch optimization will elevate the user experience.

4. **Asset Hosting**:
   - Stockfish CDN script (`cdn.jsdelivr.net`) and Google Fonts currently rely on external networks; in offline or restricted network environments, hosting local static assets will ensure reliable game review and font rendering.

---

*Report compiled by Explorer 2.*
