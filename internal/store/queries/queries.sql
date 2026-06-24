-- name: UpsertUser :one
INSERT INTO users (google_sub, email, display_name, avatar_url)
VALUES ($1, $2, $3, $4)
ON CONFLICT (google_sub) DO UPDATE
SET email = EXCLUDED.email,
    avatar_url = EXCLUDED.avatar_url
RETURNING *;

-- name: EnsureRatings :exec
-- Seed the three category rows at the default rating if absent.
INSERT INTO ratings (user_id, category)
VALUES ($1, 'bullet'), ($1, 'blitz'), ($1, 'rapid')
ON CONFLICT (user_id, category) DO NOTHING;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: ListRatings :many
SELECT category, rating, games_played
FROM ratings
WHERE user_id = $1;

-- name: GetRating :one
SELECT rating, games_played
FROM ratings
WHERE user_id = $1 AND category = $2;

-- name: UpdateRating :exec
UPDATE ratings
SET rating = $3, games_played = games_played + 1
WHERE user_id = $1 AND category = $2;

-- name: UpdateDisplayName :one
UPDATE users SET display_name = $2 WHERE id = $1 RETURNING *;

-- name: InsertActiveGame :one
INSERT INTO games (white_id, black_id, category, base_seconds, increment_sec, status)
VALUES ($1, $2, $3, $4, $5, 'active')
RETURNING id;

-- name: FinishGame :exec
UPDATE games
SET status = 'finished',
    result = $2,
    result_reason = $3,
    pgn = $4,
    white_rating_before = $5,
    white_rating_after = $6,
    black_rating_before = $7,
    black_rating_after = $8,
    ended_at = $9
WHERE id = $1;

-- name: AbortGame :exec
UPDATE games
SET status = 'aborted',
    result = NULL,
    result_reason = $2,
    pgn = $3,
    ended_at = $4
WHERE id = $1;

-- name: Leaderboard :many
SELECT u.display_name, r.rating, r.games_played
FROM ratings r
JOIN users u ON u.id = r.user_id
WHERE r.category = $1
ORDER BY r.rating DESC, u.display_name ASC
LIMIT $2;

-- name: GamesForUser :many
SELECT *
FROM games
WHERE (white_id = $1 OR black_id = $1) AND status = 'finished'
ORDER BY ended_at DESC
LIMIT $2;

-- name: GameByID :one
SELECT * FROM games WHERE id = $1;

-- name: InsertGameMessage :one
INSERT INTO game_messages (game_id, sender_id, body)
VALUES ($1, $2, $3)
RETURNING id, created_at;

-- name: GameMessages :many
SELECT m.body, m.created_at, u.display_name AS sender_name, m.sender_id
FROM game_messages m
JOIN users u ON u.id = m.sender_id
WHERE m.game_id = $1 AND m.hidden = false
ORDER BY m.created_at;

-- name: AdminListUsers :many
SELECT id, email, display_name, created_at FROM users ORDER BY created_at DESC LIMIT $1;

-- name: AdminAllRatings :many
SELECT user_id, category, rating, games_played FROM ratings;

-- name: AdminSetRating :exec
UPDATE ratings SET rating = $3, games_played = $4 WHERE user_id = $1 AND category = $2;

-- name: AdminListGames :many
SELECT g.*, wu.display_name AS white_name, bu.display_name AS black_name
FROM games g
JOIN users wu ON wu.id = g.white_id
JOIN users bu ON bu.id = g.black_id
ORDER BY g.started_at DESC
LIMIT $1;

-- name: AdminGamesByUser :many
SELECT g.*, wu.display_name AS white_name, bu.display_name AS black_name
FROM games g
JOIN users wu ON wu.id = g.white_id
JOIN users bu ON bu.id = g.black_id
WHERE g.white_id = $1 OR g.black_id = $1
ORDER BY g.started_at DESC
LIMIT $2;

-- name: AdminSetGameVoided :exec
UPDATE games SET voided = $2 WHERE id = $1;

-- name: AdminSetMessageHidden :exec
UPDATE game_messages SET hidden = $2 WHERE id = $1;

-- name: AdminGameMessages :many
SELECT m.id, m.body, m.created_at, m.hidden, m.sender_id, u.display_name AS sender_name
FROM game_messages m
JOIN users u ON u.id = m.sender_id
WHERE m.game_id = $1
ORDER BY m.created_at;
