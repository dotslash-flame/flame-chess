package hub

import (
	"sync"
	"testing"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/wire"
)

type lockedConn struct {
	uid, name string
	mu        sync.Mutex
	sent      []any
	closed    bool
}

func (c *lockedConn) UserID() string      { return c.uid }
func (c *lockedConn) DisplayName() string { return c.name }
func (c *lockedConn) Send(v any)          { c.mu.Lock(); c.sent = append(c.sent, v); c.mu.Unlock() }
func (c *lockedConn) Close()              { c.mu.Lock(); c.closed = true; c.mu.Unlock() }

func (c *lockedConn) count(match func(any) bool) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, v := range c.sent {
		if match(v) {
			n++
		}
	}
	return n
}

func eventually(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}

func TestReconnectReattachesNewConn(t *testing.T) {
	h := New(Options{
		Rng:    func(int) int { return 0 },
		NextID: func() string { return "game-1" },
		Now:    func() time.Time { return time.Unix(0, 0) },
	})
	ca := &lockedConn{uid: "a", name: "Alice"}
	cb := &lockedConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})
	h.handle(queueJoinCmd{userID: "a", base: 300, increment: 0})
	h.handle(queueJoinCmd{userID: "b", base: 300, increment: 0})

	a2 := &lockedConn{uid: "a", name: "Alice"}
	h.handle(registerCmd{conn: a2})

	isStart := func(v any) bool { _, ok := v.(wire.GameStart); return ok }
	eventually(t, func() bool { return a2.count(isStart) >= 1 },
		"reconnecting socket never received a rejoin game.start")
	if !ca.closed {
		t.Error("the stale socket should be closed on reconnect")
	}
}
func rematchSetup(t *testing.T) (*Hub, *fakeConn, *fakeConn) {
	t.Helper()
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})
	h.handle(queueJoinCmd{userID: "a", base: 300, increment: 0})
	h.handle(queueJoinCmd{userID: "b", base: 300, increment: 0})
	h.handle(gameEndedCmd{gameID: "game-1"})
	return h, ca, cb
}

func TestRematchOfferAcceptSwapsColors(t *testing.T) {
	h, ca, cb := rematchSetup(t)
	if s, _ := lastOf[wire.GameStart](ca); s.Color != "white" {
		t.Fatalf("pre-rematch a color = %q, want white", s.Color)
	}

	h.handle(offerRematchCmd{userID: "a", gameID: "game-1"})
	if o, ok := lastOf[wire.RematchOffered](cb); !ok || o.From != "a" {
		t.Fatalf("b should receive rematch.offered from a, got %+v ok=%v", o, ok)
	}
	h.handle(respondRematchCmd{userID: "b", gameID: "game-1", accept: true})

	sa, _ := lastOf[wire.GameStart](ca)
	sb, _ := lastOf[wire.GameStart](cb)
	if sa.Color != "black" || sb.Color != "white" {
		t.Errorf("rematch colors a/b = %q/%q, want black/white (swapped)", sa.Color, sb.Color)
	}
	if _, live := h.rematches["game-1"]; live {
		t.Error("rematch ctx must be consumed on accept")
	}
}

func TestRematchMutualOfferStartsOnce(t *testing.T) {
	h, ca, _ := rematchSetup(t)
	h.handle(offerRematchCmd{userID: "a", gameID: "game-1"})
	h.handle(offerRematchCmd{userID: "b", gameID: "game-1"})

	if countOf[wire.GameStart](ca) != 2 {
		t.Errorf("a game.start count = %d, want 2 (mutual offer starts exactly once)", countOf[wire.GameStart](ca))
	}
}

func TestRematchDeclineNotifiesOfferer(t *testing.T) {
	h, ca, _ := rematchSetup(t)
	h.handle(offerRematchCmd{userID: "a", gameID: "game-1"})
	h.handle(respondRematchCmd{userID: "b", gameID: "game-1", accept: false})

	if d, ok := lastOf[wire.RematchDeclined](ca); !ok || d.GameID != "game-1" {
		t.Errorf("offerer should get rematch.declined, got %+v ok=%v", d, ok)
	}
	if _, live := h.rematches["game-1"]; live {
		t.Error("declined rematch ctx must be dropped")
	}
}

func TestRematchOffererDisconnectInvalidates(t *testing.T) {
	h, _, cb := rematchSetup(t)
	h.handle(offerRematchCmd{userID: "a", gameID: "game-1"})

	h.handle(unregisterCmd{conn: &fakeConn{uid: "a", name: "Alice"}})
	h.handle(unregisterCmd{conn: h.online["a"]})

	if _, live := h.rematches["game-1"]; live {
		t.Error("offerer disconnect must invalidate the rematch ctx")
	}
	if d, ok := lastOf[wire.RematchDeclined](cb); !ok || d.GameID != "game-1" {
		t.Errorf("waiting opponent should get rematch.declined, got %+v ok=%v", d, ok)
	}
}

func TestRematchUnknownGameUnavailable(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	h.handle(registerCmd{conn: ca})

	h.handle(offerRematchCmd{userID: "a", gameID: "nope"})

	if e, ok := lastOf[wire.Error](ca); !ok || e.Code != wire.CodeRematchUnavailable {
		t.Errorf("error = %+v ok=%v, want rematch_unavailable", e, ok)
	}
}

func TestSpectateJoinLeaveAttaches(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	cc := &fakeConn{uid: "c", name: "Cara"}
	for _, c := range []*fakeConn{ca, cb, cc} {
		h.handle(registerCmd{conn: c})
	}
	h.handle(queueJoinCmd{userID: "a", base: 300, increment: 0})
	h.handle(queueJoinCmd{userID: "b", base: 300, increment: 0})

	h.handle(spectateJoinCmd{userID: "c", gameID: "game-1"})
	if h.spectatorGame["c"] != "game-1" {
		t.Fatalf("spectatorGame[c] = %q, want game-1", h.spectatorGame["c"])
	}
	h.handle(spectateLeaveCmd{userID: "c"})
	if _, ok := h.spectatorGame["c"]; ok {
		t.Error("spectate.leave should detach the watcher")
	}
}

func TestSpectateOwnGameRejected(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})
	h.handle(queueJoinCmd{userID: "a", base: 300, increment: 0})
	h.handle(queueJoinCmd{userID: "b", base: 300, increment: 0})

	h.handle(spectateJoinCmd{userID: "a", gameID: "game-1"})

	if e, ok := lastOf[wire.Error](ca); !ok || e.Code != wire.CodeUnknownGame {
		t.Errorf("error = %+v ok=%v, want unknown_game for spectating own game", e, ok)
	}
	if _, ok := h.spectatorGame["a"]; ok {
		t.Error("a player must not be registered as a spectator of their own game")
	}
}

func TestSpectateUnknownGameRejected(t *testing.T) {
	h := newTestHub()
	cc := &fakeConn{uid: "c", name: "Cara"}
	h.handle(registerCmd{conn: cc})

	h.handle(spectateJoinCmd{userID: "c", gameID: "nope"})

	if e, ok := lastOf[wire.Error](cc); !ok || e.Code != wire.CodeUnknownGame {
		t.Errorf("error = %+v ok=%v, want unknown_game", e, ok)
	}
}

func TestGamesLiveBroadcastOnStartAndEnd(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})

	h.handle(queueJoinCmd{userID: "a", base: 300, increment: 0})
	h.handle(queueJoinCmd{userID: "b", base: 300, increment: 0})

	gl, ok := lastOf[wire.GamesLive](ca)
	if !ok || len(gl.Games) != 1 {
		t.Fatalf("after start, games.live = %+v ok=%v, want 1 game", gl, ok)
	}
	g := gl.Games[0]
	if g.GameID != "game-1" || g.White != "Alice" || g.Black != "Bob" {
		t.Errorf("live game = %+v, want game-1/Alice/Bob", g)
	}

	h.handle(gameEndedCmd{gameID: "game-1"})
	if gl, _ := lastOf[wire.GamesLive](ca); len(gl.Games) != 0 {
		t.Errorf("after end, games.live has %d games, want 0", len(gl.Games))
	}
}
