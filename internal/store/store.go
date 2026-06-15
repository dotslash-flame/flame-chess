package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dotslash-flame/flame-chess/internal/store/db"
)

var ErrDisplayNameTaken = errors.New("display name already taken")

type Store struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool, q: db.New(pool)}, nil
}

func (s *Store) Close() { s.pool.Close() }

type User struct {
	ID          string
	GoogleSub   string
	Email       string
	DisplayName string
	AvatarURL   string
}

type CategoryRating struct {
	Rating      int
	GamesPlayed int
}

type Me struct {
	User    User
	Ratings map[string]CategoryRating
}

type LeaderboardEntry struct {
	DisplayName string
	Rating      int
	GamesPlayed int
}

type GameRow struct {
	ID           string
	WhiteID      string
	BlackID      string
	Category     string
	Status       string
	Result       string
	ResultReason string
	PGN          string
	WhiteBefore  int
	WhiteAfter   int
	BlackBefore  int
	BlackAfter   int
	StartedAt    time.Time
	EndedAt      time.Time
}

type FinishParams struct {
	GameID      string
	Category    string
	Result      string
	Reason      string
	PGN         string
	WhiteID     string
	BlackID     string
	WhiteBefore int
	WhiteAfter  int
	BlackBefore int
	BlackAfter  int
	EndedAt     time.Time
}

func (s *Store) UpsertUser(ctx context.Context, googleSub, email, displayName, avatarURL string) (User, error) {
	u, err := s.q.UpsertUser(ctx, db.UpsertUserParams{
		GoogleSub:   googleSub,
		Email:       email,
		DisplayName: displayName,
		AvatarUrl:   textOrNull(avatarURL),
	})
	if err != nil {
		return User{}, err
	}
	return toUser(u), nil
}

func (s *Store) EnsureRatings(ctx context.Context, userID string) error {
	return s.q.EnsureRatings(ctx, userID)
}

func (s *Store) GetRating(ctx context.Context, userID, category string) (CategoryRating, error) {
	r, err := s.q.GetRating(ctx, db.GetRatingParams{UserID: userID, Category: category})
	if err != nil {
		return CategoryRating{}, err
	}
	return CategoryRating{Rating: int(r.Rating), GamesPlayed: int(r.GamesPlayed)}, nil
}

func (s *Store) GetMe(ctx context.Context, userID string) (Me, error) {
	u, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	rows, err := s.q.ListRatings(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	ratings := make(map[string]CategoryRating, len(rows))
	for _, r := range rows {
		ratings[r.Category] = CategoryRating{Rating: int(r.Rating), GamesPlayed: int(r.GamesPlayed)}
	}
	return Me{User: toUser(u), Ratings: ratings}, nil
}

func (s *Store) UpdateDisplayName(ctx context.Context, userID, name string) (User, error) {
	u, err := s.q.UpdateDisplayName(ctx, db.UpdateDisplayNameParams{ID: userID, DisplayName: name})
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrDisplayNameTaken
		}
		return User{}, err
	}
	return toUser(u), nil
}

func (s *Store) InsertActiveGame(ctx context.Context, whiteID, blackID, category string, baseSeconds, incrementSec int) (string, error) {
	return s.q.InsertActiveGame(ctx, db.InsertActiveGameParams{
		WhiteID:      whiteID,
		BlackID:      blackID,
		Category:     category,
		BaseSeconds:  int32(baseSeconds),
		IncrementSec: int32(incrementSec),
	})
}

func (s *Store) FinishAndRate(ctx context.Context, p FinishParams) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)

	if err := q.UpdateRating(ctx, db.UpdateRatingParams{UserID: p.WhiteID, Category: p.Category, Rating: int32(p.WhiteAfter)}); err != nil {
		return err
	}
	if err := q.UpdateRating(ctx, db.UpdateRatingParams{UserID: p.BlackID, Category: p.Category, Rating: int32(p.BlackAfter)}); err != nil {
		return err
	}
	if err := q.FinishGame(ctx, db.FinishGameParams{
		ID:                p.GameID,
		Result:            text(p.Result),
		ResultReason:      text(p.Reason),
		Pgn:               text(p.PGN),
		WhiteRatingBefore: int4(p.WhiteBefore),
		WhiteRatingAfter:  int4(p.WhiteAfter),
		BlackRatingBefore: int4(p.BlackBefore),
		BlackRatingAfter:  int4(p.BlackAfter),
		EndedAt:           tstz(p.EndedAt),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) AbortGame(ctx context.Context, gameID, reason, pgn string, endedAt time.Time) error {
	return s.q.AbortGame(ctx, db.AbortGameParams{
		ID:           gameID,
		ResultReason: textOrNull(reason),
		Pgn:          textOrNull(pgn),
		EndedAt:      tstz(endedAt),
	})
}

func (s *Store) Leaderboard(ctx context.Context, category string, limit int) ([]LeaderboardEntry, error) {
	rows, err := s.q.Leaderboard(ctx, db.LeaderboardParams{Category: category, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]LeaderboardEntry, len(rows))
	for i, r := range rows {
		out[i] = LeaderboardEntry{DisplayName: r.DisplayName, Rating: int(r.Rating), GamesPlayed: int(r.GamesPlayed)}
	}
	return out, nil
}

func (s *Store) GamesForUser(ctx context.Context, userID string, limit int) ([]GameRow, error) {
	rows, err := s.q.GamesForUser(ctx, db.GamesForUserParams{WhiteID: userID, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]GameRow, len(rows))
	for i, g := range rows {
		out[i] = GameRow{
			ID:           g.ID,
			WhiteID:      g.WhiteID,
			BlackID:      g.BlackID,
			Category:     g.Category,
			Status:       g.Status,
			Result:       g.Result.String,
			ResultReason: g.ResultReason.String,
			PGN:          g.Pgn.String,
			WhiteBefore:  int(g.WhiteRatingBefore.Int32),
			WhiteAfter:   int(g.WhiteRatingAfter.Int32),
			BlackBefore:  int(g.BlackRatingBefore.Int32),
			BlackAfter:   int(g.BlackRatingAfter.Int32),
			StartedAt:    g.StartedAt.Time,
			EndedAt:      g.EndedAt.Time,
		}
	}
	return out, nil
}

func toUser(u db.User) User {
	return User{
		ID:          u.ID,
		GoogleSub:   u.GoogleSub,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarUrl.String,
	}
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }

func textOrNull(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func int4(n int) pgtype.Int4 { return pgtype.Int4{Int32: int32(n), Valid: true} }

func tstz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
