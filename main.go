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
	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/workflow/engine"
	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
	"github.com/pablogore/shipwright/internal/workflow/providers"
)

// defaultWorkflowManifestPath is --workflow's registered default (design.md
// D-N: "Default .shipwright/workflow.yaml").
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
		"workflow", flags.workflow,
		"environment", flags.env,
		"verbose", flags.verbose)

	// Get the application instance
	c.app = app.NewApp(app.GetContainer())

	// The manifest-driven entrypoint (design.md D-N) is now the CLI's sole
	// dispatch path — the legacy --pipeline preset registry, its factory
	// functions, and every flag that only made sense against a named
	// preset (--pipeline, --list-pipelines, --only-build, --only-test,
	// --skip-push) were deleted in this work unit (tasks.md 11.3/11.4),
	// which is why --step/--list-steps no longer need a mode switch: they
	// always target the manifest now (runWorkflowCLI handles both). A
	// missing/unreadable manifest fails closed naming the expected path
	// (tasks.md 9.1) — there is no other path left to fall back to.
	return c.runWorkflowCLI(ctx, flags)
}

// Flags represents the command line flags.
type Flags struct {
	coverage   float64
	branch     string
	env        string
	executor   string
	gitAuth    string
	gitRef     string
	health     bool
	listSteps  bool
	local      bool
	step       string
	verbose    bool
	version    bool
	configFile string

	// workflow is --workflow's path value — the manifest path for the
	// sole CLI entrypoint (design.md D-N). Defaults to
	// defaultWorkflowManifestPath when not explicitly passed.
	workflow string
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

	flagSet.Float64Var(&flags.coverage, "coverage", 90, "Minimum coverage percentage required (in: 90 for 90%)")
	flagSet.StringVar(&flags.branch, "branch", "develop", "Branch name")
	flagSet.StringVar(&flags.env, "env", "dev", "Environment: dev, staging, prod")
	flagSet.StringVar(&flags.executor, "executor", "", "Executor to use: native, docker (empty for auto-detection)")
	flagSet.BoolVar(&flags.verbose, "verbose", false, "Verbose mode")
	flagSet.StringVar(&flags.gitRef, "git-ref", "main", "Branch name (default: main)")
	flagSet.StringVar(&flags.step, "step", "", "Workflow step to execute — the needs-transitive closure of this step id")
	flagSet.StringVar(&flags.gitAuth, "git-auth", defaultGitAuth, "Git authentication method: ssh or https")
	flagSet.BoolVar(&flags.listSteps, "list-steps", false, "List available workflow steps")
	flagSet.StringVar(&flags.configFile, "config", ".shipwright.yml", "Configuration file path")
	flagSet.BoolVar(&flags.version, "version", false, "Show version information")
	flagSet.BoolVar(&flags.local, "local", false, "Run pipeline locally without Docker")
	flagSet.BoolVar(&flags.health, "health", false, "Run health checks for Dagger, registry, and Git")
	flagSet.StringVar(&flags.workflow, "workflow", defaultWorkflowManifestPath,
		"Path to a declarative workflow manifest — the CLI's sole entrypoint (design.md D-N). "+
			"--step/--list-steps target this manifest")

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

	// Plugins are wired here too, with a nil Dagger client, so --list-steps
	// resolves against the SAME provider set an actual run does. Without
	// this, a manifest step naming a plugin-contributed provider would be
	// reported as unregistered by --list-steps while executing correctly —
	// a listing that lies about the run.
	return runWithPluginLifecycle(ctx, c.app, reg, nil, func() error {
		return reportWorkflowSteps(ctx, m, reg)
	})
}

// reportWorkflowSteps resolves and logs every step's provider. Split out of
// listWorkflowSteps so the plugin lifecycle wraps exactly the resolution and
// reporting work.
func reportWorkflowSteps(ctx context.Context, m *manifest.Manifest, reg *providers.Registry) error {
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

	return runWithPluginLifecycle(ctx, c.app, reg, client, func() error {
		return c.runWorkflowEngine(ctx, m, g, flags, reg, client)
	})
}

// pluginLifecycle is the plugin load/cleanup surface the workflow entrypoint
// depends on, satisfied by *app.App. Declaring it here (consumer-side) keeps
// runWithPluginLifecycle's deferred-cleanup contract unit-testable without a
// DI container or a live Dagger engine.
type pluginLifecycle interface {
	LoadAndInitializePlugins(ctx context.Context, reg *providers.Registry, client *dagger.Client) error
	CleanupPlugins(ctx context.Context) error
}

// runWithPluginLifecycle wires the configured plugins into reg, runs run, and
// always cleans the plugins up afterwards.
//
// Cleanup is deferred BEFORE the load error is checked on purpose: plugin
// loading initializes plugins one at a time, so a failure on the third plugin
// leaves the first two initialized and needing cleanup. Cleanup failure is
// logged, never propagated — it must not mask the workflow's real outcome.
func runWithPluginLifecycle(
	ctx context.Context,
	lc pluginLifecycle,
	reg *providers.Registry,
	client *dagger.Client,
	run func() error,
) error {
	defer func() {
		if err := lc.CleanupPlugins(ctx); err != nil {
			logger.L().WarnContext(ctx, "Plugin cleanup failed", "error", err)
		}
	}()

	if err := lc.LoadAndInitializePlugins(ctx, reg, client); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}

	return run()
}

// runWorkflowEngine assembles engine.Config and calls engine.Execute. It runs
// inside runWithPluginLifecycle, so reg already carries both the in-repo
// providers (RegisterDefaults) and every provider the loaded plugins
// contributed.
func (c *CLI) runWorkflowEngine(
	ctx context.Context,
	m *manifest.Manifest,
	g *graph.Graph,
	flags *Flags,
	reg *providers.Registry,
	client *dagger.Client,
) error {
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
// "daggerClient" component. Real execution genuinely needs a live Dagger
// engine, unlike loadWorkflowManifest/listWorkflowSteps above, which never
// touch it — listWorkflowSteps deliberately wires plugins with a nil client
// for exactly that reason.
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
