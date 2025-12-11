package cache

import (
	"context"
	"time"
)

// Strategy defines the interface for cache strategies.
// Different strategies can use local files, distributed caches, CI/CD caches, etc.
type Strategy interface {
	// Get retrieves a value from cache by key.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in cache with the specified key and TTL.
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error

	// Invalidate removes a value from cache by key.
	Invalidate(ctx context.Context, key string) error

	// GetWithFallback retrieves a value from cache, or executes fallback function if not found.
	GetWithFallback(ctx context.Context, key string, ttl time.Duration, fallback func() ([]byte, error)) ([]byte, error)

	// WarmUp pre-warms the cache with the specified keys.
	WarmUp(ctx context.Context, keys []string, generator func(string) ([]byte, error)) error
}
