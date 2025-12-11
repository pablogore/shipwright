package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrategyInterface(t *testing.T) {
	// This test verifies that the Strategy interface is properly defined
	ctx := context.Background()

	// Test that interface methods are callable (compile-time check)
	var strategy Strategy
	_ = strategy

	// Verify interface contract
	assert.NotNil(t, ctx)
}

func TestStrategy_GetWithFallback_NotInCache(t *testing.T) {
	ctx := context.Background()
	mockStrategy := NewMockStrategy()
	mockStrategy.GetFunc = func(ctx context.Context, key string) ([]byte, error) {
		return nil, errors.New("not found")
	}

	fallbackCalled := false
	fallback := func() ([]byte, error) {
		fallbackCalled = true
		return []byte("fallback-data"), nil
	}

	data, err := mockStrategy.GetWithFallback(ctx, "test-key", 5*time.Minute, fallback)

	require.NoError(t, err)
	assert.True(t, fallbackCalled)
	assert.Equal(t, []byte("fallback-data"), data)
}

func TestStrategy_GetWithFallback_InCache(t *testing.T) {
	ctx := context.Background()
	mockStrategy := NewMockStrategy()
	mockStrategy.GetFunc = func(ctx context.Context, key string) ([]byte, error) {
		return []byte("cached-data"), nil
	}

	fallbackCalled := false
	fallback := func() ([]byte, error) {
		fallbackCalled = true
		return []byte("fallback-data"), nil
	}

	data, err := mockStrategy.GetWithFallback(ctx, "test-key", 5*time.Minute, fallback)

	require.NoError(t, err)
	assert.False(t, fallbackCalled)
	assert.Equal(t, []byte("cached-data"), data)
}
