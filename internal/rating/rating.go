package rating

import "math"

func Expected(you, opp int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(opp-you)/400.0))
}

func K(gamesPlayed int) int {
	if gamesPlayed < 30 {
		return 40
	}
	return 20
}

func NewRating(old, opp int, score float64, gamesPlayed int) int {
	delta := float64(K(gamesPlayed)) * (score - Expected(old, opp))
	return int(math.Round(float64(old) + delta))
}

func Outcome(result string) (whiteScore, blackScore float64, rated bool) {
	switch result {
	case "1-0":
		return 1, 0, true
	case "0-1":
		return 0, 1, true
	case "1/2-1/2":
		return 0.5, 0.5, true
	default:
		return 0, 0, false
	}
}
