package executors

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// NativeExecutor executes pipeline steps natively without Docker containers.
// It uses the Go toolchain and other tools installed on the host system.
type NativeExecutor struct {
	goVersion string
	workDir   string
}

// NewNativeExecutor creates a new NativeExecutor.
//
// Parameters:
//   - goVersion: The Go version to use (for compatibility checks).
//   - workDir: The working directory for execution.
//
// Returns:
//   - A new NativeExecutor instance.
func NewNativeExecutor(goVersion string, workDir string) *NativeExecutor {
	if workDir == "" {
		workDir = "."
	}
	return &NativeExecutor{
		goVersion: goVersion,
		workDir:   workDir,
	}
}

// Name returns the name of the executor.
func (e *NativeExecutor) Name() string {
	return "native"
}

// CanExecute checks if native execution is possible.
// Native execution requires Go to be installed on the system.
func (e *NativeExecutor) CanExecute(ctx context.Context) bool {
	// Check if Go is available
	cmd := exec.CommandContext(ctx, "go", "version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// ExecuteStep executes a single pipeline step natively.
//
// Parameters:
//   - ctx: The context for managing execution.
//   - stepName: The name of the step to execute.
//
// Returns:
//   - An error if the step execution fails, otherwise nil.
func (e *NativeExecutor) ExecuteStep(ctx context.Context, stepName string) error {
	switch stepName {
	case "setup":
		return e.executeSetup(ctx)
	case "build":
		return e.executeBuild(ctx)
	case "test":
		return e.executeTest(ctx)
	case "lint":
		return e.executeLint(ctx)
	case "security":
		return e.executeSecurity(ctx)
	default:
		return fmt.Errorf("unsupported step for native execution: %s", stepName)
	}
}

// SupportsParallel indicates whether native executor supports parallel execution.
func (e *NativeExecutor) SupportsParallel() bool {
	return true
}

// GetCacheStrategy returns nil for native executor (cache handled by CI/CD system).
func (e *NativeExecutor) GetCacheStrategy() CacheStrategy {
	return nil
}

// executeSetup performs native setup (go mod download, go mod tidy).
func (e *NativeExecutor) executeSetup(ctx context.Context) error {
	if !e.isGoProject() {
		return errors.New("not a Go project - setup step requires Go modules")
	}

	// Download dependencies
	cmd := exec.CommandContext(ctx, "go", "mod", "download")
	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download dependencies: %w", err)
	}

	// Tidy modules
	cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy modules: %w", err)
	}

	return nil
}

// executeBuild performs native build (go build).
func (e *NativeExecutor) executeBuild(ctx context.Context) error {
	if !e.isGoProject() {
		return errors.New("not a Go project - build step requires Go modules")
	}

	cmd := exec.CommandContext(ctx, "go", "build", "-v", "./...")
	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build application: %w", err)
	}

	return nil
}

// executeTest performs native testing (go test).
func (e *NativeExecutor) executeTest(ctx context.Context) error {
	if !e.isGoProject() {
		return errors.New("not a Go project - test step requires Go modules")
	}

	cmd := exec.CommandContext(ctx, "go", "test", "-v", "-race", "./...")
	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run tests: %w", err)
	}

	return nil
}

// executeLint performs native linting (golangci-lint if available).
func (e *NativeExecutor) executeLint(ctx context.Context) error {
	// Check if golangci-lint is available
	cmd := exec.CommandContext(ctx, "golangci-lint", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("golangci-lint not available: %w", err)
	}

	cmd = exec.CommandContext(ctx, "golangci-lint", "run", "./...")
	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("golangci-lint found issues: %w", err)
	}

	return nil
}

// executeSecurity performs native security scanning (govulncheck if available).
func (e *NativeExecutor) executeSecurity(ctx context.Context) error {
	// Check if govulncheck is available
	cmd := exec.CommandContext(ctx, "govulncheck", "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("govulncheck not available: %w", err)
	}

	cmd = exec.CommandContext(ctx, "govulncheck", "./...")
	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("govulncheck found vulnerabilities: %w", err)
	}

	return nil
}

// isGoProject checks if the current directory is a Go project.
func (e *NativeExecutor) isGoProject() bool {
	goModPath := filepath.Join(e.workDir, "go.mod")
	_, err := os.Stat(goModPath)
	return err == nil
}
