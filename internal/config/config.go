package config

import "os"

type Config struct {
	DatabaseURL string
	ListenAddr  string
	DevMode     bool
	AdminGroup  string
	TemplateDir string
	StaticDir   string
	CSRFSecret  string
}

func Load() *Config {
	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		ListenAddr:  envOr("LISTEN_ADDR", ":8080"),
		DevMode:     os.Getenv("DEV_MODE") == "true",
		AdminGroup:  envOr("ADMIN_GROUP", "admin"),
		TemplateDir: envOr("TEMPLATE_DIR", "templates"),
		StaticDir:   envOr("STATIC_DIR", "static"),
		CSRFSecret:  os.Getenv("CSRF_SECRET"),
	}
	if cfg.DatabaseURL == "" {
		panic("DATABASE_URL is required")
	}
	if cfg.CSRFSecret == "" && !cfg.DevMode {
		panic("CSRF_SECRET is required in production")
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
