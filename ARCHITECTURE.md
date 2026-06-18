FlameChess Architecture

Multiplayer-only online chess for Flame University. Go backend (single binary),
actor-per-game concurrency, Postgres persistence, Docker on the homeserver.


ai genreated components ON statUses

down_arrow here


Components
- internal/config — env-var configuration loader.
- cmd/server — entrypoint: load config, wire router, listen.
- internal/httpapi — HTTP routing (REST + /healthz)
- internal/auth — session cookies + Google OAuth + suffix gate.
- internal/game — per-game actor: board, clocks, move validation.
- internal/hub — matchmaking pools, presence, challenges.
- internal/ws — websocket layer.
- internal/store — Postgres access (pgx + sqlc).
- internal/rating — per-category Elo.
- internal/recorder — per-game seams: Recorder records finished games + applies Elo on game end; ChatRecorder (recorder/chat.go) best-effort persists in-game messages.
- internal/anticheat — (later, standalone) post-game analysis.
- migrations/ — goose SQL migrations.

Status
- [x] Phase 1: Skeleton — config, /healthz, schema, Docker compose.
- [x] Phase 2: Chess core (game package).
- [x] Phase 3: WS + Hub + quick-match (first playable).
- [x] Phase 4: Auth (Google OAuth + suffix gate).
- [x] Phase 5: Persistence & ratings.
- [x] Phase 6: Challenges.
- [x] Phase 7: Extras (chat, rematch, spectating).
- [ ] Phase 8: Test client.
- [ ] Phase 9: Anti-cheat (standalone).

Phase 7 notes (extras)
- Reconnect/abandon grace: the actor owns a per-game grace timer (`RECONNECT_GRACE_SECONDS`).
  On disconnect the clock keeps running and the opponent/spectators get `opponent.disconnected`;
  reconnect re-attaches the new socket via `Hub.handleRegister` → `Actor.Rejoin`. Grace expiry
  ends the game `abandoned` (new `Game.AbandonedBy`). All mutation stays on the actor goroutine —
  the grace timer fires into the `Run` select, never mutating from a timer goroutine.
- Rematch lives in the Hub (the actor exits on finish): a short-TTL `rematches` registry keyed on
  the just-finished game id (`REMATCH_TTL_SECONDS`, lazy expiry + sweep on game-end). Accepted
  rematches funnel through `startGameColors` with swapped colors, so the recorder/Elo seam applies.
- Chat is persisted: new `game_messages` table (migration `00002`), `ChatRecorder` seam
  (best-effort async insert), live relay to both players + spectators. Replay endpoint
  `GET /api/games/{id}` returns the game row + chat.
- Spectating: the actor holds a `spectators` set fanned out by `broadcast()`; the Hub keeps a
  `liveGames` registry (broadcast as `games.live` on start/end and via `GET /api/games/live`) and a
  `spectatorGame` map for disconnect cleanup.
