package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"dagger.io/dagger"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
	docker_go "github.com/getsyntegrity/syntegrity-dagger/internal/pipelines/docker-go"
	gokit "github.com/getsyntegrity/syntegrity-dagger/internal/pipelines/go-kit"
	infra "github.com/getsyntegrity/syntegrity-dagger/internal/pipelines/infra"
)

// Static errors for err113 compliance.
var (
	ErrComponentNotFound          = errors.New("component not found")
	ErrFailedToCreateDaggerClient = errors.New("failed to create Dagger client")
	ErrPipelineNotFound           = errors.New("pipeline not found")
	ErrInvalidConfiguration       = errors.New("invalid configuration")
)

// LoggerAdapter adapts slog to interfaces.Logger
type LoggerAdapter struct {
	logger *slog.Logger
}

// Debug logs a debug message
func (l *LoggerAdapter) Debug(msg string, fields ...any) {
	l.logger.Debug(msg, fields...)
}

// Info logs an info message
func (l *LoggerAdapter) Info(msg string, fields ...any) {
	l.logger.Info(msg, fields...)
}

// Warn logs a warning message
func (l *LoggerAdapter) Warn(msg string, fields ...any) {
	l.logger.Warn(msg, fields...)
}

// Error logs an error message
func (l *LoggerAdapter) Error(msg string, fields ...any) {
	l.logger.Error(msg, fields...)
}

// Fatal logs a fatal message
func (l *LoggerAdapter) Fatal(msg string, fields ...any) {
	l.logger.Error(msg, fields...)
	// In a real implementation, this would call os.Exit(1)
}

// WithField adds a single field to the logger context
func (l *LoggerAdapter) WithField(_ string, _ any) interfaces.Logger {
	// For now, return the same logger since the API might be different
	// This would need to be implemented based on the actual API
	return l
}

// WithFields adds multiple fields to the logger context
func (l *LoggerAdapter) WithFields(_ map[string]any) interfaces.Logger {
	// For now, return the same logger since the API might be different
	// This would need to be implemented based on the actual API
	return l
}

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
				fmt.Printf("Failed to close component %s: %v\n", name, err)
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
	c.registerPipelineComponents()
	c.registerSecurityComponents()
	c.registerLoggingComponents()
	c.registerStepComponents()
	c.registerHookComponents()
}

// registerDaggerComponents registers Dagger-related components.
func (c *Container) registerDaggerComponents() {
	// Dagger Client
	c.Register("daggerClient", func() (any, error) {
		timeout := c.config.GetDuration("dagger.timeout")
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		ctx, cancel := context.WithTimeout(c.ctx, timeout)
		defer cancel()

		client, err := dagger.Connect(ctx, dagger.WithLogOutput(nil))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrFailedToCreateDaggerClient, err)
		}

		return client, nil
	})
}

// registerPipelineComponents registers pipeline-related components.
func (c *Container) registerPipelineComponents() {
	// Pipeline Registry
	c.Register("pipelineRegistry", func() (any, error) {
		registry := NewPipelineRegistry()

		// Register default pipelines
		registry.Register("go-kit", NewGoKitPipeline)
		registry.Register("docker-go", NewDockerGoPipeline)
		registry.Register("infra", NewInfraPipeline)

		return registry, nil
	})
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

// CreateLogger creates a new logger instance using the configuration.
func (c *Container) CreateLogger() *slog.Logger {
	loggingConfig := c.config.Logging()

	// Create a simple slog logger
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if loggingConfig.Level == "debug" {
		opts.Level = slog.LevelDebug
	}

	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

// registerLoggingComponents registers logging-related components.
func (c *Container) registerLoggingComponents() {
	// Logger using Syntegrity go-kit-logger directly
	c.Register("logger", func() (any, error) {
		// Create logger using the CreateLogger method
		logger := c.CreateLogger()

		// Return logger adapter that implements interfaces.Logger
		return &LoggerAdapter{logger: logger}, nil
	})
}

// GetPipelineRegistry implements PipelineProvider interface.
func (c *Container) GetPipelineRegistry() (interfaces.PipelineRegistry, error) {
	registry, err := c.Get("pipelineRegistry")
	if err != nil {
		return nil, err
	}
	return registry.(interfaces.PipelineRegistry), nil
}

// GetPipeline implements PipelineProvider interface.
func (c *Container) GetPipeline(name string) (interfaces.Pipeline, error) {
	registry, err := c.GetPipelineRegistry()
	if err != nil {
		return nil, err
	}

	client, err := c.GetDaggerClient()
	if err != nil {
		return nil, err
	}

	return registry.Get(name, client, c.config)
}

// GetDaggerClient implements PipelineProvider interface.
func (c *Container) GetDaggerClient() (*dagger.Client, error) {
	client, err := c.Get("daggerClient")
	if err != nil {
		return nil, err
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
func (c *Container) GetLogger() (interfaces.Logger, error) {
	logger, err := c.Get("logger")
	if err != nil {
		return nil, err
	}
	return logger.(interfaces.Logger), nil
}

// GetConfiguration returns the configuration instance.
func (c *Container) GetConfiguration() interfaces.Configuration {
	return c.config
}

// PipelineRegistry implements the pipeline registry.
type PipelineRegistry struct {
	pipelines map[string]func(*dagger.Client, interfaces.Configuration) interfaces.Pipeline
}

// NewPipelineRegistry creates a new pipeline registry.
func NewPipelineRegistry() *PipelineRegistry {
	return &PipelineRegistry{
		pipelines: make(map[string]func(*dagger.Client, interfaces.Configuration) interfaces.Pipeline),
	}
}

// Register adds a new pipeline to the registry.
func (r *PipelineRegistry) Register(name string, factory func(*dagger.Client, interfaces.Configuration) interfaces.Pipeline) {
	r.pipelines[name] = factory
}

// Get retrieves a pipeline by its name.
func (r *PipelineRegistry) Get(name string, client *dagger.Client, cfg interfaces.Configuration) (interfaces.Pipeline, error) {
	factory, ok := r.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrPipelineNotFound, name)
	}
	return factory(client, cfg), nil
}

// List returns the names of all registered pipelines.
func (r *PipelineRegistry) List() []string {
	names := make([]string, 0, len(r.pipelines))
	for name := range r.pipelines {
		names = append(names, name)
	}
	return names
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
	fmt.Printf("[DEBUG] %s %v\n", msg, fields)
}

// Info logs an info message.
func (l *Logger) Info(msg string, fields ...any) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, fields ...any) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}

// Error logs an error message.
func (l *Logger) Error(msg string, fields ...any) {
	fmt.Printf("[ERROR] %s %v\n", msg, fields)
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(msg string, fields ...any) {
	fmt.Printf("[FATAL] %s %v\n", msg, fields)
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
		logger, err := c.Get("logger")
		if err != nil {
			return nil, fmt.Errorf("failed to get logger: %w", err)
		}
		log := logger.(*LoggerAdapter)

		client, err := c.Get("daggerClient")
		if err != nil {
			// Dagger client might not be available, pass nil
			client = nil
		}
		var daggerClient *dagger.Client
		if client != nil {
			daggerClient = client.(*dagger.Client)
		}

		// Register default steps with logger and client
		_ = registry.RegisterStep("setup", NewSetupStepHandler(c.config, daggerClient, log))
		_ = registry.RegisterStep("build", NewBuildStepHandler(c.config, daggerClient, log))
		_ = registry.RegisterStep("test", NewTestStepHandler(c.config, daggerClient, log))
		_ = registry.RegisterStep("lint", NewLintStepHandler(c.config, daggerClient, log))
		_ = registry.RegisterStep("security", NewSecurityStepHandler(c.config, daggerClient, log))
		_ = registry.RegisterStep("tag", NewTagStepHandler(c.config, daggerClient, log))
		_ = registry.RegisterStep("package", NewPackageStepHandler(c.config, daggerClient, log))
		_ = registry.RegisterStep("push", NewPushStepHandler(c.config, daggerClient, log))
		_ = registry.RegisterStep("release", NewReleaseStepHandler(c.config, daggerClient, log))

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

// registerHookComponents registers hook-related components.
func (c *Container) registerHookComponents() {
	// Hook Manager
	c.Register("hookManager", func() (any, error) {
		return NewHookManager(), nil
	})
}

// PipelineAdapter adapts pipelines.Pipeline to interfaces.Pipeline
type PipelineAdapter struct {
	pipeline pipelines.Pipeline
}

// NewPipelineAdapter creates a new pipeline adapter
func NewPipelineAdapter(pipeline pipelines.Pipeline) *PipelineAdapter {
	return &PipelineAdapter{pipeline: pipeline}
}

// Name returns the name of the pipeline
func (p *PipelineAdapter) Name() string {
	return p.pipeline.Name()
}

// GetAvailableSteps returns the available steps for the pipeline
func (p *PipelineAdapter) GetAvailableSteps() []string {
	return []string{"setup", "build", "test", "package", "tag", "push"}
}

// ExecuteStep executes a specific step
func (p *PipelineAdapter) ExecuteStep(ctx context.Context, stepName string) error {
	switch stepName {
	case "setup":
		return p.pipeline.Setup(ctx)
	case "build":
		return p.pipeline.Build(ctx)
	case "test":
		return p.pipeline.Test(ctx)
	case "package":
		return p.pipeline.Package(ctx)
	case "tag":
		return p.pipeline.Tag(ctx)
	case "push":
		return p.pipeline.Push(ctx)
	default:
		return fmt.Errorf("unknown step: %s", stepName)
	}
}

// BeforeStep returns a hook function to execute before a step
func (p *PipelineAdapter) BeforeStep(ctx context.Context, stepName string) interfaces.HookFunc {
	hook := p.pipeline.BeforeStep(ctx, stepName)
	if hook == nil {
		return func(_ context.Context) error { return nil }
	}
	// Convert pipelines.HookFunc to interfaces.HookFunc
	return interfaces.HookFunc(hook)
}

// AfterStep returns a hook function to execute after a step
func (p *PipelineAdapter) AfterStep(ctx context.Context, stepName string) interfaces.HookFunc {
	hook := p.pipeline.AfterStep(ctx, stepName)
	if hook == nil {
		return func(_ context.Context) error { return nil }
	}
	// Convert pipelines.HookFunc to interfaces.HookFunc
	return interfaces.HookFunc(hook)
}

// GetStepConfig returns configuration for a step
func (p *PipelineAdapter) GetStepConfig(stepName string) interfaces.StepConfig {
	return interfaces.StepConfig{
		Name:        stepName,
		Description: fmt.Sprintf("Execute %s step", stepName),
		Required:    true,
		Timeout:     5 * time.Minute,
		Retries:     0,
	}
}

// ValidateStep validates a step name
func (p *PipelineAdapter) ValidateStep(stepName string) error {
	validSteps := p.GetAvailableSteps()
	for _, step := range validSteps {
		if step == stepName {
			return nil
		}
	}
	return fmt.Errorf("invalid step: %s", stepName)
}

// convertConfig converts interfaces.Configuration to pipelines.Config
func convertConfig(cfg interfaces.Configuration) pipelines.Config {
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
		Version:       cfg.GetString("service.version"),
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

// Pipeline factory functions
func NewGoKitPipeline(client *dagger.Client, cfg interfaces.Configuration) interfaces.Pipeline {
	pipelineConfig := convertConfig(cfg)
	pipeline := gokit.New(client, pipelineConfig)
	return NewPipelineAdapter(pipeline)
}

func NewDockerGoPipeline(client *dagger.Client, cfg interfaces.Configuration) interfaces.Pipeline {
	pipelineConfig := convertConfig(cfg)
	pipeline := docker_go.New(client, pipelineConfig)
	return NewPipelineAdapter(pipeline)
}

func NewInfraPipeline(client *dagger.Client, cfg interfaces.Configuration) interfaces.Pipeline {
	pipelineConfig := convertConfig(cfg)
	pipeline := infra.New(client, pipelineConfig)
	return NewPipelineAdapter(pipeline)
}
