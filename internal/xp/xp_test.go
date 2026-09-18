package xp

import "testing"

// Anchor values from runescape.wiki/w/Experience and oldschool.runescape.wiki/w/Experience
// (both games share the identical table).
func TestTotalXPAnchors(t *testing.T) {
	cases := map[int]int64{
		1: 0, 2: 83, 3: 174, 5: 388, 10: 1154, 20: 4470, 30: 13363,
		50: 101333, 70: 737627, 92: 6517253, 99: 13034431, 100: 14391160,
		110: 38737661, 120: 104273167, 126: 188884740,
	}
	for level, want := range cases {
		if got := TotalXP(level); got != want {
			t.Errorf("TotalXP(%d) = %d, want %d", level, got, want)
		}
	}
}

func TestLevelFromXP(t *testing.T) {
	cases := []struct {
		xp   int64
		want int
	}{
		{0, 1}, {82, 1}, {83, 2}, {13034430, 98}, {13034431, 99},
		{13034650, 99}, {MaxXP, MaxLevel}, {-5, 1},
	}
	for _, c := range cases {
		if got := LevelFromXP(c.xp); got != c.want {
			t.Errorf("LevelFromXP(%d) = %d, want %d", c.xp, got, c.want)
		}
	}
}

func TestProgress(t *testing.T) {
	into, needed := Progress(99, 13034431+500)
	if needed != 1356729 { // TotalXP(100)-TotalXP(99) = 14391160-13034431
		t.Errorf("needed = %d, want 1356729", needed)
	}
	if into != 500 {
		t.Errorf("into = %d, want 500", into)
	}

	// Clamping: XP beyond the level's ceiling must not go negative or overflow.
	into, needed = Progress(10, 999999999)
	if into != needed {
		t.Errorf("clamped progress into=%d needed=%d", into, needed)
	}

	if _, n := Progress(MaxLevel, MaxXP); n != 0 {
		t.Errorf("maxed level should report no next-level need, got %d", n)
	}
}

func TestCombatLevel(t *testing.T) {
	// Fresh player: all levels 1 (Constitution starts at 10) -> combat 3.
	if got := CombatLevel(1, 1, 1, 10, 1, 1, 1, 1, 1); got != 3 {
		t.Errorf("fresh player combat = %d, want 3", got)
	}
	// Maxed: 120 Att/Str/Mag/Rng/Necr, 99 Def/Con/Pray/Summ -> combat 152.
	if got := CombatLevel(120, 120, 99, 99, 99, 120, 120, 99, 120); got != 152 {
		t.Errorf("maxed combat = %d, want 152", got)
	}
}
