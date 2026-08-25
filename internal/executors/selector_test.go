package executors

import (
	"context"
	"testing"

	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSelector(t *testing.T) {
	selector := NewSelector()

	assert.NotNil(t, selector)
	assert.NotNil(t, selector.executors)
}

func TestSelector_RegisterExecutor(t *testing.T) {
	selector := NewSelector()
	mockExecutor := NewMockExecutor()

	selector.RegisterExecutor("test-executor", mockExecutor)

	executors := selector.ListExecutors()
	assert.Contains(t, executors, "test-executor")
}

func TestSelector_SelectExecutor_PreferredExecutor(t *testing.T) {
	ctx := context.Background()
	selector := NewSelector()
	mockExecutor := NewMockExecutor()
	mockExecutor.CanExecuteFunc = func(ctx context.Context) bool {
		return true
	}

	selector.RegisterExecutor("test-executor", mockExecutor)

	executor, err := selector.SelectExecutor(ctx, nil, pipelines.Config{}, "test-executor")

	require.NoError(t, err)
	assert.Equal(t, mockExecutor, executor)
}

func TestSelector_SelectExecutor_PreferredExecutorNotFound(t *testing.T) {
	ctx := context.Background()
	selector := NewSelector()

	_, err := selector.SelectExecutor(ctx, nil, pipelines.Config{}, "non-existent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "executor not found")
}

func TestSelector_SelectExecutor_AutoDetect_Native(t *testing.T) {
	ctx := context.Background()
	selector := NewSelector()
	nativeExecutor := NewNativeExecutor("1.25.5", ".")
	selector.RegisterExecutor("native", nativeExecutor)

	executor, err := selector.SelectExecutor(ctx, nil, pipelines.Config{}, "")

	require.NoError(t, err)
	assert.Equal(t, nativeExecutor, executor)
}

func TestSelector_ListExecutors(t *testing.T) {
	selector := NewSelector()
	mock1 := NewMockExecutor()
	mock2 := NewMockExecutor()

	selector.RegisterExecutor("executor1", mock1)
	selector.RegisterExecutor("executor2", mock2)

	executors := selector.ListExecutors()

	assert.Len(t, executors, 2)
	assert.Contains(t, executors, "executor1")
	assert.Contains(t, executors, "executor2")
}
