package shared

import (
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
type GoBuilder struct {
	Client       *dagger.Client
	Source       *dagger.Directory
	GoModCache   *dagger.CacheVolume
	GoBuildCache *dagger.CacheVolume
	GoVersion    string // Configurable Go version
}

// NewGoBuilder creates a new instance of GoBuilder.
//
// Parameters:
//   - client: The Dagger client used for pipeline operations.
//   - src: The source directory containing the Go project files.
//   - version: The Go version to use for the build.
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
// Parameters:
//   - ctx: The context for managing execution.
//   - outPath: The output path where the compiled binary will be exported.
//   - target: The name of the output binary file.
//   - env: A map of custom environment variables to set during the build.
//
// Returns:
//   - A string representing the path to the exported binary.
//   - An error if the build process fails.
func (b *GoBuilder) Build(ctx context.Context, outPath string, target string, env map[string]string) (string, error) {
	goImage := "golang:" + b.GoVersion

	// Initialize the container with the specified Go version and mount directories
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

	// Run Go commands to tidy dependencies and build the binary
	container = container.WithExec([]string{"go", "mod", "tidy"})
	container = container.WithExec([]string{"go", "build", "-ldflags=-s -w", "-o", target, "main.go"})

	// Export the built binary to the specified output path
	file := container.File("/app/" + target)
	_, err := file.Export(ctx, outPath)
	if err != nil {
		return "", err
	}
	return outPath, nil
}
