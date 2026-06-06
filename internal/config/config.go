package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port               string
	DatabaseURL        string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	SessionHMACSecret  string
	AllowedEmailSuffix string
	ReconnectGraceSecs int
	StartingRating     int
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
	return c, nil
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
