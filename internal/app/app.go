package app

import (
	"context"
	"fmt"
	"sync"

	"dagger.io/dagger"

	"github.com/getsyntegrity/go-kit-logger/pkg/logger"
	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
	"github.com/getsyntegrity/syntegrity-dagger/internal/plugins"
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
	logger.L().InfoContext(ctx, "Starting Syntegrity Dagger application...")
	return a.container.Start(ctx)
}

// Stop stops the application.
func (a *App) Stop(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Stopping Syntegrity Dagger application...")
	return a.container.Stop(ctx)
}

// GetContainer returns the application's container.
func (a *App) GetContainer() *Container {
	return a.container
}

// RunPipeline executes a pipeline with the given name.
func (a *App) RunPipeline(ctx context.Context, pipelineName string) error {
	logger.L().InfoContext(ctx, "Running pipeline", "name", pipelineName)

	// Get pipeline once to determine available steps
	initialPipeline, err := a.container.GetPipeline(pipelineName)
	if err != nil {
		return fmt.Errorf("failed to get pipeline %s: %w", pipelineName, err)
	}

	// Load and initialize plugins before executing pipeline
	if err := a.loadAndInitializePlugins(ctx, pipelineName, initialPipeline); err != nil {
		logger.L().WarnContext(ctx, "Failed to load plugins, continuing without plugins", "error", err)
		// Don't fail the pipeline if plugins fail to load
	}

	// Execute pipeline steps
	steps := initialPipeline.GetAvailableSteps()

	for _, step := range steps {
		logger.L().InfoContext(ctx, "Executing pipeline step", "step", step)

		// Get a fresh pipeline instance before each step to ensure
		// we have a valid Dagger client (connection may be lost between steps)
		pipeline, err := a.container.GetPipeline(pipelineName)
		if err != nil {
			return fmt.Errorf("failed to get pipeline %s for step %s: %w", pipelineName, step, err)
		}

		stepErr := pipeline.ExecuteStep(ctx, step)
		if stepErr != nil {
			return fmt.Errorf("pipeline step %s failed: %w", step, stepErr)
		}

		logger.L().InfoContext(ctx, "Pipeline step completed", "step", step)
	}

	logger.L().InfoContext(ctx, "Pipeline completed successfully", "name", pipelineName)

	// Cleanup plugins after pipeline execution
	if err := a.cleanupPlugins(ctx); err != nil {
		logger.L().WarnContext(ctx, "Failed to cleanup plugins", "error", err)
		// Don't fail the pipeline if plugin cleanup fails
	}

	return nil
}

// RunPipelineStep executes a single pipeline step.
func (a *App) RunPipelineStep(ctx context.Context, pipelineName, step string) error {
	logger.L().InfoContext(ctx, "Running pipeline step", "pipeline", pipelineName, "step", step)

	pipeline, err := a.container.GetPipeline(pipelineName)
	if err != nil {
		return fmt.Errorf("failed to get pipeline %s: %w", pipelineName, err)
	}

	// Execute single step
	stepErr := pipeline.ExecuteStep(ctx, step)
	if stepErr != nil {
		return fmt.Errorf("pipeline step %s failed: %w", step, stepErr)
	}

	logger.L().InfoContext(ctx, "Pipeline step completed successfully", "pipeline", pipelineName, "step", step)
	return nil
}

// ListPipelines returns a list of available pipelines.
func (a *App) ListPipelines() ([]string, error) {
	registry, err := a.container.GetPipelineRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline registry: %w", err)
	}

	return registry.List(), nil
}

// GetPipelineInfo returns information about a specific pipeline.
func (a *App) GetPipelineInfo(pipelineName string) (map[string]any, error) {
	pipeline, err := a.container.GetPipeline(pipelineName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline %s: %w", pipelineName, err)
	}

	info := map[string]any{
		"name": pipeline.Name(),
		"steps": []string{
			"setup", "build", "test", "tag", "package", "push", "lint", "security", "release",
		},
	}

	return info, nil
}

// loadAndInitializePlugins loads and initializes plugins for the pipeline.
func (a *App) loadAndInitializePlugins(ctx context.Context, pipelineName string, pipeline interfaces.Pipeline) error {
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

	// Get logger (using go-kit-logger directly)
	// Create plugin context
	pluginCtx := plugins.NewPluginContext(
		daggerClient,
		cfg,
		hookManager.(interfaces.HookManager),
		stepRegistry.(interfaces.StepRegistry),
		pipeline,
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
