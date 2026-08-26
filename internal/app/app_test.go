package app

import (
	"testing"

	"github.com/pablogore/shipwright/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	app := NewApp(container)

	assert.NotNil(t, app)
	assert.Equal(t, container, app.container)
}

func TestApp_GetContainer(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)
	app := NewApp(container)

	retrievedContainer := app.GetContainer()
	assert.Equal(t, container, retrievedContainer)
}

func TestApp_WithNilContainer(t *testing.T) {
	app := NewApp(nil)

	assert.NotNil(t, app)
	assert.Nil(t, app.container)

	// These should panic with nil container
	assert.Panics(t, func() {
		_ = app.Start(t.Context())
	})

	assert.Panics(t, func() {
		_ = app.Stop(t.Context())
	})

	container := app.GetContainer()
	assert.Nil(t, container)
}

// Test global app functions
func TestGetContainer_NotInitialized(t *testing.T) {
	// Reset global state
	Reset()

	// Test getting container when not initialized
	assert.Panics(t, func() {
		GetContainer()
	})
}

func TestGetContainer_Initialized(t *testing.T) {
	// Reset global state
	Reset()

	cfg, _ := config.NewConfigurationWrapper()
	ctx := t.Context()

	// Initialize container
	err := Initialize(ctx, cfg)
	require.NoError(t, err)

	// Test getting container
	container := GetContainer()
	assert.NotNil(t, container)
}

func TestInitialize_Success(t *testing.T) {
	// Reset global state
	Reset()

	cfg, _ := config.NewConfigurationWrapper()
	ctx := t.Context()

	// Test successful initialization
	err := Initialize(ctx, cfg)
	require.NoError(t, err)

	// Verify container is set
	container := GetContainer()
	assert.NotNil(t, container)
}

func TestInitialize_MultipleCalls(t *testing.T) {
	// Reset global state
	Reset()

	cfg, _ := config.NewConfigurationWrapper()
	ctx := t.Context()

	// First initialization
	err := Initialize(ctx, cfg)
	require.NoError(t, err)

	container1 := GetContainer()

	// Second initialization should not change the container
	err = Initialize(ctx, cfg)
	require.NoError(t, err)

	container2 := GetContainer()
	assert.Equal(t, container1, container2)
}

func TestReset(t *testing.T) {
	// Reset global state
	Reset()

	cfg, _ := config.NewConfigurationWrapper()
	ctx := t.Context()

	// Initialize container
	err := Initialize(ctx, cfg)
	require.NoError(t, err)

	container := GetContainer()
	assert.NotNil(t, container)

	// Reset
	Reset()

	// Container should be nil after reset
	assert.Panics(t, func() {
		GetContainer()
	})
}

func TestApp_Start_Success(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Start the container to register all components
	err := container.Start(t.Context())
	require.NoError(t, err)

	app := NewApp(container)

	// Test successful start
	err = app.Start(t.Context())
	require.NoError(t, err)
}

func TestApp_Stop_Success(t *testing.T) {
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	// Start the container to register all components
	err := container.Start(t.Context())
	require.NoError(t, err)

	app := NewApp(container)

	// Test successful stop
	err = app.Stop(t.Context())
	require.NoError(t, err)
}

func TestApp_Start_WithLoggerError(t *testing.T) {
	// Create a container
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	app := NewApp(container)

	// Test start - App uses logger.L() which is global, so it doesn't fail on logger errors
	// The container.Start() may fail for other reasons, but not logger-related
	err := app.Start(t.Context())
	// Start should succeed as it only calls container.Start() which registers components
	require.NoError(t, err)
}

func TestApp_Stop_WithLoggerError(t *testing.T) {
	// Create a container
	cfg, _ := config.NewConfigurationWrapper()
	container := NewContainer(t.Context(), cfg)

	app := NewApp(container)

	// Test stop - App uses logger.L() which is global, so it doesn't fail on logger errors
	// The container.Stop() may fail for other reasons, but not logger-related
	err := app.Stop(t.Context())
	// Stop should succeed as it only calls container.Stop() which cleans up components
	require.NoError(t, err)
}
