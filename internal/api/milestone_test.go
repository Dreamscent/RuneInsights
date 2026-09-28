package api

import (
	"testing"

	"runeinsights/internal/db"
	"runeinsights/internal/rates"
)

func TestMilestonePickAndETA(t *testing.T) {
	// Standard anchors: 99=13,034,431 · 110=38,737,661 · 120=104,273,167 · cap=200M.
	// Invention anchors: 99=36,073,511 · 110=56,412,678 · 120=80,618,654 · 150=194,927,409.
	cases := []struct {
		name      string
		key       string
		xp        int64
		wantLabel string
		wantThr   int64
		perHour   float64
		wantEta   *float64
	}{
		{"fresh", "attack", 0, "level 99", 13_034_431, 250_000, fptr(13_034_431. / 250_000)},
		{"just-under-99", "attack", 13_034_430, "level 99", 13_034_431, 250_000, fptr(1 / 250_000.)},
		{"at-99-picks-110", "attack", 13_034_431, "level 110", 38_737_661, 0, nil},
		{"between-110", "attack", 20_573_218, "level 110", 38_737_661, 250_000, fptr((38_737_661. - 20_573_218) / 250_000)},
		{"at-110-picks-120", "attack", 38_737_661, "level 120", 104_273_167, 0, nil},
		{"just-under-120", "attack", 104_273_166, "level 120", 104_273_167, 0, nil},
		{"at-120-picks-200m", "attack", 104_273_167, "200m xp", 200_000_000, 0, nil},
		{"at-cap-zero-remaining", "attack", 200_000_000, "200m xp", 200_000_000, 250_000, nil},
		{"above-cap-zero-remaining", "attack", 200_000_001, "200m xp", 200_000_000, 0, nil},
		{"invention-at-99-picks-110", "invention", 36_073_511, "level 110", 56_412_678, 0, nil},
		{"invention-under-120", "invention", 70_000_000, "level 120", 80_618_654, 0, nil},
		{"invention-at-150-cap", "invention", 194_927_409, "200m xp", 200_000_000, 0, nil},
		{"untrained-neg-xp", "attack", -1, "level 99", 13_034_431, 0, nil},
	}
	for _, c := range cases {
		sv := buildSkillView(
			db.SkillSnapshot{Key: c.key, Name: c.key, Level: 1, XP: c.xp},
			map[string]float64{},
			rates.Entry{PerHour: c.perHour},
		)
		if sv.Next.Label != c.wantLabel || sv.Next.XP != c.wantThr {
			t.Errorf("%s: got %q thr %d, want %q thr %d", c.name, sv.Next.Label, sv.Next.XP, c.wantLabel, c.wantThr)
		}
		rem := c.wantThr - c.xp
		if rem < 0 {
			rem = 0
		}
		if sv.Next.Remaining != rem {
			t.Errorf("%s: remaining %d, want %d", c.name, sv.Next.Remaining, rem)
		}
		if c.wantEta == nil {
			if sv.Next.EtaHours != nil {
				t.Errorf("%s: eta %v, want nil", c.name, *sv.Next.EtaHours)
			}
		} else {
			if sv.Next.EtaHours == nil || *sv.Next.EtaHours != *c.wantEta {
				t.Errorf("%s: eta %.6f, want %.6f", c.name, *sv.Next.EtaHours, *c.wantEta)
			}
		}
	}
}

// (needed-into)/perHour must be the next-level ETA.
func TestNextLevelETA(t *testing.T) {
	sv := buildSkillView(
		db.SkillSnapshot{Key: "attack", Name: "Attack", Level: 99, XP: 13_034_931}, // 500 into L99
		map[string]float64{},
		rates.Entry{PerHour: 250_000},
	)
	// span 99->100 = 1,356,729; into=500 -> eta=(1356729-500)/250000
	want := float64(1_356_729-500) / 250_000
	if sv.NextLevelEtaHours == nil || *sv.NextLevelEtaHours != want {
		t.Errorf("nextLevelEta = %v, want %v", sv.NextLevelEtaHours, want)
	}
	// Maxed: at 200M no further ETA either.
	maxed := buildSkillView(
		db.SkillSnapshot{Key: "attack", Name: "Attack", Level: 126, XP: 200_000_000},
		map[string]float64{},
		rates.Entry{PerHour: 250_000},
	)
	if maxed.NextLevelEtaHours != nil || !maxed.Maxed {
		t.Errorf("maxed skill should have no ETA and Maxed=true, got %v/%v", maxed.NextLevelEtaHours, maxed.Maxed)
	}
}

func fptr(f float64) *float64 { return &f }
