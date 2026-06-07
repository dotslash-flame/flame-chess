package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORSAllowsListedOrigin(t *testing.T) {
	h := corsMiddleware([]string{"https://app.flame.edu.in"}, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Origin", "https://app.flame.edu.in")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.flame.edu.in" {
		t.Errorf("ACAO = %q, want the echoed origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("ACAC = %q, want true", got)
	}
	if rec.Header().Get("Vary") == "" {
		t.Error("expected Vary: Origin")
	}
}

func TestCORSRejectsUnlistedOrigin(t *testing.T) {
	h := corsMiddleware([]string{"https://app.flame.edu.in"}, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("unlisted origin must not receive ACAO header")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("request should still pass through: status %d", rec.Code)
	}
}

func TestCORSPreflightShortCircuits(t *testing.T) {
	h := corsMiddleware([]string{"https://app.flame.edu.in"}, okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/api/me", nil)
	req.Header.Set("Origin", "https://app.flame.edu.in")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("preflight missing Access-Control-Allow-Methods")
	}
}

func TestCORSDisabledWhenNoOrigins(t *testing.T) {
	h := corsMiddleware(nil, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Origin", "https://app.flame.edu.in")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should be absent when no origins configured")
	}
}
