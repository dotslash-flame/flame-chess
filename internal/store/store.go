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
	Voided       bool
	StartedAt    time.Time
	EndedAt      time.Time
}

type ChatRow struct {
	SenderID   string
	SenderName string
	Body       string
	CreatedAt  time.Time
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
			Voided:       g.Voided,
			StartedAt:    g.StartedAt.Time,
			EndedAt:      g.EndedAt.Time,
		}
	}
	return out, nil
}

func (s *Store) GameByID(ctx context.Context, gameID string) (GameRow, error) {
	g, err := s.q.GameByID(ctx, gameID)
	if err != nil {
		return GameRow{}, err
	}
	return GameRow{
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
		Voided:       g.Voided,
		StartedAt:    g.StartedAt.Time,
		EndedAt:      g.EndedAt.Time,
	}, nil
}

func (s *Store) InsertGameMessage(ctx context.Context, gameID, senderID, body string) (int64, time.Time, error) {
	row, err := s.q.InsertGameMessage(ctx, db.InsertGameMessageParams{
		GameID:   gameID,
		SenderID: senderID,
		Body:     body,
	})
	if err != nil {
		return 0, time.Time{}, err
	}
	return row.ID, row.CreatedAt.Time, nil
}

func (s *Store) GameMessages(ctx context.Context, gameID string) ([]ChatRow, error) {
	rows, err := s.q.GameMessages(ctx, gameID)
	if err != nil {
		return nil, err
	}
	out := make([]ChatRow, len(rows))
	for i, m := range rows {
		out[i] = ChatRow{
			SenderID:   m.SenderID,
			SenderName: m.SenderName,
			Body:       m.Body,
			CreatedAt:  m.CreatedAt.Time,
		}
	}
	return out, nil
}

type AdminUser struct {
	ID          string
	Email       string
	DisplayName string
	CreatedAt   time.Time
	Ratings     map[string]CategoryRating
}

type AdminGameRow struct {
	ID           string
	WhiteID      string
	BlackID      string
	WhiteName    string
	BlackName    string
	Category     string
	Status       string
	Result       string
	ResultReason string
	WhiteBefore  int
	WhiteAfter   int
	BlackBefore  int
	BlackAfter   int
	Voided       bool
	StartedAt    time.Time
	EndedAt      time.Time
}

type AdminChatRow struct {
	ID         int64
	SenderID   string
	SenderName string
	Body       string
	Hidden     bool
	CreatedAt  time.Time
}

func (s *Store) AdminListUsersWithRatings(ctx context.Context, limit int) ([]AdminUser, error) {
	us, err := s.q.AdminListUsers(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	rs, err := s.q.AdminAllRatings(ctx)
	if err != nil {
		return nil, err
	}
	byUser := make(map[string]map[string]CategoryRating)
	for _, r := range rs {
		m := byUser[r.UserID]
		if m == nil {
			m = make(map[string]CategoryRating)
			byUser[r.UserID] = m
		}
		m[r.Category] = CategoryRating{Rating: int(r.Rating), GamesPlayed: int(r.GamesPlayed)}
	}
	out := make([]AdminUser, len(us))
	for i, u := range us {
		out[i] = AdminUser{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, CreatedAt: u.CreatedAt.Time, Ratings: byUser[u.ID]}
	}
	return out, nil
}

func (s *Store) AdminSetRating(ctx context.Context, userID, category string, rating, gamesPlayed int) error {
	if gamesPlayed < 0 {
		gamesPlayed = 0
	}
	return s.q.AdminSetRating(ctx, db.AdminSetRatingParams{
		UserID: userID, Category: category, Rating: int32(rating), GamesPlayed: int32(gamesPlayed),
	})
}

func (s *Store) AdminListGames(ctx context.Context, limit int) ([]AdminGameRow, error) {
	rows, err := s.q.AdminListGames(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]AdminGameRow, len(rows))
	for i, g := range rows {
		out[i] = AdminGameRow{
			ID: g.ID, WhiteID: g.WhiteID, BlackID: g.BlackID, WhiteName: g.WhiteName, BlackName: g.BlackName,
			Category: g.Category, Status: g.Status, Result: g.Result.String, ResultReason: g.ResultReason.String,
			WhiteBefore: int(g.WhiteRatingBefore.Int32), WhiteAfter: int(g.WhiteRatingAfter.Int32),
			BlackBefore: int(g.BlackRatingBefore.Int32), BlackAfter: int(g.BlackRatingAfter.Int32),
			Voided: g.Voided, StartedAt: g.StartedAt.Time, EndedAt: g.EndedAt.Time,
		}
	}
	return out, nil
}

func (s *Store) AdminGamesByUser(ctx context.Context, userID string, limit int) ([]AdminGameRow, error) {
	rows, err := s.q.AdminGamesByUser(ctx, db.AdminGamesByUserParams{WhiteID: userID, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]AdminGameRow, len(rows))
	for i, g := range rows {
		out[i] = AdminGameRow{
			ID: g.ID, WhiteID: g.WhiteID, BlackID: g.BlackID, WhiteName: g.WhiteName, BlackName: g.BlackName,
			Category: g.Category, Status: g.Status, Result: g.Result.String, ResultReason: g.ResultReason.String,
			WhiteBefore: int(g.WhiteRatingBefore.Int32), WhiteAfter: int(g.WhiteRatingAfter.Int32),
			BlackBefore: int(g.BlackRatingBefore.Int32), BlackAfter: int(g.BlackRatingAfter.Int32),
			Voided: g.Voided, StartedAt: g.StartedAt.Time, EndedAt: g.EndedAt.Time,
		}
	}
	return out, nil
}

func (s *Store) AdminVoidGame(ctx context.Context, gameID string, voided bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)

	g, err := q.GameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if g.Voided == voided {
		return nil
	}
	rated := g.Status == "finished" && g.Result.Valid && g.WhiteRatingBefore.Valid && g.WhiteRatingAfter.Valid
	if rated {
		adj := func(userID string, before, after int32) error {
			cur, err := q.GetRating(ctx, db.GetRatingParams{UserID: userID, Category: g.Category})
			if err != nil {
				return err
			}
			gp := cur.GamesPlayed
			var rating int32
			if voided {
				rating, gp = before, gp-1
			} else {
				rating, gp = after, gp+1
			}
			if gp < 0 {
				gp = 0
			}
			return q.AdminSetRating(ctx, db.AdminSetRatingParams{UserID: userID, Category: g.Category, Rating: rating, GamesPlayed: gp})
		}
		if err := adj(g.WhiteID, g.WhiteRatingBefore.Int32, g.WhiteRatingAfter.Int32); err != nil {
			return err
		}
		if err := adj(g.BlackID, g.BlackRatingBefore.Int32, g.BlackRatingAfter.Int32); err != nil {
			return err
		}
	}
	if err := q.AdminSetGameVoided(ctx, db.AdminSetGameVoidedParams{ID: gameID, Voided: voided}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) AdminSetMessageHidden(ctx context.Context, msgID int64, hidden bool) error {
	return s.q.AdminSetMessageHidden(ctx, db.AdminSetMessageHiddenParams{ID: msgID, Hidden: hidden})
}

func (s *Store) AdminGameMessages(ctx context.Context, gameID string) ([]AdminChatRow, error) {
	rows, err := s.q.AdminGameMessages(ctx, gameID)
	if err != nil {
		return nil, err
	}
	out := make([]AdminChatRow, len(rows))
	for i, m := range rows {
		out[i] = AdminChatRow{ID: m.ID, SenderID: m.SenderID, SenderName: m.SenderName, Body: m.Body, Hidden: m.Hidden, CreatedAt: m.CreatedAt.Time}
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
