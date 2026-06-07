package auth

import (
	"strings"
	"testing"
)

func TestAllowedEmail(t *testing.T) {
	const suffix = "flame.edu.in"

	if !AllowedEmail("student@flame.edu.in", true, suffix) {
		t.Error("verified matching-suffix email should be allowed")
	}
	if AllowedEmail("student@flame.edu.in", false, suffix) {
		t.Error("unverified email should be rejected")
	}
	if AllowedEmail("someone@gmail.com", true, suffix) {
		t.Error("wrong-suffix email should be rejected")
	}
	if !AllowedEmail("Student@FLAME.EDU.IN", true, suffix) {
		t.Error("matching should be case-insensitive")
	}
	if AllowedEmail("", true, suffix) {
		t.Error("empty email should be rejected")
	}
	if AllowedEmail("attacker@notflame.edu.in", true, suffix) {
		t.Error("suffix must be a domain boundary, not a bare substring")
	}
}

func TestParseUserInfoBoolVerified(t *testing.T) {
	body := []byte(`{"sub":"123","email":"a@flame.edu.in","email_verified":true,"name":"Ann"}`)
	u, err := parseUserInfo(body)
	if err != nil {
		t.Fatalf("parseUserInfo: %v", err)
	}
	if u.Sub != "123" || u.Email != "a@flame.edu.in" || u.Name != "Ann" || !u.EmailVerified {
		t.Errorf("unexpected user: %+v", u)
	}
}

func TestParseUserInfoStringVerified(t *testing.T) {
	body := []byte(`{"sub":"123","email":"a@flame.edu.in","email_verified":"true","name":"Ann"}`)
	u, err := parseUserInfo(body)
	if err != nil {
		t.Fatalf("parseUserInfo: %v", err)
	}
	if !u.EmailVerified {
		t.Errorf("email_verified string \"true\" should parse as true: %+v", u)
	}
}

func TestParseUserInfoStringFalse(t *testing.T) {
	body := []byte(`{"sub":"123","email":"a@flame.edu.in","email_verified":"false"}`)
	u, err := parseUserInfo(body)
	if err != nil {
		t.Fatalf("parseUserInfo: %v", err)
	}
	if u.EmailVerified {
		t.Errorf("email_verified string \"false\" should parse as false: %+v", u)
	}
}

func TestParseUserInfoMissingVerified(t *testing.T) {
	body := []byte(`{"sub":"123","email":"a@flame.edu.in"}`)
	u, err := parseUserInfo(body)
	if err != nil {
		t.Fatalf("parseUserInfo: %v", err)
	}
	if u.EmailVerified {
		t.Errorf("missing email_verified should default to false: %+v", u)
	}
}

func TestParseUserInfoBadJSON(t *testing.T) {
	if _, err := parseUserInfo([]byte("not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoginURLContainsParams(t *testing.T) {
	o := NewGoogleOAuth("my-client-id", "secret", "http://localhost:8080/auth/google/callback", "flame.edu.in")
	url := o.LoginURL("xyz-state")
	for _, want := range []string{"my-client-id", "state=xyz-state", "redirect_uri", "accounts.google.com"} {
		if !strings.Contains(url, want) {
			t.Errorf("LoginURL missing %q: %s", want, url)
		}
	}
}
