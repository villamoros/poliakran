// Package app implementa los casos de uso del contexto de Catálogo.
//
// La autorización se decide aquí, en cada caso de uso, preguntando al
// Authorizer central (ADR-0005). Así cualquier adaptador (HTTP hoy, agentes de
// IA mañana, ADR-0007) pasa por las mismas reglas.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/villamoros/poliakran/backend/internal/catalog/domain"
	"github.com/villamoros/poliakran/backend/internal/platform/authz"
)

// Store es el puerto de persistencia del Catálogo. Todas las operaciones de
// administración reciben el tenant para que nunca se lean ni escriban cursos
// de otro tenant (ADR-0004).
type Store interface {
	// Create inserta el curso. Devuelve domain.ErrSlugTaken si el slug ya
	// existe en el tenant.
	Create(ctx context.Context, c domain.Course) (domain.Course, error)
	// Get devuelve domain.ErrCourseNotFound si no existe en el tenant.
	Get(ctx context.Context, tenantID, id string) (domain.Course, error)
	List(ctx context.Context, tenantID string) ([]domain.Course, error)
	// Update guarda el curso solo si su versión en la base de datos es
	// expectedVersion; si no, devuelve domain.ErrVersionConflict.
	Update(ctx context.Context, c domain.Course, expectedVersion int) (domain.Course, error)

	ListPublished(ctx context.Context, tenantID string, q PublicQuery) ([]domain.Course, int, error)
	// GetPublishedBySlug devuelve domain.ErrCourseNotFound si el curso no
	// existe o no está publicado.
	GetPublishedBySlug(ctx context.Context, tenantID, slug string) (domain.Course, error)
}

// PublicQuery filtra y pagina el catálogo público.
type PublicQuery struct {
	Category string
	Limit    int
	Offset   int
}

const (
	DefaultPageSize = 24
	MaxPageSize     = 100
	maxSlugAttempts = 20
)

type Service struct {
	store Store
	authz authz.Authorizer
	// catalogTenantID es el tenant cuyo catálogo es público (ADR-0008).
	catalogTenantID string
	now             func() time.Time
	log             *slog.Logger
}

func NewService(store Store, az authz.Authorizer, catalogTenantID string, log *slog.Logger) *Service {
	return &Service{store: store, authz: az, catalogTenantID: catalogTenantID, now: time.Now, log: log}
}

// --- Catálogo público (LMS-US-003) ---

// ListPublished devuelve los cursos publicados del catálogo público y el total.
// Los borradores y los retirados nunca aparecen.
func (s *Service) ListPublished(ctx context.Context, q PublicQuery) ([]domain.Course, int, error) {
	if q.Limit <= 0 {
		q.Limit = DefaultPageSize
	}
	q.Limit = min(q.Limit, MaxPageSize)
	q.Offset = max(q.Offset, 0)
	return s.store.ListPublished(ctx, s.catalogTenantID, q)
}

func (s *Service) GetPublished(ctx context.Context, slug string) (domain.Course, error) {
	return s.store.GetPublishedBySlug(ctx, s.catalogTenantID, slug)
}

// --- Administración (LMS-US-004, LMS-US-006) ---

// Create crea un curso en borrador en el tenant activo del principal.
func (s *Service) Create(ctx context.Context, p authz.Principal, in domain.Details) (domain.Course, error) {
	if err := s.authz.Authorize(ctx, p, authz.ActionCatalogManage, authz.Resource{TenantID: p.TenantID}); err != nil {
		return domain.Course{}, err
	}
	d, err := domain.NewDetails(in)
	if err != nil {
		return domain.Course{}, err
	}
	now := s.now()
	c := domain.Course{TenantID: p.TenantID, Status: domain.StatusDraft, CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now}
	if err := c.Apply(d, now); err != nil {
		return domain.Course{}, err
	}

	// El slug se deriva del título y es único por tenant: si ya existe se
	// añade un sufijo numérico.
	base := domain.Slugify(d.Title)
	for i := 1; i <= maxSlugAttempts; i++ {
		c.Slug = base
		if i > 1 {
			c.Slug = fmt.Sprintf("%s-%d", base, i)
		}
		created, err := s.store.Create(ctx, c)
		if errors.Is(err, domain.ErrSlugTaken) {
			continue
		}
		if err != nil {
			return domain.Course{}, err
		}
		s.log.InfoContext(ctx, "course_created", "course_id", created.ID, "tenant_id", created.TenantID, "user_id", p.UserID)
		return created, nil
	}
	return domain.Course{}, fmt.Errorf("no se encontró un slug libre para %q", base)
}

func (s *Service) List(ctx context.Context, p authz.Principal) ([]domain.Course, error) {
	if err := s.authz.Authorize(ctx, p, authz.ActionCatalogManage, authz.Resource{TenantID: p.TenantID}); err != nil {
		return nil, err
	}
	return s.store.List(ctx, p.TenantID)
}

func (s *Service) Get(ctx context.Context, p authz.Principal, id string) (domain.Course, error) {
	return s.load(ctx, p, authz.ActionCatalogManage, id)
}

// Update sustituye la información descriptiva. expectedVersion es la versión
// que el cliente leyó: si otro cambio llegó antes, se rechaza con
// domain.ErrVersionConflict en lugar de sobrescribirlo.
func (s *Service) Update(ctx context.Context, p authz.Principal, id string, expectedVersion int, in domain.Details) (domain.Course, error) {
	c, err := s.load(ctx, p, authz.ActionCatalogManage, id)
	if err != nil {
		return domain.Course{}, err
	}
	if c.Version != expectedVersion {
		return domain.Course{}, domain.ErrVersionConflict
	}
	d, err := domain.NewDetails(in)
	if err != nil {
		return domain.Course{}, err
	}
	if err := c.Apply(d, s.now()); err != nil {
		return domain.Course{}, err
	}
	updated, err := s.store.Update(ctx, c, expectedVersion)
	if err != nil {
		return domain.Course{}, err
	}
	s.log.InfoContext(ctx, "course_updated", "course_id", c.ID, "tenant_id", c.TenantID, "user_id", p.UserID, "version", updated.Version)
	return updated, nil
}

// Publish publica el curso si cumple las condiciones mínimas. Es idempotente.
func (s *Service) Publish(ctx context.Context, p authz.Principal, id string) (domain.Course, error) {
	return s.transition(ctx, p, id, "course_published", (*domain.Course).Publish)
}

// Withdraw retira un curso publicado del catálogo. Es idempotente.
func (s *Service) Withdraw(ctx context.Context, p authz.Principal, id string) (domain.Course, error) {
	return s.transition(ctx, p, id, "course_withdrawn", (*domain.Course).Withdraw)
}

func (s *Service) transition(ctx context.Context, p authz.Principal, id, event string, apply func(*domain.Course, time.Time) (bool, error)) (domain.Course, error) {
	c, err := s.load(ctx, p, authz.ActionCatalogPublish, id)
	if err != nil {
		return domain.Course{}, err
	}
	version := c.Version
	changed, err := apply(&c, s.now())
	if err != nil || !changed {
		return c, err
	}
	updated, err := s.store.Update(ctx, c, version)
	if err != nil {
		return domain.Course{}, err
	}
	s.log.InfoContext(ctx, event, "course_id", c.ID, "tenant_id", c.TenantID, "user_id", p.UserID)
	return updated, nil
}

// load lee el curso dentro del tenant activo y autoriza la acción sobre él.
// Un curso de otro tenant se comporta como inexistente.
func (s *Service) load(ctx context.Context, p authz.Principal, action authz.Action, id string) (domain.Course, error) {
	if err := s.authz.Authorize(ctx, p, action, authz.Resource{TenantID: p.TenantID}); err != nil {
		return domain.Course{}, err
	}
	c, err := s.store.Get(ctx, p.TenantID, id)
	if err != nil {
		return domain.Course{}, err
	}
	if err := s.authz.Authorize(ctx, p, action, authz.Resource{TenantID: c.TenantID}); err != nil {
		return domain.Course{}, err
	}
	return c, nil
}
