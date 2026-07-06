package cache

import (
	"context"
	"sync"
	"time"
)

type item struct {
	value   []byte
	expires time.Time
}

// Memory is a goroutine-safe, in-process TTL cache with a background janitor.
type Memory struct {
	mu    sync.RWMutex
	items map[string]item
	stop  chan struct{}
}

// NewMemory builds an in-process cache and starts its janitor.
func NewMemory() *Memory {
	m := &Memory{items: make(map[string]item), stop: make(chan struct{})}
	go m.janitor()
	return m
}

func (m *Memory) Get(_ context.Context, key string) ([]byte, bool) {
	m.mu.RLock()
	it, ok := m.items[key]
	m.mu.RUnlock()
	if !ok || time.Now().After(it.expires) {
		return nil, false
	}
	return it.value, true
}

func (m *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration) {
	m.mu.Lock()
	m.items[key] = item{value: value, expires: time.Now().Add(ttl)}
	m.mu.Unlock()
}

func (m *Memory) Delete(_ context.Context, key string) {
	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()
}

// Close stops the janitor.
func (m *Memory) Close() error {
	close(m.stop)
	return nil
}

func (m *Memory) janitor() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-t.C:
			now := time.Now()
			m.mu.Lock()
			for k, it := range m.items {
				if now.After(it.expires) {
					delete(m.items, k)
				}
			}
			m.mu.Unlock()
		}
	}
}
