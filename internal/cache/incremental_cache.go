package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// IncrementalCache implements Strategy for file-based incremental caching.
// It caches results based on file content hashes, invalidating only when files change.
type IncrementalCache struct {
	baseDir    string
	fileHashes map[string]string
}

// NewIncrementalCache creates a new IncrementalCache.
//
// Parameters:
//   - baseDir: Base directory for cache files (defaults to .cache/incremental if empty).
//
// Returns:
//   - A new IncrementalCache instance.
func NewIncrementalCache(baseDir string) *IncrementalCache {
	if baseDir == "" {
		baseDir = ".cache/incremental"
	}
	return &IncrementalCache{
		baseDir:    baseDir,
		fileHashes: make(map[string]string),
	}
}

// Get retrieves a value from cache by key.
func (c *IncrementalCache) Get(ctx context.Context, key string) ([]byte, error) {
	cachePath := c.getCachePath(key)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	return data, nil
}

// Set stores a value in cache with the specified key and TTL.
func (c *IncrementalCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	cachePath := c.getCachePath(key)

	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write cache file
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Invalidate removes a value from cache by key.
func (c *IncrementalCache) Invalidate(ctx context.Context, key string) error {
	cachePath := c.getCachePath(key)
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	return nil
}

// GetWithFallback retrieves a value from cache, or executes fallback function if not found.
func (c *IncrementalCache) GetWithFallback(ctx context.Context, key string, ttl time.Duration, fallback func() ([]byte, error)) ([]byte, error) {
	// Try to get from cache
	data, err := c.Get(ctx, key)
	if err == nil && data != nil {
		return data, nil
	}

	// Cache miss, execute fallback
	data, err = fallback()
	if err != nil {
		return nil, err
	}

	// Store in cache for next time
	if setErr := c.Set(ctx, key, data, ttl); setErr != nil {
		// Non-critical error, log but don't fail
		_ = setErr
	}

	return data, nil
}

// WarmUp pre-warms the cache with the specified keys.
func (c *IncrementalCache) WarmUp(ctx context.Context, keys []string, generator func(string) ([]byte, error)) error {
	for _, key := range keys {
		// Check if already cached
		_, err := c.Get(ctx, key)
		if err == nil {
			// Already cached, skip
			continue
		}

		// Generate and cache
		data, err := generator(key)
		if err != nil {
			return fmt.Errorf("failed to generate cache for key %s: %w", key, err)
		}

		if err := c.Set(ctx, key, data, 24*time.Hour); err != nil {
			return fmt.Errorf("failed to cache key %s: %w", key, err)
		}
	}

	return nil
}

// GetCacheKeyForFiles generates a cache key based on file paths and their content hashes.
//
// Parameters:
//   - files: List of file paths to include in the cache key.
//
// Returns:
//   - A cache key string based on file hashes.
func (c *IncrementalCache) GetCacheKeyForFiles(files []string) (string, error) {
	hasher := sha256.New()

	for _, file := range files {
		// Read file content
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", file, err)
		}

		// Include file path and content in hash
		hasher.Write([]byte(file))
		hasher.Write(data)
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash[:]), nil
}

// IsCacheValid checks if cache is still valid for the given files.
//
// Parameters:
//   - cacheKey: The cache key to check.
//   - files: List of file paths to validate against.
//
// Returns:
//   - True if cache is valid, false otherwise.
func (c *IncrementalCache) IsCacheValid(ctx context.Context, cacheKey string, files []string) (bool, error) {
	// Check if cache exists
	_, err := c.Get(ctx, cacheKey)
	if err != nil {
		return false, nil
	}

	// Generate current hash for files
	currentHash, err := c.GetCacheKeyForFiles(files)
	if err != nil {
		return false, err
	}

	// Compare with stored hash
	storedHash, exists := c.fileHashes[cacheKey]
	if !exists {
		// No stored hash, consider invalid
		return false, nil
	}

	return currentHash == storedHash, nil
}

// getCachePath returns the file path for a cache key.
func (c *IncrementalCache) getCachePath(key string) string {
	// Hash key to create safe filename
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])
	return filepath.Join(c.baseDir, hashStr[:2], hashStr[2:4], hashStr)
}

