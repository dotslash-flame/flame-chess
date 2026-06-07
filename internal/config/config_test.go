package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SESSION_HMAC_SECRET", "secret")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.Port != "8080" {
		t.Errorf("Port = %q, want 8080", c.Port)
	}
	if c.AllowedEmailSuffix != "flame.edu.in" {
		t.Errorf("AllowedEmailSuffix = %q, want flame.edu.in", c.AllowedEmailSuffix)
	}
	if c.ReconnectGraceSecs != 30 {
		t.Errorf("ReconnectGraceSecs = %d, want 30", c.ReconnectGraceSecs)
	}
	if c.StartingRating != 800 {
		t.Errorf("StartingRating = %d, want 800", c.StartingRating)
	}
	if !c.DevLogin {
		t.Error("DevLogin should default to true")
	}
}

func TestDevLoginDisabled(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SESSION_HMAC_SECRET", "secret")
	t.Setenv("DEV_LOGIN", "false")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback")
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.DevLogin {
		t.Error("DevLogin should be false when DEV_LOGIN=false")
	}
}

func TestDevLoginDisabledRequiresGoogleCreds(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SESSION_HMAC_SECRET", "secret")
	t.Setenv("DEV_LOGIN", "false")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when DEV_LOGIN=false and google creds incomplete")
	}
}

func TestCORSAllowedOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SESSION_HMAC_SECRET", "secret")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(c.CORSAllowedOrigins) != 0 {
		t.Errorf("default CORSAllowedOrigins = %v, want empty", c.CORSAllowedOrigins)
	}

	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.flame.edu.in, http://localhost:5173 ,")
	c, err = Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := []string{"https://app.flame.edu.in", "http://localhost:5173"}
	if len(c.CORSAllowedOrigins) != len(want) {
		t.Fatalf("CORSAllowedOrigins = %v, want %v", c.CORSAllowedOrigins, want)
	}
	for i := range want {
		if c.CORSAllowedOrigins[i] != want[i] {
			t.Errorf("origin[%d] = %q, want %q", i, c.CORSAllowedOrigins[i], want[i])
		}
	}
}

func TestPostLoginRedirectDefaultAndOverride(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SESSION_HMAC_SECRET", "secret")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.PostLoginRedirect != "/" {
		t.Errorf("PostLoginRedirect default = %q, want /", c.PostLoginRedirect)
	}

	t.Setenv("APP_REDIRECT_URL", "/lobby")
	c, err = Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.PostLoginRedirect != "/lobby" {
		t.Errorf("PostLoginRedirect = %q, want /lobby", c.PostLoginRedirect)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SESSION_HMAC_SECRET", "secret")
	t.Setenv("PORT", "9000")
	t.Setenv("ALLOWED_EMAIL_SUFFIX", "example.com")
	t.Setenv("RECONNECT_GRACE_SECONDS", "45")
	t.Setenv("STARTING_RATING", "1000")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.Port != "9000" || c.AllowedEmailSuffix != "example.com" ||
		c.ReconnectGraceSecs != 45 || c.StartingRating != 1000 {
		t.Errorf("overrides not applied: %+v", c)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_HMAC_SECRET", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for missing DATABASE_URL/SESSION_HMAC_SECRET")
	}
}

func TestLoadBadInt(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SESSION_HMAC_SECRET", "secret")
	t.Setenv("RECONNECT_GRACE_SECONDS", "notanumber")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for non-integer RECONNECT_GRACE_SECONDS")
	}
}
