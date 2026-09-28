package xp

import "testing"

// Anchor values from the official elite (Invention) cumulative XP table
// (runescape.wiki, Module:Experience/elitedataunr).
func TestInventionTotalXPAnchors(t *testing.T) {
	cases := map[int]int64{
		1: 0, 2: 830, 10: 11275, 99: 36073511, 110: 56412678,
		120: 80618654, 149: 189921255, 150: 194927409,
	}
	for level, want := range cases {
		if got := InventionTotalXP(level); got != want {
			t.Errorf("InventionTotalXP(%d) = %d, want %d", level, got, want)
		}
	}
}

func TestInventionLevelFromXP(t *testing.T) {
	cases := []struct {
		xp   int64
		want int
	}{
		{0, 1}, {829, 1}, {830, 2}, {36073510, 98}, {36073511, 99},
		{MaxXP, MaxInventionLevel}, {-5, 1},
	}
	for _, c := range cases {
		if got := InventionLevelFromXP(c.xp); got != c.want {
			t.Errorf("InventionLevelFromXP(%d) = %d, want %d", c.xp, got, c.want)
		}
	}
}

func TestInventionProgress(t *testing.T) {
	// Level 99 -> 100 span: 37608773-36073511 = 1535262.
	into, needed := InventionProgress(99, 36073511+500)
	if needed != 1535262 {
		t.Errorf("needed = %d, want 1535262", needed)
	}
	if into != 500 {
		t.Errorf("into = %d, want 500", into)
	}
	// Clamping: XP past the cap must never go negative/overflow.
	into, needed = InventionProgress(10, MaxXP)
	if into != needed {
		t.Errorf("clamped progress into=%d needed=%d", into, needed)
	}
	// At the virtual cap (150) there is no next level.
	if _, n := InventionProgress(MaxInventionLevel, MaxXP); n != 0 {
		t.Errorf("maxed elite level should report no next-level need, got %d", n)
	}
}
