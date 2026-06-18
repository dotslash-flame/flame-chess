package recorder

import (
	"context"
	"log"

	"github.com/dotslash-flame/flame-chess/internal/store"
)

type Chat struct {
	store  *store.Store
	gameID string
}

func NewChat(s *store.Store, gameID string) *Chat {
	return &Chat{store: s, gameID: gameID}
}

func (c *Chat) RecordChat(senderID, body string) {
	if c == nil || c.store == nil {
		return
	}
	gameID, store := c.gameID, c.store
	go func() {
		if _, _, err := store.InsertGameMessage(context.Background(), gameID, senderID, body); err != nil {
			log.Printf("recorder: insert chat for game %s: %v", gameID, err)
		}
	}()
}
