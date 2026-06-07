package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/dotslash-flame/flame-chess/internal/auth"
	"github.com/dotslash-flame/flame-chess/internal/config"
	"github.com/dotslash-flame/flame-chess/internal/hub"
	"github.com/dotslash-flame/flame-chess/internal/ws"
	"github.com/dotslash-flame/flame-chess/web"
)

func NewRouter(h *hub.Hub, cfg *config.Config) http.Handler {
	secret := cfg.SessionHMACSecret
	crossOrigin := len(cfg.CORSAllowedOrigins) > 0
	cp := cookiePolicy{secure: crossOrigin || !cfg.DevLogin, sameSite: http.SameSiteLaxMode}
	if crossOrigin {
		cp.sameSite = http.SameSiteNoneMode
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(web.Index)
	})
	mux.HandleFunc("GET /healthz", healthz)

	if cfg.DevLogin {
		mux.HandleFunc("POST /auth/dev-login", devLoginHandler(secret, cp))
	}

	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" && cfg.GoogleRedirectURL != "" {
		oauth := auth.NewGoogleOAuth(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL, cfg.AllowedEmailSuffix)
		mux.HandleFunc("GET /auth/google/login", googleLoginHandler(oauth, cp))
		mux.HandleFunc("GET /auth/google/callback", googleCallbackHandler(oauth, secret, cfg.PostLoginRedirect, cp))
	}

	mux.HandleFunc("POST /auth/logout", logoutHandler(cp))
	mux.HandleFunc("GET /api/me", meHandler(secret))
	mux.Handle("GET /ws", ws.Handler(h, secret))
	return corsMiddleware(cfg.CORSAllowedOrigins, mux)
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func devLoginHandler(secret string, cp cookiePolicy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.FormValue("name")
		if name == "" {
			name = "anon"
		}
		id := auth.Identity{UserID: auth.UserIDForName(name), DisplayName: name}
		setSessionCookie(w, auth.Sign(id, secret), cp)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(id)
	}
}
