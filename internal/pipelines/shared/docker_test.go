package shared

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain skips this package's tests only when no Dagger/Docker engine is
// actually reachable, instead of unconditionally disabling the whole suite.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := dagger.Connect(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "skipping shared package tests: no Dagger/Docker engine available:", err)
		return 0
	}
	client.Close()

	return m.Run()
}

func TestNewDockerDeployer(t *testing.T) {
	ctx := t.Context()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	// Test data
	imageName := "registry.example.com/test/image"
	src := client.Directory()
	tag := "test-tag"
	user := "test-user"
	pass := "test-pass"

	// Create deployer
	deployer := NewDockerDeployer(client, imageName, src, tag, user, pass)

	// Verify
	assert.NotNil(t, deployer)
	assert.Equal(t, client, deployer.Client)
	assert.Equal(t, imageName, deployer.ImageName)
	assert.Equal(t, src, deployer.Source)
	assert.Equal(t, user, deployer.RegistryUser)
	assert.NotNil(t, deployer.RegistryPass)
	assert.Equal(t, tag, deployer.Tag)
}

func TestDockerDeployer_BuildAndPush(t *testing.T) {
	// Setup
	ctx := t.Context()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a simple Dockerfile
	dockerfile := `FROM alpine:latest
CMD ["echo", "Hello, World!"]`
	if err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte(dockerfile), 0o644); err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}

	// Create source directory in Dagger
	src := client.Host().Directory(tmpDir)
	imageName := "registry.example.com/test/image"
	tag := "test-tag"
	user := "test-user"
	pass := "test-pass"

	deployer := NewDockerDeployer(client, imageName, src, tag, user, pass)

	// Test
	err = deployer.BuildAndPush(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error al publicar la imagen")
}

func TestDockerDeployer_BuildAndPush_Error(t *testing.T) {
	ctx := t.Context()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	// Test data with invalid Dockerfile
	imageName := "registry.example.com/test/image"
	src := client.Directory()
	src = src.WithNewFile("Dockerfile", "INVALID DOCKERFILE")
	tag := "test-tag"
	user := "test-user"
	pass := "test-pass"

	// Create deployer
	deployer := NewDockerDeployer(client, imageName, src, tag, user, pass)

	// Test
	err = deployer.BuildAndPush(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error al publicar la imagen")
}
