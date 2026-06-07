package hub

import (
	"testing"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/wire"
)

type fakeConn struct {
	uid    string
	name   string
	sent   []any
	closed bool
}

func (f *fakeConn) UserID() string      { return f.uid }
func (f *fakeConn) DisplayName() string { return f.name }
func (f *fakeConn) Send(v any)          { f.sent = append(f.sent, v) }
func (f *fakeConn) Close()              { f.closed = true }

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

func newTestHub() *Hub {
	h := New(Options{
		Rng:    func(int) int { return 0 },
		NextID: func() string { return "game-1" },
		Now:    func() time.Time { return time.Unix(0, 0) },
	})
	h.run = func(*gameActor) {}
	return h
}

func TestPairTwoPlayers(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})

	h.handle(queueJoinCmd{userID: "a", base: 300, increment: 0})
	if _, ok := lastOf[wire.QueueWaiting](ca); !ok {
		t.Fatal("first joiner should get queue.waiting")
	}

	h.handle(queueJoinCmd{userID: "b", base: 300, increment: 0})
	sa, ok := lastOf[wire.GameStart](ca)
	if !ok {
		t.Fatal("a got no game.start")
	}
	sb, ok := lastOf[wire.GameStart](cb)
	if !ok {
		t.Fatal("b got no game.start")
	}
	if sa.GameID != sb.GameID {
		t.Errorf("game ids differ: %q vs %q", sa.GameID, sb.GameID)
	}
	if sa.Color == sb.Color {
		t.Errorf("both players got color %q, want opposite", sa.Color)
	}
	if sa.Color != "white" || sb.Color != "black" {
		t.Errorf("colors = %q/%q, want white/black", sa.Color, sb.Color)
	}
	if sa.Opponent != "Bob" || sb.Opponent != "Alice" {
		t.Errorf("opponents = %q/%q, want Bob/Alice", sa.Opponent, sb.Opponent)
	}
	if sa.Clocks.WhiteMs != 300000 {
		t.Errorf("white_ms = %d, want 300000", sa.Clocks.WhiteMs)
	}
}

func TestDifferentPoolsDontMatch(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})

	h.handle(queueJoinCmd{userID: "a", base: 300, increment: 0})
	h.handle(queueJoinCmd{userID: "b", base: 180, increment: 0})

	if countOf[wire.GameStart](ca) != 0 || countOf[wire.GameStart](cb) != 0 {
		t.Error("players in different pools must not be matched")
	}
}

func TestQueueLeaveRemovesWaiter(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})

	h.handle(queueJoinCmd{userID: "a", base: 300, increment: 0})
	h.handle(queueLeaveCmd{userID: "a"})
	h.handle(queueJoinCmd{userID: "b", base: 300, increment: 0})

	if countOf[wire.GameStart](cb) != 0 {
		t.Error("b should not be matched after a left the pool")
	}
	if _, ok := lastOf[wire.QueueWaiting](cb); !ok {
		t.Error("b should be waiting")
	}
}

func TestOnlineCountBroadcast(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}

	h.handle(registerCmd{conn: ca})
	if oc, ok := lastOf[wire.OnlineCount](ca); !ok || oc.N != 1 {
		t.Fatalf("after register a, count = %+v ok=%v, want 1", oc, ok)
	}
	h.handle(registerCmd{conn: cb})
	if oc, _ := lastOf[wire.OnlineCount](cb); oc.N != 2 {
		t.Errorf("after register b, count = %d, want 2", oc.N)
	}
	h.handle(unregisterCmd{conn: ca})
	if oc, _ := lastOf[wire.OnlineCount](cb); oc.N != 1 {
		t.Errorf("after unregister a, count = %d, want 1", oc.N)
	}
}

func TestReRegisterClosesOldConn(t *testing.T) {
	h := newTestHub()
	c1 := &fakeConn{uid: "a", name: "Alice"}
	c2 := &fakeConn{uid: "a", name: "Alice"}

	h.handle(registerCmd{conn: c1})
	h.handle(registerCmd{conn: c2})

	if !c1.closed {
		t.Error("old conn should be closed on re-register (newest wins)")
	}
	if oc, _ := lastOf[wire.OnlineCount](c2); oc.N != 1 {
		t.Errorf("online count = %d, want 1 (unchanged)", oc.N)
	}
	h.handle(unregisterCmd{conn: c1})
	if _, ok := h.online["a"]; !ok {
		t.Error("re-registered user must still be online after stale unregister")
	}
}

func TestGameEndedFreesPlayers(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})
	h.handle(queueJoinCmd{userID: "a", base: 300, increment: 0})
	h.handle(queueJoinCmd{userID: "b", base: 300, increment: 0})

	h.handle(gameEndedCmd{gameID: "game-1"})

	h.handle(queueJoinCmd{userID: "a", base: 300, increment: 0})
	if _, ok := lastOf[wire.QueueWaiting](ca); !ok {
		t.Error("a should be able to re-queue after game ended")
	}
}
