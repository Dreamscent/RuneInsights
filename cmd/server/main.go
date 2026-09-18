// Command server runs the RuneInsights dashboard: REST API, snapshot tracker
// and (when built) the static frontend.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"runeinsights/internal/api"
	"runeinsights/internal/config"
	"runeinsights/internal/db"
	"runeinsights/internal/hiscore"
	"runeinsights/internal/rates"
	"runeinsights/internal/tracker"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	cfg, cfgSource := config.Load(env("CONFIG", "./config.json"))
	log.Printf("config: %s (port %d, host %s)", cfgSource, cfg.Port, cfg.Host)

	d, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer d.Close()

	client := hiscore.NewClient()
	tr := tracker.New(d, client)
	a := api.New(d, tr, client, rates.New(cfg.SkillRatesPath))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go tr.Run(ctx)

	mux := a.Routes()
	if _, err := os.Stat(filepath.Join(cfg.StaticDir, "index.html")); err == nil {
		mux.Handle("/", spaHandler(cfg.StaticDir))
		log.Printf("serving frontend from %s", cfg.StaticDir)
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("RuneInsights API is running.\nFrontend build not found; run `npm install && npm run build` in web/.\n"))
		})
		log.Printf("frontend build not found at %s (running API only)", cfg.StaticDir)
	}

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("RuneInsights listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Print("shutting down")
	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(sctx)
}

// spaHandler serves files from dir and falls back to index.html so the
// single-page app can own client-side routes.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean("/" + r.URL.Path)
		full := filepath.Join(dir, clean)
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
