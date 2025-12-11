package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"log/slog"

	"dagger.io/dagger"

	"github.com/getsyntegrity/syntegrity-dagger/internal/app"
	"github.com/getsyntegrity/syntegrity-dagger/internal/config"
	"github.com/getsyntegrity/syntegrity-dagger/internal/executors"
	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
)

// CLI represents the command line interface for the application.
type CLI struct {
	app        *app.App
	yamlConfig *config.YAMLConfig
}

// NewCLI creates a new CLI instance.
func NewCLI() *CLI {
	return &CLI{}
}

// Run executes the CLI with the given arguments.
func (c *CLI) Run(args []string) error {
	ctx := context.Background()

	// Parse command line flags
	flags, err := c.parseFlags(args)
	if err != nil {
		return fmt.Errorf("failed to parse command line arguments: %w", err)
	}

	// Handle version flag
	if flags.version {
		c.showVersion()
		return nil
	}

	// Handle health check flag
	if flags.health {
		return c.runHealthChecks(ctx, flags)
	}

	// Load configuration
	cfg, err := config.NewConfigurationWrapper()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Load YAML configuration if specified
	if flags.configFile != "" {
		var yamlCfg *config.YAMLConfig
		yamlCfg, err = c.loadYAMLConfig(cfg, flags.configFile)
		if err != nil {
			return fmt.Errorf("failed to load YAML configuration: %w", err)
		}
		c.yamlConfig = yamlCfg
	}

	// Override configuration with CLI flags
	c.overrideConfig(cfg, flags)

	// Initialize the application
	err = app.Initialize(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}
	defer app.Reset()

	// Log successful initialization using the global logger
	slog.Info("Syntegrity Dagger initialized successfully",
		"pipeline", flags.pipelineName,
		"environment", flags.env,
		"verbose", flags.verbose)

	// Get the application instance
	c.app = app.NewApp(app.GetContainer())

	// Handle special commands
	if flags.listPipelines {
		return c.listAvailablePipelines(ctx)
	}

	if flags.listSteps {
		return c.listAvailableSteps(ctx, flags)
	}

	// Execute based on flags
	if flags.step != "" {
		return c.executeSingleStep(ctx, flags)
	}

	return c.executePipeline(ctx, flags)
}

// Flags represents the command line flags.
type Flags struct {
	pipelineName  string
	coverage      float64
	branch        string
	env           string
	executor      string
	gitAuth       string
	gitRef        string
	health        bool
	listPipelines bool
	listSteps     bool
	local         bool
	onlyBuild     bool
	onlyTest      bool
	skipPush      bool
	step          string
	verbose       bool
	version       bool
	configFile    string
}

// parseFlags parses command line arguments.
// Returns parsed flags and an error if flag parsing fails.
func (c *CLI) parseFlags(args []string) (*Flags, error) {
	flagSet := flag.NewFlagSet("syntegrity-dagger", flag.ContinueOnError)

	defaultGitAuth := "ssh"
	if os.Getenv("CI_JOB_TOKEN") != "" {
		defaultGitAuth = "https"
	}

	flags := &Flags{}

	flagSet.StringVar(&flags.pipelineName, "pipeline", "go-service", "Name of the pipeline to be executed")
	flagSet.Float64Var(&flags.coverage, "coverage", 90, "Minimum coverage percentage required (in: 90 for 90%)")
	flagSet.StringVar(&flags.branch, "branch", "develop", "Branch name")
	flagSet.StringVar(&flags.env, "env", "dev", "Environment: dev, staging, prod")
	flagSet.StringVar(&flags.executor, "executor", "", "Executor to use: native, docker (empty for auto-detection)")
	flagSet.BoolVar(&flags.skipPush, "skip-push", false, "Skip image push")
	flagSet.BoolVar(&flags.onlyBuild, "only-build", false, "Run build only")
	flagSet.BoolVar(&flags.onlyTest, "only-test", false, "Run only tests")
	flagSet.BoolVar(&flags.verbose, "verbose", false, "Verbose mode")
	flagSet.StringVar(&flags.gitRef, "git-ref", "main", "Branch name (default: main)")
	flagSet.StringVar(&flags.step, "step", "", "Individual pipeline step to execute")
	flagSet.StringVar(&flags.gitAuth, "git-auth", defaultGitAuth, "Git authentication method: ssh or https")
	flagSet.BoolVar(&flags.listSteps, "list-steps", false, "List available steps for a pipeline")
	flagSet.BoolVar(&flags.listPipelines, "list-pipelines", false, "List available pipelines")
	flagSet.StringVar(&flags.configFile, "config", ".syntegrity-dagger.yml", "Configuration file path")
	flagSet.BoolVar(&flags.version, "version", false, "Show version information")
	flagSet.BoolVar(&flags.local, "local", false, "Run pipeline locally without Docker")
	flagSet.BoolVar(&flags.health, "health", false, "Run health checks for Dagger, registry, and Git")

	if err := flagSet.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}
	return flags, nil
}

// overrideConfig overrides configuration with CLI flags.
func (c *CLI) overrideConfig(cfg interfaces.Configuration, flags *Flags) {
	// Override configuration values with CLI flags
	if flags.coverage != 90 {
		cfg.Set("pipeline.coverage", flags.coverage)
	}
	if flags.env != "dev" {
		cfg.Set("environment", flags.env)
	}
	if flags.skipPush {
		cfg.Set("pipeline.skip_push", true)
	}
	if flags.onlyBuild {
		cfg.Set("pipeline.only_build", true)
	}
	if flags.onlyTest {
		cfg.Set("pipeline.only_test", true)
	}
	if flags.verbose {
		cfg.Set("pipeline.verbose", true)
	}
	if flags.gitRef != "main" {
		cfg.Set("git.ref", flags.gitRef)
	}
	if flags.gitAuth != "ssh" {
		cfg.Set("git.protocol", flags.gitAuth)
	}
}

// executePipeline executes a complete pipeline.
func (c *CLI) executePipeline(ctx context.Context, flags *Flags) error {
	// Auto-detect local execution if not in CI and no executor specified
	shouldUseLocal := flags.local || (flags.executor == "" && !c.isCIEnvironment() && flags.executor == "")

	if shouldUseLocal {
		fmt.Printf("🏠 Running pipeline locally: %s (%s)\n", flags.pipelineName, flags.env)
		fmt.Printf("📊 Coverage threshold: %.1f%%\n", flags.coverage)
		fmt.Printf("🌿 Git ref: %s\n", flags.gitRef)
		fmt.Printf("⚡ Using native execution (no Docker required)\n")

		return c.executePipelineLocally(ctx, flags)
	}

	fmt.Printf("🚀 Running pipeline: %s (%s)\n", flags.pipelineName, flags.env)
	fmt.Printf("📊 Coverage threshold: %.1f%%\n", flags.coverage)
	fmt.Printf("🌿 Git ref: %s\n", flags.gitRef)

	// Use executor selector if executor is specified
	if flags.executor != "" {
		return c.executePipelineWithExecutor(ctx, flags)
	}

	return c.app.RunPipeline(ctx, flags.pipelineName)
}

// isCIEnvironment checks if we're running in a CI environment.
func (c *CLI) isCIEnvironment() bool {
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

// executePipelineWithExecutor executes pipeline using the specified executor.
func (c *CLI) executePipelineWithExecutor(ctx context.Context, flags *Flags) error {
	container := c.app.GetContainer()

	// Get executor selector
	selector, err := container.Get("executorSelector")
	if err != nil {
		return fmt.Errorf("failed to get executor selector: %w", err)
	}

	executorSelector := selector.(*executors.Selector)

	// Get Dagger client (may be nil for native execution)
	var daggerClient *dagger.Client
	client, err := container.Get("daggerClient")
	if err == nil && client != nil {
		daggerClient = client.(*dagger.Client)
	}

	// Get config
	cfg := container.GetConfiguration()
	pipelineConfig := app.ConvertConfigToPipelinesConfig(cfg)

	// Select executor
	executor, err := executorSelector.SelectExecutor(ctx, daggerClient, pipelineConfig, flags.executor)
	if err != nil {
		return fmt.Errorf("failed to select executor: %w", err)
	}

	fmt.Printf("🔧 Using executor: %s\n", executor.Name())

	// Get pipeline steps from configuration
	steps := []string{"setup", "build", "test"}
	if flags.onlyBuild {
		steps = []string{"setup", "build"}
	} else if flags.onlyTest {
		steps = []string{"setup", "test"}
	}

	// Execute steps using selected executor
	for _, step := range steps {
		fmt.Printf("▶️  Executing step: %s\n", step)
		if err := executor.ExecuteStep(ctx, step); err != nil {
			return fmt.Errorf("step %s failed: %w", step, err)
		}
		fmt.Printf("✅ Step completed: %s\n", step)
	}

	return nil
}

// executeSingleStep executes a single pipeline step.
func (c *CLI) executeSingleStep(ctx context.Context, flags *Flags) error {
	// Auto-detect local execution if not in CI and no executor specified
	shouldUseLocal := flags.local || (flags.executor == "" && !c.isCIEnvironment())

	if shouldUseLocal {
		fmt.Printf("🏠 Executing step locally '%s' in pipeline: %s\n", flags.step, flags.pipelineName)
		return c.executeStepLocally(ctx, flags)
	}

	// Use executor selector if executor is specified
	if flags.executor != "" {
		return c.executeStepWithExecutor(ctx, flags)
	}

	fmt.Printf("🎯 Executing step '%s' in pipeline: %s\n", flags.step, flags.pipelineName)

	return c.app.RunPipelineStep(ctx, flags.pipelineName, flags.step)
}

// executeStepWithExecutor executes a single step using the specified executor.
func (c *CLI) executeStepWithExecutor(ctx context.Context, flags *Flags) error {
	container := c.app.GetContainer()

	// Get executor selector
	selector, err := container.Get("executorSelector")
	if err != nil {
		return fmt.Errorf("failed to get executor selector: %w", err)
	}

	executorSelector := selector.(*executors.Selector)

	// Get Dagger client (may be nil for native execution)
	var daggerClient *dagger.Client
	client, err := container.Get("daggerClient")
	if err == nil && client != nil {
		daggerClient = client.(*dagger.Client)
	}

	// Get config
	cfg := container.GetConfiguration()
	pipelineConfig := app.ConvertConfigToPipelinesConfig(cfg)

	// Select executor
	executor, err := executorSelector.SelectExecutor(ctx, daggerClient, pipelineConfig, flags.executor)
	if err != nil {
		return fmt.Errorf("failed to select executor: %w", err)
	}

	fmt.Printf("🔧 Using executor: %s\n", executor.Name())
	fmt.Printf("▶️  Executing step: %s\n", flags.step)

	if err := executor.ExecuteStep(ctx, flags.step); err != nil {
		return fmt.Errorf("step %s failed: %w", flags.step, err)
	}

	fmt.Printf("✅ Step completed: %s\n", flags.step)
	return nil
}

// listAvailableSteps lists available steps for a pipeline.
func (c *CLI) listAvailableSteps(_ context.Context, flags *Flags) error {
	container := c.app.GetContainer()
	stepRegistry, err := container.Get("stepRegistry")
	if err != nil {
		return fmt.Errorf("failed to get step registry: %w", err)
	}

	registry := stepRegistry.(interfaces.StepRegistry)
	steps := registry.ListSteps()

	fmt.Printf("Available steps for pipeline '%s':\n", flags.pipelineName)

	for i, step := range steps {
		cfg, err := registry.GetStepConfig(step)
		if err != nil {
			fmt.Printf("  %d. %s (error getting config)\n", i+1, step)

			continue
		}

		fmt.Printf("  %d. %s - %s\n", i+1, step, cfg.Description)
		if cfg.Required {
			fmt.Printf("     (Required: Yes, Timeout: %v)\n", cfg.Timeout)
		} else {
			fmt.Printf("     (Required: No, Timeout: %v)\n", cfg.Timeout)
		}
	}

	return nil
}

// listAvailablePipelines lists available pipelines.
func (c *CLI) listAvailablePipelines(_ context.Context) error {
	container := c.app.GetContainer()
	registry, err := container.Get("pipelineRegistry")
	if err != nil {
		return fmt.Errorf("failed to get pipeline registry: %w", err)
	}

	pipelineRegistry := registry.(interfaces.PipelineRegistry)
	pipelines := pipelineRegistry.List()

	fmt.Println("Available pipelines:")
	for index, pipelineName := range pipelines {
		fmt.Printf("  %d. %s\n", index+1, pipelineName)
	}

	return nil
}

// loadYAMLConfig loads and applies YAML configuration
// Returns the parsed YAMLConfig for later use
func (c *CLI) loadYAMLConfig(cfg interfaces.Configuration, configFile string) (*config.YAMLConfig, error) {
	parser := config.NewYAMLParser()

	// Parse YAML file
	yamlConfig, err := parser.ParseFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML file: %w", err)
	}

	// Validate configuration
	if err := parser.ValidateConfig(yamlConfig); err != nil {
		return nil, fmt.Errorf("invalid YAML configuration: %w", err)
	}

	// Apply to main configuration
	if err := parser.ApplyToConfiguration(yamlConfig, cfg); err != nil {
		return nil, fmt.Errorf("failed to apply YAML configuration: %w", err)
	}

	fmt.Printf("📋 Loaded configuration from: %s\n", configFile)
	fmt.Printf("🎯 Pipeline: %s\n", yamlConfig.Pipeline.Name)
	fmt.Printf("📝 Steps: %v\n", yamlConfig.Pipeline.Steps)

	return yamlConfig, nil
}

// executePipelineLocally executes a pipeline locally without Docker
func (c *CLI) executePipelineLocally(ctx context.Context, flags *Flags) error {
	// Get logger and config from the app
	container := c.app.GetContainer()
	logger, err := container.GetLogger()
	if err != nil {
		return fmt.Errorf("failed to get logger: %w", err)
	}

	cfg := container.GetConfiguration()

	// Create local executor
	localExecutor := app.NewLocalExecutor(logger, cfg)

	// Get pipeline steps from configuration
	var steps []string

	// First, try to get steps from YAML config if available
	if c.yamlConfig != nil {
		parser := config.NewYAMLParser()
		steps = parser.GetSteps(c.yamlConfig)
		logger.Info("Using steps from YAML configuration", "steps", steps)
	}

	// If no steps from YAML, try to get from configuration
	if len(steps) == 0 {
		if stepsConfig := cfg.Get("pipeline.steps"); stepsConfig != nil {
			if stepsList, ok := stepsConfig.([]string); ok {
				steps = stepsList
				logger.Info("Using steps from configuration", "steps", steps)
			}
		}
	}

	// Handle CLI flags that override step selection
	if flags.onlyBuild {
		steps = []string{"setup", "build"}
		logger.Info("Using only-build steps", "steps", steps)
	} else if flags.onlyTest {
		steps = []string{"setup", "test"}
		logger.Info("Using only-test steps", "steps", steps)
	} else if len(steps) == 0 {
		// Default steps if nothing is configured
		steps = []string{"setup", "build", "test"}
		logger.Info("Using default steps", "steps", steps)
	}

	// Filter out steps that are not supported in local mode
	// Some steps like "push", "tag", "package" may not be applicable locally
	supportedLocalSteps := map[string]bool{
		"setup":    true,
		"build":    true,
		"test":     true,
		"lint":     true,
		"security": true,
	}

	filteredSteps := make([]string, 0, len(steps))
	for _, step := range steps {
		if supportedLocalSteps[step] {
			filteredSteps = append(filteredSteps, step)
		} else {
			logger.Warn("Skipping step not supported in local mode", "step", step)
		}
	}

	if len(filteredSteps) == 0 {
		return fmt.Errorf("no valid steps to execute in local mode")
	}

	// Execute steps in order
	for _, step := range filteredSteps {
		logger.Info("Running pipeline step", "pipeline", flags.pipelineName, "step", step)

		if err := localExecutor.ExecuteStep(ctx, step); err != nil {
			return fmt.Errorf("pipeline step %s failed: %w", step, err)
		}

		logger.Info("Pipeline step completed", "step", step)
	}

	logger.Info("Pipeline completed successfully", "name", flags.pipelineName)
	return nil
}

// executeStepLocally executes a single step locally without Docker
func (c *CLI) executeStepLocally(ctx context.Context, flags *Flags) error {
	// Get logger and config from the app
	container := c.app.GetContainer()
	logger, err := container.GetLogger()
	if err != nil {
		return fmt.Errorf("failed to get logger: %w", err)
	}

	// Create local executor
	localExecutor := app.NewLocalExecutor(logger, container.GetConfiguration())

	// Execute the step
	logger.Info("Running pipeline step", "pipeline", flags.pipelineName, "step", flags.step)

	if err = localExecutor.ExecuteStep(ctx, flags.step); err != nil {
		return fmt.Errorf("pipeline step %s failed: %w", flags.step, err)
	}

	logger.Info("Pipeline step completed", "step", flags.step)
	return nil
}

// Version and build information - set at build time
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// showVersion displays version information
func (c *CLI) showVersion() {
	fmt.Printf("Syntegrity Dagger %s\n", Version)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Build time: %s\n", BuildTime)
	fmt.Printf("Git commit: %s\n", GitCommit)
}

// runHealthChecks executes health checks for all configured services
func (c *CLI) runHealthChecks(ctx context.Context, flags *Flags) error {
	// Load configuration
	cfg, err := config.NewConfigurationWrapper()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Load YAML configuration if specified
	if flags.configFile != "" {
		var yamlCfg *config.YAMLConfig
		yamlCfg, err = c.loadYAMLConfig(cfg, flags.configFile)
		if err != nil {
			return fmt.Errorf("failed to load YAML configuration: %w", err)
		}
		c.yamlConfig = yamlCfg
	}

	fmt.Println("🏥 Running health checks...")
	fmt.Println()

	return app.RunHealthChecks(ctx, cfg)
}

func main() {
	cli := NewCLI()

	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}
