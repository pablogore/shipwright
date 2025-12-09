// Package goservice provides the go-service pipeline implementation.
// This is a generic pipeline for Go microservices.
package goservice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"dagger.io/dagger"

	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines/shared"
)

// Pipeline represents a generic pipeline for Go microservices.
//
// Fields:
//   - Client: The Dagger client used for container operations.
//   - Config: The configuration for the pipeline.
//   - Src: The source directory of the cloned repository.
//   - Cloner: The cloner used for cloning the repository.
//   - Image: The Docker image container built during the Package step.
type Pipeline struct {
	Client pipelines.DaggerClient
	Config pipelines.Config
	Src    pipelines.DaggerDirectory
	Cloner shared.Cloner
	Image  *dagger.Container
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
	return "go-service"
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

// Test runs tests for the Go microservice with coverage.
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
	fmt.Println("🧪 running tests for Go microservice...")
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

// Lint runs golangci-lint on the Go microservice source code.
// It uses the .golangci.yml configuration file if present.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - An error if the linting process fails, otherwise nil.
func (p *Pipeline) Lint(ctx context.Context) error {
	if p.Src == nil {
		return errors.New("pipeline not set up: source directory is nil")
	}
	fmt.Println("🔍 Running golangci-lint on Go microservice...")
	// Extract real types for shared functions (only if using adapter, not mocks)
	if adapter, ok := p.Client.(*pipelines.DaggerAdapter); ok {
		realClient := adapter.GetRealClient()
		if srcAdapter, ok := p.Src.(*pipelines.DaggerDirectoryAdapter); ok {
			realSrc := srcAdapter.GetRealDirectory()
			goVersion := p.Config.GoVersion
			if goVersion == "" {
				goVersion = "1.25.1"
			}

			// Create a container with Go and golangci-lint
			// Use golangci/golangci-lint image which includes golangci-lint
			container := realClient.Container().
				From("golangci/golangci-lint:latest").
				WithMountedDirectory("/app", realSrc).
				WithWorkdir("/app").
				WithEnvVariable("GO111MODULE", "on").
				WithEnvVariable("CGO_ENABLED", "0")

			// Check if .golangci.yml exists in the source
			// If it exists, golangci-lint will use it automatically
			// Run golangci-lint with default timeout (5 minutes)
			// The timeout can be configured in .golangci.yml or via command line
			_, err := container.
				WithExec([]string{"golangci-lint", "run", "--timeout", "5m", "./..."}).
				Sync(ctx)
			if err != nil {
				return fmt.Errorf("golangci-lint found issues: %w", err)
			}

			fmt.Println("✅ Linting completed successfully")
			return nil
		}
	}
	// For mocks, return an error indicating this requires real client
	return errors.New("Lint method requires real Dagger client, not mock")
}

// Vuln runs vulnerability scanning on the Go microservice using govulncheck.
// It checks for known vulnerabilities in dependencies and the Go standard library.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - An error if vulnerabilities are found, otherwise nil.
func (p *Pipeline) Vuln(ctx context.Context) error {
	if p.Src == nil {
		return errors.New("pipeline not set up: source directory is nil")
	}
	fmt.Println("🔒 Running vulnerability scan on Go microservice...")
	// Extract real types for shared functions (only if using adapter, not mocks)
	if adapter, ok := p.Client.(*pipelines.DaggerAdapter); ok {
		realClient := adapter.GetRealClient()
		if srcAdapter, ok := p.Src.(*pipelines.DaggerDirectoryAdapter); ok {
			realSrc := srcAdapter.GetRealDirectory()
			goVersion := p.Config.GoVersion
			if goVersion == "" {
				goVersion = "1.25.1"
			}

			// Create a container with Go and install govulncheck
			// Use golang image which includes Go toolchain
			container := realClient.Container().
				From("golang:" + goVersion).
				WithMountedDirectory("/app", realSrc).
				WithWorkdir("/app").
				WithEnvVariable("GO111MODULE", "on").
				WithEnvVariable("CGO_ENABLED", "0")

			// Install govulncheck
			container = container.WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"})

			// Run govulncheck on all packages
			output, err := container.
				WithExec([]string{"govulncheck", "./..."}).
				Stdout(ctx)
			if err != nil {
				// Check if the error is due to vulnerabilities found
				// govulncheck returns non-zero exit code when vulnerabilities are found
				stderr, _ := container.
					WithExec([]string{"govulncheck", "./..."}).
					Stderr(ctx)
				
				// Check if vulnerabilities were found in the output
				combinedOutput := output + stderr
				if strings.Contains(combinedOutput, "Your code is affected") || 
				   strings.Contains(combinedOutput, "Vulnerabilities found") {
					return fmt.Errorf("security vulnerabilities detected:\n%s", combinedOutput)
				}
				// If it's a different error, return it
				return fmt.Errorf("govulncheck failed: %w\nOutput: %s", err, combinedOutput)
			}

			// Check output for vulnerabilities even if exit code is 0
			if strings.Contains(output, "Your code is affected") || 
			   strings.Contains(output, "Vulnerabilities found") {
				return fmt.Errorf("security vulnerabilities detected:\n%s", output)
			}

			fmt.Println("✅ No security vulnerabilities found")
			return nil
		}
	}
	// For mocks, return an error indicating this requires real client
	return errors.New("Vuln method requires real Dagger client, not mock")
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

// Package creates a Docker image from the built Go binary.
// It reads the binary from bin/app and creates a minimal Docker image.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - An error if the packaging process fails, otherwise nil.
func (p *Pipeline) Package(ctx context.Context) error {
	if p.Src == nil {
		return errors.New("pipeline not set up: source directory is nil")
	}

	fmt.Println("📦 Packaging Go binary into Docker image...")

	// Extract real types for shared functions (only if using adapter, not mocks)
	if adapter, ok := p.Client.(*pipelines.DaggerAdapter); ok {
		realClient := adapter.GetRealClient()
		if _, ok := p.Src.(*pipelines.DaggerDirectoryAdapter); ok {
			// Read the built binary from bin/app on the host
			// The binary was exported to the host during the Build step
			binaryDir := realClient.Host().Directory("bin")
			binaryFile := binaryDir.File("app")

			// Create a minimal Docker image with the binary
			// Using alpine:latest as base for a small but functional image
			image := realClient.Container().
				From("alpine:latest").
				WithFile("/app", binaryFile).
				WithExec([]string{"chmod", "+x", "/app"}).
				WithEntrypoint([]string{"/app"})

			p.Image = image
			fmt.Println("✅ Docker image created successfully")
			return nil
		}
	}
	// For mocks, return an error indicating this requires real client
	return errors.New("Package method requires real Dagger client, not mock")
}

// Tag generates a version tag.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - An error if the tagging process fails, otherwise nil.
func (p *Pipeline) Tag(ctx context.Context) error {
	fmt.Println("🧪 Generating tag...")
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

// Push publishes the Docker image to the container registry.
// It uses the registry configuration from the pipeline config.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - An error if the push process fails, otherwise nil.
func (p *Pipeline) Push(ctx context.Context) error {
	if p.Image == nil {
		return errors.New("no image to push - run Package step first")
	}

	fmt.Println("📤 Pushing Docker image to registry...")

	// Extract real types for shared functions (only if using adapter, not mocks)
	if adapter, ok := p.Client.(*pipelines.DaggerAdapter); ok {
		realClient := adapter.GetRealClient()

		// Determine registry configuration
		registry := p.Config.Registry
		if registry == "" {
			registry = p.Config.RegistryURL
		}
		if registry == "" {
			return errors.New("registry URL not configured")
		}

		// Determine image tag
		tag := p.Config.ImageTag
		if tag == "" {
			// Try to read from .tag_name file or use default
			if p.Config.BuildTag != "" {
				tag = p.Config.BuildTag
			} else {
				tag = "latest"
			}
		}

		// Determine image name
		imageName := p.Config.ImageName
		if imageName == "" {
			imageName = p.Name()
		}

		fullImageRef := fmt.Sprintf("%s/%s:%s", registry, imageName, tag)

		// Get credentials
		var username string
		var secret *dagger.Secret

		// Check if we're in CI environment
		isCI := isCIEnvironment()
		if isCI {
			// Try GitHub token first
			if token := os.Getenv("GITHUB_TOKEN"); token != "" {
				username = "x-access-token"
				secret = realClient.SetSecret("github-token", token)
				fmt.Println("🔐 using GitHub CI authentication")
			} else if token := os.Getenv("CI_JOB_TOKEN"); token != "" {
				// GitLab CI token
				username = "gitlab-ci-token"
				secret = realClient.SetSecret("ci-job-token", token)
				fmt.Println("🔐 using GitLab CI authentication")
			} else {
				return errors.New("no CI token available (GITHUB_TOKEN or CI_JOB_TOKEN)")
			}
		} else {
			// Local environment - use config credentials
			username = p.Config.RegistryUser
			if username == "" {
				return errors.New("registry user not configured for local environment")
			}
			password := p.Config.RegistryPass
			if password == "" {
				return errors.New("registry password not configured for local environment")
			}
			secret = realClient.SetSecret("local-registry-password", password)
			fmt.Println("🔐 using local authentication")
		}

		// Add registry authentication and publish
		imageWithAuth := p.Image.WithRegistryAuth(registry, username, secret)
		url, err := imageWithAuth.Publish(ctx, fullImageRef)
		if err != nil {
			return fmt.Errorf("failed to publish image: %w", err)
		}

		fmt.Printf("✅ Image published successfully: %s\n", url)
		return nil
	}
	// For mocks, return an error indicating this requires real client
	return errors.New("Push method requires real Dagger client, not mock")
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
