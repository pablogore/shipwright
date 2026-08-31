package plugins

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"dagger.io/dagger"

	"github.com/pablogore/kit-logger/pkg/logger"

	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/internal/workflow/providers"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// nomadProviderName/nomadProviderVersion are the providers.Ref identity this
// plugin registers its shipwright.Deployer under. A manifest step reaches it
// with:
//
//	capability: deploy
//	uses:
//	  provider: nomad-deploy
//	  version: "1"
const (
	nomadProviderName    = "nomad-deploy"
	nomadProviderVersion = "1"
)

// defaultNomadAddr/defaultNomadJobFile are the local-development fallbacks
// used when neither the manifest's `with` block nor plugin configuration
// supplies a value.
const (
	defaultNomadAddr    = "http://localhost:4646"
	defaultNomadJobFile = "nomad.hcl"
)

// nomadTokenSecretName is the Dagger secret name the deploy credential is
// bound to. The credential is only ever handed to the container as a
// *dagger.Secret via WithSecretVariable — never interpolated into a command
// string or a plain WithEnvVariable (the security skill's secrets rule).
const nomadTokenSecretName = "NOMAD_TOKEN"

// ErrNomadDeploymentNotImplemented is returned by Deploy after validation and
// staging succeed. stageNomadJob builds a Nomad CLI container but never
// executes it, and runNomadCommand only checks for a local Nomad binary and
// logs — neither performs a real `nomad job run` or calls the Nomad API. A
// shipwright.Deployer's returned string is recorded by engine.Execute as the
// step's successful output, so fabricating a "nomad://..." reference here
// would let a workflow report a deployment that never happened. Deploy fails
// closed with this sentinel instead, until real execution is implemented.
var ErrNomadDeploymentNotImplemented = errors.New("nomad-deploy: real Nomad execution is not implemented yet — no deployment occurred")

// NomadDeployPlugin adds Nomad deployment capability to Shipwright workflows.
//
// It is BOTH the Plugin (lifecycle: Name/Version/Initialize/Cleanup) and the
// shipwright.Deployer (Layer 1 capability) it contributes. Initialize
// registers a Deployer factory into the run's *providers.Registry through
// PluginContext.GetProviderRegistry(); the factory returns a per-step copy of
// this plugin bound to that step's typed `with` values, which is what
// internal/workflow/engine.dispatchDeploy actually calls.
//
// Why this replaced the previous hook/step registration: the earlier
// implementation registered a "push"-after hook on interfaces.HookManager and
// a "deploy-nomad" handler on interfaces.StepRegistry. Both were only ever
// executed by the preset-driven step flow that this change's preset-deletion
// work unit removed — engine.Execute consults neither. Those registrations
// therefore made the plugin LOOK wired while guaranteeing its deploy logic
// could never run. They are removed rather than kept as compatibility dead
// weight.
type NomadDeployPlugin struct {
	config  map[string]interface{}
	name    string
	version string

	// nomadAddr/job/jobFile are the resolved per-step settings. They are set
	// only on the copies the registered provider factory produces; the
	// long-lived plugin instance registered with PluginRegistry leaves them
	// empty.
	nomadAddr string
	job       string
	jobFile   string

	// daggerClient is the run's Dagger client accessor, captured from the
	// PluginContext at Initialize time. It is a func (not a *dagger.Client)
	// because a run may wire no client at all — for example --list-steps,
	// which resolves providers without connecting to a Dagger engine.
	daggerClient func() (*dagger.Client, error)
}

// Compile-time proof that the plugin really satisfies the public Layer 1
// deploy contract engine.dispatchDeploy resolves.
var _ shipwright.Deployer = (*NomadDeployPlugin)(nil)

// NewNomadDeployPlugin creates a new Nomad deploy plugin.
func NewNomadDeployPlugin() Plugin {
	return &NomadDeployPlugin{
		config:  make(map[string]interface{}),
		name:    nomadProviderName,
		version: "1.0.0",
	}
}

// Name returns the plugin name.
func (p *NomadDeployPlugin) Name() string {
	return p.name
}

// Version returns the plugin version.
func (p *NomadDeployPlugin) Version() string {
	return p.version
}

// Initialize reads plugin configuration and registers this plugin's
// shipwright.Deployer into the run's provider registry.
//
// A nil registry (no workflow wired for this run) is not an error: the plugin
// simply contributes nothing, matching PluginContext.GetProviderRegistry's
// documented contract.
func (p *NomadDeployPlugin) Initialize(ctx context.Context, pluginCtx PluginContext) error {
	logger.L().InfoContext(ctx, "Initializing Nomad deploy plugin")

	if cfg := pluginCtx.GetConfiguration(); cfg != nil {
		if nomadConfig := cfg.Get("plugins.nomad-deploy"); nomadConfig != nil {
			if configMap, ok := nomadConfig.(map[string]interface{}); ok {
				p.config = configMap
			}
		}
	}

	p.daggerClient = pluginCtx.GetDaggerClient

	reg := pluginCtx.GetProviderRegistry()
	if reg == nil {
		logger.L().WarnContext(ctx,
			"Nomad deploy plugin initialized without a provider registry — no deploy provider contributed",
			"plugin", p.name)
		return nil
	}

	reg.RegisterDeployer(
		providers.Ref{Name: nomadProviderName, Version: nomadProviderVersion},
		p.withSchema(),
		p.newDeployer,
	)

	logger.L().InfoContext(ctx, "Nomad deploy provider registered",
		"provider", nomadProviderName,
		"version", nomadProviderVersion)

	return nil
}

// withSchema declares the `with` fields a deploy step may pass, so
// providers.checkWithSchema rejects a kind mismatch at resolution time rather
// than letting an unexpected value reach Deploy.
//
// artifactRef/environment/creds are the engine's own deploy field names
// (internal/workflow/engine/execute.go's deployArtifactRefField and
// siblings); nomadAddr/job/jobFile are this provider's own settings.
func (p *NomadDeployPlugin) withSchema() providers.WithSchema {
	return providers.WithSchema{
		"artifactRef": interp.KindString,
		"environment": interp.KindString,
		"creds":       interp.KindSecret,
		"nomadAddr":   interp.KindString,
		"job":         interp.KindString,
		"jobFile":     interp.KindString,
	}
}

// newDeployer is the provider factory registered with the Registry. It binds
// one step's typed values onto a copy of the plugin, falling back to plugin
// configuration and then to the documented defaults.
func (p *NomadDeployPlugin) newDeployer(v providers.Values) shipwright.Deployer {
	return &NomadDeployPlugin{
		config:       p.config,
		name:         p.name,
		version:      p.version,
		nomadAddr:    valueOr(v, "nomadAddr", p.getConfigString("nomad_addr", "")),
		job:          valueOr(v, "job", p.getConfigString("nomad_job", "")),
		jobFile:      valueOr(v, "jobFile", p.getConfigString("nomad_job_file", defaultNomadJobFile)),
		daggerClient: p.daggerClient,
	}
}

// Deploy implements shipwright.Deployer: it deploys an already-published
// artifact reference into a named environment and returns the deployment
// reference the engine records as this step's output.
func (p *NomadDeployPlugin) Deploy(
	ctx context.Context,
	artifactRef string,
	environment string,
	creds *dagger.Secret,
) (string, error) {
	if artifactRef == "" {
		return "", errors.New("nomad-deploy: artifactRef must not be empty")
	}
	if p.job == "" && p.jobFile == "" {
		return "", errors.New("nomad-deploy: a nomad job or job file must be specified")
	}

	addr := p.resolvedNomadAddr()

	logger.L().InfoContext(ctx, "Deploying to Nomad",
		"artifact", artifactRef,
		"environment", environment,
		"nomad_addr", addr)

	if err := p.stageNomadJob(ctx, addr, environment, creds); err != nil {
		return "", err
	}

	if err := p.runNomadCommand(ctx, addr, p.jobFile, artifactRef); err != nil {
		return "", fmt.Errorf("nomad-deploy: %w", err)
	}

	return "", fmt.Errorf("%w (artifact=%s environment=%s)", ErrNomadDeploymentNotImplemented, artifactRef, environment)
}

// Cleanup cleans up the plugin.
func (p *NomadDeployPlugin) Cleanup(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Cleaning up Nomad deploy plugin")
	return nil
}

// resolvedNomadAddr returns the configured Nomad address, or the documented
// local-development default.
func (p *NomadDeployPlugin) resolvedNomadAddr() string {
	if p.nomadAddr != "" {
		return p.nomadAddr
	}
	return defaultNomadAddr
}

// stageNomadJob prepares the Nomad CLI container carrying the job definition.
// It is a no-op when the run wired no Dagger client (for example a resolution
// only path such as --list-steps, or a unit test) — the local CLI fallback in
// runNomadCommand still applies.
//
// The deploy credential crosses into the container exclusively as a
// *dagger.Secret through WithSecretVariable, and the job body is written with
// WithNewFile rather than interpolated into an `sh -c` string (the security
// skill's secrets and command-construction rules).
//
// Scope, stated explicitly: the staged container is BUILT and deliberately
// NOT executed, exactly as the previous hook-based deployToNomad did.
// Actually running `nomad job run` inside it needs a reachable Nomad server
// and is out of scope for this remediation, which restores the plugin's
// reachability rather than adding new deployment behavior.
//nolint:unparam // always nil today by design: staging deliberately never executes (see doc comment above); the error return stays part of the signature for when real `nomad job run` execution lands
func (p *NomadDeployPlugin) stageNomadJob(
	ctx context.Context,
	addr string,
	environment string,
	creds *dagger.Secret,
) error {
	if p.daggerClient == nil {
		return nil
	}

	client, err := p.daggerClient()
	if err != nil || client == nil {
		logger.L().WarnContext(ctx, "Dagger client unavailable, using local Nomad CLI path only")
		return nil
	}

	container := client.Container().
		From("hashicorp/nomad:2.0.5").
		WithEnvVariable("NOMAD_ADDR", addr)

	if environment != "" {
		container = container.WithEnvVariable("NOMAD_NAMESPACE", environment)
	}
	if creds != nil {
		container = container.WithSecretVariable(nomadTokenSecretName, creds)
	}
	if p.job != "" {
		container = container.WithNewFile("/tmp/job.hcl", p.job)
	}
	_ = container // built, not executed — see this function's doc comment.

	return nil
}

// runNomadCommand exercises the local Nomad CLI path when it is available. A
// missing CLI is deliberately not a failure: it keeps a workflow authored
// against Nomad runnable on a machine with no Nomad installed, the same
// tolerance the previous hook-based implementation had. Like stageNomadJob,
// issuing the real `nomad job run` invocation is out of scope here.
//nolint:unparam // always nil today by design: this only checks for a local Nomad CLI and logs (see doc comment above); the error return stays part of the signature for when real `nomad job run` execution lands
func (p *NomadDeployPlugin) runNomadCommand(ctx context.Context, nomadAddr, jobFile, artifactRef string) error {
	if _, err := exec.LookPath("nomad"); err != nil {
		logger.L().WarnContext(ctx, "Nomad CLI not found, skipping local deployment",
			"hint", "Install the Nomad CLI or target the Nomad API directly")
		return nil
	}

	logger.L().InfoContext(ctx, "Would deploy Nomad job",
		"nomad_addr", nomadAddr,
		"file", jobFile,
		"artifact", artifactRef)

	return nil
}

// getConfigString gets a string value from config with default.
func (p *NomadDeployPlugin) getConfigString(key, defaultValue string) string {
	if value, ok := p.config[key].(string); ok && value != "" {
		return value
	}
	return defaultValue
}

// valueOr returns v[key]'s string form, or fallback when the field is absent
// or not string-representable. A secret Value never yields a string here
// (interp.Value has no string accessor for KindSecret), so a credential can
// never silently become one of these plain settings.
func valueOr(v providers.Values, key, fallback string) string {
	val, ok := v[key]
	if !ok {
		return fallback
	}
	s, ok := val.String()
	if !ok || s == "" {
		return fallback
	}
	return s
}
