package plugins

import (
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/internal/pipelines"
)

func TestNewPluginContext(t *testing.T) {
	t.Run("CreatesPluginContextWithAllFields", func(t *testing.T) {
		// Arrange
		cfg := NewMockConfiguration()
		hookManager := NewMockHookManager()
		stepRegistry := NewMockStepRegistry()
		pipeline := NewMockPipeline()
		pipelineConfig := pipelines.Config{}

		// Act
		ctx := NewPluginContext(nil, cfg, hookManager, stepRegistry, pipeline, pipelineConfig, nil)

		// Assert
		require.NotNil(t, ctx)
		assert.Equal(t, cfg, ctx.GetConfiguration())
		assert.Equal(t, hookManager, ctx.GetHookManager())
		assert.Equal(t, stepRegistry, ctx.GetStepRegistry())
		assert.Equal(t, pipeline, ctx.GetPipeline())
		assert.Equal(t, pipelineConfig, ctx.GetPipelineConfig())
	})
}

func TestPluginContext_GetDaggerClient(t *testing.T) {
	t.Run("ReturnsErrorWhenClientIsNil", func(t *testing.T) {
		// Arrange
		ctx := NewPluginContext(nil, nil, nil, nil, nil, pipelines.Config{}, nil)

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
		ctx := NewPluginContext(daggerClient, nil, nil, nil, nil, pipelines.Config{}, nil)

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
		ctx := NewPluginContext(nil, cfg, nil, nil, nil, pipelines.Config{}, nil)

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
		ctx := NewPluginContext(nil, nil, hookManager, nil, nil, pipelines.Config{}, nil)

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
		ctx := NewPluginContext(nil, nil, nil, stepRegistry, nil, pipelines.Config{}, nil)

		// Act
		result := ctx.GetStepRegistry()

		// Assert
		assert.Equal(t, stepRegistry, result)
	})
}

func TestPluginContext_GetPipeline(t *testing.T) {
	t.Run("ReturnsPipeline", func(t *testing.T) {
		// Arrange
		pipeline := NewMockPipeline()
		ctx := NewPluginContext(nil, nil, nil, nil, pipeline, pipelines.Config{}, nil)

		// Act
		result := ctx.GetPipeline()

		// Assert
		assert.Equal(t, pipeline, result)
	})
}

func TestPluginContext_GetPipelineConfig(t *testing.T) {
	t.Run("ReturnsPipelineConfig", func(t *testing.T) {
		// Arrange
		expectedConfig := pipelines.Config{
			Env: "test",
		}
		ctx := NewPluginContext(nil, nil, nil, nil, nil, expectedConfig, nil)

		// Act
		result := ctx.GetPipelineConfig()

		// Assert
		assert.Equal(t, expectedConfig, result)
	})
}

func TestPluginContext_GetLogger(t *testing.T) {
	t.Run("ReturnsLogger", func(t *testing.T) {
		// Arrange
		logger := NewMockLogger()
		ctx := NewPluginContext(nil, nil, nil, nil, nil, pipelines.Config{}, logger)

		// Act
		result := ctx.GetLogger()

		// Assert
		assert.Equal(t, logger, result)
	})
}
