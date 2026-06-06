package game

import (
	"errors"
	"time"

	"github.com/notnil/chess"
)

type Status int

const (
	StatusActive Status = iota
	StatusFinished
	StatusAborted
)

var (
	ErrNotActive   = errors.New("game is not active")
	ErrIllegalMove = errors.New("illegal move")
	ErrNoClaim     = errors.New("no eligible draw to claim")
)

type Game struct {
	chess    *chess.Game
	clock    *Clock
	category Category
	status   Status
	result   string
	reason   string
}

func NewGame(baseSeconds, incrementSeconds int, startedAt time.Time) *Game {
	clk := NewClock(
		time.Duration(baseSeconds)*time.Second,
		time.Duration(incrementSeconds)*time.Second,
	)
	clk.Start(startedAt)
	return &Game{
		chess:    chess.NewGame(),
		clock:    clk,
		category: CategoryForBaseSeconds(baseSeconds),
		status:   StatusActive,
	}
}

func (g *Game) Move(uci string, now time.Time) error {
	if g.status != StatusActive {
		return ErrNotActive
	}
	if g.clock.Flagged(now) {
		g.finishTimeout()
		return nil
	}
	m, err := chess.UCINotation{}.Decode(g.chess.Position(), uci)
	if err != nil {
		return ErrIllegalMove
	}
	if err := g.chess.Move(m); err != nil {
		return ErrIllegalMove
	}
	g.clock.Press(now)
	g.syncOutcome()
	return nil
}

func (g *Game) Resign(color chess.Color) error {
	if g.status != StatusActive {
		return ErrNotActive
	}
	g.chess.Resign(color)
	g.syncOutcome()
	return nil
}

func (g *Game) AgreeDraw() error {
	if g.status != StatusActive {
		return ErrNotActive
	}
	if err := g.chess.Draw(chess.DrawOffer); err != nil {
		return err
	}
	g.syncOutcome()
	return nil
}

func (g *Game) ClaimDraw() error {
	if g.status != StatusActive {
		return ErrNotActive
	}
	for _, m := range g.chess.EligibleDraws() {
		if m == chess.ThreefoldRepetition || m == chess.FiftyMoveRule {
			if err := g.chess.Draw(m); err != nil {
				return err
			}
			g.syncOutcome()
			return nil
		}
	}
	return ErrNoClaim
}

func (g *Game) TimeoutCheck(now time.Time) bool {
	if g.status != StatusActive {
		return false
	}
	if g.clock.Flagged(now) {
		g.finishTimeout()
		return true
	}
	return false
}

func (g *Game) finishTimeout() {
	loser := g.chess.Position().Turn()
	g.status = StatusFinished
	g.reason = "timeout"
	if loser == chess.White {
		g.result = "0-1"
	} else {
		g.result = "1-0"
	}
}

func (g *Game) syncOutcome() {
	if g.chess.Outcome() == chess.NoOutcome {
		return
	}
	g.status = StatusFinished
	g.result = g.chess.Outcome().String()
	g.reason = methodReason(g.chess.Method())
}

func methodReason(m chess.Method) string {
	switch m {
	case chess.Checkmate:
		return "checkmate"
	case chess.Stalemate:
		return "stalemate"
	case chess.InsufficientMaterial:
		return "insufficient"
	case chess.ThreefoldRepetition, chess.FivefoldRepetition:
		return "threefold"
	case chess.FiftyMoveRule, chess.SeventyFiveMoveRule:
		return "fifty_move"
	case chess.Resignation:
		return "resign"
	case chess.DrawOffer:
		return "draw_agreed"
	default:
		return ""
	}
}

func newGameFromChess(cg *chess.Game) *Game {
	clk := NewClock(5*time.Minute, 0)
	clk.Start(time.Unix(0, 0))
	g := &Game{
		chess:    cg,
		clock:    clk,
		category: CategoryBlitz,
		status:   StatusActive,
	}
	g.syncOutcome()
	return g
}

// UTITKIY functions

func (g *Game) FEN() string        { return g.chess.FEN() }
func (g *Game) PGN() string        { return g.chess.String() }
func (g *Game) Turn() chess.Color  { return g.chess.Position().Turn() }
func (g *Game) Status() Status     { return g.status }
func (g *Game) Result() string     { return g.result }
func (g *Game) Reason() string     { return g.reason }
func (g *Game) Category() Category { return g.category }

func (g *Game) RemainingMillis(color chess.Color, now time.Time) int64 {
	return g.clock.RemainingAt(clockIndex(color), now).Milliseconds()
}

func clockIndex(c chess.Color) int {
	if c == chess.White {
		return white
	}
	return black
}
