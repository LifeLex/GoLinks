package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_BurstThenDenyThenRefill(t *testing.T) {
	l := New(3, 60) // burst 3, refill 1 token/sec
	now := time.Now()
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allow("ip") {
			t.Fatalf("attempt %d denied, want allowed (within burst)", i+1)
		}
	}
	if l.Allow("ip") {
		t.Error("4th attempt allowed, want denied (burst exhausted)")
	}

	// Advance one second → one token refilled.
	now = now.Add(time.Second)
	if !l.Allow("ip") {
		t.Error("attempt after 1s refill denied, want allowed")
	}
	if l.Allow("ip") {
		t.Error("second attempt after 1s refill allowed, want denied")
	}
}

func TestLimiter_PerKeyIsolation(t *testing.T) {
	l := New(1, 0) // burst 1, no refill
	if !l.Allow("a") {
		t.Fatal("first attempt for 'a' denied")
	}
	if l.Allow("a") {
		t.Fatal("second attempt for 'a' allowed, want denied")
	}
	if !l.Allow("b") {
		t.Fatal("attempt for 'b' denied — keys must be independent")
	}
}

func TestLimiter_Cleanup(t *testing.T) {
	l := New(1, 0)
	now := time.Now()
	l.now = func() time.Time { return now }

	l.Allow("a") // creates and drains the bucket for "a"
	now = now.Add(2 * time.Hour)
	l.Cleanup(time.Hour)

	// The idle bucket was removed, so "a" starts fresh and is allowed again.
	if !l.Allow("a") {
		t.Error("after cleanup, fresh bucket should allow")
	}
}
