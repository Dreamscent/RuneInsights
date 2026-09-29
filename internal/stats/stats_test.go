package stats

import (
	"path/filepath"
	"testing"
	"time"

	"runeinsights/internal/db"
)

// seedSnapshot builds a skill set where every filler skill sits at the level-99
// cap XP, attack levels 98 -> 99, and dungeoneering stays at its in-game cap of
// 120 (its virtual level crosses 119 -> 120, but the capped display level
// Jagex reports never changes).
func seedSnapshot(at time.Time) ([]db.SkillSnapshot, []db.ActivitySnapshot) {
	const level99XP = 13_034_431 // XP required for level 99
	none := []db.ActivitySnapshot{}

	skills := []db.SkillSnapshot{
		{Key: "attack", Name: "Attack", Level: 98, XP: 11_805_606, Rank: 0},
		{Key: "dungeoneering", Name: "Dungeoneering", Level: 120, XP: 104_000_000, Rank: 0},
	}
	overallXP := int64(11_805_606 + 104_000_000)
	for i := 0; i < 27; i++ {
		skills = append(skills, db.SkillSnapshot{
			Key: "filler" + string(rune('a'+i)), Name: "Filler",
			Level: 99, XP: level99XP, Rank: 0,
		})
		overallXP += level99XP
	}
	skills = append(skills, db.SkillSnapshot{
		Key: "overall", Name: "Overall",
		Level: 98 + 120 + 27*99, XP: overallXP, Rank: 0,
	})
	return skills, none
}

// TestOverallLevelsGainedUsesCappedTotalLevel verifies that Overall's
// LevelsGained equals the change of the stored (capped) total level, and in
// particular does NOT count virtual levels past the in-game cap — while
// per-skill gains keep their virtual semantics.
func TestOverallLevelsGainedUsesCappedTotalLevel(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	p, err := d.CreatePlayer("Test", "main")
	if err != nil {
		t.Fatalf("create player: %v", err)
	}

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	sA, actA := seedSnapshot(base)
	if _, err := d.InsertSnapshot(p.ID, base, sA, actA); err != nil {
		t.Fatalf("insert snapshot A: %v", err)
	}

	later := base.Add(7 * 24 * time.Hour)
	sB, actB := seedSnapshot(later)
	// Snapshot B: attack hit the level-99 cap XP; dungeoneering gained XP
	// within its capped level 120 (virtual 119 -> 120, capped level stays 120).
	for i := range sB {
		switch sB[i].Key {
		case "attack":
			sB[i].Level, sB[i].XP = 99, 13_034_431
		case "dungeoneering":
			sB[i].XP = 115_000_000 // virtual 120, capped level unchanged
		case "overall":
			sB[i].Level++
			sB[i].XP += (13_034_431 - 11_805_606) + 11_000_000
		}
	}
	if _, err := d.InsertSnapshot(p.ID, later, sB, actB); err != nil {
		t.Fatalf("insert snapshot B: %v", err)
	}

	r, err := ComputeRates(d, p.ID, Week, later.Add(time.Hour))
	if err != nil {
		t.Fatalf("ComputeRates: %v", err)
	}

	if got := r.Overall.LevelsGained; got != 1 {
		t.Errorf("Overall.LevelsGained = %d, want 1 (only attack leveled within caps)", got)
	}

	byKey := map[string]SkillGain{}
	for _, s := range r.Skills {
		byKey[s.Key] = s
	}
	if atk := byKey["attack"]; atk.LevelsGained != 1 {
		t.Errorf("attack LevelsGained = %d, want 1", atk.LevelsGained)
	}
	// Dungeoneering gained 11M XP inside its capped level, so the virtual level
	// diff is 1 — per-skill badges keep virtual semantics — but it must not
	// inflate the total level.
	if dung := byKey["dungeoneering"]; dung.LevelsGained != 1 {
		t.Errorf("dungeoneering LevelsGained = %d, want 1 (virtual semantics kept)", dung.LevelsGained)
	}
	if dung := byKey["dungeoneering"]; dung.Gain != 11_000_000 {
		t.Errorf("dungeoneering XP gain = %d, want 11000000", dung.Gain)
	}
}
