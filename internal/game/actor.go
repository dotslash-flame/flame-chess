package game

import (
	"strings"
	"time"

	"github.com/notnil/chess"

	"github.com/dotslash-flame/flame-chess/internal/wire"
)

const maxChatRunes = 500

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
	rec   Recorder
	chat  ChatRecorder

	hasOffer  bool
	offeredBy chess.Color
	finished  bool

	graceSecs    int
	disconnected map[chess.Color]bool
	graceTimer   *time.Timer
	graceColor   chess.Color

	spectators map[string]wire.Conn
}

type EndInfo struct {
	Result   string
	Reason   string
	PGN      string
	Category Category
	Rated    bool
}

type Recorder interface {
	Record(EndInfo) *wire.GameRatings
}

type ChatRecorder interface {
	RecordChat(senderID, body string)
}

// this func wires a game to its two player conns. white/black are the
// connection handles for those colors. graceSecs is the reconnect grace
// window (0 → abandon immediately on disconnect). chat may be nil.
func NewActor(id string, g *Game, white, black wire.Conn, onEnd func(gameID string), rec Recorder, graceSecs int, chat ChatRecorder) *Actor {
	return &Actor{
		id:           id,
		game:         g,
		conns:        map[chess.Color]wire.Conn{chess.White: white, chess.Black: black},
		color:        map[string]chess.Color{white.UserID(): chess.White, black.UserID(): chess.Black},
		cmds:         make(chan command, 16),
		now:          time.Now,
		onEnd:        onEnd,
		rec:          rec,
		chat:         chat,
		graceSecs:    graceSecs,
		disconnected: map[chess.Color]bool{},
		spectators:   map[string]wire.Conn{},
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

func (a *Actor) Rejoin(userID string, conn wire.Conn) {
	a.cmds <- rejoinCmd{userID: userID, conn: conn}
}

func (a *Actor) Chat(userID, text string) { a.cmds <- chatCmd{userID: userID, text: text} }

func (a *Actor) AddSpectator(userID string, conn wire.Conn) {
	a.cmds <- addSpectatorCmd{userID: userID, conn: conn}
}

func (a *Actor) RemoveSpectator(userID string) { a.cmds <- removeSpectatorCmd{userID: userID} }

func (a *Actor) Run() {
	defer a.recoverGuard()
	a.timer = time.NewTimer(a.remaining())
	defer a.timer.Stop()
	for !a.finished {
		var graceC <-chan time.Time
		if a.graceTimer != nil {
			graceC = a.graceTimer.C
		}
		select {
		case c, ok := <-a.cmds:
			if !ok {
				return
			}
			a.handle(c)
		case <-a.timer.C:
			a.handleTimeout()
		case <-graceC:
			a.handleGraceExpired()
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
type rejoinCmd struct {
	userID string
	conn   wire.Conn
}
type chatCmd struct {
	userID string
	text   string
}
type addSpectatorCmd struct {
	userID string
	conn   wire.Conn
}
type removeSpectatorCmd struct{ userID string }

func (moveCmd) isCommand()            {}
func (resignCmd) isCommand()          {}
func (drawOfferCmd) isCommand()       {}
func (drawRespondCmd) isCommand()     {}
func (playerGoneCmd) isCommand()      {}
func (rejoinCmd) isCommand()          {}
func (chatCmd) isCommand()            {}
func (addSpectatorCmd) isCommand()    {}
func (removeSpectatorCmd) isCommand() {}

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
	case rejoinCmd:
		a.handleRejoin(cmd)
	case chatCmd:
		a.handleChat(cmd)
	case addSpectatorCmd:
		a.handleAddSpectator(cmd)
	case removeSpectatorCmd:
		a.handleRemoveSpectator(cmd)
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

func (a *Actor) handlePlayerGone(c playerGoneCmd) {
	if a.game.Status() != StatusActive {
		return
	}
	col, ok := a.color[c.userID]
	if !ok || a.disconnected[col] {
		return
	}
	a.disconnected[col] = true
	msg := wire.NewOpponentDisconnected(colorName(col), a.graceSecs)
	a.sendColor(opposite(col), msg)
	a.sendSpectators(msg)

	if a.graceSecs <= 0 {
		a.abandon(col)
		return
	}

	a.graceColor = col
	a.stopGraceTimer()
	a.graceTimer = time.NewTimer(time.Duration(a.graceSecs) * time.Second)
}

func (a *Actor) handleGraceExpired() {
	a.graceTimer = nil
	col := a.graceColor
	if a.game.Status() != StatusActive || !a.disconnected[col] {
		return
	}
	a.abandon(col)
}

func (a *Actor) abandon(loser chess.Color) {
	if err := a.game.AbandonedBy(loser); err != nil {
		return
	}
	a.finish()
}

func (a *Actor) handleRejoin(c rejoinCmd) {
	col, ok := a.color[c.userID]
	if !ok {
		if c.conn != nil {
			c.conn.Send(wire.NewError(wire.CodeNotInGame, "not a player in this game"))
		}
		return
	}
	if a.game.Status() != StatusActive {
		if c.conn != nil {
			c.conn.Send(wire.GameOver{
				Type:   wire.TypeGameOver,
				GameID: a.id,
				Result: a.game.Result(),
				Reason: a.game.Reason(),
			})
		}
		return
	}
	a.conns[col] = c.conn
	delete(a.disconnected, col)
	if a.graceColor == col {
		a.stopGraceTimer()
	}
	c.conn.Send(wire.GameStart{
		Type:     wire.TypeGameStart,
		GameID:   a.id,
		Color:    colorName(col),
		Opponent: a.opponentName(col),
		Clocks: wire.Clocks{
			WhiteMs: a.game.RemainingMillis(chess.White, a.now()),
			BlackMs: a.game.RemainingMillis(chess.Black, a.now()),
		},
		FEN: a.game.FEN(),
	})
	c.conn.Send(a.stateSnapshot())
	msg := wire.NewOpponentReconnected(colorName(col))
	a.sendColor(opposite(col), msg)
	a.sendSpectators(msg)
}

func (a *Actor) handleChat(c chatCmd) {
	col, ok := a.color[c.userID]
	if !ok {
		return
	}
	text := strings.TrimSpace(c.text)
	if text == "" {
		return
	}
	if r := []rune(text); len(r) > maxChatRunes {
		text = string(r[:maxChatRunes])
	}
	fromName := ""
	if conn, ok := a.conns[col]; ok && conn != nil {
		fromName = conn.DisplayName()
	}
	a.broadcast(wire.NewChatMsg(a.id, c.userID, fromName, text, a.now().UnixMilli()))
	if a.chat != nil {
		a.chat.RecordChat(c.userID, text)
	}
}

func (a *Actor) handleAddSpectator(c addSpectatorCmd) {
	if c.conn == nil {
		return
	}
	a.spectators[c.userID] = c.conn
	c.conn.Send(wire.GameStart{
		Type:     wire.TypeGameStart,
		GameID:   a.id,
		Color:    wire.ColorSpectator,
		Opponent: "",
		White:    a.playerName(chess.White),
		Black:    a.playerName(chess.Black),
		Clocks: wire.Clocks{
			WhiteMs: a.game.RemainingMillis(chess.White, a.now()),
			BlackMs: a.game.RemainingMillis(chess.Black, a.now()),
		},
		FEN: a.game.FEN(),
	})
	c.conn.Send(a.stateSnapshot())
}

func (a *Actor) handleRemoveSpectator(c removeSpectatorCmd) {
	delete(a.spectators, c.userID)
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
	var ratings *wire.GameRatings
	if a.rec != nil {
		ratings = a.rec.Record(EndInfo{
			Result:   a.game.Result(),
			Reason:   a.game.Reason(),
			PGN:      a.game.PGN(),
			Category: a.game.Category(),
			Rated:    a.game.Rated(),
		})
	}
	a.broadcast(wire.GameOver{
		Type:    wire.TypeGameOver,
		GameID:  a.id,
		Result:  a.game.Result(),
		Reason:  a.game.Reason(),
		Ratings: ratings,
	})
	if a.timer != nil {
		a.timer.Stop()
	}
	a.stopGraceTimer()
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
		if c != nil {
			c.Send(v)
		}
	}
	a.sendSpectators(v)
}

func (a *Actor) sendSpectators(v any) {
	for _, c := range a.spectators {
		if c != nil {
			c.Send(v)
		}
	}
}

func (a *Actor) stopGraceTimer() {
	if a.graceTimer == nil {
		return
	}
	if !a.graceTimer.Stop() {
		select {
		case <-a.graceTimer.C:
		default:
		}
	}
	a.graceTimer = nil
}

func (a *Actor) stateSnapshot() wire.GameState {
	return wire.GameState{
		Type:    wire.TypeGameState,
		GameID:  a.id,
		FEN:     a.game.FEN(),
		WhiteMs: a.game.RemainingMillis(chess.White, a.now()),
		BlackMs: a.game.RemainingMillis(chess.Black, a.now()),
		Turn:    colorName(a.game.Turn()),
	}
}

func (a *Actor) playerName(col chess.Color) string {
	if c, ok := a.conns[col]; ok && c != nil {
		return c.DisplayName()
	}
	return ""
}

func (a *Actor) opponentName(col chess.Color) string { return a.playerName(opposite(col)) }

func (a *Actor) sendColor(col chess.Color, v any) {
	if c, ok := a.conns[col]; ok && c != nil {
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
