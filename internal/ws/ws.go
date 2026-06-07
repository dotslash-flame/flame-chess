package ws

//ws package is the trnsport layer, it holds the authenticted requiests.
// translates ws frames(data) to hub commands
import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/dotslash-flame/flame-chess/internal/auth"
	"github.com/dotslash-flame/flame-chess/internal/hub"
	"github.com/dotslash-flame/flame-chess/internal/wire"
)

const SessionCookie = "fc_session"

type Router interface {
	Register(wire.Conn)
	Unregister(wire.Conn)
	QueueJoin(userID, category string, base, increment int)
	QueueLeave(userID string)
	Route(userID string, a hub.GameAction)
}

type Conn struct {
	sock   *websocket.Conn
	id     auth.Identity
	router Router

	send   chan []byte
	ctx    context.Context
	cancel context.CancelFunc

	closed    chan struct{}
	closeOnce sync.Once
}

func newConn(sock *websocket.Conn, id auth.Identity, router Router) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	return &Conn{
		sock:   sock,
		id:     id,
		router: router,
		send:   make(chan []byte, 32),
		ctx:    ctx,
		cancel: cancel,
		closed: make(chan struct{}),
	}
}

func (c *Conn) UserID() string      { return c.id.UserID }
func (c *Conn) DisplayName() string { return c.id.DisplayName }

func (c *Conn) Send(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case <-c.closed:
	case c.send <- data:
	default:
		c.Close()
	}
}

func (c *Conn) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		c.cancel()
		_ = c.sock.CloseNow()
	})
}

func (c *Conn) readPump() {
	for {
		_, data, err := c.sock.Read(c.ctx)
		if err != nil {
			return
		}
		c.dispatch(data)
	}
}

func (c *Conn) writePump() {
	for {
		select {
		case <-c.closed:
			return
		case data := <-c.send:
			wctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			err := c.sock.Write(wctx, websocket.MessageText, data)
			cancel()
			if err != nil {
				c.Close()
				return
			}
		}
	}
}

func (c *Conn) dispatch(raw []byte) {
	typ, err := wire.DecodeType(raw)
	if err != nil {
		c.Send(wire.NewError(wire.CodeBadMessage, "malformed message"))
		return
	}
	switch typ {
	case wire.TypeQueueJoin:
		var m wire.QueueJoin
		if json.Unmarshal(raw, &m) != nil {
			c.Send(wire.NewError(wire.CodeBadMessage, "bad queue.join"))
			return
		}
		c.router.QueueJoin(c.id.UserID, m.Category, m.Base, m.Increment)
	case wire.TypeQueueLeave:
		c.router.QueueLeave(c.id.UserID)
	case wire.TypeMove:
		var m wire.Move
		if json.Unmarshal(raw, &m) != nil {
			c.Send(wire.NewError(wire.CodeBadMessage, "bad move"))
			return
		}
		c.router.Route(c.id.UserID, hub.GameAction{GameID: m.GameID, Kind: "move", UCI: m.UCI})
	case wire.TypeResign:
		var m wire.Resign
		_ = json.Unmarshal(raw, &m)
		c.router.Route(c.id.UserID, hub.GameAction{GameID: m.GameID, Kind: "resign"})
	case wire.TypeDrawOffer:
		var m wire.DrawOffer
		_ = json.Unmarshal(raw, &m)
		c.router.Route(c.id.UserID, hub.GameAction{GameID: m.GameID, Kind: "draw_offer"})
	case wire.TypeDrawRespond:
		var m wire.DrawRespond
		_ = json.Unmarshal(raw, &m)
		c.router.Route(c.id.UserID, hub.GameAction{GameID: m.GameID, Kind: "draw_respond", Accept: m.Accept})
	case wire.TypePing:
		c.Send(wire.NewPong())
	default:
		c.Send(wire.NewError(wire.CodeBadMessage, "unknown type"))
	}
}

func Handler(router Router, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := identity(r, secret)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		sock, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		conn := newConn(sock, id, router)
		router.Register(conn)
		go conn.writePump()
		conn.readPump()
		conn.Close()
		router.Unregister(conn)
	}
}

func identity(r *http.Request, secret string) (auth.Identity, error) {
	ck, err := r.Cookie(SessionCookie)
	if err != nil {
		return auth.Identity{}, err
	}
	return auth.Verify(ck.Value, secret)
}
