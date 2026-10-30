package config

import (
	"testing"
	"time"
)

func TestLoadDefaultsAndValidation(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("APP_ENV", "development")
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SessionTTL != 7*24*time.Hour || !cfg.CookieSecure || cfg.HTTPAddr != ":8080" {
		t.Fatalf("valores por defecto inesperados: %+v", cfg)
	}

	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_COOKIE_SECURE", "false")
	if _, err := Load(""); err == nil {
		t.Fatal("producción sin cookie Secure debe fallar")
	}

	t.Setenv("APP_ENV", "staging")
	if _, err := Load(""); err == nil {
		t.Fatal("APP_ENV desconocido debe fallar")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(""); err == nil {
		t.Fatal("se esperaba error sin DATABASE_URL")
	}
}
