package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const userInfoEndpoint = "https://openidconnect.googleapis.com/v1/userinfo"

func AllowedEmail(email string, verified bool, suffix string) bool {
	if !verified || email == "" {
		return false
	}
	return strings.HasSuffix(strings.ToLower(email), "@"+strings.ToLower(suffix))
}

type GoogleUser struct {
	Sub           string
	Email         string
	Name          string
	EmailVerified bool
}

type GoogleOAuth struct {
	conf   *oauth2.Config
	suffix string
}

func NewGoogleOAuth(clientID, clientSecret, redirectURL, suffix string) *GoogleOAuth {
	return &GoogleOAuth{
		conf: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
		},
		suffix: suffix,
	}
}

func (g *GoogleOAuth) Suffix() string { return g.suffix }

func (g *GoogleOAuth) LoginURL(state string) string {
	return g.conf.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (g *GoogleOAuth) Exchange(ctx context.Context, code string) (GoogleUser, error) {
	tok, err := g.conf.Exchange(ctx, code)
	if err != nil {
		return GoogleUser{}, fmt.Errorf("token exchange: %w", err)
	}
	resp, err := g.conf.Client(ctx, tok).Get(userInfoEndpoint)
	if err != nil {
		return GoogleUser{}, fmt.Errorf("userinfo request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return GoogleUser{}, fmt.Errorf("userinfo status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GoogleUser{}, fmt.Errorf("read userinfo: %w", err)
	}
	return parseUserInfo(body)
}

func parseUserInfo(body []byte) (GoogleUser, error) {
	var raw struct {
		Sub           string          `json:"sub"`
		Email         string          `json:"email"`
		Name          string          `json:"name"`
		EmailVerified json.RawMessage `json:"email_verified"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return GoogleUser{}, fmt.Errorf("parse userinfo: %w", err)
	}
	return GoogleUser{
		Sub:           raw.Sub,
		Email:         raw.Email,
		Name:          raw.Name,
		EmailVerified: parseFlexibleBool(raw.EmailVerified),
	}, nil
}

func parseFlexibleBool(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.EqualFold(s, "true")
	}
	return false
}
