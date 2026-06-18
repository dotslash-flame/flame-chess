-- +goose Up
CREATE TABLE game_messages (
  id          bigserial PRIMARY KEY,
  game_id     uuid NOT NULL REFERENCES games(id) ON DELETE CASCADE,
  sender_id   uuid NOT NULL REFERENCES users(id),
  body        text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_game_messages_game ON game_messages (game_id, created_at);

-- +goose Down
DROP TABLE game_messages;
