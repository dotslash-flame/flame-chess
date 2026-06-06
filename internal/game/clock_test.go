package game

import (
	"testing"
	"time"
)

func TestClockNewHasFullTimeBothSides(t *testing.T) {
	c := NewClock(5*time.Minute, 3*time.Second)
	if c.RemainingAt(white, time.Unix(0, 0)) != 5*time.Minute {
		t.Errorf("white remaining = %v, want 5m", c.RemainingAt(white, time.Unix(0, 0)))
	}
	if c.RemainingAt(black, time.Unix(0, 0)) != 5*time.Minute {
		t.Errorf("black remaining = %v, want 5m", c.RemainingAt(black, time.Unix(0, 0)))
	}
}

func TestClockRunningSideCountsDown(t *testing.T) {
	start := time.Unix(0, 0)
	c := NewClock(time.Minute, 0)
	c.Start(start)
	// 10s later, white (side to move) blakc is untouched, hopefully get 50s
	now := start.Add(10 * time.Second)
	if got := c.RemainingAt(white, now); got != 50*time.Second {
		t.Errorf("white remaining = %v, want 50s", got)
	}
	if got := c.RemainingAt(black, now); got != time.Minute {
		t.Errorf("black remaining = %v, want 60s", got)
	}
}

func TestClockPressDeductsAddsIncrementSwitchesTurn(t *testing.T) {
	start := time.Unix(0, 0)
	c := NewClock(time.Minute, 3*time.Second)
	c.Start(start)
	flagged := c.Press(start.Add(10 * time.Second))
	if flagged {
		t.Fatal("unexpected flag")
	}
	if got := c.RemainingAt(white, start.Add(10*time.Second)); got != 53*time.Second {
		t.Errorf("white remaining = %v, want 53s", got)
	}
	if c.Turn() != black {
		t.Errorf("turn = %d, want black(%d)", c.Turn(), black)
	}
}

func TestClockPressFlags(t *testing.T) {
	start := time.Unix(0, 0)
	c := NewClock(5*time.Second, 2*time.Second)
	c.Start(start)
	flagged := c.Press(start.Add(6 * time.Second))
	if !flagged {
		t.Fatal("expected flag")
	}
	if got := c.RemainingAt(white, start.Add(6*time.Second)); got != 0 {
		t.Errorf("white remaining = %v, want 0 (no increment on flag)", got)
	}
}

func TestClockFlaggedReportsCurrentSide(t *testing.T) {
	start := time.Unix(0, 0)
	c := NewClock(5*time.Second, 0)
	c.Start(start)
	if c.Flagged(start.Add(4 * time.Second)) {
		t.Error("should not be flagged at 4s")
	}
	if !c.Flagged(start.Add(5 * time.Second)) {
		t.Error("should be flagged at exactly 5s (<=0)")
	}
}

func TestClockRunning(t *testing.T) {
	start := time.Unix(0, 0)
	c := NewClock(5*time.Second, 0)
	if c.Running() {
		t.Error("a fresh clock should not be running until Start")
	}
	c.Start(start)
	if !c.Running() {
		t.Error("clock should be running after Start")
	}
	if !c.Press(start.Add(6 * time.Second)) {
		t.Fatal("expected flag")
	}
	if c.Running() {
		t.Error("clock should not be running after a flag")
	}
}
