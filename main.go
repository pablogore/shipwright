package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/getsyntegrity/syntegrity-dagger/internal/app"
	"github.com/getsyntegrity/syntegrity-dagger/internal/config"
	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
)

// CLI represents the command line interface for the application.
type CLI struct {
	app *app.App
}

// NewCLI creates a new CLI instance.
func NewCLI() *CLI {
	return &CLI{}
}

// Run executes the CLI with the given arguments.
func (c *CLI) Run(args []string) error {
	ctx := context.Background()

	// Parse command line flags
	flags := c.parseFlags(args)

	// Handle version flag
	if flags.version {
		c.showVersion()
		return nil
	}

	// Load configuration
	cfg, err := config.NewConfigurationWrapper()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Load YAML configuration if specified
	if flags.configFile != "" {
		err = c.loadYAMLConfig(cfg, flags.configFile)
		if err != nil {
			return fmt.Errorf("failed to load YAML configuration: %w", err)
		}
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
	log.Println("Syntegrity Dagger initialized successfully",
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
	skipPush      bool
	onlyBuild     bool
	onlyTest      bool
	verbose       bool
	gitRef        string
	step          string
	gitAuth       string
	listSteps     bool
	listPipelines bool
	configFile    string
	version       bool
	local         bool
}

// parseFlags parses command line arguments.
func (c *CLI) parseFlags(args []string) *Flags {
	fs := flag.NewFlagSet("syntegrity-dagger", flag.ExitOnError)

	defaultGitAuth := "ssh"
	if os.Getenv("CI_JOB_TOKEN") != "" {
		defaultGitAuth = "https"
	}

	flags := &Flags{}

	fs.StringVar(&flags.pipelineName, "pipeline", "go-kit", "Name of the pipeline to be executed")
	fs.Float64Var(&flags.coverage, "coverage", 90, "Minimum coverage percentage required (in: 90 for 90%)")
	fs.StringVar(&flags.branch, "branch", "develop", "Branch name")
	fs.StringVar(&flags.env, "env", "dev", "Environment: dev, staging, prod")
	fs.BoolVar(&flags.skipPush, "skip-push", false, "Skip image push")
	fs.BoolVar(&flags.onlyBuild, "only-build", false, "Run build only")
	fs.BoolVar(&flags.onlyTest, "only-test", false, "Run only tests")
	fs.BoolVar(&flags.verbose, "verbose", false, "Verbose mode")
	fs.StringVar(&flags.gitRef, "git-ref", "main", "Branch name (default: main)")
	fs.StringVar(&flags.step, "step", "", "Individual pipeline step to execute")
	fs.StringVar(&flags.gitAuth, "git-auth", defaultGitAuth, "Git authentication method: ssh or https")
	fs.BoolVar(&flags.listSteps, "list-steps", false, "List available steps for a pipeline")
	fs.BoolVar(&flags.listPipelines, "list-pipelines", false, "List available pipelines")
	fs.StringVar(&flags.configFile, "config", ".syntegrity-dagger.yml", "Configuration file path")
	fs.BoolVar(&flags.version, "version", false, "Show version information")
	fs.BoolVar(&flags.local, "local", false, "Run pipeline locally without Docker")

	fs.Parse(args)
	return flags
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
	if flags.local {
		fmt.Printf("🏠 Running pipeline locally: %s (%s)\n", flags.pipelineName, flags.env)
		fmt.Printf("📊 Coverage threshold: %.1f%%\n", flags.coverage)
		fmt.Printf("🌿 Git ref: %s\n", flags.gitRef)
		return c.executePipelineLocally(ctx, flags)
	}

	fmt.Printf("🚀 Running pipeline: %s (%s)\n", flags.pipelineName, flags.env)
	fmt.Printf("📊 Coverage threshold: %.1f%%\n", flags.coverage)
	fmt.Printf("🌿 Git ref: %s\n", flags.gitRef)

	return c.app.RunPipeline(ctx, flags.pipelineName)
}

// executeSingleStep executes a single pipeline step.
func (c *CLI) executeSingleStep(ctx context.Context, flags *Flags) error {
	if flags.local {
		fmt.Printf("🏠 Executing step locally '%s' in pipeline: %s\n", flags.step, flags.pipelineName)
		return c.executeStepLocally(ctx, flags)
	}

	fmt.Printf("🎯 Executing step '%s' in pipeline: %s\n", flags.step, flags.pipelineName)

	return c.app.RunPipelineStep(ctx, flags.pipelineName, flags.step)
}

// listAvailableSteps lists available steps for a pipeline.
func (c *CLI) listAvailableSteps(ctx context.Context, flags *Flags) error {
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
func (c *CLI) listAvailablePipelines(ctx context.Context) error {
	container := c.app.GetContainer()
	registry, err := container.Get("pipelineRegistry")
	if err != nil {
		return fmt.Errorf("failed to get pipeline registry: %w", err)
	}

	pipelineRegistry := registry.(interfaces.PipelineRegistry)
	pipelines := pipelineRegistry.List()

	fmt.Println("Available pipelines:")
	for i, pipelineName := range pipelines {
		fmt.Printf("  %d. %s\n", i+1, pipelineName)
	}

	return nil
}

// loadYAMLConfig loads and applies YAML configuration
func (c *CLI) loadYAMLConfig(cfg interfaces.Configuration, configFile string) error {
	parser := config.NewYAMLParser()

	// Parse YAML file
	yamlConfig, err := parser.ParseFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse YAML file: %w", err)
	}

	// Validate configuration
	if err := parser.ValidateConfig(yamlConfig); err != nil {
		return fmt.Errorf("invalid YAML configuration: %w", err)
	}

	// Apply to main configuration
	if err := parser.ApplyToConfiguration(yamlConfig, cfg); err != nil {
		return fmt.Errorf("failed to apply YAML configuration: %w", err)
	}

	fmt.Printf("📋 Loaded configuration from: %s\n", configFile)
	fmt.Printf("🎯 Pipeline: %s\n", yamlConfig.Pipeline.Name)
	fmt.Printf("📝 Steps: %v\n", yamlConfig.Pipeline.Steps)

	return nil
}

// executePipelineLocally executes a pipeline locally without Docker
func (c *CLI) executePipelineLocally(ctx context.Context, flags *Flags) error {
	// Get logger and config from the app
	container := c.app.GetContainer()
	logger, err := container.GetLogger()
	if err != nil {
		return fmt.Errorf("failed to get logger: %w", err)
	}

	config := container.GetConfiguration()

	// Create local executor
	localExecutor := app.NewLocalExecutor(logger, config)

	// Get pipeline steps from configuration
	steps := []string{"setup", "build", "test"}
	if flags.onlyBuild {
		steps = []string{"setup", "build"}
	} else if flags.onlyTest {
		steps = []string{"setup", "test"}
	}

	// Execute steps in order
	for _, step := range steps {
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
	localExecutor := app.NewLocalExecutor(logger,  container.GetConfiguration())

	// Execute the step
	logger.Info("Running pipeline step", "pipeline", flags.pipelineName, "step", flags.step)

	if err = localExecutor.ExecuteStep(ctx, flags.step); err != nil {
		return fmt.Errorf("pipeline step %s failed: %w", flags.step, err)
	}

	logger.Info("Pipeline step completed", "step", flags.step)
	return nil
}

// showVersion displays version information
func (c *CLI) showVersion() {
	fmt.Printf("Syntegrity Dagger v1.0.0\n")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Build time: %s\n", "2024-01-01T00:00:00Z") // This would be set at build time
}

func main() {
	cli := NewCLI()

	if err := cli.Run(os.Args[1:]); err != nil {
		log.Fatalf("❌ %v", err)
	}
}
