package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotslash-flame/flame-chess/internal/auth"
	"github.com/dotslash-flame/flame-chess/internal/hub"
	"github.com/dotslash-flame/flame-chess/internal/ws"
)

const testSecret = "test-secret"

func testRouter() http.Handler {
	return NewRouter(hub.New(hub.Options{}), testSecret, true)
}

func TestHealthz(t *testing.T) {
	srv := httptest.NewServer(testRouter())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want ok", body["status"])
	}
}

func TestDevLoginSetsVerifiableCookie(t *testing.T) {
	srv := httptest.NewServer(testRouter())
	defer srv.Close()

	resp, err := http.PostForm(srv.URL+"/auth/dev-login", map[string][]string{"name": {"Alice"}})
	if err != nil {
		t.Fatalf("POST /auth/dev-login: %v", err)
	}
	defer resp.Body.Close()

	var ck *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == ws.SessionCookie {
			ck = c
		}
	}
	if ck == nil {
		t.Fatal("no session cookie set")
	}
	id, err := auth.Verify(ck.Value, testSecret)
	if err != nil {
		t.Fatalf("cookie did not verify: %v", err)
	}
	if id.DisplayName != "Alice" {
		t.Errorf("cookie name = %q, want Alice", id.DisplayName)
	}
}

func TestWSRejectsMissingCookie(t *testing.T) {
	srv := httptest.NewServer(testRouter())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ws")
	if err != nil {
		t.Fatalf("GET /ws: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestWSRejectsInvalidCookie(t *testing.T) {
	srv := httptest.NewServer(testRouter())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/ws", nil)
	req.AddCookie(&http.Cookie{Name: ws.SessionCookie, Value: "garbage.deadbeef"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /ws: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}
