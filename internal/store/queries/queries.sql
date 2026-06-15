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
