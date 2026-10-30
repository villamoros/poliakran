package security

import (
	"bytes"
	"strings"
	"testing"
)

var fastParams = Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

func TestArgon2RoundTrip(t *testing.T) {
	h := NewArgon2Hasher(fastParams)
	enc, err := h.Hash("contraseña-segura")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, "$argon2id$v=19$m=1024,t=1,p=1$") {
		t.Fatalf("formato inesperado: %s", enc)
	}
	if strings.Contains(enc, "contraseña-segura") {
		t.Fatal("el hash contiene la contraseña")
	}
	if ok, err := h.Verify("contraseña-segura", enc); err != nil || !ok {
		t.Fatalf("Verify correcta = %v, %v", ok, err)
	}
	if ok, _ := h.Verify("otra-contraseña", enc); ok {
		t.Fatal("Verify aceptó una contraseña incorrecta")
	}
	enc2, _ := h.Hash("contraseña-segura")
	if enc == enc2 {
		t.Fatal("dos hashes de la misma contraseña deben diferir (sal aleatoria)")
	}
}

func TestArgon2RejectsMalformedHash(t *testing.T) {
	h := NewArgon2Hasher(fastParams)
	for _, bad := range []string{"", "plain", "$bcrypt$x$y$z$w", "$argon2id$v=19$m=x$a$b"} {
		if _, err := h.Verify("x", bad); err == nil {
			t.Fatalf("se esperaba error para %q", bad)
		}
	}
}

func TestOpaqueTokens(t *testing.T) {
	var tk OpaqueTokens
	a, ha, err := tk.New()
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := tk.New()
	if a == b {
		t.Fatal("los tokens deben ser únicos")
	}
	if len(a) < 43 {
		t.Fatalf("token demasiado corto: %d", len(a))
	}
	if !bytes.Equal(ha, tk.Hash(a)) {
		t.Fatal("Hash debe ser determinista")
	}
}
