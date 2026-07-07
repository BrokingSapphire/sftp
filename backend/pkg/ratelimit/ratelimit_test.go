package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestBurstThenBlock(t *testing.T) {
	l := New(1, 3)
	defer l.Close()
	// Freeze time so refill doesn't interfere.
	now := time.Unix(0, 0)
	l.nowFn = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allow("k") {
			t.Fatalf("request %d should be allowed within burst", i)
		}
	}
	if l.Allow("k") {
		t.Fatal("4th request should be blocked (burst exhausted)")
	}
}

func TestRefill(t *testing.T) {
	l := New(10, 1) // 10/sec, burst 1
	defer l.Close()
	now := time.Unix(0, 0)
	l.nowFn = func() time.Time { return now }

	if !l.Allow("k") {
		t.Fatal("first allowed")
	}
	if l.Allow("k") {
		t.Fatal("immediately blocked")
	}
	now = now.Add(200 * time.Millisecond) // +2 tokens at 10/s
	if !l.Allow("k") {
		t.Fatal("should refill after 200ms")
	}
}

func TestKeysIndependent(t *testing.T) {
	l := New(1, 1)
	defer l.Close()
	if !l.Allow("a") || !l.Allow("b") {
		t.Fatal("different keys have independent buckets")
	}
}

func TestConcurrent(t *testing.T) {
	l := New(1000, 1000)
	defer l.Close()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); l.Allow("shared") }()
	}
	wg.Wait() // must not race/panic
}
