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
