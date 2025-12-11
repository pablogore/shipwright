package executors

import (
	"context"
)

// CacheStrategy defines the interface for cache strategies.
// This will be moved to internal/cache in Phase 2.
type CacheStrategy interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, data []byte, ttl interface{}) error
}

// Executor defines the interface for pipeline execution backends.
// Different executors can run steps natively, in Docker, Kubernetes, etc.
type Executor interface {
	// Name returns the name of the executor.
	Name() string

	// CanExecute checks if this executor can run in the current environment.
	CanExecute(ctx context.Context) bool

	// ExecuteStep executes a single pipeline step.
	ExecuteStep(ctx context.Context, stepName string) error

	// SupportsParallel indicates whether this executor supports parallel step execution.
	SupportsParallel() bool

	// GetCacheStrategy returns the cache strategy used by this executor.
	GetCacheStrategy() CacheStrategy
}
