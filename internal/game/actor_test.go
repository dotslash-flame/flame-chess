package game

import (
	"testing"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/wire"
)

type fakeConn struct {
	uid  string
	sent []any
}

func (f *fakeConn) UserID() string      { return f.uid }
func (f *fakeConn) DisplayName() string { return f.uid }
func (f *fakeConn) Send(v any)          { f.sent = append(f.sent, v) }
func (f *fakeConn) Close()              {}

func lastOf[T any](f *fakeConn) (T, bool) {
	var zero T
	for i := len(f.sent) - 1; i >= 0; i-- {
		if v, ok := f.sent[i].(T); ok {
			return v, true
		}
	}
	return zero, false
}

func countOf[T any](f *fakeConn) int {
	n := 0
	for _, v := range f.sent {
		if _, ok := v.(T); ok {
			n++
		}
	}
	return n
}

type testClock struct{ t time.Time }

func (c *testClock) now() time.Time { return c.t }
func (c *testClock) advance(d time.Duration) {
	c.t = c.t.Add(d)
}

const (
	whiteUID = "white-user"
	blackUID = "black-user"
)

func newTestActor(base, inc int) (*Actor, *fakeConn, *fakeConn, *testClock, *[]string) {
	w := &fakeConn{uid: whiteUID}
	b := &fakeConn{uid: blackUID}
	clk := &testClock{t: time.Unix(0, 0)}
	g := NewGame(base, inc, clk.t)
	ended := &[]string{}
	a := NewActor("g1", g, w, b, func(id string) { *ended = append(*ended, id) }, nil)
	a.now = clk.now
	return a, w, b, clk, ended
}

func TestActorLegalMoveBroadcastsState(t *testing.T) {
	a, w, b, clk, _ := newTestActor(300, 0)

	clk.advance(1 * time.Second)
	a.handle(moveCmd{userID: whiteUID, uci: "e2e4"})

	for _, c := range []*fakeConn{w, b} {
		st, ok := lastOf[wire.GameState](c)
		if !ok {
			t.Fatalf("conn %s got no game.state", c.uid)
		}
		if st.Turn != "black" {
			t.Errorf("turn = %q, want black", st.Turn)
		}
		if st.LastMove != "e2e4" {
			t.Errorf("last_move = %q, want e2e4", st.LastMove)
		}
		if st.WhiteMs != 299000 {
			t.Errorf("white_ms = %d, want 299000", st.WhiteMs)
		}
	}
}

func TestActorIllegalMove(t *testing.T) {
	a, w, b, _, _ := newTestActor(300, 0)

	a.handle(moveCmd{userID: whiteUID, uci: "e2e5"})

	e, ok := lastOf[wire.Error](w)
	if !ok || e.Code != wire.CodeIllegalMove {
		t.Fatalf("mover error = %+v ok=%v, want illegal_move", e, ok)
	}
	if countOf[wire.GameState](w) != 0 || countOf[wire.GameState](b) != 0 {
		t.Error("no game.state should be emitted on illegal move")
	}
	if a.game.Status() != StatusActive {
		t.Error("state changed after illegal move")
	}
}

func TestActorOutOfTurnMove(t *testing.T) {
	a, _, b, _, _ := newTestActor(300, 0)

	a.handle(moveCmd{userID: blackUID, uci: "e7e5"})

	e, ok := lastOf[wire.Error](b)
	if !ok || e.Code != wire.CodeNotYourTurn {
		t.Fatalf("error = %+v ok=%v, want not_your_turn", e, ok)
	}
}

func TestActorFoolsMate(t *testing.T) {
	a, w, b, clk, ended := newTestActor(300, 0)

	moves := []struct {
		uid, uci string
	}{
		{whiteUID, "f2f3"},
		{blackUID, "e7e5"},
		{whiteUID, "g2g4"},
		{blackUID, "d8h4"},
	}
	for i, m := range moves {
		clk.advance(1 * time.Second)
		a.handle(moveCmd{userID: m.uid, uci: m.uci})
		_ = i
	}

	for _, c := range []*fakeConn{w, b} {
		over, ok := lastOf[wire.GameOver](c)
		if !ok {
			t.Fatalf("conn %s got no game.over", c.uid)
		}
		if over.Result != "0-1" || over.Reason != "checkmate" {
			t.Errorf("game.over = %+v, want 0-1/checkmate", over)
		}
	}
	if len(*ended) != 1 {
		t.Errorf("onEnd called %d times, want 1", len(*ended))
	}
}

func TestActorResign(t *testing.T) {
	a, w, _, _, _ := newTestActor(300, 0)

	a.handle(resignCmd{userID: whiteUID})

	over, ok := lastOf[wire.GameOver](w)
	if !ok {
		t.Fatal("no game.over after resign")
	}
	if over.Result != "0-1" || over.Reason != "resign" {
		t.Errorf("game.over = %+v, want 0-1/resign", over)
	}
}

func TestActorDrawOfferAccept(t *testing.T) {
	a, w, b, _, _ := newTestActor(300, 0)

	a.handle(drawOfferCmd{userID: whiteUID})
	off, ok := lastOf[wire.DrawOffered](b)
	if !ok {
		t.Fatal("opponent got no draw.offered")
	}
	if off.From != whiteUID {
		t.Errorf("draw.offered from = %q, want %q", off.From, whiteUID)
	}

	a.handle(drawRespondCmd{userID: blackUID, accept: true})
	for _, c := range []*fakeConn{w, b} {
		over, ok := lastOf[wire.GameOver](c)
		if !ok {
			t.Fatalf("conn %s no game.over after draw accept", c.uid)
		}
		if over.Result != "1/2-1/2" || over.Reason != "draw_agreed" {
			t.Errorf("game.over = %+v, want 1/2-1/2/draw_agreed", over)
		}
	}
}

func TestActorDrawOfferDecline(t *testing.T) {
	a, _, _, _, _ := newTestActor(300, 0)

	a.handle(drawOfferCmd{userID: whiteUID})
	a.handle(drawRespondCmd{userID: blackUID, accept: false})

	if a.game.Status() != StatusActive {
		t.Error("game should still be active after declined draw")
	}
}

func TestActorTimeout(t *testing.T) {
	a, w, b, clk, ended := newTestActor(5, 0)

	clk.advance(6 * time.Second)
	a.handleTimeout()

	for _, c := range []*fakeConn{w, b} {
		over, ok := lastOf[wire.GameOver](c)
		if !ok {
			t.Fatalf("conn %s got no game.over on timeout", c.uid)
		}
		if over.Result != "0-1" || over.Reason != "timeout" {
			t.Errorf("game.over = %+v, want 0-1/timeout", over)
		}
	}
	if len(*ended) != 1 {
		t.Errorf("onEnd called %d times, want 1", len(*ended))
	}
}

func TestActorMoveAfterOver(t *testing.T) {
	a, w, _, _, _ := newTestActor(300, 0)
	a.handle(resignCmd{userID: whiteUID})

	a.handle(moveCmd{userID: whiteUID, uci: "e2e4"})

	e, ok := lastOf[wire.Error](w)
	if !ok || e.Code != wire.CodeGameNotActive {
		t.Fatalf("error = %+v ok=%v, want game_not_active", e, ok)
	}
}
