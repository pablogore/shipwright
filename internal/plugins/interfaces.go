package plugins

import (
	"context"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// Capabilities bundles the Layer 1 capability interfaces (pkg/shipwright:
// Builder, Tester, Artifactor, Deployer, Runner) that are wired for the
// current run, exposed to plugins via PluginContext.GetCapabilities().
//
// This replaces the retired PluginContext.GetPipeline() (WU10, tasks.md
// 10.1/10.2, design.md's composition-model migration) — the composition
// contract's extension point is now the five capability interfaces
// directly, never a bundled "Pipeline" type. Capabilities itself carries
// no bundle identity or name (design.md D-F): it is a plain data carrier
// exposing whichever capabilities are wired for this run, not a named
// capability-set preset. Any field MAY be nil — a run need not wire every
// capability, and Deployer/Runner have no concrete implementation yet
// (pkg/shipwright.DeployConfig/RunConfig are empty at this change).
// Testers is a slice because multiple independent Tester implementations
// MAY compose over the same input (design.md D-F's orthogonality win —
// unit test, lint, vulnerability scan are three separate Testers).
type Capabilities struct {
	Builder    shipwright.Builder
	Testers    []shipwright.Tester
	Artifactor shipwright.Artifactor
	Deployer   shipwright.Deployer
	Runner     shipwright.Runner
}

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

	// GetCapabilities returns the Layer 1 capability bundle wired for the
	// current run (replaces the retired GetPipeline()).
	GetCapabilities() Capabilities

	// GetConfig returns the pipeline-specific configuration (replaces the
	// retired GetPipelineConfig()).
	GetConfig() pipelines.Config

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
