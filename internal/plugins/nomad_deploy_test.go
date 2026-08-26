package plugins

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/workflow/providers"
)

func TestNewNomadDeployPlugin(t *testing.T) {
	t.Run("CreatesNomadDeployPlugin", func(t *testing.T) {
		// Act
		plugin := NewNomadDeployPlugin()

		// Assert
		require.NotNil(t, plugin)
		assert.Implements(t, (*Plugin)(nil), plugin)
		assert.Equal(t, "nomad-deploy", plugin.Name())
		assert.Equal(t, "1.0.0", plugin.Version())
	})
}

func TestNomadDeployPlugin_Name(t *testing.T) {
	t.Run("ReturnsPluginName", func(t *testing.T) {
		// Arrange
		plugin := NewNomadDeployPlugin().(*NomadDeployPlugin)

		// Act
		name := plugin.Name()

		// Assert
		assert.Equal(t, "nomad-deploy", name)
	})
}

func TestNomadDeployPlugin_Version(t *testing.T) {
	t.Run("ReturnsPluginVersion", func(t *testing.T) {
		// Arrange
		plugin := NewNomadDeployPlugin().(*NomadDeployPlugin)

		// Act
		version := plugin.Version()

		// Assert
		assert.Equal(t, "1.0.0", version)
	})
}

func TestNomadDeployPlugin_Initialize(t *testing.T) {
	t.Run("InitializesPluginSuccessfully", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		plugin := NewNomadDeployPlugin().(*NomadDeployPlugin)
		cfg := NewMockConfiguration()
		cfg.GetFunc = func(string) any { return nil }

		pluginCtx := NewMockPluginContext()
		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }
		pluginCtx.GetProviderRegistryFunc = func() *providers.Registry { return providers.NewRegistry() }

		// Act
		err := plugin.Initialize(ctx, pluginCtx)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("LoadsConfigurationFromConfig", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		plugin := NewNomadDeployPlugin().(*NomadDeployPlugin)
		cfg := NewMockConfiguration()

		configMap := map[string]interface{}{
			"nomad_addr":     "http://custom-nomad:4646",
			"nomad_job_file": "custom.hcl",
		}
		cfg.GetFunc = func(key string) any {
			if key == "plugins.nomad-deploy" {
				return configMap
			}
			return nil
		}

		pluginCtx := NewMockPluginContext()
		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }
		pluginCtx.GetProviderRegistryFunc = func() *providers.Registry { return providers.NewRegistry() }

		// Act
		err := plugin.Initialize(ctx, pluginCtx)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "http://custom-nomad:4646", plugin.getConfigString("nomad_addr", ""))
		assert.Equal(t, "custom.hcl", plugin.getConfigString("nomad_job_file", ""))
	})

	// PluginConfigurationFeedsProviderDefaults proves plugin configuration is
	// not merely stored: it becomes the fallback for a `with` field the
	// manifest step omits.
	t.Run("PluginConfigurationFeedsProviderDefaults", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		reg := providers.NewRegistry()
		cfg := NewMockConfiguration()
		cfg.GetFunc = func(key string) any {
			if key == "plugins.nomad-deploy" {
				return map[string]interface{}{"nomad_addr": "http://configured:4646"}
			}
			return nil
		}

		pluginCtx := NewMockPluginContext()
		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }
		pluginCtx.GetProviderRegistryFunc = func() *providers.Registry { return reg }

		// Act
		require.NoError(t, NewNomadDeployPlugin().Initialize(ctx, pluginCtx))
		deployer, err := reg.ResolveDeployer(deployRef, providers.Values{})

		// Assert
		require.NoError(t, err)
		np, ok := deployer.(*NomadDeployPlugin)
		require.True(t, ok)
		assert.Equal(t, "http://configured:4646", np.nomadAddr)
	})

	t.Run("ToleratesNilConfiguration", func(t *testing.T) {
		// Arrange
		pluginCtx := NewMockPluginContext()
		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return nil }
		pluginCtx.GetProviderRegistryFunc = func() *providers.Registry { return providers.NewRegistry() }

		// Act / Assert
		assert.NoError(t, NewNomadDeployPlugin().Initialize(context.Background(), pluginCtx))
	})
}

func TestNomadDeployPlugin_Cleanup(t *testing.T) {
	t.Run("CleansUpSuccessfully", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		plugin := NewNomadDeployPlugin().(*NomadDeployPlugin)

		// Act
		err := plugin.Cleanup(ctx)

		// Assert
		assert.NoError(t, err)
	})
}

func TestNomadDeployPlugin_getConfigString(t *testing.T) {
	t.Run("ReturnsConfigValueWhenPresent", func(t *testing.T) {
		// Arrange
		plugin := &NomadDeployPlugin{
			config: map[string]interface{}{
				"test_key": "test_value",
			},
		}

		// Act
		result := plugin.getConfigString("test_key", "default")

		// Assert
		assert.Equal(t, "test_value", result)
	})

	t.Run("ReturnsDefaultWhenKeyMissing", func(t *testing.T) {
		// Arrange
		plugin := &NomadDeployPlugin{
			config: make(map[string]interface{}),
		}

		// Act
		result := plugin.getConfigString("missing_key", "default_value")

		// Assert
		assert.Equal(t, "default_value", result)
	})

	t.Run("ReturnsDefaultWhenValueIsEmpty", func(t *testing.T) {
		// Arrange
		plugin := &NomadDeployPlugin{
			config: map[string]interface{}{
				"empty_key": "",
			},
		}

		// Act
		result := plugin.getConfigString("empty_key", "default_value")

		// Assert
		assert.Equal(t, "default_value", result)
	})
}

func TestValueOr(t *testing.T) {
	tests := []struct {
		name     string
		values   providers.Values
		key      string
		fallback string
		want     string
	}{
		{name: "absent field uses fallback", values: providers.Values{}, key: "jobFile", fallback: "nomad.hcl", want: "nomad.hcl"},
		{name: "present field wins", values: providers.Values{"jobFile": mustString("api.hcl")}, key: "jobFile", fallback: "nomad.hcl", want: "api.hcl"},
		{name: "empty value uses fallback", values: providers.Values{"jobFile": mustString("")}, key: "jobFile", fallback: "nomad.hcl", want: "nomad.hcl"},
		{name: "secret value never becomes a plain setting", values: providers.Values{"jobFile": mustSecret()}, key: "jobFile", fallback: "nomad.hcl", want: "nomad.hcl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, valueOr(tt.values, tt.key, tt.fallback))
		})
	}
}
