package httpapi

import (
	"context"
	"errors"
	"strconv"

	"github.com/dotslash-flame/flame-chess/internal/store"
)

// fakeStore is an in-memory implementation of the Store interface for tests.
type fakeStore struct {
	bySub   map[string]string // google_sub -> user id
	byID    map[string]store.User
	ratings map[string]map[string]store.CategoryRating // user id -> category -> rating
	games   map[string][]store.GameRow                 // user id -> finished games
	seq     int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		bySub:   map[string]string{},
		byID:    map[string]store.User{},
		ratings: map[string]map[string]store.CategoryRating{},
		games:   map[string][]store.GameRow{},
	}
}

func (f *fakeStore) seedUser(id, name, email string) {
	f.byID[id] = store.User{ID: id, DisplayName: name, Email: email}
	f.ratings[id] = map[string]store.CategoryRating{
		"bullet": {Rating: 800},
		"blitz":  {Rating: 800},
		"rapid":  {Rating: 800},
	}
}

func (f *fakeStore) UpsertUser(_ context.Context, googleSub, email, displayName, avatarURL string) (store.User, error) {
	if id, ok := f.bySub[googleSub]; ok {
		u := f.byID[id]
		u.Email = email // re-login refreshes email/avatar but not the display name
		u.AvatarURL = avatarURL
		f.byID[id] = u
		return u, nil
	}
	f.seq++
	id := "uuid-" + strconv.Itoa(f.seq)
	u := store.User{ID: id, GoogleSub: googleSub, Email: email, DisplayName: displayName, AvatarURL: avatarURL}
	f.bySub[googleSub] = id
	f.byID[id] = u
	return u, nil
}

func (f *fakeStore) EnsureRatings(_ context.Context, userID string) error {
	if _, ok := f.ratings[userID]; !ok {
		f.ratings[userID] = map[string]store.CategoryRating{
			"bullet": {Rating: 800},
			"blitz":  {Rating: 800},
			"rapid":  {Rating: 800},
		}
	}
	return nil
}

func (f *fakeStore) GetMe(_ context.Context, userID string) (store.Me, error) {
	u, ok := f.byID[userID]
	if !ok {
		return store.Me{}, errors.New("user not found")
	}
	return store.Me{User: u, Ratings: f.ratings[userID]}, nil
}

func (f *fakeStore) UpdateDisplayName(_ context.Context, userID, name string) (store.User, error) {
	for id, u := range f.byID {
		if id != userID && u.DisplayName == name {
			return store.User{}, store.ErrDisplayNameTaken
		}
	}
	u, ok := f.byID[userID]
	if !ok {
		return store.User{}, errors.New("user not found")
	}
	u.DisplayName = name
	f.byID[userID] = u
	return u, nil
}

func (f *fakeStore) Leaderboard(_ context.Context, category string, limit int) ([]store.LeaderboardEntry, error) {
	var out []store.LeaderboardEntry
	for id, cats := range f.ratings {
		r, ok := cats[category]
		if !ok {
			continue
		}
		out = append(out, store.LeaderboardEntry{
			DisplayName: f.byID[id].DisplayName,
			Rating:      r.Rating,
			GamesPlayed: r.GamesPlayed,
		})
	}
	// Insertion-sort by rating desc for deterministic ordering.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Rating > out[j-1].Rating; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeStore) GamesForUser(_ context.Context, userID string, limit int) ([]store.GameRow, error) {
	rows := f.games[userID]
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}
