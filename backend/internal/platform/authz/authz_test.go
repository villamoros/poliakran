package authz

import (
	"context"
	"errors"
	"testing"
)

func TestPolicy(t *testing.T) {
	pol := NewPolicy()
	student := Principal{UserID: "u1", TenantID: "t1", Roles: []Role{RoleStudent}}
	admin := Principal{UserID: "u2", TenantID: "t1", Roles: []Role{RoleAdmin}}
	owner := Principal{UserID: "u3", TenantID: "t1", Roles: []Role{RoleOwner}}
	instructor := Principal{UserID: "u4", TenantID: "t1", Roles: []Role{RoleInstructor}}

	tests := []struct {
		name string
		p    Principal
		a    Action
		r    Resource
		want error
	}{
		{"sin sesión", Principal{}, ActionIdentityReadSelf, Resource{}, ErrUnauthenticated},
		{"lee su propio perfil", student, ActionIdentityReadSelf, Resource{OwnerID: "u1"}, nil},
		{"no lee perfil ajeno", student, ActionIdentityReadSelf, Resource{OwnerID: "u2"}, ErrForbidden},
		{"estudiante no gestiona catálogo", student, ActionCatalogManage, Resource{TenantID: "t1"}, ErrForbidden},
		{"admin gestiona catálogo de su tenant", admin, ActionCatalogManage, Resource{TenantID: "t1"}, nil},
		{"admin no cruza tenants", admin, ActionCatalogManage, Resource{TenantID: "t2"}, ErrForbidden},
		{"owner gestiona catálogo", owner, ActionCatalogManage, Resource{TenantID: "t1"}, nil},
		{"instructor no gestiona catálogo todavía", instructor, ActionCatalogManage, Resource{TenantID: "t1"}, ErrForbidden},
		{"admin publica en su tenant", admin, ActionCatalogPublish, Resource{TenantID: "t1"}, nil},
		{"owner publica en su tenant", owner, ActionCatalogPublish, Resource{TenantID: "t1"}, nil},
		{"estudiante no publica", student, ActionCatalogPublish, Resource{TenantID: "t1"}, ErrForbidden},
		{"instructor no publica", instructor, ActionCatalogPublish, Resource{TenantID: "t1"}, ErrForbidden},
		{"admin no publica en otro tenant", admin, ActionCatalogPublish, Resource{TenantID: "t2"}, ErrForbidden},
		{"sin sesión no publica", Principal{}, ActionCatalogPublish, Resource{TenantID: "t1"}, ErrUnauthenticated},
		{"acción desconocida se deniega", admin, Action("x:y"), Resource{}, ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pol.Authorize(context.Background(), tt.p, tt.a, tt.r)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Authorize() = %v, se esperaba %v", err, tt.want)
			}
		})
	}
}
