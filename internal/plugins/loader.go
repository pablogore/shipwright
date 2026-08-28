package plugins

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sort"

	"github.com/pablogore/kit-logger/pkg/logger"
)

// loader implements PluginLoader interface.
type loader struct {
	builtinPlugins map[string]func() Plugin
}

// NewLoader creates a new plugin loader.
func NewLoader() PluginLoader {
	return &loader{
		builtinPlugins: make(map[string]func() Plugin),
	}
}

// RegisterBuiltin registers a built-in plugin factory.
func (l *loader) RegisterBuiltin(name string, factory func() Plugin) {
	l.builtinPlugins[name] = factory
}

// ListBuiltins returns every compile-time-registered builtin plugin name.
// Sorted so the load order LoadBuiltinPlugins produces is deterministic —
// plugin initialization registers providers, and a non-deterministic order
// would make a duplicate-registration bug intermittent instead of reliable.
func (l *loader) ListBuiltins() []string {
	names := make([]string, 0, len(l.builtinPlugins))
	for name := range l.builtinPlugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// LoadFromFile loads a plugin from a Go plugin file (.so).
// Note: Go plugins require building with `go build -buildmode=plugin`.
func (l *loader) LoadFromFile(ctx context.Context, filePath string) (Plugin, error) {
	if filePath == "" {
		return nil, errors.New("file path cannot be empty")
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve plugin path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin file not found: %s", absPath)
	}

	// Load plugin
	p, err := plugin.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for Plugin symbol
	symbol, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export Plugin symbol: %w", err)
	}

	// Type assert to Plugin
	pluginInstance, ok := symbol.(Plugin)
	if !ok {
		return nil, errors.New("plugin does not implement Plugin interface")
	}

	logger.L().InfoContext(ctx, "Plugin loaded from file", "path", absPath, "name", pluginInstance.Name())

	return pluginInstance, nil
}

// LoadFromConfig loads a plugin from configuration.
// Configuration format:
//
//	plugins:
//	  plugin-name:
//	    type: builtin|file
//	    path: /path/to/plugin.so  # for file type
//	    config:
//	      key: value
func (l *loader) LoadFromConfig(ctx context.Context, config map[string]interface{}) (Plugin, error) {
	if config == nil {
		return nil, errors.New("plugin configuration cannot be nil")
	}

	pluginType, ok := config["type"].(string)
	if !ok {
		pluginType = "builtin" // Default to builtin
	}

	switch pluginType {
	case "builtin":
		name, ok := config["name"].(string)
		if !ok {
			return nil, errors.New("builtin plugin requires 'name' field")
		}
		return l.LoadBuiltin(ctx, name)

	case "file":
		path, ok := config["path"].(string)
		if !ok {
			return nil, errors.New("file plugin requires 'path' field")
		}
		return l.LoadFromFile(ctx, path)

	default:
		return nil, fmt.Errorf("unknown plugin type: %s", pluginType)
	}
}

// LoadBuiltin loads a built-in plugin by name.
func (l *loader) LoadBuiltin(ctx context.Context, name string) (Plugin, error) {
	if name == "" {
		return nil, errors.New("plugin name cannot be empty")
	}

	factory, exists := l.builtinPlugins[name]
	if !exists {
		return nil, fmt.Errorf("builtin plugin %s not found", name)
	}

	pluginInstance := factory()
	if pluginInstance == nil {
		return nil, fmt.Errorf("builtin plugin %s factory returned nil", name)
	}

	logger.L().InfoContext(ctx, "Builtin plugin loaded", "name", name)

	return pluginInstance, nil
}
