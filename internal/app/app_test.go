package app

import (
	"context"
	"fmt"
	"testing"

	"dagger.io/dagger"
	"github.com/getsyntegrity/syntegrity-dagger/internal/config"
	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	app := NewApp(container)

	assert.NotNil(t, app)
	assert.Equal(t, container, app.container)
}

func TestApp_GetContainer(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)
	app := NewApp(container)

	retrievedContainer := app.GetContainer()
	assert.Equal(t, container, retrievedContainer)
}

func TestApp_RunPipeline(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)
	app := NewApp(container)

	// This will fail because we don't have a real pipeline executor, but we can test the structure
	err := app.RunPipeline(t.Context(), "test-pipeline")
	require.Error(t, err) // Expected to fail due to no real pipeline executor
}

func TestApp_RunPipelineStep(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)
	app := NewApp(container)

	// This will fail because we don't have a real pipeline executor, but we can test the structure
	err := app.RunPipelineStep(t.Context(), "test-pipeline", "test-step")
	require.Error(t, err) // Expected to fail due to no real pipeline executor
}

func TestApp_ListPipelines(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)
	app := NewApp(container)

	pipelines, err := app.ListPipelines()
	require.Error(t, err) // Expected to fail due to no real pipeline registry
	assert.Empty(t, pipelines)
}

func TestApp_GetPipelineInfo(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)
	app := NewApp(container)

	steps, err := app.GetPipelineInfo("test-pipeline")
	require.Error(t, err) // Expected to fail due to no real pipeline registry
	assert.Empty(t, steps)
}

func TestApp_WithNilContainer(t *testing.T) {
	app := NewApp(nil)

	assert.NotNil(t, app)
	assert.Nil(t, app.container)

	// These should panic with nil container
	assert.Panics(t, func() {
		_ = app.Start(t.Context())
	})

	assert.Panics(t, func() {
		_ = app.Stop(t.Context())
	})

	container := app.GetContainer()
	assert.Nil(t, container)

	// These should panic with nil container
	assert.Panics(t, func() {
		_, _ = app.ListPipelines()
	})

	assert.Panics(t, func() {
		_, _ = app.GetPipelineInfo("test")
	})
}

// Test global app functions
func TestGetContainer_NotInitialized(t *testing.T) {
	// Reset global state
	Reset()

	// Test getting container when not initialized
	assert.Panics(t, func() {
		GetContainer()
	})
}

func TestGetContainer_Initialized(t *testing.T) {
	// Reset global state
	Reset()

	cfg, _ := config.NewConfigurationWrapper()
	ctx := t.Context()

	// Initialize container
	err := Initialize(ctx, cfg)
	require.NoError(t, err)

	// Test getting container
	container := GetContainer()
	assert.NotNil(t, container)
}

func TestInitialize_Success(t *testing.T) {
	// Reset global state
	Reset()

	cfg, _ := config.NewConfigurationWrapper()
	ctx := t.Context()

	// Test successful initialization
	err := Initialize(ctx, cfg)
	require.NoError(t, err)

	// Verify container is set
	container := GetContainer()
	assert.NotNil(t, container)
}

func TestInitialize_MultipleCalls(t *testing.T) {
	// Reset global state
	Reset()

	cfg, _ := config.NewConfigurationWrapper()
	ctx := t.Context()

	// First initialization
	err := Initialize(ctx, cfg)
	require.NoError(t, err)

	container1 := GetContainer()

	// Second initialization should not change the container
	err = Initialize(ctx, cfg)
	require.NoError(t, err)

	container2 := GetContainer()
	assert.Equal(t, container1, container2)
}

func TestReset(t *testing.T) {
	// Reset global state
	Reset()

	cfg, _ := config.NewConfigurationWrapper()
	ctx := t.Context()

	// Initialize container
	err := Initialize(ctx, cfg)
	require.NoError(t, err)

	container := GetContainer()
	assert.NotNil(t, container)

	// Reset
	Reset()

	// Container should be nil after reset
	assert.Panics(t, func() {
		GetContainer()
	})
}

func TestApp_Start_Success(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Start the container to register all components
	err := container.Start(t.Context())
	require.NoError(t, err)

	app := NewApp(container)

	// Test successful start
	err = app.Start(t.Context())
	require.NoError(t, err)
}

func TestApp_Stop_Success(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Start the container to register all components
	err := container.Start(t.Context())
	require.NoError(t, err)

	app := NewApp(container)

	// Test successful stop
	err = app.Stop(t.Context())
	require.NoError(t, err)
}

func TestApp_Start_WithLoggerError(t *testing.T) {
	// Create a container
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	app := NewApp(container)

	// Test start - App uses logger.L() which is global, so it doesn't fail on logger errors
	// The container.Start() may fail for other reasons, but not logger-related
	err := app.Start(t.Context())
	// Start should succeed as it only calls container.Start() which registers components
	require.NoError(t, err)
}

func TestApp_Stop_WithLoggerError(t *testing.T) {
	// Create a container
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	app := NewApp(container)

	// Test stop - App uses logger.L() which is global, so it doesn't fail on logger errors
	// The container.Stop() may fail for other reasons, but not logger-related
	err := app.Stop(t.Context())
	// Stop should succeed as it only calls container.Stop() which cleans up components
	require.NoError(t, err)
}

// Test App pipeline methods
func TestApp_RunPipeline_Success(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Start the container to register all components
	err := container.Start(t.Context())
	require.NoError(t, err)

	app := NewApp(container)

	// Test RunPipeline - this will fail because we don't have a real pipeline
	err = app.RunPipeline(t.Context(), "test-pipeline")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get pipeline")
}

func TestApp_RunPipeline_LoggerError(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	app := NewApp(container)

	// Test RunPipeline - App uses logger.L() which is global, so it doesn't fail on logger errors
	// The actual error will be about pipeline not found, not logger
	err := app.RunPipeline(t.Context(), "test-pipeline")
	require.Error(t, err)
	// The error will be about pipeline registry or pipeline not found, not logger
	assert.Contains(t, err.Error(), "failed to get pipeline")
}

func TestApp_RunPipelineStep_Success(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Start the container to register all components
	err := container.Start(t.Context())
	require.NoError(t, err)

	app := NewApp(container)

	// Test RunPipelineStep - this will fail because we don't have a real pipeline
	err = app.RunPipelineStep(t.Context(), "test-pipeline", "test-step")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get pipeline")
}

func TestApp_RunPipelineStep_LoggerError(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	app := NewApp(container)

	// Test RunPipelineStep - App uses logger.L() which is global, so it doesn't fail on logger errors
	// The actual error will be about pipeline not found, not logger
	err := app.RunPipelineStep(t.Context(), "test-pipeline", "test-step")
	require.Error(t, err)
	// The error will be about pipeline registry or pipeline not found, not logger
	assert.Contains(t, err.Error(), "failed to get pipeline")
}

func TestApp_ListPipelines_Success(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Start the container to register all components
	err := container.Start(t.Context())
	require.NoError(t, err)

	app := NewApp(container)

	// Test ListPipelines - this should succeed and return the registered pipelines
	pipelines, err := app.ListPipelines()
	require.NoError(t, err)
	assert.NotEmpty(t, pipelines)
	assert.Contains(t, pipelines, "go-service")
	assert.Contains(t, pipelines, "infra")
	// docker-go functionality has been merged into go-service
}

func TestApp_ListPipelines_RegistryError(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Don't start the container so pipeline registry is not registered
	app := NewApp(container)

	// Test ListPipelines with registry error
	pipelines, err := app.ListPipelines()
	require.Error(t, err)
	assert.Empty(t, pipelines)
	assert.Contains(t, err.Error(), "failed to get pipeline registry")
}

func TestApp_GetPipelineInfo_Success(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Start the container to register all components
	err := container.Start(t.Context())
	require.NoError(t, err)

	app := NewApp(container)

	// Test GetPipelineInfo - this will fail because we don't have a real pipeline registry
	steps, err := app.GetPipelineInfo("test-pipeline")
	require.Error(t, err)
	assert.Empty(t, steps)
}

func TestApp_GetPipelineInfo_RegistryError(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Don't start the container so pipeline registry is not registered
	app := NewApp(container)

	// Test GetPipelineInfo with registry error
	steps, err := app.GetPipelineInfo("test-pipeline")
	require.Error(t, err)
	assert.Empty(t, steps)
	assert.Contains(t, err.Error(), "failed to get pipeline")
}

func TestApp_GetPipelineInfo_SuccessfulExecution(t *testing.T) {
	// Create mock pipeline
	mockPipeline := NewMockPipeline()
	mockPipeline.NameFunc = func() string { return "test-pipeline" }

	// Create a test container that we can control
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Create mock pipeline registry
	mockRegistry := NewMockPipelineRegistry()
	mockRegistry.GetFunc = func(name string, client *dagger.Client, cfg interfaces.Configuration) (interfaces.Pipeline, error) {
		if name == "test-pipeline" {
			return mockPipeline, nil
		}
		return nil, nil
	}

	// Create mock dagger client
	mockDaggerClient := &dagger.Client{}

	// Register mock components
	container.Register("pipelineRegistry", func() (any, error) {
		return mockRegistry, nil
	})
	container.Register("daggerClient", func() (any, error) {
		return mockDaggerClient, nil
	})

	// Create app with test container
	app := NewApp(container)

	// Test successful GetPipelineInfo execution
	info, err := app.GetPipelineInfo("test-pipeline")
	require.NoError(t, err)
	assert.NotEmpty(t, info)

	// Verify the returned info structure
	assert.Equal(t, "test-pipeline", info["name"])
	steps, ok := info["steps"].([]string)
	assert.True(t, ok)
	assert.Contains(t, steps, "setup")
	assert.Contains(t, steps, "build")
	assert.Contains(t, steps, "test")
	assert.Contains(t, steps, "tag")
	assert.Contains(t, steps, "package")
	assert.Contains(t, steps, "push")
	assert.Contains(t, steps, "lint")
	assert.Contains(t, steps, "security")
	assert.Contains(t, steps, "release")
}

// Test successful pipeline execution scenarios using a test container
func TestApp_RunPipeline_SuccessfulExecution(t *testing.T) {
	// Create mock logger
	mockLogger := NewMockLogger()
	// Note: With manual mocks, we don't verify exact call counts like gomock
	// The logger will be called during execution, but we don't assert on it

	// Create mock pipeline
	mockPipeline := NewMockPipeline()
	mockPipeline.GetAvailableStepsFunc = func() []string { return []string{"build", "test"} }
	mockPipeline.ExecuteStepFunc = func(ctx context.Context, stepName string) error { return nil }

	// Create a test container that we can control
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Create mock pipeline registry
	// GetPipeline is called once initially to get steps, then once per step
	// So for 2 steps (build, test), we expect 3 calls total: 1 initial + 2 steps
	callCount := 0
	mockRegistry := NewMockPipelineRegistry()
	mockRegistry.GetFunc = func(name string, client *dagger.Client, cfg interfaces.Configuration) (interfaces.Pipeline, error) {
		callCount++
		if name == "test-pipeline" {
			return mockPipeline, nil
		}
		return nil, nil
	}

	// Create mock dagger client
	mockDaggerClient := &dagger.Client{}

	// Register mock components
	container.Register("logger", func() (any, error) {
		return mockLogger, nil
	})
	container.Register("pipelineRegistry", func() (any, error) {
		return mockRegistry, nil
	})
	container.Register("daggerClient", func() (any, error) {
		return mockDaggerClient, nil
	})

	// Create app with test container
	app := NewApp(container)

	// Test successful pipeline execution
	err := app.RunPipeline(t.Context(), "test-pipeline")
	require.NoError(t, err)
}

func TestApp_RunPipeline_StepExecutionError(t *testing.T) {
	// Create mock logger
	mockLogger := NewMockLogger()
	// Note: With manual mocks, we don't verify exact call counts like gomock

	// Create mock pipeline
	mockPipeline := NewMockPipeline()
	mockPipeline.GetAvailableStepsFunc = func() []string { return []string{"build", "test"} }
	mockPipeline.ExecuteStepFunc = func(ctx context.Context, stepName string) error {
		if stepName == "build" {
			return fmt.Errorf("build failed")
		}
		return nil
	}

	// Create a test container that we can control
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Create mock pipeline registry
	// GetPipeline is called once initially to get steps, then once per step
	// So for 1 step (build), we expect 2 calls total: 1 initial + 1 step
	callCount := 0
	mockRegistry := NewMockPipelineRegistry()
	mockRegistry.GetFunc = func(name string, client *dagger.Client, cfg interfaces.Configuration) (interfaces.Pipeline, error) {
		callCount++
		if name == "test-pipeline" {
			return mockPipeline, nil
		}
		return nil, nil
	}

	// Create mock dagger client
	mockDaggerClient := &dagger.Client{}

	// Register mock components
	container.Register("logger", func() (any, error) {
		return mockLogger, nil
	})
	container.Register("pipelineRegistry", func() (any, error) {
		return mockRegistry, nil
	})
	container.Register("daggerClient", func() (any, error) {
		return mockDaggerClient, nil
	})

	// Create app with test container
	app := NewApp(container)

	// Test pipeline execution with step error
	err := app.RunPipeline(t.Context(), "test-pipeline")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline step build failed")
}

func TestApp_RunPipelineStep_SuccessfulExecution(t *testing.T) {
	// Create mock logger
	mockLogger := NewMockLogger()
	// Note: With manual mocks, we don't verify exact call counts like gomock

	// Create mock pipeline
	mockPipeline := NewMockPipeline()
	mockPipeline.ExecuteStepFunc = func(ctx context.Context, stepName string) error { return nil }

	// Create a test container that we can control
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Create mock pipeline registry
	mockRegistry := NewMockPipelineRegistry()
	mockRegistry.GetFunc = func(name string, client *dagger.Client, cfg interfaces.Configuration) (interfaces.Pipeline, error) {
		if name == "test-pipeline" {
			return mockPipeline, nil
		}
		return nil, nil
	}

	// Create mock dagger client
	mockDaggerClient := &dagger.Client{}

	// Register mock components
	container.Register("logger", func() (any, error) {
		return mockLogger, nil
	})
	container.Register("pipelineRegistry", func() (any, error) {
		return mockRegistry, nil
	})
	container.Register("daggerClient", func() (any, error) {
		return mockDaggerClient, nil
	})

	// Create app with test container
	app := NewApp(container)

	// Test successful step execution
	err := app.RunPipelineStep(t.Context(), "test-pipeline", "build")
	require.NoError(t, err)
}

func TestApp_RunPipelineStep_StepExecutionError(t *testing.T) {
	// Create mock logger
	mockLogger := NewMockLogger()
	// Note: With manual mocks, we don't verify exact call counts like gomock

	// Create mock pipeline
	mockPipeline := NewMockPipeline()
	mockPipeline.ExecuteStepFunc = func(ctx context.Context, stepName string) error {
		return fmt.Errorf("build failed")
	}

	// Create a test container that we can control
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Create mock pipeline registry
	mockRegistry := NewMockPipelineRegistry()
	mockRegistry.GetFunc = func(name string, client *dagger.Client, cfg interfaces.Configuration) (interfaces.Pipeline, error) {
		if name == "test-pipeline" {
			return mockPipeline, nil
		}
		return nil, nil
	}

	// Create mock dagger client
	mockDaggerClient := &dagger.Client{}

	// Register mock components
	container.Register("logger", func() (any, error) {
		return mockLogger, nil
	})
	container.Register("pipelineRegistry", func() (any, error) {
		return mockRegistry, nil
	})
	container.Register("daggerClient", func() (any, error) {
		return mockDaggerClient, nil
	})

	// Create app with test container
	app := NewApp(container)

	// Test step execution with error
	err := app.RunPipelineStep(t.Context(), "test-pipeline", "build")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline step build failed")
}
