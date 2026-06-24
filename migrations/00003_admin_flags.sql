-- +goose Up
-- Reversible admin overlays: void a game (annul its rating effect) and hide a chat message.
ALTER TABLE games ADD COLUMN voided boolean NOT NULL DEFAULT false;
ALTER TABLE game_messages ADD COLUMN hidden boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE games DROP COLUMN voided;
ALTER TABLE game_messages DROP COLUMN hidden;
