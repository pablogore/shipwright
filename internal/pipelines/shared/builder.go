package shared

import (
	"fmt"
	"strings"
	"time"

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
	// Verify Dagger connection before proceeding, reconnect if necessary
	client, err := verifyConnection(ctx, b.Client)
	if err != nil {
		return "", fmt.Errorf("Dagger connection verification failed: %w", err)
	}

	// Update client if reconnection occurred
	if client != b.Client {
		b.Client = client
		// Recreate cache volumes with new client
		b.GoModCache = client.CacheVolume("go-mod-cache")
		b.GoBuildCache = client.CacheVolume("go-build-cache")
	}

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
	exportErr := exportWithRetry(ctx, file, outPath)
	if exportErr != nil {
		// If export failed with connection error, try to reconnect and rebuild the file
		if isConnectionError(exportErr) {
			fmt.Printf("⚠️  Connection lost during export, attempting to reconnect and retry...\n")
			
			// Reconnect the client
			reconnectedClient, reconnectErr := verifyConnection(ctx, b.Client)
			if reconnectErr != nil {
				return "", fmt.Errorf("failed to reconnect during export: %w (original error: %v)", reconnectErr, exportErr)
			}
			
			// Update client and recreate container with new client
			if reconnectedClient != b.Client {
				b.Client = reconnectedClient
				// Recreate cache volumes with new client
				b.GoModCache = reconnectedClient.CacheVolume("go-mod-cache")
				b.GoBuildCache = reconnectedClient.CacheVolume("go-build-cache")
				
				// Rebuild the container and file with the new client
				// Note: We need to rebuild the entire container chain
				goImage := "golang:" + b.GoVersion
				newContainer := b.Client.Container().
					From(goImage).
					WithMountedDirectory("/app", b.Source).
					WithMountedCache("/go/pkg/mod", b.GoModCache).
					WithMountedCache("/root/.cache/go-build", b.GoBuildCache).
					WithWorkdir("/app").
					WithEnvVariable("GOPATH", "/go").
					WithEnvVariable("GOCACHE", "/root/.cache/go-build")
				
				// Re-add custom environment variables
				for k, v := range env {
					newContainer = newContainer.WithEnvVariable(k, v)
				}
				
				// Re-run Go commands
				newContainer = newContainer.WithExec([]string{"go", "mod", "tidy"})
				newContainer = newContainer.WithExec([]string{"go", "build", "-ldflags=-s -w", "-o", target, "main.go"})
				
				// Get the file from the new container
				newFile := newContainer.File("/app/" + target)
				
				// Retry export with the new file
				if retryErr := exportWithRetry(ctx, newFile, outPath); retryErr != nil {
					return "", fmt.Errorf("failed to export after reconnection: %w (original error: %v)", retryErr, exportErr)
				}
				fmt.Printf("✅ Export succeeded after reconnection\n")
				return outPath, nil
			}
		}
		// If it's not a connection error or reconnection didn't help, return the original error
		return "", exportErr
	}
	return outPath, nil
}

// isConnectionError checks if an error is connection-related.
//
// Parameters:
//   - err: The error to check.
//
// Returns:
//   - true if the error is connection-related, false otherwise.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "read tcp") ||
		strings.Contains(errStr, "write tcp") ||
		strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "no connection could be made")
}

// verifyConnection performs a lightweight operation to verify Dagger client connectivity.
// If the connection is lost, it attempts to reconnect automatically.
//
// Parameters:
//   - ctx: The context for managing execution.
//   - client: The Dagger client to verify (may be updated if reconnection is needed).
//
// Returns:
//   - A new client if reconnection was successful, or the original client if connection is valid.
//   - An error if the connection is not available and reconnection failed.
func verifyConnection(ctx context.Context, client *dagger.Client) (*dagger.Client, error) {
	// Perform a lightweight operation to verify connectivity with retries
	// This handles transient connection issues during verification
	maxVerifyRetries := 2
	var verifyErr error
	for attempt := 0; attempt <= maxVerifyRetries; attempt++ {
		if attempt > 0 {
			// Brief wait before retry
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context deadline exceeded during connection verification: %w", verifyErr)
			case <-time.After(1 * time.Second):
			}
		}

		_, verifyErr = client.Container().From("alpine:latest").ID(ctx)
		if verifyErr == nil {
			return client, nil // Connection is valid
		}

		// If it's not a connection error, fail immediately
		if !isConnectionError(verifyErr) {
			return nil, fmt.Errorf("failed to verify Dagger connection: %w", verifyErr)
		}

		// Connection error - retry verification once more before attempting reconnection
		if attempt < maxVerifyRetries {
			fmt.Printf("⚠️  Connection verification attempt %d/%d failed: %v. Retrying...\n",
				attempt+1, maxVerifyRetries+1, verifyErr)
		}
	}

	// All verification attempts failed with connection errors - attempt to reconnect
	fmt.Printf("⚠️  Dagger connection lost, attempting to reconnect...\n")

	// Close the old client if possible
	if client != nil {
		_ = client.Close()
	}

	// Attempt to reconnect with retries
	maxRetries := 3
	retryDelay := 2 * time.Second
	var newClient *dagger.Client
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying (exponential backoff)
			waitTime := retryDelay * time.Duration(1<<uint(attempt-1))
			if waitTime > 10*time.Second {
				waitTime = 10 * time.Second // Cap at 10 seconds
			}

			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context deadline exceeded while reconnecting after %d attempts: %w", attempt, lastErr)
			case <-time.After(waitTime):
				// Continue with retry
			}
		}

		newClient, lastErr = dagger.Connect(ctx, dagger.WithLogOutput(nil))
		if lastErr == nil {
			// Verify the new connection works
			_, verifyErr := newClient.Container().From("alpine:latest").ID(ctx)
			if verifyErr == nil {
				fmt.Printf("✅ Dagger connection reestablished successfully\n")
				return newClient, nil
			}
			// New client also failed, close it and retry
			_ = newClient.Close()
			lastErr = verifyErr
		}

		// Log retry attempt if there are more retries remaining
		if attempt < maxRetries {
			fmt.Printf("⚠️  Reconnection attempt %d/%d failed: %v. Retrying...\n",
				attempt+1, maxRetries+1, lastErr)
		}
	}

	// All reconnection attempts failed
	return nil, fmt.Errorf("failed to reconnect to Dagger engine after %d attempts: %w. "+
		"Ensure Dagger engine is running (try 'dagger run echo test' to verify). "+
		"In GitHub Actions, ensure Docker service is available.",
		maxRetries+1, lastErr)
}

// exportWithRetry wraps file.Export with exponential backoff retry logic.
//
// Parameters:
//   - ctx: The context for managing execution.
//   - file: The Dagger file to export.
//   - outPath: The output path where the file will be exported.
//
// Returns:
//   - An error if all retry attempts fail.
func exportWithRetry(ctx context.Context, file *dagger.File, outPath string) error {
	maxRetries := 3
	retryDelay := 2 * time.Second
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying (exponential backoff)
			waitTime := retryDelay * time.Duration(1<<uint(attempt-1))
			if waitTime > 10*time.Second {
				waitTime = 10 * time.Second // Cap at 10 seconds
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("context deadline exceeded after %d attempts: %w", attempt, lastErr)
			case <-time.After(waitTime):
				// Continue with retry
			}
		}

		_, lastErr = file.Export(ctx, outPath)
		if lastErr == nil {
			return nil // Success
		}

		// Only retry on connection-related errors
		if !isConnectionError(lastErr) {
			return lastErr // Fail immediately for non-connection errors
		}

		// Log retry attempt if there are more retries remaining
		if attempt < maxRetries {
			fmt.Printf("⚠️  Export attempt %d/%d failed with connection error: %v. Retrying...\n",
				attempt+1, maxRetries+1, lastErr)
		}
	}

	// All retries failed
	return fmt.Errorf("failed to export file after %d attempts: %w", maxRetries+1, lastErr)
}
