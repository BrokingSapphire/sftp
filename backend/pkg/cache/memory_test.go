package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemorySetGet(t *testing.T) {
	c := NewMemory()
	defer c.Close()
	ctx := context.Background()

	c.Set(ctx, "k", []byte("v"), time.Minute)
	if v, ok := c.Get(ctx, "k"); !ok || string(v) != "v" {
		t.Fatalf("get: %q ok=%v", v, ok)
	}

	c.Delete(ctx, "k")
	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatal("expected key deleted")
	}
}

func TestMemoryExpiry(t *testing.T) {
	c := NewMemory()
	defer c.Close()
	ctx := context.Background()

	c.Set(ctx, "k", []byte("v"), 20*time.Millisecond)
	if _, ok := c.Get(ctx, "k"); !ok {
		t.Fatal("expected present before expiry")
	}
	time.Sleep(35 * time.Millisecond)
	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatal("expected expired")
	}
}

func TestMemoryMiss(t *testing.T) {
	c := NewMemory()
	defer c.Close()
	if _, ok := c.Get(context.Background(), "nope"); ok {
		t.Fatal("expected miss")
	}
}
