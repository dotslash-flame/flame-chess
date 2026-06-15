package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode"

	"github.com/dotslash-flame/flame-chess/internal/auth"
	"github.com/dotslash-flame/flame-chess/internal/store"
	"github.com/dotslash-flame/flame-chess/internal/ws"
)

const oauthStateCookie = "fc_oauth_state"

const (
	displayNameMin = 1
	displayNameMax = 30
)

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

func identityFrom(r *http.Request, secret string) (auth.Identity, bool) {
	ck, err := r.Cookie(ws.SessionCookie)
	if err != nil {
		return auth.Identity{}, false
	}
	id, err := auth.Verify(ck.Value, secret)
	if err != nil {
		return auth.Identity{}, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type ratingDTO struct {
	Rating      int `json:"rating"`
	GamesPlayed int `json:"games_played"`
}

func meBody(me store.Me) map[string]any {
	ratings := make(map[string]ratingDTO, len(me.Ratings))
	for cat, r := range me.Ratings {
		ratings[cat] = ratingDTO{Rating: r.Rating, GamesPlayed: r.GamesPlayed}
	}
	return map[string]any{
		"uid":          me.User.ID,
		"email":        me.User.Email,
		"display_name": me.User.DisplayName,
		"ratings":      ratings,
	}
}

func meHandler(st Store, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r, secret)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		me, err := st.GetMe(r.Context(), id.UserID)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, meBody(me))
	}
}

func patchMeHandler(st Store, secret string, cp cookiePolicy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r, secret)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
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
		user, err := st.UpdateDisplayName(r.Context(), id.UserID, name)
		if err != nil {
			if errors.Is(err, store.ErrDisplayNameTaken) {
				http.Error(w, "display name already taken", http.StatusConflict)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		setSessionCookie(w, auth.Sign(auth.Identity{
			UserID:      user.ID,
			DisplayName: user.DisplayName,
			Email:       user.Email,
		}, secret), cp)

		me, err := st.GetMe(r.Context(), user.ID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, meBody(me))
	}
}

func validateDisplayName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if n := len([]rune(name)); n < displayNameMin || n > displayNameMax {
		return "", errors.New("display name must be 1–30 characters")
	}
	for _, c := range name {
		if unicode.IsLetter(c) || unicode.IsDigit(c) || strings.ContainsRune(" _-.", c) {
			continue
		}
		return "", errors.New("display name has invalid characters")
	}
	return name, nil
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

func googleCallbackHandler(oauth *auth.GoogleOAuth, st Store, secret, redirect string, cp cookiePolicy) http.HandlerFunc {
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

		dbUser, err := persistLogin(r, st, user.Sub, user.Email, user.Name, user.Picture)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		setSessionCookie(w, auth.Sign(auth.Identity{
			UserID:      dbUser.ID,
			DisplayName: dbUser.DisplayName,
			Email:       dbUser.Email,
		}, secret), cp)
		http.Redirect(w, r, redirect, http.StatusFound)
	}
}

func persistLogin(r *http.Request, st Store, sub, email, name, avatar string) (store.User, error) {
	u, err := st.UpsertUser(r.Context(), sub, email, name, avatar)
	if err != nil {
		return store.User{}, err
	}
	if err := st.EnsureRatings(r.Context(), u.ID); err != nil {
		return store.User{}, err
	}
	return u, nil
}
