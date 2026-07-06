package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis is a Redis/Valkey-backed cache for multi-instance deployments. Errors
// degrade gracefully to cache misses so a Redis blip never breaks a request.
type Redis struct {
	c *redis.Client
}

// NewRedis connects to Redis and verifies reachability.
func NewRedis(addr, password string, db int) (*Redis, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &Redis{c: client}, nil
}

func (r *Redis) Get(ctx context.Context, key string) ([]byte, bool) {
	b, err := r.c.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	return b, true
}

func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) {
	_ = r.c.Set(ctx, key, value, ttl).Err()
}

func (r *Redis) Delete(ctx context.Context, key string) {
	_ = r.c.Del(ctx, key).Err()
}

func (r *Redis) Close() error { return r.c.Close() }
