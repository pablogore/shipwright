package shared

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"dagger.io/dagger"
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
		From("golang:" + goVersion).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithEnvVariable("GO111MODULE", "on").
		WithEnvVariable("CGO_ENABLED", "0")

	// Run tests with coverage and race detection
	output, err := container.
		WithExec([]string{"go", "test", "-v", "-race", "-coverprofile=/tmp/coverage.out", "./..."}).
		Stdout(ctx)
	if err != nil {
		return fmt.Errorf("failed to run tests: %w\nOutput: %s", err, output)
	}

	// Parse coverage from the output
	coverageOutput, err := container.
		WithExec([]string{"go", "tool", "cover", "-func=/tmp/coverage.out"}).
		Stdout(ctx)
	if err != nil {
		return fmt.Errorf("failed to get coverage: %w", err)
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

	fmt.Printf("✅ Test coverage: %.2f%%\n", coverageValue)
	return nil
}
