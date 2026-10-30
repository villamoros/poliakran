package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/villamoros/poliakran/backend/internal/platform/logging"
)

// registerAdmin registra un usuario, le concede el rol con el comando de
// arranque (LMS-US-014) y devuelve un cliente con sesión iniciada.
func registerAdmin(t *testing.T, srvURL string, role string) *http.Client {
	t.Helper()
	_, pool := setup(t)
	email := uniqueEmail(t)
	password := "contraseña-segura"
	if r := call(t, newClient(t), "POST", srvURL+"/api/v1/auth/register", map[string]string{
		"email": email, "displayName": "Admin Akran", "password": password,
	}); r.Status != http.StatusCreated {
		t.Fatalf("registro: %d %v", r.Status, r.Body)
	}
	if err := grantRole(context.Background(), pool, logging.New(io.Discard, "error"), []string{email, role}); err != nil {
		t.Fatalf("grant-role: %v", err)
	}
	c := newClient(t)
	if r := call(t, c, "POST", srvURL+"/api/v1/auth/login", map[string]string{"email": email, "password": password}); r.Status != http.StatusOK {
		t.Fatalf("login: %d %v", r.Status, r.Body)
	}
	return c
}

func registerStudent(t *testing.T, srvURL string) *http.Client {
	t.Helper()
	c := newClient(t)
	if r := call(t, c, "POST", srvURL+"/api/v1/auth/register", map[string]string{
		"email": uniqueEmail(t), "displayName": "Estudiante", "password": "contraseña-segura",
	}); r.Status != http.StatusCreated {
		t.Fatalf("registro: %d %v", r.Status, r.Body)
	}
	return c
}

func courseOf(r apiResponse) map[string]any {
	c, _ := r.Body["course"].(map[string]any)
	return c
}

func publicSlugs(t *testing.T, srvURL string) []string {
	t.Helper()
	r := call(t, newClient(t), "GET", srvURL+"/api/v1/catalog/courses?limit=100", nil)
	if r.Status != http.StatusOK {
		t.Fatalf("catálogo: %d %v", r.Status, r.Body)
	}
	var slugs []string
	for _, it := range r.Body["items"].([]any) {
		item := it.(map[string]any)
		if _, leaked := item["id"]; leaked {
			t.Fatalf("el catálogo público no debe exponer ids internos: %v", item)
		}
		slugs = append(slugs, item["slug"].(string))
	}
	return slugs
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestCatalogAdministrationAndPublicCatalog(t *testing.T) {
	srv, _ := setup(t)
	admin := registerAdmin(t, srv.URL, "admin")

	// Quien administra la plataforma inicia sesión en ella.
	me := call(t, admin, "GET", srv.URL+"/api/v1/auth/me", nil)
	if tenant := me.Body["tenant"].(map[string]any); tenant["kind"] != "platform" || fmt.Sprint(me.Body["roles"]) != "[admin]" {
		t.Fatalf("el admin debe entrar en el tenant de plataforma: %v", me.Body)
	}

	// LMS-US-004: campos obligatorios.
	bad := call(t, admin, "POST", srv.URL+"/api/v1/admin/courses", map[string]any{"title": "", "summary": "corta"})
	if bad.Status != http.StatusUnprocessableEntity || errorCode(bad) != "validation_failed" {
		t.Fatalf("validación: %d %v", bad.Status, bad.Body)
	}

	// Se crea como borrador.
	title := fmt.Sprintf("Fundamentos de SQL %d", time.Now().UnixNano())
	created := call(t, admin, "POST", srv.URL+"/api/v1/admin/courses", map[string]any{
		"title": title, "summary": "Consultas básicas para analizar datos con SQL.",
	})
	if created.Status != http.StatusCreated {
		t.Fatalf("crear: %d %v", created.Status, created.Body)
	}
	course := courseOf(created)
	id, slug := course["id"].(string), course["slug"].(string)
	if course["status"] != "draft" || !strings.HasPrefix(slug, "fundamentos-de-sql-") {
		t.Fatalf("curso creado inesperado: %v", course)
	}
	if created.Header.Get("Location") != "/api/v1/admin/courses/"+id {
		t.Fatalf("Location inesperado: %s", created.Header.Get("Location"))
	}

	// LMS-US-003: un borrador no aparece en el catálogo ni tiene detalle público.
	if contains(publicSlugs(t, srv.URL), slug) {
		t.Fatal("un borrador no debe aparecer en el catálogo")
	}
	if r := call(t, newClient(t), "GET", srv.URL+"/api/v1/catalog/courses/"+slug, nil); r.Status != http.StatusNotFound {
		t.Fatalf("detalle de borrador: se esperaba 404, se obtuvo %d", r.Status)
	}

	// LMS-US-006: no se publica incompleto.
	incomplete := call(t, admin, "POST", srv.URL+"/api/v1/admin/courses/"+id+"/publish", nil)
	if incomplete.Status != http.StatusUnprocessableEntity || errorCode(incomplete) != "publish_requirements_unmet" {
		t.Fatalf("publicar incompleto: %d %v", incomplete.Status, incomplete.Body)
	}

	// Edición antes de publicar, con bloqueo optimista.
	edit := map[string]any{
		"version": 1, "title": title, "summary": "Consultas básicas para analizar datos con SQL.",
		"description": strings.Repeat("Aprenderás a consultar, filtrar y agrupar datos. ", 2),
		"category":    "bases-de-datos", "level": "basico", "durationHours": 8,
		"objectives":    []string{"Escribir consultas SELECT", "Agrupar resultados"},
		"prerequisites": "Ninguno.",
	}
	updated := call(t, admin, "PUT", srv.URL+"/api/v1/admin/courses/"+id, edit)
	if updated.Status != http.StatusOK || courseOf(updated)["version"] != float64(2) {
		t.Fatalf("editar: %d %v", updated.Status, updated.Body)
	}
	if stale := call(t, admin, "PUT", srv.URL+"/api/v1/admin/courses/"+id, edit); stale.Status != http.StatusConflict || errorCode(stale) != "version_conflict" {
		t.Fatalf("edición repetida con versión antigua: %d %v", stale.Status, stale.Body)
	}

	// Publicar es idempotente y lleva el curso al catálogo.
	pub := call(t, admin, "POST", srv.URL+"/api/v1/admin/courses/"+id+"/publish", nil)
	if pub.Status != http.StatusOK || courseOf(pub)["status"] != "published" || courseOf(pub)["publishedAt"] == nil {
		t.Fatalf("publicar: %d %v", pub.Status, pub.Body)
	}
	again := call(t, admin, "POST", srv.URL+"/api/v1/admin/courses/"+id+"/publish", nil)
	if again.Status != http.StatusOK || courseOf(again)["version"] != courseOf(pub)["version"] {
		t.Fatalf("publicar dos veces: %d %v", again.Status, again.Body)
	}
	if !contains(publicSlugs(t, srv.URL), slug) {
		t.Fatal("el curso publicado debe aparecer en el catálogo")
	}
	detail := call(t, newClient(t), "GET", srv.URL+"/api/v1/catalog/courses/"+slug, nil)
	dc := courseOf(detail)
	if detail.Status != http.StatusOK || dc["title"] != title || len(dc["objectives"].([]any)) != 2 {
		t.Fatalf("detalle público: %d %v", detail.Status, detail.Body)
	}
	for _, internal := range []string{"id", "version", "status", "tenantId"} {
		if _, leaked := dc[internal]; leaked {
			t.Fatalf("el detalle público expone %q: %v", internal, dc)
		}
	}

	// Filtro por categoría.
	cat := call(t, newClient(t), "GET", srv.URL+"/api/v1/catalog/courses?category=programacion&limit=100", nil)
	for _, it := range cat.Body["items"].([]any) {
		if it.(map[string]any)["slug"] == slug {
			t.Fatal("el filtro de categoría no debe incluir cursos de otra categoría")
		}
	}

	// Retirar lo saca del catálogo sin borrarlo.
	wd := call(t, admin, "POST", srv.URL+"/api/v1/admin/courses/"+id+"/withdraw", nil)
	if wd.Status != http.StatusOK || courseOf(wd)["status"] != "withdrawn" {
		t.Fatalf("retirar: %d %v", wd.Status, wd.Body)
	}
	if contains(publicSlugs(t, srv.URL), slug) {
		t.Fatal("un curso retirado no debe aparecer en el catálogo")
	}
	if r := call(t, admin, "GET", srv.URL+"/api/v1/admin/courses/"+id, nil); r.Status != http.StatusOK || courseOf(r)["status"] != "withdrawn" {
		t.Fatalf("el curso retirado sigue disponible para administración: %d", r.Status)
	}
	list := call(t, admin, "GET", srv.URL+"/api/v1/admin/courses", nil)
	if list.Status != http.StatusOK || len(list.Body["items"].([]any)) == 0 {
		t.Fatalf("listado de administración: %d %v", list.Status, list.Body)
	}
}

func TestAdministrativeOperationsAreProtected(t *testing.T) {
	srv, _ := setup(t)
	admin := registerAdmin(t, srv.URL, "owner")
	created := call(t, admin, "POST", srv.URL+"/api/v1/admin/courses", map[string]any{
		"title": "Curso protegido", "summary": "Solo la administración puede gestionarlo.",
	})
	if created.Status != http.StatusCreated {
		t.Fatalf("owner crea curso: %d %v", created.Status, created.Body)
	}
	id := courseOf(created)["id"].(string)

	anon := newClient(t)
	student := registerStudent(t, srv.URL)
	draft := map[string]any{"title": "Intento", "summary": "Intento de modificación no autorizada."}
	edit := map[string]any{"title": "Intento", "summary": "Intento de modificación no autorizada.", "version": 1}
	routes := []struct {
		method, path string
		body         any
	}{
		{"GET", "/api/v1/admin/courses", nil},
		{"POST", "/api/v1/admin/courses", draft},
		{"GET", "/api/v1/admin/courses/" + id, nil},
		{"PUT", "/api/v1/admin/courses/" + id, edit},
		{"POST", "/api/v1/admin/courses/" + id + "/publish", nil},
		{"POST", "/api/v1/admin/courses/" + id + "/withdraw", nil},
	}
	for _, rt := range routes {
		if r := call(t, anon, rt.method, srv.URL+rt.path, rt.body); r.Status != http.StatusUnauthorized {
			t.Errorf("%s %s sin sesión: se esperaba 401, se obtuvo %d", rt.method, rt.path, r.Status)
		}
		// Un estudiante actúa en su tenant personal: no tiene permiso de
		// gestión, y el curso de la plataforma no es visible desde allí.
		if r := call(t, student, rt.method, srv.URL+rt.path, rt.body); r.Status != http.StatusForbidden {
			t.Errorf("%s %s como estudiante: se esperaba 403, se obtuvo %d %v", rt.method, rt.path, r.Status, r.Body)
		}
	}
	if r := call(t, admin, "GET", srv.URL+"/api/v1/admin/courses/no-es-uuid", nil); r.Status != http.StatusNotFound {
		t.Fatalf("id mal formado: se esperaba 404, se obtuvo %d", r.Status)
	}
	if r := call(t, admin, "POST", srv.URL+"/api/v1/admin/courses/"+id+"/withdraw", nil); r.Status != http.StatusConflict {
		t.Fatalf("retirar un borrador: se esperaba 409, se obtuvo %d", r.Status)
	}
	if r := call(t, newClient(t), "GET", srv.URL+"/api/v1/catalog/courses?limit=0", nil); r.Status != http.StatusBadRequest {
		t.Fatalf("paginación inválida: se esperaba 400, se obtuvo %d", r.Status)
	}
}

func TestGrantRoleRequiresRegisteredUser(t *testing.T) {
	_, pool := setup(t)
	log := logging.New(io.Discard, "error")
	if err := grantRole(context.Background(), pool, log, []string{"nadie-" + uniqueEmail(t), "owner"}); err == nil {
		t.Fatal("conceder rol a un correo no registrado debe fallar")
	}
	if err := grantRole(context.Background(), pool, log, []string{"x@example.com"}); err == nil {
		t.Fatal("faltan argumentos: debe fallar")
	}
}
