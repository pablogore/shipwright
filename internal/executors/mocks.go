package executors

import (
	"context"
)

// MockExecutor implements Executor interface for testing.
type MockExecutor struct {
	NameFunc             func() string
	CanExecuteFunc       func(ctx context.Context) bool
	ExecuteStepFunc      func(ctx context.Context, stepName string) error
	SupportsParallelFunc func() bool
	GetCacheStrategyFunc func() CacheStrategy
}

// Name returns the name of the executor.
func (m *MockExecutor) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock-executor"
}

// CanExecute checks if this executor can run in the current environment.
func (m *MockExecutor) CanExecute(ctx context.Context) bool {
	if m.CanExecuteFunc != nil {
		return m.CanExecuteFunc(ctx)
	}
	return true
}

// ExecuteStep executes a single pipeline step.
func (m *MockExecutor) ExecuteStep(ctx context.Context, stepName string) error {
	if m.ExecuteStepFunc != nil {
		return m.ExecuteStepFunc(ctx, stepName)
	}
	return nil
}

// SupportsParallel indicates whether this executor supports parallel step execution.
func (m *MockExecutor) SupportsParallel() bool {
	if m.SupportsParallelFunc != nil {
		return m.SupportsParallelFunc()
	}
	return true
}

// GetCacheStrategy returns the cache strategy used by this executor.
func (m *MockExecutor) GetCacheStrategy() CacheStrategy {
	if m.GetCacheStrategyFunc != nil {
		return m.GetCacheStrategyFunc()
	}
	return nil
}

// NewMockExecutor creates a new MockExecutor with default behavior.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{}
}

// NewMockExecutorWithError creates a MockExecutor that returns an error on ExecuteStep.
func NewMockExecutorWithError(err error) *MockExecutor {
	return &MockExecutor{
		ExecuteStepFunc: func(_ context.Context, _ string) error {
			return err
		},
	}
}
