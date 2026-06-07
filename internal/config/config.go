package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port               string
	DatabaseURL        string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	SessionHMACSecret  string
	AllowedEmailSuffix string
	PostLoginRedirect  string
	CORSAllowedOrigins []string
	ReconnectGraceSecs int
	StartingRating     int
	DevLogin           bool
}

func Load() (*Config, error) {
	c := &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		SessionHMACSecret:  os.Getenv("SESSION_HMAC_SECRET"),
		AllowedEmailSuffix: getEnv("ALLOWED_EMAIL_SUFFIX", "flame.edu.in"),
		PostLoginRedirect:  getEnv("APP_REDIRECT_URL", "/"),
		CORSAllowedOrigins: splitAndTrim(os.Getenv("CORS_ALLOWED_ORIGINS")),
		DevLogin:           getEnv("DEV_LOGIN", "true") != "false",
	}

	grace, err := getEnvInt("RECONNECT_GRACE_SECONDS", 30)
	if err != nil {
		return nil, err
	}
	c.ReconnectGraceSecs = grace

	rating, err := getEnvInt("STARTING_RATING", 800)
	if err != nil {
		return nil, err
	}
	c.StartingRating = rating

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.SessionHMACSecret == "" {
		return nil, fmt.Errorf("SESSION_HMAC_SECRET is required")
	}
	if !c.DevLogin {
		if c.GoogleClientID == "" || c.GoogleClientSecret == "" || c.GoogleRedirectURL == "" {
			return nil, fmt.Errorf("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL are required when DEV_LOGIN=false")
		}
	}
	return c, nil
}

func splitAndTrim(csv string) []string {
	var out []string
	for _, part := range strings.Split(csv, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return n, nil
}
