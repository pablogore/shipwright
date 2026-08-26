package plugins

import (
	"context"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/pablogore/shipwright/internal/workflow/providers"
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
//
// Extension point (single, current): during Initialize a plugin registers a
// Layer 1 capability implementation (pkg/shipwright:
// Builder/Tester/Artifactor/Deployer/Runner) into the *providers.Registry
// reachable through PluginContext.GetProviderRegistry(), using WU7's own
// Register*/Resolve* primitives. That registry is the exact one
// internal/workflow/engine.Execute resolves each manifest step against, so a
// plugin-contributed provider is indistinguishable from an in-repo one at
// resolution time (internal/workflow/providers/register.go's RegisterDefaults
// is the same registration shape).
//
// The older extension points — HookManager.RegisterHook and
// StepRegistry.RegisterStep — are NOT the workflow extension mechanism. Their
// only executor was the legacy preset-driven step flow deleted in this
// change's preset-deletion work unit; engine.Execute never consults either.
// A plugin registering into them today registers behavior nothing will ever
// run. The accessors remain on PluginContext for the pre-existing
// non-workflow consumers under examples/, and are deliberately unused by the
// in-repo plugin.
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
	//
	// Deprecated as an extension point: nothing in the workflow engine
	// executes StepRegistry steps — see the Plugin doc comment.
	GetStepRegistry() interfaces.StepRegistry

	// GetCapabilities returns the Layer 1 capability bundle wired for the
	// current run (replaces the retired GetPipeline()).
	GetCapabilities() Capabilities

	// GetProviderRegistry returns the typed provider registry
	// (internal/workflow/providers, WU7) the current workflow run resolves
	// every manifest step against, so a plugin can CONTRIBUTE a capability
	// implementation into it during Initialize:
	//
	//	reg := pluginCtx.GetProviderRegistry()
	//	reg.RegisterDeployer(providers.Ref{Name: "nomad-deploy", Version: "1"},
	//	    providers.WithSchema{...}, func(v providers.Values) shipwright.Deployer { ... })
	//
	// This is deliberately the registry ITSELF rather than one new Plugin
	// method per capability: WU7 already ships, and already tests, five typed
	// Register*/Resolve* pairs, so the bridge adds one accessor instead of a
	// parallel five-method surface that would have to be kept in sync.
	//
	// Returns nil when the current run wired no registry (for example a unit
	// test constructing a bare PluginContext). A plugin MUST treat nil as
	// "no provider extension point available" and skip registration rather
	// than panic.
	//
	// SECURITY (design.md D-I): the returned registry only ever receives
	// factories from Go code compiled into this binary. Plugins reaching this
	// accessor are loaded through PluginRegistry.LoadBuiltinPlugins, which
	// resolves ONLY compile-time-registered builtin factories and never
	// touches PluginLoader.LoadFromFile/LoadFromConfig — the two paths that
	// can reach plugin.Open. A plugin-contributed provider is therefore still
	// "already compiled, self-registered" exactly as D-I requires.
	GetProviderRegistry() *providers.Registry

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
	//
	// SECURITY: this path can reach PluginLoader.LoadFromConfig, whose
	// `type: file` branch calls plugin.Open on a config-supplied path
	// (arbitrary in-process native code). It is NOT used by the --workflow
	// entrypoint — see LoadBuiltinPlugins — and must stay reserved for
	// operator-initiated, out-of-band plugin installation (the security
	// skill's "prefer LoadBuiltin over LoadFromFile" rule).
	LoadPluginsFromConfig(ctx context.Context, pluginCtx PluginContext) error

	// LoadBuiltinPlugins registers and initializes every plugin the
	// PluginLoader has a compile-time builtin factory for
	// (PluginLoader.RegisterBuiltin, called from the DI container's
	// registerPluginComponents). This is the ONLY plugin load path the
	// --workflow CLI entrypoint uses.
	//
	// It never consults configuration and never calls LoadFromConfig or
	// LoadFromFile, so plugin.Open is unreachable from it — the fail-closed
	// counterpart to design.md D-I's "providers resolve only to already
	// compiled, self-registered implementations".
	//
	// Already-registered plugins are skipped, so calling it more than once in
	// a process is safe.
	LoadBuiltinPlugins(ctx context.Context, pluginCtx PluginContext) error

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

	// ListBuiltins returns the names of every compile-time-registered
	// builtin plugin factory, in unspecified order. This is what lets
	// PluginRegistry.LoadBuiltinPlugins load the full builtin set WITHOUT
	// going through configuration — the config path is the only one that can
	// reach plugin.Open.
	ListBuiltins() []string
}
