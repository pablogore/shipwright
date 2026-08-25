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

	// Run tests with coverage and race detection
	// Add retry logic for connection errors with exponential backoff
	maxRetries := 5
	retryDelay := 3 * time.Second
	var output string

	// Give daemon a moment to be ready before first attempt
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled before test execution: %w", ctx.Err())
	case <-time.After(2 * time.Second):
		// Continue with first attempt
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying (exponential backoff)
			waitTime := retryDelay * time.Duration(1<<uint(attempt-1))
			if waitTime > 15*time.Second {
				waitTime = 15 * time.Second // Cap at 15 seconds
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during test retry: %w", ctx.Err())
			case <-time.After(waitTime):
				// Continue with retry
			}
		}

		var testErr error
		output, testErr = container.
			WithExec([]string{"go", "test", "-v", "-race", "-coverprofile=/tmp/coverage.out", "./..."}).
			Stdout(ctx)
		err = testErr

		if err == nil {
			break // Success
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
			return fmt.Errorf("failed to run tests: %w\nOutput: %s", err, output)
		}

		// Log retry attempt
		logger.L().WarnContext(ctx, "Test connection attempt failed, retrying",
			"attempt", attempt+1,
			"max_retries", maxRetries+1,
			"error", err)
	}

	if err != nil {
		return fmt.Errorf("failed to run tests after %d attempts: %w\nOutput: %s", maxRetries+1, err, output)
	}

	// Parse coverage from the output
	// Add retry logic for connection errors during coverage retrieval
	var coverageOutput string
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying (exponential backoff)
			waitTime := retryDelay * time.Duration(1<<uint(attempt-1))
			if waitTime > 15*time.Second {
				waitTime = 15 * time.Second // Cap at 15 seconds
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during coverage retry: %w", ctx.Err())
			case <-time.After(waitTime):
				// Continue with retry
			}
		}

		var coverageErr error
		coverageOutput, coverageErr = container.
			WithExec([]string{"go", "tool", "cover", "-func=/tmp/coverage.out"}).
			Stdout(ctx)
		err = coverageErr

		if err == nil {
			break // Success
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
			return fmt.Errorf("failed to get coverage: %w", err)
		}

		// Log retry attempt
		logger.L().WarnContext(ctx, "Coverage connection attempt failed, retrying",
			"attempt", attempt+1,
			"max_retries", maxRetries+1,
			"error", err)
	}

	if err != nil {
		return fmt.Errorf("failed to get coverage after %d attempts: %w", maxRetries+1, err)
	}

	// Extract the coverage percentage
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
	if coverageValue < coverage {
		return fmt.Errorf("coverage %.2f%% is below the required threshold of %.2f%%", coverageValue, coverage)
	}

	logger.L().InfoContext(ctx, "Test coverage", "coverage", coverageValue)
	return nil
}
