// Package hiscore is a client for Jagex's official RS3 hiscores API.
//
//	GET https://secure.runescape.com/m=hiscore/index_lite.json?player=X
//
// The .json variant of index_lite returns:
//
//	{"name": "...", "skills": [{id,name,rank,level,xp}], "activities": [{id,name,rank,score}]}
//
// Unranked entries report -1. Untouched skills report xp -1 (normalized to 0 here).
// Unknown players yield an HTTP 404 or a body containing "error404".
package hiscore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://secure.runescape.com"

// ErrNotFound is returned when the player does not exist on the hiscore table.
var ErrNotFound = errors.New("player not found")

// UserAgent must look like a browser or Jagex rejects the request.
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"

type Skill struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Rank  int    `json:"rank"`
	Level int    `json:"level"`
	XP    int64  `json:"xp"`
}

type Activity struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Rank  int    `json:"rank"`
	Score int64  `json:"score"`
}

type Player struct {
	Name       string     `json:"name"`
	Skills     []Skill    `json:"skills"`
	Activities []Activity `json:"activities"`
}

// Key is the stable storage key for a skill/activity name.
func Key(name string) string { return strings.ToLower(name) }

// Account types and their hiscore module paths.
var modulePath = map[string]string{
	"normal":   "hiscore",
	"ironman":  "hiscore_ironman",
	"hardcore": "hiscore_hardcore_ironman",
}

func ModulePath(accountType string) string {
	if m, ok := modulePath[accountType]; ok {
		return m
	}
	return "hiscore"
}

// Client fetches hiscores with a conservative global rate limit (~1 req/s).
type Client struct {
	httpc *http.Client
	tok   chan struct{}
}

func NewClient() *Client {
	c := &Client{
		httpc: &http.Client{Timeout: 25 * time.Second},
		tok:   make(chan struct{}, 4),
	}
	go func() {
		for {
			time.Sleep(time.Second)
			select {
			case c.tok <- struct{}{}:
			default: // bucket full; drop the token
			}
		}
	}()
	return c
}

func (c *Client) wait(ctx context.Context) error {
	select {
	case <-c.tok:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// FetchPlayer retrieves the player's current hiscores for the given account type.
func (c *Client) FetchPlayer(ctx context.Context, accountType, name string) (*Player, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	u := baseURL + "/m=" + ModulePath(accountType) + "/index_lite.json?player=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hiscore fetch: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("hiscore read: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound || bytes.Contains(body, []byte("error404")) {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hiscore: unexpected status %d for %q", resp.StatusCode, name)
	}
	var p Player
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("hiscore: invalid JSON: %w", err)
	}
	for i := range p.Skills {
		if p.Skills[i].XP < 0 {
			p.Skills[i].XP = 0 // untouched skills report -1
		}
	}
	return &p, nil
}

// LeaderboardRow is one entry from the top-50 ranking endpoint.
type LeaderboardRow struct {
	Rank     int    `json:"rank"`
	Name     string `json:"name"`
	Score    int64  `json:"score"` // XP for skills, score for activities
	Level    int    `json:"level,omitempty"`
}

// FetchLeaderboard returns up to 50 top players for a table. The upstream
// semantics are numeric: table = skill/activity id (0 = overall) and
// category = 0 (skills) or 1 (activities). Values arrive as comma-formatted
// strings, e.g. {"name":"...","score":"200,000,000","rank":"1"}.
func (c *Client) FetchLeaderboard(ctx context.Context, table, category string) ([]LeaderboardRow, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/m=hiscore/ranking.json?table=%s&category=%s&size=50",
		baseURL, url.QueryEscape(table), url.QueryEscape(category))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("leaderboard fetch: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leaderboard: unexpected status %d", resp.StatusCode)
	}
	var raw []struct {
		Rank  string `json:"rank"`
		Name  string `json:"name"`
		Score string `json:"score"`
		Level string `json:"level"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("leaderboard: invalid JSON: %w", err)
	}
	rows := make([]LeaderboardRow, 0, len(raw))
	for _, r := range raw {
		row := LeaderboardRow{Name: r.Name}
		row.Rank, _ = strconv.Atoi(strings.ReplaceAll(r.Rank, ",", ""))
		s, _ := strconv.ParseInt(strings.ReplaceAll(r.Score, ",", ""), 10, 64)
		row.Score = s
		if r.Level != "" && r.Level != "-1" {
			row.Level, _ = strconv.Atoi(strings.ReplaceAll(r.Level, ",", ""))
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// ClanMember is one row of the clan members CSV
// (header: "Clanmate, Clan Rank, Total XP, Kills"; rows sorted by clan rank).
type ClanMember struct {
	Name     string `json:"name"`
	Rank     int    `json:"rank"` // 0=Owner, 1=General, ... 6=Recruit
	OverallXP int64 `json:"overallXp"`
	Kills    int64  `json:"kills"`
}

var clanRankOrdinal = map[string]int{
	"owner": 0, "deputy owner": 1, "general": 2, "captain": 3,
	"lieutenant": 4, "sergeant": 5, "corporal": 6, "recruit": 7,
}

// FetchClanMembers retrieves a clan's member list (CSV, sorted by clan rank).
func (c *Client) FetchClanMembers(ctx context.Context, clanName string) ([]ClanMember, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	u := baseURL + "/m=clan-hiscores/members_lite.ws?clanName=" + url.QueryEscape(clanName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("clan fetch: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound || bytes.Contains(body, []byte("error404")) || bytes.Equal(bytes.TrimSpace(body), []byte("")) {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clan: unexpected status %d", resp.StatusCode)
	}
	var out []ClanMember
	for _, line := range strings.Split(strings.ReplaceAll(string(body), "\r", ""), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Clanmate") {
			continue // header
		}
		f := strings.Split(line, ",")
		if len(f) < 3 {
			continue
		}
		m := ClanMember{Name: strings.TrimSpace(f[0])}
		m.Rank = clanRankOrdinal[strings.ToLower(strings.TrimSpace(f[1]))]
		xp, _ := strconv.ParseInt(strings.ReplaceAll(strings.TrimSpace(f[2]), ",", ""), 10, 64)
		m.OverallXP = xp
		if len(f) > 3 {
			k, _ := strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(f[3]), ",", ""))
			m.Kills = int64(k)
		}
		out = append(out, m)
	}
	return out, nil
}
