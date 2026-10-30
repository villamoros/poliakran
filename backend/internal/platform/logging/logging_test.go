package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestSensitiveValuesAreRedacted(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, "info")

	log.Info("intento", "password", "Secreta-123", "Token", "abc", "user_id", "u-1")

	out := buf.String()
	for _, leaked := range []string{"Secreta-123", `"abc"`} {
		if strings.Contains(out, leaked) {
			t.Fatalf("el log expone un valor sensible %q: %s", leaked, out)
		}
	}
	if !strings.Contains(out, `"user_id":"u-1"`) {
		t.Fatalf("se esperaba conservar campos no sensibles: %s", out)
	}
}
