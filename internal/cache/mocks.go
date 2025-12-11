package cache

import (
	"context"
	"time"
)

// MockStrategy implements Strategy interface for testing.
type MockStrategy struct {
	GetFunc             func(ctx context.Context, key string) ([]byte, error)
	SetFunc             func(ctx context.Context, key string, data []byte, ttl time.Duration) error
	InvalidateFunc      func(ctx context.Context, key string) error
	GetWithFallbackFunc func(ctx context.Context, key string, ttl time.Duration, fallback func() ([]byte, error)) ([]byte, error)
	WarmUpFunc          func(ctx context.Context, keys []string, generator func(string) ([]byte, error)) error
}

// Get retrieves a value from cache by key.
func (m *MockStrategy) Get(ctx context.Context, key string) ([]byte, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return nil, nil
}

// Set stores a value in cache with the specified key and TTL.
func (m *MockStrategy) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, data, ttl)
	}
	return nil
}

// Invalidate removes a value from cache by key.
func (m *MockStrategy) Invalidate(ctx context.Context, key string) error {
	if m.InvalidateFunc != nil {
		return m.InvalidateFunc(ctx, key)
	}
	return nil
}

// GetWithFallback retrieves a value from cache, or executes fallback function if not found.
func (m *MockStrategy) GetWithFallback(ctx context.Context, key string, ttl time.Duration, fallback func() ([]byte, error)) ([]byte, error) {
	if m.GetWithFallbackFunc != nil {
		return m.GetWithFallbackFunc(ctx, key, ttl, fallback)
	}

	// Default implementation: try cache first, then fallback
	data, err := m.Get(ctx, key)
	if err == nil && data != nil {
		return data, nil
	}

	return fallback()
}

// WarmUp pre-warms the cache with the specified keys.
func (m *MockStrategy) WarmUp(ctx context.Context, keys []string, generator func(string) ([]byte, error)) error {
	if m.WarmUpFunc != nil {
		return m.WarmUpFunc(ctx, keys, generator)
	}
	return nil
}

// NewMockStrategy creates a new MockStrategy with default behavior.
func NewMockStrategy() *MockStrategy {
	return &MockStrategy{}
}
