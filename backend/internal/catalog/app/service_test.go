package app_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/villamoros/poliakran/backend/internal/catalog/app"
	"github.com/villamoros/poliakran/backend/internal/catalog/domain"
	"github.com/villamoros/poliakran/backend/internal/platform/authz"
	"github.com/villamoros/poliakran/backend/internal/platform/logging"
)

// memStore es un doble en memoria del puerto Store.
type memStore struct {
	mu      sync.Mutex
	courses map[string]domain.Course
	seq     int
}

func newMemStore() *memStore { return &memStore{courses: map[string]domain.Course{}} }

func (m *memStore) Create(_ context.Context, c domain.Course) (domain.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, o := range m.courses {
		if o.TenantID == c.TenantID && o.Slug == c.Slug {
			return domain.Course{}, domain.ErrSlugTaken
		}
	}
	m.seq++
	c.ID = fmt.Sprintf("course-%d", m.seq)
	c.Version = 1
	m.courses[c.ID] = c
	return c, nil
}

func (m *memStore) Get(_ context.Context, tenantID, id string) (domain.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.courses[id]
	if !ok || c.TenantID != tenantID {
		return domain.Course{}, domain.ErrCourseNotFound
	}
	return c, nil
}

func (m *memStore) List(_ context.Context, tenantID string) ([]domain.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Course
	for _, c := range m.courses {
		if c.TenantID == tenantID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *memStore) Update(_ context.Context, c domain.Course, expected int) (domain.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.courses[c.ID]
	if !ok || cur.TenantID != c.TenantID || cur.Version != expected {
		return domain.Course{}, domain.ErrVersionConflict
	}
	c.Version = expected + 1
	m.courses[c.ID] = c
	return c, nil
}

func (m *memStore) ListPublished(_ context.Context, tenantID string, q app.PublicQuery) ([]domain.Course, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Course
	for _, c := range m.courses {
		if c.TenantID == tenantID && c.Status == domain.StatusPublished && (q.Category == "" || c.Category == q.Category) {
			out = append(out, c)
		}
	}
	total := len(out)
	end := min(q.Offset+q.Limit, total)
	if q.Offset >= total {
		return nil, total, nil
	}
	return out[q.Offset:end], total, nil
}

func (m *memStore) GetPublishedBySlug(_ context.Context, tenantID, slug string) (domain.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.courses {
		if c.TenantID == tenantID && c.Slug == slug && c.Status == domain.StatusPublished {
			return c, nil
		}
	}
	return domain.Course{}, domain.ErrCourseNotFound
}

const platform = "tenant-platform"

var (
	admin   = authz.Principal{UserID: "u-admin", TenantID: platform, Roles: []authz.Role{authz.RoleAdmin}}
	student = authz.Principal{UserID: "u-student", TenantID: "tenant-personal", Roles: []authz.Role{authz.RoleStudent}}
	orgAdm  = authz.Principal{UserID: "u-org", TenantID: "tenant-org", Roles: []authz.Role{authz.RoleAdmin}}
)

func newService(store app.Store, logOut io.Writer) *app.Service {
	return app.NewService(store, authz.NewPolicy(), platform, logging.New(logOut, "debug"))
}

func completeDetails(title string) domain.Details {
	return domain.Details{
		Title: title, Summary: "Un curso corto y práctico de competencias digitales.",
		Description: strings.Repeat("Descripción completa del curso. ", 3),
		Category:    "programacion", Level: "basico", DurationHours: 4,
		Objectives: []string{"Resolver un problema real"},
	}
}

func TestCreateStartsAsDraftWithUniqueSlug(t *testing.T) {
	ctx := context.Background()
	svc := newService(newMemStore(), io.Discard)

	a, err := svc.Create(ctx, admin, domain.Details{Title: "Introducción a Go", Summary: "Fundamentos del lenguaje Go paso a paso."})
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != domain.StatusDraft || a.TenantID != platform || a.CreatedBy != admin.UserID {
		t.Fatalf("curso inesperado: %+v", a)
	}
	b, err := svc.Create(ctx, admin, domain.Details{Title: "Introducción a Go", Summary: "Otra edición del mismo curso de Go."})
	if err != nil {
		t.Fatal(err)
	}
	if a.Slug != "introduccion-a-go" || b.Slug != "introduccion-a-go-2" {
		t.Fatalf("slugs inesperados: %q, %q", a.Slug, b.Slug)
	}
}

func TestOnlyAuthorizedPrincipalsManageCourses(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	svc := newService(store, io.Discard)
	draft := domain.Details{Title: "Curso de SQL", Summary: "Consultas básicas para analizar datos."}

	if _, err := svc.Create(ctx, student, draft); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("un estudiante no crea cursos: %v", err)
	}
	if _, err := svc.Create(ctx, authz.Principal{}, draft); !errors.Is(err, authz.ErrUnauthenticated) {
		t.Fatalf("sin sesión no se crean cursos: %v", err)
	}
	c, _ := svc.Create(ctx, admin, completeDetails("Curso de SQL"))

	// Otro tenant no ve ni modifica el curso: se comporta como inexistente.
	if _, err := svc.Get(ctx, orgAdm, c.ID); !errors.Is(err, domain.ErrCourseNotFound) {
		t.Fatalf("admin de otro tenant: %v", err)
	}
	if _, err := svc.Publish(ctx, orgAdm, c.ID); !errors.Is(err, domain.ErrCourseNotFound) {
		t.Fatalf("admin de otro tenant no publica: %v", err)
	}
	if _, err := svc.Publish(ctx, student, c.ID); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("un estudiante no publica: %v", err)
	}
	if list, _ := svc.List(ctx, orgAdm); len(list) != 0 {
		t.Fatalf("el listado de administración no cruza tenants: %v", list)
	}
}

func TestUpdateUsesOptimisticLocking(t *testing.T) {
	ctx := context.Background()
	svc := newService(newMemStore(), io.Discard)
	c, _ := svc.Create(ctx, admin, domain.Details{Title: "Curso de SQL", Summary: "Consultas básicas para analizar datos."})

	d := completeDetails("Curso de SQL avanzado")
	updated, err := svc.Update(ctx, admin, c.ID, c.Version, d)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Curso de SQL avanzado" || updated.Version != c.Version+1 {
		t.Fatalf("edición inesperada: %+v", updated)
	}
	if updated.Slug != c.Slug {
		t.Fatal("el slug no cambia al editar: las URL publicadas son estables")
	}
	// Repetir la misma petición con la versión antigua no sobrescribe.
	if _, err := svc.Update(ctx, admin, c.ID, c.Version, d); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("versión obsoleta: %v", err)
	}
}

func TestPublicCatalogShowsOnlyPublishedPlatformCourses(t *testing.T) {
	ctx := context.Background()
	svc := newService(newMemStore(), io.Discard)

	pub, _ := svc.Create(ctx, admin, completeDetails("Desarrollo web con HTML"))
	_, _ = svc.Create(ctx, admin, completeDetails("Borrador sin publicar"))
	wd, _ := svc.Create(ctx, admin, completeDetails("Curso que se retira"))
	org, _ := svc.Create(ctx, orgAdm, completeDetails("Curso interno de una empresa"))
	for _, c := range []domain.Course{pub, wd} {
		if _, err := svc.Publish(ctx, admin, c.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Publish(ctx, orgAdm, org.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Withdraw(ctx, admin, wd.ID); err != nil {
		t.Fatal(err)
	}

	items, total, err := svc.ListPublished(ctx, app.PublicQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != pub.ID {
		t.Fatalf("el catálogo público solo debe mostrar el curso publicado de la plataforma: %+v", items)
	}
	if _, err := svc.GetPublished(ctx, wd.Slug); !errors.Is(err, domain.ErrCourseNotFound) {
		t.Fatalf("un curso retirado no tiene detalle público: %v", err)
	}
	if _, err := svc.GetPublished(ctx, pub.Slug); err != nil {
		t.Fatalf("detalle público: %v", err)
	}
}

func TestPublishIsIdempotentAndChecksConditions(t *testing.T) {
	ctx := context.Background()
	svc := newService(newMemStore(), io.Discard)
	c, _ := svc.Create(ctx, admin, domain.Details{Title: "Curso de SQL", Summary: "Consultas básicas para analizar datos."})

	var ve *domain.ValidationError
	if _, err := svc.Publish(ctx, admin, c.ID); !errors.As(err, &ve) || !ve.Publish {
		t.Fatalf("un curso incompleto no se publica: %v", err)
	}
	c, _ = svc.Update(ctx, admin, c.ID, c.Version, completeDetails("Curso de SQL"))
	first, err := svc.Publish(ctx, admin, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Publish(ctx, admin, c.ID)
	if err != nil || second.Version != first.Version {
		t.Fatalf("publicar dos veces no debe cambiar nada: %v, v%d → v%d", err, first.Version, second.Version)
	}
}

func TestAuditLogsContainOnlyIdentifiers(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	svc := newService(newMemStore(), &buf)
	c, _ := svc.Create(ctx, admin, completeDetails("Curso secreto de prueba"))
	_, _ = svc.Publish(ctx, admin, c.ID)

	out := buf.String()
	for _, event := range []string{"course_created", "course_published"} {
		if !strings.Contains(out, event) {
			t.Fatalf("falta el evento %s:\n%s", event, out)
		}
	}
	if strings.Contains(out, "Curso secreto") {
		t.Fatalf("los eventos no deben incluir contenido del curso:\n%s", out)
	}
}
