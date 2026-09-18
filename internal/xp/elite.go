package xp

// Invention (elite skill) uses its own experience curve, different from the
// standard one: level 84 (not 92) is halfway to 99, and its virtual levels go
// to 150. Values are the official in-game cumulative-experience table
// published by the RuneScape wiki (Module:Experience/elitedataunr).
//
//	99 -> 36,073,511 XP   110 -> 56,412,678   120 -> 80,618,654

// MaxInventionLevel is Invention's highest virtual level.
const MaxInventionLevel = 150

// inventionTable[level] = total XP required to reach that Invention level.
var inventionTable [MaxInventionLevel + 2]int64

func init() {
	inventionTable = [MaxInventionLevel + 2]int64{
		1: 0, 2: 830, 3: 1861, 4: 2902, 5: 3980, 6: 5126, 7: 6380, 8: 7787,
		9: 9400, 10: 11275, 11: 13605, 12: 16372, 13: 19656, 14: 23546, 15: 28134,
		16: 33520, 17: 39809, 18: 47109, 19: 55535, 20: 65209, 21: 77190,
		22: 90811, 23: 106221, 24: 123573, 25: 143025, 26: 164742, 27: 188893,
		28: 215651, 29: 245196, 30: 277713, 31: 316311, 32: 358547, 33: 404634,
		34: 454796, 35: 509259, 36: 568254, 37: 632019, 38: 700797, 39: 774834,
		40: 854383, 41: 946227, 42: 1044569, 43: 1149696, 44: 1261903,
		45: 1381488, 46: 1508756, 47: 1644015, 48: 1787581, 49: 1939773,
		50: 2100917, 51: 2283490, 52: 2476369, 53: 2679917, 54: 2894505,
		55: 3120508, 56: 3358307, 57: 3608290, 58: 3870846, 59: 4146374,
		60: 4435275, 61: 4758122, 62: 5096111, 63: 5449685, 64: 5819299,
		65: 6205407, 66: 6608473, 67: 7028964, 68: 7467354, 69: 7924122,
		70: 8399751, 71: 8925664, 72: 9472665, 73: 10041285, 74: 10632061,
		75: 11245538, 76: 11882262, 77: 12542789, 78: 13227679, 79: 13937496,
		80: 14672812, 81: 15478994, 82: 16313404, 83: 17176661, 84: 18069395,
		85: 18992239, 86: 19945833, 87: 20930821, 88: 21947856, 89: 22997593,
		90: 24080695, 91: 25259906, 92: 26475754, 93: 27728955, 94: 29020233,
		95: 30350318, 96: 31719944, 97: 33129852, 98: 34580790, 99: 36073511,
		100: 37608773, 101: 39270442, 102: 40978509, 103: 42733789,
		104: 44537107, 105: 46389292, 106: 48291180, 107: 50243611,
		108: 52247435, 109: 54303504, 110: 56412678, 111: 58575824,
		112: 60793812, 113: 63067521, 114: 65397835, 115: 67785643,
		116: 70231841, 117: 72737330, 118: 75303019, 119: 77929820,
		120: 80618654, 121: 83370445, 122: 86186124, 123: 89066630,
		124: 92012904, 125: 95025896, 126: 98106559, 127: 101255855,
		128: 104474750, 129: 107764216, 130: 111125230, 131: 114558777,
		132: 118065845, 133: 121647430, 134: 125304532, 135: 129038159,
		136: 132849323, 137: 136739041, 138: 140708338, 139: 144758242,
		140: 148889790, 141: 153104021, 142: 157401983, 143: 161784728,
		144: 166253312, 145: 170808801, 146: 175452262, 147: 180184770,
		148: 185007406, 149: 189921255, 150: 194927409,
	}
}

// InventionTotalXP returns the total experience required to reach the given
// Invention (elite) level.
func InventionTotalXP(level int) int64 {
	if level <= 1 {
		return 0
	}
	if level > MaxInventionLevel+1 {
		level = MaxInventionLevel + 1
	}
	return inventionTable[level]
}

// InventionLevelFromXP returns the highest Invention level whose XP
// requirement is <= x (virtual level, capped at 150).
func InventionLevelFromXP(x int64) int {
	if x < 0 {
		x = 0
	}
	lo, hi := 1, MaxInventionLevel
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if inventionTable[mid] <= x {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// InventionXPToNext returns the XP needed to advance from Invention level L to
// L+1, or -1 at/above MaxInventionLevel.
func InventionXPToNext(level int) int64 {
	if level < 1 || level >= MaxInventionLevel {
		return -1
	}
	return inventionTable[level+1] - inventionTable[level]
}

// InventionProgress mirrors Progress on the elite curve.
func InventionProgress(level int, x int64) (into, needed int64) {
	needed = InventionXPToNext(level)
	if needed < 0 {
		return 0, 0
	}
	into = x - inventionTable[level]
	if into < 0 {
		into = 0
	}
	if into > needed {
		into = needed
	}
	return into, needed
}
