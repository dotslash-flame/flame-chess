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
