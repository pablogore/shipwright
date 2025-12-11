package executors

import (
	"context"
	"errors"
	"fmt"

	"dagger.io/dagger"

	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines/shared"
)

// DockerExecutor executes pipeline steps using Dagger/Docker containers.
// This is the default executor that uses Dagger for containerized execution.
type DockerExecutor struct {
	client *dagger.Client
	config pipelines.Config
	src    pipelines.DaggerDirectory
}

// NewDockerExecutor creates a new DockerExecutor.
//
// Parameters:
//   - client: The Dagger client for container operations (can be nil for testing).
//   - config: The pipeline configuration.
//
// Returns:
//   - A new DockerExecutor instance.
func NewDockerExecutor(client *dagger.Client, config pipelines.Config) *DockerExecutor {
	var src pipelines.DaggerDirectory
	if client != nil {
		daggerClient := pipelines.NewDaggerAdapter(client)
		src = daggerClient.Host().Directory(".", pipelines.DaggerHostDirectoryOpts{
			Exclude: []string{"**/node_modules", "**/.git", "**/.dagger-cache"},
		})
	}

	return &DockerExecutor{
		client: client,
		config: config,
		src:    src,
	}
}

// Name returns the name of the executor.
func (e *DockerExecutor) Name() string {
	return "docker"
}

// CanExecute checks if Docker execution is possible.
// Docker execution requires a valid Dagger client.
func (e *DockerExecutor) CanExecute(ctx context.Context) bool {
	return e.client != nil
}

// ExecuteStep executes a single pipeline step using Dagger.
//
// Parameters:
//   - ctx: The context for managing execution.
//   - stepName: The name of the step to execute.
//
// Returns:
//   - An error if the step execution fails, otherwise nil.
func (e *DockerExecutor) ExecuteStep(ctx context.Context, stepName string) error {
	if e.client == nil {
		return errors.New("Docker executor requires Dagger client")
	}

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
	case "package":
		return e.executePackage(ctx)
	default:
		return fmt.Errorf("unsupported step for Docker execution: %s", stepName)
	}
}

// SupportsParallel indicates whether Docker executor supports parallel execution.
func (e *DockerExecutor) SupportsParallel() bool {
	return true
}

// GetCacheStrategy returns nil for Docker executor (cache handled by Dagger).
func (e *DockerExecutor) GetCacheStrategy() CacheStrategy {
	return nil
}

// executeSetup performs setup using Dagger.
func (e *DockerExecutor) executeSetup(ctx context.Context) error {
	// Setup is handled by pipeline Setup method, not executor
	return nil
}

// executeBuild performs build using Dagger (delegates to pipeline).
func (e *DockerExecutor) executeBuild(ctx context.Context) error {
	// Build logic is in pipeline, executor just coordinates
	return nil
}

// executeTest performs testing using Dagger.
func (e *DockerExecutor) executeTest(ctx context.Context) error {
	if e.src == nil {
		return errors.New("source directory not set")
	}

	if e.client == nil {
		return errors.New("Dagger client not available")
	}

	// Convert to adapter to get real client
	daggerClient := pipelines.NewDaggerAdapter(e.client)
	realClient := daggerClient.GetRealClient()

	srcAdapter, ok := e.src.(*pipelines.DaggerDirectoryAdapter)
	if !ok {
		return errors.New("invalid source directory adapter")
	}

	realSrc := srcAdapter.GetRealDirectory()
	goVersion := e.config.GoVersion
	if goVersion == "" {
		goVersion = "1.25.5"
	}

	return shared.RunTestsWithCoverage(ctx, realClient, realSrc, e.config.Coverage, goVersion)
}

// executeLint performs linting using Dagger.
func (e *DockerExecutor) executeLint(ctx context.Context) error {
	// Lint logic is in pipeline, executor coordinates
	return nil
}

// executeSecurity performs security scanning using Dagger.
func (e *DockerExecutor) executeSecurity(ctx context.Context) error {
	// Security logic is in pipeline, executor coordinates
	return nil
}

// executePackage performs packaging using Dagger.
func (e *DockerExecutor) executePackage(ctx context.Context) error {
	// Package logic is in pipeline, executor coordinates
	return nil
}
