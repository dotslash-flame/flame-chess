package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/dotslash-flame/flame-chess/internal/auth"
	"github.com/dotslash-flame/flame-chess/internal/ws"
)

const oauthStateCookie = "fc_oauth_state"

type cookiePolicy struct {
	secure   bool
	sameSite http.SameSite
}

func setSessionCookie(w http.ResponseWriter, value string, cp cookiePolicy) {
	http.SetCookie(w, &http.Cookie{
		Name:     ws.SessionCookie,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   cp.secure,
		SameSite: cp.sameSite,
	})
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func meHandler(secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ck, err := r.Cookie(ws.SessionCookie)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := auth.Verify(ck.Value, secret)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"uid":          id.UserID,
			"email":        id.Email,
			"display_name": id.DisplayName,
		})
	}
}

func logoutHandler(cp cookiePolicy) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     ws.SessionCookie,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   cp.secure,
			SameSite: cp.sameSite,
			MaxAge:   -1,
		})
		w.WriteHeader(http.StatusNoContent)
	}
}

func googleLoginHandler(oauth *auth.GoogleOAuth, cp cookiePolicy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := randomState()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     oauthStateCookie,
			Value:    state,
			Path:     "/",
			HttpOnly: true,
			Secure:   cp.secure,
			SameSite: cp.sameSite,
			MaxAge:   300,
		})
		http.Redirect(w, r, oauth.LoginURL(state), http.StatusFound)
	}
}

func googleCallbackHandler(oauth *auth.GoogleOAuth, secret, redirect string, cp cookiePolicy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		ck, err := r.Cookie(oauthStateCookie)
		if err != nil || state == "" || ck.Value != state {
			http.Error(w, "invalid oauth state", http.StatusBadRequest)
			return
		}

		user, err := oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "oauth exchange failed", http.StatusBadGateway)
			return
		}

		if !auth.AllowedEmail(user.Email, user.EmailVerified, oauth.Suffix()) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("<!doctype html><title>Flame accounts only</title><h1>Flame accounts only</h1><p>Sign in with your Flame University Google account.</p>"))
			return
		}

		id := auth.Identity{
			UserID:      auth.UserIDForSub(user.Sub),
			DisplayName: user.Name,
			Email:       user.Email,
		}
		setSessionCookie(w, auth.Sign(id, secret), cp)
		http.Redirect(w, r, redirect, http.StatusFound)
	}
}
