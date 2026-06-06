package game

import (
	"testing"
	"time"

	"github.com/notnil/chess"
)

func newTestGame() *Game {
	return NewGame(300, 3, time.Unix(0, 0))
}

func TestNewGameStartsActiveWhiteToMove(t *testing.T) {
	g := newTestGame()
	if g.Status() != StatusActive {
		t.Errorf("status = %v, want active", g.Status())
	}
	if g.Turn() != chess.White {
		t.Errorf("turn = %v, want white", g.Turn())
	}
	if g.Category() != CategoryBlitz {
		t.Errorf("category = %v, want blitz", g.Category())
	}
}

func TestLegalMoveAdvancesTurn(t *testing.T) {
	g := newTestGame()
	if err := g.Move("e2e4", time.Unix(1, 0)); err != nil {
		t.Fatalf("Move e2e4: %v", err)
	}
	if g.Turn() != chess.Black {
		t.Errorf("turn = %v, want black", g.Turn())
	}
}

func TestIllegalMoveRejected(t *testing.T) {
	g := newTestGame()
	if err := g.Move("e2e5", time.Unix(1, 0)); err == nil {
		t.Fatal("expected error for illegal move e2e5")
	}
	if g.Turn() != chess.White || g.Status() != StatusActive {
		t.Errorf("state changed after illegal move: turn=%v status=%v", g.Turn(), g.Status())
	}
}

func TestMoveAfterGameOverRejected(t *testing.T) {
	g := newTestGame()
	_ = g.Resign(chess.White)
	if err := g.Move("e2e4", time.Unix(1, 0)); err == nil {
		t.Fatal("expected ErrNotActive after game finished")
	}
}

func TestCheckmateFoolsMate(t *testing.T) {
	g := newTestGame()
	moves := []string{"f2f3", "e7e5", "g2g4", "d8h4"}
	for i, mv := range moves {
		if err := g.Move(mv, time.Unix(int64(i+1), 0)); err != nil {
			t.Fatalf("move %s: %v", mv, err)
		}
	}
	if g.Status() != StatusFinished {
		t.Fatalf("status = %v, want finished", g.Status())
	}
	if g.Result() != "0-1" {
		t.Errorf("result = %q, want 0-1", g.Result())
	}
	if g.Reason() != "checkmate" {
		t.Errorf("reason = %q, want checkmate", g.Reason())
	}
}

func TestResignWhiteLoses(t *testing.T) {
	g := newTestGame()
	if err := g.Resign(chess.White); err != nil {
		t.Fatalf("resign: %v", err)
	}
	if g.Result() != "0-1" || g.Reason() != "resign" {
		t.Errorf("result/reason = %q/%q, want 0-1/resign", g.Result(), g.Reason())
	}
}

func TestAgreeDraw(t *testing.T) {
	g := newTestGame()
	if err := g.AgreeDraw(); err != nil {
		t.Fatalf("agree draw: %v", err)
	}
	if g.Result() != "1/2-1/2" || g.Reason() != "draw_agreed" {
		t.Errorf("result/reason = %q/%q, want 1/2-1/2/draw_agreed", g.Result(), g.Reason())
	}
}

func TestStalemateFromPosition(t *testing.T) {
	fen, err := chess.FEN("7k/8/6K1/8/8/8/5Q2/8 w - - 0 1")
	if err != nil {
		t.Fatalf("fen: %v", err)
	}
	g := newGameFromChess(chess.NewGame(fen))
	if err := g.Move("f2f7", time.Unix(1, 0)); err != nil {
		t.Fatalf("Qf7: %v", err)
	}
	if g.Result() != "1/2-1/2" || g.Reason() != "stalemate" {
		t.Errorf("result/reason = %q/%q, want 1/2-1/2/stalemate", g.Result(), g.Reason())
	}
}

func TestInsufficientMaterialAfterCapture(t *testing.T) {
	fen, err := chess.FEN("4k3/8/8/8/8/8/4r3/4K3 w - - 0 1")
	if err != nil {
		t.Fatalf("fen: %v", err)
	}
	g := newGameFromChess(chess.NewGame(fen))
	if err := g.Move("e1e2", time.Unix(1, 0)); err != nil {
		t.Fatalf("Kxe2: %v", err)
	}
	if g.Result() != "1/2-1/2" || g.Reason() != "insufficient" {
		t.Errorf("result/reason = %q/%q, want 1/2-1/2/insufficient", g.Result(), g.Reason())
	}
}

func TestClaimThreefoldRepetition(t *testing.T) {
	g := newTestGame()
	moves := []string{
		"g1f3", "g8f6", "f3g1", "f6g8",
		"g1f3", "g8f6", "f3g1", "f6g8",
	}
	for i, mv := range moves {
		if err := g.Move(mv, time.Unix(int64(i+1), 0)); err != nil {
			t.Fatalf("move %s: %v", mv, err)
		}
	}
	if err := g.ClaimDraw(); err != nil {
		t.Fatalf("claim threefold: %v", err)
	}
	if g.Result() != "1/2-1/2" || g.Reason() != "threefold" {
		t.Errorf("result/reason = %q/%q, want 1/2-1/2/threefold", g.Result(), g.Reason())
	}
}

func TestTimeoutOnMoveAfterFlag(t *testing.T) {
	g := NewGame(5, 0, time.Unix(0, 0))
	if err := g.Move("e2e4", time.Unix(6, 0)); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if g.Status() != StatusFinished || g.Result() != "0-1" || g.Reason() != "timeout" {
		t.Errorf("got status=%v result=%q reason=%q, want finished/0-1/timeout",
			g.Status(), g.Result(), g.Reason())
	}
}

func TestTimeoutCheck(t *testing.T) {
	g := NewGame(5, 0, time.Unix(0, 0))
	if g.TimeoutCheck(time.Unix(4, 0)) {
		t.Error("should not time out at 4s")
	}
	if !g.TimeoutCheck(time.Unix(5, 0)) {
		t.Error("should time out at 5s")
	}
	if g.Result() != "0-1" || g.Reason() != "timeout" {
		t.Errorf("result/reason = %q/%q, want 0-1/timeout", g.Result(), g.Reason())
	}
}
