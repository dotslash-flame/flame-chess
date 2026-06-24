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
	GameByID(ctx context.Context, gameID string) (store.GameRow, error)
	GameMessages(ctx context.Context, gameID string) ([]store.ChatRow, error)
	AdminListUsersWithRatings(ctx context.Context, limit int) ([]store.AdminUser, error)
	AdminSetRating(ctx context.Context, userID, category string, rating, gamesPlayed int) error
	AdminListGames(ctx context.Context, limit int) ([]store.AdminGameRow, error)
	AdminGamesByUser(ctx context.Context, userID string, limit int) ([]store.AdminGameRow, error)
	AdminVoidGame(ctx context.Context, gameID string, voided bool) error
	AdminSetMessageHidden(ctx context.Context, msgID int64, hidden bool) error
	AdminGameMessages(ctx context.Context, gameID string) ([]store.AdminChatRow, error)
}
