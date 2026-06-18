package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/store"
	"github.com/dotslash-flame/flame-chess/internal/ws"
)

func TestLeaderboardRanksByRatingDesc(t *testing.T) {
	st := newFakeStore()
	st.seedUser("u-1", "Alice", "a@flame.edu.in")
	st.seedUser("u-2", "Bob", "b@flame.edu.in")
	st.ratings["u-1"]["blitz"] = store.CategoryRating{Rating: 900, GamesPlayed: 5}
	st.ratings["u-2"]["blitz"] = store.CategoryRating{Rating: 1100, GamesPlayed: 8}

	req := httptest.NewRequest(http.MethodGet, "/api/leaderboard?category=blitz", nil)
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: signedFor("u-1")})
	rec := httptest.NewRecorder()

	leaderboardHandler(st, testSecret)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out []struct {
		Rank        int    `json:"rank"`
		DisplayName string `json:"display_name"`
		Rating      int    `json:"rating"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 2 || out[0].DisplayName != "Bob" || out[0].Rank != 1 || out[1].DisplayName != "Alice" {
		t.Errorf("unexpected leaderboard: %+v", out)
	}
}

func TestGameDetailReturnsGameAndMessages(t *testing.T) {
	st := newFakeStore()
	st.seedUser("u-1", "Alice", "a@flame.edu.in")
	st.gamesByID["g-1"] = store.GameRow{
		ID: "g-1", WhiteID: "u-1", BlackID: "u-2", Category: "blitz",
		Status: "finished", Result: "1-0", ResultReason: "checkmate", PGN: "1. e4",
	}
	st.messages["g-1"] = []store.ChatRow{
		{SenderID: "u-1", SenderName: "Alice", Body: "gg", CreatedAt: time.Unix(1, 0)},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/g-1", nil)
	req.SetPathValue("id", "g-1")
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: signedFor("u-1")})
	rec := httptest.NewRecorder()

	gameDetailHandler(st, testSecret)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out struct {
		Game     map[string]any `json:"game"`
		Messages []struct {
			SenderName string `json:"sender_name"`
			Body       string `json:"body"`
			TS         int64  `json:"ts"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Game["id"] != "g-1" || out.Game["result"] != "1-0" {
		t.Errorf("game = %+v, want g-1/1-0", out.Game)
	}
	if len(out.Messages) != 1 || out.Messages[0].Body != "gg" || out.Messages[0].SenderName != "Alice" {
		t.Errorf("messages = %+v, want one gg/Alice", out.Messages)
	}
}

func TestGameDetailUnknownIs404(t *testing.T) {
	st := newFakeStore()
	st.seedUser("u-1", "Alice", "a@flame.edu.in")
	req := httptest.NewRequest(http.MethodGet, "/api/games/nope", nil)
	req.SetPathValue("id", "nope")
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: signedFor("u-1")})
	rec := httptest.NewRecorder()

	gameDetailHandler(st, testSecret)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestLeaderboardRequiresAuth(t *testing.T) {
	rec := httptest.NewRecorder()
	leaderboardHandler(newFakeStore(), testSecret)(rec, httptest.NewRequest(http.MethodGet, "/api/leaderboard", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestGamesReturnsHistoryWithOpponentAndColor(t *testing.T) {
	st := newFakeStore()
	st.seedUser("u-1", "Alice", "a@flame.edu.in")
	st.seedUser("u-2", "Bob", "b@flame.edu.in")
	st.games["u-1"] = []store.GameRow{{
		ID:          "g-1",
		WhiteID:     "u-2",
		BlackID:     "u-1",
		Category:    "blitz",
		Status:      "finished",
		Result:      "0-1",
		BlackBefore: 800,
		BlackAfter:  820,
		EndedAt:     time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/games", nil)
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: signedFor("u-1")})
	rec := httptest.NewRecorder()

	gamesHandler(st, testSecret)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out []struct {
		ID           string `json:"id"`
		Opponent     string `json:"opponent"`
		Color        string `json:"color"`
		Result       string `json:"result"`
		RatingBefore int    `json:"rating_before"`
		RatingAfter  int    `json:"rating_after"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d games, want 1", len(out))
	}
	g := out[0]
	if g.Opponent != "Bob" || g.Color != "black" || g.RatingBefore != 800 || g.RatingAfter != 820 {
		t.Errorf("unexpected game: %+v", g)
	}
}
