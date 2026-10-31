// Comando api arranca el backend del LMS (monolito modular).
//
//	go run ./cmd/api            # arranca el servidor HTTP
//	go run ./cmd/api migrate up # aplica migraciones (también: status, down, version)
//	go run ./cmd/api grant-role correo@ejemplo.com owner
//	                            # concede owner/admin de la plataforma a un usuario ya registrado
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	cataloghttp "github.com/villamoros/poliakran/backend/internal/catalog/adapters/httpapi"
	catalogpg "github.com/villamoros/poliakran/backend/internal/catalog/adapters/postgres"
	catalog "github.com/villamoros/poliakran/backend/internal/catalog/app"
	"github.com/villamoros/poliakran/backend/internal/identity/adapters/httpapi"
	"github.com/villamoros/poliakran/backend/internal/identity/adapters/postgres"
	"github.com/villamoros/poliakran/backend/internal/identity/adapters/security"
	"github.com/villamoros/poliakran/backend/internal/identity/app"
	"github.com/villamoros/poliakran/backend/internal/identity/domain"
	"github.com/villamoros/poliakran/backend/internal/platform/authz"
	"github.com/villamoros/poliakran/backend/internal/platform/config"
	"github.com/villamoros/poliakran/backend/internal/platform/database"
	"github.com/villamoros/poliakran/backend/internal/platform/httpx"
	"github.com/villamoros/poliakran/backend/internal/platform/logging"
	"github.com/villamoros/poliakran/backend/internal/platform/ratelimit"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.Load(".env")
	if err != nil {
		return err
	}
	log := logging.New(os.Stdout, cfg.LogLevel).With("service", "poliakran-api", "env", string(cfg.Env))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if len(args) > 0 && args[0] == "migrate" {
		cmd := "up"
		if len(args) > 1 {
			cmd = args[1]
		}
		return database.Migrate(ctx, pool, log, cmd, args[min(2, len(args)):]...)
	}

	if len(args) > 0 && args[0] == "grant-role" {
		return grantRole(ctx, pool, log, args[1:])
	}

	if cfg.MigrateOnStart {
		if err := database.Migrate(ctx, pool, log, "up"); err != nil {
			return fmt.Errorf("migraciones: %w", err)
		}
	}

	handler, err := NewServer(cfg, pool, log)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server_started", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("server_stopping")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}
	log.Info("server_stopped")
	return nil
}

// NewServer compone los módulos y devuelve el handler HTTP raíz.
func NewServer(cfg config.Config, pool *pgxpool.Pool, log *slog.Logger) (http.Handler, error) {
	authorizer := authz.NewPolicy()

	identitySvc, err := app.NewService(
		postgres.NewStore(pool),
		security.NewArgon2Hasher(security.DefaultArgon2Params),
		security.OpaqueTokens{},
		cfg.SessionTTL,
		log.With("module", "identity"),
	)
	if err != nil {
		return nil, err
	}
	identityHTTP := httpapi.NewHandler(
		identitySvc, authorizer,
		httpapi.CookieConfig{Name: cfg.CookieName, Secure: cfg.CookieSecure},
		ratelimit.New(cfg.AuthRateLimit, cfg.AuthRateLimitWindow),
		cfg.TrustProxy,
		log.With("module", "identity"),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			httpx.Error(w, http.StatusServiceUnavailable, "not_ready", "La base de datos no responde.", nil)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	identityHTTP.Register(mux)

	// El Catálogo recibe del composition root el tenant de plataforma y el
	// middleware de autenticación: no importa nada del contexto de Identidad.
	platformTenantID, err := identitySvc.PlatformTenantID(context.Background())
	if err != nil {
		return nil, err
	}
	catalogSvc := catalog.NewService(catalogpg.NewStore(pool), authorizer, platformTenantID, log.With("module", "catalog"))
	cataloghttp.NewHandler(catalogSvc, identityHTTP.RequireAuth, log.With("module", "catalog")).Register(mux)

	return httpx.Chain(mux,
		httpx.RequestID,
		httpx.AccessLog(log),
		httpx.Recover(log),
		httpx.SecurityHeaders,
		httpx.CSRFGuard(cfg.AllowedOrigins),
	), nil
}

// grantRole implementa el arranque seguro de LMS-US-014: el primer owner se
// crea desde la línea de comandos del servidor, nunca desde la API pública.
func grantRole(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger, args []string) error {
	if len(args) != 2 {
		return errors.New("uso: api grant-role <correo> <owner|admin>")
	}
	svc, err := app.NewService(postgres.NewStore(pool), security.NewArgon2Hasher(security.DefaultArgon2Params),
		security.OpaqueTokens{}, time.Hour, log.With("module", "identity"))
	if err != nil {
		return err
	}
	if err := svc.GrantPlatformRole(ctx, args[0], authz.Role(args[1])); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return errors.New("no hay ningún usuario registrado con ese correo; regístralo primero en /registro")
		}
		return err
	}
	fmt.Printf("Rol %s concedido en la plataforma. El usuario debe volver a iniciar sesión.\n", args[1])
	return nil
}
