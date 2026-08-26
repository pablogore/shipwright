package plugins

import (
	"errors"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/pipelines"
)

// pluginContext implements PluginContext interface.
type pluginContext struct {
	configuration  interfaces.Configuration
	daggerClient   *dagger.Client
	hookManager    interfaces.HookManager
	logger         interfaces.Logger
	capabilities   Capabilities
	pipelineConfig pipelines.Config
	stepRegistry   interfaces.StepRegistry
}

// NewPluginContext creates a new plugin context.
func NewPluginContext(
	daggerClient *dagger.Client,
	configuration interfaces.Configuration,
	hookManager interfaces.HookManager,
	stepRegistry interfaces.StepRegistry,
	capabilities Capabilities,
	pipelineConfig pipelines.Config,
	logger interfaces.Logger,
) PluginContext {
	return &pluginContext{
		configuration:  configuration,
		daggerClient:   daggerClient,
		hookManager:    hookManager,
		logger:         logger,
		capabilities:   capabilities,
		pipelineConfig: pipelineConfig,
		stepRegistry:   stepRegistry,
	}
}

// GetDaggerClient returns the Dagger client for container operations.
func (pc *pluginContext) GetDaggerClient() (*dagger.Client, error) {
	if pc.daggerClient == nil {
		return nil, errors.New("dagger client not available")
	}
	return pc.daggerClient, nil
}

// GetConfiguration returns the pipeline configuration.
func (pc *pluginContext) GetConfiguration() interfaces.Configuration {
	return pc.configuration
}

// GetHookManager returns the hook manager for registering hooks.
func (pc *pluginContext) GetHookManager() interfaces.HookManager {
	return pc.hookManager
}

// GetStepRegistry returns the step registry for adding custom steps.
func (pc *pluginContext) GetStepRegistry() interfaces.StepRegistry {
	return pc.stepRegistry
}

// GetCapabilities returns the Layer 1 capability bundle wired for the
// current run.
func (pc *pluginContext) GetCapabilities() Capabilities {
	return pc.capabilities
}

// GetConfig returns the pipeline-specific configuration.
func (pc *pluginContext) GetConfig() pipelines.Config {
	return pc.pipelineConfig
}

// GetLogger returns the logger instance.
func (pc *pluginContext) GetLogger() interfaces.Logger {
	return pc.logger
}
