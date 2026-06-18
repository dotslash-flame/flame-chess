package game

import (
	"strings"
	"testing"
	"time"

	"github.com/notnil/chess"

	"github.com/dotslash-flame/flame-chess/internal/wire"
)

type fakeChat struct {
	calls   int
	lastTxt string
}

func (f *fakeChat) RecordChat(_, body string) {
	f.calls++
	f.lastTxt = body
}

func TestActorPlayerGoneNotifiesOpponentNotGoneSide(t *testing.T) {
	a, w, b, _, _ := newTestActor(300, 0)
	a.graceSecs = 30

	a.handle(playerGoneCmd{userID: whiteUID})

	d, ok := lastOf[wire.OpponentDisconnected](b)
	if !ok {
		t.Fatal("opponent got no opponent.disconnected")
	}
	if d.Color != "white" || d.GraceSeconds != 30 {
		t.Errorf("disconnected = %+v, want white/30", d)
	}
	if countOf[wire.OpponentDisconnected](w) != 0 {
		t.Error("the gone player must not receive opponent.disconnected")
	}
	if a.game.Status() != StatusActive {
		t.Error("game must stay active during grace")
	}
}

func TestActorRejoinBeforeExpiryResumes(t *testing.T) {
	a, _, b, _, _ := newTestActor(300, 0)
	a.graceSecs = 30
	a.handle(playerGoneCmd{userID: whiteUID})

	w2 := &fakeConn{uid: whiteUID}
	a.handle(rejoinCmd{userID: whiteUID, conn: w2})

	if _, ok := lastOf[wire.GameStart](w2); !ok {
		t.Error("rejoiner got no game.start snapshot")
	}
	if _, ok := lastOf[wire.GameState](w2); !ok {
		t.Error("rejoiner got no game.state snapshot")
	}
	if _, ok := lastOf[wire.OpponentReconnected](b); !ok {
		t.Error("opponent got no opponent.reconnected")
	}
	if a.disconnected[chess.White] {
		t.Error("white should no longer be marked disconnected")
	}
	if a.game.Status() != StatusActive {
		t.Error("game must continue after rejoin")
	}
}

func TestActorAbandonOnGraceExpiry(t *testing.T) {
	a, w, b, clk, ended := newTestActor(300, 0)
	a.graceSecs = 30
	rec := &fakeRec{}
	a.rec = rec

	clk.advance(time.Second)
	a.handle(moveCmd{userID: whiteUID, uci: "e2e4"})
	a.handle(playerGoneCmd{userID: whiteUID})
	a.handleGraceExpired()

	for _, c := range []*fakeConn{w, b} {
		over, ok := lastOf[wire.GameOver](c)
		if !ok {
			t.Fatalf("conn %s got no game.over on abandon", c.uid)
		}
		if over.Result != "0-1" || over.Reason != "abandoned" {
			t.Errorf("game.over = %+v, want 0-1/abandoned", over)
		}
	}
	if rec.calls != 1 || !rec.last.Rated {
		t.Errorf("recorder calls=%d rated=%v, want 1/true", rec.calls, rec.last.Rated)
	}
	if len(*ended) != 1 {
		t.Errorf("onEnd called %d times, want 1", len(*ended))
	}
}

func TestActorGraceZeroAbandonsImmediately(t *testing.T) {
	a, _, b, _, _ := newTestActor(300, 0)
	a.handle(playerGoneCmd{userID: whiteUID})

	over, ok := lastOf[wire.GameOver](b)
	if !ok {
		t.Fatal("opponent got no game.over with zero grace")
	}
	if over.Reason != "abandoned" {
		t.Errorf("reason = %q, want abandoned", over.Reason)
	}
}

func TestActorPlayerGoneAfterOverIsNoop(t *testing.T) {
	a, _, _, _, ended := newTestActor(300, 0)
	a.handle(resignCmd{userID: whiteUID})
	before := len(*ended)

	a.handle(playerGoneCmd{userID: whiteUID})

	if len(*ended) != before {
		t.Error("player gone after game over should be a no-op")
	}
}

func TestActorChatRelaysToPlayersAndSpectators(t *testing.T) {
	a, w, b, _, _ := newTestActor(300, 0)
	chat := &fakeChat{}
	a.chat = chat
	spec := &fakeConn{uid: "watcher"}
	a.handle(addSpectatorCmd{userID: "watcher", conn: spec})

	a.handle(chatCmd{userID: whiteUID, text: "  hello  "})

	for _, c := range []*fakeConn{w, b, spec} {
		m, ok := lastOf[wire.ChatMsg](c)
		if !ok {
			t.Fatalf("conn %s got no chat.msg", c.uid)
		}
		if m.Text != "hello" || m.From != whiteUID {
			t.Errorf("chat.msg = %+v, want hello/%s", m, whiteUID)
		}
	}
	if chat.calls != 1 || chat.lastTxt != "hello" {
		t.Errorf("RecordChat calls=%d last=%q, want 1/hello", chat.calls, chat.lastTxt)
	}
}

func TestActorChatFromSpectatorIgnored(t *testing.T) {
	a, w, _, _, _ := newTestActor(300, 0)
	spec := &fakeConn{uid: "watcher"}
	a.handle(addSpectatorCmd{userID: "watcher", conn: spec})

	a.handle(chatCmd{userID: "watcher", text: "hi"})

	if countOf[wire.ChatMsg](w) != 0 {
		t.Error("spectator chat must not be relayed")
	}
}

func TestActorChatEmptyDropped(t *testing.T) {
	a, w, _, _, _ := newTestActor(300, 0)
	a.handle(chatCmd{userID: whiteUID, text: "   "})
	if countOf[wire.ChatMsg](w) != 0 {
		t.Error("whitespace-only chat must be dropped")
	}
}

func TestActorChatTruncatedTo500(t *testing.T) {
	a, w, _, _, _ := newTestActor(300, 0)
	a.handle(chatCmd{userID: whiteUID, text: strings.Repeat("x", 600)})
	m, ok := lastOf[wire.ChatMsg](w)
	if !ok {
		t.Fatal("no chat.msg")
	}
	if n := len([]rune(m.Text)); n != maxChatRunes {
		t.Errorf("chat length = %d, want %d", n, maxChatRunes)
	}
}

func TestActorSpectatorSnapshotAndLiveUpdates(t *testing.T) {
	a, _, _, clk, _ := newTestActor(300, 0)
	spec := &fakeConn{uid: "watcher"}

	a.handle(addSpectatorCmd{userID: "watcher", conn: spec})
	gs, ok := lastOf[wire.GameStart](spec)
	if !ok {
		t.Fatal("spectator got no snapshot game.start")
	}
	if gs.Color != wire.ColorSpectator {
		t.Errorf("snapshot color = %q, want spectator", gs.Color)
	}
	if gs.White != whiteUID || gs.Black != blackUID {
		t.Errorf("snapshot names = %q/%q, want %q/%q", gs.White, gs.Black, whiteUID, blackUID)
	}
	if countOf[wire.GameState](spec) != 1 {
		t.Errorf("snapshot game.state count = %d, want 1", countOf[wire.GameState](spec))
	}

	clk.advance(time.Second)
	a.handle(moveCmd{userID: whiteUID, uci: "e2e4"})
	if countOf[wire.GameState](spec) != 2 {
		t.Error("spectator should receive live game.state on a move")
	}
}

func TestActorSpectatorNeverSeesDrawOffer(t *testing.T) {
	a, _, _, _, _ := newTestActor(300, 0)
	spec := &fakeConn{uid: "watcher"}
	a.handle(addSpectatorCmd{userID: "watcher", conn: spec})

	a.handle(drawOfferCmd{userID: whiteUID})

	if countOf[wire.DrawOffered](spec) != 0 {
		t.Error("spectators must not receive draw.offered (player-only)")
	}
}

func TestActorRemoveSpectatorStopsDelivery(t *testing.T) {
	a, _, _, clk, _ := newTestActor(300, 0)
	spec := &fakeConn{uid: "watcher"}
	a.handle(addSpectatorCmd{userID: "watcher", conn: spec})
	a.handle(removeSpectatorCmd{userID: "watcher"})

	before := countOf[wire.GameState](spec)
	clk.advance(time.Second)
	a.handle(moveCmd{userID: whiteUID, uci: "e2e4"})
	if countOf[wire.GameState](spec) != before {
		t.Error("removed spectator should receive no further game.state")
	}
}

// helpers

type fakeRec struct {
	calls int
	last  EndInfo
}

func (f *fakeRec) Record(info EndInfo) *wire.GameRatings {
	f.calls++
	f.last = info
	return nil
}
