package game

import "testing"

func TestCategoryForBaseSeconds(t *testing.T) {
	cases := []struct {
		base int
		want Category
	}{
		{60, CategoryBullet},  // 1+0 bullet
		{179, CategoryBullet}, // just under 3 min
		{180, CategoryBlitz},  // 3 min boundary = blitz
		{300, CategoryBlitz},  // 5+0 blitz
		{599, CategoryBlitz},  // just under 10 min
		{600, CategoryRapid},  // 10 min boundary = rapid
		{1800, CategoryRapid}, // 30 min classical = rapid bucket
	}
	for _, c := range cases {
		if got := CategoryForBaseSeconds(c.base); got != c.want {
			t.Errorf("CategoryForBaseSeconds(%d) = %q, want %q", c.base, got, c.want)
		}
	}
}
