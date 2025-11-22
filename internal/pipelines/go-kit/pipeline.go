// Package gokit provides the go-kit pipeline implementation.
package gokit

import (
	"context"
	"errors"
	"fmt"
	"os"

	"dagger.io/dagger"

	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines/shared"
)

// Pipeline represents a pipeline for the go-kit project.
//
// Fields:
//   - Client: The Dagger client used for container operations.
//   - Config: The configuration for the pipeline.
//   - Src: The source directory of the cloned repository.
//   - Cloner: The cloner used for cloning the repository.
type Pipeline struct {
	Client pipelines.DaggerClient
	Config pipelines.Config
	Src    pipelines.DaggerDirectory
	Cloner shared.Cloner
}

// New creates a new instance of Pipeline.
//
// Parameters:
//   - client: The Dagger client used for container operations.
//   - cfg: The configuration for the pipeline.
//
// Returns:
//   - A new instance of Pipeline.
func New(client *dagger.Client, cfg pipelines.Config) pipelines.Pipeline {
	var daggerClient pipelines.DaggerClient
	var src pipelines.DaggerDirectory
	var cloner shared.Cloner

	// Handle nil client gracefully
	if client != nil {
		// Convert real Dagger client to our interface using adapter
		daggerClient = pipelines.NewDaggerAdapter(client)
		src = daggerClient.Host().Directory(".", pipelines.DaggerHostDirectoryOpts{
			Exclude: []string{"**/node_modules", "**/.git", "**/.dagger-cache"},
		})

		if cfg.GitProtocol == "ssh" {
			cloner = &shared.SSHCloner{}
		} else {
			cloner = &shared.HTTPSCloner{}
		}
	}
	// If client is nil, all fields will remain nil

	return &Pipeline{
		Client: daggerClient,
		Config: cfg,
		Src:    src,
		Cloner: cloner,
	}
}

// Name returns the name of the pipeline.
//
// Returns:
//   - A string representing the name of the pipeline.
func (p *Pipeline) Name() string {
	return "go-kit"
}

// Setup prepares the Go microservice pipeline.
// In CI environments, it uses the current directory directly.
// In local environments, it clones the repository if needed.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - An error if the setup process fails, otherwise nil.
func (p *Pipeline) Setup(ctx context.Context) error {
	// Check if client is nil or not an adapter
	if p.Client == nil {
		return errors.New("Setup method requires real Dagger client, not nil")
	}

	// Check if we're in a CI environment
	isCI := isCIEnvironment()

	if isCI {
		// In CI, we're already in the repository, use current directory
		if adapter, ok := p.Client.(*pipelines.DaggerAdapter); ok {
			realClient := adapter.GetRealClient()
			p.Src = pipelines.NewDaggerAdapter(realClient).Host().Directory(".", pipelines.DaggerHostDirectoryOpts{
				Exclude: []string{"**/node_modules", "**/.git", "**/.dagger-cache"},
			})
			return nil
		}
		return errors.New("Setup method requires real Dagger client, not mock")
	}

	// In local environment, clone if cloner is available
	if p.Cloner != nil {
		// Check if client is an adapter (real client) or mock
		if adapter, ok := p.Client.(*pipelines.DaggerAdapter); ok {
			// Extract real client from adapter
			realClient := adapter.GetRealClient()
			_, err := p.Cloner.Clone(ctx, realClient, shared.GitCloneOpts{})
			if err != nil {
				return err
			}
			// Convert real directory to our interface
			p.Src = pipelines.NewDaggerAdapter(realClient).Host().Directory(".", pipelines.DaggerHostDirectoryOpts{})
		} else {
			// This is a mock client, return error
			return errors.New("Setup method requires real Dagger client, not mock")
		}
	}
	return nil
}

// isCIEnvironment checks if we're running in a CI environment.
func isCIEnvironment() bool {
	// Check common CI environment variables
	ciVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"JENKINS_URL",
		"CIRCLECI",
		"TRAVIS",
		"BUILDKITE",
		"TEAMCITY_VERSION",
	}

	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// Test runs the tests for the go-kit project with coverage.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - An error if the tests fail, otherwise nil.
func (p *Pipeline) Test(ctx context.Context) error {
	if p.Src == nil {
		return errors.New("pipeline not set up: source directory is nil")
	}
	fmt.Println("🧪 running tests for go-kit...")
	// Extract real types for shared functions (only if using adapter, not mocks)
	if adapter, ok := p.Client.(*pipelines.DaggerAdapter); ok {
		realClient := adapter.GetRealClient()
		if srcAdapter, ok := p.Src.(*pipelines.DaggerDirectoryAdapter); ok {
			realSrc := srcAdapter.GetRealDirectory()
			return shared.RunTestsWithCoverage(ctx, realClient, realSrc, p.Config.Coverage, p.Config.GoVersion)
		}
	}
	// For mocks, return an error indicating this requires real client
	return errors.New("Test method requires real Dagger client, not mock")
}

// Build is a placeholder for the build step of the pipeline.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - Always returns nil as it is not implemented.
func (p *Pipeline) Build(ctx context.Context) error {
	if p.Src == nil {
		return errors.New("pipeline not set up: source directory is nil")
	}
	// Extract real types for shared functions (only if using adapter, not mocks)
	if adapter, ok := p.Client.(*pipelines.DaggerAdapter); ok {
		realClient := adapter.GetRealClient()
		if srcAdapter, ok := p.Src.(*pipelines.DaggerDirectoryAdapter); ok {
			realSrc := srcAdapter.GetRealDirectory()
			goVersion := p.Config.GoVersion
			if goVersion == "" {
				goVersion = "1.25.1"
			}
			builder := shared.NewGoBuilder(realClient, realSrc, goVersion)
			outPath := "bin/app"
			_, err := builder.Build(ctx, outPath, outPath, map[string]string{"CGO_ENABLED": "0"})
			if err != nil {
				return fmt.Errorf("failed to build Go binary: %w", err)
			}
			fmt.Printf("✅ Binary built successfully at %s\n", outPath)
			return nil
		}
	}
	// For mocks, return an error indicating this requires real client
	return errors.New("Build method requires real Dagger client, not mock")
}

// Package is a placeholder for the packaging step of the pipeline.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - Always returns nil as it is not implemented.
func (p *Pipeline) Package(_ context.Context) error {
	return errors.New("not implemented")
}

// Tag generates a tag for the go-kit project.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - An error if the tagging process fails, otherwise nil.
func (p *Pipeline) Tag(ctx context.Context) error {
	fmt.Println("🧪 Generating tag for go-kit...")
	// Extract real types for shared functions (only if using adapter, not mocks)
	if adapter, ok := p.Client.(*pipelines.DaggerAdapter); ok {
		realClient := adapter.GetRealClient()
		if srcAdapter, ok := p.Src.(*pipelines.DaggerDirectoryAdapter); ok {
			realSrc := srcAdapter.GetRealDirectory()
			tag, err := shared.GenerateTag(ctx, realClient, realSrc)
			if err != nil {
				return fmt.Errorf("❌ Error generating tag: %w", err)
			}
			fmt.Printf("✅ Tag generated: %s\n", tag)
			return nil
		}
	}
	// For mocks, return an error indicating this requires real client
	return errors.New("Tag method requires real Dagger client, not mock")
}

// Push is a placeholder for the push step of the pipeline.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - Always returns nil as it is not implemented.
func (p *Pipeline) Push(_ context.Context) error {
	return errors.New("not implemented")
}

// BeforeStep is a hook that executes custom logic before a specific pipeline step.
//
// Parameters:
//   - ctx: The context for managing execution.
//   - step: The name of the step.
//
// Returns:
//   - Always returns nil as it is not implemented.
func (p *Pipeline) BeforeStep(_ context.Context, _ string) pipelines.HookFunc {
	return nil
}

// AfterStep is a hook that executes custom logic after a specific pipeline step.
//
// Parameters:
//   - ctx: The context for managing execution.
//   - step: The name of the step.
//
// Returns:
//   - Always returns nil as it is not implemented.
func (p *Pipeline) AfterStep(_ context.Context, _ string) pipelines.HookFunc {
	return nil
}

func (p *Pipeline) Cleanup(_ context.Context, _ *dagger.Client) error {
	return errors.New("not implemented")
}
