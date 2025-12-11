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

// LocalFileCache implements Strategy using local file system.
// Suitable for development and local CI/CD runs.
type LocalFileCache struct {
	baseDir string
}

// NewLocalFileCache creates a new LocalFileCache.
//
// Parameters:
//   - baseDir: Base directory for cache files (defaults to .cache if empty).
//
// Returns:
//   - A new LocalFileCache instance.
func NewLocalFileCache(baseDir string) *LocalFileCache {
	if baseDir == "" {
		baseDir = ".cache"
	}
	return &LocalFileCache{
		baseDir: baseDir,
	}
}

// Get retrieves a value from cache by key.
func (c *LocalFileCache) Get(ctx context.Context, key string) ([]byte, error) {
	cachePath := c.getCachePath(key)

	// Check if file exists
	info, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to stat cache file: %w", err)
	}

	// Check if cache is expired (using file modification time)
	// For simplicity, we use file mtime as expiration indicator
	// In production, TTL would be stored in metadata
	if time.Since(info.ModTime()) > 24*time.Hour {
		// Cache expired, remove it
		_ = os.Remove(cachePath)
		return nil, fmt.Errorf("cache expired for key: %s", key)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	return data, nil
}

// Set stores a value in cache with the specified key and TTL.
func (c *LocalFileCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	cachePath := c.getCachePath(key)

	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write cache file
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Set modification time to now + TTL for expiration tracking
	expirationTime := time.Now().Add(ttl)
	if err := os.Chtimes(cachePath, expirationTime, expirationTime); err != nil {
		// Non-critical error, log but don't fail
		_ = err
	}

	return nil
}

// Invalidate removes a value from cache by key.
func (c *LocalFileCache) Invalidate(ctx context.Context, key string) error {
	cachePath := c.getCachePath(key)
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	return nil
}

// GetWithFallback retrieves a value from cache, or executes fallback function if not found.
func (c *LocalFileCache) GetWithFallback(ctx context.Context, key string, ttl time.Duration, fallback func() ([]byte, error)) ([]byte, error) {
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
func (c *LocalFileCache) WarmUp(ctx context.Context, keys []string, generator func(string) ([]byte, error)) error {
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

// getCachePath returns the file path for a cache key.
func (c *LocalFileCache) getCachePath(key string) string {
	// Hash key to create safe filename
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])
	return filepath.Join(c.baseDir, hashStr[:2], hashStr[2:4], hashStr)
}
