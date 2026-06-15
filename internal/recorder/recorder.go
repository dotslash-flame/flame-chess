// changes the elo based on the game
package recorder

import (
	"context"
	"log"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/game"
	"github.com/dotslash-flame/flame-chess/internal/rating"
	"github.com/dotslash-flame/flame-chess/internal/store"
	"github.com/dotslash-flame/flame-chess/internal/wire"
)

type Recorder struct {
	store   *store.Store
	gameID  string
	whiteID string
	blackID string
}

func New(s *store.Store, gameID, whiteID, blackID string) *Recorder {
	return &Recorder{store: s, gameID: gameID, whiteID: whiteID, blackID: blackID}
}

func (r *Recorder) Record(info game.EndInfo) *wire.GameRatings {
	ctx := context.Background()
	now := time.Now()
	category := string(info.Category)

	if !info.Rated {
		if err := r.store.AbortGame(ctx, r.gameID, info.Reason, info.PGN, now); err != nil {
			log.Printf("recorder: abort game %s: %v", r.gameID, err)
		}
		return nil
	}

	white, err := r.store.GetRating(ctx, r.whiteID, category)
	if err != nil {
		log.Printf("recorder: read white rating %s: %v", r.gameID, err)
		return nil
	}
	black, err := r.store.GetRating(ctx, r.blackID, category)
	if err != nil {
		log.Printf("recorder: read black rating %s: %v", r.gameID, err)
		return nil
	}

	whiteScore, blackScore, _ := rating.Outcome(info.Result)
	whiteAfter := rating.NewRating(white.Rating, black.Rating, whiteScore, white.GamesPlayed)
	blackAfter := rating.NewRating(black.Rating, white.Rating, blackScore, black.GamesPlayed)

	if err := r.store.FinishAndRate(ctx, store.FinishParams{
		GameID:      r.gameID,
		Category:    category,
		Result:      info.Result,
		Reason:      info.Reason,
		PGN:         info.PGN,
		WhiteID:     r.whiteID,
		BlackID:     r.blackID,
		WhiteBefore: white.Rating,
		WhiteAfter:  whiteAfter,
		BlackBefore: black.Rating,
		BlackAfter:  blackAfter,
		EndedAt:     now,
	}); err != nil {
		log.Printf("recorder: finish+rate game %s: %v", r.gameID, err)
		return nil
	}

	return &wire.GameRatings{
		White: wire.RatingChange{Before: white.Rating, After: whiteAfter, Delta: whiteAfter - white.Rating},
		Black: wire.RatingChange{Before: black.Rating, After: blackAfter, Delta: blackAfter - black.Rating},
	}
}
