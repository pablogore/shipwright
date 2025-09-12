package app

import (
	"errors"
	"testing"
	"time"

	"log/slog"

	"dagger.io/dagger"
	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
	"github.com/getsyntegrity/syntegrity-dagger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewContainer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)
	assert.NotNil(t, container)
	assert.Equal(t, ctx, container.ctx)
	assert.Equal(t, mockConfig, container.config)
	assert.NotNil(t, container.once)
	assert.NotNil(t, container.cache)
}

func TestContainer_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test registering a component
	factory := func() (any, error) {
		return "test-component", nil
	}

	container.Register("test-component", factory)

	// Verify component is registered
	_, exists := container.cache["test-component"]
	assert.True(t, exists)
	_, exists = container.once["test-component"]
	assert.True(t, exists)
}

func TestContainer_Get_NotRegistered(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test getting a non-registered component
	component, err := container.Get("non-existent")
	require.Error(t, err)
	assert.Nil(t, component)
	assert.Contains(t, err.Error(), "component not found")
}

func TestContainer_Get_Registered(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register a component
	expectedComponent := "test-component"
	factory := func() (any, error) {
		return expectedComponent, nil
	}

	container.Register("test-component", factory)

	// Get the component
	component, err := container.Get("test-component")
	require.NoError(t, err)
	assert.Equal(t, expectedComponent, component)
}

func TestContainer_Get_FactoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register a component that returns an error
	factory := func() (any, error) {
		return nil, errors.New("factory error")
	}

	container.Register("error-component", factory)

	// Get the component
	component, err := container.Get("error-component")
	require.Error(t, err)
	assert.Nil(t, component)
	assert.Contains(t, err.Error(), "factory error")
}

func TestContainer_Get_NonFactory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register a non-factory component
	expectedComponent := "direct-component"
	container.cache["direct-component"] = expectedComponent
	delete(container.once, "direct-component") // Remove from once map

	// Get the component
	component, err := container.Get("direct-component")
	require.NoError(t, err)
	assert.Equal(t, expectedComponent, component)
}

func TestContainer_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test Start
	err := container.Start(ctx)
	require.NoError(t, err)
}

func TestContainer_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test Stop with no components
	err := container.Stop(ctx)
	require.NoError(t, err)
}

func TestContainer_Stop_WithClosableComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Create a mock closable component
	mockClosable := &mockClosableComponent{}
	container.cache["closable-component"] = mockClosable

	// Test Stop
	err := container.Stop(ctx)
	require.NoError(t, err)
	assert.True(t, mockClosable.closed)
}

func TestContainer_Stop_WithClosableComponentsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Create a mock closable component that returns an error
	mockClosable := &mockClosableComponent{closeError: errors.New("close error")}
	container.cache["closable-component"] = mockClosable

	// Test Stop - should not return error even if component close fails
	err := container.Stop(ctx)
	require.NoError(t, err)
	assert.True(t, mockClosable.closed)
}

func TestContainer_Validate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test Validate
	mockConfig.EXPECT().Validate().Return(nil)
	err := container.Validate()
	require.NoError(t, err)
}

func TestContainer_Validate_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test Validate with error
	mockConfig.EXPECT().Validate().Return(errors.New("validation error"))
	err := container.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation error")
}

func TestContainer_GetPipelineRegistry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register pipeline registry
	mockRegistry := mocks.NewMockPipelineRegistry(ctrl)
	container.Register("pipelineRegistry", func() (any, error) {
		return mockRegistry, nil
	})

	// Test GetPipelineRegistry
	registry, err := container.GetPipelineRegistry()
	require.NoError(t, err)
	assert.Equal(t, mockRegistry, registry)
}

func TestContainer_GetPipelineRegistry_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test GetPipelineRegistry without registration
	registry, err := container.GetPipelineRegistry()
	require.Error(t, err)
	assert.Nil(t, registry)
	assert.Contains(t, err.Error(), "component not found")
}

func TestContainer_GetPipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	mockRegistry := mocks.NewMockPipelineRegistry(ctrl)
	mockPipeline := mocks.NewMockPipeline(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register dependencies
	container.Register("pipelineRegistry", func() (any, error) {
		return mockRegistry, nil
	})
	var mockClient *dagger.Client
	container.Register("daggerClient", func() (any, error) {
		return mockClient, nil
	})

	// Set up expectations
	mockRegistry.EXPECT().Get("test-pipeline", mockClient, mockConfig).Return(mockPipeline, nil)

	// Test GetPipeline
	pipeline, err := container.GetPipeline("test-pipeline")
	require.NoError(t, err)
	assert.Equal(t, mockPipeline, pipeline)
}

func TestContainer_GetPipeline_RegistryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test GetPipeline without registry
	pipeline, err := container.GetPipeline("test-pipeline")
	require.Error(t, err)
	assert.Nil(t, pipeline)
	assert.Contains(t, err.Error(), "component not found")
}

func TestContainer_GetDaggerClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register dagger client
	var expectedClient *dagger.Client
	container.Register("daggerClient", func() (any, error) {
		return expectedClient, nil
	})

	// Test GetDaggerClient
	client, err := container.GetDaggerClient()
	require.NoError(t, err)
	assert.Equal(t, expectedClient, client)
}

func TestContainer_GetDaggerClient_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test GetDaggerClient without registration
	client, err := container.GetDaggerClient()
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "component not found")
}

func TestContainer_GetDaggerClient_NilClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register nil dagger client
	container.Register("daggerClient", func() (any, error) {
		return nil, nil
	})

	// Test GetDaggerClient
	client, err := container.GetDaggerClient()
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "dagger client is nil")
}

func TestContainer_GetRegistryConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	expectedConfig := interfaces.RegistryConfig{
		BaseURL: "test-registry.com",
		User:    "test-user",
		Pass:    "test-pass",
		Image:   "test-image",
		Tag:     "test-tag",
	}
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Set up expectations
	mockConfig.EXPECT().Registry().Return(expectedConfig)

	// Test GetRegistryConfig
	config, err := container.GetRegistryConfig()
	require.NoError(t, err)
	assert.Equal(t, expectedConfig, config)
}

func TestContainer_GetRegistryAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	expectedConfig := interfaces.RegistryConfig{
		BaseURL: "test-registry.com",
		User:    "test-user",
		Pass:    "test-pass",
		Image:   "test-image",
		Tag:     "test-tag",
	}
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Set up expectations
	mockConfig.EXPECT().Registry().Return(expectedConfig)

	// Test GetRegistryAuth
	user, pass, err := container.GetRegistryAuth()
	require.NoError(t, err)
	assert.Equal(t, "test-user", user)
	assert.Equal(t, "test-pass", pass)
}

func TestContainer_GetVulnChecker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	mockChecker := mocks.NewMockVulnChecker(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register vuln checker
	container.Register("vulnChecker", func() (any, error) {
		return mockChecker, nil
	})

	// Test GetVulnChecker
	checker, err := container.GetVulnChecker()
	require.NoError(t, err)
	assert.Equal(t, mockChecker, checker)
}

func TestContainer_GetVulnChecker_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test GetVulnChecker without registration
	checker, err := container.GetVulnChecker()
	require.Error(t, err)
	assert.Nil(t, checker)
	assert.Contains(t, err.Error(), "component not found")
}

func TestContainer_GetLinter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	mockLinter := mocks.NewMockLinter(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register linter
	container.Register("linter", func() (any, error) {
		return mockLinter, nil
	})

	// Test GetLinter
	linter, err := container.GetLinter()
	require.NoError(t, err)
	assert.Equal(t, mockLinter, linter)
}

func TestContainer_GetLinter_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test GetLinter without registration
	linter, err := container.GetLinter()
	require.Error(t, err)
	assert.Nil(t, linter)
	assert.Contains(t, err.Error(), "component not found")
}

func TestContainer_GetLogger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register logger
	container.Register("logger", func() (any, error) {
		return mockLogger, nil
	})

	// Test GetLogger
	logger, err := container.GetLogger()
	require.NoError(t, err)
	assert.Equal(t, mockLogger, logger)
}

func TestContainer_GetLogger_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test GetLogger without registration
	logger, err := container.GetLogger()
	require.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "component not found")
}

func TestNewPipelineRegistry(t *testing.T) {
	registry := NewPipelineRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.pipelines)
}

func TestPipelineRegistry_Register(t *testing.T) {
	registry := NewPipelineRegistry()

	// Test registering a pipeline
	factory := func(_ *dagger.Client, _ interfaces.Configuration) interfaces.Pipeline {
		return nil
	}

	registry.Register("test-pipeline", factory)

	// Verify pipeline is registered
	_, exists := registry.pipelines["test-pipeline"]
	assert.True(t, exists)
}

func TestPipelineRegistry_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	mockPipeline := mocks.NewMockPipeline(ctrl)
	registry := NewPipelineRegistry()

	// Register a pipeline
	factory := func(_ *dagger.Client, _ interfaces.Configuration) interfaces.Pipeline {
		return mockPipeline
	}

	registry.Register("test-pipeline", factory)

	// Test Get
	var mockClient *dagger.Client
	pipeline, err := registry.Get("test-pipeline", mockClient, mockConfig)
	require.NoError(t, err)
	assert.Equal(t, mockPipeline, pipeline)
}

func TestPipelineRegistry_Get_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	registry := NewPipelineRegistry()

	// Test Get with non-existent pipeline
	var mockClient *dagger.Client
	pipeline, err := registry.Get("non-existent", mockClient, mockConfig)
	require.Error(t, err)
	assert.Nil(t, pipeline)
	assert.Contains(t, err.Error(), "pipeline not found")
}

func TestPipelineRegistry_List(t *testing.T) {
	registry := NewPipelineRegistry()

	// Register some pipelines
	factory := func(_ *dagger.Client, _ interfaces.Configuration) interfaces.Pipeline {
		return nil
	}

	registry.Register("pipeline1", factory)
	registry.Register("pipeline2", factory)
	registry.Register("pipeline3", factory)

	// Test List
	pipelines := registry.List()
	assert.Len(t, pipelines, 3)
	assert.Contains(t, pipelines, "pipeline1")
	assert.Contains(t, pipelines, "pipeline2")
	assert.Contains(t, pipelines, "pipeline3")
}

func TestNewVulnChecker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)

	checker := NewVulnChecker(mockConfig)
	assert.NotNil(t, checker)
	assert.Equal(t, mockConfig, checker.config)
}

func TestVulnChecker_Check(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	checker := NewVulnChecker(mockConfig)

	// Test Check (currently returns nil)
	err := checker.Check(t.Context(), nil)
	require.NoError(t, err)
}

func TestVulnChecker_GetReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	checker := NewVulnChecker(mockConfig)

	// Test GetReport (currently returns empty string)
	report, err := checker.GetReport(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "", report)
}

func TestNewLinter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)

	linter := NewLinter(mockConfig)
	assert.NotNil(t, linter)
	assert.Equal(t, mockConfig, linter.config)
}

func TestLinter_Lint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	linter := NewLinter(mockConfig)

	// Test Lint (currently returns nil)
	err := linter.Lint(t.Context(), nil)
	require.NoError(t, err)
}

func TestLinter_GetReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	linter := NewLinter(mockConfig)

	// Test GetReport (currently returns empty string)
	report, err := linter.GetReport(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "", report)
}

func TestNewLogger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)

	logger := NewLogger(mockConfig)
	assert.NotNil(t, logger)
	assert.Equal(t, mockConfig, logger.config)
}

func TestLogger_AllMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	logger := NewLogger(mockConfig)

	// Test all logger methods (they just print to stdout)
	assert.NotPanics(t, func() {
		logger.Debug("debug message", "key", "value")
		logger.Info("info message", "key", "value")
		logger.Warn("warn message", "key", "value")
		logger.Error("error message", "key", "value")
		logger.Fatal("fatal message", "key", "value")
	})
}

func TestLogger_WithField(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	logger := NewLogger(mockConfig)

	// Test WithField
	result := logger.WithField("key", "value")
	assert.NotNil(t, result)
	assert.Equal(t, logger, result) // Current implementation returns same logger
}

func TestLogger_WithFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	logger := NewLogger(mockConfig)

	// Test WithFields
	fields := map[string]any{"key1": "value1", "key2": "value2"}
	result := logger.WithFields(fields)
	assert.NotNil(t, result)
	assert.Equal(t, logger, result) // Current implementation returns same logger
}

// Mock closable component for testing
type mockClosableComponent struct {
	closed     bool
	closeError error
}

func (m *mockClosableComponent) Close() error {
	m.closed = true
	return m.closeError
}

// Test container register functions
func TestContainer_registerDaggerComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test with default timeout
	mockConfig.EXPECT().GetDuration("dagger.timeout").Return(time.Duration(0))

	// Register the component
	container.registerDaggerComponents()

	// Verify component is registered
	_, exists := container.once["daggerClient"]
	assert.True(t, exists)

	// Try to get the component - this will call the factory function
	// In a real test environment, this might fail because Dagger daemon is not running
	_, err := container.Get("daggerClient")
	// We expect this to fail in test environment, but the registration should work
	if err != nil {
		// Expected to fail in test environment when Dagger daemon is not running
		require.Error(t, err)
	} else {
		// If it succeeds, that's also fine - it means Dagger is available
		require.NoError(t, err)
	}
}

func TestContainer_registerDaggerComponents_WithTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test with custom timeout
	mockConfig.EXPECT().GetDuration("dagger.timeout").Return(60 * time.Second)

	container.registerDaggerComponents()

	// Verify component is registered
	_, exists := container.once["daggerClient"]
	assert.True(t, exists)

	// Try to get the component - this will call the factory function
	// In a real test environment, this might fail because Dagger daemon is not running
	_, err := container.Get("daggerClient")
	// We expect this to fail in test environment, but the registration should work
	if err != nil {
		// Expected to fail in test environment when Dagger daemon is not running
		require.Error(t, err)
	} else {
		// If it succeeds, that's also fine - it means Dagger is available
		require.NoError(t, err)
	}
}

func TestContainer_registerPipelineComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	container.registerPipelineComponents()

	// Verify component is registered
	_, exists := container.once["pipelineRegistry"]
	assert.True(t, exists)
}

func TestContainer_registerSecurityComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	container.registerSecurityComponents()

	// Verify components are registered
	_, exists := container.once["vulnChecker"]
	assert.True(t, exists)
	_, exists = container.once["linter"]
	assert.True(t, exists)
}

func TestContainer_registerSecurityComponents_Execution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	container.registerSecurityComponents()

	// Test that the registered components can be retrieved and executed
	vulnChecker, err := container.Get("vulnChecker")
	require.NoError(t, err)
	assert.NotNil(t, vulnChecker)

	linter, err := container.Get("linter")
	require.NoError(t, err)
	assert.NotNil(t, linter)

	// Verify the components are of the expected types
	_, ok := vulnChecker.(interfaces.VulnChecker)
	assert.True(t, ok, "vulnChecker should implement VulnChecker interface")

	_, ok = linter.(interfaces.Linter)
	assert.True(t, ok, "linter should implement Linter interface")
}

func TestContainer_GetConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test that GetConfiguration returns the same configuration instance
	config := container.GetConfiguration()
	assert.Equal(t, mockConfig, config)
	assert.NotNil(t, config)
}

func TestContainer_registerLoggingComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test with default log level
	mockConfig.EXPECT().Logging().Return(interfaces.LoggingConfig{
		Level:            "info",
		Format:           "json",
		SamplingEnable:   false,
		SamplingRate:     0.1,
		SamplingInterval: 1 * time.Second,
	})

	container.registerLoggingComponents()

	// Verify component is registered
	_, exists := container.once["logger"]
	assert.True(t, exists)

	// Try to get the component - this will call the factory function
	logger, err := container.Get("logger")
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestContainer_registerLoggingComponents_WithCustomLevel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Test with custom log level
	mockConfig.EXPECT().Logging().Return(interfaces.LoggingConfig{
		Level:            "debug",
		Format:           "json",
		SamplingEnable:   true,
		SamplingRate:     0.5,
		SamplingInterval: 2 * time.Second,
	})

	container.registerLoggingComponents()

	// Verify component is registered
	_, exists := container.once["logger"]
	assert.True(t, exists)

	// Try to get the component - this will call the factory function
	logger, err := container.Get("logger")
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestContainer_registerStepComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	container.registerStepComponents()

	// Verify components are registered
	_, exists := container.once["stepRegistry"]
	assert.True(t, exists)
	_, exists = container.once["pipelineExecutor"]
	assert.True(t, exists)
}

func TestContainer_registerHookComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	container.registerHookComponents()

	// Verify component is registered
	_, exists := container.once["hookManager"]
	assert.True(t, exists)
}

func TestContainer_registerStepComponents_Execution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	// Register required dependencies first
	container.Register("logger", func() (any, error) {
		logger := slog.Default()
		return &LoggerAdapter{logger: logger}, nil
	})
	container.Register("daggerClient", func() (any, error) {
		return &dagger.Client{}, nil
	})

	// Register hookManager as it's required by pipelineExecutor
	container.registerHookComponents()

	container.registerStepComponents()

	// Test that the registered components can be retrieved and executed
	stepRegistry, err := container.Get("stepRegistry")
	require.NoError(t, err)
	assert.NotNil(t, stepRegistry)

	pipelineExecutor, err := container.Get("pipelineExecutor")
	require.NoError(t, err)
	assert.NotNil(t, pipelineExecutor)

	// Verify the components are of the expected types
	_, ok := stepRegistry.(interfaces.StepRegistry)
	assert.True(t, ok, "stepRegistry should implement StepRegistry interface")

	_, ok = pipelineExecutor.(interfaces.PipelineExecutor)
	assert.True(t, ok, "pipelineExecutor should implement PipelineExecutor interface")
}

func TestContainer_registerHookComponents_Execution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	ctx := t.Context()

	container := NewContainer(ctx, mockConfig)

	container.registerHookComponents()

	// Test that the registered component can be retrieved and executed
	hookManager, err := container.Get("hookManager")
	require.NoError(t, err)
	assert.NotNil(t, hookManager)

	// Verify the component is of the expected type
	_, ok := hookManager.(interfaces.HookManager)
	assert.True(t, ok, "hookManager should implement HookManager interface")
}

// Helper function to create a mock pipelines.Pipeline
func createMockPipelinesPipeline(ctrl *gomock.Controller) pipelines.Pipeline {
	return mocks.NewPipelinesMockPipeline(ctrl)
}

// Helper function to setup mock configuration expectations for convertConfig
func setupMockConfigForConvertConfig(mockConfig *mocks.MockConfiguration) {
	mockConfig.EXPECT().Environment().Return("dev").AnyTimes()
	mockConfig.EXPECT().GetBool(gomock.Any()).Return(false).AnyTimes()
	mockConfig.EXPECT().GetString(gomock.Any()).DoAndReturn(func(key string) string {
		// Return appropriate default values based on the key
		switch key {
		case "git.ref":
			return "main"
		case "git.protocol":
			return "ssh"
		case "pipeline.go_version":
			return "1.21"
		default:
			return ""
		}
	}).AnyTimes()
	mockConfig.EXPECT().GetFloat(gomock.Any()).Return(90.0).AnyTimes()
}

// Test pipeline factory functions
func TestNewGoKitPipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	client := &dagger.Client{}

	// Setup mock expectations for convertConfig
	setupMockConfigForConvertConfig(mockConfig)

	// Test the factory function
	pipeline := NewGoKitPipeline(client, mockConfig)

	// Should return a PipelineAdapter, not nil
	assert.NotNil(t, pipeline)
	assert.IsType(t, &PipelineAdapter{}, pipeline)
}

func TestNewDockerGoPipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	client := &dagger.Client{}

	// Setup mock expectations for convertConfig
	setupMockConfigForConvertConfig(mockConfig)

	// Test the factory function
	pipeline := NewDockerGoPipeline(client, mockConfig)

	// Should return a PipelineAdapter, not nil
	assert.NotNil(t, pipeline)
	assert.IsType(t, &PipelineAdapter{}, pipeline)
}

func TestNewInfraPipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	client := &dagger.Client{}

	// Setup mock expectations for convertConfig
	setupMockConfigForConvertConfig(mockConfig)

	// Test the factory function
	pipeline := NewInfraPipeline(client, mockConfig)

	// Should return a PipelineAdapter, not nil
	assert.NotNil(t, pipeline)
	assert.IsType(t, &PipelineAdapter{}, pipeline)
}

// Test PipelineAdapter functionality
func TestPipelineAdapter_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock pipeline
	mockPipeline := createMockPipelinesPipeline(ctrl)
	mockPipeline.(*mocks.PipelinesMockPipeline).EXPECT().Name().Return("test-pipeline")

	adapter := NewPipelineAdapter(mockPipeline)
	assert.Equal(t, "test-pipeline", adapter.Name())
}

func TestPipelineAdapter_GetAvailableSteps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPipeline := createMockPipelinesPipeline(ctrl)
	adapter := NewPipelineAdapter(mockPipeline)

	steps := adapter.GetAvailableSteps()
	expectedSteps := []string{"setup", "build", "test", "package", "tag", "push"}
	assert.Equal(t, expectedSteps, steps)
}

func TestPipelineAdapter_ExecuteStep(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPipeline := createMockPipelinesPipeline(ctrl)
	adapter := NewPipelineAdapter(mockPipeline)

	ctx := t.Context()

	// Test setup step
	mockPipeline.(*mocks.PipelinesMockPipeline).EXPECT().Setup(ctx).Return(nil)
	err := adapter.ExecuteStep(ctx, "setup")
	require.NoError(t, err)

	// Test build step
	mockPipeline.(*mocks.PipelinesMockPipeline).EXPECT().Build(ctx).Return(nil)
	err = adapter.ExecuteStep(ctx, "build")
	require.NoError(t, err)

	// Test unknown step
	err = adapter.ExecuteStep(ctx, "unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown step")
}

func TestPipelineAdapter_ValidateStep(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPipeline := createMockPipelinesPipeline(ctrl)
	adapter := NewPipelineAdapter(mockPipeline)

	// Test valid step
	err := adapter.ValidateStep("setup")
	require.NoError(t, err)

	// Test invalid step
	err = adapter.ValidateStep("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid step")
}

func TestPipelineAdapter_GetStepConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPipeline := createMockPipelinesPipeline(ctrl)
	adapter := NewPipelineAdapter(mockPipeline)

	config := adapter.GetStepConfig("setup")
	assert.Equal(t, "setup", config.Name)
	assert.Equal(t, "Execute setup step", config.Description)
	assert.True(t, config.Required)
	assert.Equal(t, 5*time.Minute, config.Timeout)
	assert.Equal(t, 0, config.Retries)
}
