package auth

import "testing"

func TestSignVerifyRoundTrip(t *testing.T) {
	secret := "topsecret"
	id := Identity{UserID: "u-123", DisplayName: "Alice"}

	cookie := Sign(id, secret)
	if cookie == "" {
		t.Fatal("Sign returned empty cookie")
	}

	got, err := Verify(cookie, secret)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != id {
		t.Errorf("round-trip identity = %+v, want %+v", got, id)
	}
}

func TestVerifyTamperedPayloadFails(t *testing.T) {
	secret := "topsecret"
	cookie := Sign(Identity{UserID: "u-1", DisplayName: "Bob"}, secret)

	tampered := "X" + cookie[1:]
	if _, err := Verify(tampered, secret); err == nil {
		t.Fatal("expected error for tampered cookie")
	}
}

func TestVerifyWrongSecretFails(t *testing.T) {
	cookie := Sign(Identity{UserID: "u-1", DisplayName: "Bob"}, "right")
	if _, err := Verify(cookie, "wrong"); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestVerifyEmptyFails(t *testing.T) {
	if _, err := Verify("", "secret"); err == nil {
		t.Fatal("expected error for empty cookie")
	}
}

func TestVerifyMalformedFails(t *testing.T) {
	if _, err := Verify("no-dot-here", "secret"); err == nil {
		t.Fatal("expected error for cookie with no separator")
	}
}

func TestSignVerifyRoundTripWithEmail(t *testing.T) {
	secret := "topsecret"
	id := Identity{UserID: "u-123", DisplayName: "Alice", Email: "alice@flame.edu.in"}

	got, err := Verify(Sign(id, secret), secret)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != id {
		t.Errorf("round-trip identity = %+v, want %+v", got, id)
	}
}

func TestVerifyOldCookieWithoutEmail(t *testing.T) {
	secret := "topsecret"
	id := Identity{UserID: "u-1", DisplayName: "Bob"}

	got, err := Verify(Sign(id, secret), secret)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Email != "" {
		t.Errorf("expected empty email, got %q", got.Email)
	}
	if got != id {
		t.Errorf("round-trip identity = %+v, want %+v", got, id)
	}
}

func TestUserIDForSubStableAndDistinct(t *testing.T) {
	s1 := UserIDForSub("google-sub-123")
	s2 := UserIDForSub("google-sub-123")
	other := UserIDForSub("google-sub-456")

	if s1 == "" {
		t.Fatal("UserIDForSub returned empty")
	}
	if s1 != s2 {
		t.Errorf("same sub produced different ids: %q vs %q", s1, s2)
	}
	if s1 == other {
		t.Errorf("different subs produced same id: %q", s1)
	}
	if UserIDForSub("Alice") == UserIDForName("Alice") {
		t.Errorf("sub and name namespaces collide for %q", "Alice")
	}
}

func TestUserIDForNameStable(t *testing.T) {
	a1 := UserIDForName("Alice")
	a2 := UserIDForName("Alice")
	b := UserIDForName("Bob")

	if a1 == "" {
		t.Fatal("UserIDForName returned empty")
	}
	if a1 != a2 {
		t.Errorf("same name produced different ids: %q vs %q", a1, a2)
	}
	if a1 == b {
		t.Errorf("different names produced same id: %q", a1)
	}
}
