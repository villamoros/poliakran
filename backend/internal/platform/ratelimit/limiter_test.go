package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBlocksAfterLimitAndResets(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	l := New(2, time.Minute)
	l.now = func() time.Time { return now }

	if !l.Allow("ip") || !l.Allow("ip") {
		t.Fatal("los dos primeros intentos deben permitirse")
	}
	if l.Allow("ip") {
		t.Fatal("el tercer intento debe bloquearse")
	}
	if !l.Allow("otra-ip") {
		t.Fatal("cada clave tiene su propio contador")
	}

	now = now.Add(time.Minute + time.Second)
	if !l.Allow("ip") {
		t.Fatal("la ventana debe reiniciarse")
	}
}
