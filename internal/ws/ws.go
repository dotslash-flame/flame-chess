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
	CreateDirectChallenge(userID, opponentID string, base, increment int)
	AcceptChallenge(userID, token string)
	DeclineChallenge(userID, token string)
	CancelChallenge(userID, token string)
	OfferRematch(userID, gameID string)
	RespondRematch(userID, gameID string, accept bool)
	SpectateJoin(userID, gameID string)
	SpectateLeave(userID string)
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
	case wire.TypeChallengeCreateDirect:
		var m wire.ChallengeCreateDirect
		if json.Unmarshal(raw, &m) != nil {
			c.Send(wire.NewError(wire.CodeBadMessage, "bad challenge.create_direct"))
			return
		}
		c.router.CreateDirectChallenge(c.id.UserID, m.OpponentID, m.Base, m.Increment)
	case wire.TypeChallengeAccept:
		var m wire.ChallengeAccept
		if json.Unmarshal(raw, &m) != nil {
			c.Send(wire.NewError(wire.CodeBadMessage, "bad challenge.accept"))
			return
		}
		c.router.AcceptChallenge(c.id.UserID, m.Token)
	case wire.TypeChallengeDecline:
		var m wire.ChallengeDecline
		if json.Unmarshal(raw, &m) != nil {
			c.Send(wire.NewError(wire.CodeBadMessage, "bad challenge.decline"))
			return
		}
		c.router.DeclineChallenge(c.id.UserID, m.Token)
	case wire.TypeChallengeCancel:
		var m wire.ChallengeCancel
		if json.Unmarshal(raw, &m) != nil {
			c.Send(wire.NewError(wire.CodeBadMessage, "bad challenge.cancel"))
			return
		}
		c.router.CancelChallenge(c.id.UserID, m.Token)
	case wire.TypeRematchOffer:
		var m wire.RematchOffer
		if json.Unmarshal(raw, &m) != nil {
			c.Send(wire.NewError(wire.CodeBadMessage, "bad rematch.offer"))
			return
		}
		c.router.OfferRematch(c.id.UserID, m.GameID)
	case wire.TypeRematchRespond:
		var m wire.RematchRespond
		if json.Unmarshal(raw, &m) != nil {
			c.Send(wire.NewError(wire.CodeBadMessage, "bad rematch.respond"))
			return
		}
		c.router.RespondRematch(c.id.UserID, m.GameID, m.Accept)
	case wire.TypeChatSend:
		var m wire.ChatSend
		if json.Unmarshal(raw, &m) != nil {
			c.Send(wire.NewError(wire.CodeBadMessage, "bad chat.send"))
			return
		}
		c.router.Route(c.id.UserID, hub.GameAction{GameID: m.GameID, Kind: "chat", Text: m.Text})
	case wire.TypeSpectateJoin:
		var m wire.SpectateJoin
		if json.Unmarshal(raw, &m) != nil {
			c.Send(wire.NewError(wire.CodeBadMessage, "bad spectate.join"))
			return
		}
		c.router.SpectateJoin(c.id.UserID, m.GameID)
	case wire.TypeSpectateLeave:
		c.router.SpectateLeave(c.id.UserID)
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
