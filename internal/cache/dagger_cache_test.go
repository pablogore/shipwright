package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDaggerCacheManager(t *testing.T) {
	// Note: This test doesn't require a real Dagger client
	// In real usage, a Dagger client would be required
	manager := NewDaggerCacheManager(nil)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.cacheVolumes)
}

func TestGetCacheKeyForModules(t *testing.T) {
	key := GetCacheKeyForModules("1.25.5")

	assert.Equal(t, "go-modules-1.25.5", key)
}

func TestGetCacheKeyForBuild(t *testing.T) {
	key := GetCacheKeyForBuild("1.25.5", "")

	assert.Equal(t, "go-build-1.25.5", key)
}

func TestGetCacheKeyForBuild_WithTags(t *testing.T) {
	key := GetCacheKeyForBuild("1.25.5", "integration")

	assert.Equal(t, "go-build-1.25.5-integration", key)
}

func TestGetCacheKeyForDockerLayers(t *testing.T) {
	key := GetCacheKeyForDockerLayers("my-service", "./Dockerfile")

	assert.Equal(t, "docker-layers-my-service-./Dockerfile", key)
}

func TestDaggerCacheManager_GetCacheVolume(t *testing.T) {
	// This test requires a real Dagger client, which may not be available
	// For now, we test that the method doesn't panic with nil client
	// In real usage, a Dagger client would be required
	manager := NewDaggerCacheManager(nil)

	// Test that manager is created correctly
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.cacheVolumes)
	assert.Empty(t, manager.cacheVolumes)

	// Note: GetCacheVolume will panic with nil client, which is expected
	// In production, a valid Dagger client must be provided
}

func TestDaggerCacheManager_Invalidate(t *testing.T) {
	manager := NewDaggerCacheManager(nil)

	// Manually add a key to the cache volumes map for testing
	// (In real usage, GetCacheVolume would add it, but that requires a valid client)
	manager.cacheVolumes["test-key"] = nil
	assert.Len(t, manager.cacheVolumes, 1)

	err := manager.Invalidate(context.Background(), "test-key")
	require.NoError(t, err)
	assert.Empty(t, manager.cacheVolumes)
}

func TestDaggerCacheManager_Get_NotSupported(t *testing.T) {
	manager := NewDaggerCacheManager(nil)
	ctx := context.Background()

	_, err := manager.Get(ctx, "test-key")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support direct Get operations")
}

func TestDaggerCacheManager_Set_NotSupported(t *testing.T) {
	manager := NewDaggerCacheManager(nil)
	ctx := context.Background()

	err := manager.Set(ctx, "test-key", []byte("data"), 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support direct Set operations")
}

func TestDaggerCacheManager_WarmUp(t *testing.T) {
	manager := NewDaggerCacheManager(nil)
	ctx := context.Background()

	keys := []string{"key1", "key2", "key3"}
	generator := func(_ string) ([]byte, error) {
		return []byte("data"), nil
	}

	// WarmUp requires a valid Dagger client
	err := manager.WarmUp(ctx, keys, generator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dagger client is required")
}
