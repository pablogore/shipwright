package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
)

// StepRegistry implements the StepRegistry interface for managing pipeline steps.
type StepRegistry struct {
	handlers map[string]interfaces.StepHandler
	configs  map[string]interfaces.StepConfig
	mutex    sync.RWMutex
}

// NewStepRegistry creates a new step registry.
func NewStepRegistry() interfaces.StepRegistry {
	return &StepRegistry{
		handlers: make(map[string]interfaces.StepHandler),
		configs:  make(map[string]interfaces.StepConfig),
	}
}

// RegisterStep registers a new step handler.
func (sr *StepRegistry) RegisterStep(stepName string, handler interfaces.StepHandler) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if stepName == "" {
		return errors.New("step name cannot be empty")
	}

	if handler == nil {
		return errors.New("step handler cannot be nil")
	}

	// Validate that the handler can handle this step
	if !handler.CanHandle(stepName) {
		return fmt.Errorf("handler cannot handle step: %s", stepName)
	}

	// Get step configuration from handler
	config := handler.GetStepInfo(stepName)

	// Validate the configuration
	if err := handler.Validate(stepName, config); err != nil {
		return fmt.Errorf("invalid step configuration for %s: %w", stepName, err)
	}

	// Register the handler and config
	sr.handlers[stepName] = handler
	sr.configs[stepName] = config

	return nil
}

// GetStepHandler returns the handler for a specific step.
func (sr *StepRegistry) GetStepHandler(stepName string) (interfaces.StepHandler, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	handler, exists := sr.handlers[stepName]
	if !exists {
		return nil, fmt.Errorf("step handler not found: %s", stepName)
	}

	return handler, nil
}

// ListSteps returns a list of all registered step names.
func (sr *StepRegistry) ListSteps() []string {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	steps := make([]string, 0, len(sr.handlers))
	for stepName := range sr.handlers {
		steps = append(steps, stepName)
	}

	return steps
}

// GetStepConfig returns the configuration for a specific step.
func (sr *StepRegistry) GetStepConfig(stepName string) (interfaces.StepConfig, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	config, exists := sr.configs[stepName]
	if !exists {
		return interfaces.StepConfig{}, fmt.Errorf("step configuration not found: %s", stepName)
	}

	return config, nil
}

// ValidateStep validates a step configuration.
func (sr *StepRegistry) ValidateStep(stepName string) error {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	handler, exists := sr.handlers[stepName]
	if !exists {
		return fmt.Errorf("step handler not found: %s", stepName)
	}

	config, exists := sr.configs[stepName]
	if !exists {
		return fmt.Errorf("step configuration not found: %s", stepName)
	}

	return handler.Validate(stepName, config)
}

// ExecuteStep executes a specific step with its handler.
func (sr *StepRegistry) ExecuteStep(ctx context.Context, stepName string) error {
	handler, err := sr.GetStepHandler(stepName)
	if err != nil {
		return fmt.Errorf("failed to get step handler: %w", err)
	}

	config, err := sr.GetStepConfig(stepName)
	if err != nil {
		return fmt.Errorf("failed to get step configuration: %w", err)
	}

	// Create a context with timeout if specified
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	// Execute the step with retries
	var lastErr error
	for attempt := 0; attempt <= config.Retries; attempt++ {
		if attempt > 0 {
			// Wait before retry (exponential backoff)
			waitTime := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}
		}

		lastErr = handler.Execute(ctx, stepName, config)
		if lastErr == nil {
			return nil // Success
		}
	}

	return fmt.Errorf("step %s failed after %d attempts: %w", stepName, config.Retries+1, lastErr)
}

// GetStepInfo returns detailed information about a step.
func (sr *StepRegistry) GetStepInfo(stepName string) (map[string]any, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	_, exists := sr.handlers[stepName]
	if !exists {
		return nil, fmt.Errorf("step handler not found: %s", stepName)
	}

	config, exists := sr.configs[stepName]
	if !exists {
		return nil, fmt.Errorf("step configuration not found: %s", stepName)
	}

	info := map[string]any{
		"name":        config.Name,
		"description": config.Description,
		"required":    config.Required,
		"parallel":    config.Parallel,
		"timeout":     config.Timeout.String(),
		"retries":     config.Retries,
		"depends_on":  config.DependsOn,
		"conditions":  config.Conditions,
		"metadata":    config.Metadata,
	}

	return info, nil
}

// UnregisterStep removes a step from the registry.
func (sr *StepRegistry) UnregisterStep(stepName string) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if _, exists := sr.handlers[stepName]; !exists {
		return fmt.Errorf("step handler not found: %s", stepName)
	}

	delete(sr.handlers, stepName)
	delete(sr.configs, stepName)

	return nil
}

// ClearAllSteps removes all registered steps.
func (sr *StepRegistry) ClearAllSteps() {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	sr.handlers = make(map[string]interfaces.StepHandler)
	sr.configs = make(map[string]interfaces.StepConfig)
}

// GetStepCount returns the number of registered steps.
func (sr *StepRegistry) GetStepCount() int {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	return len(sr.handlers)
}

// GetRequiredSteps returns a list of all required steps.
func (sr *StepRegistry) GetRequiredSteps() []string {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	var required []string
	for stepName, config := range sr.configs {
		if config.Required {
			required = append(required, stepName)
		}
	}

	return required
}

// GetOptionalSteps returns a list of all optional steps.
func (sr *StepRegistry) GetOptionalSteps() []string {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	var optional []string
	for stepName, config := range sr.configs {
		if !config.Required {
			optional = append(optional, stepName)
		}
	}

	return optional
}

// GetParallelSteps returns a list of all steps that can run in parallel.
func (sr *StepRegistry) GetParallelSteps() []string {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	var parallel []string
	for stepName, config := range sr.configs {
		if config.Parallel {
			parallel = append(parallel, stepName)
		}
	}

	return parallel
}

// ValidateDependencies validates that all step dependencies are satisfied.
func (sr *StepRegistry) ValidateDependencies() error {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	for stepName, config := range sr.configs {
		for _, dep := range config.DependsOn {
			if _, exists := sr.handlers[dep]; !exists {
				return fmt.Errorf("step %s depends on non-existent step: %s", stepName, dep)
			}
		}
	}

	return nil
}

// GetExecutionOrder returns the steps in dependency order.
func (sr *StepRegistry) GetExecutionOrder() ([]string, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	// Validate dependencies first
	if err := sr.validateDependenciesInternal(); err != nil {
		return nil, err
	}

	// Topological sort to determine execution order
	visited := make(map[string]bool)
	tempVisited := make(map[string]bool)
	var result []string

	var visit func(string) error
	visit = func(stepName string) error {
		if tempVisited[stepName] {
			return fmt.Errorf("circular dependency detected involving step: %s", stepName)
		}
		if visited[stepName] {
			return nil
		}

		tempVisited[stepName] = true
		config := sr.configs[stepName]
		for _, dep := range config.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		tempVisited[stepName] = false
		visited[stepName] = true
		result = append(result, stepName)
		return nil
	}

	for stepName := range sr.handlers {
		if !visited[stepName] {
			if err := visit(stepName); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// validateDependenciesInternal is an internal method for dependency validation.
func (sr *StepRegistry) validateDependenciesInternal() error {
	for stepName, config := range sr.configs {
		for _, dep := range config.DependsOn {
			if _, exists := sr.handlers[dep]; !exists {
				return fmt.Errorf("step %s depends on non-existent step: %s", stepName, dep)
			}
		}
	}
	return nil
}
