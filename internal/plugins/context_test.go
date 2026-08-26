package plugins

import (
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

func TestNewPluginContext(t *testing.T) {
	t.Run("CreatesPluginContextWithAllFields", func(t *testing.T) {
		// Arrange
		cfg := NewMockConfiguration()
		hookManager := NewMockHookManager()
		stepRegistry := NewMockStepRegistry()
		caps := Capabilities{}
		pipelineConfig := pipelines.Config{}

		// Act
		ctx := NewPluginContext(nil, cfg, hookManager, stepRegistry, caps, nil, pipelineConfig, nil)

		// Assert
		require.NotNil(t, ctx)
		assert.Equal(t, cfg, ctx.GetConfiguration())
		assert.Equal(t, hookManager, ctx.GetHookManager())
		assert.Equal(t, stepRegistry, ctx.GetStepRegistry())
		assert.Equal(t, caps, ctx.GetCapabilities())
		assert.Equal(t, pipelineConfig, ctx.GetConfig())
	})
}

func TestPluginContext_GetDaggerClient(t *testing.T) {
	t.Run("ReturnsErrorWhenClientIsNil", func(t *testing.T) {
		// Arrange
		ctx := NewPluginContext(nil, nil, nil, nil, Capabilities{}, nil, pipelines.Config{}, nil)

		// Act
		client, err := ctx.GetDaggerClient()

		// Assert
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "dagger client not available")
	})

	t.Run("ReturnsClientWhenAvailable", func(t *testing.T) {
		// Arrange
		// Create a non-nil Dagger client pointer (even though it's not functional)
		// This covers the return path at line 49 when client is not nil
		daggerClient := &dagger.Client{}
		ctx := NewPluginContext(daggerClient, nil, nil, nil, Capabilities{}, nil, pipelines.Config{}, nil)

		// Act
		client, err := ctx.GetDaggerClient()

		// Assert
		// Should return the client without error (covers line 49)
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, daggerClient, client)
	})
}

func TestPluginContext_GetConfiguration(t *testing.T) {
	t.Run("ReturnsConfiguration", func(t *testing.T) {
		// Arrange
		cfg := NewMockConfiguration()
		ctx := NewPluginContext(nil, cfg, nil, nil, Capabilities{}, nil, pipelines.Config{}, nil)

		// Act
		result := ctx.GetConfiguration()

		// Assert
		assert.Equal(t, cfg, result)
	})
}

func TestPluginContext_GetHookManager(t *testing.T) {
	t.Run("ReturnsHookManager", func(t *testing.T) {
		// Arrange
		hookManager := NewMockHookManager()
		ctx := NewPluginContext(nil, nil, hookManager, nil, Capabilities{}, nil, pipelines.Config{}, nil)

		// Act
		result := ctx.GetHookManager()

		// Assert
		assert.Equal(t, hookManager, result)
	})
}

func TestPluginContext_GetStepRegistry(t *testing.T) {
	t.Run("ReturnsStepRegistry", func(t *testing.T) {
		// Arrange
		stepRegistry := NewMockStepRegistry()
		ctx := NewPluginContext(nil, nil, nil, stepRegistry, Capabilities{}, nil, pipelines.Config{}, nil)

		// Act
		result := ctx.GetStepRegistry()

		// Assert
		assert.Equal(t, stepRegistry, result)
	})
}

func TestPluginContext_GetCapabilities(t *testing.T) {
	t.Run("ReturnsCapabilities", func(t *testing.T) {
		// Arrange
		caps := Capabilities{Testers: []shipwright.Tester{}}
		ctx := NewPluginContext(nil, nil, nil, nil, caps, nil, pipelines.Config{}, nil)

		// Act
		result := ctx.GetCapabilities()

		// Assert
		assert.Equal(t, caps, result)
	})
}

func TestPluginContext_GetConfig(t *testing.T) {
	t.Run("ReturnsConfig", func(t *testing.T) {
		// Arrange
		expectedConfig := pipelines.Config{
			Env: "test",
		}
		ctx := NewPluginContext(nil, nil, nil, nil, Capabilities{}, nil, expectedConfig, nil)

		// Act
		result := ctx.GetConfig()

		// Assert
		assert.Equal(t, expectedConfig, result)
	})
}

func TestPluginContext_GetLogger(t *testing.T) {
	t.Run("ReturnsLogger", func(t *testing.T) {
		// Arrange
		logger := NewMockLogger()
		ctx := NewPluginContext(nil, nil, nil, nil, Capabilities{}, nil, pipelines.Config{}, logger)

		// Act
		result := ctx.GetLogger()

		// Assert
		assert.Equal(t, logger, result)
	})
}
