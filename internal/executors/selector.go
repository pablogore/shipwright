package executors

import (
	"context"
	"errors"
	"fmt"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/pipelines"
)

// Selector selects the appropriate executor based on environment and configuration.
type Selector struct {
	executors map[string]Executor
}

// NewSelector creates a new executor selector.
func NewSelector() *Selector {
	return &Selector{
		executors: make(map[string]Executor),
	}
}

// RegisterExecutor registers an executor with the selector.
func (s *Selector) RegisterExecutor(name string, executor Executor) {
	s.executors[name] = executor
}

// SelectExecutor selects the appropriate executor for the current environment.
//
// Parameters:
//   - ctx: The context for managing execution.
//   - client: The Dagger client (can be nil for native execution).
//   - config: The pipeline configuration.
//   - preferredExecutor: Preferred executor name (empty for auto-detection).
//
// Returns:
//   - The selected executor.
//   - An error if no suitable executor is found.
func (s *Selector) SelectExecutor(ctx context.Context, client *dagger.Client, config pipelines.Config, preferredExecutor string) (Executor, error) {
	// If preferred executor is specified, use it if available
	if preferredExecutor != "" {
		executor, ok := s.executors[preferredExecutor]
		if !ok {
			return nil, fmt.Errorf("executor not found: %s", preferredExecutor)
		}
		if executor.CanExecute(ctx) {
			return executor, nil
		}
		return nil, fmt.Errorf("executor %s cannot execute in current environment", preferredExecutor)
	}

	// Auto-detect: try native first (faster), then Docker
	// Check native executor
	if native, ok := s.executors["native"]; ok {
		if native.CanExecute(ctx) {
			return native, nil
		}
	}

	// Fallback to Docker executor
	if docker, ok := s.executors["docker"]; ok {
		if docker.CanExecute(ctx) {
			return docker, nil
		}
	}

	return nil, errors.New("no suitable executor available")
}

// ListExecutors returns a list of all registered executor names.
func (s *Selector) ListExecutors() []string {
	names := make([]string, 0, len(s.executors))
	for name := range s.executors {
		names = append(names, name)
	}
	return names
}
