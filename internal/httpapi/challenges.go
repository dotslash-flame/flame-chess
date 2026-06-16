package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/dotslash-flame/flame-chess/internal/hub"
)

func challengesHandler(h *hub.Hub, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r, secret)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req struct {
			Base      int `json:"base"`
			Increment int `json:"increment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		token := h.CreateChallenge(id.UserID, id.DisplayName, "", req.Base, req.Increment)
		url := requestScheme(r) + "://" + r.Host + "/?c=" + token
		writeJSON(w, http.StatusOK, map[string]string{"token": token, "url": url})
	}
}

func requestScheme(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
