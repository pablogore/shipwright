package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/getsyntegrity/go-kit-logger/pkg/logger"
	"github.com/pablogore/shipwright/internal/interfaces"
)

// LocalExecutor executes pipeline steps locally without Docker
type LocalExecutor struct {
	config interfaces.Configuration
}

// NewLocalExecutor creates a new local executor
func NewLocalExecutor(config interfaces.Configuration) *LocalExecutor {
	return &LocalExecutor{
		config: config,
	}
}

// ExecuteStep executes a pipeline step locally
func (le *LocalExecutor) ExecuteStep(ctx context.Context, stepName string) error {
	logger.L().InfoContext(ctx, "Executing step locally", "step", stepName)

	// List of supported steps in local mode
	supportedSteps := map[string]bool{
		"setup":    true,
		"build":    true,
		"test":     true,
		"lint":     true,
		"security": true,
	}

	// Check if step is supported
	if !supportedSteps[stepName] {
		// For steps that require cloud services, log a warning and skip
		cloudOnlySteps := map[string]bool{
			"push":    true,
			"release": true,
			"tag":     true,
			"package": true,
		}

		if cloudOnlySteps[stepName] {
			logger.L().WarnContext(ctx, "Step requires cloud services and is skipped in local mode", "step", stepName)
			return nil // Skip silently in local mode
		}

		return fmt.Errorf("unsupported step for local execution: %s. Supported steps: setup, build, test, lint, security", stepName)
	}

	switch stepName {
	case "setup":
		return le.executeSetup(ctx)
	case "build":
		return le.executeBuild(ctx)
	case "test":
		return le.executeTest(ctx)
	case "lint":
		return le.executeLint(ctx)
	case "security":
		return le.executeSecurity(ctx)
	default:
		return fmt.Errorf("unsupported step for local execution: %s", stepName)
	}
}

// executeSetup performs local setup
func (le *LocalExecutor) executeSetup(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Setting up local environment")

	// Check if we're in a Go project
	if !le.isGoProject() {
		return errors.New("not a Go project - setup step requires Go modules")
	}

	// Download dependencies
	logger.L().InfoContext(ctx, "Downloading Go dependencies")
	cmd := exec.CommandContext(ctx, "go", "mod", "download")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download dependencies: %w", err)
	}

	// Tidy modules
	logger.L().InfoContext(ctx, "Tidying Go modules")
	cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy modules: %w", err)
	}

	logger.L().InfoContext(ctx, "Setup completed successfully")
	return nil
}

// executeBuild performs local build
func (le *LocalExecutor) executeBuild(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Building application locally")

	if !le.isGoProject() {
		return errors.New("not a Go project - build step requires Go modules")
	}

	// Build the application
	cmd := exec.CommandContext(ctx, "go", "build", "-v", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build application: %w", err)
	}

	logger.L().InfoContext(ctx, "Build completed successfully")
	return nil
}

// executeTest performs local testing
func (le *LocalExecutor) executeTest(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Running tests locally")

	if !le.isGoProject() {
		return errors.New("not a Go project - test step requires Go modules")
	}

	// Run tests with coverage
	coverageThreshold := le.getCoverageThreshold()
	logger.L().InfoContext(ctx, "Running tests with coverage", "threshold", coverageThreshold)

	// Create coverage directory if it doesn't exist
	coverageDir := "coverage"
	if err := os.MkdirAll(coverageDir, 0755); err != nil {
		return fmt.Errorf("failed to create coverage directory: %w", err)
	}

	// Run tests with coverage
	cmd := exec.CommandContext(ctx, "go", "test", "-v", "-coverprofile=coverage/coverage.out", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run tests: %w", err)
	}

	// Generate coverage report
	logger.L().InfoContext(ctx, "Generating coverage report")
	cmd = exec.CommandContext(ctx, "go", "tool", "cover", "-html=coverage/coverage.out", "-o", "coverage/coverage.html")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logger.L().WarnContext(ctx, "Failed to generate HTML coverage report", "error", err)
	}

	// Check coverage threshold
	if err := le.checkCoverageThreshold(ctx, coverageThreshold); err != nil {
		return fmt.Errorf("coverage threshold not met: %w", err)
	}

	logger.L().InfoContext(ctx, "Tests completed successfully")
	return nil
}

// executeLint performs local linting
func (le *LocalExecutor) executeLint(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Running linters locally")

	if !le.isGoProject() {
		return errors.New("not a Go project - lint step requires Go modules")
	}

	// Run go vet
	logger.L().InfoContext(ctx, "Running go vet")
	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go vet failed: %w", err)
	}

	// Run go fmt check
	logger.L().InfoContext(ctx, "Checking code formatting")
	cmd = exec.CommandContext(ctx, "bash", "-c", "test -z \"$(go fmt ./...)\"")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("code formatting check failed - run 'go fmt ./...' to fix: %w", err)
	}

	// Try to run golangci-lint if available
	if le.isCommandAvailable("golangci-lint") {
		logger.L().InfoContext(ctx, "Running golangci-lint")
		cmd = exec.CommandContext(ctx, "golangci-lint", "run")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			logger.L().WarnContext(ctx, "golangci-lint found issues", "error", err)
			// Don't fail the build for linting issues in local mode
		}
	} else {
		logger.L().InfoContext(ctx, "golangci-lint not available - skipping advanced linting")
	}

	logger.L().InfoContext(ctx, "Linting completed successfully")
	return nil
}

// executeSecurity performs local security checks
func (le *LocalExecutor) executeSecurity(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Running security checks locally")

	if !le.isGoProject() {
		return errors.New("not a Go project - security step requires Go modules")
	}

	// Try to run gosec if available
	if le.isCommandAvailable("gosec") {
		logger.L().InfoContext(ctx, "Running gosec security scanner")
		cmd := exec.CommandContext(ctx, "gosec", "./...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			logger.L().WarnContext(ctx, "gosec found security issues", "error", err)
			// Don't fail the build for security issues in local mode
		}
	} else {
		logger.L().InfoContext(ctx, "gosec not available - skipping security scanning")
	}

	// Try to run govulncheck if available (Go 1.18+)
	if le.isCommandAvailable("govulncheck") {
		logger.L().InfoContext(ctx, "Running govulncheck")
		cmd := exec.CommandContext(ctx, "govulncheck", "./...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			logger.L().WarnContext(ctx, "govulncheck found vulnerabilities", "error", err)
			// Don't fail the build for vulnerabilities in local mode
		}
	} else {
		logger.L().InfoContext(ctx, "govulncheck not available - skipping vulnerability check")
	}

	logger.L().InfoContext(ctx, "Security checks completed successfully")
	return nil
}

// isGoProject checks if the current directory is a Go project
func (le *LocalExecutor) isGoProject() bool {
	_, err := os.Stat("go.mod")
	return err == nil
}

// isCommandAvailable checks if a command is available in PATH
func (le *LocalExecutor) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// getCoverageThreshold gets the coverage threshold from configuration
func (le *LocalExecutor) getCoverageThreshold() float64 {
	// Try to get from configuration
	if coverage := le.config.Get("pipeline.coverage"); coverage != nil {
		if threshold, ok := coverage.(float64); ok {
			return threshold
		}
	}
	// Default threshold
	return 90.0
}

// checkCoverageThreshold checks if coverage meets the threshold
func (le *LocalExecutor) checkCoverageThreshold(ctx context.Context, threshold float64) error {
	coverageFile := "coverage/coverage.out"
	if _, err := os.Stat(coverageFile); os.IsNotExist(err) {
		logger.L().WarnContext(ctx, "Coverage file not found - skipping threshold check")
		return nil
	}

	// Parse coverage output to get percentage
	cmd := exec.CommandContext(ctx, "go", "tool", "cover", "-func=coverage/coverage.out")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to parse coverage: %w", err)
	}

	// Extract total coverage percentage
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "total:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				var coverage float64
				if _, err := fmt.Sscanf(parts[2], "%f%%", &coverage); err == nil {
					logger.L().InfoContext(ctx, "Coverage check", "current", coverage, "threshold", threshold)
					if coverage < threshold {
						return fmt.Errorf("coverage %.2f%% is below threshold %.2f%%", coverage, threshold)
					}
					return nil
				}
			}
		}
	}

	logger.L().WarnContext(ctx, "Could not parse coverage percentage - skipping threshold check")
	return nil
}
