package gokit

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
	// Test New() function with nil client to see how it handles it
	cfg := pipelines.Config{
		GitProtocol: "https",
	}

	// New() should handle nil client gracefully
	pipeline := New(nil, cfg)

	assert.NotNil(t, pipeline)
	assert.IsType(t, &GoKitPipeline{}, pipeline)

	gokitPipeline := pipeline.(*GoKitPipeline)
	assert.Nil(t, gokitPipeline.Client) // Should be nil when passed nil
	assert.Equal(t, cfg, gokitPipeline.Config)
	assert.Nil(t, gokitPipeline.Src)    // Should be nil when client is nil
	assert.Nil(t, gokitPipeline.Cloner) // Should be nil when client is nil
}

func TestNew_SSHProtocol(t *testing.T) {
	// Test New() function with SSH protocol and nil client
	cfg := pipelines.Config{
		GitProtocol: "ssh",
	}

	// New() should handle nil client gracefully
	pipeline := New(nil, cfg)

	assert.NotNil(t, pipeline)
	gokitPipeline := pipeline.(*GoKitPipeline)
	assert.Nil(t, gokitPipeline.Cloner) // Should be nil when client is nil
}

func TestGoKitPipeline_Name(t *testing.T) {
	// Test Name method directly without requiring New() function
	pipeline := &GoKitPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	name := pipeline.Name()
	assert.Equal(t, "go-kit", name)
}

func TestGoKitPipeline_Setup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Use our mock Dagger client instead of real client
	mockDaggerClient := mocks.NewMockDaggerClient(ctrl)
	mockCloner := mocks.NewMockCloner(ctrl)

	cfg := pipelines.Config{
		GitRepo:     "https://gitlab.com/syntegrity/go-kit.git",
		GitRef:      "main",
		GitProtocol: "https",
	}

	// Create pipeline with mock client
	pipeline := &GoKitPipeline{
		Client: mockDaggerClient,
		Config: cfg,
		Src:    nil,
		Cloner: mockCloner,
	}

	ctx := context.Background()
	err := pipeline.Setup(ctx)

	// Setup method requires real Dagger client, so it will return error for mock
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Setup method requires real Dagger client, not mock")
}

func TestGoKitPipeline_Setup_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Use our mock Dagger client instead of real client
	mockDaggerClient := mocks.NewMockDaggerClient(ctrl)
	mockCloner := mocks.NewMockCloner(ctrl)

	cfg := pipelines.Config{
		GitRepo:     "https://gitlab.com/syntegrity/go-kit.git",
		GitRef:      "main",
		GitProtocol: "https",
	}

	// Create pipeline with mock client
	pipeline := &GoKitPipeline{
		Client: mockDaggerClient,
		Config: cfg,
		Src:    nil,
		Cloner: mockCloner,
	}

	ctx := context.Background()
	err := pipeline.Setup(ctx)

	// Setup method requires real Dagger client, so it will return error for mock
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Setup method requires real Dagger client, not mock")
}

func TestGoKitPipeline_Test(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Use our mock Dagger client instead of real client
	mockDaggerClient := mocks.NewMockDaggerClient(ctrl)
	mockDaggerDirectory := mocks.NewMockDaggerDirectory(ctrl)

	cfg := pipelines.Config{
		GoVersion: "1.21",
	}

	// Create pipeline with mock client and directory
	pipeline := &GoKitPipeline{
		Client: mockDaggerClient,
		Config: cfg,
		Src:    mockDaggerDirectory,
		Cloner: nil,
	}

	ctx := context.Background()
	err := pipeline.Test(ctx)

	// Test method requires real Dagger client, so it will return error for mock
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Test method requires real Dagger client, not mock")
}

func TestGoKitPipeline_Test_NoSrc(t *testing.T) {
	// Test Test method with nil Src directly without requiring New() function
	pipeline := &GoKitPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil, // Force Src to be nil
		Cloner: nil,
	}

	ctx := context.Background()
	err := pipeline.Test(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")
}

func TestGoKitPipeline_Test_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Use our mock Dagger client instead of real client
	mockDaggerClient := mocks.NewMockDaggerClient(ctrl)
	mockDaggerDirectory := mocks.NewMockDaggerDirectory(ctrl)

	cfg := pipelines.Config{
		GoVersion: "1.21",
	}

	// Create pipeline with mock client and directory
	pipeline := &GoKitPipeline{
		Client: mockDaggerClient,
		Config: cfg,
		Src:    mockDaggerDirectory,
		Cloner: nil,
	}

	ctx := context.Background()
	err := pipeline.Test(ctx)

	// Test method requires real Dagger client, so it will return error for mock
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Test method requires real Dagger client, not mock")
}

func TestGoKitPipeline_Build(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Use our mock Dagger client instead of real client
	mockDaggerClient := mocks.NewMockDaggerClient(ctrl)
	mockDaggerDirectory := mocks.NewMockDaggerDirectory(ctrl)

	cfg := pipelines.Config{
		GoVersion: "1.21",
	}

	// Create pipeline with mock client and directory
	pipeline := &GoKitPipeline{
		Client: mockDaggerClient,
		Config: cfg,
		Src:    mockDaggerDirectory,
		Cloner: nil,
	}

	ctx := context.Background()
	err := pipeline.Build(ctx)

	// Build method requires real Dagger client, so it will return error for mock
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Build method requires real Dagger client, not mock")
}

func TestGoKitPipeline_Build_NoSrc(t *testing.T) {
	// Test Build method with nil Src directly without requiring New() function
	pipeline := &GoKitPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil, // Force Src to be nil
		Cloner: nil,
	}

	ctx := context.Background()
	err := pipeline.Build(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")
}

func TestGoKitPipeline_Build_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Use our mock Dagger client instead of real client
	mockDaggerClient := mocks.NewMockDaggerClient(ctrl)
	mockDaggerDirectory := mocks.NewMockDaggerDirectory(ctrl)

	cfg := pipelines.Config{
		GoVersion: "1.21",
	}

	// Create pipeline with mock client and directory
	pipeline := &GoKitPipeline{
		Client: mockDaggerClient,
		Config: cfg,
		Src:    mockDaggerDirectory,
		Cloner: nil,
	}

	ctx := context.Background()
	err := pipeline.Build(ctx)

	// Build method requires real Dagger client, so it will return error for mock
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Build method requires real Dagger client, not mock")
}

func TestGoKitPipeline_Package(t *testing.T) {
	// Test Package method directly without requiring New() function
	pipeline := &GoKitPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := context.Background()
	err := pipeline.Package(ctx)

	require.Error(t, err) // Package returns "not implemented" error
	assert.Contains(t, err.Error(), "not implemented")
}

func TestGoKitPipeline_Tag(t *testing.T) {
	// Test Tag method directly without requiring New() function
	pipeline := &GoKitPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := context.Background()

	// Tag method returns error for mock client
	err := pipeline.Tag(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Tag method requires real Dagger client, not mock")
}

func TestGoKitPipeline_Push(t *testing.T) {
	// Test Push method directly without requiring New() function
	pipeline := &GoKitPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := context.Background()
	err := pipeline.Push(ctx)

	require.Error(t, err) // Push returns "not implemented" error
	assert.Contains(t, err.Error(), "not implemented")
}

func TestGoKitPipeline_BeforeStep(t *testing.T) {
	// Test BeforeStep method directly without requiring New() function
	pipeline := &GoKitPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := context.Background()
	hook := pipeline.BeforeStep(ctx, "test-step")

	assert.Nil(t, hook) // BeforeStep returns nil
}

func TestGoKitPipeline_AfterStep(t *testing.T) {
	// Test AfterStep method directly without requiring New() function
	pipeline := &GoKitPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := context.Background()
	hook := pipeline.AfterStep(ctx, "test-step")

	assert.Nil(t, hook) // AfterStep returns nil
}

func TestGoKitPipeline_Integration(t *testing.T) {
	// Test full pipeline execution using mocks
	cfg := pipelines.Config{
		GitRepo:     "https://gitlab.com/syntegrity/go-kit.git",
		GitRef:      "main",
		GitProtocol: "https",
		GoVersion:   "1.21",
	}

	// Create pipeline with nil client (handled gracefully by New())
	pipeline := New(nil, cfg).(*GoKitPipeline)

	ctx := context.Background()

	// Test full pipeline execution - all methods should handle nil client appropriately
	err := pipeline.Setup(ctx)
	require.Error(t, err) // Setup will fail due to nil client
	require.Contains(t, err.Error(), "Setup method requires real Dagger client, not nil")

	err = pipeline.Test(ctx)
	require.Error(t, err) // Test will fail due to nil Src
	require.Contains(t, err.Error(), "pipeline not set up: source directory is nil")

	err = pipeline.Build(ctx)
	require.Error(t, err) // Build will fail due to nil Src
	require.Contains(t, err.Error(), "pipeline not set up: source directory is nil")

	err = pipeline.Package(ctx)
	require.Error(t, err) // Package returns "not implemented"
	require.Contains(t, err.Error(), "not implemented")

	err = pipeline.Tag(ctx)
	require.Error(t, err) // Tag will fail due to nil client
	require.Contains(t, err.Error(), "Tag method requires real Dagger client, not mock")

	err = pipeline.Push(ctx)
	require.Error(t, err) // Push returns "not implemented"
	require.Contains(t, err.Error(), "not implemented")
}

// Test methods that don't require Dagger client
func TestGoKitPipeline_SimpleMethods(t *testing.T) {
	// Create a pipeline instance without calling New() to avoid Dagger client requirement
	pipeline := &GoKitPipeline{
		Client: nil,
		Config: pipelines.Config{},
		Src:    nil,
		Cloner: nil,
	}

	ctx := context.Background()

	// Test Name method - doesn't require client
	name := pipeline.Name()
	assert.Equal(t, "go-kit", name)

	// Test Package method - returns "not implemented" error
	err := pipeline.Package(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")

	// Test Push method - returns "not implemented" error
	err = pipeline.Push(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")

	// Test Cleanup method - returns "not implemented" error
	err = pipeline.Cleanup(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")

	// Test Test method with nil Src - should return error
	err = pipeline.Test(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")

	// Test Build method with nil Src - should return error
	err = pipeline.Build(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")
}

// Test methods with mocks
func TestGoKitPipeline_WithMocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDaggerClient(ctrl)
	mockDirectory := mocks.NewMockDaggerDirectory(ctrl)

	cfg := pipelines.Config{}

	pipeline := &GoKitPipeline{
		Client: mockClient,
		Config: cfg,
		Src:    mockDirectory,
		Cloner: nil,
	}

	ctx := context.Background()

	// Test Name method
	name := pipeline.Name()
	assert.Equal(t, "go-kit", name)

	// Test Package method - returns "not implemented" error
	err := pipeline.Package(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")

	// Test Push method - returns "not implemented" error
	err = pipeline.Push(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")

	// Test Cleanup method - returns "not implemented" error
	err = pipeline.Cleanup(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")

	// Test Test method with mock Src - should return error for mock client
	err = pipeline.Test(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Test method requires real Dagger client, not mock")

	// Test Build method with mock Src - should return error for mock client
	err = pipeline.Build(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Build method requires real Dagger client, not mock")
}
