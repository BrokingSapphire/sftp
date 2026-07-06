// Package cache provides a small key/value cache abstraction with TTL, used to
// keep hot, read-mostly data (like a user's effective permissions) out of the
// database on every request. It ships an in-process implementation and an
// optional Redis-backed one for multi-instance deployments.
package cache

import (
	"context"
	"time"
)

// Cache is a minimal TTL key/value store over byte slices.
type Cache interface {
	// Get returns the value and true if present and unexpired.
	Get(ctx context.Context, key string) ([]byte, bool)
	// Set stores value under key for ttl.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration)
	// Delete removes a key.
	Delete(ctx context.Context, key string)
	// Close releases any resources.
	Close() error
}
