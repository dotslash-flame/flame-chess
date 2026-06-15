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

type meResponse struct {
	UID         string                                       `json:"uid"`
	Email       string                                       `json:"email"`
	DisplayName string                                       `json:"display_name"`
	Ratings     map[string]struct{ Rating, GamesPlayed int } `json:"ratings"`
}

func signedFor(id string) string {
	return auth.Sign(auth.Identity{UserID: id}, testSecret)
}

func TestMeReturnsIdentityAndRatingsForValidCookie(t *testing.T) {
	st := newFakeStore()
	st.seedUser("u-1", "Alice", "alice@flame.edu.in")

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: signedFor("u-1")})
	rec := httptest.NewRecorder()

	meHandler(st, testSecret)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body meResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.UID != "u-1" || body.Email != "alice@flame.edu.in" || body.DisplayName != "Alice" {
		t.Errorf("unexpected body: %+v", body)
	}
	if body.Ratings["blitz"].Rating != 800 {
		t.Errorf("blitz rating = %d, want 800", body.Ratings["blitz"].Rating)
	}
}

func TestMeRejectsUnknownUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: signedFor("u-stale")})
	rec := httptest.NewRecorder()
	meHandler(newFakeStore(), testSecret)(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMeRejectsMissingCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	meHandler(newFakeStore(), testSecret)(rec, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMeRejectsGarbageCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: "garbage.deadbeef"})
	rec := httptest.NewRecorder()
	meHandler(newFakeStore(), testSecret)(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestPatchMeUpdatesNameAndReSignsCookie(t *testing.T) {
	st := newFakeStore()
	st.seedUser("u-1", "Alice", "alice@flame.edu.in")

	req := httptest.NewRequest(http.MethodPatch, "/api/me", strings.NewReader(`{"display_name":"Alice2"}`))
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: signedFor("u-1")})
	rec := httptest.NewRecorder()

	patchMeHandler(st, testSecret, cookiePolicy{})(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body meResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DisplayName != "Alice2" {
		t.Errorf("display_name = %q, want Alice2", body.DisplayName)
	}
	var ck *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == ws.SessionCookie {
			ck = c
		}
	}
	if ck == nil {
		t.Fatal("no re-signed session cookie")
	}
	id, err := auth.Verify(ck.Value, testSecret)
	if err != nil || id.DisplayName != "Alice2" {
		t.Errorf("re-signed cookie name = %q (err %v), want Alice2", id.DisplayName, err)
	}
}

func TestPatchMeRejectsTakenName(t *testing.T) {
	st := newFakeStore()
	st.seedUser("u-1", "Alice", "alice@flame.edu.in")
	st.seedUser("u-2", "Bob", "bob@flame.edu.in")

	req := httptest.NewRequest(http.MethodPatch, "/api/me", strings.NewReader(`{"display_name":"Bob"}`))
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: signedFor("u-1")})
	rec := httptest.NewRecorder()

	patchMeHandler(st, testSecret, cookiePolicy{})(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
}

func TestPatchMeRejectsInvalidName(t *testing.T) {
	st := newFakeStore()
	st.seedUser("u-1", "Alice", "alice@flame.edu.in")

	req := httptest.NewRequest(http.MethodPatch, "/api/me", strings.NewReader(`{"display_name":"   "}`))
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: signedFor("u-1")})
	rec := httptest.NewRecorder()

	patchMeHandler(st, testSecret, cookiePolicy{})(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
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
	googleCallbackHandler(oauth, newFakeStore(), testSecret, "/", cookiePolicy{})(rec, httptest.NewRequest(http.MethodGet, "/auth/google/callback", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestGoogleCallbackRejectsMismatchedState(t *testing.T) {
	oauth := auth.NewGoogleOAuth("cid", "secret", "http://localhost/cb", "flame.edu.in")
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "different"})
	rec := httptest.NewRecorder()
	googleCallbackHandler(oauth, newFakeStore(), testSecret, "/", cookiePolicy{})(rec, req)
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
