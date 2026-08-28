package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GitHubActionsCache implements Strategy using GitHub Actions cache.
// This cache strategy integrates with GitHub Actions cache action.
// Note: Actual cache operations are handled by GitHub Actions cache action,
// this implementation provides the interface and helper methods.
type GitHubActionsCache struct {
	cacheDir string
}

// NewGitHubActionsCache creates a new GitHubActionsCache.
//
// Parameters:
//   - cacheDir: Directory where cache files are stored (defaults to .cache if empty).
//
// Returns:
//   - A new GitHubActionsCache instance.
func NewGitHubActionsCache(cacheDir string) *GitHubActionsCache {
	if cacheDir == "" {
		cacheDir = ".cache"
	}
	return &GitHubActionsCache{
		cacheDir: cacheDir,
	}
}

// Get retrieves a value from cache by key.
// In GitHub Actions, cache is restored by the cache action before this runs.
// This method reads from the local cache directory.
func (c *GitHubActionsCache) Get(_ context.Context, key string) ([]byte, error) {
	// In GitHub Actions, the cache action restores files to cacheDir
	// We read from the restored cache directory
	cachePath := filepath.Join(c.cacheDir, key)

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
// In GitHub Actions, files are saved to cacheDir and the cache action saves them.
func (c *GitHubActionsCache) Set(_ context.Context, key string, data []byte, _ time.Duration) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachePath := filepath.Join(c.cacheDir, key)

	// Write cache file
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// In GitHub Actions, the cache action will save files from cacheDir
	// The TTL is handled by GitHub Actions cache retention policy
	return nil
}

// Invalidate removes a value from cache by key.
func (c *GitHubActionsCache) Invalidate(_ context.Context, key string) error {
	cachePath := filepath.Join(c.cacheDir, key)
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	return nil
}

// GetWithFallback retrieves a value from cache, or executes fallback function if not found.
func (c *GitHubActionsCache) GetWithFallback(ctx context.Context, key string, ttl time.Duration, fallback func() ([]byte, error)) ([]byte, error) {
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
func (c *GitHubActionsCache) WarmUp(ctx context.Context, keys []string, generator func(string) ([]byte, error)) error {
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

// IsGitHubActions checks if we're running in GitHub Actions environment.
func IsGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") != ""
}

// GetCacheKey generates a cache key for GitHub Actions cache action.
// The key should be unique per cache entry and include version information.
func GetCacheKey(prefix string, version string, additional ...string) string {
	key := fmt.Sprintf("%s-%s", prefix, version)
	var keySb139 strings.Builder
	for _, add := range additional {
		keySb139.WriteString("-" + add)
	}
	key += keySb139.String()
	return key
}
