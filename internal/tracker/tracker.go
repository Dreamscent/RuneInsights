// Package tracker periodically polls the hiscores and stores snapshots.
package tracker

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"runeinsights/internal/db"
	"runeinsights/internal/hiscore"
)

// MinRefreshGap prevents hammering the upstream API after a manual refresh.
const MinRefreshGap = 5 * time.Minute

var (
	ErrTooSoon = errors.New("refreshed too recently")
	ErrBusy    = errors.New("a refresh is already in progress")
)

type Tracker struct {
	db     *db.DB
	client *hiscore.Client

	mu     sync.Mutex
	active map[int64]bool
}

func New(d *db.DB, c *hiscore.Client) *Tracker {
	return &Tracker{db: d, client: c, active: map[int64]bool{}}
}

// Run polls players whose individual interval has elapsed until ctx is done.
func (t *Tracker) Run(ctx context.Context) {
	t.Tick(ctx)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.Tick(ctx)
		}
	}
}

// Tick refreshes every player that is due. It returns immediately; upstream
// calls run in the background, staggered to respect the rate limit.
func (t *Tracker) Tick(ctx context.Context) {
	players, err := t.db.ListPlayers()
	if err != nil {
		log.Printf("tracker: list players: %v", err)
		return
	}
	now := time.Now()
	for _, p := range players {
		if !t.due(p, now) || !t.claim(p.ID) {
			continue
		}
		go func(id int64) {
			defer t.release(id)
			cctx, cancel := context.WithTimeout(ctx, 90*time.Second)
			defer cancel()
			if err := t.refresh(cctx, id, false); err != nil {
				log.Printf("tracker: background refresh of player %d failed: %v", id, err)
			}
		}(p.ID)
		time.Sleep(1500 * time.Millisecond)
	}
}

func (t *Tracker) due(p db.Player, now time.Time) bool {
	if p.LastFetchedAt == nil {
		return true
	}
	return now.Sub(*p.LastFetchedAt) >= time.Duration(p.IntervalMin)*time.Minute
}

func (t *Tracker) claim(id int64) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.active[id] {
		return false
	}
	t.active[id] = true
	return true
}

func (t *Tracker) release(id int64) {
	t.mu.Lock()
	delete(t.active, id)
	t.mu.Unlock()
}

// RefreshNow is a user-triggered refresh; it enforces MinRefreshGap.
func (t *Tracker) RefreshNow(ctx context.Context, id int64) error {
	if !t.claim(id) {
		return ErrBusy
	}
	defer t.release(id)
	return t.refresh(ctx, id, true)
}

func (t *Tracker) refresh(ctx context.Context, id int64, manual bool) error {
	p, err := t.db.GetPlayer(id)
	if err != nil {
		return err
	}
	if p.LastFetchedAt != nil && time.Since(*p.LastFetchedAt) < MinRefreshGap {
		if manual {
			return ErrTooSoon
		}
		return nil
	}

	pl, err := t.client.FetchPlayer(ctx, p.AccountType, p.Name)
	if err != nil {
		return err
	}

	skills := make([]db.SkillSnapshot, 0, len(pl.Skills))
	for _, s := range pl.Skills {
		skills = append(skills, db.SkillSnapshot{
			Key:   hiscore.Key(s.Name),
			Name:  s.Name,
			Level: s.Level,
			XP:    s.XP,
			Rank:  s.Rank,
		})
	}
	activities := make([]db.ActivitySnapshot, 0, len(pl.Activities))
	for _, a := range pl.Activities {
		activities = append(activities, db.ActivitySnapshot{
			Key:   hiscore.Key(a.Name),
			Name:  a.Name,
			Score: a.Score,
			Rank:  a.Rank,
		})
	}

	now := time.Now()
	if _, err := t.db.InsertSnapshot(id, now, skills, activities); err != nil {
		return err
	}
	return t.db.SetLastFetched(id, now)
}

// CapturePlayer fetches and stores a first snapshot for a freshly added player.
func (t *Tracker) CapturePlayer(ctx context.Context, p *db.Player, pl *hiscore.Player) error {
	skills := make([]db.SkillSnapshot, 0, len(pl.Skills))
	for _, s := range pl.Skills {
		skills = append(skills, db.SkillSnapshot{
			Key:   hiscore.Key(s.Name),
			Name:  s.Name,
			Level: s.Level,
			XP:    s.XP,
			Rank:  s.Rank,
		})
	}
	activities := make([]db.ActivitySnapshot, 0, len(pl.Activities))
	for _, a := range pl.Activities {
		activities = append(activities, db.ActivitySnapshot{
			Key:   hiscore.Key(a.Name),
			Name:  a.Name,
			Score: a.Score,
			Rank:  a.Rank,
		})
	}
	now := time.Now()
	if _, err := t.db.InsertSnapshot(p.ID, now, skills, activities); err != nil {
		return err
	}
	return t.db.SetLastFetched(p.ID, now)
}
