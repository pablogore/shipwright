package cache

import (
	"context"
	"fmt"
	"time"

	"dagger.io/dagger"
)

// DaggerCacheManager manages cache for Dagger container layers.
// It uses Dagger's built-in cache volumes for efficient layer caching.
type DaggerCacheManager struct {
	client       *dagger.Client
	cacheVolumes map[string]*dagger.CacheVolume
}

// NewDaggerCacheManager creates a new DaggerCacheManager.
//
// Parameters:
//   - client: The Dagger client for cache operations.
//
// Returns:
//   - A new DaggerCacheManager instance.
func NewDaggerCacheManager(client *dagger.Client) *DaggerCacheManager {
	return &DaggerCacheManager{
		client:       client,
		cacheVolumes: make(map[string]*dagger.CacheVolume),
	}
}

// GetCacheVolume returns or creates a Dagger cache volume for the given key.
//
// Parameters:
//   - key: The cache key identifier.
//
// Returns:
//   - A Dagger CacheVolume for the key.
func (d *DaggerCacheManager) GetCacheVolume(key string) *dagger.CacheVolume {
	if volume, exists := d.cacheVolumes[key]; exists {
		return volume
	}

	// Create new cache volume
	volume := d.client.CacheVolume(key)
	d.cacheVolumes[key] = volume
	return volume
}

// GetCacheKeyForModules generates a cache key for Go modules.
func GetCacheKeyForModules(goVersion string) string {
	return fmt.Sprintf("go-modules-%s", goVersion)
}

// GetCacheKeyForBuild generates a cache key for build artifacts.
func GetCacheKeyForBuild(goVersion string, buildTags string) string {
	if buildTags != "" {
		return fmt.Sprintf("go-build-%s-%s", goVersion, buildTags)
	}
	return fmt.Sprintf("go-build-%s", goVersion)
}

// GetCacheKeyForDockerLayers generates a cache key for Docker image layers.
func GetCacheKeyForDockerLayers(imageName string, dockerfilePath string) string {
	return fmt.Sprintf("docker-layers-%s-%s", imageName, dockerfilePath)
}

// MountCacheVolume mounts a cache volume in a container at the specified path.
//
// Parameters:
//   - container: The Dagger container.
//   - cacheKey: The cache key.
//   - mountPath: The path where to mount the cache.
//
// Returns:
//   - The container with cache volume mounted.
func MountCacheVolume(container *dagger.Container, cacheKey string, mountPath string, manager *DaggerCacheManager) *dagger.Container {
	volume := manager.GetCacheVolume(cacheKey)
	return container.WithMountedCache(mountPath, volume)
}

// Strategy implementation for DaggerCacheManager
// Note: Dagger cache is handled differently than file-based cache,
// so this is a simplified adapter.

// Get is not directly applicable to Dagger cache volumes.
// Dagger handles cache automatically through mounted volumes.
func (d *DaggerCacheManager) Get(ctx context.Context, key string) ([]byte, error) {
	// Dagger cache is handled through mounted volumes, not direct get/set
	// This method is for interface compliance
	return nil, fmt.Errorf("Dagger cache does not support direct Get operations - use mounted volumes")
}

// Set is not directly applicable to Dagger cache volumes.
// Dagger handles cache automatically through mounted volumes.
func (d *DaggerCacheManager) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	// Dagger cache is handled through mounted volumes, not direct get/set
	// This method is for interface compliance
	return fmt.Errorf("Dagger cache does not support direct Set operations - use mounted volumes")
}

// Invalidate removes a cache volume (clears the cache).
func (d *DaggerCacheManager) Invalidate(ctx context.Context, key string) error {
	// Remove from map - Dagger will create new volume on next use
	delete(d.cacheVolumes, key)
	return nil
}

// GetWithFallback is not directly applicable to Dagger cache.
func (d *DaggerCacheManager) GetWithFallback(ctx context.Context, key string, ttl time.Duration, fallback func() ([]byte, error)) ([]byte, error) {
	// Dagger cache works differently - use fallback directly
	return fallback()
}

// WarmUp pre-warms Dagger cache by creating volumes.
func (d *DaggerCacheManager) WarmUp(ctx context.Context, keys []string, generator func(string) ([]byte, error)) error {
	// Pre-create cache volumes
	// Note: This requires a valid Dagger client
	if d.client == nil {
		return fmt.Errorf("Dagger client is required for cache operations")
	}
	for _, key := range keys {
		d.GetCacheVolume(key)
	}
	return nil
}
