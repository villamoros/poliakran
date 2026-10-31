// Package httpapi expone el Catálogo como API HTTP JSON: el catálogo público
// (sin sesión) y la administración de cursos (con sesión y autorización).
package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/villamoros/poliakran/backend/internal/catalog/app"
	"github.com/villamoros/poliakran/backend/internal/catalog/domain"
	"github.com/villamoros/poliakran/backend/internal/platform/authz"
	"github.com/villamoros/poliakran/backend/internal/platform/httpx"
)

type Handler struct {
	svc *app.Service
	// requireAuth lo aporta el composition root (Identidad) para que este
	// contexto no dependa de otro (ADR-0001, regla 5).
	requireAuth func(http.Handler) http.Handler
	log         *slog.Logger
}

func NewHandler(svc *app.Service, requireAuth func(http.Handler) http.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, requireAuth: requireAuth, log: log}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/catalog/courses", h.listPublished)
	mux.HandleFunc("GET /api/v1/catalog/courses/{slug}", h.getPublished)

	mux.Handle("GET /api/v1/admin/courses", h.requireAuth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/admin/courses", h.requireAuth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/admin/courses/{id}", h.requireAuth(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/v1/admin/courses/{id}", h.requireAuth(http.HandlerFunc(h.update)))
	mux.Handle("POST /api/v1/admin/courses/{id}/publish", h.requireAuth(http.HandlerFunc(h.publish)))
	mux.Handle("POST /api/v1/admin/courses/{id}/withdraw", h.requireAuth(http.HandlerFunc(h.withdraw)))
}

// publicCourseDTO es lo que ve un visitante: sin datos internos de gestión.
type publicCourseDTO struct {
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary"`
	Description   string     `json:"description,omitempty"`
	Category      string     `json:"category"`
	Level         string     `json:"level"`
	DurationHours int        `json:"durationHours"`
	Objectives    []string   `json:"objectives,omitempty"`
	Prerequisites string     `json:"prerequisites,omitempty"`
	PublishedAt   *time.Time `json:"publishedAt"`
}

func toPublicDTO(c domain.Course, full bool) publicCourseDTO {
	dto := publicCourseDTO{
		Slug: c.Slug, Title: c.Title, Summary: c.Summary, Category: c.Category,
		Level: c.Level, DurationHours: c.DurationHours, PublishedAt: c.PublishedAt,
	}
	if full {
		dto.Description, dto.Objectives, dto.Prerequisites = c.Description, c.Objectives, c.Prerequisites
	}
	return dto
}

type adminCourseDTO struct {
	ID            string     `json:"id"`
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary"`
	Description   string     `json:"description"`
	Category      string     `json:"category"`
	Level         string     `json:"level"`
	DurationHours int        `json:"durationHours"`
	Objectives    []string   `json:"objectives"`
	Prerequisites string     `json:"prerequisites"`
	Status        string     `json:"status"`
	PublishedAt   *time.Time `json:"publishedAt"`
	WithdrawnAt   *time.Time `json:"withdrawnAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	Version       int        `json:"version"`
}

func toAdminDTO(c domain.Course) adminCourseDTO {
	return adminCourseDTO{
		ID: c.ID, Slug: c.Slug, Title: c.Title, Summary: c.Summary, Description: c.Description,
		Category: c.Category, Level: c.Level, DurationHours: c.DurationHours,
		Objectives: c.Objectives, Prerequisites: c.Prerequisites, Status: string(c.Status),
		PublishedAt: c.PublishedAt, WithdrawnAt: c.WithdrawnAt,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, Version: c.Version,
	}
}

// --- Catálogo público ---

func (h *Handler) listPublished(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, errL := atoiDefault(q.Get("limit"), app.DefaultPageSize)
	offset, errO := atoiDefault(q.Get("offset"), 0)
	if errL != nil || errO != nil || limit < 1 || limit > app.MaxPageSize || offset < 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid_query", "Parámetros de paginación inválidos.", nil)
		return
	}
	courses, total, err := h.svc.ListPublished(r.Context(), app.PublicQuery{Category: q.Get("category"), Limit: limit, Offset: offset})
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	items := make([]publicCourseDTO, 0, len(courses))
	for _, c := range courses {
		items = append(items, toPublicDTO(c, false))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

func (h *Handler) getPublished(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.GetPublished(r.Context(), r.PathValue("slug"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"course": toPublicDTO(c, true)})
}

// --- Administración ---

type courseRequest struct {
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	Description   string   `json:"description"`
	Category      string   `json:"category"`
	Level         string   `json:"level"`
	DurationHours int      `json:"durationHours"`
	Objectives    []string `json:"objectives"`
	Prerequisites string   `json:"prerequisites"`
}

func (req courseRequest) details() domain.Details {
	return domain.Details{
		Title: req.Title, Summary: req.Summary, Description: req.Description,
		Category: req.Category, Level: req.Level, DurationHours: req.DurationHours,
		Objectives: req.Objectives, Prerequisites: req.Prerequisites,
	}
}

type updateRequest struct {
	courseRequest
	// Version es la versión que el cliente leyó (bloqueo optimista).
	Version int `json:"version"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, _ := authz.PrincipalFrom(r.Context())
	courses, err := h.svc.List(r.Context(), p)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]adminCourseDTO, 0, len(courses))
	for _, c := range courses {
		items = append(items, toAdminDTO(c))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	p, _ := authz.PrincipalFrom(r.Context())
	var req courseRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", err.Error(), nil)
		return
	}
	c, err := h.svc.Create(r.Context(), p, req.details())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/admin/courses/"+c.ID)
	httpx.JSON(w, http.StatusCreated, map[string]any{"course": toAdminDTO(c)})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := courseID(w, r)
	if !ok {
		return
	}
	p, _ := authz.PrincipalFrom(r.Context())
	c, err := h.svc.Get(r.Context(), p, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"course": toAdminDTO(c)})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := courseID(w, r)
	if !ok {
		return
	}
	var req updateRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", err.Error(), nil)
		return
	}
	if req.Version < 1 {
		httpx.Error(w, http.StatusUnprocessableEntity, "validation_failed", "Revisa los datos del formulario.",
			map[string]string{"version": "Indica la versión del curso que editaste."})
		return
	}
	p, _ := authz.PrincipalFrom(r.Context())
	c, err := h.svc.Update(r.Context(), p, id, req.Version, req.details())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"course": toAdminDTO(c)})
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.svc.Publish)
}

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.svc.Withdraw)
}

func (h *Handler) transition(w http.ResponseWriter, r *http.Request, op func(ctx context.Context, p authz.Principal, id string) (domain.Course, error)) {
	id, ok := courseID(w, r)
	if !ok {
		return
	}
	p, _ := authz.PrincipalFrom(r.Context())
	c, err := op(r.Context(), p, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"course": toAdminDTO(c)})
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// courseID valida el identificador de la ruta; uno mal formado no puede
// existir, así que se responde 404 sin consultar la base de datos.
func courseID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		httpx.Error(w, http.StatusNotFound, "course_not_found", "El curso no existe.", nil)
		return "", false
	}
	return id, true
}

// fail traduce los errores de dominio y de autorización al contrato de la API.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve) && ve.Publish:
		httpx.Error(w, http.StatusUnprocessableEntity, "publish_requirements_unmet",
			"El curso aún no cumple las condiciones para publicarse.", ve.Fields)
	case errors.As(err, &ve):
		httpx.Error(w, http.StatusUnprocessableEntity, "validation_failed", "Revisa los datos del formulario.", ve.Fields)
	case errors.Is(err, authz.ErrUnauthenticated):
		httpx.Error(w, http.StatusUnauthorized, "unauthenticated", "Inicia sesión para continuar.", nil)
	case errors.Is(err, authz.ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "No tienes permiso para esta operación.", nil)
	case errors.Is(err, domain.ErrCourseNotFound):
		httpx.Error(w, http.StatusNotFound, "course_not_found", "El curso no existe.", nil)
	case errors.Is(err, domain.ErrVersionConflict):
		httpx.Error(w, http.StatusConflict, "version_conflict",
			"El curso cambió mientras lo editabas. Recarga para ver la última versión.", nil)
	case errors.Is(err, domain.ErrInvalidTransition):
		httpx.Error(w, http.StatusConflict, "invalid_transition", "Solo se puede retirar un curso publicado.", nil)
	default:
		h.internalError(w, r, err)
	}
}

func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	h.log.ErrorContext(r.Context(), "internal_error", "request_id", httpx.RequestIDFrom(r.Context()), "error", err.Error())
	httpx.Error(w, http.StatusInternalServerError, "internal_error", "Error interno del servidor.", nil)
}

func atoiDefault(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	return strconv.Atoi(s)
}
