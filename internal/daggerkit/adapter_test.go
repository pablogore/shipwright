package daggerkit

import (
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
)

// TestNewDaggerAdapter_GetRealClient proves the client roundtrips through
// the adapter unchanged — the only part of DaggerAdapter that doesn't need
// a live Dagger engine to exercise.
func TestNewDaggerAdapter_GetRealClient(t *testing.T) {
	client := &dagger.Client{}
	adapter := NewDaggerAdapter(client)

	real, ok := adapter.(*DaggerAdapter)
	assert.True(t, ok)
	assert.Same(t, client, real.GetRealClient())
}

// TestNewDaggerDirectoryAdapter_GetRealDirectory proves the directory
// roundtrips through the adapter unchanged.
func TestNewDaggerDirectoryAdapter_GetRealDirectory(t *testing.T) {
	directory := &dagger.Directory{}
	adapter := NewDaggerDirectoryAdapter(directory)

	assert.Same(t, directory, adapter.GetRealDirectory())
}

// TestDaggerContainerAdapter_WithMountedDirectory_PanicsOnNonAdapterDirectory
// proves the defensive type-assertion guard fires before ever touching the
// real container — a mock DaggerDirectory reaching a real adapter means the
// test wiring, not the container, is wrong.
func TestDaggerContainerAdapter_WithMountedDirectory_PanicsOnNonAdapterDirectory(t *testing.T) {
	adapter := &DaggerContainerAdapter{container: &dagger.Container{}}

	assert.PanicsWithValue(t,
		"daggerkit: WithMountedDirectory called on DaggerContainerAdapter with a non-adapter DaggerDirectory",
		func() { adapter.WithMountedDirectory("/work", &MockDaggerDirectory{}) },
	)
}

// TestDaggerContainerAdapter_WithMountedCache_PanicsOnNonAdapterCacheVolume
// is WithMountedDirectory's counterpart for the cache-volume guard.
func TestDaggerContainerAdapter_WithMountedCache_PanicsOnNonAdapterCacheVolume(t *testing.T) {
	adapter := &DaggerContainerAdapter{container: &dagger.Container{}}

	assert.PanicsWithValue(t,
		"daggerkit: WithMountedCache called on DaggerContainerAdapter with a non-adapter DaggerCacheVolume",
		func() { adapter.WithMountedCache("/cache", &MockDaggerCacheVolume{}) },
	)
}

// TestDaggerContainerAdapter_WithDirectory_PanicsOnNonAdapterDirectory is
// WithMountedDirectory's counterpart for the WithDirectory guard.
func TestDaggerContainerAdapter_WithDirectory_PanicsOnNonAdapterDirectory(t *testing.T) {
	adapter := &DaggerContainerAdapter{container: &dagger.Container{}}

	assert.PanicsWithValue(t,
		"daggerkit: WithDirectory called on DaggerContainerAdapter with a non-adapter DaggerDirectory",
		func() { adapter.WithDirectory("/out", &MockDaggerDirectory{}) },
	)
}
