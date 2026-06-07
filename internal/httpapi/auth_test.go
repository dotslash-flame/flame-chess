package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotslash-flame/flame-chess/internal/auth"
	"github.com/dotslash-flame/flame-chess/internal/ws"
)

func TestMeReturnsIdentityForValidCookie(t *testing.T) {
	id := auth.Identity{UserID: "u-1", DisplayName: "Alice", Email: "alice@flame.edu.in"}
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: auth.Sign(id, testSecret)})
	rec := httptest.NewRecorder()

	meHandler(testSecret)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["uid"] != "u-1" || body["email"] != "alice@flame.edu.in" || body["display_name"] != "Alice" {
		t.Errorf("unexpected body: %+v", body)
	}
}

func TestMeRejectsMissingCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	meHandler(testSecret)(rec, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMeRejectsGarbageCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: "garbage.deadbeef"})
	rec := httptest.NewRecorder()
	meHandler(testSecret)(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	logoutHandler(cookiePolicy{})(rec, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	var ck *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == ws.SessionCookie {
			ck = c
		}
	}
	if ck == nil {
		t.Fatal("logout did not set a session cookie to clear")
	}
	if ck.MaxAge >= 0 && ck.Value != "" {
		t.Errorf("cookie not cleared: MaxAge=%d value=%q", ck.MaxAge, ck.Value)
	}
}

func TestGoogleCallbackRejectsMissingState(t *testing.T) {
	oauth := auth.NewGoogleOAuth("cid", "secret", "http://localhost/cb", "flame.edu.in")
	rec := httptest.NewRecorder()
	googleCallbackHandler(oauth, testSecret, "/", cookiePolicy{})(rec, httptest.NewRequest(http.MethodGet, "/auth/google/callback", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestGoogleCallbackRejectsMismatchedState(t *testing.T) {
	oauth := auth.NewGoogleOAuth("cid", "secret", "http://localhost/cb", "flame.edu.in")
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "different"})
	rec := httptest.NewRecorder()
	googleCallbackHandler(oauth, testSecret, "/", cookiePolicy{})(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestGoogleLoginRedirectsAndSetsState(t *testing.T) {
	oauth := auth.NewGoogleOAuth("my-client-id", "secret", "http://localhost/cb", "flame.edu.in")
	rec := httptest.NewRecorder()
	googleLoginHandler(oauth, cookiePolicy{})(rec, httptest.NewRequest(http.MethodGet, "/auth/google/login", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "accounts.google.com") || !strings.Contains(loc, "my-client-id") {
		t.Errorf("unexpected redirect location: %s", loc)
	}

	var state *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == oauthStateCookie {
			state = c
		}
	}
	if state == nil || state.Value == "" {
		t.Fatal("login did not set fc_oauth_state cookie")
	}
	if !strings.Contains(loc, "state="+state.Value) {
		t.Errorf("redirect state does not match cookie: loc=%s cookie=%s", loc, state.Value)
	}
}
