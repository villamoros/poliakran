// Package domain contiene el modelo y las reglas del contexto de Identidad.
// No depende de HTTP, de PostgreSQL ni de ningún proveedor externo.
package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

type User struct {
	ID          string
	Email       string
	DisplayName string
	Status      string
	CreatedAt   time.Time
}

type Tenant struct {
	ID   string
	Slug string
	Name string
	Kind string // "personal", "organization" o "platform"
}

const (
	TenantKindPersonal     = "personal"
	TenantKindOrganization = "organization"
	TenantKindPlatform     = "platform"

	MinPasswordLength = 10
	MaxPasswordLength = 128
	MaxEmailLength    = 254
)

var (
	ErrEmailTaken         = errors.New("identity: el correo ya está registrado")
	ErrInvalidCredentials = errors.New("identity: credenciales inválidas")
	ErrSessionNotFound    = errors.New("identity: sesión inválida o expirada")
	ErrUserDisabled       = errors.New("identity: usuario deshabilitado")
	ErrUserNotFound       = errors.New("identity: usuario no encontrado")
)

// ValidationError agrupa los errores de validación por campo.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "identity: datos inválidos" }

// NormalizeEmail produce la forma canónica usada para detectar duplicados.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Registration son los datos de alta ya validados.
type Registration struct {
	Email       string
	DisplayName string
	Password    string
}

// NewRegistration valida los datos de registro (LMS-US-001).
func NewRegistration(email, displayName, password string) (Registration, error) {
	fields := map[string]string{}

	email = strings.TrimSpace(email)
	if msg := validateEmail(email); msg != "" {
		fields["email"] = msg
	}

	displayName = strings.Join(strings.Fields(displayName), " ")
	switch n := utf8.RuneCountInString(displayName); {
	case n == 0:
		fields["displayName"] = "Escribe tu nombre."
	case n < 2 || n > 80:
		fields["displayName"] = "El nombre debe tener entre 2 y 80 caracteres."
	}

	if msg := validatePassword(password, email); msg != "" {
		fields["password"] = msg
	}

	if len(fields) > 0 {
		return Registration{}, &ValidationError{Fields: fields}
	}
	return Registration{Email: email, DisplayName: displayName, Password: password}, nil
}

func validateEmail(email string) string {
	if email == "" {
		return "Escribe tu correo electrónico."
	}
	if len(email) > MaxEmailLength {
		return "El correo es demasiado largo."
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email[strings.LastIndex(email, "@"):], ".") {
		return "Escribe un correo electrónico válido."
	}
	return ""
}

func validatePassword(password, email string) string {
	n := utf8.RuneCountInString(password)
	switch {
	case n == 0:
		return "Escribe una contraseña."
	case n < MinPasswordLength:
		return "La contraseña debe tener al menos 10 caracteres."
	case n > MaxPasswordLength:
		return "La contraseña no puede superar 128 caracteres."
	case strings.TrimSpace(password) == "":
		return "La contraseña no puede estar formada solo por espacios."
	case email != "" && strings.EqualFold(password, email):
		return "La contraseña no puede ser igual a tu correo."
	}
	return ""
}
