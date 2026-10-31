// Package domain contiene el modelo y las reglas del contexto de Catálogo:
// el curso, su información descriptiva y su ciclo de publicación.
// No depende de HTTP, de PostgreSQL ni de otros contextos.
package domain

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Status es el estado del ciclo de vida: borrador → publicado → retirado.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusWithdrawn Status = "withdrawn"
)

// Categorías iniciales del catálogo (documento maestro, §6.2).
var Categories = []string{"programacion", "bases-de-datos", "desarrollo-web", "herramientas-software", "otras"}

// Niveles de dificultad.
var Levels = []string{"basico", "intermedio", "avanzado"}

const (
	MinTitleLength       = 5
	MaxTitleLength       = 120
	MinSummaryLength     = 20
	MaxSummaryLength     = 280
	MinPublishDescLength = 50
	MaxDescriptionLength = 10000
	MaxPrerequisites     = 1000
	MaxObjectives        = 10
	MaxObjectiveLength   = 200
	MaxDurationHours     = 500
)

var (
	ErrCourseNotFound    = errors.New("catalog: curso no encontrado")
	ErrVersionConflict   = errors.New("catalog: el curso cambió desde que se leyó")
	ErrInvalidTransition = errors.New("catalog: transición de estado no permitida")
	ErrSlugTaken         = errors.New("catalog: el slug ya existe en el tenant")
)

// ValidationError agrupa los errores por campo. Publish indica que los datos
// son válidos como borrador pero no cumplen las condiciones de publicación.
type ValidationError struct {
	Fields  map[string]string
	Publish bool
}

func (e *ValidationError) Error() string {
	if e.Publish {
		return "catalog: el curso no cumple las condiciones de publicación"
	}
	return "catalog: datos inválidos"
}

// Course es un curso del catálogo de un tenant.
type Course struct {
	ID            string
	TenantID      string
	Slug          string
	Title         string
	Summary       string
	Description   string
	Category      string
	Level         string
	DurationHours int
	Objectives    []string
	Prerequisites string
	Status        Status
	PublishedAt   *time.Time
	WithdrawnAt   *time.Time
	CreatedBy     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	// Version se incrementa en cada cambio y permite detectar ediciones
	// concurrentes o repetidas (bloqueo optimista).
	Version int
}

// Details es la información descriptiva editable de un curso.
type Details struct {
	Title         string
	Summary       string
	Description   string
	Category      string
	Level         string
	DurationHours int
	Objectives    []string
	Prerequisites string
}

// NewDetails normaliza y valida los datos de un curso. Solo título y
// descripción breve son obligatorios para guardar un borrador (LMS-US-004);
// el resto se exige al publicar (ver CheckPublishable).
func NewDetails(in Details) (Details, error) {
	fields := map[string]string{}
	d := Details{
		Title:         collapseSpaces(in.Title),
		Summary:       collapseSpaces(in.Summary),
		Description:   strings.TrimSpace(in.Description),
		Category:      strings.TrimSpace(in.Category),
		Level:         strings.TrimSpace(in.Level),
		DurationHours: in.DurationHours,
		Prerequisites: strings.TrimSpace(in.Prerequisites),
		Objectives:    []string{},
	}

	switch n := utf8.RuneCountInString(d.Title); {
	case n == 0:
		fields["title"] = "Escribe el título del curso."
	case n < MinTitleLength || n > MaxTitleLength:
		fields["title"] = "El título debe tener entre 5 y 120 caracteres."
	}
	switch n := utf8.RuneCountInString(d.Summary); {
	case n == 0:
		fields["summary"] = "Escribe una descripción breve."
	case n < MinSummaryLength || n > MaxSummaryLength:
		fields["summary"] = "La descripción breve debe tener entre 20 y 280 caracteres."
	}
	if utf8.RuneCountInString(d.Description) > MaxDescriptionLength {
		fields["description"] = "La descripción completa no puede superar 10 000 caracteres."
	}
	if d.Category != "" && !slices.Contains(Categories, d.Category) {
		fields["category"] = "Elige una categoría de la lista."
	}
	if d.Level != "" && !slices.Contains(Levels, d.Level) {
		fields["level"] = "Elige un nivel de la lista."
	}
	if d.DurationHours < 0 || d.DurationHours > MaxDurationHours {
		fields["durationHours"] = "La duración debe estar entre 1 y 500 horas."
	}
	if utf8.RuneCountInString(d.Prerequisites) > MaxPrerequisites {
		fields["prerequisites"] = "Los requisitos previos no pueden superar 1000 caracteres."
	}

	for _, o := range in.Objectives {
		if o = collapseSpaces(o); o != "" {
			d.Objectives = append(d.Objectives, o)
		}
	}
	if len(d.Objectives) > MaxObjectives {
		fields["objectives"] = "Indica como máximo 10 objetivos."
	}
	for _, o := range d.Objectives {
		if n := utf8.RuneCountInString(o); n < 3 || n > MaxObjectiveLength {
			fields["objectives"] = "Cada objetivo debe tener entre 3 y 200 caracteres."
			break
		}
	}

	if len(fields) > 0 {
		return Details{}, &ValidationError{Fields: fields}
	}
	return d, nil
}

// CheckPublishable comprueba las condiciones mínimas de publicación
// (LMS-US-006): un curso incompleto no puede llegar al catálogo público.
// En el Sprint 3 se añadirá la exigencia de contenido (módulos y lecciones).
func (d Details) CheckPublishable() error {
	fields := map[string]string{}
	if utf8.RuneCountInString(d.Description) < MinPublishDescLength {
		fields["description"] = "Para publicar, la descripción completa debe tener al menos 50 caracteres."
	}
	if d.Category == "" {
		fields["category"] = "Para publicar, elige una categoría."
	}
	if d.Level == "" {
		fields["level"] = "Para publicar, elige un nivel."
	}
	if d.DurationHours <= 0 {
		fields["durationHours"] = "Para publicar, indica la duración estimada."
	}
	if len(d.Objectives) == 0 {
		fields["objectives"] = "Para publicar, indica al menos un objetivo de aprendizaje."
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields, Publish: true}
	}
	return nil
}

// Details devuelve la información descriptiva del curso.
func (c Course) Details() Details {
	return Details{
		Title: c.Title, Summary: c.Summary, Description: c.Description,
		Category: c.Category, Level: c.Level, DurationHours: c.DurationHours,
		Objectives: c.Objectives, Prerequisites: c.Prerequisites,
	}
}

// Apply sustituye la información descriptiva. Un curso publicado debe seguir
// cumpliendo las condiciones de publicación tras la edición.
func (c *Course) Apply(d Details, now time.Time) error {
	if c.Status == StatusPublished {
		if err := d.CheckPublishable(); err != nil {
			return err
		}
	}
	c.Title, c.Summary, c.Description = d.Title, d.Summary, d.Description
	c.Category, c.Level, c.DurationHours = d.Category, d.Level, d.DurationHours
	c.Objectives, c.Prerequisites = d.Objectives, d.Prerequisites
	c.UpdatedAt = now
	return nil
}

// Publish pasa el curso a publicado. Devuelve changed=false si ya lo estaba,
// de modo que repetir la operación no es un error.
func (c *Course) Publish(now time.Time) (changed bool, err error) {
	switch c.Status {
	case StatusPublished:
		return false, nil
	case StatusDraft, StatusWithdrawn:
		if err := c.Details().CheckPublishable(); err != nil {
			return false, err
		}
		c.Status = StatusPublished
		c.PublishedAt = &now
		c.WithdrawnAt = nil
		c.UpdatedAt = now
		return true, nil
	}
	return false, ErrInvalidTransition
}

// Withdraw retira un curso publicado del catálogo. No borra nada: el
// historial de inscripciones y progreso se conserva (documento maestro, §6.3).
func (c *Course) Withdraw(now time.Time) (changed bool, err error) {
	switch c.Status {
	case StatusWithdrawn:
		return false, nil
	case StatusPublished:
		c.Status = StatusWithdrawn
		c.WithdrawnAt = &now
		c.UpdatedAt = now
		return true, nil
	}
	return false, ErrInvalidTransition
}

// Slugify genera el identificador legible de la URL a partir del título:
// "Introducción a Go" → "introduccion-a-go".
func Slugify(title string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	plain, _, _ := transform.String(t, strings.ToLower(title))
	var b strings.Builder
	dash := false
	for _, r := range plain {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.TrimSuffix(b.String(), "-")
	if len(s) > 80 {
		s = strings.TrimSuffix(s[:80], "-")
	}
	if s == "" {
		s = "curso"
	}
	return s
}

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
