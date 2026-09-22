// Package api exposes the REST endpoints consumed by the dashboard.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"runeinsights/internal/db"
	"runeinsights/internal/hiscore"
	"runeinsights/internal/rates"
	"runeinsights/internal/stats"
	"runeinsights/internal/tracker"
	"runeinsights/internal/xp"
)

type API struct {
	db      *db.DB
	tracker *tracker.Tracker
	client  *hiscore.Client
	rates   *rates.Store
}

func New(d *db.DB, t *tracker.Tracker, c *hiscore.Client, rs *rates.Store) *API {
	return &API{db: d, tracker: t, client: c, rates: rs}
}

func (a *API) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/players", a.listPlayers)
	mux.HandleFunc("POST /api/players", a.createPlayer)
	mux.HandleFunc("GET /api/players/{id}", a.getPlayer)
	mux.HandleFunc("PATCH /api/players/{id}", a.updatePlayer)
	mux.HandleFunc("DELETE /api/players/{id}", a.deletePlayer)
	mux.HandleFunc("POST /api/players/{id}/refresh", a.refreshPlayer)
	mux.HandleFunc("GET /api/players/{id}/skills", a.playerSkills)
	mux.HandleFunc("GET /api/players/{id}/rates", a.playerRates)
	mux.HandleFunc("GET /api/players/{id}/history", a.playerHistory)
	mux.HandleFunc("GET /api/players/{id}/activities", a.playerActivities)
	mux.HandleFunc("PUT /api/players/{id}/focus", a.putFocus)
	mux.HandleFunc("GET /api/players/{id}/daily-gains", a.playerDailyGains)
	mux.HandleFunc("GET /api/leaderboard", a.leaderboard)
	mux.HandleFunc("GET /api/clans/{name}", a.clan)
	mux.HandleFunc("GET /api/skill-rates", a.getSkillRates)
	mux.HandleFunc("PUT /api/skill-rates/{skill}", a.putSkillRate)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusNotFound, "unknown API endpoint")
	})
	return mux
}

// ---------------------------------------------------------------------------
// helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (a *API) playerFromPath(w http.ResponseWriter, r *http.Request) (*db.Player, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid player id")
		return nil, false
	}
	p, err := a.db.GetPlayer(id)
	if errors.Is(err, db.ErrPlayerNotFound) {
		writeErr(w, http.StatusNotFound, "player not found")
		return nil, false
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	return p, true
}

// skillMaxLevel is the highest level each skill displays in-game
// (runescape.wiki/w/Skills). Skills without an entry cap at 99.
var skillMaxLevel = map[string]int{
	"attack": 120, "strength": 120, "ranged": 120, "magic": 120, "construction": 120,
	"dungeoneering": 120, "slayer": 120, "herblore": 120, "farming": 120,
	"archaeology": 120, "necromancy": 120, "thieving": 120, "invention": 120,
	"mining": 110, "smithing": 110, "woodcutting": 110, "fletching": 110,
	"firemaking": 110, "runecraft": 110, "crafting": 110, "hunter": 110,
}

var skillOrder = []string{
	"overall", "attack", "defence", "strength", "hitpoints", "ranged", "prayer", "magic",
	"cooking", "woodcutting", "fletching", "fishing", "firemaking", "crafting", "smithing",
	"mining", "herblore", "agility", "thieving", "slayer", "farming", "runecraft", "hunter",
	"construction", "summoning", "dungeoneering", "divination", "invention", "archaeology", "necromancy",
}

var skillOrderIndex = func() map[string]int {
	m := make(map[string]int, len(skillOrder))
	for i, k := range skillOrder {
		m[k] = i
	}
	return m
}()

func skillLess(a, b string) bool {
	ia, oka := skillOrderIndex[a]
	ib, okb := skillOrderIndex[b]
	if oka && okb {
		return ia < ib
	}
	if oka != okb {
		return oka
	}
	return a < b
}

// ---------------------------------------------------------------------------
// player CRUD

func (a *API) listPlayers(w http.ResponseWriter, r *http.Request) {
	players, err := a.db.ListPlayers()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, players)
}

func (a *API) createPlayer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		AccountType string `json:"accountType"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	switch body.AccountType {
	case "":
		body.AccountType = "normal"
	case "normal", "ironman", "hardcore":
	default:
		writeErr(w, http.StatusBadRequest, "account type must be normal, ironman or hardcore")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	pl, err := a.client.FetchPlayer(ctx, body.AccountType, name)
	if errors.Is(err, hiscore.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "player not found on the RS3 hiscores")
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}

	p, err := a.db.CreatePlayer(pl.Name, body.AccountType)
	if errors.Is(err, db.ErrPlayerExists) {
		writeErr(w, http.StatusConflict, "player is already being tracked")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.tracker.CapturePlayer(ctx, p, pl); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	p, _ = a.db.GetPlayer(p.ID)
	writeJSON(w, http.StatusCreated, p)
}

func (a *API) getPlayer(w http.ResponseWriter, r *http.Request) {
	p, ok := a.playerFromPath(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *API) updatePlayer(w http.ResponseWriter, r *http.Request) {
	p, ok := a.playerFromPath(w, r)
	if !ok {
		return
	}
	var body struct {
		AccountType *string `json:"accountType"`
		IntervalMin *int    `json:"intervalMin"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	accountType := p.AccountType
	if body.AccountType != nil {
		switch *body.AccountType {
		case "normal", "ironman", "hardcore":
			accountType = *body.AccountType
		default:
			writeErr(w, http.StatusBadRequest, "account type must be normal, ironman or hardcore")
			return
		}
	}
	interval := p.IntervalMin
	if body.IntervalMin != nil {
		interval = *body.IntervalMin
	}
	if err := a.db.UpdatePlayer(p.ID, accountType, interval); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	p, _ = a.db.GetPlayer(p.ID)
	writeJSON(w, http.StatusOK, p)
}

func (a *API) deletePlayer(w http.ResponseWriter, r *http.Request) {
	p, ok := a.playerFromPath(w, r)
	if !ok {
		return
	}
	if err := a.db.DeletePlayer(p.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) refreshPlayer(w http.ResponseWriter, r *http.Request) {
	p, ok := a.playerFromPath(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	err := a.tracker.RefreshNow(ctx, p.ID)
	switch {
	case errors.Is(err, tracker.ErrTooSoon):
		writeErr(w, http.StatusTooManyRequests, "this player was refreshed less than 5 minutes ago")
		return
	case errors.Is(err, tracker.ErrBusy):
		writeErr(w, http.StatusConflict, "a refresh is already running")
		return
	case errors.Is(err, hiscore.ErrNotFound):
		writeErr(w, http.StatusNotFound, "player no longer exists on the hiscores")
		return
	case err != nil:
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	p, _ = a.db.GetPlayer(p.ID)
	writeJSON(w, http.StatusOK, p)
}

// ---------------------------------------------------------------------------
// stats

// MilestoneView describes a skill's automatically computed next milestone
// (level 99, 110, 120 or the 200M XP cap).
type MilestoneView struct {
	Label     string   `json:"label"`
	XP        int64    `json:"xp"`
	Remaining int64    `json:"remaining"`
	Pct       float64  `json:"pct"`
	EtaHours  *float64 `json:"etaHours"`
}

// milestoneLabels mirror the in-game milestone tiers. Invention (elite) uses
// its own curve and additionally has level 150 as a virtual cap.
var milestoneLabels = []int{99, 110, 120}
var inventionMilestones = []int{99, 110, 120, 150}

type SkillView struct {
	Key          string  `json:"key"`
	Name         string  `json:"name"`
	Level        int     `json:"level"`
	VirtualLevel int     `json:"virtualLevel"`
	XP           int64   `json:"xp"`
	Rank         *int    `json:"rank"`
	XPIntoLevel  int64   `json:"xpIntoLevel"`
	XPToNext     int64   `json:"xpToNext"`
	Maxed        bool    `json:"maxed"`
	Elite        bool    `json:"elite"`
	RatePerDay   float64 `json:"ratePerDay"`

	// Training-rate estimates come from the human-editable skill_rates.csv.
	XpPerHour       float64  `json:"xpPerHour"`
	Method          string   `json:"method"`
	NextLevelEtaHours *float64 `json:"nextLevelEtaHours"`
	Next            *MilestoneView `json:"next"`
}

type SkillsResponse struct {
	Player          *db.Player `json:"player"`
	Overall         *SkillView `json:"overall"`
	CombatLevel     int        `json:"combatLevel"`
	Skills          []SkillView `json:"skills"`
	SnapshotAt      *time.Time `json:"snapshotAt"`
	SnapshotCount   int        `json:"snapshotCount"`
	CollectingSince *time.Time `json:"collectingSince"`
	RatesAvailable  bool       `json:"ratesAvailable"`
	MilestoneCounts MilestoneCounts `json:"milestoneCounts"`
	// TotalLevel is the hiscore overall level; MaxTotalLevel is the sum of
	// every skill's in-game maximum level (varies per skill: 99/110/120).
	MaxTotalLevel int      `json:"maxTotalLevel"`
	Focus         []string `json:"focus"`
}

// capXP returns the XP threshold at which a skill reaches its displayed cap.
func capXP(key string) int64 {
	lvl, ok := skillMaxLevel[key]
	if !ok {
		lvl = 99
	}
	if key == "invention" {
		return xp.InventionTotalXP(lvl)
	}
	return xp.TotalXP(lvl)
}
// (thresholds on each skill's own XP curve; Invention uses the elite curve).
type MilestoneCounts struct {
	SkillsAt99        int `json:"skillsAt99"`
	SkillsAt110       int `json:"skillsAt110"`
	SkillsAt120       int `json:"skillsAt120"`
	SkillsAt200m      int `json:"skillsAt200m"`
	SkillsAtLevelCap  int `json:"skillsAtLevelCap"`
	Total             int `json:"total"`
}

// SkillsAtLevelCap counts skills that have reached their own in-game maximum
// level (caps vary per skill: 99, 110 or 120).

func countMilestones(snap *db.Snapshot) MilestoneCounts {
	c := MilestoneCounts{}
	if snap == nil {
		return c
	}
	for key, sk := range snap.Skills {
		if key == "overall" {
			continue
		}
		c.Total++
		invention := key == "invention"
		at := func(lvl int) int64 {
			if invention {
				return xp.InventionTotalXP(lvl)
			}
			return xp.TotalXP(lvl)
		}
		if sk.XP >= at(99) {
			c.SkillsAt99++
		}
		if sk.XP >= at(110) {
			c.SkillsAt110++
		}
		if sk.XP >= at(120) {
			c.SkillsAt120++
		}
		if sk.XP >= capXP(key) {
			c.SkillsAtLevelCap++
		}
		if sk.XP >= xp.MaxXP {
			c.SkillsAt200m++
		}
	}
	return c
}

func buildSkillView(s db.SkillSnapshot, rates map[string]float64, entry rates.Entry) SkillView {
	// Every elite skill has its own XP curve; today that is Invention only.
	elite := s.Key == "invention"
	var vlevel int
	var into, needed int64
	if elite {
		vlevel = xp.InventionLevelFromXP(s.XP)
		into, needed = xp.InventionProgress(vlevel, s.XP)
	} else {
		vlevel = xp.LevelFromXP(s.XP)
		into, needed = xp.Progress(vlevel, s.XP)
	}
	maxed := needed <= 0 || s.XP >= xp.MaxXP

	v := SkillView{
		Key:          s.Key,
		Name:         s.Name,
		Level:        s.Level,
		VirtualLevel: vlevel,
		XP:           s.XP,
		XPIntoLevel:  into,
		XPToNext:     needed,
		Maxed:        maxed,
		Elite:        elite,
		RatePerDay:   rates[s.Key],
		XpPerHour:    entry.PerHour,
		Method:       entry.Method,
	}
	if s.Rank >= 0 {
		rank := s.Rank
		v.Rank = &rank
	}

	// ETA to the next level at the configured training rate.
	if entry.PerHour > 0 && needed > 0 {
		eta := float64(needed-into) / entry.PerHour
		v.NextLevelEtaHours = &eta
	}

	// Next milestone: first of 99 / 110 / 120 whose XP requirement exceeds the
	// current XP, otherwise the 200M cap. Uses the skill's own XP curve.
	milestoneXP := xp.MaxXP
	label := "200m xp"
	milestones := milestoneLabels
	if elite {
		milestones = inventionMilestones
	}
	for _, lvl := range milestones {
		need := xp.TotalXP(lvl)
		if elite {
			need = xp.InventionTotalXP(lvl)
		}
		if need > s.XP {
			milestoneXP = need
			label = fmt.Sprintf("level %d", lvl)
			break
		}
	}
	remaining := milestoneXP - s.XP
	if remaining < 0 {
		remaining = 0
	}
	pct := float64(s.XP) / float64(milestoneXP) * 100
	if pct > 100 {
		pct = 100
	}
	if pct < 0 {
		pct = 0
	}
	m := &MilestoneView{Label: label, XP: milestoneXP, Remaining: remaining, Pct: pct}
	if entry.PerHour > 0 && remaining > 0 {
		eta := float64(remaining) / entry.PerHour
		m.EtaHours = &eta
	}
	v.Next = m
	return v
}

func (a *API) playerSkills(w http.ResponseWriter, r *http.Request) {
	p, ok := a.playerFromPath(w, r)
	if !ok {
		return
	}
	snap, err := a.db.LatestSnapshot(p.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ETA estimates use the trailing 30-day pace; for newer players with less
	// history, fall back to whatever pace data exists (week, then day).
	rates, err := stats.RateMap(a.db, p.ID, stats.Month)
	if err != nil {
		rates = map[string]float64{}
	}
	if len(rates) == 0 {
		if weekRates, e := stats.RateMap(a.db, p.ID, stats.Week); e == nil {
			rates = weekRates
		}
	}
	if len(rates) == 0 {
		if dayRates, e := stats.RateMap(a.db, p.ID, stats.Day); e == nil {
			rates = dayRates
		}
	}
	count, _ := a.db.SnapshotCount(p.ID)
	since, _ := a.db.FirstSnapshotAt(p.ID)

	resp := SkillsResponse{
		Player:          p,
		Skills:          []SkillView{},
		SnapshotCount:   count,
		CollectingSince: since,
		RatesAvailable:  a.rates.Loaded(),
	}
	if focus, ferr := a.db.FocusList(p.ID); ferr == nil {
		resp.Focus = focus
	}
	if snap == nil {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	at := snap.FetchedAt
	resp.SnapshotAt = &at
	resp.MilestoneCounts = countMilestones(snap)

	keys := make([]string, 0, len(snap.Skills))
	for k := range snap.Skills {
		if k != "overall" {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return skillLess(keys[i], keys[j]) })
	for _, k := range keys {
		resp.Skills = append(resp.Skills, buildSkillView(snap.Skills[k], rates, a.rates.Get(k)))
	}
	if ov, ok := snap.Skills["overall"]; ok {
		v := buildSkillView(ov, rates, a.rates.Get("overall"))
		resp.Overall = &v
	}
	resp.CombatLevel = combatLevel(snap)
	if focus, ferr := a.db.FocusList(p.ID); ferr == nil {
		resp.Focus = focus
	}
	maxTotal := 0
	for key := range snap.Skills {
		if key == "overall" {
			continue
		}
		lvl, ok := skillMaxLevel[key]
		if !ok {
			lvl = 99
		}
		maxTotal += lvl
	}
	resp.MaxTotalLevel = maxTotal
	writeJSON(w, http.StatusOK, resp)
}

func combatLevel(snap *db.Snapshot) int {
	level := func(keys ...string) (int, bool) {
		for _, k := range keys {
			if s, ok := snap.Skills[k]; ok {
				return s.Level, true
			}
		}
		return 0, false
	}
	get := func(def int, keys ...string) int {
		if v, ok := level(keys...); ok {
			return v
		}
		return def
	}
	return xp.CombatLevel(
		get(1, "attack"),
		get(1, "strength"),
		get(1, "defence"),
		get(10, "hitpoints", "constitution"),
		get(1, "prayer"),
		get(1, "ranged"),
		get(1, "magic"),
		get(1, "summoning"),
		get(1, "necromancy"),
	)
}

func (a *API) playerRates(w http.ResponseWriter, r *http.Request) {
	p, ok := a.playerFromPath(w, r)
	if !ok {
		return
	}
	period, valid := stats.ParsePeriod(r.URL.Query().Get("period"))
	if !valid {
		writeErr(w, http.StatusBadRequest, "period must be day, week, month or year")
		return
	}
	rates, err := stats.ComputeRates(a.db, p.ID, period, time.Now())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rates)
}

func (a *API) playerHistory(w http.ResponseWriter, r *http.Request) {
	p, ok := a.playerFromPath(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	skill := strings.ToLower(strings.TrimSpace(q.Get("skill")))
	if skill == "" {
		skill = "overall"
	}
	now := time.Now()
	to := now
	if s := q.Get("to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			to = t
		}
	}
	from := to.Add(-30 * 24 * time.Hour)
	if s := q.Get("from"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			from = t
		}
	}
	points, err := a.db.HistoryXP(p.ID, skill, from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"skill":  skill,
		"from":   from,
		"to":     to,
		"points": points,
	})
}

type ActivityView struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Score int64  `json:"score"`
	Rank  *int   `json:"rank"`
}

func (a *API) playerActivities(w http.ResponseWriter, r *http.Request) {
	p, ok := a.playerFromPath(w, r)
	if !ok {
		return
	}
	snap, err := a.db.LatestSnapshot(p.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Optional period deltas for activities, same semantics as skill rates.
	var gains map[string]int64
	if qs := r.URL.Query().Get("period"); qs != "" {
		period, valid := stats.ParsePeriod(qs)
		if !valid {
			writeErr(w, http.StatusBadRequest, "period must be day, week, month or year")
			return
		}
		if snap != nil {
			if base, err := a.db.SnapshotBefore(p.ID, snap.FetchedAt.Add(-period.Duration())); err == nil && base != nil {
				gains = map[string]int64{}
				for k, cur := range snap.Activities {
					if prev, ok := base.Activities[k]; ok {
						d := cur.Score - prev.Score
						if d > 0 {
							gains[k] = d
						}
					}
				}
			}
		}
	}

	out := []ActivityView{}
	if snap != nil {
		keys := make([]string, 0, len(snap.Activities))
		for k := range snap.Activities {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			s := snap.Activities[k]
			v := ActivityView{Key: s.Key, Name: s.Name, Score: s.Score}
			if s.Rank >= 0 {
				rank := s.Rank
				v.Rank = &rank
			}
			out = append(out, v)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"player": p, "activities": out, "gains": gains})
}

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
// community data

func (a *API) leaderboard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	table := q.Get("table")
	if table == "" {
		table = "overall"
	}
	category := q.Get("category")
	if category == "" {
		category = "skills"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	rows, err := a.client.FetchLeaderboard(ctx, table, category)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"table":    table,
		"category": category,
		"rows":     rows,
	})
}

func (a *API) clan(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		writeErr(w, http.StatusBadRequest, "clan name is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	members, err := a.client.FetchClanMembers(ctx, name)
	if errors.Is(err, hiscore.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "clan not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"clan": name, "members": members})
}

// playerDailyGains exposes per-day XP deltas plus consistency summary
// (best day, active days, current streak) built from the retained snapshots.
func (a *API) playerDailyGains(w http.ResponseWriter, r *http.Request) {
	p, ok := a.playerFromPath(w, r)
	if !ok {
		return
	}
	data, err := stats.DailyGains(a.db, p.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"player": p, "consistency": data})
}

// putFocus replaces the player's ordered list of focused skills.
func (a *API) putFocus(w http.ResponseWriter, r *http.Request) {
	p, ok := a.playerFromPath(w, r)
	if !ok {
		return
	}
	var body struct {
		Skills []string `json:"skills"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	keys := make([]string, 0, len(body.Skills))
	seen := map[string]bool{}
	for _, raw := range body.Skills {
		k := strings.ToLower(strings.TrimSpace(raw))
		if _, known := skillOrderIndex[k]; k == "" || !known {
			continue // ignore unknown skills
		}
		if seen[k] {
			continue
		}
		seen[k] = true
		keys = append(keys, k)
	}
	if err := a.db.SetFocus(p.ID, keys); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	focus, _ := a.db.FocusList(p.ID)
	writeJSON(w, http.StatusOK, map[string]any{"player": p, "focus": focus})
}

// getSkillRate returns the grouped, human-editable training-rate data.
func (a *API) getSkillRates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ratesAvailable": a.rates.Loaded(),
		"skills":         a.rates.Rows(),
	})
}

// putSkillRate replaces all method rows of one skill in skill_rates.csv.
// Body: {"methods":[{"perHour":350000,"method":"Afk Croesus Front"}, ...]}
// The first entry becomes the default used in the skills page.
func (a *API) putSkillRate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Methods []rates.Entry `json:"methods"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	skill := strings.TrimSpace(r.PathValue("skill"))
	if skill == "" {
		writeErr(w, http.StatusBadRequest, "skill is required")
		return
	}
	cleaned := make([]rates.Entry, 0, len(body.Methods))
	for _, m := range body.Methods {
		if m.PerHour < 0 {
			m.PerHour = 0
		}
		m.Method = strings.TrimSpace(m.Method)
		cleaned = append(cleaned, m)
	}
	if len(cleaned) == 0 {
		cleaned = []rates.Entry{{}}
	}
	if err := a.rates.Upsert(skill, cleaned); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"skill":   strings.ToLower(skill),
		"methods": a.rates.Methods(strings.ToLower(skill)),
	})
}
