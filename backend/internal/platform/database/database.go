// Package database abre el pool de conexiones a PostgreSQL y aplica migraciones.
package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/villamoros/poliakran/backend/migrations"
)

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("configurando el pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("conectando a PostgreSQL: %w", err)
	}
	return pool, nil
}

// Migrate ejecuta un comando de goose ("up", "down", "status", "version"...)
// sobre las migraciones incrustadas.
func Migrate(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger, command string, args ...string) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(gooseLogger{log})
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.RunContext(ctx, command, db, ".", args...)
}

type gooseLogger struct{ log *slog.Logger }

func (l gooseLogger) Printf(format string, v ...any) {
	l.log.Info(fmt.Sprintf(format, v...), "component", "migrations")
}

func (l gooseLogger) Fatalf(format string, v ...any) {
	l.log.Error(fmt.Sprintf(format, v...), "component", "migrations")
}
