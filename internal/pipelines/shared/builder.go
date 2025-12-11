package shared

import (
	"fmt"

	"dagger.io/dagger"
	"golang.org/x/net/context"
)

// GoBuilder encapsulates the logic for building Go binaries.
//
// Fields:
//   - Client: The Dagger client used to execute pipeline operations.
//   - Source: The source directory containing the Go project files.
//   - GoModCache: The cache volume for Go modules.
//   - GoBuildCache: The cache volume for Go build artifacts.
//   - GoVersion: The configurable version of Go to use for the build.
//
// Note: Dagger handles connection management internally. The client should be
// created once at the application level and reused throughout the pipeline.
type GoBuilder struct {
	Client       *dagger.Client
	Source       *dagger.Directory
	GoModCache   *dagger.CacheVolume
	GoBuildCache *dagger.CacheVolume
	GoVersion    string
}

// NewGoBuilder creates a new instance of GoBuilder.
//
// Parameters:
//   - client: The Dagger client used for pipeline operations. Must be a valid,
//     connected client. Connection management is handled by Dagger internally.
//   - src: The source directory containing the Go project files.
//   - version: The Go version to use for the build (e.g., "1.21", "1.25.5").
//
// Returns:
//   - A pointer to a new GoBuilder instance.
func NewGoBuilder(client *dagger.Client, src *dagger.Directory, version string) *GoBuilder {
	return &GoBuilder{
		Client:       client,
		Source:       src,
		GoModCache:   client.CacheVolume("go-mod-cache"),
		GoBuildCache: client.CacheVolume("go-build-cache"),
		GoVersion:    version,
	}
}

// Build compiles the Go binary based on the provided parameters.
//
// This method uses Dagger's immutable API to build a container, compile the Go
// binary, and export it to the host filesystem. Dagger handles connection
// management internally, so no manual reconnection logic is needed.
//
// Parameters:
//   - ctx: The context for managing execution and cancellation.
//   - outPath: The output path where the compiled binary will be exported.
//   - target: The name of the output binary file.
//   - env: A map of custom environment variables to set during the build.
//
// Returns:
//   - A string representing the path to the exported binary (same as outPath).
//   - An error if the build process fails. Errors are propagated from Dagger
//     and may include connection issues, build failures, or export errors.
func (b *GoBuilder) Build(ctx context.Context, outPath string, target string, env map[string]string) (string, error) {
	goImage := "golang:" + b.GoVersion

	// Build the container with Go toolchain and mount source code
	container := b.Client.Container().
		From(goImage).
		WithMountedDirectory("/app", b.Source).
		WithMountedCache("/go/pkg/mod", b.GoModCache).
		WithMountedCache("/root/.cache/go-build", b.GoBuildCache).
		WithWorkdir("/app").
		WithEnvVariable("GOPATH", "/go").
		WithEnvVariable("GOCACHE", "/root/.cache/go-build")

	// Add custom environment variables
	for k, v := range env {
		container = container.WithEnvVariable(k, v)
	}

	// Run Go commands: tidy dependencies and build the binary
	// Using Sync() to ensure commands complete before proceeding
	_, err := container.
		WithExec([]string{"go", "mod", "tidy"}).
		WithExec([]string{"go", "build", "-ldflags=-s -w", "-o", target, "main.go"}).
		Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to build Go binary: %w", err)
	}

	// Export the built binary to the host filesystem
	file := container.File("/app/" + target)
	_, err = file.Export(ctx, outPath)
	if err != nil {
		return "", fmt.Errorf("failed to export binary to %s: %w", outPath, err)
	}

	return outPath, nil
}
