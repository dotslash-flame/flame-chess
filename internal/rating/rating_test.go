package rating

import (
	"math"
	"testing"
)

func almost(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestExpectedEqualRatings(t *testing.T) {
	if got := Expected(800, 800); !almost(got, 0.5) {
		t.Fatalf("Expected(800,800) = %v, want 0.5", got)
	}
}

func TestExpectedSymmetry(t *testing.T) {
	cases := [][2]int{{800, 1200}, {1500, 1000}, {900, 905}, {2000, 800}}
	for _, c := range cases {
		a := Expected(c[0], c[1])
		b := Expected(c[1], c[0])
		if !almost(a+b, 1.0) {
			t.Fatalf("Expected(%d,%d)+Expected(%d,%d) = %v, want 1.0", c[0], c[1], c[1], c[0], a+b)
		}
	}
}

func TestExpectedHigherRatedFavored(t *testing.T) {
	if got := Expected(1200, 800); got <= 0.5 {
		t.Fatalf("Expected(1200,800) = %v, want > 0.5", got)
	}
	if got := Expected(1200, 800); math.Abs(got-0.9090909) > 1e-6 {
		t.Fatalf("Expected(1200,800) = %v, want ~0.909", got)
	}
}

func TestKFactorBoundary(t *testing.T) {
	for _, n := range []int{0, 1, 29} {
		if got := K(n); got != 40 {
			t.Fatalf("K(%d) = %d, want 40", n, got)
		}
	}
	for _, n := range []int{30, 31, 100} {
		if got := K(n); got != 20 {
			t.Fatalf("K(%d) = %d, want 20", n, got)
		}
	}
}

func TestNewRatingEqualDrawNoChange(t *testing.T) {
	if got := NewRating(800, 800, 0.5, 0); got != 800 {
		t.Fatalf("NewRating equal-draw = %d, want 800", got)
	}
}

func TestNewRatingEqualWin(t *testing.T) {
	if got := NewRating(800, 800, 1, 0); got != 820 {
		t.Fatalf("NewRating equal-win = %d, want 820", got)
	}
	if got := NewRating(800, 800, 1, 30); got != 810 {
		t.Fatalf("NewRating equal-win 30g = %d, want 810", got)
	}
}

func TestNewRatingLossFloorsToward(t *testing.T) {
	if got := NewRating(800, 800, 0, 0); got != 780 {
		t.Fatalf("NewRating equal-loss = %d, want 780", got)
	}
}

func TestNewRatingRounds(t *testing.T) {
	if got := NewRating(800, 1200, 1, 0); got != 836 {
		t.Fatalf("NewRating underdog-win = %d, want 836", got)
	}
}

func TestNewRatingZeroSumApprox(t *testing.T) {
	win := NewRating(800, 800, 1, 0) - 800
	loss := NewRating(800, 800, 0, 0) - 800
	if win != -loss {
		t.Fatalf("win delta %d, loss delta %d; want mirrored", win, loss)
	}
}

func TestOutcome(t *testing.T) {
	cases := []struct {
		result       string
		white, black float64
		rated        bool
	}{
		{"1-0", 1, 0, true},
		{"0-1", 0, 1, true},
		{"1/2-1/2", 0.5, 0.5, true},
		{"", 0, 0, false},
		{"aborted", 0, 0, false},
		{"garbage", 0, 0, false},
	}
	for _, c := range cases {
		w, b, rated := Outcome(c.result)
		if rated != c.rated || !almost(w, c.white) || !almost(b, c.black) {
			t.Fatalf("Outcome(%q) = (%v,%v,%v), want (%v,%v,%v)", c.result, w, b, rated, c.white, c.black, c.rated)
		}
	}
}
