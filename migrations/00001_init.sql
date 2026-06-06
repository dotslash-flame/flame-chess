-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub    text UNIQUE NOT NULL,
    email         text UNIQUE NOT NULL,
    display_name  text UNIQUE NOT NULL,
    avatar_url    text,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ratings (
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category      text NOT NULL CHECK (category IN ('bullet','blitz','rapid')),
    rating        int  NOT NULL DEFAULT 800,
    games_played  int  NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, category)
);

CREATE TABLE games (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    white_id             uuid NOT NULL REFERENCES users(id),
    black_id             uuid NOT NULL REFERENCES users(id),
    category             text NOT NULL CHECK (category IN ('bullet','blitz','rapid')),
    base_seconds         int  NOT NULL,
    increment_sec        int  NOT NULL,
    status               text NOT NULL CHECK (status IN ('active','finished','aborted')),
    result               text CHECK (result IN ('1-0','0-1','1/2-1/2')),
    result_reason        text,
    pgn                  text,
    white_rating_before  int,
    white_rating_after   int,
    black_rating_before  int,
    black_rating_after   int,
    started_at           timestamptz NOT NULL DEFAULT now(),
    ended_at             timestamptz
);

CREATE INDEX idx_games_white ON games(white_id);
CREATE INDEX idx_games_black ON games(black_id);
CREATE INDEX idx_ratings_category_rating ON ratings(category, rating DESC);

-- +goose Down
DROP TABLE games;
DROP TABLE ratings;
DROP TABLE users;
