package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"dagger.io/dagger"

	"github.com/pablogore/kit-logger/pkg/logger"

	golang "github.com/pablogore/shipwright/providers/go"
	godaggerkit "github.com/pablogore/shipwright/providers/go/daggerkit"

	"github.com/pablogore/shipwright/internal/config"
	"github.com/pablogore/shipwright/internal/executors"
	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/pablogore/shipwright/internal/plugins"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// Static errors for err113 compliance.
var (
	ErrComponentNotFound          = errors.New("component not found")
	ErrFailedToCreateDaggerClient = errors.New("failed to create Dagger client")
	ErrInvalidConfiguration       = errors.New("invalid configuration")
)

// Container manages dependency injection for the application.
type Container struct {
	ctx    context.Context
	config interfaces.Configuration
	once   map[string]*sync.Once
	cache  map[string]any
}

// NewContainer creates a new container instance.
func NewContainer(ctx context.Context, config interfaces.Configuration) *Container {
	return &Container{
		ctx:    ctx,
		config: config,
		once:   make(map[string]*sync.Once),
		cache:  make(map[string]any),
	}
}

// Register registers a component factory function.
func (c *Container) Register(name string, factory func() (any, error)) {
	c.once[name] = &sync.Once{}
	c.cache[name] = factory
}

// Get retrieves a component, creating it if necessary.
func (c *Container) Get(name string) (any, error) {
	factory, exists := c.cache[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrComponentNotFound, name)
	}

	once, exists := c.once[name]
	if !exists {
		return c.cache[name], nil
	}

	var err error
	once.Do(func() {
		if fn, ok := factory.(func() (any, error)); ok {
			c.cache[name], err = fn()
		}
	})
	if err != nil {
		return nil, err
	}

	return c.cache[name], nil
}

// Start initializes all registered components.
func (c *Container) Start(_ context.Context) error {
	c.registerComponents()
	return nil
}

// Stop stops the container and cleans up resources.
func (c *Container) Stop(_ context.Context) error {
	// Clean up components that need cleanup
	for name, component := range c.cache {
		if closer, ok := component.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				// Log error but continue cleanup
				logger.L().ErrorContext(context.Background(), "Failed to close component",
					"component", name,
					"error", err)
			}
		}
	}
	return nil
}

// Validate validates the container configuration.
func (c *Container) Validate() error {
	return c.config.Validate()
}

// registerComponents registers all application components.
func (c *Container) registerComponents() {
	c.registerDaggerComponents()
	c.registerSecurityComponents()
	c.registerLoggingComponents()
	c.registerExecutorComponents()
	c.registerStepComponents()
	c.registerHookComponents()
	c.registerPluginComponents()
}

// registerDaggerComponents registers Dagger-related components.
func (c *Container) registerDaggerComponents() {
	// Dagger Client
	c.Register("daggerClient", func() (any, error) {
		timeout := c.config.GetDuration("dagger.timeout")
		if timeout == 0 {
			// Increased default timeout to 120 seconds for CI environments
			// where Dagger engine may take longer to start, especially on first run
			// The daemon needs time to pull images and initialize
			timeout = 120 * time.Second
		}

		// Use a longer context for connection attempts
		// This allows the Dagger engine time to start up
		ctx, cancel := context.WithTimeout(c.ctx, timeout)
		defer cancel()

		// Verify Docker is available before attempting to connect
		// Dagger requires Docker to be running
		if err := c.verifyDockerAvailable(ctx); err != nil {
			return nil, fmt.Errorf("%w: Docker not available: %v. "+
				"Dagger requires Docker to be running. In GitHub Actions, ensure Docker service is configured",
				ErrFailedToCreateDaggerClient, err)
		}

		// Give Docker a moment to be fully ready before attempting Dagger connection
		// This helps avoid race conditions where Docker is available but not fully initialized
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: context cancelled while waiting for Docker to be ready",
				ErrFailedToCreateDaggerClient)
		case <-time.After(2 * time.Second):
			// Continue with connection attempt
		}

		// Retry connection with exponential backoff
		// This helps handle cases where the engine is still starting
		// Increased retries and delays to handle slow daemon startup
		maxRetries := 10
		retryDelay := 3 * time.Second

		var client *dagger.Client
		var lastErr error

		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				// Wait before retrying (exponential backoff)
				waitTime := retryDelay * time.Duration(1<<uint(attempt-1))
				if waitTime > 15*time.Second {
					waitTime = 15 * time.Second // Cap at 15 seconds
				}

				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("%w: context deadline exceeded after %d attempts: %v",
						ErrFailedToCreateDaggerClient, attempt, lastErr)
				case <-time.After(waitTime):
					// Continue with retry
				}
			}

			// Connect to Dagger engine (which uses Docker)
			// dagger.Connect() automatically manages the Dagger engine via Docker
			// It may take time for the daemon to start, especially on first run
			client, lastErr = dagger.Connect(ctx, dagger.WithLogOutput(nil))
			if lastErr == nil {
				// Connection successful, but verify daemon is actually ready
				// by performing a simple operation
				verifyCtx, verifyCancel := context.WithTimeout(ctx, 10*time.Second)
				_, verifyErr := client.Container().From("alpine:latest").ID(verifyCtx)
				verifyCancel()

				if verifyErr == nil {
					// Daemon is ready and responding
					return client, nil
				}

				// Connection succeeded but daemon not ready yet
				// Close the client and retry
				client.Close()
				lastErr = fmt.Errorf("daemon connected but not ready: %w", verifyErr)
			}

			// Log retry attempt (if logger is available)
			if attempt < maxRetries {
				logger.L().WarnContext(ctx, "Dagger connection attempt failed, retrying",
					"attempt", attempt+1,
					"max_retries", maxRetries+1,
					"error", lastErr)
			}
		}

		// All retries failed
		return nil, fmt.Errorf("%w: failed after %d attempts: %v. "+
			"Ensure Dagger engine is running (try 'dagger run echo test' to verify). "+
			"In GitHub Actions, ensure Docker service is available. "+
			"The daemon may take up to 60 seconds to start on first run",
			ErrFailedToCreateDaggerClient, maxRetries+1, lastErr)
	})
}

// verifyDockerAvailable checks if Docker is available and running.
// This is required for Dagger to function properly.
func (c *Container) verifyDockerAvailable(ctx context.Context) error {
	// Try to execute a simple Docker command to verify it's available
	// Use a short timeout to avoid hanging
	verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Check if docker command is available by trying to get version
	// This is a lightweight check that doesn't require Docker daemon to be fully ready
	// but indicates Docker CLI is available
	cmd := exec.CommandContext(verifyCtx, "docker", "version", "--format", "{{.Server.Version}}")
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Docker might not be available or daemon not running
		// Return a helpful error message
		return fmt.Errorf("docker not available or daemon not running: %v (output: %s). "+
			"In GitHub Actions, add 'services: docker:' to your workflow",
			err, string(output))
	}

	// Docker is available
	if len(output) > 0 {
		logger.L().InfoContext(ctx, "Docker is available", "version", strings.TrimSpace(string(output)))
	}
	return nil
}

// registerSecurityComponents registers security-related components.
func (c *Container) registerSecurityComponents() {
	// Vulnerability Checker
	c.Register("vulnChecker", func() (any, error) {
		return NewVulnChecker(c.config), nil
	})

	// Linter
	c.Register("linter", func() (any, error) {
		return NewLinter(c.config), nil
	})
}

// CreateLogger initializes go-kit-logger with configuration.
func (c *Container) CreateLogger() error {
	loggingConfig := c.config.Logging()

	// Initialize go-kit-logger with configuration
	opts := []func(*logger.Config){
		logger.WithLevel(loggingConfig.Level),
	}

	if loggingConfig.Format != "" {
		opts = append(opts, logger.WithEncoding(loggingConfig.Format))
	}

	// Create logger with options
	_ = logger.NewLogger(opts...)

	return nil
}

// registerLoggingComponents registers logging-related components.
func (c *Container) registerLoggingComponents() {
	// Initialize go-kit-logger
	if err := c.CreateLogger(); err != nil {
		// If logger initialization fails, use default logger
		_ = logger.NewLogger()
	}
	// No need to register logger - use logger.L() directly throughout the codebase
}

// registerExecutorComponents registers executor-related components.
func (c *Container) registerExecutorComponents() {
	// Executor Selector
	c.Register("executorSelector", func() (any, error) {
		selector := executors.NewSelector()

		// Get Dagger client (may be nil)
		client, err := c.Get("daggerClient")
		var daggerClient *dagger.Client
		if err == nil && client != nil {
			daggerClient = client.(*dagger.Client)
		}

		// Convert config to pipelines.Config for executors
		pipelineConfig := ConvertConfigToPipelinesConfig(c.config)

		// Register native executor
		nativeExecutor := executors.NewNativeExecutor(
			c.config.Pipeline().GoVersion,
			".",
		)
		selector.RegisterExecutor("native", nativeExecutor)

		// Register Docker executor
		dockerExecutor := executors.NewDockerExecutor(daggerClient, pipelineConfig)
		selector.RegisterExecutor("docker", dockerExecutor)

		return selector, nil
	})
}

// GetDaggerClient returns the cached Dagger client.
// Returns the cached Dagger client. Connection management is handled by Dagger internally.
func (c *Container) GetDaggerClient() (*dagger.Client, error) {
	client, err := c.Get("daggerClient")
	if err != nil {
		return nil, fmt.Errorf("failed to get Dagger client: %w", err)
	}
	if client == nil {
		return nil, errors.New("dagger client is nil")
	}
	return client.(*dagger.Client), nil
}

// GetRegistryConfig implements RegistryProvider interface.
func (c *Container) GetRegistryConfig() (interfaces.RegistryConfig, error) {
	return c.config.Registry(), nil
}

// GetRegistryAuth implements RegistryProvider interface.
func (c *Container) GetRegistryAuth() (string, string, error) {
	registry := c.config.Registry()
	return registry.User, registry.Pass, nil
}

// GetVulnChecker implements SecurityProvider interface.
func (c *Container) GetVulnChecker() (interfaces.VulnChecker, error) {
	checker, err := c.Get("vulnChecker")
	if err != nil {
		return nil, err
	}
	return checker.(interfaces.VulnChecker), nil
}

// GetLinter implements SecurityProvider interface.
func (c *Container) GetLinter() (interfaces.Linter, error) {
	linter, err := c.Get("linter")
	if err != nil {
		return nil, err
	}
	return linter.(interfaces.Linter), nil
}

// GetLogger implements LoggingProvider interface.
// GetLogger is deprecated - use logger.L() directly from go-kit-logger
// This method is kept for backward compatibility but returns nil
func (c *Container) GetLogger() (interfaces.Logger, error) {
	return nil, errors.New("GetLogger is deprecated - use logger.L() directly from go-kit-logger")
}

// GetConfiguration returns the configuration instance.
func (c *Container) GetConfiguration() interfaces.Configuration {
	return c.config
}

// VulnChecker implements vulnerability checking.
type VulnChecker struct {
	config interfaces.Configuration
}

// NewVulnChecker creates a new vulnerability checker.
func NewVulnChecker(config interfaces.Configuration) *VulnChecker {
	return &VulnChecker{config: config}
}

// Check runs vulnerability checks on the source code.
func (v *VulnChecker) Check(_ context.Context, _ *dagger.Directory) error {
	// Implementation will be added later
	return nil
}

// GetReport returns the vulnerability report.
func (v *VulnChecker) GetReport(_ context.Context) (string, error) {
	// Implementation will be added later
	return "", nil
}

// Linter implements code linting.
type Linter struct {
	config interfaces.Configuration
}

// NewLinter creates a new linter.
func NewLinter(config interfaces.Configuration) *Linter {
	return &Linter{config: config}
}

// Lint runs linting on the source code.
func (l *Linter) Lint(_ context.Context, _ *dagger.Directory) error {
	// Implementation will be added later
	return nil
}

// GetReport returns the linting report.
func (l *Linter) GetReport(_ context.Context) (string, error) {
	// Implementation will be added later
	return "", nil
}

// Logger implements structured logging.
type Logger struct {
	config interfaces.Configuration
}

// NewLogger creates a new logger.
func NewLogger(config interfaces.Configuration) *Logger {
	return &Logger{config: config}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, fields ...any) {
	logger.L().DebugContext(context.Background(), msg, fields...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, fields ...any) {
	logger.L().InfoContext(context.Background(), msg, fields...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, fields ...any) {
	logger.L().WarnContext(context.Background(), msg, fields...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, fields ...any) {
	logger.L().ErrorContext(context.Background(), msg, fields...)
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(msg string, fields ...any) {
	logger.L().ErrorContext(context.Background(), msg, fields...)
	// In a real implementation, this would call os.Exit(1)
}

// WithField adds a field to the logger.
func (l *Logger) WithField(_ string, _ any) interfaces.Logger {
	// Simple implementation - in real scenario would return a new logger instance
	return l
}

// WithFields adds multiple fields to the logger.
func (l *Logger) WithFields(_ map[string]any) interfaces.Logger {
	// Simple implementation - in real scenario would return a new logger instance
	return l
}

// registerStepComponents registers step-related components.
func (c *Container) registerStepComponents() {
	// Step Registry
	c.Register("stepRegistry", func() (any, error) {
		registry := NewStepRegistry()

		// Get logger and dagger client
		client, err := c.Get("daggerClient")
		if err != nil {
			// Dagger client might not be available, pass nil
			client = nil
		}
		var daggerClient *dagger.Client
		if client != nil {
			daggerClient = client.(*dagger.Client)
		}

		// Register default steps with client (logger is used directly via logger.L())
		_ = registry.RegisterStep("setup", NewSetupStepHandler(c.config, daggerClient))
		_ = registry.RegisterStep("build", NewBuildStepHandler(c.config, daggerClient))
		_ = registry.RegisterStep("test", NewTestStepHandler(c.config, daggerClient))
		_ = registry.RegisterStep("lint", NewLintStepHandler(c.config, daggerClient))
		_ = registry.RegisterStep("security", NewSecurityStepHandler(c.config, daggerClient))
		_ = registry.RegisterStep("tag", NewTagStepHandler(c.config, daggerClient))
		_ = registry.RegisterStep("package", NewPackageStepHandler(c.config, daggerClient))
		_ = registry.RegisterStep("push", NewPushStepHandler(c.config, daggerClient))
		_ = registry.RegisterStep("release", NewReleaseStepHandler(c.config, daggerClient))

		return registry, nil
	})

	// Pipeline Executor
	c.Register("pipelineExecutor", func() (any, error) {
		registry, err := c.Get("stepRegistry")
		if err != nil {
			return nil, err
		}

		hookManager, err := c.Get("hookManager")
		if err != nil {
			return nil, err
		}

		return NewPipelineExecutor(registry.(interfaces.StepRegistry), hookManager.(interfaces.HookManager)), nil
	})
}

// registerPluginComponents registers plugin-related components.
func (c *Container) registerPluginComponents() {
	// Plugin Loader
	c.Register("pluginLoader", func() (any, error) {
		loader := plugins.NewLoader()

		// Register built-in plugins
		loader.RegisterBuiltin("nomad-deploy", plugins.NewNomadDeployPlugin)

		return loader, nil
	})

	// Plugin Registry
	c.Register("pluginRegistry", func() (any, error) {
		loader, err := c.Get("pluginLoader")
		if err != nil {
			return nil, fmt.Errorf("failed to get plugin loader: %w", err)
		}

		return plugins.NewRegistry(loader.(plugins.PluginLoader)), nil
	})
}

// registerHookComponents registers hook-related components.
func (c *Container) registerHookComponents() {
	// Hook Manager
	c.Register("hookManager", func() (any, error) {
		return NewHookManager(), nil
	})
}

// validateConvertedConfig validates URLs and versions in the configuration.
// This is called after convertConfig to ensure all values are valid.
func validateConvertedConfig(cfg interfaces.Configuration) error {
	// Validate registry URL if provided
	registryURL := cfg.GetString("registry.base_url")
	if registryURL != "" {
		if err := config.ValidateRegistryURL(registryURL); err != nil {
			return fmt.Errorf("invalid registry URL: %w", err)
		}
	}

	// Validate Go version if provided
	goVersion := cfg.GetString("pipeline.go_version")
	if goVersion != "" {
		if err := config.ValidateGoVersion(goVersion); err != nil {
			return fmt.Errorf("invalid Go version: %w", err)
		}
	}

	// Validate Git repository URL if provided
	gitRepo := cfg.GetString("git.repo")
	if gitRepo != "" {
		if err := config.ValidateGitRepoURL(gitRepo); err != nil {
			return fmt.Errorf("invalid Git repository URL: %w", err)
		}
	}

	// Validate environment if provided
	env := cfg.Environment()
	if env != "" {
		if err := config.ValidateEnvironment(env); err != nil {
			return fmt.Errorf("invalid environment: %w", err)
		}
	}

	return nil
}

// convertConfig is a private helper that converts interfaces.Configuration to pipelines.Config.
// This is used internally and in tests.
func convertConfig(cfg interfaces.Configuration) pipelines.Config {
	return ConvertConfigToPipelinesConfig(cfg)
}

// ConvertConfigToPipelinesConfig converts interfaces.Configuration to pipelines.Config.
// This function is exported to allow external packages to convert configuration.
func ConvertConfigToPipelinesConfig(cfg interfaces.Configuration) pipelines.Config {
	return pipelines.Config{
		Env:           cfg.Environment(),
		SkipPush:      cfg.GetBool("pipeline.skip_push"),
		OnlyTest:      cfg.GetBool("pipeline.only_test"),
		OnlyBuild:     cfg.GetBool("pipeline.only_build"),
		Verbose:       cfg.GetBool("pipeline.verbose"),
		GitRepo:       cfg.GetString("git.repo"),
		GitRef:        cfg.GetString("git.ref"),
		GitProtocol:   cfg.GetString("git.protocol"),
		GitUserEmail:  cfg.GetString("git.user_email"),
		GitUserName:   cfg.GetString("git.user_name"),
		RegistryURL:   cfg.GetString("registry.base_url"),
		RegistryUser:  cfg.GetString("registry.user"),
		RegistryPass:  cfg.GetString("registry.pass"),
		BuildTag:      cfg.GetString("registry.tag"),
		CommitSHA:     cfg.GetString("git.ref"),
		BranchName:    cfg.GetString("git.ref"),
		Token:         cfg.GetString("registry.pass"),
		Coverage:      cfg.GetFloat("pipeline.coverage"),
		GoVersion:     cfg.GetString("pipeline.go_version"),
		JavaVersion:   cfg.GetString("pipeline.java_version"),
		SSHPrivateKey: cfg.GetString("git.ssh_key"),
	}
}

// BuildCapabilities constructs the Layer 1 capability bundle
// (pkg/shipwright: Builder/Tester/Artifactor/Deployer/Runner) backed by the
// standalone providers/go implementations (originally Phase 3's go-service
// decomposition, extracted from internal/capabilities into its own module),
// wired from the same pipelines.Config the DI container
// already produces for the legacy --pipeline path. This is what
// PluginContext.GetCapabilities() (replacing the retired GetPipeline(),
// WU10 tasks.md 10.1/10.2) exposes to plugins.
//
// Deployer and Runner are always left nil: no concrete implementation
// exists yet for either (pkg/shipwright.DeployConfig/RunConfig are empty
// at this change, per design.md D-D) — Capabilities' fields are
// independently optional by design, so a partial bundle is correct, not a
// gap. When client is nil (no Dagger connection, e.g. most non-container
// CLI paths), an empty Capabilities{} is returned rather than constructing
// implementations against a nil client.
func BuildCapabilities(client *dagger.Client, cfg pipelines.Config) plugins.Capabilities {
	if client == nil {
		return plugins.Capabilities{}
	}

	goClient := godaggerkit.NewDaggerAdapter(client)

	builder := &golang.GoBuilder{
		Client: goClient,
		Config: shipwright.BuildConfig{
			GoVersion:   cfg.GoVersion,
			JavaVersion: cfg.JavaVersion,
		},
	}

	testers := []shipwright.Tester{
		&golang.GoUnitTester{
			Client:    goClient,
			Config:    shipwright.TestConfig{Coverage: cfg.Coverage},
			GoVersion: cfg.GoVersion,
		},
		&golang.GoLinter{Client: goClient},
		&golang.GoVulnScanner{Client: goClient, GoVersion: cfg.GoVersion},
	}

	artifactConfig := shipwright.ArtifactConfig{
		Registry:     cfg.Registry,
		RegistryURL:  cfg.RegistryURL,
		RegistryUser: cfg.RegistryUser,
		ImageName:    cfg.ImageName,
		ImageTag:     cfg.ImageTag,
		BuildTag:     cfg.BuildTag,
		CommitSHA:    cfg.CommitSHA,
		BranchName:   cfg.BranchName,
		Version:      cfg.Version,
	}
	if cfg.RegistryPass != "" {
		artifactConfig.RegistryPass = client.SetSecret("registry-pass", cfg.RegistryPass)
	}
	if cfg.RegistryToken != "" {
		artifactConfig.RegistryToken = client.SetSecret("registry-token", cfg.RegistryToken)
	}
	if cfg.Token != "" {
		artifactConfig.Token = client.SetSecret("generic-token", cfg.Token)
	}

	artifactor := &golang.ContainerPublisher{
		Client: goClient,
		Config: artifactConfig,
	}

	return plugins.Capabilities{
		Builder:    builder,
		Testers:    testers,
		Artifactor: artifactor,
	}
}
