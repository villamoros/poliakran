package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewRegistration(t *testing.T) {
	tests := []struct {
		name, email, displayName, password string
		wantField                          string
	}{
		{"válido", "ana@example.com", "Ana Pérez", "contraseña-segura", ""},
		{"correo vacío", "", "Ana", "contraseña-segura", "email"},
		{"correo sin dominio", "ana@", "Ana", "contraseña-segura", "email"},
		{"correo sin punto en dominio", "ana@localhost", "Ana", "contraseña-segura", "email"},
		{"correo con nombre", "Ana <ana@example.com>", "Ana", "contraseña-segura", "email"},
		{"nombre vacío", "ana@example.com", "   ", "contraseña-segura", "displayName"},
		{"nombre corto", "ana@example.com", "A", "contraseña-segura", "displayName"},
		{"contraseña corta", "ana@example.com", "Ana", "corta", "password"},
		{"contraseña larga", "ana@example.com", "Ana", strings.Repeat("x", 129), "password"},
		{"contraseña solo espacios", "ana@example.com", "Ana", strings.Repeat(" ", 12), "password"},
		{"contraseña igual al correo", "ana@example.com", "Ana", "ANA@example.com", "password"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRegistration(tt.email, tt.displayName, tt.password)
			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("error inesperado: %v", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("se esperaba ValidationError, se obtuvo %v", err)
			}
			if _, ok := ve.Fields[tt.wantField]; !ok {
				t.Fatalf("se esperaba error en %q, campos: %v", tt.wantField, ve.Fields)
			}
		})
	}
}

func TestNewRegistrationNormalizesWhitespace(t *testing.T) {
	r, err := NewRegistration("  ana@example.com ", "  Ana   Pérez ", "contraseña-segura")
	if err != nil {
		t.Fatal(err)
	}
	if r.Email != "ana@example.com" || r.DisplayName != "Ana Pérez" {
		t.Fatalf("normalización inesperada: %+v", r)
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Ana@Example.COM "); got != "ana@example.com" {
		t.Fatalf("NormalizeEmail = %q", got)
	}
}
