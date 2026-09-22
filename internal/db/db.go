// Package db is the SQLite persistence layer.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrPlayerExists   = errors.New("player already tracked")
	ErrPlayerNotFound = errors.New("no such player")
)

// timeLayout is used for every timestamp column. All values are stored in UTC
// with a fixed layout so that lexicographic string comparison == chronological
// comparison, which lets us use plain "fetched_at <= ?" range filters.
const timeLayout = time.RFC3339

func fmtTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) time.Time {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

type DB struct {
	*sql.DB
}

// Open opens (creating if needed) the SQLite database at path.
func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// A single connection sidesteps SQLITE_BUSY entirely; the workload is tiny.
	sqlDB.SetMaxOpenConns(1)
	d := &DB{sqlDB}
	if err := d.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return d, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS players (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	name           TEXT NOT NULL UNIQUE COLLATE NOCASE,
	account_type   TEXT NOT NULL DEFAULT 'normal',
	interval_min   INTEGER NOT NULL DEFAULT 30,
	created_at     TEXT NOT NULL,
	last_fetched_at TEXT
);
CREATE TABLE IF NOT EXISTS snapshots (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	player_id  INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
	fetched_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_snapshots_player_time ON snapshots(player_id, fetched_at);
CREATE TABLE IF NOT EXISTS skill_snapshots (
	snapshot_id INTEGER NOT NULL REFERENCES snapshots(id) ON DELETE CASCADE,
	skill_key   TEXT NOT NULL,
	name        TEXT NOT NULL,
	level       INTEGER NOT NULL,
	xp          INTEGER NOT NULL,
	rank        INTEGER NOT NULL,
	PRIMARY KEY (snapshot_id, skill_key)
);
CREATE TABLE IF NOT EXISTS focus_skills (
	player_id  INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
	skill_key  TEXT NOT NULL,
	position   INTEGER NOT NULL,
	PRIMARY KEY (player_id, skill_key)
);
CREATE TABLE IF NOT EXISTS activity_snapshots (
	snapshot_id  INTEGER NOT NULL REFERENCES snapshots(id) ON DELETE CASCADE,
	activity_key TEXT NOT NULL,
	name         TEXT NOT NULL,
	score        INTEGER NOT NULL,
	rank         INTEGER NOT NULL,
	PRIMARY KEY (snapshot_id, activity_key)
);
`

func (d *DB) migrate() error {
	_, err := d.Exec(schema)
	return err
}

type Player struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	AccountType   string     `json:"accountType"`
	IntervalMin   int        `json:"intervalMin"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastFetchedAt *time.Time `json:"lastFetchedAt"`
}

type SkillSnapshot struct {
	Key   string
	Name  string
	Level int
	XP    int64
	Rank  int
}

type ActivitySnapshot struct {
	Key   string
	Name  string
	Score int64
	Rank  int
}

type Snapshot struct {
	ID         int64
	PlayerID   int64
	FetchedAt  time.Time
	Skills     map[string]SkillSnapshot
	Activities map[string]ActivitySnapshot
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (d *DB) CreatePlayer(name, accountType string) (*Player, error) {
	if accountType == "" {
		accountType = "normal"
	}
	res, err := d.Exec(
		`INSERT INTO players(name, account_type, interval_min, created_at) VALUES(?,?,?,?)`,
		name, accountType, 30, fmtTime(time.Now()))
	if err != nil {
		if isUnique(err) {
			return nil, ErrPlayerExists
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return d.GetPlayer(id)
}

func (d *DB) GetPlayer(id int64) (*Player, error) {
	row := d.QueryRow(
		`SELECT id, name, account_type, interval_min, created_at, last_fetched_at FROM players WHERE id=?`, id)
	return scanPlayer(row)
}

func scanPlayer(row *sql.Row) (*Player, error) {
	var p Player
	var created string
	var last sql.NullString
	if err := row.Scan(&p.ID, &p.Name, &p.AccountType, &p.IntervalMin, &created, &last); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlayerNotFound
		}
		return nil, err
	}
	p.CreatedAt = parseTime(created)
	if last.Valid {
		t := parseTime(last.String)
		p.LastFetchedAt = &t
	}
	return &p, nil
}

func (d *DB) ListPlayers() ([]Player, error) {
	rows, err := d.Query(
		`SELECT id, name, account_type, interval_min, created_at, last_fetched_at FROM players ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Player{}
	for rows.Next() {
		var p Player
		var created string
		var last sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &p.AccountType, &p.IntervalMin, &created, &last); err != nil {
			return nil, err
		}
		p.CreatedAt = parseTime(created)
		if last.Valid {
			t := parseTime(last.String)
			p.LastFetchedAt = &t
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (d *DB) UpdatePlayer(id int64, accountType string, intervalMin int) error {
	if intervalMin < 15 {
		intervalMin = 15
	}
	if intervalMin > 240 {
		intervalMin = 240
	}
	res, err := d.Exec(`UPDATE players SET account_type=?, interval_min=? WHERE id=?`, accountType, intervalMin, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrPlayerNotFound
	}
	return nil
}

func (d *DB) DeletePlayer(id int64) error {
	res, err := d.Exec(`DELETE FROM players WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrPlayerNotFound
	}
	return nil
}

func (d *DB) SetLastFetched(id int64, t time.Time) error {
	_, err := d.Exec(`UPDATE players SET last_fetched_at=? WHERE id=?`, fmtTime(t), id)
	return err
}

func (d *DB) InsertSnapshot(playerID int64, at time.Time, skills []SkillSnapshot, activities []ActivitySnapshot) (int64, error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO snapshots(player_id, fetched_at) VALUES(?,?)`, playerID, fmtTime(at))
	if err != nil {
		return 0, err
	}
	snapID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// DB-size housekeeping: keep only the latest snapshot per UTC day.
	if _, err := tx.Exec(
		`DELETE FROM snapshots WHERE player_id=? AND id!=? AND substr(fetched_at,1,10)=substr(?,1,10)`,
		playerID, snapID, fmtTime(at)); err != nil {
		return 0, err
	}

	sk, err := tx.Prepare(`INSERT INTO skill_snapshots(snapshot_id, skill_key, name, level, xp, rank) VALUES(?,?,?,?,?,?)`)
	if err != nil {
		return 0, err
	}
	defer sk.Close()
	for _, s := range skills {
		if _, err := sk.Exec(snapID, s.Key, s.Name, s.Level, s.XP, s.Rank); err != nil {
			return 0, err
		}
	}
	ak, err := tx.Prepare(`INSERT INTO activity_snapshots(snapshot_id, activity_key, name, score, rank) VALUES(?,?,?,?,?)`)
	if err != nil {
		return 0, err
	}
	defer ak.Close()
	for _, a := range activities {
		if _, err := ak.Exec(snapID, a.Key, a.Name, a.Score, a.Rank); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return snapID, nil
}

func (d *DB) loadSnapshot(id int64) (*Snapshot, error) {
	s := &Snapshot{Skills: map[string]SkillSnapshot{}, Activities: map[string]ActivitySnapshot{}}
	var playerID int64
	var at string
	if err := d.QueryRow(`SELECT player_id, fetched_at FROM snapshots WHERE id=?`, id).Scan(&playerID, &at); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	s.ID = id
	s.PlayerID = playerID
	s.FetchedAt = parseTime(at)

	rows, err := d.Query(`SELECT skill_key, name, level, xp, rank FROM skill_snapshots WHERE snapshot_id=?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k SkillSnapshot
		if err := rows.Scan(&k.Key, &k.Name, &k.Level, &k.XP, &k.Rank); err != nil {
			return nil, err
		}
		s.Skills[k.Key] = k
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	arows, err := d.Query(`SELECT activity_key, name, score, rank FROM activity_snapshots WHERE snapshot_id=?`, id)
	if err != nil {
		return nil, err
	}
	defer arows.Close()
	for arows.Next() {
		var a ActivitySnapshot
		if err := arows.Scan(&a.Key, &a.Name, &a.Score, &a.Rank); err != nil {
			return nil, err
		}
		s.Activities[a.Key] = a
	}
	return s, arows.Err()
}

// LatestSnapshot returns the most recent snapshot for a player, or nil if none.
func (d *DB) LatestSnapshot(playerID int64) (*Snapshot, error) {
	var id int64
	err := d.QueryRow(
		`SELECT id FROM snapshots WHERE player_id=? ORDER BY fetched_at DESC, id DESC LIMIT 1`, playerID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.loadSnapshot(id)
}

// SnapshotBefore returns the most recent snapshot taken at or before t, or nil.
func (d *DB) SnapshotBefore(playerID int64, t time.Time) (*Snapshot, error) {
	var id int64
	err := d.QueryRow(
		`SELECT id FROM snapshots WHERE player_id=? AND fetched_at<=? ORDER BY fetched_at DESC, id DESC LIMIT 1`,
		playerID, fmtTime(t)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.loadSnapshot(id)
}

// FirstSnapshot returns the earliest snapshot for a player, or nil.
func (d *DB) FirstSnapshot(playerID int64) (*Snapshot, error) {
	var id int64
	err := d.QueryRow(
		`SELECT id FROM snapshots WHERE player_id=? ORDER BY fetched_at ASC, id ASC LIMIT 1`, playerID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.loadSnapshot(id)
}

type HistoryPoint struct {
	At time.Time `json:"at"`
	XP int64     `json:"xp"`
}

// HistoryXP returns the XP progression of one skill between from and to.
func (d *DB) HistoryXP(playerID int64, skillKey string, from, to time.Time) ([]HistoryPoint, error) {
	rows, err := d.Query(`
		SELECT s.fetched_at, ss.xp
		FROM snapshots s
		JOIN skill_snapshots ss ON ss.snapshot_id = s.id
		WHERE s.player_id=? AND ss.skill_key=? AND s.fetched_at>=? AND s.fetched_at<=?
		ORDER BY s.fetched_at`, playerID, skillKey, fmtTime(from), fmtTime(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HistoryPoint{}
	for rows.Next() {
		var at string
		var x int64
		if err := rows.Scan(&at, &x); err != nil {
			return nil, err
		}
		out = append(out, HistoryPoint{At: parseTime(at), XP: x})
	}
	return out, rows.Err()
}

func (d *DB) SnapshotCount(playerID int64) (int, error) {
	var n int
	err := d.QueryRow(`SELECT COUNT(*) FROM snapshots WHERE player_id=?`, playerID).Scan(&n)
	return n, err
}

// FirstSnapshotAt returns the timestamp of the oldest snapshot, or nil.
func (d *DB) FirstSnapshotAt(playerID int64) (*time.Time, error) {
	var at string
	err := d.QueryRow(
		`SELECT fetched_at FROM snapshots WHERE player_id=? ORDER BY fetched_at ASC LIMIT 1`, playerID).Scan(&at)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t := parseTime(at)
	return &t, nil
}




// FocusList returns the player's focused skill keys in pin order.
func (d *DB) FocusList(playerID int64) ([]string, error) {
	rows, err := d.Query(
		`SELECT skill_key FROM focus_skills WHERE player_id=? ORDER BY position`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// SetFocus replaces the ordered list of focused skills for a player.
func (d *DB) SetFocus(playerID int64, keys []string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM focus_skills WHERE player_id=?`, playerID); err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO focus_skills(player_id, skill_key, position) VALUES(?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, k := range keys {
		if _, err := stmt.Exec(playerID, k, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}
