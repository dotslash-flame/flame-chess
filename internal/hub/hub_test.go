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

func TestDirectChallengeCreateAndAccept(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})

	h.handle(createChallengeCmd{creator: "a", creatorName: "Alice", opponent: "b", base: 300, increment: 0})

	inc, ok := lastOf[wire.ChallengeIncoming](cb)
	if !ok {
		t.Fatal("target should receive challenge.incoming")
	}
	if inc.From != "a" || inc.FromName != "Alice" || inc.Category != "blitz" {
		t.Errorf("incoming = %+v, want from a/Alice/blitz", inc)
	}
	if _, ok := lastOf[wire.ChallengeCreated](ca); !ok {
		t.Error("creator should receive challenge.created ack")
	}

	h.handle(acceptChallengeCmd{userID: "b", token: inc.Token})

	sa, ok := lastOf[wire.GameStart](ca)
	if !ok {
		t.Fatal("creator got no game.start after accept")
	}
	sb, ok := lastOf[wire.GameStart](cb)
	if !ok {
		t.Fatal("accepter got no game.start after accept")
	}
	if sa.GameID != sb.GameID {
		t.Errorf("game ids differ: %q vs %q", sa.GameID, sb.GameID)
	}
	if _, live := h.challenges[inc.Token]; live {
		t.Error("token must be single-use (deleted after accept)")
	}
}

func TestAcceptUnknownToken(t *testing.T) {
	h := newTestHub()
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: cb})

	h.handle(acceptChallengeCmd{userID: "b", token: "nope"})

	if e, ok := lastOf[wire.Error](cb); !ok || e.Code != wire.CodeUnknownChallenge {
		t.Errorf("error = %+v ok=%v, want unknown_challenge", e, ok)
	}
}

func TestAcceptOwnLinkIsSelfChallenge(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	h.handle(registerCmd{conn: ca})

	h.handle(createChallengeCmd{creator: "a", creatorName: "Alice", opponent: "", base: 300})
	h.handle(acceptChallengeCmd{userID: "a", token: "game-1"})

	if e, ok := lastOf[wire.Error](ca); !ok || e.Code != wire.CodeChallengeSelf {
		t.Errorf("error = %+v ok=%v, want challenge_self", e, ok)
	}
}

func TestAcceptCreatorOffline(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})

	h.handle(createChallengeCmd{creator: "a", creatorName: "Alice", opponent: "b", base: 300})
	inc, _ := lastOf[wire.ChallengeIncoming](cb)

	delete(h.online, "a")
	h.handle(acceptChallengeCmd{userID: "b", token: inc.Token})

	if e, ok := lastOf[wire.Error](cb); !ok || e.Code != wire.CodeOpponentOffline {
		t.Errorf("error = %+v ok=%v, want opponent_offline", e, ok)
	}
}

func TestCreateDirectBusyTargetInGame(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})
	h.userGame["b"] = "some-game"

	h.handle(createChallengeCmd{creator: "a", creatorName: "Alice", opponent: "b", base: 300})

	if e, ok := lastOf[wire.Error](ca); !ok || e.Code != wire.CodeBusy {
		t.Errorf("error = %+v ok=%v, want busy", e, ok)
	}
	if _, got := lastOf[wire.ChallengeIncoming](cb); got {
		t.Error("busy target must not receive an incoming challenge")
	}
}

func TestDeclineNotifiesCreator(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})

	h.handle(createChallengeCmd{creator: "a", creatorName: "Alice", opponent: "b", base: 300})
	inc, _ := lastOf[wire.ChallengeIncoming](cb)
	h.handle(declineChallengeCmd{userID: "b", token: inc.Token})

	if d, ok := lastOf[wire.ChallengeDeclined](ca); !ok || d.Token != inc.Token {
		t.Errorf("declined = %+v ok=%v, want token %q", d, ok, inc.Token)
	}
	if _, live := h.challenges[inc.Token]; live {
		t.Error("declined challenge must be removed")
	}
}

func TestUnregisterCreatorSendsGone(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}
	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})

	h.handle(createChallengeCmd{creator: "a", creatorName: "Alice", opponent: "b", base: 300})
	inc, _ := lastOf[wire.ChallengeIncoming](cb)

	h.handle(unregisterCmd{conn: ca})

	if g, ok := lastOf[wire.ChallengeGone](cb); !ok || g.Token != inc.Token {
		t.Errorf("gone = %+v ok=%v, want token %q", g, ok, inc.Token)
	}
	if _, live := h.challenges[inc.Token]; live {
		t.Error("creator disconnect must drop the challenge")
	}
}

func TestOnlineListBroadcast(t *testing.T) {
	h := newTestHub()
	ca := &fakeConn{uid: "a", name: "Alice"}
	cb := &fakeConn{uid: "b", name: "Bob"}

	h.handle(registerCmd{conn: ca})
	h.handle(registerCmd{conn: cb})

	list, ok := lastOf[wire.OnlineList](ca)
	if !ok {
		t.Fatal("expected online.list after registers")
	}
	if len(list.Users) != 2 {
		t.Fatalf("online.list has %d users, want 2", len(list.Users))
	}
	names := map[string]string{}
	for _, u := range list.Users {
		names[u.UID] = u.Name
	}
	if names["a"] != "Alice" || names["b"] != "Bob" {
		t.Errorf("online.list names = %+v, want a=Alice b=Bob", names)
	}

	h.handle(unregisterCmd{conn: ca})
	if list, _ := lastOf[wire.OnlineList](cb); len(list.Users) != 1 {
		t.Errorf("after unregister, online.list has %d users, want 1", len(list.Users))
	}
}
