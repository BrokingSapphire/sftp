// Package ratelimit provides a small, dependency-free token-bucket limiter keyed
// by an arbitrary string (client IP, user id, …). It is safe for concurrent use
// and self-cleans idle buckets.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter is a keyed token-bucket rate limiter.
type Limiter struct {
	rate     float64 // tokens added per second
	burst    float64 // bucket capacity
	mu       sync.Mutex
	buckets  map[string]*bucket
	stop     chan struct{}
	nowFn    func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// New builds a limiter allowing `burst` requests immediately and refilling at
// `perSecond` tokens/second.
func New(perSecond, burst float64) *Limiter {
	l := &Limiter{
		rate:    perSecond,
		burst:   burst,
		buckets: make(map[string]*bucket),
		stop:    make(chan struct{}),
		nowFn:   time.Now,
	}
	go l.janitor()
	return l
}

// Allow reports whether a request for key may proceed, consuming one token.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.nowFn()
	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucket{tokens: l.burst - 1, last: now}
		return true
	}
	// Refill based on elapsed time, capped at burst.
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Close stops the background janitor.
func (l *Limiter) Close() { close(l.stop) }

func (l *Limiter) janitor() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-t.C:
			cutoff := l.nowFn().Add(-10 * time.Minute)
			l.mu.Lock()
			for k, b := range l.buckets {
				if b.last.Before(cutoff) {
					delete(l.buckets, k)
				}
			}
			l.mu.Unlock()
		}
	}
}
