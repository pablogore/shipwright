package plugins

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/pablogore/kit-logger/pkg/logger"
)

// registry implements PluginRegistry interface.
type registry struct {
	loader  PluginLoader
	mutex   sync.RWMutex
	plugins map[string]Plugin
}

// NewRegistry creates a new plugin registry.
func NewRegistry(loader PluginLoader) PluginRegistry {
	return &registry{
		loader:  loader,
		plugins: make(map[string]Plugin),
	}
}

// RegisterPlugin registers a plugin with the registry.
func (r *registry) RegisterPlugin(plugin Plugin) error {
	if plugin == nil {
		return errors.New("plugin cannot be nil")
	}

	name := plugin.Name()
	if name == "" {
		return errors.New("plugin name cannot be empty")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s is already registered", name)
	}

	r.plugins[name] = plugin
	logger.L().InfoContext(context.Background(), "Plugin registered", "name", name, "version", plugin.Version())

	return nil
}

// LoadPlugin loads a plugin by name from configuration or file.
func (r *registry) LoadPlugin(ctx context.Context, name string, pluginCtx PluginContext) error {
	if name == "" {
		return errors.New("plugin name cannot be empty")
	}

	// Check if plugin is already loaded
	r.mutex.RLock()
	if _, exists := r.plugins[name]; exists {
		r.mutex.RUnlock()
		return fmt.Errorf("plugin %s is already loaded", name)
	}
	r.mutex.RUnlock()

	// Try to load from builtin plugins first
	if r.loader != nil {
		plugin, err := r.loader.LoadBuiltin(ctx, name)
		if err == nil && plugin != nil {
			return r.registerAndInitialize(ctx, plugin, pluginCtx)
		}
	}

	// Try to load from configuration
	if pluginCtx != nil {
		cfg := pluginCtx.GetConfiguration()
		if pluginsConfig := cfg.Get("plugins"); pluginsConfig != nil {
			if pluginsMap, ok := pluginsConfig.(map[string]interface{}); ok {
				if pluginConfig, exists := pluginsMap[name]; exists {
					if pluginMap, ok := pluginConfig.(map[string]interface{}); ok && r.loader != nil {
						plugin, err := r.loader.LoadFromConfig(ctx, pluginMap)
						if err == nil && plugin != nil {
							return r.registerAndInitialize(ctx, plugin, pluginCtx)
						}
					}
				}
			}
		}
	}

	return fmt.Errorf("plugin %s not found", name)
}

// LoadPluginsFromConfig loads all plugins specified in configuration.
func (r *registry) LoadPluginsFromConfig(ctx context.Context, pluginCtx PluginContext) error {
	if pluginCtx == nil {
		return errors.New("plugin context cannot be nil")
	}

	cfg := pluginCtx.GetConfiguration()
	pluginsConfig := cfg.Get("plugins")

	if pluginsConfig == nil {
		// No plugins configured, this is OK
		return nil
	}

	pluginsMap, ok := pluginsConfig.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid plugins configuration format")
	}

	var errors []error
	for name := range pluginsMap {
		if err := r.LoadPlugin(ctx, name, pluginCtx); err != nil {
			logger.L().ErrorContext(ctx, "Failed to load plugin", "name", name, "error", err)
			errors = append(errors, fmt.Errorf("failed to load plugin %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load some plugins: %v", errors)
	}

	return nil
}

// registerAndInitialize registers and initializes a plugin.
func (r *registry) registerAndInitialize(ctx context.Context, plugin Plugin, pluginCtx PluginContext) error {
	if err := r.RegisterPlugin(plugin); err != nil {
		return fmt.Errorf("failed to register plugin: %w", err)
	}

	if err := plugin.Initialize(ctx, pluginCtx); err != nil {
		// Remove plugin if initialization fails
		r.mutex.Lock()
		delete(r.plugins, plugin.Name())
		r.mutex.Unlock()
		return fmt.Errorf("failed to initialize plugin %s: %w", plugin.Name(), err)
	}

	logger.L().InfoContext(ctx, "Plugin initialized", "name", plugin.Name(), "version", plugin.Version())

	return nil
}

// GetPlugin retrieves a registered plugin by name.
func (r *registry) GetPlugin(name string) (Plugin, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// ListPlugins returns all registered plugin names.
func (r *registry) ListPlugins() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}

	return names
}

// InitializeAll initializes all registered plugins.
func (r *registry) InitializeAll(ctx context.Context, pluginCtx PluginContext) error {
	r.mutex.RLock()
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	r.mutex.RUnlock()

	var errors []error
	for _, plugin := range plugins {
		if err := plugin.Initialize(ctx, pluginCtx); err != nil {
			logger.L().ErrorContext(ctx, "Failed to initialize plugin", "name", plugin.Name(), "error", err)
			errors = append(errors, fmt.Errorf("plugin %s: %w", plugin.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to initialize some plugins: %v", errors)
	}

	return nil
}

// CleanupAll cleans up all registered plugins.
func (r *registry) CleanupAll(ctx context.Context) error {
	r.mutex.RLock()
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	r.mutex.RUnlock()

	var errors []error
	for _, plugin := range plugins {
		if err := plugin.Cleanup(ctx); err != nil {
			logger.L().ErrorContext(ctx, "Failed to cleanup plugin", "name", plugin.Name(), "error", err)
			errors = append(errors, fmt.Errorf("plugin %s: %w", plugin.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to cleanup some plugins: %v", errors)
	}

	return nil
}
