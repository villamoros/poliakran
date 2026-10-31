package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/villamoros/poliakran/backend/internal/identity/domain"
	"github.com/villamoros/poliakran/backend/internal/platform/authz"
)

type Service struct {
	store      Store
	hasher     PasswordHasher
	tokens     Tokens
	sessionTTL time.Duration
	now        func() time.Time
	log        *slog.Logger

	// dummyHash permite gastar el mismo tiempo de verificación cuando el
	// correo no existe, para no revelar qué cuentas están registradas.
	dummyHash string
}

func NewService(store Store, hasher PasswordHasher, tokens Tokens, sessionTTL time.Duration, log *slog.Logger) (*Service, error) {
	dummy, err := hasher.Hash("poliakran-dummy-password")
	if err != nil {
		return nil, fmt.Errorf("preparando hash de referencia: %w", err)
	}
	return &Service{
		store: store, hasher: hasher, tokens: tokens,
		sessionTTL: sessionTTL, now: time.Now, log: log, dummyHash: dummy,
	}, nil
}

type RegisterInput struct {
	Email       string
	DisplayName string
	Password    string
	UserAgent   string
}

type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
}

// AuthResult es la sesión recién emitida. Token solo debe viajar en la cookie.
type AuthResult struct {
	User      domain.User
	Token     string
	ExpiresAt time.Time
}

// Register implementa LMS-US-001: crea la cuenta de estudiante y abre sesión.
func (s *Service) Register(ctx context.Context, in RegisterInput) (AuthResult, error) {
	reg, err := domain.NewRegistration(in.Email, in.DisplayName, in.Password)
	if err != nil {
		return AuthResult{}, err
	}
	hash, err := s.hasher.Hash(reg.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash de contraseña: %w", err)
	}
	user, tenant, err := s.store.RegisterStudent(ctx, NewUser{
		Email:           reg.Email,
		EmailNormalized: domain.NormalizeEmail(reg.Email),
		DisplayName:     reg.DisplayName,
		PasswordHash:    hash,
	})
	if err != nil {
		return AuthResult{}, err
	}
	s.log.InfoContext(ctx, "user_registered", "user_id", user.ID, "tenant_id", tenant.ID)
	return s.openSession(ctx, user, tenant.ID, in.UserAgent)
}

// Login implementa LMS-US-002. Cualquier fallo de credenciales devuelve el
// mismo error genérico.
func (s *Service) Login(ctx context.Context, in LoginInput) (AuthResult, error) {
	if in.Email == "" || in.Password == "" || len(in.Password) > 4*domain.MaxPasswordLength {
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	creds, err := s.store.FindCredentialsByEmail(ctx, domain.NormalizeEmail(in.Email))
	if errors.Is(err, domain.ErrInvalidCredentials) {
		_, _ = s.hasher.Verify(in.Password, s.dummyHash)
		s.log.InfoContext(ctx, "login_failed", "reason", "unknown_email")
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return AuthResult{}, err
	}
	ok, err := s.hasher.Verify(in.Password, creds.PasswordHash)
	if err != nil {
		return AuthResult{}, fmt.Errorf("verificando contraseña: %w", err)
	}
	if !ok {
		s.log.InfoContext(ctx, "login_failed", "reason", "bad_password", "user_id", creds.User.ID)
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	if creds.User.Status != "active" {
		s.log.InfoContext(ctx, "login_failed", "reason", "user_disabled", "user_id", creds.User.ID)
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	tenantID, err := s.store.DefaultTenantID(ctx, creds.User.ID)
	if err != nil {
		return AuthResult{}, err
	}
	s.log.InfoContext(ctx, "login_succeeded", "user_id", creds.User.ID, "tenant_id", tenantID)
	return s.openSession(ctx, creds.User, tenantID, in.UserAgent)
}

// Authenticate resuelve un token de sesión en un principal autorizable.
func (s *Service) Authenticate(ctx context.Context, token string) (SessionView, authz.Principal, error) {
	if token == "" {
		return SessionView{}, authz.Principal{}, domain.ErrSessionNotFound
	}
	view, err := s.store.FindActiveSession(ctx, s.tokens.Hash(token), s.now())
	if err != nil {
		return SessionView{}, authz.Principal{}, err
	}
	if view.User.Status != "active" {
		return SessionView{}, authz.Principal{}, domain.ErrSessionNotFound
	}
	return view, authz.Principal{UserID: view.User.ID, TenantID: view.Tenant.ID, Roles: view.Roles}, nil
}

// Logout revoca la sesión. Es idempotente: revocar dos veces no es un error.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.RevokeSession(ctx, s.tokens.Hash(token), s.now())
}

// PlatformTenantID devuelve el tenant cuyo catálogo es público (ADR-0008).
func (s *Service) PlatformTenantID(ctx context.Context) (string, error) {
	return s.store.PlatformTenantID(ctx)
}

// GrantPlatformRole concede un rol de gestión en el tenant de plataforma.
// Solo se usa desde el comando de arranque (LMS-US-014), nunca desde la API
// pública: así se crea el primer owner de forma segura.
func (s *Service) GrantPlatformRole(ctx context.Context, email string, role authz.Role) error {
	if role != authz.RoleOwner && role != authz.RoleAdmin {
		return fmt.Errorf("rol no admitido para la plataforma: %q", role)
	}
	tenantID, err := s.store.PlatformTenantID(ctx)
	if err != nil {
		return err
	}
	if err := s.store.GrantRole(ctx, domain.NormalizeEmail(email), tenantID, role); err != nil {
		return err
	}
	s.log.InfoContext(ctx, "platform_role_granted", "tenant_id", tenantID, "role", string(role))
	return nil
}

func (s *Service) openSession(ctx context.Context, user domain.User, tenantID, userAgent string) (AuthResult, error) {
	token, hash, err := s.tokens.New()
	if err != nil {
		return AuthResult{}, fmt.Errorf("generando token: %w", err)
	}
	if len(userAgent) > 512 {
		userAgent = userAgent[:512]
	}
	expires := s.now().Add(s.sessionTTL)
	if err := s.store.CreateSession(ctx, NewSession{
		TokenHash: hash, UserID: user.ID, TenantID: tenantID,
		UserAgent: userAgent, ExpiresAt: expires,
	}); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{User: user, Token: token, ExpiresAt: expires}, nil
}
