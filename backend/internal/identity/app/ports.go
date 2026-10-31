// Package app implementa los casos de uso del contexto de Identidad.
// Define los puertos que la infraestructura debe satisfacer (arquitectura hexagonal).
package app

import (
	"context"
	"time"

	"github.com/villamoros/poliakran/backend/internal/identity/domain"
	"github.com/villamoros/poliakran/backend/internal/platform/authz"
)

// NewUser es lo que el caso de uso pide persistir al registrar.
type NewUser struct {
	Email           string
	EmailNormalized string
	DisplayName     string
	PasswordHash    string
}

// Credentials es lo necesario para verificar un inicio de sesión.
type Credentials struct {
	User         domain.User
	PasswordHash string
}

// NewSession describe una sesión a crear. TokenHash nunca es el token en claro.
type NewSession struct {
	TokenHash []byte
	UserID    string
	TenantID  string
	UserAgent string
	ExpiresAt time.Time
}

// SessionView es la sesión resuelta con la identidad y el tenant activos.
type SessionView struct {
	User   domain.User
	Tenant domain.Tenant
	Roles  []authz.Role
}

// Store es el puerto de persistencia de Identidad.
type Store interface {
	// RegisterStudent crea, en una sola transacción, el usuario, su tenant
	// personal y la membresía de estudiante. Devuelve domain.ErrEmailTaken
	// si el correo normalizado ya existe.
	RegisterStudent(ctx context.Context, u NewUser) (domain.User, domain.Tenant, error)
	FindCredentialsByEmail(ctx context.Context, emailNormalized string) (Credentials, error)
	// DefaultTenantID devuelve el tenant con el que inicia sesión el usuario.
	DefaultTenantID(ctx context.Context, userID string) (string, error)
	// PlatformTenantID devuelve el tenant del propio Politécnico Akran (ADR-0008).
	PlatformTenantID(ctx context.Context) (string, error)
	// GrantRole crea o actualiza la membresía del usuario en el tenant.
	// Devuelve domain.ErrUserNotFound si el correo no está registrado.
	GrantRole(ctx context.Context, emailNormalized, tenantID string, role authz.Role) error
	CreateSession(ctx context.Context, s NewSession) error
	// FindActiveSession devuelve domain.ErrSessionNotFound si no existe,
	// expiró o fue revocada.
	FindActiveSession(ctx context.Context, tokenHash []byte, now time.Time) (SessionView, error)
	RevokeSession(ctx context.Context, tokenHash []byte, now time.Time) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encodedHash string) (bool, error)
}

// Tokens genera tokens de sesión opacos y su hash para almacenarlos.
type Tokens interface {
	New() (token string, hash []byte, err error)
	Hash(token string) []byte
}
