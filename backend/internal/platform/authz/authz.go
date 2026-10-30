// Package authz centraliza la autorización del backend.
//
// Ningún handler decide por su cuenta si una operación está permitida: pregunta
// a un Authorizer con el principal, la acción y el recurso. Las reglas viven en
// una política única y versionada en código (ADR-0005).
package authz

import (
	"context"
	"errors"
	"slices"
)

type Role string

const (
	RoleStudent    Role = "student"
	RoleInstructor Role = "instructor"
	RoleAdmin      Role = "admin"
	RoleOwner      Role = "owner"
)

// Principal es la identidad autenticada que actúa dentro de un tenant.
type Principal struct {
	UserID   string
	TenantID string
	Roles    []Role
}

func (p Principal) HasRole(roles ...Role) bool {
	return slices.ContainsFunc(p.Roles, func(r Role) bool { return slices.Contains(roles, r) })
}

type Action string

const (
	ActionIdentityReadSelf Action = "identity:read_self"
	// Acciones de próximos sprints; se declaran aquí para fijar el vocabulario.
	ActionCatalogManage Action = "catalog:manage"
)

// Resource describe el objeto sobre el que se actúa. TenantID vacío significa
// recurso global; OwnerID vacío significa que no tiene propietario individual.
type Resource struct {
	TenantID string
	OwnerID  string
}

var (
	ErrUnauthenticated = errors.New("authz: no autenticado")
	ErrForbidden       = errors.New("authz: operación no permitida")
)

type Authorizer interface {
	Authorize(ctx context.Context, p Principal, a Action, r Resource) error
}

// rule decide sobre una acción ya filtrada por tenant.
type rule func(p Principal, r Resource) bool

// Policy es la política basada en roles y propiedad del MVP.
type Policy struct {
	rules map[Action]rule
}

func NewPolicy() *Policy {
	return &Policy{rules: map[Action]rule{
		ActionIdentityReadSelf: func(p Principal, r Resource) bool { return r.OwnerID == p.UserID },
		ActionCatalogManage:    func(p Principal, _ Resource) bool { return p.HasRole(RoleAdmin, RoleOwner) },
	}}
}

func (pol *Policy) Authorize(_ context.Context, p Principal, a Action, r Resource) error {
	if p.UserID == "" {
		return ErrUnauthenticated
	}
	// Aislamiento entre tenants: nunca se actúa sobre recursos de otro tenant.
	if r.TenantID != "" && r.TenantID != p.TenantID {
		return ErrForbidden
	}
	allow, ok := pol.rules[a]
	if !ok || !allow(p, r) {
		return ErrForbidden // denegar por defecto
	}
	return nil
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
