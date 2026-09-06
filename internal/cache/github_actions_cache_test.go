package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitHubActionsCache(t *testing.T) {
	cache := NewGitHubActionsCache("/tmp/test-cache")

	assert.NotNil(t, cache)
	assert.Equal(t, "/tmp/test-cache", cache.cacheDir)
}

func TestNewGitHubActionsCache_DefaultDir(t *testing.T) {
	cache := NewGitHubActionsCache("")

	assert.NotNil(t, cache)
	assert.Equal(t, ".cache", cache.cacheDir)
}

func TestGitHubActionsCache_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewGitHubActionsCache(tmpDir)
	ctx := context.Background()

	key := "test-key"
	data := []byte("test-data")

	err := cache.Set(ctx, key, data, 1*time.Hour)
	require.NoError(t, err)

	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)
}

func TestGitHubActionsCache_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewGitHubActionsCache(tmpDir)
	ctx := context.Background()

	_, err := cache.Get(ctx, "non-existent-key")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache key not found")
}

func TestGitHubActionsCache_Invalidate(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewGitHubActionsCache(tmpDir)
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

func TestIsGitHubActions_NotInGitHub(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")

	result := IsGitHubActions()
	assert.False(t, result)
}

func TestIsGitHubActions_InGitHub(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")

	result := IsGitHubActions()
	assert.True(t, result)
}

func TestGetCacheKey(t *testing.T) {
	key := GetCacheKey("go-modules", "1.25.5")

	assert.Equal(t, "go-modules-1.25.5", key)
}

func TestGetCacheKey_WithAdditional(t *testing.T) {
	key := GetCacheKey("go-modules", "1.25.5", "linux", "amd64")

	assert.Equal(t, "go-modules-1.25.5-linux-amd64", key)
}
