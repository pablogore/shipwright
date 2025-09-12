package app

import (
	"fmt"
	"testing"

	"dagger.io/dagger"
	"github.com/getsyntegrity/syntegrity-dagger/internal/config"
	"github.com/getsyntegrity/syntegrity-dagger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
	// Create a container that will fail to get logger
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Remove logger registration to cause error
	delete(container.once, "logger")

	app := NewApp(container)

	// Test start with logger error
	err := app.Start(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get logger")
}

func TestApp_Stop_WithLoggerError(t *testing.T) {
	// Create a container that will fail to get logger
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Remove logger registration to cause error
	delete(container.once, "logger")

	app := NewApp(container)

	// Test stop with logger error
	err := app.Stop(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get logger")
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

	// Remove logger registration to cause error
	delete(container.once, "logger")

	app := NewApp(container)

	// Test RunPipeline with logger error
	err := app.RunPipeline(t.Context(), "test-pipeline")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get logger")
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

	// Remove logger registration to cause error
	delete(container.once, "logger")

	app := NewApp(container)

	// Test RunPipelineStep with logger error
	err := app.RunPipelineStep(t.Context(), "test-pipeline", "test-step")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get logger")
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
	assert.Contains(t, pipelines, "go-kit")
	assert.Contains(t, pipelines, "docker-go")
	assert.Contains(t, pipelines, "infra")
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
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock pipeline
	mockPipeline := mocks.NewMockPipeline(ctrl)
	mockPipeline.EXPECT().Name().Return("test-pipeline").Times(1)

	// Create a test container that we can control
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Create mock pipeline registry
	mockRegistry := mocks.NewMockPipelineRegistry(ctrl)
	mockRegistry.EXPECT().Get("test-pipeline", gomock.Any(), gomock.Any()).Return(mockPipeline, nil).Times(1)

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
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock logger
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info("Running pipeline", "name", "test-pipeline").Times(1)
	mockLogger.EXPECT().Info("Executing pipeline step", "step", "build").Times(1)
	mockLogger.EXPECT().Info("Pipeline step completed", "step", "build").Times(1)
	mockLogger.EXPECT().Info("Executing pipeline step", "step", "test").Times(1)
	mockLogger.EXPECT().Info("Pipeline step completed", "step", "test").Times(1)
	mockLogger.EXPECT().Info("Pipeline completed successfully", "name", "test-pipeline").Times(1)

	// Create mock pipeline
	mockPipeline := mocks.NewMockPipeline(ctrl)
	mockPipeline.EXPECT().GetAvailableSteps().Return([]string{"build", "test"}).Times(1)
	mockPipeline.EXPECT().ExecuteStep(gomock.Any(), "build").Return(nil).Times(1)
	mockPipeline.EXPECT().ExecuteStep(gomock.Any(), "test").Return(nil).Times(1)

	// Create a test container that we can control
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Create mock pipeline registry
	mockRegistry := mocks.NewMockPipelineRegistry(ctrl)
	mockRegistry.EXPECT().Get("test-pipeline", gomock.Any(), gomock.Any()).Return(mockPipeline, nil).Times(1)

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
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock logger
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info("Running pipeline", "name", "test-pipeline").Times(1)
	mockLogger.EXPECT().Info("Executing pipeline step", "step", "build").Times(1)

	// Create mock pipeline
	mockPipeline := mocks.NewMockPipeline(ctrl)
	mockPipeline.EXPECT().GetAvailableSteps().Return([]string{"build", "test"}).Times(1)
	mockPipeline.EXPECT().ExecuteStep(gomock.Any(), "build").Return(fmt.Errorf("build failed")).Times(1)

	// Create a test container that we can control
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Create mock pipeline registry
	mockRegistry := mocks.NewMockPipelineRegistry(ctrl)
	mockRegistry.EXPECT().Get("test-pipeline", gomock.Any(), gomock.Any()).Return(mockPipeline, nil).Times(1)

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
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock logger
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info("Running pipeline step", "pipeline", "test-pipeline", "step", "build").Times(1)
	mockLogger.EXPECT().Info("Pipeline step completed successfully", "pipeline", "test-pipeline", "step", "build").Times(1)

	// Create mock pipeline
	mockPipeline := mocks.NewMockPipeline(ctrl)
	mockPipeline.EXPECT().ExecuteStep(gomock.Any(), "build").Return(nil).Times(1)

	// Create a test container that we can control
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Create mock pipeline registry
	mockRegistry := mocks.NewMockPipelineRegistry(ctrl)
	mockRegistry.EXPECT().Get("test-pipeline", gomock.Any(), gomock.Any()).Return(mockPipeline, nil).Times(1)

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
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock logger
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info("Running pipeline step", "pipeline", "test-pipeline", "step", "build").Times(1)

	// Create mock pipeline
	mockPipeline := mocks.NewMockPipeline(ctrl)
	mockPipeline.EXPECT().ExecuteStep(gomock.Any(), "build").Return(fmt.Errorf("build failed")).Times(1)

	// Create a test container that we can control
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Create mock pipeline registry
	mockRegistry := mocks.NewMockPipelineRegistry(ctrl)
	mockRegistry.EXPECT().Get("test-pipeline", gomock.Any(), gomock.Any()).Return(mockPipeline, nil).Times(1)

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
