package game

import "time"

const (
	white = 0
	black = 1
)

// Fischer style chess clock.
// pass an explicit now the clock never reads the wall clock itself
type Clock struct {
	remaining   [2]time.Duration
	increment   time.Duration
	turn        int
	turnStarted time.Time
	running     bool
}

func NewClock(base, increment time.Duration) *Clock {
	return &Clock{
		remaining: [2]time.Duration{base, base},
		increment: increment,
		turn:      white,
	}
}

func (c *Clock) Start(now time.Time) {
	c.running = true
	c.turnStarted = now
}

func (c *Clock) Turn() int { return c.turn }

func (c *Clock) Running() bool { return c.running }

func (c *Clock) RemainingAt(color int, now time.Time) time.Duration {
	r := c.remaining[color]
	if c.running && color == c.turn {
		r -= now.Sub(c.turnStarted)
	}
	return r
}

func (c *Clock) Flagged(now time.Time) bool {
	return c.RemainingAt(c.turn, now) <= 0
}

func (c *Clock) Press(now time.Time) (flagged bool) {
	c.remaining[c.turn] -= now.Sub(c.turnStarted)
	if c.remaining[c.turn] <= 0 {
		c.remaining[c.turn] = 0
		c.running = false
		return true
	}
	c.remaining[c.turn] += c.increment
	c.turn = 1 - c.turn
	c.turnStarted = now
	return false
}
