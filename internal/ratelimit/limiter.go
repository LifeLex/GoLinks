// Package ratelimit provides a small in-memory, token-bucket rate limiter keyed
// by an arbitrary string (e.g. a client IP). It is process-local — adequate for
// the single-instance deployment GoLinks targets. A distributed limiter would
// be needed for a multi-instance deployment (see ROADMAP.md).
package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// Limiter is a thread-safe token-bucket limiter. Each key gets its own bucket
// that starts full (burst tokens) and refills over time.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	burst   float64
	refill  float64 // tokens per second
	now     func() time.Time
}

// New creates a limiter that allows up to burst events immediately per key, then
// refills at refillPerMinute tokens per minute.
func New(burst, refillPerMinute int) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		burst:   float64(burst),
		refill:  float64(refillPerMinute) / 60.0,
		now:     time.Now,
	}
}

// Allow reports whether an event for key is permitted, consuming one token when
// it is.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastSeen: now}
		l.buckets[key] = b
	}

	// Refill proportionally to elapsed time, capped at burst.
	b.tokens = min(l.burst, b.tokens+now.Sub(b.lastSeen).Seconds()*l.refill)
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Cleanup drops buckets untouched for longer than maxIdle, bounding memory.
func (l *Limiter) Cleanup(maxIdle time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > maxIdle {
			delete(l.buckets, key)
		}
	}
}
