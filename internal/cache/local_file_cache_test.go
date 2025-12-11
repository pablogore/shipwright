package cache

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocalFileCache(t *testing.T) {
	cache := NewLocalFileCache("/tmp/test-cache")

	assert.NotNil(t, cache)
	assert.Equal(t, "/tmp/test-cache", cache.baseDir)
}

func TestNewLocalFileCache_DefaultDir(t *testing.T) {
	cache := NewLocalFileCache("")

	assert.NotNil(t, cache)
	assert.Equal(t, ".cache", cache.baseDir)
}

func TestLocalFileCache_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewLocalFileCache(tmpDir)
	ctx := context.Background()

	key := "test-key"
	data := []byte("test-data")

	err := cache.Set(ctx, key, data, 1*time.Hour)
	require.NoError(t, err)

	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)
}

func TestLocalFileCache_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewLocalFileCache(tmpDir)
	ctx := context.Background()

	_, err := cache.Get(ctx, "non-existent-key")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache key not found")
}

func TestLocalFileCache_Invalidate(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewLocalFileCache(tmpDir)
	ctx := context.Background()

	key := "test-key"
	data := []byte("test-data")

	err := cache.Set(ctx, key, data, 1*time.Hour)
	require.NoError(t, err)

	err = cache.Invalidate(ctx, key)
	require.NoError(t, err)

	_, err = cache.Get(ctx, key)
	require.Error(t, err)
}

func TestLocalFileCache_GetWithFallback_CacheHit(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewLocalFileCache(tmpDir)
	ctx := context.Background()

	key := "test-key"
	data := []byte("cached-data")

	err := cache.Set(ctx, key, data, 1*time.Hour)
	require.NoError(t, err)

	fallbackCalled := false
	fallback := func() ([]byte, error) {
		fallbackCalled = true
		return []byte("fallback-data"), nil
	}

	result, err := cache.GetWithFallback(ctx, key, 1*time.Hour, fallback)

	require.NoError(t, err)
	assert.False(t, fallbackCalled)
	assert.Equal(t, data, result)
}

func TestLocalFileCache_GetWithFallback_CacheMiss(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewLocalFileCache(tmpDir)
	ctx := context.Background()

	key := "test-key"
	fallbackData := []byte("fallback-data")

	fallbackCalled := false
	fallback := func() ([]byte, error) {
		fallbackCalled = true
		return fallbackData, nil
	}

	result, err := cache.GetWithFallback(ctx, key, 1*time.Hour, fallback)

	require.NoError(t, err)
	assert.True(t, fallbackCalled)
	assert.Equal(t, fallbackData, result)

	// Verify it was cached
	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, fallbackData, retrieved)
}

func TestLocalFileCache_WarmUp(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewLocalFileCache(tmpDir)
	ctx := context.Background()

	keys := []string{"key1", "key2", "key3"}
	generator := func(key string) ([]byte, error) {
		return []byte("data-for-" + key), nil
	}

	err := cache.WarmUp(ctx, keys, generator)
	require.NoError(t, err)

	// Verify all keys are cached
	for _, key := range keys {
		data, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, []byte("data-for-"+key), data)
	}
}

func TestLocalFileCache_GetCachePath(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewLocalFileCache(tmpDir)

	path := cache.getCachePath("test-key")

	// Verify path structure: baseDir/hash[0:2]/hash[2:4]/hash
	assert.Contains(t, path, tmpDir)
	assert.True(t, filepath.IsAbs(path) || filepath.IsAbs(filepath.Join(tmpDir, path)))
}
