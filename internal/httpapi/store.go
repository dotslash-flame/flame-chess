package httpapi

import (
	"context"

	"github.com/dotslash-flame/flame-chess/internal/store"
)

// store is a subset of *Store.store
type Store interface {
	UpsertUser(ctx context.Context, googleSub, email, displayName, avatarURL string) (store.User, error)
	EnsureRatings(ctx context.Context, userID string) error
	GetMe(ctx context.Context, userID string) (store.Me, error)
	UpdateDisplayName(ctx context.Context, userID, name string) (store.User, error)
	Leaderboard(ctx context.Context, category string, limit int) ([]store.LeaderboardEntry, error)
	GamesForUser(ctx context.Context, userID string, limit int) ([]store.GameRow, error)
}
