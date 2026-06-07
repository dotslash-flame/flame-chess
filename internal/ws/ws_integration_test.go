package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/dotslash-flame/flame-chess/internal/auth"
	"github.com/dotslash-flame/flame-chess/internal/hub"
	"github.com/dotslash-flame/flame-chess/internal/wire"
	"github.com/dotslash-flame/flame-chess/internal/ws"
)

const secret = "integration-secret"

func dialClient(t *testing.T, serverURL, name string) *websocket.Conn {
	t.Helper()
	id := auth.Identity{UserID: auth.UserIDForName(name), DisplayName: name}
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	hdr := http.Header{}
	hdr.Set("Cookie", ws.SessionCookie+"="+auth.Sign(id, secret))
	c, _, err := websocket.Dial(context.Background(), wsURL+"/ws", &websocket.DialOptions{HTTPHeader: hdr})
	if err != nil {
		t.Fatalf("dial %s: %v", name, err)
	}
	t.Cleanup(func() { _ = c.CloseNow() })
	return c
}

func readUntil(t *testing.T, c *websocket.Conn, typ string) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			t.Fatalf("waiting for %q: %v", typ, err)
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		if m["type"] == typ {
			return m
		}
	}
}

func send(t *testing.T, c *websocket.Conn, v any) {
	t.Helper()
	data, _ := json.Marshal(v)
	if err := c.Write(context.Background(), websocket.MessageText, data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	h := hub.New(hub.Options{NextID: func() string { return "g1" }})
	mux := http.NewServeMux()
	mux.Handle("GET /ws", ws.Handler(h, secret))
	done := make(chan struct{})
	go h.Run(done)
	srv := httptest.NewServer(mux)
	t.Cleanup(func() { close(done); srv.Close() })
	return srv
}

func TestTwoClientFoolsMate(t *testing.T) {
	srv := newServer(t)
	ca := dialClient(t, srv.URL, "Alice")
	cb := dialClient(t, srv.URL, "Bob")

	send(t, ca, wire.QueueJoin{Type: wire.TypeQueueJoin, Category: "blitz", Base: 60, Increment: 0})
	send(t, cb, wire.QueueJoin{Type: wire.TypeQueueJoin, Category: "blitz", Base: 60, Increment: 0})

	sa := readUntil(t, ca, wire.TypeGameStart)
	sb := readUntil(t, cb, wire.TypeGameStart)
	if sa["color"] == sb["color"] {
		t.Fatalf("both got color %v, want opposite", sa["color"])
	}
	white, black := ca, cb
	if sa["color"] == "black" {
		white, black = cb, ca
	}

	moves := []struct {
		c   *websocket.Conn
		uci string
	}{
		{white, "f2f3"},
		{black, "e7e5"},
		{white, "g2g4"},
		{black, "d8h4"},
	}
	for _, m := range moves {
		send(t, m.c, wire.Move{Type: wire.TypeMove, GameID: "g1", UCI: m.uci})
		readUntil(t, m.c, wire.TypeGameState)
	}

	for _, c := range []*websocket.Conn{ca, cb} {
		over := readUntil(t, c, wire.TypeGameOver)
		if over["result"] != "0-1" || over["reason"] != "checkmate" {
			t.Errorf("game.over = %v, want 0-1/checkmate", over)
		}
	}
}

func TestTwoClientTimeout(t *testing.T) {
	srv := newServer(t)
	ca := dialClient(t, srv.URL, "Alice")
	cb := dialClient(t, srv.URL, "Bob")

	send(t, ca, wire.QueueJoin{Type: wire.TypeQueueJoin, Category: "bullet", Base: 1, Increment: 0})
	send(t, cb, wire.QueueJoin{Type: wire.TypeQueueJoin, Category: "bullet", Base: 1, Increment: 0})

	readUntil(t, ca, wire.TypeGameStart)
	readUntil(t, cb, wire.TypeGameStart)

	for _, c := range []*websocket.Conn{ca, cb} {
		over := readUntil(t, c, wire.TypeGameOver)
		if over["reason"] != "timeout" {
			t.Errorf("reason = %v, want timeout", over["reason"])
		}
	}
}
