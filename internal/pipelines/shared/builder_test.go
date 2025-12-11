package shared

import (
	"os"
	"path/filepath"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoBuilder(t *testing.T) {
	// Setup
	ctx := t.Context()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	src := client.Directory()
	version := "1.21"

	// Test
	builder := NewGoBuilder(client, src, version)

	// Verify
	assert.NotNil(t, builder)
	assert.Equal(t, client, builder.Client)
	assert.Equal(t, src, builder.Source)
	assert.NotNil(t, builder.GoModCache)
	assert.NotNil(t, builder.GoBuildCache)
	assert.Equal(t, version, builder.GoVersion)
}

func TestGoBuilder_Build(t *testing.T) {
	// Setup
	ctx := t.Context()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a simple Go project
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	goMod := `module test

go 1.21`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create source directory in Dagger
	src := client.Host().Directory(tmpDir)
	builder := NewGoBuilder(client, src, "1.21")

	// Test
	outPath := filepath.Join(tmpDir, "test-binary")
	target := "test-binary"
	env := map[string]string{
		"CGO_ENABLED": "0",
	}

	builtPath, err := builder.Build(ctx, outPath, target, env)
	require.NoError(t, err)
	assert.Equal(t, outPath, builtPath)

	// Verify the binary exists and is executable
	info, err := os.Stat(builtPath)
	require.NoError(t, err)
	assert.True(t, info.Mode().IsRegular())
	assert.NotEqual(t, 0, info.Mode()&0o111) // Check if executable
}

func TestGoBuilder_Build_Error_InvalidSource(t *testing.T) {
	// Setup
	ctx := t.Context()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	// Create an empty source directory
	src := client.Directory()
	builder := NewGoBuilder(client, src, "1.21")

	// Test
	outPath := "/tmp/test-binary"
	target := "test-binary"
	env := map[string]string{}

	builtPath, err := builder.Build(ctx, outPath, target, env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build Go binary")
	assert.Empty(t, builtPath)
}

func TestGoBuilder_Build_Error_InvalidGoVersion(t *testing.T) {
	// Setup
	ctx := t.Context()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a simple Go project
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Create source directory in Dagger
	src := client.Host().Directory(tmpDir)
	builder := NewGoBuilder(client, src, "invalid-version")

	// Test
	outPath := filepath.Join(tmpDir, "test-binary")
	target := "test-binary"
	env := map[string]string{}

	builtPath, err := builder.Build(ctx, outPath, target, env)
	require.Error(t, err)
	// Error could be "failed to build Go binary" or "failed to resolve image"
	assert.Contains(t, err.Error(), "failed")
	assert.Empty(t, builtPath)
}
