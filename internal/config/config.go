// Package config loads optional configuration from a JSON file.
//
// Resolution order (later wins):
//  1. built-in defaults
//  2. config file (path from CONFIG, default ./config.json) - missing file is fine
//  3. environment variables (PORT, DB_PATH, STATIC_DIR, SKILL_RATES_PATH)
//
// Relative paths are resolved against the directory containing the config
// file (or the executable's directory when no config file exists), so the
// server behaves the same no matter which working directory it is started
// from.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	// Host to bind; the default "0.0.0.0" exposes the dashboard on all
	// network interfaces, "127.0.0.1" serves localhost only.
	Host string `json:"host"`
	Port int    `json:"port"`

	DBPath         string `json:"dbPath"`
	StaticDir      string `json:"staticDir"`
	SkillRatesPath string `json:"skillRatesPath"`
}

func defaults() Config {
	return Config{
		Host:           "0.0.0.0",
		Port:           8080,
		DBPath:         "./data/rs.db",
		StaticDir:      "./web/dist",
		SkillRatesPath: "./data/skill_rates.csv",
	}
}

// Load returns the effective configuration and a description of its source.
func Load(file string) (Config, string) {
	cfg := defaults()
	source := ""
	var baseDir string

	if data, err := os.ReadFile(file); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			fmt.Printf("warning: %s is not valid JSON (%v); falling back to defaults\n", file, err)
			cfg = defaults()
		} else {
			source = file
			baseDir = dirOrCwd(file)
		}
	}
	if baseDir == "" {
		baseDir = dirOfExecutable()
	}

	applyEnv(&cfg)
	cfg.resolveRelativeTo(baseDir)
	return cfg, source
}

func applyEnv(c *Config) {
	if v := os.Getenv("HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("PORT"); v != "" {
		c.Port = parseInt(v, c.Port)
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		c.DBPath = resolveFromCwdIfRelative(v)
	}
	if v := os.Getenv("STATIC_DIR"); v != "" {
		c.StaticDir = resolveFromCwdIfRelative(v)
	}
	if v := os.Getenv("SKILL_RATES_PATH"); v != "" {
		c.SkillRatesPath = resolveFromCwdIfRelative(v)
	}
}

// env-provided absolute or CWD-relative paths stay as given (relative ones
// are made absolute against the current directory, which is the documented
// env-var behaviour).
func resolveFromCwdIfRelative(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	return filepath.Join(cwd, p)
}

func (c *Config) resolveRelativeTo(dir string) {
	c.DBPath = resolveIfRelative(dir, c.DBPath)
	c.StaticDir = resolveIfRelative(dir, c.StaticDir)
	c.SkillRatesPath = resolveIfRelative(dir, c.SkillRatesPath)
}

func resolveIfRelative(dir, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(dir, p)
}

func dirOfExecutable() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

func dirOrCwd(p string) string {
	if d := filepath.Dir(filepath.Clean(p)); d != "" && d != "." {
		return d
	}
	return "."
}

// Addr renders the TCP address the HTTP server should listen on.
func (c Config) Addr() string {
	host := c.Host
	if host == "" {
		return fmt.Sprintf(":%d", c.Port)
	}
	return fmt.Sprintf("%s:%d", host, c.Port)
}

func parseInt(s string, def int) int {
	n := 0
	if _, err := fmt.Sscanf(s, "%d", &n); err == nil && n > 0 && n < 65536 {
		return n
	}
	return def
}
