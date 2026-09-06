package plugins

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

func TestMockPlugin(t *testing.T) {
	t.Run("ReturnsDefaultValues", func(t *testing.T) {
		// Arrange
		plugin := NewMockPlugin()

		// Act
		name := plugin.Name()
		version := plugin.Version()
		err := plugin.Cleanup(context.Background())
		initErr := plugin.Initialize(context.Background(), nil)

		// Assert
		assert.Equal(t, "mock-plugin", name)
		assert.Equal(t, "1.0.0", version)
		assert.NoError(t, err)
		assert.NoError(t, initErr)
	})

	t.Run("UsesCustomFunctions", func(t *testing.T) {
		// Arrange
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "custom-plugin" }
		plugin.VersionFunc = func() string { return "2.0.0" }
		plugin.CleanupFunc = func(_ context.Context) error { return errors.New("cleanup error") }
		plugin.InitializeFunc = func(_ context.Context, _ PluginContext) error {
			return errors.New("init error")
		}

		// Act
		name := plugin.Name()
		version := plugin.Version()
		err := plugin.Cleanup(context.Background())
		initErr := plugin.Initialize(context.Background(), nil)

		// Assert
		assert.Equal(t, "custom-plugin", name)
		assert.Equal(t, "2.0.0", version)
		require.Error(t, err)
		assert.Error(t, initErr)
	})
}

func TestMockPluginContext(t *testing.T) {
	t.Run("ReturnsDefaultValues", func(t *testing.T) {
		// Arrange
		ctx := NewMockPluginContext()

		// Act
		cfg := ctx.GetConfiguration()
		client, clientErr := ctx.GetDaggerClient()
		hookManager := ctx.GetHookManager()
		logger := ctx.GetLogger()
		caps := ctx.GetCapabilities()
		pipelineConfig := ctx.GetConfig()
		stepRegistry := ctx.GetStepRegistry()

		// Assert
		assert.Nil(t, cfg)
		assert.Nil(t, client)
		require.NoError(t, clientErr)
		assert.Nil(t, hookManager)
		assert.Nil(t, logger)
		assert.Equal(t, Capabilities{}, caps)
		assert.Equal(t, pipelines.Config{}, pipelineConfig)
		assert.Nil(t, stepRegistry)
	})

	t.Run("UsesCustomFunctions", func(t *testing.T) {
		// Arrange
		ctx := NewMockPluginContext()
		mockCfg := NewMockConfiguration()
		mockHookManager := NewMockHookManager()
		mockStepRegistry := NewMockStepRegistry()
		mockCaps := Capabilities{Testers: []shipwright.Tester{}}
		mockLogger := NewMockLogger()
		daggerClient := &dagger.Client{}

		ctx.GetConfigurationFunc = func() interfaces.Configuration { return mockCfg }
		ctx.GetDaggerClientFunc = func() (*dagger.Client, error) { return daggerClient, nil }
		ctx.GetHookManagerFunc = func() interfaces.HookManager { return mockHookManager }
		ctx.GetStepRegistryFunc = func() interfaces.StepRegistry { return mockStepRegistry }
		ctx.GetCapabilitiesFunc = func() Capabilities { return mockCaps }
		ctx.GetLoggerFunc = func() interfaces.Logger { return mockLogger }
		ctx.GetConfigFunc = func() pipelines.Config {
			return pipelines.Config{Env: "test"}
		}

		// Act
		cfg := ctx.GetConfiguration()
		client, clientErr := ctx.GetDaggerClient()
		hookManager := ctx.GetHookManager()
		logger := ctx.GetLogger()
		caps := ctx.GetCapabilities()
		pipelineConfig := ctx.GetConfig()
		stepRegistry := ctx.GetStepRegistry()

		// Assert
		assert.Equal(t, mockCfg, cfg)
		assert.Equal(t, daggerClient, client)
		require.NoError(t, clientErr)
		assert.Equal(t, mockHookManager, hookManager)
		assert.Equal(t, mockLogger, logger)
		assert.Equal(t, mockCaps, caps)
		assert.Equal(t, "test", pipelineConfig.Env)
		assert.Equal(t, mockStepRegistry, stepRegistry)
	})
}

func TestMockPluginLoader(t *testing.T) {
	t.Run("ReturnsDefaultValues", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()

		// Act
		plugin, err := loader.LoadBuiltin(context.Background(), "test")
		plugin2, err2 := loader.LoadFromConfig(context.Background(), map[string]interface{}{})
		plugin3, err3 := loader.LoadFromFile(context.Background(), "/path/to/plugin.so")

		// Assert
		assert.Nil(t, plugin)
		require.NoError(t, err)
		assert.Nil(t, plugin2)
		require.NoError(t, err2)
		assert.Nil(t, plugin3)
		assert.NoError(t, err3)
	})

	t.Run("UsesCustomFunctions", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		testPlugin := NewMockPlugin()
		testPlugin.NameFunc = func() string { return "test-plugin" }

		loader.LoadBuiltinFunc = func(_ context.Context, name string) (Plugin, error) {
			if name == "test-plugin" {
				return testPlugin, nil
			}
			return nil, errors.New("not found")
		}
		loader.LoadFromConfigFunc = func(_ context.Context, _ map[string]interface{}) (Plugin, error) {
			return testPlugin, nil
		}
		loader.LoadFromFileFunc = func(_ context.Context, _ string) (Plugin, error) {
			return testPlugin, nil
		}

		// Act
		plugin1, err1 := loader.LoadBuiltin(context.Background(), "test-plugin")
		plugin2, err2 := loader.LoadFromConfig(context.Background(), map[string]interface{}{})
		plugin3, err3 := loader.LoadFromFile(context.Background(), "/path/to/plugin.so")

		// Assert
		assert.Equal(t, testPlugin, plugin1)
		require.NoError(t, err1)
		assert.Equal(t, testPlugin, plugin2)
		require.NoError(t, err2)
		assert.Equal(t, testPlugin, plugin3)
		assert.NoError(t, err3)
	})

	t.Run("RegisterBuiltinCallsFunction", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		called := false
		loader.RegisterBuiltinFunc = func(name string, factory func() Plugin) {
			called = true
			assert.Equal(t, "test-plugin", name)
			assert.NotNil(t, factory)
		}

		// Act
		loader.RegisterBuiltin("test-plugin", func() Plugin {
			return NewMockPlugin()
		})

		// Assert
		assert.True(t, called)
	})
}
