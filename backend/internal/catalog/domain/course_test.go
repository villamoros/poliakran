package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func validDetails() Details {
	return Details{
		Title:         "Introducción a Go",
		Summary:       "Aprende los fundamentos del lenguaje Go en pocas horas.",
		Description:   strings.Repeat("Contenido descriptivo del curso. ", 3),
		Category:      "programacion",
		Level:         "basico",
		DurationHours: 6,
		Objectives:    []string{"Escribir programas sencillos en Go"},
	}
}

func fieldsOf(t *testing.T, err error) map[string]string {
	t.Helper()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("se esperaba ValidationError, se obtuvo %v", err)
	}
	return ve.Fields
}

func TestNewDetailsRequiresTitleAndSummaryOnly(t *testing.T) {
	d, err := NewDetails(Details{Title: "  Curso   de SQL ", Summary: "Consultas básicas para analizar datos."})
	if err != nil {
		t.Fatalf("un borrador mínimo debe ser válido: %v", err)
	}
	if d.Title != "Curso de SQL" {
		t.Fatalf("el título debe normalizar espacios: %q", d.Title)
	}

	fields := fieldsOf(t, func() error { _, err := NewDetails(Details{}); return err }())
	for _, f := range []string{"title", "summary"} {
		if fields[f] == "" {
			t.Fatalf("falta error de %s: %v", f, fields)
		}
	}
}

func TestNewDetailsRejectsInvalidValues(t *testing.T) {
	in := validDetails()
	in.Category = "cocina"
	in.Level = "experto"
	in.DurationHours = 501
	in.Objectives = []string{"ok", "Objetivo válido"}
	fields := fieldsOf(t, func() error { _, err := NewDetails(in); return err }())
	for _, f := range []string{"category", "level", "durationHours", "objectives"} {
		if fields[f] == "" {
			t.Fatalf("falta error de %s: %v", f, fields)
		}
	}
}

func TestNewDetailsDropsEmptyObjectives(t *testing.T) {
	in := validDetails()
	in.Objectives = []string{"  ", "Diseñar tablas", ""}
	d, err := NewDetails(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Objectives) != 1 || d.Objectives[0] != "Diseñar tablas" {
		t.Fatalf("objetivos inesperados: %q", d.Objectives)
	}
}

func TestPublishRequiresMinimumConditions(t *testing.T) {
	now := time.Now()
	c := Course{Status: StatusDraft}
	_ = c.Apply(Details{Title: "Curso de SQL", Summary: "Consultas básicas para analizar datos."}, now)

	_, err := c.Publish(now)
	fields := fieldsOf(t, err)
	if !err.(*ValidationError).Publish {
		t.Fatal("el error debe marcarse como de publicación")
	}
	for _, f := range []string{"description", "category", "level", "durationHours", "objectives"} {
		if fields[f] == "" {
			t.Fatalf("falta condición %s: %v", f, fields)
		}
	}
	if c.Status != StatusDraft {
		t.Fatal("un curso incompleto debe seguir en borrador")
	}
}

func TestLifecycle(t *testing.T) {
	now := time.Now()
	c := Course{Status: StatusDraft}
	_ = c.Apply(validDetails(), now)

	if _, err := c.Withdraw(now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("un borrador no se puede retirar: %v", err)
	}
	if changed, err := c.Publish(now); err != nil || !changed || c.Status != StatusPublished || c.PublishedAt == nil {
		t.Fatalf("publicar: changed=%v err=%v %+v", changed, err, c)
	}
	if changed, err := c.Publish(now); err != nil || changed {
		t.Fatalf("publicar de nuevo debe ser idempotente: changed=%v err=%v", changed, err)
	}
	if changed, err := c.Withdraw(now); err != nil || !changed || c.Status != StatusWithdrawn || c.WithdrawnAt == nil {
		t.Fatalf("retirar: changed=%v err=%v", changed, err)
	}
	if changed, err := c.Withdraw(now); err != nil || changed {
		t.Fatalf("retirar de nuevo debe ser idempotente: changed=%v err=%v", changed, err)
	}
	if changed, err := c.Publish(now); err != nil || !changed || c.WithdrawnAt != nil {
		t.Fatalf("un curso retirado puede volver a publicarse: changed=%v err=%v", changed, err)
	}
}

func TestPublishedCourseMustStayPublishable(t *testing.T) {
	now := time.Now()
	c := Course{Status: StatusDraft}
	_ = c.Apply(validDetails(), now)
	_, _ = c.Publish(now)

	d := validDetails()
	d.Description = ""
	if err := c.Apply(d, now); err == nil {
		t.Fatal("no se puede dejar incompleto un curso publicado")
	}
	if c.Description == "" {
		t.Fatal("una edición rechazada no debe modificar el curso")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Introducción a Go":            "introduccion-a-go",
		"  Bases de Datos: SQL & más ": "bases-de-datos-sql-mas",
		"¡¡¡":                          "curso",
		"Año 2026 — Ñandú":             "ano-2026-nandu",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, se esperaba %q", in, got, want)
		}
	}
	if got := Slugify(strings.Repeat("palabra ", 30)); len(got) > 80 || strings.HasSuffix(got, "-") {
		t.Errorf("slug largo mal recortado: %q", got)
	}
}
