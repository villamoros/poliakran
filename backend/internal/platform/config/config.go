// Package config carga la configuración del backend desde variables de entorno.
// En desarrollo puede leer un archivo .env; las variables ya definidas en el
// entorno siempre tienen prioridad sobre el archivo.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Env string

const (
	EnvDevelopment Env = "development"
	EnvTest        Env = "test"
	EnvProduction  Env = "production"
)

type Config struct {
	Env            Env
	HTTPAddr       string
	DatabaseURL    string
	LogLevel       string
	MigrateOnStart bool

	SessionTTL     time.Duration
	CookieName     string
	CookieSecure   bool
	AllowedOrigins []string
	TrustProxy     bool

	AuthRateLimit       int
	AuthRateLimitWindow time.Duration
}

// Load lee la configuración. Si dotenvPath no está vacío y el archivo existe,
// sus valores se usan como respaldo de las variables de entorno.
func Load(dotenvPath string) (Config, error) {
	if dotenvPath != "" {
		if err := loadDotenv(dotenvPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("leyendo %s: %w", dotenvPath, err)
		}
	}

	cfg := Config{
		Env:            Env(get("APP_ENV", string(EnvDevelopment))),
		HTTPAddr:       get("HTTP_ADDR", ":8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		LogLevel:       get("LOG_LEVEL", "info"),
		CookieName:     get("SESSION_COOKIE_NAME", "pa_session"),
		AllowedOrigins: splitList(get("ALLOWED_ORIGINS", "http://localhost:3000")),
	}

	var err error
	if cfg.MigrateOnStart, err = getBool("MIGRATE_ON_START", false); err != nil {
		return Config{}, err
	}
	if cfg.CookieSecure, err = getBool("SESSION_COOKIE_SECURE", true); err != nil {
		return Config{}, err
	}
	if cfg.TrustProxy, err = getBool("TRUST_PROXY", false); err != nil {
		return Config{}, err
	}
	if cfg.SessionTTL, err = getDuration("SESSION_TTL", 7*24*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.AuthRateLimitWindow, err = getDuration("AUTH_RATE_LIMIT_WINDOW", time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.AuthRateLimit, err = getInt("AUTH_RATE_LIMIT", 10); err != nil {
		return Config{}, err
	}

	return cfg, cfg.validate()
}

func (c Config) validate() error {
	switch c.Env {
	case EnvDevelopment, EnvTest, EnvProduction:
	default:
		return fmt.Errorf("APP_ENV inválido: %q", c.Env)
	}
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL es obligatoria")
	}
	if c.Env == EnvProduction && !c.CookieSecure {
		return errors.New("SESSION_COOKIE_SECURE debe ser true en producción")
	}
	if c.SessionTTL <= 0 {
		return errors.New("SESSION_TTL debe ser positivo")
	}
	return nil
}

func get(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getBool(key string, fallback bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: valor booleano inválido %q", key, v)
	}
	return b, nil
}

func getInt(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: entero inválido %q", key, v)
	}
	return n, nil
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: duración inválida %q", key, v)
	}
	return d, nil
}

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadDotenv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
	return s.Err()
}
