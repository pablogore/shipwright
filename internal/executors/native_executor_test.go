package executors

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNativeExecutor(t *testing.T) {
	executor := NewNativeExecutor("1.25.5", "/tmp/test")

	assert.NotNil(t, executor)
	assert.Equal(t, "1.25.5", executor.goVersion)
	assert.Equal(t, "/tmp/test", executor.workDir)
}

func TestNewNativeExecutor_EmptyWorkDir(t *testing.T) {
	executor := NewNativeExecutor("1.25.5", "")

	assert.NotNil(t, executor)
	assert.Equal(t, ".", executor.workDir)
}

func TestNativeExecutor_Name(t *testing.T) {
	executor := NewNativeExecutor("1.25.5", ".")

	assert.Equal(t, "native", executor.Name())
}

func TestNativeExecutor_CanExecute(t *testing.T) {
	ctx := context.Background()
	executor := NewNativeExecutor("1.25.5", ".")

	// Should return true if Go is installed (which it should be in test environment)
	result := executor.CanExecute(ctx)
	assert.True(t, result)
}

func TestNativeExecutor_SupportsParallel(t *testing.T) {
	executor := NewNativeExecutor("1.25.5", ".")

	assert.True(t, executor.SupportsParallel())
}

func TestNativeExecutor_GetCacheStrategy(t *testing.T) {
	executor := NewNativeExecutor("1.25.5", ".")

	strategy := executor.GetCacheStrategy()
	assert.Nil(t, strategy) // Native executor doesn't provide cache strategy
}

func TestNativeExecutor_ExecuteStep_UnsupportedStep(t *testing.T) {
	ctx := context.Background()
	executor := NewNativeExecutor("1.25.5", ".")

	err := executor.ExecuteStep(ctx, "unsupported-step")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported step for native execution")
}

func TestNativeExecutor_ExecuteStep_Setup_NotGoProject(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	executor := NewNativeExecutor("1.25.5", tmpDir)

	err := executor.ExecuteStep(ctx, "setup")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a Go project")
}

func TestNativeExecutor_ExecuteStep_Setup_GoProject(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create a minimal go.mod file
	goModContent := "module test\n\ngo 1.25.5\n"
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	executor := NewNativeExecutor("1.25.5", tmpDir)

	err = executor.ExecuteStep(ctx, "setup")
	// May fail if go mod download fails, but should not fail with "not a Go project"
	if err != nil {
		assert.NotContains(t, err.Error(), "not a Go project")
	}
}

func TestNativeExecutor_IsGoProject_True(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := "module test\n\ngo 1.25.5\n"
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	executor := NewNativeExecutor("1.25.5", tmpDir)

	// Use reflection or make isGoProject public for testing
	// For now, we test through ExecuteStep
	result := executor.isGoProject()
	assert.True(t, result)
}

func TestNativeExecutor_IsGoProject_False(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewNativeExecutor("1.25.5", tmpDir)

	result := executor.isGoProject()
	assert.False(t, result)
}
