package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// struct that cookies will carry, helps us identify the user
type Identity struct {
	UserID      string `json:"uid"`
	DisplayName string `json:"name"`
	Email       string `json:"email,omitempty"`
}

var ErrInvalidSession = errors.New("invalid session cookie")

// encodes identity struct and creates payload
func Sign(id Identity, secret string) string {
	raw, _ := json.Marshal(id)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + hex.EncodeToString(mac(payload, secret))
}

func Verify(cookie, secret string) (Identity, error) {
	payload, sig, ok := strings.Cut(cookie, ".")
	if !ok || payload == "" || sig == "" {
		return Identity{}, ErrInvalidSession
	}
	want, err := hex.DecodeString(sig)
	if err != nil {
		return Identity{}, ErrInvalidSession
	}
	if !hmac.Equal(want, mac(payload, secret)) {
		return Identity{}, ErrInvalidSession
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return Identity{}, ErrInvalidSession
	}
	var id Identity
	if err := json.Unmarshal(raw, &id); err != nil {
		return Identity{}, ErrInvalidSession
	}
	return id, nil
}

func UserIDForName(name string) string {
	sum := sha256.Sum256([]byte("flamechess-uid:" + name))
	return "u-" + hex.EncodeToString(sum[:8])
}

func UserIDForSub(sub string) string {
	sum := sha256.Sum256([]byte("flamechess-uid-sub:" + sub))
	return "u-" + hex.EncodeToString(sum[:8])
}

func mac(payload, secret string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return h.Sum(nil)
}
