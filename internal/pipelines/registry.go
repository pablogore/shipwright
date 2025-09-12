package pipelines

import (
	"fmt"

	"dagger.io/dagger"
)

// Package pipelines provides pipeline registry and registration logic.

// Registry centralizes the list of pipelines.
// It maintains a map of pipeline names to their corresponding factory functions.
type Registry struct {
	// pipelines is a map where the key is the pipeline name and the value is a factory function
	// that creates a Pipeline instance using a Dagger client and configuration.
	pipelines map[string]func(*dagger.Client, Config) Pipeline
}

// NewRegistry creates a new instance of the Registry.
//
// Returns:
//   - A pointer to a newly initialized Registry.
func NewRegistry() *Registry {
	return &Registry{pipelines: make(map[string]func(*dagger.Client, Config) Pipeline)}
}

// Register adds a new pipeline to the registry.
//
// Parameters:
//   - name: The name of the pipeline to register.
//   - factory: A factory function that creates a Pipeline instance using a Dagger client and configuration.
func (r *Registry) Register(name string, factory func(*dagger.Client, Config) Pipeline) {
	r.pipelines[name] = factory
}

// Get retrieves a pipeline by its name.
//
// Parameters:
//   - name: The name of the pipeline to retrieve.
//   - client: A pointer to the Dagger client used for pipeline operations.
//   - cfg: The configuration for the pipeline.
//
// Returns:
//   - A Pipeline instance if found.
//   - An error if the pipeline is not found in the registry.
func (r *Registry) Get(name string, client *dagger.Client, cfg Config) (Pipeline, error) {
	factory, ok := r.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", name)
	}
	return factory(client, cfg), nil
}

// List returns the names of all registered pipelines.
//
// Returns:
//   - A slice of strings containing the names of all registered pipelines.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.pipelines))
	for name := range r.pipelines {
		names = append(names, name)
	}
	return names
}
