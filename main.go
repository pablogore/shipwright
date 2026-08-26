package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"dagger.io/dagger"
	"github.com/pablogore/kit-logger/pkg/logger"

	"github.com/pablogore/shipwright/internal/app"
	"github.com/pablogore/shipwright/internal/config"
	"github.com/pablogore/shipwright/internal/executors"
	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/workflow/engine"
	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
	"github.com/pablogore/shipwright/internal/workflow/providers"
)

// defaultWorkflowManifestPath is --workflow's registered default (design.md
// D-N: "Default .shipwright/workflow.yaml"). Registering it as the flag's
// literal default, rather than "", keeps the flag's own --help text
// accurate to the design; MODE SELECTION never reads this default value
// directly — see Flags.workflowSet's doc comment for why.
const defaultWorkflowManifestPath = ".shipwright/workflow.yaml"

// CLI identity constants — presented in the --help usage line, --version
// output, and structured startup log messages.
const (
	cliName           = "shipwright"
	versionLogMessage = "Shipwright version"
	initLogMessage    = "Shipwright initialized successfully"
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

	// Load YAML configuration if specified and file exists
	// The config file is optional - if it doesn't exist, we use defaults
	if flags.configFile != "" {
		// Check if file exists before trying to load it
		if _, err := os.Stat(flags.configFile); err == nil {
			var yamlCfg *config.YAMLConfig
			yamlCfg, err = c.loadYAMLConfig(ctx, cfg, flags.configFile)
			if err != nil {
				return fmt.Errorf("failed to load YAML configuration: %w", err)
			}
			c.yamlConfig = yamlCfg
		} else {
			// File doesn't exist - this is OK, we'll use defaults
			// Only warn if the user explicitly specified --config
			if flags.configFile != ".shipwright.yml" {
				logger.L().WarnContext(ctx, "Configuration file not found, using defaults",
					"config_file", flags.configFile)
			}
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

	// Log successful initialization using go-kit-logger
	logger.L().InfoContext(ctx, initLogMessage,
		"pipeline", flags.pipelineName,
		"environment", flags.env,
		"verbose", flags.verbose)

	// Get the application instance
	c.app = app.NewApp(app.GetContainer())

	// The manifest-driven entrypoint (design.md D-N) takes over dispatch
	// entirely once the user explicitly requests it via --workflow — it
	// never falls through to the legacy --pipeline dispatch below, and a
	// missing/unreadable manifest fails closed with no implicit legacy
	// fallback (tasks.md 9.1). Everything from here down to the end of
	// Run is the pre-existing legacy --pipeline path, untouched (tasks.md
	// 9.5): both paths coexist so no merged state of this change ever
	// leaves the CLI able to run nothing (design.md D-N sequencing note).
	if flags.workflowSet {
		return c.runWorkflowCLI(ctx, flags)
	}

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

	// workflow is --workflow's path value (always populated — either the
	// user's explicit value or defaultWorkflowManifestPath). workflowSet
	// is true ONLY when the user explicitly passed --workflow on the
	// command line (detected via flag.FlagSet.Visit in parseFlags, never
	// from workflow's value alone). This is the mode switch between the
	// legacy --pipeline path and the new manifest-driven path: a bare
	// invocation (no --workflow) MUST behave EXACTLY as it did before this
	// flag existed (design.md D-N, tasks.md 9.5 — --pipeline and every
	// other legacy flag stay fully functional and untouched in this work
	// unit), which a value-based check ("workflow != \"\"") could not
	// guarantee once workflow has a non-empty registered default.
	workflow    string
	workflowSet bool
}

// parseFlags parses command line arguments.
// Returns parsed flags and an error if flag parsing fails.
func (c *CLI) parseFlags(args []string) (*Flags, error) {
	flagSet := flag.NewFlagSet(cliName, flag.ContinueOnError)

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
	flagSet.StringVar(&flags.configFile, "config", ".shipwright.yml", "Configuration file path")
	flagSet.BoolVar(&flags.version, "version", false, "Show version information")
	flagSet.BoolVar(&flags.local, "local", false, "Run pipeline locally without Docker")
	flagSet.BoolVar(&flags.health, "health", false, "Run health checks for Dagger, registry, and Git")
	flagSet.StringVar(&flags.workflow, "workflow", defaultWorkflowManifestPath,
		"Path to a declarative workflow manifest — the new manifest-driven entrypoint (design.md D-N). "+
			"Providing this flag switches the CLI to workflow mode; --step/--list-steps then target the "+
			"manifest instead of the legacy --pipeline preset")

	if err := flagSet.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	// flagSet.Visit iterates ONLY the flags actually set on the command
	// line — this is the one reliable way to distinguish "the user typed
	// --workflow" from "--workflow's registered default happens to be
	// non-empty" (see Flags.workflowSet's doc comment).
	flagSet.Visit(func(f *flag.Flag) {
		if f.Name == "workflow" {
			flags.workflowSet = true
		}
	})

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

	// Warn in CI if executing full pipeline (better to use individual steps)
	if c.isCIEnvironment() && flags.step == "" {
		logger.L().WarnContext(ctx, "Executing full pipeline in CI environment",
			"recommendation", "For better visibility in CI/CD UI, execute steps individually using --step flag",
			"example", cliName+" --pipeline go-service --step build")
	}

	if shouldUseLocal {
		logger.L().InfoContext(ctx, "Running pipeline locally",
			"pipeline", flags.pipelineName,
			"environment", flags.env,
			"coverage", flags.coverage,
			"git_ref", flags.gitRef)
		logger.L().InfoContext(ctx, "Using native execution (no Docker required)")

		return c.executePipelineLocally(ctx, flags)
	}

	logger.L().InfoContext(ctx, "Running pipeline",
		"pipeline", flags.pipelineName,
		"environment", flags.env,
		"coverage", flags.coverage,
		"git_ref", flags.gitRef)

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

	logger.L().InfoContext(ctx, "Using executor", "executor", executor.Name())

	// Get pipeline steps from configuration
	steps := []string{"setup", "build", "test"}
	if flags.onlyBuild {
		steps = []string{"setup", "build"}
	} else if flags.onlyTest {
		steps = []string{"setup", "test"}
	}

	// Execute steps using selected executor
	for _, step := range steps {
		logger.L().InfoContext(ctx, "Executing step", "step", step)
		if err := executor.ExecuteStep(ctx, step); err != nil {
			return fmt.Errorf("step %s failed: %w", step, err)
		}
		logger.L().InfoContext(ctx, "Step completed", "step", step)
	}

	return nil
}

// executeSingleStep executes a single pipeline step.
func (c *CLI) executeSingleStep(ctx context.Context, flags *Flags) error {
	// Auto-detect local execution if not in CI and no executor specified
	shouldUseLocal := flags.local || (flags.executor == "" && !c.isCIEnvironment())

	if shouldUseLocal {
		logger.L().InfoContext(ctx, "Executing step locally",
			"step", flags.step,
			"pipeline", flags.pipelineName)
		return c.executeStepLocally(ctx, flags)
	}

	// Use executor selector if executor is specified
	if flags.executor != "" {
		return c.executeStepWithExecutor(ctx, flags)
	}

	// Try to use executor selector automatically (native first, then docker)
	// This allows execution without Docker when Go is available
	container := c.app.GetContainer()
	selector, err := container.Get("executorSelector")
	if err == nil {
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

		// Try to select executor automatically (native first, then docker)
		executor, err := executorSelector.SelectExecutor(ctx, daggerClient, pipelineConfig, "")
		if err == nil {
			logger.L().InfoContext(ctx, "Auto-selected executor", "executor", executor.Name())
			logger.L().InfoContext(ctx, "Executing step", "step", flags.step)

			if err := executor.ExecuteStep(ctx, flags.step); err != nil {
				return fmt.Errorf("step %s failed: %w", flags.step, err)
			}

			logger.L().InfoContext(ctx, "Step completed", "step", flags.step)
			return nil
		}
		// If executor selection fails, fall through to RunPipelineStep
	}

	// Fallback to RunPipelineStep (requires Dagger)
	logger.L().InfoContext(ctx, "Executing step in pipeline",
		"step", flags.step,
		"pipeline", flags.pipelineName)

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

	logger.L().InfoContext(ctx, "Using executor", "executor", executor.Name())
	logger.L().InfoContext(ctx, "Executing step", "step", flags.step)

	if err := executor.ExecuteStep(ctx, flags.step); err != nil {
		return fmt.Errorf("step %s failed: %w", flags.step, err)
	}

	logger.L().InfoContext(ctx, "Step completed", "step", flags.step)
	return nil
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

	logger.L().InfoContext(ctx, "Available steps for pipeline",
		"pipeline", flags.pipelineName,
		"steps_count", len(steps))

	for i, step := range steps {
		cfg, err := registry.GetStepConfig(step)
		if err != nil {
			logger.L().WarnContext(ctx, "Error getting step config",
				"step", step,
				"error", err)
			continue
		}

		logger.L().InfoContext(ctx, "Step information",
			"index", i+1,
			"step", step,
			"description", cfg.Description,
			"required", cfg.Required,
			"timeout", cfg.Timeout)
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

	logger.L().InfoContext(ctx, "Available pipelines",
		"count", len(pipelines))
	for index, pipelineName := range pipelines {
		logger.L().InfoContext(ctx, "Pipeline",
			"index", index+1,
			"name", pipelineName)
	}

	return nil
}

// loadYAMLConfig loads and applies YAML configuration
// Returns the parsed YAMLConfig for later use
func (c *CLI) loadYAMLConfig(ctx context.Context, cfg interfaces.Configuration, configFile string) (*config.YAMLConfig, error) {
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

	logger.L().InfoContext(ctx, "Loaded configuration from file",
		"config_file", configFile,
		"pipeline", yamlConfig.Pipeline.Name,
		"steps", yamlConfig.Pipeline.Steps)

	return yamlConfig, nil
}

// executePipelineLocally executes a pipeline locally without Docker
func (c *CLI) executePipelineLocally(ctx context.Context, flags *Flags) error {
	// Get config from the app
	container := c.app.GetContainer()
	cfg := container.GetConfiguration()

	// Create local executor
	localExecutor := app.NewLocalExecutor(cfg)

	// Get pipeline steps from configuration
	var steps []string

	// First, try to get steps from YAML config if available
	if c.yamlConfig != nil {
		parser := config.NewYAMLParser()
		steps = parser.GetSteps(c.yamlConfig)
		logger.L().InfoContext(ctx, "Using steps from YAML configuration", "steps", steps)
	}

	// If no steps from YAML, try to get from configuration
	if len(steps) == 0 {
		if stepsConfig := cfg.Get("pipeline.steps"); stepsConfig != nil {
			if stepsList, ok := stepsConfig.([]string); ok {
				steps = stepsList
				logger.L().InfoContext(ctx, "Using steps from configuration", "steps", steps)
			}
		}
	}

	// Handle CLI flags that override step selection
	if flags.onlyBuild {
		steps = []string{"setup", "build"}
		logger.L().InfoContext(ctx, "Using only-build steps", "steps", steps)
	} else if flags.onlyTest {
		steps = []string{"setup", "test"}
		logger.L().InfoContext(ctx, "Using only-test steps", "steps", steps)
	} else if len(steps) == 0 {
		// Default steps if nothing is configured
		steps = []string{"setup", "build", "test"}
		logger.L().InfoContext(ctx, "Using default steps", "steps", steps)
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
			logger.L().WarnContext(ctx, "Skipping step not supported in local mode", "step", step)
		}
	}

	if len(filteredSteps) == 0 {
		return fmt.Errorf("no valid steps to execute in local mode")
	}

	// Execute steps in order
	for _, step := range filteredSteps {
		logger.L().InfoContext(ctx, "Running pipeline step", "pipeline", flags.pipelineName, "step", step)

		if err := localExecutor.ExecuteStep(ctx, step); err != nil {
			return fmt.Errorf("pipeline step %s failed: %w", step, err)
		}

		logger.L().InfoContext(ctx, "Pipeline step completed", "step", step)
	}

	logger.L().InfoContext(ctx, "Pipeline completed successfully", "name", flags.pipelineName)
	return nil
}

// executeStepLocally executes a single step locally without Docker
func (c *CLI) executeStepLocally(ctx context.Context, flags *Flags) error {
	// Get config from the app
	container := c.app.GetContainer()

	// Create local executor
	localExecutor := app.NewLocalExecutor(container.GetConfiguration())

	// Execute the step
	logger.L().InfoContext(ctx, "Running pipeline step", "pipeline", flags.pipelineName, "step", flags.step)

	if err := localExecutor.ExecuteStep(ctx, flags.step); err != nil {
		return fmt.Errorf("pipeline step %s failed: %w", flags.step, err)
	}

	logger.L().InfoContext(ctx, "Pipeline step completed", "step", flags.step)
	return nil
}

// workflowStepInfo is one manifest step's --list-steps report line
// (design.md D-N: "--list-steps retargeted to list manifest step ids with
// capability and resolved provider").
type workflowStepInfo struct {
	StepID          string
	Capability      string
	ProviderName    string
	ProviderModule  string
	ProviderVersion string
}

// runWorkflowCLI is the manifest-driven entrypoint's dispatcher (design.md
// D-N). It parses+builds the manifest's graph first, unconditionally — a
// missing/invalid manifest fails closed here, before --list-steps or
// --step are even inspected (tasks.md 9.1), and this function never calls
// into any legacy --pipeline code path (tasks.md 9.5).
func (c *CLI) runWorkflowCLI(ctx context.Context, flags *Flags) error {
	m, g, err := loadWorkflowManifest(flags.workflow)
	if err != nil {
		return err
	}

	if flags.listSteps {
		return c.listWorkflowSteps(ctx, m)
	}

	runGraph, err := selectWorkflowGraph(g, flags.step)
	if err != nil {
		return err
	}

	return c.executeWorkflow(ctx, m, runGraph, flags)
}

// loadWorkflowManifest runs stages 1-3 (manifest.ParseFile) and stage 5
// (graph.Build) of the fixed seven-stage validation pipeline (design.md
// D-H) against path. manifest.ParseFile's own error already names the
// exact path it failed to open/parse (internal/workflow/manifest/parse.go:
// "manifest: open %s: %w") — this function adds no fallback of its own and
// performs no implicit legacy --pipeline behavior on failure (design.md
// D-N, tasks.md 9.1).
func loadWorkflowManifest(path string) (*manifest.Manifest, *graph.Graph, error) {
	m, err := manifest.ParseFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("workflow: %w", err)
	}

	g, err := graph.Build(m.Spec.Steps)
	if err != nil {
		return nil, nil, fmt.Errorf("workflow: %w", err)
	}

	return m, g, nil
}

// selectWorkflowGraph returns g unchanged for a full workflow run
// (stepID == ""), or stepID's needs-transitive closure via engine.Closure
// when --step names a manifest step (design.md D-N: "--step <id>
// retargeted to a manifest step id ... executes the needs-transitive
// closure of <id> in topological order and stops"). This is the ONLY
// place main.go computes reachability — the actual algorithm lives in
// internal/workflow/engine/subgraph.go (WU8), per that package's own doc
// comment naming this wiring as Phase 9's job.
func selectWorkflowGraph(g *graph.Graph, stepID string) (*graph.Graph, error) {
	if stepID == "" {
		return g, nil
	}

	closure, err := engine.Closure(g, stepID)
	if err != nil {
		return nil, fmt.Errorf("workflow: %w", err)
	}
	return closure, nil
}

// listWorkflowSteps reports every manifest step's id, declared capability,
// and resolved provider name/version (design.md D-N). Providers are
// resolved against a nil Dagger client and an empty providers.Values map:
// resolution at listing time only needs to prove a step's uses.provider/
// uses.module is registered and version-supported (internal/workflow/
// providers.checkWithSchema only rejects a with-field kind mismatch for
// fields actually present in the Values it is given — an empty map can
// never trigger one), never to actually run anything.
func (c *CLI) listWorkflowSteps(ctx context.Context, m *manifest.Manifest) error {
	reg := providers.NewRegistry()
	providers.RegisterDefaults(reg, nil)

	infos, err := resolveStepInfos(m.Spec.Steps, reg)
	if err != nil {
		return err
	}

	logger.L().InfoContext(ctx, "Available workflow steps",
		"manifest", m.Metadata.Name,
		"steps_count", len(infos))

	for i, info := range infos {
		logger.L().InfoContext(ctx, "Step information",
			"index", i+1,
			"step", info.StepID,
			"capability", info.Capability,
			"provider", info.ProviderName,
			"provider_module", info.ProviderModule,
			"provider_version", info.ProviderVersion)
	}

	return nil
}

// resolveStepInfos resolves every step's declared uses.provider/uses.module
// against reg — see listWorkflowSteps' doc comment for why an empty
// providers.Values map is always safe here.
func resolveStepInfos(steps []manifest.Step, reg *providers.Registry) ([]workflowStepInfo, error) {
	infos := make([]workflowStepInfo, 0, len(steps))
	for _, s := range steps {
		ref := providers.Ref{Name: s.Uses.Provider, Module: s.Uses.Module, Version: s.Uses.Version}
		if err := resolveCapabilityRef(reg, s.Capability, ref); err != nil {
			return nil, fmt.Errorf("workflow: step %q: %w", s.ID, err)
		}
		infos = append(infos, workflowStepInfo{
			StepID:          s.ID,
			Capability:      s.Capability,
			ProviderName:    ref.Name,
			ProviderModule:  ref.Module,
			ProviderVersion: ref.Version,
		})
	}
	return infos, nil
}

// resolveCapabilityRef dispatches ref to the Resolve* method matching
// capability — manifest.ValidateStructure (stage 3, already run by
// manifest.ParseFile) rejects any capability outside the five known
// values, so the default case here is defensive only, never reachable
// through the normal parse -> list pipeline.
func resolveCapabilityRef(reg *providers.Registry, capability string, ref providers.Ref) error {
	empty := providers.Values{}

	switch capability {
	case "build":
		_, err := reg.ResolveBuilder(ref, empty)
		return err
	case "test":
		_, err := reg.ResolveTester(ref, empty)
		return err
	case "artifact":
		_, err := reg.ResolveArtifactor(ref, empty)
		return err
	case "deploy":
		_, err := reg.ResolveDeployer(ref, empty)
		return err
	case "run":
		_, err := reg.ResolveRunner(ref, empty)
		return err
	default:
		return fmt.Errorf("unknown capability %q", capability)
	}
}

// executeWorkflow runs g's steps end-to-end (design.md D-N/D-K): a real
// Dagger client (via the app container — the same lazily-connected
// "daggerClient" component the legacy --pipeline path already uses),
// providers.RegisterDefaults' real in-repo capability implementations
// (WU3/WU7), the manifest's declared secrets/source/variables, and
// engine.Execute (WU8) for the wave-scheduled run itself. This is the
// wiring task.md 9.4 asks for — it does not reimplement any engine logic,
// it only assembles engine.Config and calls engine.Execute.
func (c *CLI) executeWorkflow(ctx context.Context, m *manifest.Manifest, g *graph.Graph, flags *Flags) error {
	client, err := c.workflowDaggerClient()
	if err != nil {
		return err
	}

	reg := providers.NewRegistry()
	providers.RegisterDefaults(reg, client)

	opts, err := engine.OptionsFromSpec(m.Spec.Execution)
	if err != nil {
		return fmt.Errorf("workflow: %w", err)
	}

	secrets, err := resolveWorkflowSecrets(client, m.Spec.Secrets)
	if err != nil {
		return fmt.Errorf("workflow: %w", err)
	}

	source, err := resolveWorkflowSource(client, m.Spec.Source)
	if err != nil {
		return fmt.Errorf("workflow: %w", err)
	}

	cfg := engine.Config{
		Steps:      m.Spec.Steps,
		Graph:      g,
		Registry:   reg,
		Source:     source,
		Variables:  m.Spec.Variables,
		Secrets:    secrets,
		Predicates: map[string]string{"branch": flags.branch},
		Options:    opts,
	}

	logger.L().InfoContext(ctx, "Executing workflow",
		"manifest", m.Metadata.Name,
		"steps", len(g.Nodes))

	res, err := engine.Execute(ctx, cfg)
	if err != nil {
		return fmt.Errorf("workflow: %w", err)
	}
	if res.Failed() {
		return fmt.Errorf("workflow: steps failed: %v", res.Failures)
	}

	logger.L().InfoContext(ctx, "Workflow completed successfully", "manifest", m.Metadata.Name)
	return nil
}

// workflowDaggerClient acquires the app container's lazily-connected
// "daggerClient" component — the identical component/connection mechanism
// the legacy --pipeline path already relies on (see
// executePipelineWithExecutor above); real execution genuinely needs a
// live Dagger engine, unlike loadWorkflowManifest/listWorkflowSteps above,
// which never touch it.
func (c *CLI) workflowDaggerClient() (*dagger.Client, error) {
	container := c.app.GetContainer()
	client, err := container.Get("daggerClient")
	if err != nil {
		return nil, fmt.Errorf("workflow: failed to acquire Dagger client: %w", err)
	}
	daggerClient, ok := client.(*dagger.Client)
	if !ok || daggerClient == nil {
		return nil, fmt.Errorf("workflow: Dagger client unavailable")
	}
	return daggerClient, nil
}

// resolveWorkflowSecrets binds every spec.secrets entry to a *dagger.Secret
// via client.SetSecret, reading each one's plaintext value from its
// declared FromEnv environment variable (internal/pipelines/shared/
// docker.go's existing client.SetSecret pattern, reused here per
// manifest/schema.go's SecretSpec doc comment). The plaintext value never
// leaves this function as anything other than the argument to SetSecret —
// engine.Config.Secrets only ever holds the resulting *dagger.Secret
// handles (design.md D-L).
func resolveWorkflowSecrets(client *dagger.Client, secrets map[string]manifest.SecretSpec) (map[string]*dagger.Secret, error) {
	if len(secrets) == 0 {
		return nil, nil
	}

	out := make(map[string]*dagger.Secret, len(secrets))
	for name, spec := range secrets {
		out[name] = client.SetSecret(name, os.Getenv(spec.FromEnv))
	}
	return out, nil
}

// resolveWorkflowSource binds spec.source to the Directory engine.Config.
// Source needs. Only a local spec.source.path is supported by this work
// unit's CLI wiring — a git-based spec.source.repo/ref source is a
// confirmed, deliberately out-of-scope gap for this work unit (flagged for
// sdd-verify, mirroring prior work units' "report what is provable, do not
// guess" / explicit-gap-flagging discipline): implementing a git clone
// path here would reach beyond "wire main.go to the already-built engine
// package" into new source-acquisition logic no earlier work unit built or
// tested.
func resolveWorkflowSource(client *dagger.Client, spec manifest.SourceSpec) (*dagger.Directory, error) {
	if spec.Repo != "" {
		return nil, fmt.Errorf("workflow: spec.source.repo (git-based source) is not supported by this CLI entrypoint yet — use spec.source.path")
	}

	path := spec.Path
	if path == "" {
		path = "."
	}
	return client.Host().Directory(path), nil
}

// Version and build information - set at build time
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// showVersion displays version information
func (c *CLI) showVersion() {
	logger.L().InfoContext(context.Background(), versionLogMessage,
		"version", Version,
		"go_version", runtime.Version(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"build_time", BuildTime,
		"git_commit", GitCommit)
}

// runHealthChecks executes health checks for all configured services
func (c *CLI) runHealthChecks(ctx context.Context, flags *Flags) error {
	// Load configuration
	cfg, err := config.NewConfigurationWrapper()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Load YAML configuration if specified and file exists
	// The config file is optional - if it doesn't exist, we use defaults
	if flags.configFile != "" {
		// Check if file exists before trying to load it
		if _, err := os.Stat(flags.configFile); err == nil {
			var yamlCfg *config.YAMLConfig
			yamlCfg, err = c.loadYAMLConfig(ctx, cfg, flags.configFile)
			if err != nil {
				return fmt.Errorf("failed to load YAML configuration: %w", err)
			}
			c.yamlConfig = yamlCfg
		}
		// If file doesn't exist, silently use defaults (no error)
	}

	logger.L().InfoContext(ctx, "Running health checks")

	return app.RunHealthChecks(ctx, cfg)
}

func main() {
	cli := NewCLI()

	if err := cli.Run(os.Args[1:]); err != nil {
		logger.L().ErrorContext(context.Background(), "CLI execution failed", "error", err)
		os.Exit(1)
	}
}
