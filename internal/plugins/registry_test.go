package plugins

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/internal/interfaces"
)

func TestNewRegistry(t *testing.T) {
	t.Run("CreatesNewRegistry", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()

		// Act
		registry := NewRegistry(loader)

		// Assert
		require.NotNil(t, registry)
		assert.Implements(t, (*PluginRegistry)(nil), registry)
	})
}

func TestRegistry_RegisterPlugin(t *testing.T) {
	t.Run("RegistersPluginSuccessfully", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "test-plugin" }
		plugin.VersionFunc = func() string { return "1.0.0" }

		// Act
		err := registry.RegisterPlugin(plugin)

		// Assert
		require.NoError(t, err)
		plugins := registry.ListPlugins()
		assert.Contains(t, plugins, "test-plugin")
	})

	t.Run("ReturnsErrorWhenPluginIsNil", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)

		// Act
		err := registry.RegisterPlugin(nil)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plugin cannot be nil")
	})

	t.Run("ReturnsErrorWhenPluginNameIsEmpty", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "" }

		// Act
		err := registry.RegisterPlugin(plugin)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plugin name cannot be empty")
	})

	t.Run("ReturnsErrorWhenPluginAlreadyRegistered", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin1 := NewMockPlugin()
		plugin1.NameFunc = func() string { return "test-plugin" }
		plugin2 := NewMockPlugin()
		plugin2.NameFunc = func() string { return "test-plugin" }

		// Act
		err1 := registry.RegisterPlugin(plugin1)
		err2 := registry.RegisterPlugin(plugin2)

		// Assert
		require.NoError(t, err1)
		require.Error(t, err2)
		assert.Contains(t, err2.Error(), "already registered")
	})
}

func TestRegistry_GetPlugin(t *testing.T) {
	t.Run("ReturnsRegisteredPlugin", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "test-plugin" }
		_ = registry.RegisterPlugin(plugin)

		// Act
		result, err := registry.GetPlugin("test-plugin")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, plugin, result)
	})

	t.Run("ReturnsErrorWhenPluginNotFound", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)

		// Act
		result, err := registry.GetPlugin("non-existent")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRegistry_ListPlugins(t *testing.T) {
	t.Run("ReturnsEmptyListWhenNoPlugins", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)

		// Act
		plugins := registry.ListPlugins()

		// Assert
		assert.Empty(t, plugins)
	})

	t.Run("ReturnsListOfRegisteredPlugins", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin1 := NewMockPlugin()
		plugin1.NameFunc = func() string { return "plugin-1" }
		plugin2 := NewMockPlugin()
		plugin2.NameFunc = func() string { return "plugin-2" }
		_ = registry.RegisterPlugin(plugin1)
		_ = registry.RegisterPlugin(plugin2)

		// Act
		plugins := registry.ListPlugins()

		// Assert
		assert.Len(t, plugins, 2)
		assert.Contains(t, plugins, "plugin-1")
		assert.Contains(t, plugins, "plugin-2")
	})
}

func TestRegistry_LoadPlugin(t *testing.T) {
	t.Run("LoadsBuiltinPlugin", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "test-plugin" }
		plugin.InitializeFunc = func(_ context.Context, _ PluginContext) error { return nil }

		loader.LoadBuiltinFunc = func(_ context.Context, name string) (Plugin, error) {
			if name == "test-plugin" {
				return plugin, nil
			}
			return nil, errors.New("not found")
		}

		registry := NewRegistry(loader).(*registry)
		pluginCtx := NewMockPluginContext()

		// Act
		err := registry.LoadPlugin(ctx, "test-plugin", pluginCtx)

		// Assert
		require.NoError(t, err)
		plugins := registry.ListPlugins()
		assert.Contains(t, plugins, "test-plugin")
	})

	t.Run("ReturnsErrorWhenPluginNameIsEmpty", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		pluginCtx := NewMockPluginContext()

		// Act
		err := registry.LoadPlugin(context.Background(), "", pluginCtx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plugin name cannot be empty")
	})

	t.Run("ReturnsErrorWhenPluginAlreadyLoaded", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "test-plugin" }
		_ = registry.RegisterPlugin(plugin)
		pluginCtx := NewMockPluginContext()

		// Act
		err := registry.LoadPlugin(context.Background(), "test-plugin", pluginCtx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already loaded")
	})

	t.Run("ReturnsErrorWhenPluginNotFound", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		loader.LoadBuiltinFunc = func(_ context.Context, _ string) (Plugin, error) {
			return nil, errors.New("not found")
		}
		registry := NewRegistry(loader).(*registry)
		pluginCtx := NewMockPluginContext()
		cfg := NewMockConfiguration()
		cfg.GetFunc = func(key string) any {
			if key == "plugins" {
				return nil
			}
			return nil
		}
		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }

		// Act
		err := registry.LoadPlugin(context.Background(), "non-existent", pluginCtx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("LoadsPluginFromConfig", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "config-plugin" }
		plugin.InitializeFunc = func(_ context.Context, _ PluginContext) error { return nil }

		loader.LoadBuiltinFunc = func(_ context.Context, _ string) (Plugin, error) {
			return nil, errors.New("not found")
		}
		loader.LoadFromConfigFunc = func(_ context.Context, _ map[string]interface{}) (Plugin, error) {
			return plugin, nil
		}

		registry := NewRegistry(loader).(*registry)
		pluginCtx := NewMockPluginContext()
		cfg := NewMockConfiguration()
		cfg.GetFunc = func(key string) any {
			if key == "plugins" {
				return map[string]interface{}{
					"config-plugin": map[string]interface{}{
						"type": "builtin",
						"name": "config-plugin",
					},
				}
			}
			return nil
		}
		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }

		// Act
		err := registry.LoadPlugin(ctx, "config-plugin", pluginCtx)

		// Assert
		require.NoError(t, err)
		plugins := registry.ListPlugins()
		assert.Contains(t, plugins, "config-plugin")
	})
}

func TestRegistry_LoadPluginsFromConfig(t *testing.T) {
	t.Run("LoadsPluginsFromConfig", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "test-plugin" }
		plugin.InitializeFunc = func(_ context.Context, _ PluginContext) error { return nil }

		loader.LoadFromConfigFunc = func(_ context.Context, _ map[string]interface{}) (Plugin, error) {
			return plugin, nil
		}

		registry := NewRegistry(loader).(*registry)
		pluginCtx := NewMockPluginContext()
		cfg := NewMockConfiguration()
		cfg.GetFunc = func(key string) any {
			if key == "plugins" {
				return map[string]interface{}{
					"test-plugin": map[string]interface{}{
						"type": "builtin",
						"name": "test-plugin",
					},
				}
			}
			return nil
		}
		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }

		// Act
		err := registry.LoadPluginsFromConfig(ctx, pluginCtx)

		// Assert
		require.NoError(t, err)
		plugins := registry.ListPlugins()
		assert.Contains(t, plugins, "test-plugin")
	})

	t.Run("ReturnsNoErrorWhenNoPluginsConfig", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		pluginCtx := NewMockPluginContext()
		cfg := NewMockConfiguration()
		cfg.GetFunc = func(key string) any {
			if key == "plugins" {
				return nil
			}
			return nil
		}
		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }

		// Act
		err := registry.LoadPluginsFromConfig(ctx, pluginCtx)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("ReturnsErrorWhenPluginContextIsNil", func(t *testing.T) {
		// Arrange
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)

		// Act
		err := registry.LoadPluginsFromConfig(context.Background(), nil)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plugin context cannot be nil")
	})

	t.Run("ReturnsErrorWhenInvalidPluginsConfigFormat", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		pluginCtx := NewMockPluginContext()
		cfg := NewMockConfiguration()
		cfg.GetFunc = func(key string) any {
			if key == "plugins" {
				return "invalid-format"
			}
			return nil
		}
		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }

		// Act
		err := registry.LoadPluginsFromConfig(ctx, pluginCtx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugins configuration format")
	})

	t.Run("ReturnsErrorWhenPluginLoadFails", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		loader.LoadBuiltinFunc = func(_ context.Context, _ string) (Plugin, error) {
			return nil, errors.New("load failed")
		}
		loader.LoadFromConfigFunc = func(_ context.Context, _ map[string]interface{}) (Plugin, error) {
			return nil, errors.New("load failed")
		}

		registry := NewRegistry(loader).(*registry)
		pluginCtx := NewMockPluginContext()
		cfg := NewMockConfiguration()
		cfg.GetFunc = func(key string) any {
			if key == "plugins" {
				return map[string]interface{}{
					"failing-plugin": map[string]interface{}{
						"type": "builtin",
						"name": "failing-plugin",
					},
				}
			}
			return nil
		}
		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }

		// Act
		err := registry.LoadPluginsFromConfig(ctx, pluginCtx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load some plugins")
	})
}

func TestRegistry_InitializeAll(t *testing.T) {
	t.Run("InitializesAllPlugins", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin1 := NewMockPlugin()
		plugin1.NameFunc = func() string { return "plugin-1" }
		plugin1.InitializeFunc = func(_ context.Context, _ PluginContext) error { return nil }
		plugin2 := NewMockPlugin()
		plugin2.NameFunc = func() string { return "plugin-2" }
		plugin2.InitializeFunc = func(_ context.Context, _ PluginContext) error { return nil }
		_ = registry.RegisterPlugin(plugin1)
		_ = registry.RegisterPlugin(plugin2)
		pluginCtx := NewMockPluginContext()

		// Act
		err := registry.InitializeAll(ctx, pluginCtx)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("ReturnsErrorWhenInitializationFails", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "test-plugin" }
		plugin.InitializeFunc = func(_ context.Context, _ PluginContext) error {
			return errors.New("initialization failed")
		}
		_ = registry.RegisterPlugin(plugin)
		pluginCtx := NewMockPluginContext()

		// Act
		err := registry.InitializeAll(ctx, pluginCtx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize")
	})
}

func TestRegistry_CleanupAll(t *testing.T) {
	t.Run("CleansUpAllPlugins", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin1 := NewMockPlugin()
		plugin1.NameFunc = func() string { return "plugin-1" }
		plugin1.CleanupFunc = func(_ context.Context) error { return nil }
		plugin2 := NewMockPlugin()
		plugin2.NameFunc = func() string { return "plugin-2" }
		plugin2.CleanupFunc = func(_ context.Context) error { return nil }
		_ = registry.RegisterPlugin(plugin1)
		_ = registry.RegisterPlugin(plugin2)

		// Act
		err := registry.CleanupAll(ctx)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("ReturnsErrorWhenCleanupFails", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "test-plugin" }
		plugin.CleanupFunc = func(_ context.Context) error {
			return errors.New("cleanup failed")
		}
		_ = registry.RegisterPlugin(plugin)

		// Act
		err := registry.CleanupAll(ctx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to cleanup")
	})
}

func TestRegistry_registerAndInitialize(t *testing.T) {
	t.Run("RegistersAndInitializesPlugin", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "test-plugin" }
		plugin.VersionFunc = func() string { return "1.0.0" }
		plugin.InitializeFunc = func(_ context.Context, _ PluginContext) error { return nil }
		pluginCtx := NewMockPluginContext()

		// Act
		err := registry.registerAndInitialize(ctx, plugin, pluginCtx)

		// Assert
		require.NoError(t, err)
		plugins := registry.ListPlugins()
		assert.Contains(t, plugins, "test-plugin")
	})

	t.Run("ReturnsErrorWhenInitializationFails", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		plugin := NewMockPlugin()
		plugin.NameFunc = func() string { return "failing-plugin" }
		plugin.VersionFunc = func() string { return "1.0.0" }
		plugin.InitializeFunc = func(_ context.Context, _ PluginContext) error {
			return errors.New("initialization failed")
		}
		pluginCtx := NewMockPluginContext()

		// Act
		err := registry.registerAndInitialize(ctx, plugin, pluginCtx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize")
		// Plugin should not be registered if initialization fails
		plugins := registry.ListPlugins()
		assert.NotContains(t, plugins, "failing-plugin")
	})

	t.Run("ReturnsErrorWhenRegistrationFails", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		loader := NewMockPluginLoader()
		registry := NewRegistry(loader).(*registry)
		// Register a plugin first
		existingPlugin := NewMockPlugin()
		existingPlugin.NameFunc = func() string { return "existing-plugin" }
		existingPlugin.VersionFunc = func() string { return "1.0.0" }
		existingPlugin.InitializeFunc = func(_ context.Context, _ PluginContext) error { return nil }
		pluginCtx := NewMockPluginContext()
		err := registry.registerAndInitialize(ctx, existingPlugin, pluginCtx)
		require.NoError(t, err)

		// Try to register the same plugin again (should fail)
		duplicatePlugin := NewMockPlugin()
		duplicatePlugin.NameFunc = func() string { return "existing-plugin" }
		duplicatePlugin.VersionFunc = func() string { return "2.0.0" }

		// Act
		err = registry.registerAndInitialize(ctx, duplicatePlugin, pluginCtx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register plugin")
	})
}
