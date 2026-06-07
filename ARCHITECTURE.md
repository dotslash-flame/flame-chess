FlameChess Architecture

Multiplayer-only online chess for Flame University. Go backend (single binary),
actor-per-game concurrency, Postgres persistence, Docker on the homeserver.


ai genreated components ON statUses

down_arrow here


Components
- internal/config — env-var configuration loader.
- cmd/server — entrypoint: load config, wire router, listen.
- internal/httpapi — HTTP routing (REST + /healthz)
- internal/game — (later) per-game actor: board, clocks, move validation.
- internal/hub — (later) matchmaking pools, presence, challenges.
- internal/ws — (later) websocket layer.
- internal/store — (later) Postgres access (pgx + sqlc).
- internal/rating — (later) per-category Elo.
- internal/anticheat — (later, standalone) post-game analysis.
- migrations/ — goose SQL migrations.

Status
- [x] Phase 1: Skeleton — config, /healthz, schema, Docker compose.
- [x] Phase 2: Chess core (game package).
- [x] Phase 3: WS + Hub + quick-match (first playable).
- [ ] Phase 4: Auth (Google OAuth + suffix gate).
- [ ] Phase 5: Persistence & ratings.
- [ ] Phase 6: Challenges.
- [ ] Phase 7: Extras (chat, rematch, spectating).
- [ ] Phase 8: Test client.
- [ ] Phase 9: Anti-cheat (standalone).
