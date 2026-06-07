package ws

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dotslash-flame/flame-chess/internal/auth"
	"github.com/dotslash-flame/flame-chess/internal/hub"
	"github.com/dotslash-flame/flame-chess/internal/wire"
)

type fakeRouter struct {
	joins  []wire.QueueJoin
	routes []hub.GameAction
}

func (f *fakeRouter) Register(wire.Conn)   {}
func (f *fakeRouter) Unregister(wire.Conn) {}
func (f *fakeRouter) QueueJoin(_ , category string, base, increment int) {
	f.joins = append(f.joins, wire.QueueJoin{Category: category, Base: base, Increment: increment})
}
func (f *fakeRouter) QueueLeave(string)                  {}
func (f *fakeRouter) Route(_ string, a hub.GameAction)   { f.routes = append(f.routes, a) }

func newTestConn(r Router) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	return &Conn{
		id:     auth.Identity{UserID: "u-1", DisplayName: "Alice"},
		router: r,
		send:   make(chan []byte, 32),
		ctx:    ctx,
		cancel: cancel,
		closed: make(chan struct{}),
	}
}

func TestDispatchMoveRoutes(t *testing.T) {
	fr := &fakeRouter{}
	c := newTestConn(fr)

	c.dispatch([]byte(`{"type":"move","game_id":"g1","uci":"e2e4"}`))

	if len(fr.routes) != 1 {
		t.Fatalf("routes = %d, want 1", len(fr.routes))
	}
	got := fr.routes[0]
	if got.Kind != "move" || got.UCI != "e2e4" || got.GameID != "g1" {
		t.Errorf("route = %+v, want move/e2e4/g1", got)
	}
}

func TestDispatchQueueJoin(t *testing.T) {
	fr := &fakeRouter{}
	c := newTestConn(fr)

	c.dispatch([]byte(`{"type":"queue.join","category":"blitz","base":300,"increment":0}`))

	if len(fr.joins) != 1 || fr.joins[0].Base != 300 {
		t.Fatalf("joins = %+v, want one base=300", fr.joins)
	}
}

func TestDispatchMalformedSendsBadMessage(t *testing.T) {
	fr := &fakeRouter{}
	c := newTestConn(fr)

	c.dispatch([]byte(`not json`))

	select {
	case data := <-c.send:
		var e wire.Error
		_ = json.Unmarshal(data, &e)
		if e.Code != wire.CodeBadMessage {
			t.Errorf("error code = %q, want bad_message", e.Code)
		}
	default:
		t.Fatal("expected an error frame queued")
	}
	if len(fr.routes) != 0 {
		t.Error("malformed frame must not route")
	}
}

func TestDispatchPing(t *testing.T) {
	fr := &fakeRouter{}
	c := newTestConn(fr)

	c.dispatch([]byte(`{"type":"ping"}`))

	data := <-c.send
	typ, _ := wire.DecodeType(data)
	if typ != wire.TypePong {
		t.Errorf("reply type = %q, want pong", typ)
	}
}
