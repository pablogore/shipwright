package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIncrementalCache(t *testing.T) {
	cache := NewIncrementalCache("/tmp/test-cache")

	assert.NotNil(t, cache)
	assert.Equal(t, "/tmp/test-cache", cache.baseDir)
}

func TestNewIncrementalCache_DefaultDir(t *testing.T) {
	cache := NewIncrementalCache("")

	assert.NotNil(t, cache)
	assert.Equal(t, ".cache/incremental", cache.baseDir)
}

func TestIncrementalCache_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewIncrementalCache(tmpDir)
	ctx := context.Background()

	key := "test-key"
	data := []byte("test-data")

	err := cache.Set(ctx, key, data, 1*time.Hour)
	require.NoError(t, err)

	retrieved, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)
}

func TestIncrementalCache_GetCacheKeyForFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewIncrementalCache(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	key, err := cache.GetCacheKeyForFiles([]string{file1, file2})
	require.NoError(t, err)
	assert.NotEmpty(t, key)

	// Same files should produce same key
	key2, err := cache.GetCacheKeyForFiles([]string{file1, file2})
	require.NoError(t, err)
	assert.Equal(t, key, key2)
}

func TestIncrementalCache_GetCacheKeyForFiles_DifferentContent(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewIncrementalCache(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	key1, err := cache.GetCacheKeyForFiles([]string{file1, file2})
	require.NoError(t, err)

	// Change file content
	err = os.WriteFile(file1, []byte("different-content"), 0644)
	require.NoError(t, err)

	key2, err := cache.GetCacheKeyForFiles([]string{file1, file2})
	require.NoError(t, err)

	// Keys should be different
	assert.NotEqual(t, key1, key2)
}

func TestIncrementalCache_IsCacheValid_NoCache(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewIncrementalCache(tmpDir)
	ctx := context.Background()

	file1 := filepath.Join(tmpDir, "file1.txt")
	err := os.WriteFile(file1, []byte("content"), 0644)
	require.NoError(t, err)

	valid, err := cache.IsCacheValid(ctx, "non-existent-key", []string{file1})
	require.NoError(t, err)
	assert.False(t, valid)
}

