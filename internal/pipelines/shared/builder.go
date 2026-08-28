package shared

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dagger.io/dagger"
	"github.com/pablogore/kit-logger/pkg/logger"
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

	const (
		maxRetries = 5
		retryDelay = 3 * time.Second
	)

	// Give daemon a moment to be ready before first attempt
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("context cancelled before build execution: %w", ctx.Err())
	case <-time.After(2 * time.Second):
		// Continue with first attempt
	}

	// Run Go commands: tidy dependencies and build the binary.
	// Using Sync() to ensure commands complete before proceeding.
	var result *dagger.Container
	err := retryOnConnectionError(ctx, maxRetries, retryDelay, "build", func() error {
		var buildErr error
		result, buildErr = container.
			WithExec([]string{"go", "mod", "tidy"}).
			WithExec([]string{"go", "build", "-ldflags=-s -w", "-o", target, "main.go"}).
			Sync(ctx)
		return buildErr
	})
	if err != nil {
		return "", fmt.Errorf("failed to build Go binary: %w", err)
	}

	// Export the built binary to the host filesystem.
	// Use the result container from Sync() for export.
	file := result.File("/app/" + target)
	err = retryOnConnectionError(ctx, maxRetries, retryDelay, "export", func() error {
		_, exportErr := file.Export(ctx, outPath)
		return exportErr
	})
	if err != nil {
		return "", fmt.Errorf("failed to export binary to %s: %w", outPath, err)
	}

	return outPath, nil
}

// retryOnConnectionError runs execFn, retrying with exponential backoff
// (capped at 15s) when the failure looks like a transient Dagger daemon
// connection error. label identifies the operation in retry logs and the
// context-cancellation error.
func retryOnConnectionError(ctx context.Context, maxRetries int, retryDelay time.Duration, label string, execFn func() error) error {
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying (exponential backoff)
			waitTime := retryDelay * time.Duration(1<<uint(attempt-1))
			if waitTime > 15*time.Second {
				waitTime = 15 * time.Second // Cap at 15 seconds
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during %s retry: %w", label, ctx.Err())
			case <-time.After(waitTime):
				// Continue with retry
			}
		}

		err = execFn()
		if err == nil {
			return nil
		}

		// Check if error is a connection error that should be retried
		errStr := err.Error()
		isConnectionError := strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "dial tcp") ||
			strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "no connection could be made")

		if !isConnectionError || attempt >= maxRetries {
			// Not a connection error or max retries reached
			return err
		}

		// Log retry attempt
		logger.L().WarnContext(ctx, label+" connection attempt failed, retrying",
			"attempt", attempt+1,
			"max_retries", maxRetries+1,
			"error", err)
	}

	return err
}
