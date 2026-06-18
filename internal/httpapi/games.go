package httpapi

import (
	"net/http"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/hub"
	"github.com/dotslash-flame/flame-chess/internal/store"
)

func gameDetailHandler(st Store, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := identityFrom(r, secret); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id := r.PathValue("id")
		g, err := st.GameByID(r.Context(), id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		msgs, err := st.GameMessages(r.Context(), id)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out := map[string]any{
			"game": map[string]any{
				"id":         g.ID,
				"white_id":   g.WhiteID,
				"black_id":   g.BlackID,
				"category":   g.Category,
				"status":     g.Status,
				"result":     g.Result,
				"reason":     g.ResultReason,
				"pgn":        g.PGN,
				"started_at": g.StartedAt.UTC().Format(time.RFC3339),
				"ended_at":   g.EndedAt.UTC().Format(time.RFC3339),
			},
			"messages": chatJSON(msgs),
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func chatJSON(rows []store.ChatRow) []map[string]any {
	out := make([]map[string]any, len(rows))
	for i, m := range rows {
		out[i] = map[string]any{
			"sender_id":   m.SenderID,
			"sender_name": m.SenderName,
			"body":        m.Body,
			"ts":          m.CreatedAt.UnixMilli(),
		}
	}
	return out
}

func liveGamesHandler(h *hub.Hub, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := identityFrom(r, secret); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"games": h.LiveGames()})
	}
}
