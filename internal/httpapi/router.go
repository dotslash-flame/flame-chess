package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/dotslash-flame/flame-chess/internal/auth"
	"github.com/dotslash-flame/flame-chess/internal/hub"
	"github.com/dotslash-flame/flame-chess/internal/ws"
	"github.com/dotslash-flame/flame-chess/web"
)

func NewRouter(h *hub.Hub, secret string, devLogin bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(web.Index)
	})
	mux.HandleFunc("GET /healthz", healthz)
	if devLogin {
		mux.HandleFunc("POST /auth/dev-login", devLoginHandler(secret))
	}
	mux.Handle("GET /ws", ws.Handler(h, secret))
	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func devLoginHandler(secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.FormValue("name")
		if name == "" {
			name = "anon"
		}
		id := auth.Identity{UserID: auth.UserIDForName(name), DisplayName: name}
		http.SetCookie(w, &http.Cookie{
			Name:     ws.SessionCookie,
			Value:    auth.Sign(id, secret),
			Path:     "/",
			HttpOnly: true,
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(id)
	}
}
