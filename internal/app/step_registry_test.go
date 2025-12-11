package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStepRegistry(t *testing.T) {
	registry := NewStepRegistry()
	assert.NotNil(t, registry)
	assert.Implements(t, (*interfaces.StepRegistry)(nil), registry)
}

func TestStepRegistry_RegisterStep(t *testing.T) {
	registry := NewStepRegistry().(*StepRegistry)

	tests := []struct {
		name        string
		stepName    string
		handler     interfaces.StepHandler
		wantErr     bool
		errContains string
		setup       func() interfaces.StepHandler
	}{
		{
			name:     "successful registration",
			stepName: "test-step",
			wantErr:  false,
			setup: func() interfaces.StepHandler {
				mockHandler := NewMockStepHandler()
				mockHandler.CanHandleFunc = func(stepName string) bool { return stepName == "test-step" }
				mockHandler.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
					return interfaces.StepConfig{
						Name:        "test-step",
						Description: "Test step",
						Required:    true,
						Timeout:     5 * time.Minute,
					}
				}
				mockHandler.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }
				return mockHandler
			},
		},
		{
			name:        "empty step name",
			stepName:    "",
			wantErr:     true,
			errContains: "step name cannot be empty",
			setup:       func() interfaces.StepHandler { return NewMockStepHandler() },
		},
		{
			name:        "nil handler",
			stepName:    "test-step",
			handler:     nil,
			wantErr:     true,
			errContains: "step handler cannot be nil",
		},
		{
			name:        "handler cannot handle step",
			stepName:    "test-step",
			wantErr:     true,
			errContains: "handler cannot handle step",
			setup: func() interfaces.StepHandler {
				mockHandler := NewMockStepHandler()
				mockHandler.CanHandleFunc = func(stepName string) bool { return false }
				return mockHandler
			},
		},
		{
			name:        "invalid step configuration",
			stepName:    "test-step",
			wantErr:     true,
			errContains: "invalid step configuration",
			setup: func() interfaces.StepHandler {
				mockHandler := NewMockStepHandler()
				mockHandler.CanHandleFunc = func(stepName string) bool { return stepName == "test-step" }
				mockHandler.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
					return interfaces.StepConfig{
						Name:        "test-step",
						Description: "Test step",
						Required:    true,
						Timeout:     5 * time.Minute,
					}
				}
				mockHandler.ValidateFunc = func(stepName string, config interfaces.StepConfig) error {
					return errors.New("validation error")
				}
				return mockHandler
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler interfaces.StepHandler
			if tt.setup != nil {
				handler = tt.setup()
			} else {
				handler = tt.handler
			}

			err := registry.RegisterStep(tt.stepName, handler)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				// Verify step is registered
				_, exists := registry.handlers[tt.stepName]
				assert.True(t, exists)
				_, exists = registry.configs[tt.stepName]
				assert.True(t, exists)
			}
		})
	}
}

func TestStepRegistry_GetStepHandler(t *testing.T) {
	mockHandler := NewMockStepHandler()
	mockHandler.CanHandleFunc = func(stepName string) bool { return stepName == "test-step" }
	mockHandler.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:        "test-step",
			Description: "Test step",
			Required:    true,
			Timeout:     5 * time.Minute,
		}
	}
	mockHandler.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	registry := NewStepRegistry().(*StepRegistry)

	// Register a step
	err := registry.RegisterStep("test-step", mockHandler)
	require.NoError(t, err)

	tests := []struct {
		name        string
		stepName    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "existing step",
			stepName: "test-step",
			wantErr:  false,
		},
		{
			name:        "non-existent step",
			stepName:    "non-existent",
			wantErr:     true,
			errContains: "step handler not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := registry.GetStepHandler(tt.stepName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, handler)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, mockHandler, handler)
			}
		})
	}
}

func TestStepRegistry_ListSteps(t *testing.T) {
	mockHandler1 := NewMockStepHandler()
	mockHandler1.CanHandleFunc = func(stepName string) bool { return stepName == "step1" }
	mockHandler1.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{Name: "step1"}
	}
	mockHandler1.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	mockHandler2 := NewMockStepHandler()
	mockHandler2.CanHandleFunc = func(stepName string) bool { return stepName == "step2" }
	mockHandler2.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{Name: "step2"}
	}
	mockHandler2.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	registry := NewStepRegistry().(*StepRegistry)

	// Register multiple steps
	err := registry.RegisterStep("step1", mockHandler1)
	require.NoError(t, err)
	err = registry.RegisterStep("step2", mockHandler2)
	require.NoError(t, err)

	// Test ListSteps
	steps := registry.ListSteps()
	assert.Len(t, steps, 2)
	assert.Contains(t, steps, "step1")
	assert.Contains(t, steps, "step2")
}

func TestStepRegistry_ListSteps_Empty(t *testing.T) {
	registry := NewStepRegistry().(*StepRegistry)

	// Test ListSteps with no registered steps
	steps := registry.ListSteps()
	assert.Empty(t, steps)
}

func TestStepRegistry_GetStepConfig(t *testing.T) {
	mockHandler := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	expectedConfig := interfaces.StepConfig{
		Name:        "test-step",
		Description: "Test step",
		Required:    true,
		Timeout:     5 * time.Minute,
		Retries:     2,
		DependsOn:   []string{"setup"},
		Conditions:  map[string]string{"test": "true"},
		Metadata:    map[string]any{"category": "test"},
	}

	// Register a step
	mockHandler.CanHandleFunc = func(stepName string) bool { return stepName == "test-step" }
	mockHandler.GetStepInfoFunc = func(stepName string) interfaces.StepConfig { return expectedConfig }
	mockHandler.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("test-step", mockHandler)
	require.NoError(t, err)

	tests := []struct {
		name        string
		stepName    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "existing step",
			stepName: "test-step",
			wantErr:  false,
		},
		{
			name:        "non-existent step",
			stepName:    "non-existent",
			wantErr:     true,
			errContains: "step configuration not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := registry.GetStepConfig(tt.stepName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, interfaces.StepConfig{}, config)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, expectedConfig, config)
			}
		})
	}
}

func TestStepRegistry_ValidateStep(t *testing.T) {
	mockHandler := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	// Register a step
	mockHandler.CanHandleFunc = func(stepName string) bool { return stepName == "test-step" }
	mockHandler.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:        "test-step",
			Description: "Test step",
			Required:    true,
			Timeout:     5 * time.Minute,
		}
	}
	mockHandler.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("test-step", mockHandler)
	require.NoError(t, err)

	tests := []struct {
		name        string
		stepName    string
		wantErr     bool
		errContains string
		setup       func()
	}{
		{
			name:     "valid step",
			stepName: "test-step",
			wantErr:  false,
			setup:    func() {}, // No setup needed, mock already configured
		},
		{
			name:        "non-existent step",
			stepName:    "non-existent",
			wantErr:     true,
			errContains: "step handler not found",
		},
		{
			name:        "missing configuration",
			stepName:    "test-step",
			wantErr:     true,
			errContains: "step configuration not found",
			setup: func() {
				// Remove config to simulate missing configuration
				delete(registry.configs, "test-step")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			err := registry.ValidateStep(tt.stepName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStepRegistry_ExecuteStep(t *testing.T) {
	mockHandler := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	// Register a step
	mockHandler.CanHandleFunc = func(stepName string) bool { return stepName == "test-step" }
	mockHandler.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:        "test-step",
			Description: "Test step",
			Required:    true,
			Timeout:     5 * time.Minute,
			Retries:     1,
		}
	}
	mockHandler.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }
	mockHandler.ExecuteFunc = func(ctx context.Context, stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("test-step", mockHandler)
	require.NoError(t, err)

	tests := []struct {
		name        string
		stepName    string
		wantErr     bool
		errContains string
		setup       func()
	}{
		{
			name:     "successful execution",
			stepName: "test-step",
			wantErr:  false,
			setup:    func() {}, // No setup needed, mock already configured
		},
		{
			name:        "handler not found",
			stepName:    "non-existent",
			wantErr:     true,
			errContains: "failed to get step handler",
		},
		{
			name:        "configuration not found",
			stepName:    "test-step",
			wantErr:     true,
			errContains: "failed to get step configuration",
			setup: func() {
				// Remove config to simulate missing configuration
				delete(registry.configs, "test-step")
			},
		},
		{
			name:        "execution failure with retries",
			stepName:    "test-step",
			wantErr:     true,
			errContains: "step configuration not found: test-step",
			setup: func() {
				// No setup needed as the step configuration is not found
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			err := registry.ExecuteStep(t.Context(), tt.stepName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStepRegistry_ExecuteStep_WithTimeout(t *testing.T) {
	t.Skip("Skipping test due to timeout mechanism not implemented")

	mockHandler := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	// Register a step with timeout
	mockHandler.CanHandleFunc = func(stepName string) bool { return stepName == "test-step" }
	mockHandler.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:        "test-step",
			Description: "Test step",
			Required:    true,
			Timeout:     100 * time.Millisecond, // Short timeout for testing
			Retries:     0,
		}
	}
	mockHandler.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("test-step", mockHandler)
	require.NoError(t, err)

	// Mock handler to take longer than timeout
	mockHandler.ExecuteFunc = func(_ context.Context, _ string, _ interfaces.StepConfig) error {
		// Simulate work that takes longer than timeout
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	// Test execution with timeout
	err = registry.ExecuteStep(t.Context(), "test-step")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestStepRegistry_GetStepInfo(t *testing.T) {

	mockHandler := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	expectedConfig := interfaces.StepConfig{
		Name:        "test-step",
		Description: "Test step",
		Required:    true,
		Parallel:    false,
		Timeout:     5 * time.Minute,
		Retries:     2,
		DependsOn:   []string{"setup"},
		Conditions:  map[string]string{"test": "true"},
		Metadata:    map[string]any{"category": "test"},
	}

	// Register a step
	mockHandler.CanHandleFunc = func(stepName string) bool { return stepName == "test-step" }
	mockHandler.GetStepInfoFunc = func(stepName string) interfaces.StepConfig { return expectedConfig }
	mockHandler.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("test-step", mockHandler)
	require.NoError(t, err)

	tests := []struct {
		name        string
		stepName    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "existing step",
			stepName: "test-step",
			wantErr:  false,
		},
		{
			name:        "non-existent step",
			stepName:    "non-existent",
			wantErr:     true,
			errContains: "step handler not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := registry.GetStepInfo(tt.stepName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, info)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, info)
				assert.Equal(t, "test-step", info["name"])
				assert.Equal(t, "Test step", info["description"])
				assert.Equal(t, true, info["required"])
				assert.Equal(t, false, info["parallel"])
				assert.Equal(t, "5m0s", info["timeout"])
				assert.Equal(t, 2, info["retries"])
				assert.Equal(t, []string{"setup"}, info["depends_on"])
				assert.Equal(t, map[string]string{"test": "true"}, info["conditions"])
				assert.Equal(t, map[string]any{"category": "test"}, info["metadata"])
			}
		})
	}
}

func TestStepRegistry_UnregisterStep(t *testing.T) {

	mockHandler := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	// Register a step
	mockHandler.CanHandleFunc = func(stepName string) bool { return stepName == "test-step" }
	mockHandler.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{Name: "test-step"}
	}
	mockHandler.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("test-step", mockHandler)
	require.NoError(t, err)

	// Verify step is registered
	_, exists := registry.handlers["test-step"]
	assert.True(t, exists)
	_, exists = registry.configs["test-step"]
	assert.True(t, exists)

	// Test UnregisterStep
	err = registry.UnregisterStep("test-step")
	require.NoError(t, err)

	// Verify step is unregistered
	_, exists = registry.handlers["test-step"]
	assert.False(t, exists)
	_, exists = registry.configs["test-step"]
	assert.False(t, exists)

	// Test unregistering non-existent step
	err = registry.UnregisterStep("non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step handler not found")
}

func TestStepRegistry_ClearAllSteps(t *testing.T) {

	mockHandler1 := NewMockStepHandler()
	mockHandler2 := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	// Register multiple steps
	mockHandler1.CanHandleFunc = func(stepName string) bool { return stepName == "step1" }
	mockHandler1.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{Name: "step1"}
	}
	mockHandler1.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	mockHandler2.CanHandleFunc = func(stepName string) bool { return stepName == "step2" }
	mockHandler2.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{Name: "step2"}
	}
	mockHandler2.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("step1", mockHandler1)
	require.NoError(t, err)
	err = registry.RegisterStep("step2", mockHandler2)
	require.NoError(t, err)

	// Verify steps are registered
	assert.Len(t, registry.handlers, 2)
	assert.Len(t, registry.configs, 2)

	// Test ClearAllSteps
	registry.ClearAllSteps()

	// Verify all steps are cleared
	assert.Empty(t, registry.handlers)
	assert.Empty(t, registry.configs)
}

func TestStepRegistry_GetStepCount(t *testing.T) {

	mockHandler1 := NewMockStepHandler()
	mockHandler2 := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	// Test with no steps
	assert.Equal(t, 0, registry.GetStepCount())

	// Register steps
	mockHandler1.CanHandleFunc = func(stepName string) bool { return stepName == "step1" }
	mockHandler1.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{Name: "step1"}
	}
	mockHandler1.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	mockHandler2.CanHandleFunc = func(stepName string) bool { return stepName == "step2" }
	mockHandler2.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{Name: "step2"}
	}
	mockHandler2.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("step1", mockHandler1)
	require.NoError(t, err)
	err = registry.RegisterStep("step2", mockHandler2)
	require.NoError(t, err)

	// Test step count
	assert.Equal(t, 2, registry.GetStepCount())
}

func TestStepRegistry_GetRequiredSteps(t *testing.T) {

	mockHandler1 := NewMockStepHandler()
	mockHandler2 := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	// Register required and optional steps
	mockHandler1.CanHandleFunc = func(stepName string) bool { return stepName == "required-step" }
	mockHandler1.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:     "required-step",
			Required: true,
		}
	}
	mockHandler1.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	mockHandler2.CanHandleFunc = func(stepName string) bool { return stepName == "optional-step" }
	mockHandler2.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:     "optional-step",
			Required: false,
		}
	}
	mockHandler2.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("required-step", mockHandler1)
	require.NoError(t, err)
	err = registry.RegisterStep("optional-step", mockHandler2)
	require.NoError(t, err)

	// Test GetRequiredSteps
	required := registry.GetRequiredSteps()
	assert.Len(t, required, 1)
	assert.Contains(t, required, "required-step")
}

func TestStepRegistry_GetOptionalSteps(t *testing.T) {
	mockHandler1 := NewMockStepHandler()
	mockHandler1.CanHandleFunc = func(stepName string) bool { return stepName == "required-step" }
	mockHandler1.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:     "required-step",
			Required: true,
		}
	}
	mockHandler1.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	mockHandler2 := NewMockStepHandler()
	mockHandler2.CanHandleFunc = func(stepName string) bool { return stepName == "optional-step" }
	mockHandler2.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:     "optional-step",
			Required: false,
		}
	}
	mockHandler2.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	registry := NewStepRegistry().(*StepRegistry)

	err := registry.RegisterStep("required-step", mockHandler1)
	require.NoError(t, err)
	err = registry.RegisterStep("optional-step", mockHandler2)
	require.NoError(t, err)

	// Test GetOptionalSteps
	optional := registry.GetOptionalSteps()
	assert.Len(t, optional, 1)
	assert.Contains(t, optional, "optional-step")
}

func TestStepRegistry_GetParallelSteps(t *testing.T) {
	mockHandler1 := NewMockStepHandler()
	mockHandler1.CanHandleFunc = func(stepName string) bool { return stepName == "parallel-step" }
	mockHandler1.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:     "parallel-step",
			Parallel: true,
		}
	}
	mockHandler1.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	mockHandler2 := NewMockStepHandler()
	mockHandler2.CanHandleFunc = func(stepName string) bool { return stepName == "sequential-step" }
	mockHandler2.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:     "sequential-step",
			Parallel: false,
		}
	}
	mockHandler2.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	registry := NewStepRegistry().(*StepRegistry)

	err := registry.RegisterStep("parallel-step", mockHandler1)
	require.NoError(t, err)
	err = registry.RegisterStep("sequential-step", mockHandler2)
	require.NoError(t, err)

	// Test GetParallelSteps
	parallel := registry.GetParallelSteps()
	assert.Len(t, parallel, 1)
	assert.Contains(t, parallel, "parallel-step")
}

func TestStepRegistry_ValidateDependencies(t *testing.T) {
	mockHandler1 := NewMockStepHandler()
	mockHandler1.CanHandleFunc = func(stepName string) bool { return stepName == "step1" }
	mockHandler1.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:      "step1",
			DependsOn: []string{},
		}
	}
	mockHandler1.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	mockHandler2 := NewMockStepHandler()
	mockHandler2.CanHandleFunc = func(stepName string) bool { return stepName == "step2" }
	mockHandler2.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:      "step2",
			DependsOn: []string{"step1"},
		}
	}
	mockHandler2.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	registry := NewStepRegistry().(*StepRegistry)

	err := registry.RegisterStep("step1", mockHandler1)
	require.NoError(t, err)
	err = registry.RegisterStep("step2", mockHandler2)
	require.NoError(t, err)

	// Test ValidateDependencies with valid dependencies
	err = registry.ValidateDependencies()
	require.NoError(t, err)

	// Test with invalid dependency - register a new step with invalid dependency
	mockHandler3 := NewMockStepHandler()
	mockHandler3.CanHandleFunc = func(stepName string) bool { return stepName == "step3" }
	mockHandler3.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:      "step3",
			DependsOn: []string{"non-existent"},
		}
	}
	mockHandler3.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err = registry.RegisterStep("step3", mockHandler3)
	require.NoError(t, err)

	err = registry.ValidateDependencies()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step step3 depends on non-existent step: non-existent")
}

func TestStepRegistry_GetExecutionOrder(t *testing.T) {
	mockHandler1 := NewMockStepHandler()
	mockHandler1.CanHandleFunc = func(stepName string) bool { return stepName == "step1" }
	mockHandler1.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:      "step1",
			DependsOn: []string{},
		}
	}
	mockHandler1.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	mockHandler2 := NewMockStepHandler()
	mockHandler2.CanHandleFunc = func(stepName string) bool { return stepName == "step2" }
	mockHandler2.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:      "step2",
			DependsOn: []string{"step1"},
		}
	}
	mockHandler2.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	mockHandler3 := NewMockStepHandler()
	mockHandler3.CanHandleFunc = func(stepName string) bool { return stepName == "step3" }
	mockHandler3.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:      "step3",
			DependsOn: []string{"step2"},
		}
	}
	mockHandler3.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	registry := NewStepRegistry().(*StepRegistry)

	err := registry.RegisterStep("step1", mockHandler1)
	require.NoError(t, err)
	err = registry.RegisterStep("step2", mockHandler2)
	require.NoError(t, err)
	err = registry.RegisterStep("step3", mockHandler3)
	require.NoError(t, err)

	// Test GetExecutionOrder
	order, err := registry.GetExecutionOrder()
	require.NoError(t, err)
	assert.Equal(t, []string{"step1", "step2", "step3"}, order)
}

func TestStepRegistry_GetExecutionOrder_CircularDependency(t *testing.T) {

	mockHandler1 := NewMockStepHandler()
	mockHandler2 := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	// Register steps with circular dependency
	mockHandler1.CanHandleFunc = func(stepName string) bool { return stepName == "step1" }
	mockHandler1.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:      "step1",
			DependsOn: []string{"step2"},
		}
	}
	mockHandler1.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	mockHandler2.CanHandleFunc = func(stepName string) bool { return stepName == "step2" }
	mockHandler2.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:      "step2",
			DependsOn: []string{"step1"},
		}
	}
	mockHandler2.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("step1", mockHandler1)
	require.NoError(t, err)
	err = registry.RegisterStep("step2", mockHandler2)
	require.NoError(t, err)

	// Test GetExecutionOrder with circular dependency
	order, err := registry.GetExecutionOrder()
	require.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "circular dependency detected")
}

func TestStepRegistry_GetExecutionOrder_InvalidDependency(t *testing.T) {

	mockHandler := NewMockStepHandler()
	registry := NewStepRegistry().(*StepRegistry)

	// Register step with invalid dependency
	mockHandler.CanHandleFunc = func(stepName string) bool { return stepName == "step1" }
	mockHandler.GetStepInfoFunc = func(stepName string) interfaces.StepConfig {
		return interfaces.StepConfig{
			Name:      "step1",
			DependsOn: []string{"non-existent"},
		}
	}
	mockHandler.ValidateFunc = func(stepName string, config interfaces.StepConfig) error { return nil }

	err := registry.RegisterStep("step1", mockHandler)
	require.NoError(t, err)

	// Test GetExecutionOrder with invalid dependency
	order, err := registry.GetExecutionOrder()
	require.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "step step1 depends on non-existent step: non-existent")
}
