package shared

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dagger.io/dagger"
	"github.com/pablogore/kit-logger/pkg/logger"
)

// RunTestsWithCoverage runs the tests for the project with coverage.
//
// Parameters:
//   - ctx: The context for managing execution.
//   - client: The Dagger client used for container operations.
//   - src: The source directory of the cloned repository.
//   - coverage: The coverage threshold for the tests.
//   - goVersion: The Go version to use for testing (e.g., "1.25.5").
//
// Returns:
//   - An error if the tests fail, otherwise nil.
func RunTestsWithCoverage(ctx context.Context, client *dagger.Client, src *dagger.Directory, coverage float64, goVersion string) error {
	// Create a temporary directory for the coverage file
	tmpDir, err := os.MkdirTemp("", "coverage")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use default Go version if not provided
	if goVersion == "" {
		goVersion = "1.25.5"
	}

	// Run the tests in a Dagger container with coverage
	container := client.Container().
		From("golang:"+goVersion).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithEnvVariable("GO111MODULE", "on").
		WithEnvVariable("CGO_ENABLED", "0")

	const (
		maxRetries = 5
		retryDelay = 3 * time.Second
	)

	// Give daemon a moment to be ready before first attempt
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled before test execution: %w", ctx.Err())
	case <-time.After(2 * time.Second):
		// Continue with first attempt
	}

	output, err := runWithRetry(ctx, maxRetries, retryDelay, "test", func() (string, error) {
		return container.
			WithExec([]string{"go", "test", "-v", "-race", "-coverprofile=/tmp/coverage.out", "./..."}).
			Stdout(ctx)
	})
	if err != nil {
		return fmt.Errorf("failed to run tests: %w\nOutput: %s", err, output)
	}

	coverageOutput, err := runWithRetry(ctx, maxRetries, retryDelay, "coverage", func() (string, error) {
		return container.
			WithExec([]string{"go", "tool", "cover", "-func=/tmp/coverage.out"}).
			Stdout(ctx)
	})
	if err != nil {
		return fmt.Errorf("failed to get coverage: %w", err)
	}

	return checkCoverageThreshold(ctx, coverageOutput, coverage)
}

// runWithRetry runs execFn against the test container, retrying with
// exponential backoff (capped at 15s) when the failure looks like a
// transient Dagger daemon connection error. label identifies the operation
// in retry logs and the context-cancellation error.
func runWithRetry(ctx context.Context, maxRetries int, retryDelay time.Duration, label string, execFn func() (string, error)) (string, error) {
	var output string
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
				return "", fmt.Errorf("context cancelled during %s retry: %w", label, ctx.Err())
			case <-time.After(waitTime):
				// Continue with retry
			}
		}

		output, err = execFn()
		if err == nil {
			return output, nil
		}

		// Check if error is a connection error that should be retried
		errStr := err.Error()
		isConnectionError := strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "dial tcp") ||
			strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "no connection could be made") ||
			strings.Contains(errStr, "context canceled")

		if !isConnectionError || attempt >= maxRetries {
			// Not a connection error or max retries reached
			return output, err
		}

		// Log retry attempt
		logger.L().WarnContext(ctx, label+" connection attempt failed, retrying",
			"attempt", attempt+1,
			"max_retries", maxRetries+1,
			"error", err)
	}

	return output, err
}

// checkCoverageThreshold extracts the total coverage percentage from
// `go tool cover`'s output and verifies it meets the required threshold.
func checkCoverageThreshold(ctx context.Context, coverageOutput string, threshold float64) error {
	coverageRegex := regexp.MustCompile(`total:\s+\(statements\)\s+(\d+\.\d+)%`)
	matches := coverageRegex.FindStringSubmatch(coverageOutput)
	if len(matches) < 2 {
		return fmt.Errorf("failed to parse coverage output: %s", coverageOutput)
	}

	coverageValue, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return fmt.Errorf("failed to parse coverage value: %w", err)
	}

	// Check if the coverage meets the threshold
	if coverageValue < threshold {
		return fmt.Errorf("coverage %.2f%% is below the required threshold of %.2f%%", coverageValue, threshold)
	}

	logger.L().InfoContext(ctx, "Test coverage", "coverage", coverageValue)
	return nil
}
