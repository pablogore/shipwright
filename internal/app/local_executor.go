package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
)

// LocalExecutor executes pipeline steps locally without Docker
type LocalExecutor struct {
	logger interfaces.Logger
	config interfaces.Configuration
}

// NewLocalExecutor creates a new local executor
func NewLocalExecutor(logger interfaces.Logger, config interfaces.Configuration) *LocalExecutor {
	return &LocalExecutor{
		logger: logger,
		config: config,
	}
}

// ExecuteStep executes a pipeline step locally
func (le *LocalExecutor) ExecuteStep(ctx context.Context, stepName string) error {
	le.logger.Info("Executing step locally", "step", stepName)

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
	le.logger.Info("Setting up local environment")

	// Check if we're in a Go project
	if !le.isGoProject() {
		return errors.New("not a Go project - setup step requires Go modules")
	}

	// Download dependencies
	le.logger.Info("Downloading Go dependencies")
	cmd := exec.CommandContext(ctx, "go", "mod", "download")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download dependencies: %w", err)
	}

	// Tidy modules
	le.logger.Info("Tidying Go modules")
	cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy modules: %w", err)
	}

	le.logger.Info("Setup completed successfully")
	return nil
}

// executeBuild performs local build
func (le *LocalExecutor) executeBuild(ctx context.Context) error {
	le.logger.Info("Building application locally")

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

	le.logger.Info("Build completed successfully")
	return nil
}

// executeTest performs local testing
func (le *LocalExecutor) executeTest(ctx context.Context) error {
	le.logger.Info("Running tests locally")

	if !le.isGoProject() {
		return errors.New("not a Go project - test step requires Go modules")
	}

	// Run tests with coverage
	coverageThreshold := le.getCoverageThreshold()
	le.logger.Info("Running tests with coverage", "threshold", coverageThreshold)

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
	le.logger.Info("Generating coverage report")
	cmd = exec.CommandContext(ctx, "go", "tool", "cover", "-html=coverage/coverage.out", "-o", "coverage/coverage.html")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		le.logger.Warn("Failed to generate HTML coverage report", "error", err)
	}

	// Check coverage threshold
	if err := le.checkCoverageThreshold(ctx, coverageThreshold); err != nil {
		return fmt.Errorf("coverage threshold not met: %w", err)
	}

	le.logger.Info("Tests completed successfully")
	return nil
}

// executeLint performs local linting
func (le *LocalExecutor) executeLint(ctx context.Context) error {
	le.logger.Info("Running linters locally")

	if !le.isGoProject() {
		return errors.New("not a Go project - lint step requires Go modules")
	}

	// Run go vet
	le.logger.Info("Running go vet")
	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go vet failed: %w", err)
	}

	// Run go fmt check
	le.logger.Info("Checking code formatting")
	cmd = exec.CommandContext(ctx, "bash", "-c", "test -z \"$(go fmt ./...)\"")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("code formatting check failed - run 'go fmt ./...' to fix: %w", err)
	}

	// Try to run golangci-lint if available
	if le.isCommandAvailable("golangci-lint") {
		le.logger.Info("Running golangci-lint")
		cmd = exec.CommandContext(ctx, "golangci-lint", "run")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			le.logger.Warn("golangci-lint found issues", "error", err)
			// Don't fail the build for linting issues in local mode
		}
	} else {
		le.logger.Info("golangci-lint not available - skipping advanced linting")
	}

	le.logger.Info("Linting completed successfully")
	return nil
}

// executeSecurity performs local security checks
func (le *LocalExecutor) executeSecurity(ctx context.Context) error {
	le.logger.Info("Running security checks locally")

	if !le.isGoProject() {
		return errors.New("not a Go project - security step requires Go modules")
	}

	// Try to run gosec if available
	if le.isCommandAvailable("gosec") {
		le.logger.Info("Running gosec security scanner")
		cmd := exec.CommandContext(ctx, "gosec", "./...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			le.logger.Warn("gosec found security issues", "error", err)
			// Don't fail the build for security issues in local mode
		}
	} else {
		le.logger.Info("gosec not available - skipping security scanning")
	}

	// Try to run govulncheck if available (Go 1.18+)
	if le.isCommandAvailable("govulncheck") {
		le.logger.Info("Running govulncheck")
		cmd := exec.CommandContext(ctx, "govulncheck", "./...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			le.logger.Warn("govulncheck found vulnerabilities", "error", err)
			// Don't fail the build for vulnerabilities in local mode
		}
	} else {
		le.logger.Info("govulncheck not available - skipping vulnerability check")
	}

	le.logger.Info("Security checks completed successfully")
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
		le.logger.Warn("Coverage file not found - skipping threshold check")
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
					le.logger.Info("Coverage check", "current", coverage, "threshold", threshold)
					if coverage < threshold {
						return fmt.Errorf("coverage %.2f%% is below threshold %.2f%%", coverage, threshold)
					}
					return nil
				}
			}
		}
	}

	le.logger.Warn("Could not parse coverage percentage - skipping threshold check")
	return nil
}
