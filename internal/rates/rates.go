// Package rates loads the human-editable skill_rates.csv that maps skills to
// training methods and expected experience per hour. A skill may have several
// method rows; the FIRST row for a skill is the default used for estimates on
// the skills page. Estimates are only shown for entries with a positive
// xp/hour value; unknown skills or zeros are reported as "no estimate".
//
// File format (encoding/csv, so quote fields that contain commas):
//
//	skill,exp_per_hour,training_method
//	Hunter,350000,Afk Croesus Front
//	Hunter,1200000,Chinning with juju on tightrope
//
// Blank lines and lines starting with '#' are ignored. Skill names are matched
// case-insensitively against the hiscore names (e.g. "Hitpoints").
package rates

import (
	"encoding/csv"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	PerHour float64 `json:"perHour"`
	Method  string  `json:"method"`
}

type SkillRows struct {
	Skill   string  `json:"skill"` // first-seen original casing
	Key     string  `json:"key"`
	Methods []Entry `json:"methods"`
}

type Store struct {
	mu      sync.Mutex
	path    string
	modTime *time.Time
	entries map[string][]Entry
	names   map[string]string
}

func New(path string) *Store {
	return &Store{path: path, entries: map[string][]Entry{}, names: map[string]string{}}
}

// Get returns the default (first) entry for a skill key. Missing skills yield
// the zero entry.
func (s *Store) Get(key string) Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refresh()
	if list := s.entries[strings.ToLower(key)]; len(list) > 0 {
		return list[0]
	}
	return Entry{}
}

// Methods returns every entry recorded for a skill, file order (default first).
func (s *Store) Methods(key string) []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refresh()
	list := s.entries[strings.ToLower(key)]
	out := make([]Entry, len(list))
	copy(out, list)
	return out
}

// Rows returns all skills with their method lists, sorted by skill name.
func (s *Store) Rows() []GroupedSkill {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refresh()
	keys := make([]string, 0, len(s.entries))
	for k := range s.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]GroupedSkill, 0, len(keys))
	for _, k := range keys {
		methods := append([]Entry{}, s.entries[k]...)
		out = append(out, GroupedSkill{Skill: s.names[k], Key: k, Methods: methods})
	}
	return out
}

type GroupedSkill struct {
	Skill   string  `json:"skill"`
	Key     string  `json:"key"`
	Methods []Entry `json:"methods"`
}

// Loaded reports whether the file exists with at least one usable row.
func (s *Store) Loaded() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refresh()
	return s.modTime != nil && len(s.entries) > 0
}

// Upsert replaces the whole method list for a skill while preserving the
// position of its first row, every comment, and row casing.
func (s *Store) Upsert(skillName string, e []Entry) error {
	skillName = strings.TrimSpace(skillName)
	key := strings.ToLower(skillName)
	if key == "" {
		return os.ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refresh()

	data, _ := os.ReadFile(s.path)
	if data == nil {
		data = []byte("skill,exp_per_hour,training_method\n")
	}

	lines := strings.Split(string(data), "\n")
	insert := -1 // index of the first row belonging to this skill
	existingCasing := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		cells := strings.SplitN(trimmed, ",", 2)
		first := strings.Trim(cells[0], `"`)
		if strings.EqualFold(strings.TrimSpace(first), "skill") {
			continue // header row
		}
		if strings.EqualFold(strings.TrimSpace(first), key) {
			if insert == -1 {
				insert = i
				existingCasing = strings.TrimSpace(first)
			}
		}
	}
	display := skillName
	if existingCasing != "" {
		display = existingCasing // keep the casing already used in the file
	}
	if insert == -1 {
		block := make([]string, 0, len(e)+1)
		for _, row := range e {
			block = append(block, encodeRow(display, row))
		}
		if len(block) == 0 {
			block = append(block, encodeRow(skillName, Entry{}))
		}
		out := strings.Join(lines, "\n")
		if !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		out += strings.Join(block, "\n") + "\n"
		if err := os.WriteFile(s.path, []byte(out), 0o644); err != nil {
			return err
		}
	} else {
		// remove all existing rows for this skill, insert the new block
		var kept []string
		for i, line := range lines {
			if i == insert {
				for _, row := range e {
					kept = append(kept, encodeRow(display, row))
				}
				continue
			}
			// skip any other rows for the same skill (they get replaced too)
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				if cells := strings.SplitN(trimmed, ",", 2); len(cells) > 0 {
					if strings.EqualFold(strings.TrimSpace(strings.Trim(cells[0], `"`)), key) {
						continue
					}
				}
			}
			kept = append(kept, line)
		}
		out := strings.Join(kept, "\n")
		if !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		if err := os.WriteFile(s.path, []byte(out), 0o644); err != nil {
			return err
		}
	}

	// reload state from the file we just wrote so caches are always fresh
	parsed, names, err2 := parse(s.path)
	if err2 == nil {
		s.entries = parsed
		s.names = names
	}
	if info, err := os.Stat(s.path); err == nil {
		t := info.ModTime()
		s.modTime = &t
	}
	return nil
}

// encodeRow renders one CSV line; methods containing commas/newlines are
// quoted per RFC 4180.
func encodeRow(skill string, e Entry) string {
	if e.Method == "" {
		return skill + "," + strconv.FormatFloat(e.PerHour, 'f', -1, 64) + ","
	}
	method := e.Method
	if strings.ContainsAny(method, ",\n") {
		method = `"` + strings.ReplaceAll(method, `"`, `""`) + `"`
	}
	return skill + "," + strconv.FormatFloat(e.PerHour, 'f', -1, 64) + "," + method
}

// refresh checks the file's mtime and re-reads it when changed.
// Callers must hold s.mu.
func (s *Store) refresh() {
	info, err := os.Stat(s.path)
	if err != nil {
		return // missing file simply means no estimates anywhere
	}
	if s.modTime != nil && info.ModTime().Equal(*s.modTime) {
		return
	}
	parsed, names, err := parse(s.path)
	if err != nil {
		// Keep the previous entries on malformed files; surface nothing.
		return
	}
	s.entries = parsed
	s.names = names
	mtime := info.ModTime()
	s.modTime = &mtime
}

func parse(path string) (map[string][]Entry, map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	out := map[string][]Entry{}
	names := map[string]string{}
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.Comment = '#'
	rows, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		if len(row) == 0 || strings.TrimSpace(row[0]) == "" {
			continue
		}
		name := strings.TrimSpace(row[0])
		key := strings.ToLower(name)
		if key == "skill" {
			continue // header row
		}
		e := Entry{}
		if len(row) >= 2 {
			raw := strings.TrimSpace(strings.ReplaceAll(row[1], ",", ""))
			e.PerHour, _ = strconv.ParseFloat(raw, 64)
			if e.PerHour < 0 {
				e.PerHour = 0
			}
		}
		if len(row) >= 3 {
			e.Method = strings.TrimSpace(row[2])
		}
		if _, seen := out[key]; !seen {
			out[key] = []Entry{}
			names[key] = name
		}
		out[key] = append(out[key], e)
	}
	return out, names, nil
}
