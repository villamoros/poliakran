// Package ratelimit ofrece un limitador en memoria de ventana fija.
// Es suficiente para una sola instancia; con varias réplicas se sustituirá por
// uno compartido (por ejemplo Redis) cuando exista esa necesidad.
package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     func() time.Time
	buckets map[string]bucket
}

type bucket struct {
	count int
	reset time.Time
}

func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, now: time.Now, buckets: map[string]bucket{}}
}

// Allow registra un intento para key y devuelve false si superó el límite.
func (l *Limiter) Allow(key string) bool {
	if l.limit <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if len(l.buckets) > 10_000 {
		for k, b := range l.buckets {
			if now.After(b.reset) {
				delete(l.buckets, k)
			}
		}
	}
	b := l.buckets[key]
	if now.After(b.reset) {
		b = bucket{reset: now.Add(l.window)}
	}
	b.count++
	l.buckets[key] = b
	return b.count <= l.limit
}
