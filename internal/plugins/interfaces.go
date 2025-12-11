package plugins

import (
	"context"

	"dagger.io/dagger"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
)

// Plugin defines the interface that all plugins must implement.
// Plugins can extend pipeline functionality by:
//   - Registering hooks for existing steps
//   - Adding new custom steps
//   - Accessing pipeline context (Dagger client, config, etc.)
type Plugin interface {
	// Name returns the unique name of the plugin.
	Name() string

	// Version returns the version of the plugin.
	Version() string

	// Initialize is called when the plugin is loaded.
	// The plugin can use this to register hooks, add steps, etc.
	Initialize(ctx context.Context, pluginCtx PluginContext) error

	// Cleanup is called when the plugin is unloaded or the pipeline completes.
	Cleanup(ctx context.Context) error
}

// PluginContext provides access to pipeline resources for plugins.
// This allows plugins to interact with the pipeline without modifying the core library.
type PluginContext interface {
	// GetDaggerClient returns the Dagger client for container operations.
	GetDaggerClient() (*dagger.Client, error)

	// GetConfiguration returns the pipeline configuration.
	GetConfiguration() interfaces.Configuration

	// GetHookManager returns the hook manager for registering hooks.
	GetHookManager() interfaces.HookManager

	// GetStepRegistry returns the step registry for adding custom steps.
	GetStepRegistry() interfaces.StepRegistry

	// GetPipeline returns the current pipeline instance.
	GetPipeline() interfaces.Pipeline

	// GetPipelineConfig returns the pipeline-specific configuration.
	GetPipelineConfig() pipelines.Config

	// GetLogger returns the logger instance.
	GetLogger() interfaces.Logger
}

// PluginRegistry manages plugin registration and lifecycle.
type PluginRegistry interface {
	// RegisterPlugin registers a plugin with the registry.
	RegisterPlugin(plugin Plugin) error

	// LoadPlugin loads a plugin by name from configuration or file.
	LoadPlugin(ctx context.Context, name string, pluginCtx PluginContext) error

	// LoadPluginsFromConfig loads all plugins specified in configuration.
	LoadPluginsFromConfig(ctx context.Context, pluginCtx PluginContext) error

	// GetPlugin retrieves a registered plugin by name.
	GetPlugin(name string) (Plugin, error)

	// ListPlugins returns all registered plugin names.
	ListPlugins() []string

	// InitializeAll initializes all registered plugins.
	InitializeAll(ctx context.Context, pluginCtx PluginContext) error

	// CleanupAll cleans up all registered plugins.
	CleanupAll(ctx context.Context) error
}

// PluginLoader loads plugins from various sources (files, config, etc.).
type PluginLoader interface {
	// LoadFromFile loads a plugin from a Go file.
	LoadFromFile(ctx context.Context, filePath string) (Plugin, error)

	// LoadFromConfig loads a plugin from configuration.
	LoadFromConfig(ctx context.Context, config map[string]interface{}) (Plugin, error)

	// LoadBuiltin loads a built-in plugin by name.
	LoadBuiltin(ctx context.Context, name string) (Plugin, error)

	// RegisterBuiltin registers a built-in plugin factory.
	RegisterBuiltin(name string, factory func() Plugin)
}
