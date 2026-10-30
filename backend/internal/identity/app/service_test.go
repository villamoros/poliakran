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
	"time"

	"github.com/villamoros/poliakran/backend/internal/identity/adapters/security"
	"github.com/villamoros/poliakran/backend/internal/identity/app"
	"github.com/villamoros/poliakran/backend/internal/identity/domain"
	"github.com/villamoros/poliakran/backend/internal/platform/authz"
	"github.com/villamoros/poliakran/backend/internal/platform/logging"
)

// memStore es un doble en memoria del puerto Store.
type memStore struct {
	mu       sync.Mutex
	users    map[string]app.Credentials // por email normalizado
	tenants  map[string]string          // user_id -> tenant_id
	sessions map[string]app.NewSession  // por hash
	revoked  map[string]bool
}

func newMemStore() *memStore {
	return &memStore{users: map[string]app.Credentials{}, tenants: map[string]string{},
		sessions: map[string]app.NewSession{}, revoked: map[string]bool{}}
}

func (m *memStore) RegisterStudent(_ context.Context, u app.NewUser) (domain.User, domain.Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[u.EmailNormalized]; ok {
		return domain.User{}, domain.Tenant{}, domain.ErrEmailTaken
	}
	user := domain.User{ID: fmt.Sprintf("user-%d", len(m.users)+1), Email: u.Email, DisplayName: u.DisplayName, Status: "active"}
	tenant := domain.Tenant{ID: "tenant-" + user.ID, Kind: domain.TenantKindPersonal, Name: u.DisplayName}
	m.users[u.EmailNormalized] = app.Credentials{User: user, PasswordHash: u.PasswordHash}
	m.tenants[user.ID] = tenant.ID
	return user, tenant, nil
}

func (m *memStore) FindCredentialsByEmail(_ context.Context, email string) (app.Credentials, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.users[email]
	if !ok {
		return app.Credentials{}, domain.ErrInvalidCredentials
	}
	return c, nil
}

func (m *memStore) DefaultTenantID(_ context.Context, userID string) (string, error) {
	return m.tenants[userID], nil
}

func (m *memStore) PlatformTenantID(context.Context) (string, error) { return "tenant-platform", nil }

func (m *memStore) GrantRole(_ context.Context, email, tenantID string, _ authz.Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.users[email]
	if !ok {
		return domain.ErrUserNotFound
	}
	m.tenants[c.User.ID] = tenantID
	return nil
}

func (m *memStore) CreateSession(_ context.Context, s app.NewSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[string(s.TokenHash)] = s
	return nil
}

func (m *memStore) FindActiveSession(_ context.Context, hash []byte, now time.Time) (app.SessionView, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[string(hash)]
	if !ok || m.revoked[string(hash)] || !s.ExpiresAt.After(now) {
		return app.SessionView{}, domain.ErrSessionNotFound
	}
	for _, c := range m.users {
		if c.User.ID == s.UserID {
			return app.SessionView{User: c.User, Tenant: domain.Tenant{ID: s.TenantID}, Roles: []authz.Role{authz.RoleStudent}}, nil
		}
	}
	return app.SessionView{}, domain.ErrSessionNotFound
}

func (m *memStore) RevokeSession(_ context.Context, hash []byte, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revoked[string(hash)] = true
	return nil
}

var fastHasher = security.NewArgon2Hasher(security.Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32})

func newService(t *testing.T, store app.Store, logOut io.Writer) *app.Service {
	t.Helper()
	svc, err := app.NewService(store, fastHasher, security.OpaqueTokens{}, time.Hour, logging.New(logOut, "debug"))
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestRegisterOpensSessionAndAuthenticates(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, newMemStore(), io.Discard)

	res, err := svc.Register(ctx, app.RegisterInput{Email: "Ana@Example.com", DisplayName: "Ana", Password: "contraseña-segura"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Token == "" {
		t.Fatal("se esperaba un token de sesión")
	}
	view, p, err := svc.Authenticate(ctx, res.Token)
	if err != nil {
		t.Fatal(err)
	}
	if p.UserID != res.User.ID || view.User.Email != "Ana@Example.com" || p.TenantID == "" {
		t.Fatalf("principal inesperado: %+v", p)
	}
}

func TestRegisterRejectsDuplicateEmailCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, newMemStore(), io.Discard)
	in := app.RegisterInput{Email: "ana@example.com", DisplayName: "Ana", Password: "contraseña-segura"}
	if _, err := svc.Register(ctx, in); err != nil {
		t.Fatal(err)
	}
	in.Email = "ANA@example.com"
	if _, err := svc.Register(ctx, in); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("se esperaba ErrEmailTaken, se obtuvo %v", err)
	}
}

func TestRegisterValidates(t *testing.T) {
	svc := newService(t, newMemStore(), io.Discard)
	_, err := svc.Register(context.Background(), app.RegisterInput{Email: "x", Password: "1"})
	var ve *domain.ValidationError
	if !errors.As(err, &ve) || len(ve.Fields) != 3 {
		t.Fatalf("se esperaban 3 errores de campo, se obtuvo %v", err)
	}
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, newMemStore(), io.Discard)
	if _, err := svc.Register(ctx, app.RegisterInput{Email: "ana@example.com", DisplayName: "Ana", Password: "contraseña-segura"}); err != nil {
		t.Fatal(err)
	}

	res, err := svc.Login(ctx, app.LoginInput{Email: " ANA@example.com ", Password: "contraseña-segura"})
	if err != nil {
		t.Fatalf("login válido falló: %v", err)
	}
	if _, _, err := svc.Authenticate(ctx, res.Token); err != nil {
		t.Fatal(err)
	}

	for name, in := range map[string]app.LoginInput{
		"contraseña incorrecta": {Email: "ana@example.com", Password: "otra-contraseña"},
		"correo desconocido":    {Email: "nadie@example.com", Password: "contraseña-segura"},
		"vacío":                 {},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Login(ctx, in); !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Fatalf("se esperaba ErrInvalidCredentials, se obtuvo %v", err)
			}
		})
	}
}

func TestLogoutRevokesSessionAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, newMemStore(), io.Discard)
	res, _ := svc.Register(ctx, app.RegisterInput{Email: "ana@example.com", DisplayName: "Ana", Password: "contraseña-segura"})

	if err := svc.Logout(ctx, res.Token); err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(ctx, res.Token); err != nil {
		t.Fatalf("logout repetido debe ser idempotente: %v", err)
	}
	if _, _, err := svc.Authenticate(ctx, res.Token); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("la sesión revocada no debe autenticar: %v", err)
	}
}

func TestAuthenticateRejectsUnknownToken(t *testing.T) {
	svc := newService(t, newMemStore(), io.Discard)
	for _, tok := range []string{"", "token-inventado"} {
		if _, _, err := svc.Authenticate(context.Background(), tok); !errors.Is(err, domain.ErrSessionNotFound) {
			t.Fatalf("token %q: se esperaba ErrSessionNotFound, se obtuvo %v", tok, err)
		}
	}
}

func TestCredentialsNeverReachLogs(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	svc := newService(t, newMemStore(), &buf)

	res, _ := svc.Register(ctx, app.RegisterInput{Email: "ana@example.com", DisplayName: "Ana", Password: "contraseña-segura"})
	_, _ = svc.Login(ctx, app.LoginInput{Email: "ana@example.com", Password: "contraseña-incorrecta"})
	_, _ = svc.Login(ctx, app.LoginInput{Email: "ana@example.com", Password: "contraseña-segura"})

	out := buf.String()
	if out == "" {
		t.Fatal("se esperaban eventos de auditoría en el log")
	}
	for _, secret := range []string{"contraseña-segura", "contraseña-incorrecta", res.Token, "ana@example.com"} {
		if strings.Contains(out, secret) {
			t.Fatalf("el log contiene un dato sensible %q:\n%s", secret, out)
		}
	}
}

func TestGrantPlatformRole(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	svc := newService(t, store, io.Discard)
	res, err := svc.Register(ctx, app.RegisterInput{Email: "ana@example.com", DisplayName: "Ana", Password: "contraseña-segura"})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.GrantPlatformRole(ctx, " ANA@example.com ", authz.RoleOwner); err != nil {
		t.Fatalf("conceder owner: %v", err)
	}
	if store.tenants[res.User.ID] != "tenant-platform" {
		t.Fatalf("la membresía no apunta al tenant de plataforma: %v", store.tenants)
	}
	if err := svc.GrantPlatformRole(ctx, "nadie@example.com", authz.RoleAdmin); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("correo desconocido: se esperaba ErrUserNotFound, se obtuvo %v", err)
	}
	if err := svc.GrantPlatformRole(ctx, "ana@example.com", authz.RoleStudent); err == nil {
		t.Fatal("solo owner y admin pueden concederse en la plataforma")
	}
}
