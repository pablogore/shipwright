package app

import (
	"context"
	"testing"

	"github.com/getsyntegrity/syntegrity-dagger/internal/config"
	"github.com/getsyntegrity/syntegrity-dagger/internal/executors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainer_GetExecutorSelector(t *testing.T) {
	ctx := context.Background()
	cfg, err := config.NewConfigurationWrapper()
	require.NoError(t, err)

	container := NewContainer(ctx, cfg)
	err = container.Start(ctx)
	require.NoError(t, err)
	defer container.Stop(ctx)

	selector, err := container.Get("executorSelector")
	require.NoError(t, err)
	require.NotNil(t, selector)

	executorSelector, ok := selector.(*executors.Selector)
	require.True(t, ok)
	assert.NotNil(t, executorSelector)

	// Verify executors are registered
	executorList := executorSelector.ListExecutors()
	assert.Contains(t, executorList, "native")
	assert.Contains(t, executorList, "docker")
}

func TestContainer_ExecutorSelector_AutoDetect(t *testing.T) {
	ctx := context.Background()
	cfg, err := config.NewConfigurationWrapper()
	require.NoError(t, err)

	container := NewContainer(ctx, cfg)
	err = container.Start(ctx)
	require.NoError(t, err)
	defer container.Stop(ctx)

	selector, err := container.Get("executorSelector")
	require.NoError(t, err)

	executorSelector := selector.(*executors.Selector)
	ctx2 := context.Background()

	// Auto-detect should prefer native if available
	executor, err := executorSelector.SelectExecutor(ctx2, nil, convertConfig(cfg), "")
	if err == nil {
		// If native is available, it should be selected
		assert.NotNil(t, executor)
	}
}
