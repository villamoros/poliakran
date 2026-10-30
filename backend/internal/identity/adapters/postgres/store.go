// Package postgres implementa el puerto Store de Identidad sobre PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/villamoros/poliakran/backend/internal/identity/app"
	"github.com/villamoros/poliakran/backend/internal/identity/domain"
	"github.com/villamoros/poliakran/backend/internal/platform/authz"
)

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

var _ app.Store = (*Store)(nil)

func (s *Store) RegisterStudent(ctx context.Context, u app.NewUser) (domain.User, domain.Tenant, error) {
	var user domain.User
	var tenant domain.Tenant

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO users (email, email_normalized, display_name, password_hash)
			VALUES ($1, $2, $3, $4)
			RETURNING id, email, display_name, status, created_at`,
			u.Email, u.EmailNormalized, u.DisplayName, u.PasswordHash,
		).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.CreatedAt)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_normalized_key" {
				return domain.ErrEmailTaken
			}
			return fmt.Errorf("insertando usuario: %w", err)
		}

		// Cada estudiante independiente tiene un tenant personal (ADR-0004).
		err = tx.QueryRow(ctx, `
			INSERT INTO tenants (slug, name, kind)
			VALUES ($1, $2, $3)
			RETURNING id, slug, name, kind`,
			"personal-"+strings.ReplaceAll(user.ID, "-", ""), user.DisplayName, domain.TenantKindPersonal,
		).Scan(&tenant.ID, &tenant.Slug, &tenant.Name, &tenant.Kind)
		if err != nil {
			return fmt.Errorf("insertando tenant personal: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO memberships (tenant_id, user_id, role) VALUES ($1, $2, $3)`,
			tenant.ID, user.ID, authz.RoleStudent)
		if err != nil {
			return fmt.Errorf("insertando membresía: %w", err)
		}
		return nil
	})
	return user, tenant, err
}

func (s *Store) FindCredentialsByEmail(ctx context.Context, emailNormalized string) (app.Credentials, error) {
	var c app.Credentials
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, display_name, status, created_at, password_hash
		FROM users WHERE email_normalized = $1`, emailNormalized,
	).Scan(&c.User.ID, &c.User.Email, &c.User.DisplayName, &c.User.Status, &c.User.CreatedAt, &c.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.Credentials{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return app.Credentials{}, fmt.Errorf("buscando credenciales: %w", err)
	}
	return c, nil
}

func (s *Store) DefaultTenantID(ctx context.Context, userID string) (string, error) {
	// Quien gestiona un tenant (owner/admin) entra en él; si no, se prioriza
	// el tenant personal y, después, la membresía más antigua (ADR-0008).
	// El cambio explícito de tenant activo llegará con LMS-US-016.
	var id string
	err := s.pool.QueryRow(ctx, `
		SELECT m.tenant_id
		FROM memberships m JOIN tenants t ON t.id = m.tenant_id
		WHERE m.user_id = $1
		ORDER BY (m.role IN ('owner', 'admin')) DESC, (t.kind = 'personal') DESC, m.created_at
		LIMIT 1`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("el usuario %s no tiene membresías", userID)
	}
	return id, err
}

func (s *Store) PlatformTenantID(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id FROM tenants WHERE kind = $1`, domain.TenantKindPlatform).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("no existe el tenant de plataforma: ¿faltan migraciones?")
	}
	return id, err
}

func (s *Store) GrantRole(ctx context.Context, emailNormalized, tenantID string, role authz.Role) error {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO memberships (tenant_id, user_id, role)
		SELECT $2, id, $3 FROM users WHERE email_normalized = $1
		ON CONFLICT (tenant_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		emailNormalized, tenantID, role)
	if err != nil {
		return fmt.Errorf("concediendo rol: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, n app.NewSession) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, tenant_id, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		n.TokenHash, n.UserID, n.TenantID, n.UserAgent, n.ExpiresAt)
	if err != nil {
		return fmt.Errorf("creando sesión: %w", err)
	}
	return nil
}

func (s *Store) FindActiveSession(ctx context.Context, tokenHash []byte, now time.Time) (app.SessionView, error) {
	var v app.SessionView
	var roles []string
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.display_name, u.status, u.created_at,
		       t.id, t.slug, t.name, t.kind,
		       COALESCE(array_agg(m.role) FILTER (WHERE m.role IS NOT NULL), '{}')
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		JOIN tenants t ON t.id = s.tenant_id
		LEFT JOIN memberships m ON m.tenant_id = s.tenant_id AND m.user_id = s.user_id
		WHERE s.token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > $2
		GROUP BY u.id, t.id`, tokenHash, now,
	).Scan(&v.User.ID, &v.User.Email, &v.User.DisplayName, &v.User.Status, &v.User.CreatedAt,
		&v.Tenant.ID, &v.Tenant.Slug, &v.Tenant.Name, &v.Tenant.Kind, &roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.SessionView{}, domain.ErrSessionNotFound
	}
	if err != nil {
		return app.SessionView{}, fmt.Errorf("buscando sesión: %w", err)
	}
	for _, r := range roles {
		v.Roles = append(v.Roles, authz.Role(r))
	}
	return v, nil
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash []byte, now time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash, now)
	if err != nil {
		return fmt.Errorf("revocando sesión: %w", err)
	}
	return nil
}
