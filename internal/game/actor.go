package game

import (
	"time"

	"github.com/notnil/chess"

	"github.com/dotslash-flame/flame-chess/internal/wire"
)

// 1 actor = 1 live game
//
//	A single goroutin processes the commands and the timeout timer
type Actor struct {
	id    string
	game  *Game
	conns map[chess.Color]wire.Conn
	color map[string]chess.Color

	cmds  chan command
	now   func() time.Time
	timer *time.Timer
	onEnd func(gameID string)

	hasOffer  bool
	offeredBy chess.Color
	finished  bool
}

// this func wires a game to its two player conns. white/black are the
// connection handles for those colors
func NewActor(id string, g *Game, white, black wire.Conn, onEnd func(gameID string)) *Actor {
	return &Actor{
		id:    id,
		game:  g,
		conns: map[chess.Color]wire.Conn{chess.White: white, chess.Black: black},
		color: map[string]chess.Color{white.UserID(): chess.White, black.UserID(): chess.Black},
		cmds:  make(chan command, 16),
		now:   time.Now,
		onEnd: onEnd,
	}
}

func (a *Actor) ID() string { return a.id }

func (a *Actor) Move(userID, uci string) { a.cmds <- moveCmd{userID: userID, uci: uci} }
func (a *Actor) Resign(userID string)    { a.cmds <- resignCmd{userID: userID} }
func (a *Actor) OfferDraw(userID string) { a.cmds <- drawOfferCmd{userID: userID} }
func (a *Actor) RespondDraw(userID string, ok bool) {
	a.cmds <- drawRespondCmd{userID: userID, accept: ok}
}
func (a *Actor) PlayerGone(userID string) { a.cmds <- playerGoneCmd{userID: userID} }

func (a *Actor) Run() {
	defer a.recoverGuard()
	a.timer = time.NewTimer(a.remaining())
	defer a.timer.Stop()
	for !a.finished {
		select {
		case c, ok := <-a.cmds:
			if !ok {
				return
			}
			a.handle(c)
		case <-a.timer.C:
			a.handleTimeout()
		}
	}
}

func (a *Actor) recoverGuard() {
	if r := recover(); r != nil {
		if a.onEnd != nil {
			a.onEnd(a.id)
		}
	}
}

type command interface{ isCommand() }

type moveCmd struct {
	userID string
	uci    string
}
type resignCmd struct{ userID string }
type drawOfferCmd struct{ userID string }
type drawRespondCmd struct {
	userID string
	accept bool
}
type playerGoneCmd struct{ userID string }

func (moveCmd) isCommand()        {}
func (resignCmd) isCommand()      {}
func (drawOfferCmd) isCommand()   {}
func (drawRespondCmd) isCommand() {}
func (playerGoneCmd) isCommand()  {}

func (a *Actor) handle(c command) {
	switch cmd := c.(type) {
	case moveCmd:
		a.handleMove(cmd)
	case resignCmd:
		a.handleResign(cmd)
	case drawOfferCmd:
		a.handleDrawOffer(cmd)
	case drawRespondCmd:
		a.handleDrawRespond(cmd)
	case playerGoneCmd:
		a.handlePlayerGone(cmd)
	}
}

func (a *Actor) handleMove(c moveCmd) {
	if a.game.Status() != StatusActive {
		a.sendTo(c.userID, wire.NewError(wire.CodeGameNotActive, "game is not active"))
		return
	}
	col, ok := a.color[c.userID]
	if !ok {
		a.sendTo(c.userID, wire.NewError(wire.CodeNotInGame, "not a player in this game"))
		return
	}
	if col != a.game.Turn() {
		a.sendTo(c.userID, wire.NewError(wire.CodeNotYourTurn, "not your turn"))
		return
	}
	if err := a.game.Move(c.uci, a.now()); err != nil {
		a.sendTo(c.userID, wire.NewError(wire.CodeIllegalMove, "illegal move"))
		return
	}
	if a.game.Status() != StatusActive {
		if a.game.Reason() != "timeout" {
			a.broadcastState(c.uci)
		}
		a.finish()
		return
	}
	a.hasOffer = false
	a.broadcastState(c.uci)
	a.resetTimer()
}

func (a *Actor) handleResign(c resignCmd) {
	if a.game.Status() != StatusActive {
		a.sendTo(c.userID, wire.NewError(wire.CodeGameNotActive, "game is not active"))
		return
	}
	col, ok := a.color[c.userID]
	if !ok {
		a.sendTo(c.userID, wire.NewError(wire.CodeNotInGame, "not a player in this game"))
		return
	}
	if err := a.game.Resign(col); err != nil {
		return
	}
	a.finish()
}

func (a *Actor) handleDrawOffer(c drawOfferCmd) {
	if a.game.Status() != StatusActive {
		a.sendTo(c.userID, wire.NewError(wire.CodeGameNotActive, "game is not active"))
		return
	}
	col, ok := a.color[c.userID]
	if !ok {
		a.sendTo(c.userID, wire.NewError(wire.CodeNotInGame, "not a player in this game"))
		return
	}
	a.hasOffer = true
	a.offeredBy = col
	a.sendColor(opposite(col), wire.DrawOffered{
		Type:   wire.TypeDrawOffered,
		GameID: a.id,
		From:   c.userID,
	})
}

func (a *Actor) handleDrawRespond(c drawRespondCmd) {
	if a.game.Status() != StatusActive {
		return
	}
	col, ok := a.color[c.userID]
	if !ok {
		return
	}
	if !a.hasOffer || a.offeredBy == col {
		return
	}
	if !c.accept {
		a.hasOffer = false
		return
	}
	if err := a.game.AgreeDraw(); err != nil {
		return
	}
	a.finish()
}

func (a *Actor) handlePlayerGone(playerGoneCmd) {
	//implement later, right now focusing on timer to end game, pretty sad but theres no solid SINGLE workaroud
	//basically, th eonline player will have to keep waiting til the timer runs out
}

func (a *Actor) handleTimeout() {
	if a.game.Status() != StatusActive {
		return
	}
	if a.game.TimeoutCheck(a.now()) {
		a.finish()
		return
	}
	a.resetTimer()
}

func (a *Actor) finish() {
	if a.finished {
		return
	}
	a.finished = true
	a.broadcast(wire.GameOver{
		Type:   wire.TypeGameOver,
		GameID: a.id,
		Result: a.game.Result(),
		Reason: a.game.Reason(),
	})
	if a.timer != nil {
		a.timer.Stop()
	}
	if a.onEnd != nil {
		a.onEnd(a.id)
	}
}

func (a *Actor) broadcastState(lastMove string) {
	a.broadcast(wire.GameState{
		Type:     wire.TypeGameState,
		GameID:   a.id,
		FEN:      a.game.FEN(),
		LastMove: lastMove,
		WhiteMs:  a.game.RemainingMillis(chess.White, a.now()),
		BlackMs:  a.game.RemainingMillis(chess.Black, a.now()),
		Turn:     colorName(a.game.Turn()),
	})
}

func (a *Actor) broadcast(v any) {
	for _, c := range a.conns {
		c.Send(v)
	}
}

func (a *Actor) sendColor(col chess.Color, v any) {
	if c, ok := a.conns[col]; ok {
		c.Send(v)
	}
}

func (a *Actor) sendTo(userID string, v any) {
	if col, ok := a.color[userID]; ok {
		a.sendColor(col, v)
	}
}

func (a *Actor) remaining() time.Duration {
	ms := a.game.RemainingMillis(a.game.Turn(), a.now())
	if ms < 0 {
		ms = 0
	}
	return time.Duration(ms) * time.Millisecond
}

func (a *Actor) resetTimer() {
	if a.timer == nil {
		return
	}
	if !a.timer.Stop() {
		select {
		case <-a.timer.C:
		default:
		}
	}
	a.timer.Reset(a.remaining())
}

func opposite(c chess.Color) chess.Color {
	if c == chess.White {
		return chess.Black
	}
	return chess.White
}

func colorName(c chess.Color) string {
	if c == chess.White {
		return "white"
	}
	return "black"
}
