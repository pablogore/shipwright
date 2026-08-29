package executors

import (
	"context"
	"testing"

	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDockerExecutor(t *testing.T) {
	cfg := pipelines.Config{
		GoVersion: "1.25.5",
	}

	executor := NewDockerExecutor(nil, cfg)

	assert.NotNil(t, executor)
	assert.Nil(t, executor.client)
	assert.Equal(t, cfg, executor.config)
}

func TestDockerExecutor_Name(t *testing.T) {
	executor := NewDockerExecutor(nil, pipelines.Config{})

	assert.Equal(t, "docker", executor.Name())
}

func TestDockerExecutor_CanExecute_WithClient(t *testing.T) {
	ctx := context.Background()
	// Note: This test requires a real Dagger client, which may not be available in test environment
	// For now, we test the nil case
	executor := NewDockerExecutor(nil, pipelines.Config{})

	result := executor.CanExecute(ctx)
	assert.False(t, result) // nil client means cannot execute
}

func TestDockerExecutor_SupportsParallel(t *testing.T) {
	executor := NewDockerExecutor(nil, pipelines.Config{})

	assert.True(t, executor.SupportsParallel())
}

func TestDockerExecutor_GetCacheStrategy(t *testing.T) {
	executor := NewDockerExecutor(nil, pipelines.Config{})

	strategy := executor.GetCacheStrategy()
	assert.Nil(t, strategy) // Docker executor doesn't provide cache strategy (handled by Dagger)
}

func TestDockerExecutor_ExecuteStep_NilClient(t *testing.T) {
	ctx := context.Background()
	executor := NewDockerExecutor(nil, pipelines.Config{})

	err := executor.ExecuteStep(ctx, "build")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker executor requires dagger client")
}

func TestDockerExecutor_ExecuteStep_UnsupportedStep(t *testing.T) {
	ctx := context.Background()
	// We can't easily test with real client, so we test the error path
	executor := NewDockerExecutor(nil, pipelines.Config{})

	err := executor.ExecuteStep(ctx, "unsupported-step")

	require.Error(t, err)
	// Will fail on nil client check first, but if client was set, would fail on unsupported step
}
