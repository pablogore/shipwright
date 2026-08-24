package goservice

import (
	"testing"

	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeline_Build_BinaryMode(t *testing.T) {
	pipeline := &Pipeline{
		Client:  nil,
		Config:  pipelines.Config{GoVersion: "1.25.5"},
		Options: GoServiceOptions{BuildMode: BuildModeBinary, BinaryName: "test-app"},
		Src:     nil,
		Cloner:  nil,
	}

	ctx := t.Context()
	err := pipeline.Build(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")
}

func TestPipeline_Build_DockerMode(t *testing.T) {
	pipeline := &Pipeline{
		Client:  nil,
		Config:  pipelines.Config{GoVersion: "1.25.5"},
		Options: GoServiceOptions{BuildMode: BuildModeDocker},
		Src:     nil,
		Cloner:  nil,
	}

	ctx := t.Context()
	err := pipeline.Build(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")
}

func TestPipeline_Build_BothMode(t *testing.T) {
	pipeline := &Pipeline{
		Client:  nil,
		Config:  pipelines.Config{GoVersion: "1.25.5"},
		Options: GoServiceOptions{BuildMode: BuildModeBoth, BinaryName: "test-app"},
		Src:     nil,
		Cloner:  nil,
	}

	ctx := t.Context()
	err := pipeline.Build(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")
}

func TestPipeline_Build_DefaultMode(t *testing.T) {
	pipeline := &Pipeline{
		Client:  nil,
		Config:  pipelines.Config{GoVersion: "1.25.5"},
		Options: GoServiceOptions{BuildMode: "", BinaryName: "test-app"},
		Src:     nil,
		Cloner:  nil,
	}

	ctx := t.Context()
	err := pipeline.Build(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")
}

func TestPipeline_Package_DockerMode_ImageAlreadyBuilt(t *testing.T) {
	// This test requires a mock Src to pass the initial validation
	// The actual check for Image being nil happens after Src validation
	// For a complete test, we would need to mock the Dagger client
	// This test verifies the logic path for Docker mode when image is not built
	pipeline := &Pipeline{
		Client:  nil,
		Config:  pipelines.Config{},
		Options: GoServiceOptions{BuildMode: BuildModeDocker},
		Src:     nil, // Will fail Src validation first
		Cloner:  nil,
		Image:   nil, // Simulating that Build was not called
	}

	ctx := t.Context()
	err := pipeline.Package(ctx)

	require.Error(t, err)
	// The error will be about Src being nil, not Image, because Src is checked first
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")
}

func TestPipeline_Package_BinaryMode(t *testing.T) {
	pipeline := &Pipeline{
		Client:  nil,
		Config:  pipelines.Config{},
		Options: GoServiceOptions{BuildMode: BuildModeBinary, BinaryName: "test-app"},
		Src:     nil,
		Cloner:  nil,
		Image:   nil,
	}

	ctx := t.Context()
	err := pipeline.Package(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")
}

func TestPipeline_Package_BothMode(t *testing.T) {
	pipeline := &Pipeline{
		Client:  nil,
		Config:  pipelines.Config{},
		Options: GoServiceOptions{BuildMode: BuildModeBoth, BinaryName: "test-app"},
		Src:     nil,
		Cloner:  nil,
		Image:   nil,
	}

	ctx := t.Context()
	err := pipeline.Package(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not set up: source directory is nil")
}
