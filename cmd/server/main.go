package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx driver for database/sql
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"

	"github.com/schliz/convoke/internal/auth"
	"github.com/schliz/convoke/internal/config"
	"github.com/schliz/convoke/internal/handler"
	"github.com/schliz/convoke/internal/middleware"
	"github.com/schliz/convoke/internal/render"
	"github.com/schliz/convoke/internal/store"
	"github.com/schliz/convoke/migrations"
)

func main() {
	// 1. Load config
	cfg := config.Load()
	slog.Info("starting convoke", "addr", cfg.ListenAddr, "dev", cfg.DevMode)

	ctx := context.Background()

	// 2. Database connection pool
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to create connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 3. Run migrations (goose requires database/sql)
	if err := runMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// 4. Create store
	s := store.New(pool)

	// 5. Create renderer
	rndr := render.New(cfg.TemplateDir, cfg.DevMode)

	// 6. CSS cache-busting
	cssPath := setupCSS(cfg.StaticDir, rndr)

	// 7. Create handler
	h := &handler.Handler{
		Store:    s,
		Renderer: rndr,
		Config:   cfg,
	}

	// 8. Middleware chains
	base := middleware.Chain(
		middleware.Logging(),
		middleware.Recovery(),
		auth.Middleware(s, cfg.AdminGroup, cfg.DevMode),
	)

	csrfSecret := cfg.CSRFSecret
	if csrfSecret == "" && cfg.DevMode {
		csrfSecret = "dev-csrf-secret-not-for-production"
	}
	withCSRF := middleware.Chain(
		base,
		middleware.CSRF(csrfSecret),
	)

	// 9. Routes
	mux := http.NewServeMux()

	// Health check — no auth, no CSRF
	mux.HandleFunc("GET /healthz", h.Wrap(h.HealthCheck))

	// Static files — no auth
	staticServer := http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticDir)))
	mux.Handle("GET /static/", staticServer)

	// Hashed CSS route for cache-busting
	if cssPath != "" {
		cssFile := filepath.Join(cfg.StaticDir, "css", "styles.css")
		mux.HandleFunc("GET "+cssPath, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			data, err := os.ReadFile(cssFile)
			if err != nil {
				http.Error(w, "CSS not found", http.StatusNotFound)
				return
			}
			w.Write(data)
		})
	}

	// Root route — redirects to unit or renders unit listing
	mux.Handle("GET /", base(h.Wrap(h.Home)))

	// Unit placeholder — will be replaced by the full unit dashboard
	mux.Handle("GET /units/{slug}", base(h.Wrap(h.UnitDashboard)))

	// Admin unit management routes
	mux.Handle("GET /admin/units/", withCSRF(auth.RequireAdmin(h.Wrap(h.AdminListUnits))))
	mux.Handle("GET /admin/units/new", withCSRF(auth.RequireAdmin(h.Wrap(h.AdminNewUnit))))
	mux.Handle("POST /admin/units/", withCSRF(auth.RequireAdmin(h.Wrap(h.AdminCreateUnit))))
	mux.Handle("GET /admin/units/{id}/edit", withCSRF(auth.RequireAdmin(h.Wrap(h.AdminEditUnit))))
	mux.Handle("POST /admin/units/{id}", withCSRF(auth.RequireAdmin(h.Wrap(h.AdminUpdateUnit))))
	mux.Handle("DELETE /admin/units/{id}", withCSRF(auth.RequireAdmin(h.Wrap(h.AdminDeleteUnit))))

	// 10. Server
	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	// 11. Graceful shutdown
	go func() {
		slog.Info("listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}

func runMigrations(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database for migrations: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	return goose.Up(db, ".")
}

func setupCSS(staticDir string, rndr *render.Renderer) string {
	cssFile := filepath.Join(staticDir, "css", "styles.css")
	data, err := os.ReadFile(cssFile)
	if err != nil {
		slog.Warn("CSS file not found, using fallback path", "path", cssFile)
		rndr.SetCSSPath("/static/css/styles.css")
		return ""
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(data))[:8]
	cssPath := fmt.Sprintf("/static/css/styles.%s.css", hash)
	rndr.SetCSSPath(cssPath)
	slog.Info("CSS cache-busting", "path", cssPath)
	return cssPath
}
