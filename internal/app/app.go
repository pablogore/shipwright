package app

import (
	"context"
	"fmt"
	"sync"

	"dagger.io/dagger"

	"github.com/pablogore/kit-logger/pkg/logger"
	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/plugins"
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
			// Logger initialization error is non-fatal, continue with default logger
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

// loadAndInitializePlugins loads and initializes plugins for the pipeline.
//
// Unreachable from any CLI path as of this work unit: it was previously
// called only from the now-deleted App.RunPipeline (tasks.md 11's preset
// deletion). Left in place, along with cleanupPlugins below, because both
// are the Layer-1-based plugin/capability wiring WU10 built
// (Container.BuildCapabilities, plugins.Capabilities,
// PluginContext.GetCapabilities/GetConfig) — not preset-specific surface —
// and this work unit's scope is limited to removing the preset registry and
// its CLI flags. Flagged for sdd-verify: this method (and cleanupPlugins)
// currently has no production caller and is a candidate for either
// reintegration with the --workflow entrypoint or removal in a later work
// unit.
func (a *App) loadAndInitializePlugins(ctx context.Context, pipelineName string) error {
	// Get plugin registry
	registry, err := a.container.Get("pluginRegistry")
	if err != nil {
		return fmt.Errorf("failed to get plugin registry: %w", err)
	}

	pluginRegistry := registry.(plugins.PluginRegistry)

	// Get Dagger client (may be nil)
	var daggerClient *dagger.Client
	client, err := a.container.Get("daggerClient")
	if err == nil && client != nil {
		daggerClient = client.(*dagger.Client)
	}

	// Get hook manager
	hookManager, err := a.container.Get("hookManager")
	if err != nil {
		return fmt.Errorf("failed to get hook manager: %w", err)
	}

	// Get step registry
	stepRegistry, err := a.container.Get("stepRegistry")
	if err != nil {
		return fmt.Errorf("failed to get step registry: %w", err)
	}

	// Get pipeline config
	cfg := a.container.GetConfiguration()
	pipelineConfig := ConvertConfigToPipelinesConfig(cfg)

	// Build the Layer 1 capability bundle exposed to plugins (replaces the
	// retired interfaces.Pipeline handoff — see Container.BuildCapabilities
	// and plugins.PluginContext.GetCapabilities()).
	pluginCapabilities := BuildCapabilities(daggerClient, pipelineConfig)

	// Get logger (using go-kit-logger directly)
	// Create plugin context
	pluginCtx := plugins.NewPluginContext(
		daggerClient,
		cfg,
		hookManager.(interfaces.HookManager),
		stepRegistry.(interfaces.StepRegistry),
		pluginCapabilities,
		pipelineConfig,
		nil, // Logger is accessed via logger.L() directly
	)

	// Load plugins from configuration
	if err := pluginRegistry.LoadPluginsFromConfig(ctx, pluginCtx); err != nil {
		return fmt.Errorf("failed to load plugins from config: %w", err)
	}

	logger.L().InfoContext(ctx, "Plugins loaded and initialized", "count", len(pluginRegistry.ListPlugins()))

	return nil
}

// cleanupPlugins cleans up all loaded plugins.
func (a *App) cleanupPlugins(ctx context.Context) error {
	// Get plugin registry
	registry, err := a.container.Get("pluginRegistry")
	if err != nil {
		return fmt.Errorf("failed to get plugin registry: %w", err)
	}

	pluginRegistry := registry.(plugins.PluginRegistry)

	// Cleanup all plugins
	if err := pluginRegistry.CleanupAll(ctx); err != nil {
		return fmt.Errorf("failed to cleanup plugins: %w", err)
	}

	return nil
}
