package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/store/db"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping DB integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run store integration tests")
	}
	ctx := context.Background()
	s, err := New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(s.Close)
	resetSchema(t, s)
	return s
}

func resetSchema(t *testing.T, s *Store) {
	t.Helper()
	raw, err := os.ReadFile("../../migrations/00001_init.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	up := string(raw)
	if i := strings.Index(up, "-- +goose Down"); i >= 0 {
		up = up[:i]
	}
	ddl := "DROP TABLE IF EXISTS games, ratings, users CASCADE;\n" + up
	if _, err := s.pool.Exec(context.Background(), ddl); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
}

func makeUser(t *testing.T, s *Store, sub, email, name string) User {
	t.Helper()
	u, err := s.UpsertUser(context.Background(), sub, email, name, "")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	if err := s.EnsureRatings(context.Background(), u.ID); err != nil {
		t.Fatalf("EnsureRatings: %v", err)
	}
	return u
}

func TestUpsertUserIdempotentAndPreservesName(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	u1, err := s.UpsertUser(ctx, "dev:alice", "alice@dev.local", "Alice", "")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if _, err := s.UpdateDisplayName(ctx, u1.ID, "AliceTheGreat"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	u2, err := s.UpsertUser(ctx, "dev:alice", "alice@dev.local", "Alice", "")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if u2.ID != u1.ID {
		t.Fatalf("upsert produced new id %q, want %q", u2.ID, u1.ID)
	}
	if u2.DisplayName != "AliceTheGreat" {
		t.Fatalf("display name clobbered on re-login: got %q", u2.DisplayName)
	}
}

func TestEnsureRatingsSeedsThreeCategories(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	u := makeUser(t, s, "dev:bob", "bob@dev.local", "Bob")

	me, err := s.GetMe(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	for _, cat := range []string{"bullet", "blitz", "rapid"} {
		r, ok := me.Ratings[cat]
		if !ok {
			t.Fatalf("missing rating for %s", cat)
		}
		if r.Rating != 800 || r.GamesPlayed != 0 {
			t.Fatalf("%s = %+v, want {800 0}", cat, r)
		}
	}
}

func TestFinishAndRateUpdatesBothPlayers(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	white := makeUser(t, s, "dev:white", "white@dev.local", "White")
	black := makeUser(t, s, "dev:black", "black@dev.local", "Black")

	gameID, err := s.InsertActiveGame(ctx, white.ID, black.ID, "blitz", 300, 0)
	if err != nil {
		t.Fatalf("InsertActiveGame: %v", err)
	}

	err = s.FinishAndRate(ctx, FinishParams{
		GameID:      gameID,
		Category:    "blitz",
		Result:      "1-0",
		Reason:      "checkmate",
		PGN:         "1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7#",
		WhiteID:     white.ID,
		BlackID:     black.ID,
		WhiteBefore: 800,
		WhiteAfter:  820,
		BlackBefore: 800,
		BlackAfter:  780,
		EndedAt:     time.Now(),
	})
	if err != nil {
		t.Fatalf("FinishAndRate: %v", err)
	}

	wr, _ := s.GetRating(ctx, white.ID, "blitz")
	if wr.Rating != 820 || wr.GamesPlayed != 1 {
		t.Fatalf("white rating = %+v, want {820 1}", wr)
	}
	br, _ := s.GetRating(ctx, black.ID, "blitz")
	if br.Rating != 780 || br.GamesPlayed != 1 {
		t.Fatalf("black rating = %+v, want {780 1}", br)
	}

	games, err := s.GamesForUser(ctx, white.ID, 10)
	if err != nil {
		t.Fatalf("GamesForUser: %v", err)
	}
	if len(games) != 1 || games[0].Status != "finished" || games[0].Result != "1-0" {
		t.Fatalf("games = %+v, want one finished 1-0", games)
	}
}

func TestLeaderboardOrdersByRatingDesc(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	low := makeUser(t, s, "dev:low", "low@dev.local", "Low")
	high := makeUser(t, s, "dev:high", "high@dev.local", "High")

	if err := s.q.UpdateRating(ctx, db.UpdateRatingParams{UserID: high.ID, Category: "blitz", Rating: 1200}); err != nil {
		t.Fatalf("bump high: %v", err)
	}
	if err := s.q.UpdateRating(ctx, db.UpdateRatingParams{UserID: low.ID, Category: "blitz", Rating: 900}); err != nil {
		t.Fatalf("bump low: %v", err)
	}

	board, err := s.Leaderboard(ctx, "blitz", 50)
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}
	if len(board) < 2 || board[0].DisplayName != "High" || board[1].DisplayName != "Low" {
		t.Fatalf("leaderboard order = %+v, want High then Low", board)
	}
}

func TestUpdateDisplayNameUniqueConflict(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	makeUser(t, s, "dev:carol", "carol@dev.local", "Carol")
	dave := makeUser(t, s, "dev:dave", "dave@dev.local", "Dave")

	_, err := s.UpdateDisplayName(ctx, dave.ID, "Carol")
	if !errors.Is(err, ErrDisplayNameTaken) {
		t.Fatalf("got %v, want ErrDisplayNameTaken", err)
	}
}
