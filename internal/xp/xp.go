// Package xp implements RuneScape experience <-> level math.
//
// The total-experience formula is the one documented on runescape.wiki
// (identical for RS3 and OSRS):
//
//	xp(L) = floor( 1/4 * sum_{n=1}^{L-1} floor( n + 300 * 2^(n/7) ) )
package xp

import "math"

const (
	MaxXP     = int64(200_000_000) // per-skill XP cap
	MaxLevel  = 126                // highest virtual level; the 200M cap sits between 126 and 127
	InGameCap = 99                 // classic display cap (RS3 shows up to 120 via True Skill Mastery)
)

// table[level] = total XP required to reach that level. Index 0 is unused;
// the slice extends one past MaxLevel so Progress can peek at "next".
var table [MaxLevel + 2]int64

func init() {
	var sum int64
	table[1] = 0
	for n := 2; n <= MaxLevel+1; n++ {
		k := n - 1
		sum += int64(math.Floor(float64(k) + 300*math.Pow(2, float64(k)/7)))
		table[n] = sum / 4
	}
}

// TotalXP returns the total experience required to reach level L.
func TotalXP(level int) int64 {
	if level <= 1 {
		return 0
	}
	if level > MaxLevel+1 {
		level = MaxLevel + 1
	}
	return table[level]
}

// LevelFromXP returns the highest level whose total XP requirement is <= xp.
// This is the "virtual" level, capped at MaxLevel.
func LevelFromXP(x int64) int {
	if x < 0 {
		x = 0
	}
	lo, hi := 1, MaxLevel
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if table[mid] <= x {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// XPToNext returns the XP needed to advance from level L to L+1,
// or -1 when L is already the maximum virtual level.
func XPToNext(level int) int64 {
	if level < 1 || level >= MaxLevel {
		return -1
	}
	return table[level+1] - table[level]
}

// Progress returns how far xp is into level L: (xpIntoLevel, xpNeededForNext).
// Values are clamped so a mismatch between the reported level and the standard
// curve (e.g. elite skills) can never produce negative progress.
func Progress(level int, x int64) (into, needed int64) {
	needed = XPToNext(level)
	if needed < 0 {
		return 0, 0
	}
	into = x - table[level]
	if into < 0 {
		into = 0
	}
	if into > needed {
		into = needed
	}
	return into, needed
}

// CombatLevel implements the current RS3 formula (runescape.wiki/w/Combat_level):
//
//	floor( (1.3*max(Att+Str, 2*Mag, 2*Rng, 2*Necr) + Def + Const + floor(Pray/2) + floor(Summ/2)) / 4 )
func CombatLevel(att, str, def, con, pray, rng, mag, summ, necr int) int {
	tri := att + str
	if v := 2 * mag; v > tri {
		tri = v
	}
	if v := 2 * rng; v > tri {
		tri = v
	}
	if v := 2 * necr; v > tri {
		tri = v
	}
	base := float64(tri)*1.3 + float64(def+con) + float64(pray/2) + float64(summ/2)
	return int(base / 4)
}
