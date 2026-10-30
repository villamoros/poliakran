package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/villamoros/poliakran/backend/internal/platform/config"
	"github.com/villamoros/poliakran/backend/internal/platform/database"
	"github.com/villamoros/poliakran/backend/internal/platform/logging"
)

// Pruebas de API e integración con PostgreSQL. Requieren TEST_DATABASE_URL
// apuntando a una base de datos desechable; si no está definida se omiten.
func setup(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida: se omiten las pruebas de integración")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	log := logging.New(io.Discard, "error")
	if err := database.Migrate(ctx, pool, log, "up"); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Env: config.EnvTest, SessionTTL: time.Hour, CookieName: "pa_session",
		AllowedOrigins: []string{"http://localhost:3000"},
		AuthRateLimit:  1000, AuthRateLimitWindow: time.Minute,
	}
	h, err := NewServer(cfg, pool, log)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, pool
}

func newClient(t *testing.T) *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func uniqueEmail(t *testing.T) string {
	return fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
}

type apiResponse struct {
	Status int
	Body   map[string]any
	Header http.Header
}

func call(t *testing.T, c *http.Client, method, url string, body any) apiResponse {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out := apiResponse{Status: res.StatusCode, Header: res.Header}
	_ = json.NewDecoder(res.Body).Decode(&out.Body)
	return out
}

func errorCode(r apiResponse) string {
	e, _ := r.Body["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

func TestIdentityFlowEndToEnd(t *testing.T) {
	srv, pool := setup(t)
	c := newClient(t)
	email := uniqueEmail(t)
	password := "contraseña-segura"

	// LMS-US-001: registro abre sesión.
	res := call(t, c, "POST", srv.URL+"/api/v1/auth/register", map[string]string{
		"email": email, "displayName": "Ana Pérez", "password": password,
	})
	if res.Status != http.StatusCreated {
		t.Fatalf("registro: status %d, body %v", res.Status, res.Body)
	}
	setCookie := res.Header.Get("Set-Cookie")
	for _, attr := range []string{"pa_session=", "HttpOnly", "SameSite=Lax", "Path=/"} {
		if !strings.Contains(setCookie, attr) {
			t.Fatalf("cookie sin %q: %s", attr, setCookie)
		}
	}
	if strings.Contains(fmt.Sprint(res.Body), "password") {
		t.Fatalf("la respuesta expone la contraseña: %v", res.Body)
	}

	me := call(t, c, "GET", srv.URL+"/api/v1/auth/me", nil)
	if me.Status != http.StatusOK {
		t.Fatalf("/me tras registro: %d %v", me.Status, me.Body)
	}
	tenant := me.Body["tenant"].(map[string]any)
	if tenant["kind"] != "personal" {
		t.Fatalf("se esperaba tenant personal: %v", tenant)
	}
	if roles := fmt.Sprint(me.Body["roles"]); roles != "[student]" {
		t.Fatalf("roles inesperados: %s", roles)
	}

	// Duplicado (sin distinguir mayúsculas).
	dup := call(t, newClient(t), "POST", srv.URL+"/api/v1/auth/register", map[string]string{
		"email": strings.ToUpper(email), "displayName": "Otra", "password": password,
	})
	if dup.Status != http.StatusConflict || errorCode(dup) != "email_taken" {
		t.Fatalf("duplicado: %d %v", dup.Status, dup.Body)
	}

	// Contraseña y token se guardan solo como hash.
	var hash string
	if err := pool.QueryRow(context.Background(),
		`SELECT password_hash FROM users WHERE email_normalized = $1`, email).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash inesperado: %s", hash)
	}
	token := strings.TrimPrefix(strings.Split(setCookie, ";")[0], "pa_session=")
	sum := sha256.Sum256([]byte(token))
	var n int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM sessions WHERE token_hash = $1`, sum[:]).Scan(&n)
	if n != 1 {
		t.Fatalf("se esperaba la sesión guardada por hash, encontradas %d", n)
	}

	// LMS-US-002: inicio de sesión.
	c2 := newClient(t)
	bad := call(t, c2, "POST", srv.URL+"/api/v1/auth/login", map[string]string{"email": email, "password": "incorrecta-123"})
	unknown := call(t, c2, "POST", srv.URL+"/api/v1/auth/login", map[string]string{"email": "no-" + email, "password": password})
	if bad.Status != http.StatusUnauthorized || unknown.Status != http.StatusUnauthorized ||
		fmt.Sprint(bad.Body) != fmt.Sprint(unknown.Body) {
		t.Fatalf("los fallos de login deben ser indistinguibles: %v / %v", bad.Body, unknown.Body)
	}
	if call(t, c2, "GET", srv.URL+"/api/v1/auth/me", nil).Status != http.StatusUnauthorized {
		t.Fatal("/me sin sesión debe devolver 401")
	}
	ok := call(t, c2, "POST", srv.URL+"/api/v1/auth/login", map[string]string{"email": email, "password": password})
	if ok.Status != http.StatusOK {
		t.Fatalf("login: %d %v", ok.Status, ok.Body)
	}
	if call(t, c2, "GET", srv.URL+"/api/v1/auth/me", nil).Status != http.StatusOK {
		t.Fatal("/me tras login debe devolver 200")
	}

	// Cierre de sesión revoca en servidor y es repetible.
	if r := call(t, c2, "POST", srv.URL+"/api/v1/auth/logout", nil); r.Status != http.StatusNoContent {
		t.Fatalf("logout: %d", r.Status)
	}
	if r := call(t, c2, "POST", srv.URL+"/api/v1/auth/logout", nil); r.Status != http.StatusNoContent {
		t.Fatalf("logout repetido: %d", r.Status)
	}
	if call(t, c2, "GET", srv.URL+"/api/v1/auth/me", nil).Status != http.StatusUnauthorized {
		t.Fatal("/me tras logout debe devolver 401")
	}
	// La primera sesión (registro) sigue activa: el logout es por sesión.
	if call(t, c, "GET", srv.URL+"/api/v1/auth/me", nil).Status != http.StatusOK {
		t.Fatal("la sesión del registro no debe verse afectada")
	}
}

func TestRegisterValidationErrors(t *testing.T) {
	srv, _ := setup(t)
	res := call(t, newClient(t), "POST", srv.URL+"/api/v1/auth/register", map[string]string{
		"email": "no-es-correo", "displayName": "", "password": "corta",
	})
	if res.Status != http.StatusUnprocessableEntity || errorCode(res) != "validation_failed" {
		t.Fatalf("status %d body %v", res.Status, res.Body)
	}
	fields := res.Body["error"].(map[string]any)["fields"].(map[string]any)
	for _, f := range []string{"email", "displayName", "password"} {
		if _, ok := fields[f]; !ok {
			t.Fatalf("falta error del campo %s: %v", f, fields)
		}
	}
}

func TestRejectsMalformedAndCrossSiteRequests(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t)

	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/auth/login", strings.NewReader("email=a&password=b"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if res, _ := c.Do(req); res.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("formulario: se esperaba 415, se obtuvo %d", res.StatusCode)
	}

	req, _ = http.NewRequest("POST", srv.URL+"/api/v1/auth/login", strings.NewReader(`{"email":"a","password":"b"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	if res, _ := c.Do(req); res.StatusCode != http.StatusForbidden {
		t.Fatalf("origen ajeno: se esperaba 403, se obtuvo %d", res.StatusCode)
	}

	res := call(t, c, "POST", srv.URL+"/api/v1/auth/login", map[string]any{"email": "a", "password": "b", "admin": true})
	if res.Status != http.StatusBadRequest {
		t.Fatalf("campos desconocidos: se esperaba 400, se obtuvo %d", res.Status)
	}
}

func TestHealthEndpoints(t *testing.T) {
	srv, _ := setup(t)
	for _, path := range []string{"/healthz", "/readyz"} {
		res := call(t, newClient(t), "GET", srv.URL+path, nil)
		if res.Status != http.StatusOK {
			t.Fatalf("%s: %d", path, res.Status)
		}
		if res.Header.Get("X-Request-ID") == "" {
			t.Fatalf("%s sin X-Request-ID", path)
		}
	}
}

func TestMigrationsAreReversible(t *testing.T) {
	_, pool := setup(t)
	ctx := context.Background()
	log := logging.New(io.Discard, "error")
	if err := database.Migrate(ctx, pool, log, "reset"); err != nil {
		t.Fatalf("down: %v", err)
	}
	if err := database.Migrate(ctx, pool, log, "up"); err != nil {
		t.Fatalf("up: %v", err)
	}
}
