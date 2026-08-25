package plugins

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/pipelines"
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
		pluginCtx := NewMockPluginContext()
		hookManager := NewMockHookManager()
		stepRegistry := NewMockStepRegistry()
		cfg := NewMockConfiguration()

		cfg.GetFunc = func(key string) any {
			if key == "plugins.nomad-deploy" {
				return nil
			}
			return nil
		}
		hookManager.RegisterHookFunc = func(stepName string, hookType interfaces.HookType, hook interfaces.HookFunc) error {
			return nil
		}
		stepRegistry.RegisterStepFunc = func(stepName string, handler interfaces.StepHandler) error {
			return nil
		}

		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }
		pluginCtx.GetHookManagerFunc = func() interfaces.HookManager { return hookManager }
		pluginCtx.GetStepRegistryFunc = func() interfaces.StepRegistry { return stepRegistry }

		// Act
		err := plugin.Initialize(ctx, pluginCtx)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("LoadsConfigurationFromConfig", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		plugin := NewNomadDeployPlugin().(*NomadDeployPlugin)
		pluginCtx := NewMockPluginContext()
		hookManager := NewMockHookManager()
		stepRegistry := NewMockStepRegistry()
		cfg := NewMockConfiguration()

		configMap := map[string]interface{}{
			"nomad_addr":     "http://custom-nomad:4646",
			"nomad_job_file": "custom.hcl",
			"auto_deploy":    true,
		}
		cfg.GetFunc = func(key string) any {
			if key == "plugins.nomad-deploy" {
				return configMap
			}
			return nil
		}
		hookManager.RegisterHookFunc = func(stepName string, hookType interfaces.HookType, hook interfaces.HookFunc) error {
			return nil
		}
		stepRegistry.RegisterStepFunc = func(stepName string, handler interfaces.StepHandler) error {
			return nil
		}

		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }
		pluginCtx.GetHookManagerFunc = func() interfaces.HookManager { return hookManager }
		pluginCtx.GetStepRegistryFunc = func() interfaces.StepRegistry { return stepRegistry }

		// Act
		err := plugin.Initialize(ctx, pluginCtx)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "http://custom-nomad:4646", plugin.getConfigString("nomad_addr", ""))
		assert.Equal(t, "custom.hcl", plugin.getConfigString("nomad_job_file", ""))
	})

	t.Run("ReturnsErrorWhenHookRegistrationFails", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		plugin := NewNomadDeployPlugin().(*NomadDeployPlugin)
		pluginCtx := NewMockPluginContext()
		hookManager := NewMockHookManager()
		cfg := NewMockConfiguration()

		cfg.GetFunc = func(key string) any {
			if key == "plugins.nomad-deploy" {
				return nil
			}
			return nil
		}
		hookManager.RegisterHookFunc = func(stepName string, hookType interfaces.HookType, hook interfaces.HookFunc) error {
			return errors.New("hook registration failed")
		}

		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }
		pluginCtx.GetHookManagerFunc = func() interfaces.HookManager { return hookManager }

		// Act
		err := plugin.Initialize(ctx, pluginCtx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register deploy hook")
	})

	t.Run("ReturnsErrorWhenStepRegistrationFails", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		plugin := NewNomadDeployPlugin().(*NomadDeployPlugin)
		pluginCtx := NewMockPluginContext()
		hookManager := NewMockHookManager()
		stepRegistry := NewMockStepRegistry()
		cfg := NewMockConfiguration()

		cfg.GetFunc = func(key string) any {
			if key == "plugins.nomad-deploy" {
				return nil
			}
			return nil
		}
		hookManager.RegisterHookFunc = func(stepName string, hookType interfaces.HookType, hook interfaces.HookFunc) error {
			return nil
		}
		stepRegistry.RegisterStepFunc = func(stepName string, handler interfaces.StepHandler) error {
			return errors.New("step registration failed")
		}

		pluginCtx.GetConfigurationFunc = func() interfaces.Configuration { return cfg }
		pluginCtx.GetHookManagerFunc = func() interfaces.HookManager { return hookManager }
		pluginCtx.GetStepRegistryFunc = func() interfaces.StepRegistry { return stepRegistry }

		// Act
		err := plugin.Initialize(ctx, pluginCtx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register deploy-nomad step")
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

func TestNomadDeployStepHandler_CanHandle(t *testing.T) {
	t.Run("ReturnsTrueForDeployNomadStep", func(t *testing.T) {
		// Arrange
		handler := &nomadDeployStepHandler{}

		// Act
		result := handler.CanHandle("deploy-nomad")

		// Assert
		assert.True(t, result)
	})

	t.Run("ReturnsFalseForOtherSteps", func(t *testing.T) {
		// Arrange
		handler := &nomadDeployStepHandler{}

		// Act
		result := handler.CanHandle("other-step")

		// Assert
		assert.False(t, result)
	})
}

func TestNomadDeployStepHandler_GetStepInfo(t *testing.T) {
	t.Run("ReturnsStepConfig", func(t *testing.T) {
		// Arrange
		handler := &nomadDeployStepHandler{}

		// Act
		config := handler.GetStepInfo("deploy-nomad")

		// Assert
		assert.Equal(t, "deploy-nomad", config.Name)
		assert.Equal(t, "Deploy application to Nomad cluster", config.Description)
		assert.False(t, config.Required)
		assert.Contains(t, config.DependsOn, "push")
	})
}

func TestNomadDeployStepHandler_Validate(t *testing.T) {
	t.Run("ValidatesDeployNomadStep", func(t *testing.T) {
		// Arrange
		handler := &nomadDeployStepHandler{}
		config := interfaces.StepConfig{
			Name: "deploy-nomad",
		}

		// Act
		err := handler.Validate("deploy-nomad", config)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("ReturnsErrorForInvalidStepName", func(t *testing.T) {
		// Arrange
		handler := &nomadDeployStepHandler{}
		config := interfaces.StepConfig{
			Name: "other-step",
		}

		// Act
		err := handler.Validate("other-step", config)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid step name")
	})
}

func TestNomadDeployStepHandler_Execute(t *testing.T) {
	t.Run("ReturnsErrorWhenHandlerCannotHandleStep", func(t *testing.T) {
		// Arrange
		handler := &nomadDeployStepHandler{}
		config := interfaces.StepConfig{
			Name: "other-step",
		}

		// Act
		err := handler.Execute(context.Background(), "other-step", config)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handler cannot handle step")
	})

	t.Run("ReturnsErrorWhenDaggerClientNotAvailable", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		pluginCtx := NewMockPluginContext()
		pluginCtx.GetDaggerClientFunc = func() (*dagger.Client, error) {
			return nil, errors.New("dagger client not available")
		}
		handler := &nomadDeployStepHandler{
			config:    make(map[string]interface{}),
			pluginCtx: pluginCtx,
		}
		config := interfaces.StepConfig{
			Name: "deploy-nomad",
		}

		// Act
		err := handler.Execute(ctx, "deploy-nomad", config)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dagger client not available")
	})

	t.Run("CoversExecutePathWithPipelineConfig", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		pluginCtx := NewMockPluginContext()
		// Note: deployToNomad requires a real Dagger engine to test properly.
		// An uninitialized Dagger client will panic when used.
		// This test documents that the Execute path exists (lines 273-288),
		// but full testing requires integration tests with a real Dagger engine.
		pluginCtx.GetDaggerClientFunc = func() (*dagger.Client, error) {
			// Return error to test the error path without triggering Dagger client operations
			return nil, errors.New("dagger client not available for testing")
		}
		pluginCtx.GetPipelineConfigFunc = func() pipelines.Config {
			return pipelines.Config{
				RegistryURL: "registry.example.com",
				ImageName:   "test-service",
				ImageTag:    "v1.0.0",
			}
		}
		handler := &nomadDeployStepHandler{
			config: map[string]interface{}{
				"nomad_job_file": "nomad.hcl",
			},
			pluginCtx: pluginCtx,
		}
		config := interfaces.StepConfig{
			Name: "deploy-nomad",
		}

		// Act
		err := handler.Execute(ctx, "deploy-nomad", config)

		// Assert
		// Should fail because Dagger client is not available
		// This tests the error path before deployToNomad is called
		// The actual deployToNomad error path (lines 284-286) requires a real Dagger engine
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dagger client not available")
	})
}

func TestNomadDeployPlugin_buildImageRefFromConfig(t *testing.T) {
	t.Run("BuildsImageRefFromConfig", func(t *testing.T) {
		// Arrange
		plugin := &NomadDeployPlugin{
			config: make(map[string]interface{}),
		}
		cfg := pipelines.Config{
			RegistryURL: "registry.example.com",
			ImageName:   "my-service",
			ImageTag:    "v1.0.0",
		}

		// Act
		imageRef := plugin.buildImageRefFromConfig(cfg)

		// Assert
		assert.Equal(t, "registry.example.com/my-service:v1.0.0", imageRef)
	})

	t.Run("UsesDefaultsWhenConfigEmpty", func(t *testing.T) {
		// Arrange
		plugin := &NomadDeployPlugin{
			config: make(map[string]interface{}),
		}
		cfg := pipelines.Config{}

		// Act
		imageRef := plugin.buildImageRefFromConfig(cfg)

		// Assert
		// Should use defaults from environment or fallback values
		assert.NotEmpty(t, imageRef)
	})

	t.Run("UsesRegistryWhenRegistryURLEmpty", func(t *testing.T) {
		// Arrange
		plugin := &NomadDeployPlugin{
			config: make(map[string]interface{}),
		}
		cfg := pipelines.Config{
			Registry:  "registry.example.com",
			ImageName: "my-service",
			ImageTag:  "v1.0.0",
		}

		// Act
		imageRef := plugin.buildImageRefFromConfig(cfg)

		// Assert
		assert.Equal(t, "registry.example.com/my-service:v1.0.0", imageRef)
	})

	t.Run("UsesBuildTagWhenImageTagEmpty", func(t *testing.T) {
		// Arrange
		plugin := &NomadDeployPlugin{
			config: make(map[string]interface{}),
		}
		cfg := pipelines.Config{
			RegistryURL: "registry.example.com",
			ImageName:   "my-service",
			BuildTag:    "v1.0.0",
		}

		// Act
		imageRef := plugin.buildImageRefFromConfig(cfg)

		// Assert
		assert.Equal(t, "registry.example.com/my-service:v1.0.0", imageRef)
	})

	t.Run("UsesEnvironmentVariablesWhenConfigEmpty", func(t *testing.T) {
		// Arrange
		plugin := &NomadDeployPlugin{
			config: make(map[string]interface{}),
		}
		cfg := pipelines.Config{}

		// Set environment variables
		originalRegistry := os.Getenv("REGISTRY_URL")
		originalImageName := os.Getenv("IMAGE_NAME")
		originalImageTag := os.Getenv("IMAGE_TAG")
		defer func() {
			if originalRegistry != "" {
				os.Setenv("REGISTRY_URL", originalRegistry)
			} else {
				os.Unsetenv("REGISTRY_URL")
			}
			if originalImageName != "" {
				os.Setenv("IMAGE_NAME", originalImageName)
			} else {
				os.Unsetenv("IMAGE_NAME")
			}
			if originalImageTag != "" {
				os.Setenv("IMAGE_TAG", originalImageTag)
			} else {
				os.Unsetenv("IMAGE_TAG")
			}
		}()

		os.Setenv("REGISTRY_URL", "env-registry.example.com")
		os.Setenv("IMAGE_NAME", "env-service")
		os.Setenv("IMAGE_TAG", "env-tag")

		// Act
		imageRef := plugin.buildImageRefFromConfig(cfg)

		// Assert
		assert.Equal(t, "env-registry.example.com/env-service:env-tag", imageRef)
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
