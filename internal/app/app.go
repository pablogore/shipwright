package app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"dagger.io/dagger"

	"github.com/pablogore/kit-logger/pkg/logger"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/plugins"
	"github.com/pablogore/shipwright/internal/workflow/providers"
)

var (
	globalContainer *Container
	containerOnce   sync.Once
	containerMutex  sync.RWMutex
)

// GetContainer returns the global container instance.
func GetContainer() *Container {
	containerMutex.RLock()
	defer containerMutex.RUnlock()

	if globalContainer == nil {
		panic("Container not initialized. Call app.Initialize() first")
	}
	return globalContainer
}

// Initialize sets up the global container instance.
func Initialize(ctx context.Context, cfg interfaces.Configuration) error {
	var initErr error

	containerOnce.Do(func() {
		container := NewContainer(ctx, cfg)

		// Initialize logger (go-kit-logger is initialized in registerLoggingComponents)
		if err := container.CreateLogger(); err != nil {
			// Logger initialization error is non-fatal, continue with default logger.
			_ = err
		}

		if err := container.Start(ctx); err != nil {
			initErr = fmt.Errorf("failed to start container: %w", err)
			return
		}

		containerMutex.Lock()
		globalContainer = container
		containerMutex.Unlock()
	})

	return initErr
}

// Reset clears the global container (useful for testing).
func Reset() {
	containerMutex.Lock()
	if globalContainer != nil {
		_ = globalContainer.Stop(context.Background())
	}
	globalContainer = nil
	containerOnce = sync.Once{} // Reset the sync.Once to allow re-initialization
	containerMutex.Unlock()
}

// App represents the main application instance.
// It manages the application lifecycle and provides access to core functionality.
//
// Features:
// - Application lifecycle management (start/stop)
// - Container management
// - Graceful shutdown
// - Component health monitoring
//
// Example usage:
//
//	container := app.GetContainer()
//	application := app.NewApp(container)
//	if err := application.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer application.Stop(ctx)
type App struct {
	container *Container
}

// NewApp creates a new application instance.
func NewApp(container *Container) *App {
	return &App{container: container}
}

// Start starts the application.
func (a *App) Start(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Starting Shipwright application...")
	return a.container.Start(ctx)
}

// Stop stops the application.
func (a *App) Stop(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Stopping Shipwright application...")
	return a.container.Stop(ctx)
}

// GetContainer returns the application's container.
func (a *App) GetContainer() *Container {
	return a.container
}

// LoadAndInitializePlugins loads every compile-time-registered builtin plugin
// and initializes it against providerRegistry — the SAME *providers.Registry
// the caller's workflow run resolves each manifest step against. A plugin
// contributes a Layer 1 capability implementation by registering into that
// registry during Initialize (plugins.PluginContext.GetProviderRegistry).
//
// daggerClient is passed explicitly rather than resolved from the container
// because the container's "daggerClient" component connects lazily: the
// --list-steps path must be able to wire plugins WITHOUT starting a Dagger
// engine, and passes nil. A nil client yields an empty capability bundle
// (BuildCapabilities' documented behavior), which is correct — resolution
// does not need a live client.
//
// SECURITY (design.md D-I): this uses PluginRegistry.LoadBuiltinPlugins, not
// LoadPluginsFromConfig. Only compile-time builtin factories are loaded, so
// plugin.Open is unreachable from the CLI's plugin wiring and every provider
// reaching the registry is still build-time-compiled code.
func (a *App) LoadAndInitializePlugins(
	ctx context.Context,
	providerRegistry *providers.Registry,
	daggerClient *dagger.Client,
) error {
	if providerRegistry == nil {
		return errors.New("provider registry cannot be nil")
	}

	registry, err := a.container.Get("pluginRegistry")
	if err != nil {
		return fmt.Errorf("failed to get plugin registry: %w", err)
	}

	pluginRegistry := registry.(plugins.PluginRegistry)

	hookManager, err := a.container.Get("hookManager")
	if err != nil {
		return fmt.Errorf("failed to get hook manager: %w", err)
	}

	stepRegistry, err := a.container.Get("stepRegistry")
	if err != nil {
		return fmt.Errorf("failed to get step registry: %w", err)
	}

	cfg := a.container.GetConfiguration()
	pipelineConfig := ConvertConfigToPipelinesConfig(cfg)

	// Build the Layer 1 capability bundle exposed to plugins (replaces the
	// retired interfaces.Pipeline handoff — see Container.BuildCapabilities
	// and plugins.PluginContext.GetCapabilities()).
	pluginCapabilities := BuildCapabilities(daggerClient, pipelineConfig)

	pluginCtx := plugins.NewPluginContext(
		daggerClient,
		cfg,
		hookManager.(interfaces.HookManager),
		stepRegistry.(interfaces.StepRegistry),
		pluginCapabilities,
		providerRegistry,
		pipelineConfig,
		nil, // Logger is accessed via logger.L() directly
	)

	if err := pluginRegistry.LoadBuiltinPlugins(ctx, pluginCtx); err != nil {
		return fmt.Errorf("failed to load builtin plugins: %w", err)
	}

	logger.L().InfoContext(ctx, "Plugins loaded and initialized", "count", len(pluginRegistry.ListPlugins()))

	return nil
}

// CleanupPlugins cleans up all loaded plugins. Callers invoke it through
// defer so it also runs on a failed workflow, not only on the success path.
func (a *App) CleanupPlugins(ctx context.Context) error {
	registry, err := a.container.Get("pluginRegistry")
	if err != nil {
		return fmt.Errorf("failed to get plugin registry: %w", err)
	}

	pluginRegistry := registry.(plugins.PluginRegistry)

	if err := pluginRegistry.CleanupAll(ctx); err != nil {
		return fmt.Errorf("failed to cleanup plugins: %w", err)
	}

	return nil
}
