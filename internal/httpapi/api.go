package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/store"
)

var validCategories = map[string]bool{"bullet": true, "blitz": true, "rapid": true}

const (
	defaultLimit = 50
	maxLimit     = 200
)

func parseLimit(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}

func leaderboardHandler(st Store, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := identityFrom(r, secret); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		category := r.URL.Query().Get("category")
		if !validCategories[category] {
			category = "blitz"
		}
		limit := parseLimit(r.URL.Query().Get("limit"))

		entries, err := st.Leaderboard(r.Context(), category, limit)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out := make([]map[string]any, len(entries))
		for i, e := range entries {
			out[i] = map[string]any{
				"rank":         i + 1,
				"display_name": e.DisplayName,
				"rating":       e.Rating,
				"games_played": e.GamesPlayed,
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func gamesHandler(st Store, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r, secret)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"))

		rows, err := st.GamesForUser(r.Context(), id.UserID, limit)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		names := opponentNames(r, st, id.UserID, rows)
		out := make([]map[string]any, len(rows))
		for i, g := range rows {
			color := "white"
			oppID := g.BlackID
			before, after := g.WhiteBefore, g.WhiteAfter
			if g.WhiteID != id.UserID {
				color = "black"
				oppID = g.WhiteID
				before, after = g.BlackBefore, g.BlackAfter
			}
			out[i] = map[string]any{
				"id":            g.ID,
				"opponent":      names[oppID],
				"color":         color,
				"result":        g.Result,
				"reason":        g.ResultReason,
				"category":      g.Category,
				"rating_before": before,
				"rating_after":  after,
				"ended_at":      g.EndedAt.UTC().Format(time.RFC3339),
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func opponentNames(r *http.Request, st Store, self string, rows []store.GameRow) map[string]string {
	names := map[string]string{}
	for _, g := range rows {
		opp := g.BlackID
		if g.WhiteID != self {
			opp = g.WhiteID
		}
		if _, done := names[opp]; done {
			continue
		}
		if me, err := st.GetMe(r.Context(), opp); err == nil {
			names[opp] = me.User.DisplayName
		} else {
			names[opp] = opp
		}
	}
	return names
}
