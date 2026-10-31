// Package postgres implementa el puerto Store del Catálogo sobre PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/villamoros/poliakran/backend/internal/catalog/app"
	"github.com/villamoros/poliakran/backend/internal/catalog/domain"
)

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

var _ app.Store = (*Store)(nil)

const courseColumns = `id, tenant_id, slug, title, summary, description, category, level,
	duration_hours, objectives, prerequisites, status, published_at, withdrawn_at,
	created_by, created_at, updated_at, version`

func scanCourse(row pgx.Row) (domain.Course, error) {
	var c domain.Course
	var status string
	err := row.Scan(&c.ID, &c.TenantID, &c.Slug, &c.Title, &c.Summary, &c.Description, &c.Category, &c.Level,
		&c.DurationHours, &c.Objectives, &c.Prerequisites, &status, &c.PublishedAt, &c.WithdrawnAt,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &c.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Course{}, domain.ErrCourseNotFound
	}
	c.Status = domain.Status(status)
	return c, err
}

func collect(rows pgx.Rows) ([]domain.Course, error) {
	defer rows.Close()
	out := []domain.Course{}
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Create(ctx context.Context, c domain.Course) (domain.Course, error) {
	created, err := scanCourse(s.pool.QueryRow(ctx, `
		INSERT INTO courses (tenant_id, slug, title, summary, description, category, level,
			duration_hours, objectives, prerequisites, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)
		RETURNING `+courseColumns,
		c.TenantID, c.Slug, c.Title, c.Summary, c.Description, c.Category, c.Level,
		c.DurationHours, c.Objectives, c.Prerequisites, c.Status, c.CreatedBy, c.CreatedAt))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "courses_tenant_slug_key" {
		return domain.Course{}, domain.ErrSlugTaken
	}
	if err != nil {
		return domain.Course{}, fmt.Errorf("insertando curso: %w", err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, tenantID, id string) (domain.Course, error) {
	c, err := scanCourse(s.pool.QueryRow(ctx,
		`SELECT `+courseColumns+` FROM courses WHERE tenant_id = $1 AND id = $2`, tenantID, id))
	if err != nil && !errors.Is(err, domain.ErrCourseNotFound) {
		return domain.Course{}, fmt.Errorf("leyendo curso: %w", err)
	}
	return c, err
}

func (s *Store) List(ctx context.Context, tenantID string) ([]domain.Course, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+courseColumns+` FROM courses WHERE tenant_id = $1 ORDER BY updated_at DESC, id`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listando cursos: %w", err)
	}
	return collect(rows)
}

func (s *Store) Update(ctx context.Context, c domain.Course, expectedVersion int) (domain.Course, error) {
	updated, err := scanCourse(s.pool.QueryRow(ctx, `
		UPDATE courses SET
			title = $4, summary = $5, description = $6, category = $7, level = $8,
			duration_hours = $9, objectives = $10, prerequisites = $11,
			status = $12, published_at = $13, withdrawn_at = $14,
			updated_at = $15, version = version + 1
		WHERE tenant_id = $1 AND id = $2 AND version = $3
		RETURNING `+courseColumns,
		c.TenantID, c.ID, expectedVersion,
		c.Title, c.Summary, c.Description, c.Category, c.Level,
		c.DurationHours, c.Objectives, c.Prerequisites,
		c.Status, c.PublishedAt, c.WithdrawnAt, c.UpdatedAt))
	if errors.Is(err, domain.ErrCourseNotFound) {
		// La fila existía al leerla: si ya no coincide es que otra
		// petición la modificó antes.
		return domain.Course{}, domain.ErrVersionConflict
	}
	if err != nil {
		return domain.Course{}, fmt.Errorf("actualizando curso: %w", err)
	}
	return updated, nil
}

func (s *Store) ListPublished(ctx context.Context, tenantID string, q app.PublicQuery) ([]domain.Course, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM courses
		WHERE tenant_id = $1 AND status = 'published' AND ($2 = '' OR category = $2)`,
		tenantID, q.Category).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("contando cursos publicados: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+courseColumns+` FROM courses
		WHERE tenant_id = $1 AND status = 'published' AND ($2 = '' OR category = $2)
		ORDER BY published_at DESC, id
		LIMIT $3 OFFSET $4`,
		tenantID, q.Category, q.Limit, q.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listando cursos publicados: %w", err)
	}
	courses, err := collect(rows)
	return courses, total, err
}

func (s *Store) GetPublishedBySlug(ctx context.Context, tenantID, slug string) (domain.Course, error) {
	c, err := scanCourse(s.pool.QueryRow(ctx,
		`SELECT `+courseColumns+` FROM courses WHERE tenant_id = $1 AND slug = $2 AND status = 'published'`,
		tenantID, slug))
	if err != nil && !errors.Is(err, domain.ErrCourseNotFound) {
		return domain.Course{}, fmt.Errorf("leyendo curso publicado: %w", err)
	}
	return c, err
}
