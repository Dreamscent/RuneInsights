package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTrimSnapshotsKeepsDailyHotAndWeeklyCold(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer d.Close()

	p, err := d.CreatePlayer("tester", "normal")
	if err != nil {
		t.Fatalf("create player: %v", err)
	}

	// Daily snapshots spanning 200 days, ending today at noon UTC.
	base := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC).AddDate(0, 0, -199)
	for i := 0; i < 200; i++ {
		at := base.AddDate(0, 0, i)
		if _, err := d.InsertSnapshot(p.ID, at,
			[]SkillSnapshot{{Key: "attack", Name: "Attack", Level: 1, XP: int64(i) * 100}},
			nil); err != nil {
			t.Fatalf("insert snapshot %d: %v", i, err)
		}
	}

	if err := d.TrimSnapshots(p.ID); err != nil {
		t.Fatalf("trim: %v", err)
	}

	cutoff := time.Now().UTC().Add(-hotRetention).Format("2006-01-02")

	rows, err := d.Query(`SELECT id, fetched_at FROM snapshots WHERE player_id=? ORDER BY id`, p.ID)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	var (
		kept     int
		hotDays  = map[string]bool{}
		coldWeek = map[int]bool{}
		cold     int
		lastCold = map[int]int64{}
	)
	for rows.Next() {
		var id int64
		var atStr string
		if err := rows.Scan(&id, &atStr); err != nil {
			t.Fatalf("scan: %v", err)
		}
		kept++
		at, err := time.Parse(timeLayout, atStr)
		if err != nil {
			t.Fatalf("parse %q: %v", atStr, err)
		}
		day := atStr[:10]
		if day >= cutoff {
			if hotDays[day] {
				t.Fatalf("duplicate hot day %q", day)
			}
			hotDays[day] = true
		} else {
			cold++
			year, w := at.ISOWeek()
			key := year*100 + w
			if coldWeek[key] {
				t.Fatalf("multiple cold snapshots in ISO week %d of %d", w, year)
			}
			coldWeek[key] = true
			if at.Format("2006-01-02") < "2026-03-15" {
				t.Fatalf("cold snapshot predates insert range: %q", atStr)
			}
			lastCold[key] = id
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	// Hot window: one snapshot per day for ~90 days (89–91 tolerant of tz/edge).
	if len(hotDays) < 89 || len(hotDays) > 91 {
		t.Fatalf("expected ~90 hot days, got %d", len(hotDays))
	}
	// Cold tail: ~111 days ≈ 16 weeks, one per week max.
	if cold == 0 || cold > 17 {
		t.Fatalf("expected 1..17 cold weekly snapshots, got %d", cold)
	}

	// Cold snapshots must not have cascaded their skill rows away.
	for _, id := range lastCold {
		var n int
		if err := d.QueryRow(`SELECT COUNT(*) FROM skill_snapshots WHERE snapshot_id=?`, id).Scan(&n); err != nil {
			t.Fatalf("cascade check: %v", err)
		}
		if n != 1 {
			t.Fatalf("cold snapshot %d has %d skill rows, want 1", id, n)
		}
	}
}
