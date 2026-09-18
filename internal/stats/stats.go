// Package stats derives gain/rate figures from stored snapshots.
package stats

import (
	"sort"
	"time"

	"runeinsights/internal/db"
)

type Period string

const (
	Day   Period = "day"
	Week  Period = "week"
	Month Period = "month"
	Year  Period = "year"
)

// ParsePeriod validates a period string, defaulting to Day.
func ParsePeriod(s string) (Period, bool) {
	switch Period(s) {
	case Day, Week, Month, Year:
		return Period(s), true
	case "":
		return Day, true
	default:
		return Day, false
	}
}

func (p Period) Duration() time.Duration {
	switch p {
	case Week:
		return 7 * 24 * time.Hour
	case Month:
		return 30 * 24 * time.Hour
	case Year:
		return 365 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

type SkillGain struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Gain        int64   `json:"gain"`
	RatePerDay  float64 `json:"ratePerDay"`
	HasBaseline bool    `json:"hasBaseline"`
	// TrackingGain is the XP gained since the very first snapshot. The UI
	// shows it as a fallback for periods whose baseline does not exist yet.
	TrackingGain  int64 `json:"trackingGain"`
	Rank          *int `json:"rank"`         // current hiscore rank (nil when unranked)
	RankChange    *int `json:"rankChange"` // latest vs baseline; negative = climbed
}

type Rates struct {
	Period       Period      `json:"period"`
	From         *time.Time  `json:"from"`
	To           *time.Time  `json:"to"`
	ElapsedHours float64     `json:"elapsedHours"`
	HasBaseline  bool        `json:"hasBaseline"`
	Overall      SkillGain   `json:"overall"`
	Skills       []SkillGain `json:"skills"`

	// Tracking totals: XP gained since the very first snapshot, independent of
	// the period. The UI shows these as a fallback when a period has no
	// baseline yet, with TrackingDays describing how long data exists for.
	TrackingGain int64     `json:"trackingGain"`
	TrackingSince *time.Time `json:"trackingSince"`
	TrackingDays  float64   `json:"trackingDays"`
}

// ComputeRates compares the latest snapshot against the latest snapshot taken
// at or before (now - period). The rate uses the real elapsed time between the
// two snapshots rather than the nominal period length, so a period is never
// overstated when snapshots are sparse.
func ComputeRates(d *db.DB, playerID int64, p Period, now time.Time) (*Rates, error) {
	out := &Rates{Period: p, Skills: []SkillGain{}}

	latest, err := d.LatestSnapshot(playerID)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return out, nil
	}
	to := latest.FetchedAt
	out.To = &to

	// tracking totals (earliest -> latest)
	if first, err := d.FirstSnapshot(playerID); err == nil && first != nil && first.ID != latest.ID {
		since := first.FetchedAt
		out.TrackingSince = &since
		out.TrackingDays = to.Sub(since).Hours() / 24
		if cur, ok := latest.Skills["overall"]; ok {
			if fv, ok2 := first.Skills["overall"]; ok2 {
				g := cur.XP - fv.XP
				if g > 0 {
					out.TrackingGain = g
				}
			}
		}
	}

	base, err := d.SnapshotBefore(playerID, to.Add(-p.Duration()))
	if err != nil {
		return nil, err
	}

	// Emit a row for every skill in the latest snapshot so the UI can show a
	// stable table even before a baseline exists.
	keys := make([]string, 0, len(latest.Skills))
	for k := range latest.Skills {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if base == nil {
		first, ferr := d.FirstSnapshot(playerID)
		for _, k := range keys {
			s := latest.Skills[k]
			g := SkillGain{Key: k, Name: s.Name, HasBaseline: false}
			if s.Rank >= 0 {
				rk := s.Rank
				g.Rank = &rk
			}
			if ferr == nil && first != nil && first.ID != latest.ID {
				if fv, ok := first.Skills[k]; ok {
					t := s.XP - fv.XP
					if t > 0 {
						g.TrackingGain = t
					}
				}
			}
			out.Skills = append(out.Skills, g)
		}
		ov := latest.Skills["overall"]
		out.Overall = SkillGain{Key: "overall", Name: "Overall", HasBaseline: false}
		_ = ov
		return out, nil
	}

	elapsed := to.Sub(base.FetchedAt)
	hours := elapsed.Hours()
	out.HasBaseline = true
	out.ElapsedHours = hours
	out.From = &base.FetchedAt

	rate := func(gain int64) float64 {
		if hours <= 0 {
			return 0
		}
		return float64(gain) / hours * 24
	}
	delta := func(cur, prev db.SkillSnapshot) int64 {
		d := cur.XP - prev.XP
		if d < 0 {
			d = 0 // upstream corrections / rank noise should never show negative gain
		}
		return d
	}

	for _, k := range keys {
		cur := latest.Skills[k]
		g := SkillGain{Key: k, Name: cur.Name, HasBaseline: false}
		if cur.Rank >= 0 {
			rk := cur.Rank
			g.Rank = &rk
		}
		if prev, ok := base.Skills[k]; ok {
			g.Gain = delta(cur, prev)
			g.RatePerDay = rate(g.Gain)
			g.HasBaseline = true
			if cur.Rank >= 0 && prev.Rank >= 0 {
				d := cur.Rank - prev.Rank
				g.RankChange = &d
			}
		}
		out.Skills = append(out.Skills, g)
	}

	out.Overall = SkillGain{Key: "overall", Name: "Overall", HasBaseline: false}
	if cur, ok := latest.Skills["overall"]; ok {
		if cur.Rank >= 0 {
			rk := cur.Rank
			out.Overall.Rank = &rk
		}
		if prev, ok2 := base.Skills["overall"]; ok2 {
			out.Overall.Gain = delta(cur, prev)
			out.Overall.RatePerDay = rate(out.Overall.Gain)
			out.Overall.HasBaseline = true
			if cur.Rank >= 0 && prev.Rank >= 0 {
				d := cur.Rank - prev.Rank
				out.Overall.RankChange = &d
			}
		}
	}
	return out, nil
}

// RateMap returns per-skill rates (XP/day) keyed by skill, for target ETA estimates.
func RateMap(d *db.DB, playerID int64, p Period) (map[string]float64, error) {
	r, err := ComputeRates(d, playerID, p, time.Now())
	if err != nil {
		return nil, err
	}
	m := map[string]float64{}
	for _, s := range r.Skills {
		if s.HasBaseline {
			m[s.Key] = s.RatePerDay
		}
	}
	if r.Overall.HasBaseline {
		m["overall"] = r.Overall.RatePerDay
	}
	return m, nil
}

// -- consistency: per-day gains from the retained one-per-day snapshots ------

type DailyGainPoint struct {
	Date string `json:"date"` // YYYY-MM-DD (UTC)
	XP   int64  `json:"xp"`
	Gain int64  `json:"gain"` // delta vs the previous retained day (-1 for the first point)
}

type Consistency struct {
	Days        []DailyGainPoint `json:"days"`
	TotalGained int64            `json:"totalGained"`
	ActiveDays  int              `json:"activeDays"`       // days with Gain > 0
	BestDate    string           `json:"bestDate"`
	BestGain    int64            `json:"bestGain"`
	Streak      int              `json:"streak"` // consecutive active days ending at the latest active day
}

// DailyGains builds one point per retained snapshot day (latest per day),
// with the XP delta between consecutive days.
func DailyGains(d *db.DB, playerID int64) (*Consistency, error) {
	rows, err := d.DB.Query(`
		SELECT substr(s.fetched_at,1,10), ss.xp
		FROM snapshots s
		JOIN skill_snapshots ss ON ss.snapshot_id = s.id AND ss.skill_key='overall'
		WHERE s.player_id=?
		ORDER BY s.fetched_at`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type point struct {
		xp   int64
		date string
	}
	perDay := map[string]int64{}
	dates := []string{}
	for rows.Next() {
		var date string
		var xp int64
		if err := rows.Scan(&date, &xp); err != nil {
			return nil, err
		}
		// rows are ascending; keep the last value seen per day (latest snapshot)
		if _, seen := perDay[date]; !seen {
			dates = append(dates, date)
		}
		perDay[date] = xp
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := &Consistency{Days: []DailyGainPoint{}}
	var prev int64
	for i, date := range dates {
		xp := perDay[date]
		gain := int64(0)
		if i > 0 {
			gain = xp - prev
			if gain < 0 {
				gain = 0
			}
		}
		if i == 0 {
			out.Days = append(out.Days, DailyGainPoint{Date: date, XP: xp, Gain: -1})
		} else {
			out.Days = append(out.Days, DailyGainPoint{Date: date, XP: xp, Gain: gain})
			if gain >= 0 {
				out.TotalGained += gain
			}
			if gain > 0 {
				out.ActiveDays++
				if gain > out.BestGain {
					out.BestGain = gain
					out.BestDate = date
				}
			}
		}
		prev = xp
	}

	// current streak: consecutive active calendar days, walking backwards
	streak := 0
	var lastCounted time.Time
	for i := len(out.Days) - 1; i >= 0; i-- {
		pt := out.Days[i]
		if pt.Gain < 0 {
			break // first point, no delta
		}
		day, err := time.ParseInLocation("2006-01-02", pt.Date, time.UTC)
		if err != nil {
			break
		}
		if pt.Gain > 0 {
			if streak > 0 {
				// must be yesterday-continuous
				if !day.Add(24*time.Hour).Equal(lastCounted) {
					break
				}
			}
			lastCounted = day
			streak++
			continue
		}
		// a zero day: skip only the trailing day (still in progress)
		if streak == 0 && i == len(out.Days)-1 {
			continue
		}
		break
	}
	out.Streak = streak
	_ = prev
	return out, nil
}

func prevOfOut(c *Consistency, i int) int64 {
	if i <= 0 {
		return 0
	}
	return c.Days[i-1].XP
}
