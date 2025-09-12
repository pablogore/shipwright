package infra

import (
	"context"
	"testing"

	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
	"github.com/getsyntegrity/syntegrity-dagger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	cfg := pipelines.Config{}

	// Test New() function with nil client to see how it handles it
	pipeline := New(nil, cfg)

	assert.NotNil(t, pipeline)
	assert.IsType(t, &SyntegrityInfraPipeline{}, pipeline)

	infraPipeline := pipeline.(*SyntegrityInfraPipeline)
	assert.Nil(t, infraPipeline.Client) // Should be nil when passed nil
	assert.Equal(t, cfg, infraPipeline.Config)
	assert.Nil(t, infraPipeline.Src) // Should be nil when client is nil
}

func TestSyntegrityInfraPipeline_Name(t *testing.T) {
	// Test Name method directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	name := pipeline.Name()
	assert.Equal(t, "syntegrity-infra", name)
}

func TestSyntegrityInfraPipeline_Test(t *testing.T) {
	// Test Test method directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()
	err := pipeline.Test(ctx)

	require.NoError(t, err) // Test method returns nil
}

func TestSyntegrityInfraPipeline_Build(t *testing.T) {
	// Test Build method directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()
	err := pipeline.Build(ctx)

	require.NoError(t, err) // Build method returns nil
}

func TestSyntegrityInfraPipeline_Package(t *testing.T) {
	// Test Package method directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()
	err := pipeline.Package(ctx)

	require.NoError(t, err) // Package method returns nil
}

func TestSyntegrityInfraPipeline_Tag(t *testing.T) {
	// Test Tag method directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()

	// Tag method panics with "implement me"
	assert.Panics(t, func() {
		_ = pipeline.Tag(ctx)
	}, "Tag method should panic with 'implement me'")
}

func TestSyntegrityInfraPipeline_Push(t *testing.T) {
	// Test Push method directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()

	// Push method panics with "implement me"
	assert.Panics(t, func() {
		_ = pipeline.Push(ctx)
	}, "Push method should panic with 'implement me'")
}

func TestSyntegrityInfraPipeline_Setup(t *testing.T) {
	// Test Setup method directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()
	err := pipeline.Setup(ctx)

	// Setup method requires real Dagger client, so it will return error for nil client
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Setup method requires real Dagger client, not nil")
}

func TestSyntegrityInfraPipeline_BeforeStep(t *testing.T) {
	// Test BeforeStep method directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()

	// BeforeStep method panics with "implement me"
	assert.Panics(t, func() {
		pipeline.BeforeStep(ctx, "test-step")
	}, "BeforeStep method should panic with 'implement me'")
}

func TestSyntegrityInfraPipeline_AfterStep(t *testing.T) {
	// Test AfterStep method directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()

	// AfterStep method panics with "implement me"
	assert.Panics(t, func() {
		pipeline.AfterStep(ctx, "test-step")
	}, "AfterStep method should panic with 'implement me'")
}

func TestSyntegrityInfraPipeline_Integration(t *testing.T) {
	// Test full pipeline execution using mocks
	cfg := pipelines.Config{
		BranchName: "main",
		Registry:   "registry.example.com",
		ImageTag:   "v1.0.0",
	}

	// Create pipeline with nil client (handled gracefully by New())
	pipeline := New(nil, cfg).(*SyntegrityInfraPipeline)

	ctx := t.Context()

	// Test full pipeline execution - all methods should handle nil client appropriately
	err := pipeline.Setup(ctx)
	require.Error(t, err) // Setup will fail due to nil client
	assert.Contains(t, err.Error(), "Setup method requires real Dagger client, not nil")

	err = pipeline.Test(ctx)
	require.NoError(t, err) // Test returns nil (no-op implementation)

	err = pipeline.Build(ctx)
	require.NoError(t, err) // Build returns nil (no-op implementation)

	err = pipeline.Package(ctx)
	require.NoError(t, err) // Package returns nil (no-op implementation)

	// Tag and Push methods should panic with "implement me"
	assert.Panics(t, func() {
		_ = pipeline.Tag(ctx)
	}, "Tag method should panic with 'implement me'")

	assert.Panics(t, func() {
		_ = pipeline.Push(ctx)
	}, "Push method should panic with 'implement me'")
}

func TestSyntegrityInfraPipeline_StructFields(t *testing.T) {
	// Test struct fields directly without requiring New() function
	cfg := pipelines.Config{
		BranchName: "main",
		Registry:   "registry.example.com",
		ImageTag:   "v1.0.0",
	}

	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: cfg,
		Src:    nil,
		Cloner: nil,
	}

	// Verify struct fields
	assert.Nil(t, pipeline.Client)
	assert.Equal(t, cfg, pipeline.Config)
	assert.Nil(t, pipeline.Src)
	assert.Nil(t, pipeline.Cloner)
}

func TestSyntegrityInfraPipeline_NoOpOperations(t *testing.T) {
	// Test no-op operations directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()

	// Test and Build operations should return nil (no-op implementation)
	err := pipeline.Test(ctx)
	require.NoError(t, err)

	err = pipeline.Build(ctx)
	require.NoError(t, err)

	err = pipeline.Package(ctx)
	require.NoError(t, err)

	// Tag and Push operations should panic with "implement me"
	assert.Panics(t, func() {
		_ = pipeline.Tag(ctx)
	}, "Tag method should panic with 'implement me'")

	assert.Panics(t, func() {
		_ = pipeline.Push(ctx)
	}, "Push method should panic with 'implement me'")

	// Setup method requires real client, so it will return error for nil client
	err = pipeline.Setup(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Setup method requires real Dagger client, not nil")
}

func TestSyntegrityInfraPipeline_ContextHandling(t *testing.T) {
	// Test context handling directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	// Test with different contexts
	ctx1 := t.Context()
	type testKey string
	ctx2 := context.WithValue(t.Context(), testKey("test-key"), "value")

	// All operations should work with any context
	err := pipeline.Test(ctx1)
	require.NoError(t, err)

	err = pipeline.Test(ctx2)
	require.NoError(t, err)

	err = pipeline.Build(ctx1)
	require.NoError(t, err)

	err = pipeline.Build(ctx2)
	require.NoError(t, err)
}

func TestSyntegrityInfraPipeline_StepNames(t *testing.T) {
	// Test step names directly without requiring New() function
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()

	// Test with different step names
	stepNames := []string{"setup", "test", "build", "package", "tag", "push", "deploy", "unknown"}

	for _, stepName := range stepNames {
		// BeforeStep and AfterStep methods panic with "implement me"
		assert.Panics(t, func() {
			pipeline.BeforeStep(ctx, stepName)
		}, "BeforeStep should panic with 'implement me' for step: %s", stepName)

		assert.Panics(t, func() {
			pipeline.AfterStep(ctx, stepName)
		}, "AfterStep should panic with 'implement me' for step: %s", stepName)
	}
}

// Test methods that don't require Dagger client
func TestSyntegrityInfraPipeline_SimpleMethods(t *testing.T) {
	// Create a pipeline instance without calling New() to avoid Dagger client requirement
	pipeline := &SyntegrityInfraPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := t.Context()

	// Test Name method - doesn't require client
	name := pipeline.Name()
	assert.Equal(t, "syntegrity-infra", name)

	// Test Test method - returns nil, doesn't require client
	err := pipeline.Test(ctx)
	require.NoError(t, err)

	// Test Build method - returns nil, doesn't require client
	err = pipeline.Build(ctx)
	require.NoError(t, err)

	// Test Package method - returns nil, doesn't require client
	err = pipeline.Package(ctx)
	require.NoError(t, err)
}

// Test methods with mocks
func TestSyntegrityInfraPipeline_WithMocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDaggerClient(ctrl)
	mockDirectory := mocks.NewMockDaggerDirectory(ctrl)

	cfg := pipelines.Config{}

	pipeline := &SyntegrityInfraPipeline{
		Client: mockClient,
		Config: cfg,
		Src:    mockDirectory,
		Cloner: nil,
	}

	ctx := t.Context()

	// Test Name method
	name := pipeline.Name()
	assert.Equal(t, "syntegrity-infra", name)

	// Test Test method
	err := pipeline.Test(ctx)
	require.NoError(t, err)

	// Test Build method
	err = pipeline.Build(ctx)
	require.NoError(t, err)

	// Test Package method
	err = pipeline.Package(ctx)
	require.NoError(t, err)
}
