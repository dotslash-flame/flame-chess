package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/auth"
	"github.com/dotslash-flame/flame-chess/internal/config"
	"github.com/dotslash-flame/flame-chess/internal/store"
)

var adminCategories = map[string]bool{"bullet": true, "blitz": true, "rapid": true}

func requireAdmin(cfg *config.Config, secret string, fn func(auth.Identity, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r, secret)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !cfg.IsAdmin(id.Email) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		fn(id, w, r)
	}
}

func adminLimit(r *http.Request, def, max int) int {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func adminGameJSON(g store.AdminGameRow) map[string]any {
	return map[string]any{
		"id": g.ID, "white_id": g.WhiteID, "black_id": g.BlackID,
		"white": g.WhiteName, "black": g.BlackName,
		"category": g.Category, "status": g.Status, "result": g.Result, "reason": g.ResultReason,
		"white_before": g.WhiteBefore, "white_after": g.WhiteAfter,
		"black_before": g.BlackBefore, "black_after": g.BlackAfter,
		"voided":     g.Voided,
		"ended_at":   timeOrEmpty(g.EndedAt),
		"started_at": timeOrEmpty(g.StartedAt),
	}
}

func timeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func adminListUsersHandler(st Store) func(auth.Identity, http.ResponseWriter, *http.Request) {
	return func(_ auth.Identity, w http.ResponseWriter, r *http.Request) {
		users, err := st.AdminListUsersWithRatings(r.Context(), adminLimit(r, 200, 500))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out := make([]map[string]any, len(users))
		for i, u := range users {
			ratings := make(map[string]ratingDTO, len(u.Ratings))
			for cat, rr := range u.Ratings {
				ratings[cat] = ratingDTO{Rating: rr.Rating, GamesPlayed: rr.GamesPlayed}
			}
			out[i] = map[string]any{
				"id": u.ID, "email": u.Email, "display_name": u.DisplayName,
				"created_at": timeOrEmpty(u.CreatedAt), "ratings": ratings,
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func adminUpdateUserHandler(st Store) func(auth.Identity, http.ResponseWriter, *http.Request) {
	return func(_ auth.Identity, w http.ResponseWriter, r *http.Request) {
		var req struct {
			DisplayName string `json:"display_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		name, err := validateDisplayName(req.DisplayName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		u, err := st.UpdateDisplayName(r.Context(), r.PathValue("id"), name)
		if err != nil {
			if errors.Is(err, store.ErrDisplayNameTaken) {
				http.Error(w, "display name already taken", http.StatusConflict)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": u.ID, "display_name": u.DisplayName})
	}
}

func adminSetRatingHandler(st Store) func(auth.Identity, http.ResponseWriter, *http.Request) {
	return func(_ auth.Identity, w http.ResponseWriter, r *http.Request) {
		cat := r.PathValue("category")
		if !adminCategories[cat] {
			http.Error(w, "invalid category", http.StatusBadRequest)
			return
		}
		var req struct {
			Rating      int `json:"rating"`
			GamesPlayed int `json:"games_played"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if req.Rating < 0 || req.Rating > 4000 {
			http.Error(w, "rating out of range", http.StatusBadRequest)
			return
		}
		if err := st.AdminSetRating(r.Context(), r.PathValue("id"), cat, req.Rating, req.GamesPlayed); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func adminListGamesHandler(st Store) func(auth.Identity, http.ResponseWriter, *http.Request) {
	return func(_ auth.Identity, w http.ResponseWriter, r *http.Request) {
		limit := adminLimit(r, 100, 500)
		var (
			rows []store.AdminGameRow
			err  error
		)
		if user := r.URL.Query().Get("user"); user != "" {
			rows, err = st.AdminGamesByUser(r.Context(), user, limit)
		} else {
			rows, err = st.AdminListGames(r.Context(), limit)
		}
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out := make([]map[string]any, len(rows))
		for i, g := range rows {
			out[i] = adminGameJSON(g)
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func adminVoidGameHandler(st Store) func(auth.Identity, http.ResponseWriter, *http.Request) {
	return func(_ auth.Identity, w http.ResponseWriter, r *http.Request) {
		var req struct {
			Voided bool `json:"voided"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if err := st.AdminVoidGame(r.Context(), r.PathValue("id"), req.Voided); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func adminHideMessageHandler(st Store) func(auth.Identity, http.ResponseWriter, *http.Request) {
	return func(_ auth.Identity, w http.ResponseWriter, r *http.Request) {
		msgID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		var req struct {
			Hidden bool `json:"hidden"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if err := st.AdminSetMessageHidden(r.Context(), msgID, req.Hidden); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func adminGameMessagesHandler(st Store) func(auth.Identity, http.ResponseWriter, *http.Request) {
	return func(_ auth.Identity, w http.ResponseWriter, r *http.Request) {
		rows, err := st.AdminGameMessages(r.Context(), r.PathValue("id"))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out := make([]map[string]any, len(rows))
		for i, m := range rows {
			out[i] = map[string]any{
				"id": m.ID, "sender_id": m.SenderID, "sender_name": m.SenderName,
				"body": m.Body, "hidden": m.Hidden, "ts": m.CreatedAt.UnixMilli(),
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}
