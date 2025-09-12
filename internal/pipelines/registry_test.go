package pipelines

import (
	"context"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.pipelines)
	assert.Empty(t, registry.pipelines)
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	// Create a mock factory function
	factory := func(_ *dagger.Client, _ Config) Pipeline {
		return &mockPipeline{name: "test-pipeline"}
	}

	// Register the pipeline
	registry.Register("test-pipeline", factory)

	// Verify it was registered
	assert.Contains(t, registry.pipelines, "test-pipeline")
	assert.NotNil(t, registry.pipelines["test-pipeline"])
}

func TestRegistry_Get_Success(t *testing.T) {
	registry := NewRegistry()

	// Register a pipeline
	expectedName := "test-pipeline"
	factory := func(_ *dagger.Client, _ Config) Pipeline {
		return &mockPipeline{name: expectedName}
	}
	registry.Register(expectedName, factory)

	// Get the pipeline
	client := &dagger.Client{}
	cfg := Config{}

	pipeline, err := registry.Get(expectedName, client, cfg)

	// Verify success
	require.NoError(t, err)
	assert.NotNil(t, pipeline)
	assert.Equal(t, expectedName, pipeline.Name())
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := NewRegistry()

	// Try to get a non-existent pipeline
	client := &dagger.Client{}
	cfg := Config{}

	pipeline, err := registry.Get("nonexistent-pipeline", client, cfg)

	// Verify error
	require.Error(t, err)
	assert.Nil(t, pipeline)
	assert.Contains(t, err.Error(), "pipeline not found")
	assert.Contains(t, err.Error(), "nonexistent-pipeline")
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Initially empty
	names := registry.List()
	assert.Empty(t, names)

	// Register some pipelines
	registry.Register("pipeline-1", func(_ *dagger.Client, _ Config) Pipeline {
		return &mockPipeline{name: "pipeline-1"}
	})
	registry.Register("pipeline-2", func(_ *dagger.Client, _ Config) Pipeline {
		return &mockPipeline{name: "pipeline-2"}
	})
	registry.Register("pipeline-3", func(_ *dagger.Client, _ Config) Pipeline {
		return &mockPipeline{name: "pipeline-3"}
	})

	// Get list
	names = registry.List()

	// Verify all names are present
	assert.Len(t, names, 3)
	assert.Contains(t, names, "pipeline-1")
	assert.Contains(t, names, "pipeline-2")
	assert.Contains(t, names, "pipeline-3")
}

// mockPipeline is a simple mock implementation of the Pipeline interface
type mockPipeline struct {
	name string
}

func (m *mockPipeline) Test(_ context.Context) error                    { return nil }
func (m *mockPipeline) Build(_ context.Context) error                   { return nil }
func (m *mockPipeline) Package(_ context.Context) error                 { return nil }
func (m *mockPipeline) Tag(_ context.Context) error                     { return nil }
func (m *mockPipeline) Name() string                                    { return m.name }
func (m *mockPipeline) Setup(_ context.Context) error                   { return nil }
func (m *mockPipeline) Push(_ context.Context) error                    { return nil }
func (m *mockPipeline) BeforeStep(_ context.Context, _ string) HookFunc { return nil }
func (m *mockPipeline) AfterStep(_ context.Context, _ string) HookFunc  { return nil }
