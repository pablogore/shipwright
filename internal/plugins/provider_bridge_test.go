package plugins

import (
	"context"
	"errors"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/internal/workflow/providers"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// deployRef is the providers.Ref NomadDeployPlugin registers itself under.
// Kept as one variable so every test below fails for the SAME reason if the
// registration identity ever drifts.
var deployRef = providers.Ref{Name: nomadProviderName, Version: nomadProviderVersion}

// mustString/mustSecret are tiny typed-Value fixtures shared by the tests in
// this package. mustSecret's nil handle is deliberate: interp.Value's own
// guarantee is that a KindSecret value has NO string accessor, which is the
// property under test — the handle itself is never dereferenced.
func mustString(s string) interp.Value { return interp.NewString(s) }

func mustSecret() interp.Value { return interp.NewSecret(nil) }

// --- PluginContext provider-registry accessor -------------------------------

// TestPluginContext_GetProviderRegistry_ReturnsInjectedRegistry is the RED
// test for the bridge itself: a plugin must be able to reach the SAME
// *providers.Registry the workflow engine will resolve against, so it can
// register a capability implementation through WU7's already-tested
// Register*/Resolve* primitives instead of a new per-capability interface
// method on Plugin.
func TestPluginContext_GetProviderRegistry_ReturnsInjectedRegistry(t *testing.T) {
	reg := providers.NewRegistry()

	ctx := NewPluginContext(nil, nil, nil, nil, Capabilities{}, reg, pipelines.Config{}, nil)

	assert.Same(t, reg, ctx.GetProviderRegistry(),
		"plugins must receive the identical registry instance the engine resolves against, not a copy")
}

// TestPluginContext_GetProviderRegistry_NilWhenNotWired proves the accessor
// is honest about an unwired run rather than fabricating an empty registry a
// plugin would silently register into and never be resolved from.
func TestPluginContext_GetProviderRegistry_NilWhenNotWired(t *testing.T) {
	ctx := NewPluginContext(nil, nil, nil, nil, Capabilities{}, nil, pipelines.Config{}, nil)

	assert.Nil(t, ctx.GetProviderRegistry())
}

// --- builtin-only loading (design.md D-I security boundary) -----------------

// TestLoader_ListBuiltins_ReturnsRegisteredNames is the RED test for the
// enumeration LoadBuiltinPlugins needs: without it there is no way to load
// every compile-time-registered plugin WITHOUT going through
// LoadFromConfig, which is the only path that can reach plugin.Open.
func TestLoader_ListBuiltins_ReturnsRegisteredNames(t *testing.T) {
	l := NewLoader()
	l.RegisterBuiltin("alpha", func() Plugin { return NewMockPlugin() })
	l.RegisterBuiltin("beta", func() Plugin { return NewMockPlugin() })

	got := l.ListBuiltins()

	assert.ElementsMatch(t, []string{"alpha", "beta"}, got)
}

func TestLoader_ListBuiltins_EmptyWhenNoneRegistered(t *testing.T) {
	assert.Empty(t, NewLoader().ListBuiltins())
}

// TestRegistry_LoadBuiltinPlugins_LoadsAndInitializesEveryBuiltin is the RED
// test for the load path the --workflow entrypoint uses.
func TestRegistry_LoadBuiltinPlugins_LoadsAndInitializesEveryBuiltin(t *testing.T) {
	var initialized []string

	loader := NewMockPluginLoader()
	loader.ListBuiltinsFunc = func() []string { return []string{"alpha"} }
	loader.LoadBuiltinFunc = func(_ context.Context, name string) (Plugin, error) {
		p := NewMockPlugin()
		p.NameFunc = func() string { return name }
		p.InitializeFunc = func(context.Context, PluginContext) error {
			initialized = append(initialized, name)
			return nil
		}
		return p, nil
	}

	r := NewRegistry(loader)
	err := r.LoadBuiltinPlugins(context.Background(), NewMockPluginContext())

	require.NoError(t, err)
	assert.Equal(t, []string{"alpha"}, initialized)
	assert.Equal(t, []string{"alpha"}, r.ListPlugins())
}

// TestRegistry_LoadBuiltinPlugins_NeverReachesFileOrConfigLoading is the
// SECURITY assertion for design.md D-I: the workflow entrypoint's plugin
// load must never touch LoadFromConfig/LoadFromFile, because
// LoadFromConfig's `type: file` branch calls plugin.Open on a config-supplied
// path — the exact arbitrary-native-code surface D-I rejected.
func TestRegistry_LoadBuiltinPlugins_NeverReachesFileOrConfigLoading(t *testing.T) {
	var fromFile, fromConfig int

	loader := NewMockPluginLoader()
	loader.ListBuiltinsFunc = func() []string { return []string{"alpha"} }
	loader.LoadBuiltinFunc = func(context.Context, string) (Plugin, error) { return NewMockPlugin(), nil }
	loader.LoadFromFileFunc = func(context.Context, string) (Plugin, error) { fromFile++; return nil, nil }
	loader.LoadFromConfigFunc = func(context.Context, map[string]interface{}) (Plugin, error) {
		fromConfig++
		return nil, nil
	}

	r := NewRegistry(loader)
	require.NoError(t, r.LoadBuiltinPlugins(context.Background(), NewMockPluginContext()))

	assert.Zero(t, fromFile, "LoadBuiltinPlugins must never call LoadFromFile (plugin.Open)")
	assert.Zero(t, fromConfig, "LoadBuiltinPlugins must never call LoadFromConfig (reaches plugin.Open via type: file)")
}

func TestRegistry_LoadBuiltinPlugins_NilContextFailsClosed(t *testing.T) {
	r := NewRegistry(NewMockPluginLoader())

	err := r.LoadBuiltinPlugins(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "plugin context")
}

func TestRegistry_LoadBuiltinPlugins_NilLoaderIsNoop(t *testing.T) {
	r := NewRegistry(nil)

	require.NoError(t, r.LoadBuiltinPlugins(context.Background(), NewMockPluginContext()))
	assert.Empty(t, r.ListPlugins())
}

// TestRegistry_LoadBuiltinPlugins_ReportsInitializeFailure proves an
// initialization failure is surfaced (named) rather than silently skipped —
// a plugin that failed to register its provider must not look like success.
func TestRegistry_LoadBuiltinPlugins_ReportsInitializeFailure(t *testing.T) {
	loader := NewMockPluginLoader()
	loader.ListBuiltinsFunc = func() []string { return []string{"broken"} }
	loader.LoadBuiltinFunc = func(_ context.Context, name string) (Plugin, error) {
		p := NewMockPlugin()
		p.NameFunc = func() string { return name }
		p.InitializeFunc = func(context.Context, PluginContext) error { return errors.New("boom") }
		return p, nil
	}

	r := NewRegistry(loader)
	err := r.LoadBuiltinPlugins(context.Background(), NewMockPluginContext())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken")
}

// TestRegistry_LoadBuiltinPlugins_SkipsAlreadyRegistered proves calling the
// load path twice (or after a manual RegisterPlugin) is not an error — the
// CLI wires it per run and must stay idempotent.
func TestRegistry_LoadBuiltinPlugins_SkipsAlreadyRegistered(t *testing.T) {
	loader := NewMockPluginLoader()
	loader.ListBuiltinsFunc = func() []string { return []string{"alpha"} }
	loader.LoadBuiltinFunc = func(_ context.Context, name string) (Plugin, error) {
		p := NewMockPlugin()
		p.NameFunc = func() string { return name }
		return p, nil
	}

	r := NewRegistry(loader)
	require.NoError(t, r.LoadBuiltinPlugins(context.Background(), NewMockPluginContext()))
	require.NoError(t, r.LoadBuiltinPlugins(context.Background(), NewMockPluginContext()))

	assert.Equal(t, []string{"alpha"}, r.ListPlugins())
}

// --- NomadDeployPlugin as a real shipwright.Deployer ------------------------

// TestNomadDeployPlugin_ImplementsShipwrightDeployer is the compile-level
// conformance assertion (testing-tdd skill's double order 4 / .dagger
// adapter-conformance pattern): before this work unit NomadDeployPlugin
// implemented no Layer 1 capability at all, so its behavior could never fire
// from engine.Execute.
func TestNomadDeployPlugin_ImplementsShipwrightDeployer(t *testing.T) {
	var d shipwright.Deployer = &NomadDeployPlugin{}
	assert.NotNil(t, d)
}

// TestNomadDeployPlugin_Initialize_RegistersDeployerIntoProviderRegistry is
// the core RED test of this remediation: after Initialize, the SAME registry
// engine.Execute resolves against must hand back a working Deployer for the
// plugin's Ref.
func TestNomadDeployPlugin_Initialize_RegistersDeployerIntoProviderRegistry(t *testing.T) {
	reg := providers.NewRegistry()
	pluginCtx := NewMockPluginContext()
	pluginCtx.GetProviderRegistryFunc = func() *providers.Registry { return reg }

	p := NewNomadDeployPlugin()
	require.NoError(t, p.Initialize(context.Background(), pluginCtx))

	got, err := reg.ResolveDeployer(deployRef, providers.Values{})
	require.NoError(t, err)
	require.NotNil(t, got)
}

// TestNomadDeployPlugin_Initialize_HonorsWithValues proves the registered
// factory reads the step's typed `with` values through WU7's Values carrier
// (never an interpolated shell string, per the security skill's sh -c rule).
func TestNomadDeployPlugin_Initialize_HonorsWithValues(t *testing.T) {
	reg := providers.NewRegistry()
	pluginCtx := NewMockPluginContext()
	pluginCtx.GetProviderRegistryFunc = func() *providers.Registry { return reg }

	require.NoError(t, NewNomadDeployPlugin().Initialize(context.Background(), pluginCtx))

	got, err := reg.ResolveDeployer(deployRef, providers.Values{
		"nomadAddr": interp.NewString("http://nomad.internal:4646"),
		"jobFile":   interp.NewString("deploy/api.hcl"),
	})
	require.NoError(t, err)

	np, ok := got.(*NomadDeployPlugin)
	require.True(t, ok, "registered factory must produce a *NomadDeployPlugin")
	assert.Equal(t, "http://nomad.internal:4646", np.nomadAddr)
	assert.Equal(t, "deploy/api.hcl", np.jobFile)
}

// TestNomadDeployPlugin_Initialize_RejectsWithKindMismatch proves the
// declared WithSchema is real: a manifest passing a secret where a string is
// required fails closed at resolution (providers.checkWithSchema), it does
// not reach Deploy.
func TestNomadDeployPlugin_Initialize_RejectsWithKindMismatch(t *testing.T) {
	reg := providers.NewRegistry()
	pluginCtx := NewMockPluginContext()
	pluginCtx.GetProviderRegistryFunc = func() *providers.Registry { return reg }

	require.NoError(t, NewNomadDeployPlugin().Initialize(context.Background(), pluginCtx))

	_, err := reg.ResolveDeployer(deployRef, providers.Values{
		"nomadAddr": interp.NewInt(4646),
	})

	var mismatch *providers.WithSchemaMismatchError
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, "nomadAddr", mismatch.Field)
}

// TestNomadDeployPlugin_Initialize_WithoutProviderRegistryIsNoop keeps the
// plugin usable in a run that wires no provider registry (for example a unit
// test constructing a bare PluginContext) instead of panicking.
func TestNomadDeployPlugin_Initialize_WithoutProviderRegistryIsNoop(t *testing.T) {
	pluginCtx := NewMockPluginContext()
	pluginCtx.GetProviderRegistryFunc = func() *providers.Registry { return nil }

	assert.NoError(t, NewNomadDeployPlugin().Initialize(context.Background(), pluginCtx))
}

// TestNomadDeployPlugin_Initialize_DoesNotRegisterHooksOrSteps locks in this
// work unit's deletion decision: HookManager/StepRegistry have no live
// production consumer after WU11 removed the legacy step-execution flow, so
// registering into them was dead weight that made the plugin LOOK wired
// while its behavior could never fire.
func TestNomadDeployPlugin_Initialize_DoesNotRegisterHooksOrSteps(t *testing.T) {
	var hooks, steps int

	hookManager := NewMockHookManager()
	hookManager.RegisterHookFunc = func(string, interfaces.HookType, interfaces.HookFunc) error { hooks++; return nil }

	stepRegistry := NewMockStepRegistry()
	stepRegistry.RegisterStepFunc = func(string, interfaces.StepHandler) error { steps++; return nil }

	pluginCtx := NewMockPluginContext()
	pluginCtx.GetHookManagerFunc = func() interfaces.HookManager { return hookManager }
	pluginCtx.GetStepRegistryFunc = func() interfaces.StepRegistry { return stepRegistry }
	pluginCtx.GetProviderRegistryFunc = providers.NewRegistry

	require.NoError(t, NewNomadDeployPlugin().Initialize(context.Background(), pluginCtx))

	assert.Zero(t, hooks, "no live consumer executes hooks after WU11 — registering one is dead weight")
	assert.Zero(t, steps, "no live consumer executes StepRegistry steps after WU11")
}

// --- Deploy behavior --------------------------------------------------------

func TestNomadDeployPlugin_Deploy_RejectsEmptyArtifactRef(t *testing.T) {
	p := &NomadDeployPlugin{jobFile: "nomad.hcl"}

	_, err := p.Deploy(context.Background(), "", "staging", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "artifactRef")
}

func TestNomadDeployPlugin_Deploy_RejectsMissingJobAndJobFile(t *testing.T) {
	p := &NomadDeployPlugin{}

	_, err := p.Deploy(context.Background(), "ghcr.io/acme/api:v1", "staging", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "job")
}

// TestNomadDeployPlugin_Deploy_ReturnsDeploymentReference asserts the
// observable contract engine.Execute records as the step's output: a
// non-empty deployment reference naming the environment and the artifact.
// TestNomadDeployPlugin_Deploy_FailsClosedNotImplemented proves Deploy never
// fabricates a deployment reference: neither stageNomadJob (builds a
// container, never executes it) nor runNomadCommand (checks for the Nomad
// CLI, never runs it) perform a real deployment, so a workflow step must see
// a distinguishable error, not a "nomad://..." success string it would
// otherwise report as a completed deployment.
func TestNomadDeployPlugin_Deploy_FailsClosedNotImplemented(t *testing.T) {
	p := &NomadDeployPlugin{nomadAddr: "http://localhost:4646", jobFile: "nomad.hcl"}

	got, err := p.Deploy(context.Background(), "ghcr.io/acme/api:v1", "staging", nil)

	require.ErrorIs(t, err, ErrNomadDeploymentNotImplemented)
	assert.Empty(t, got, "Deploy must never return a deployment reference for an unperformed deployment")
}

// TestNomadDeployPlugin_Deploy_DefaultsNomadAddr proves an unset address
// falls back to the documented local default rather than producing an
// empty NOMAD_ADDR, even though Deploy itself still fails closed.
func TestNomadDeployPlugin_Deploy_DefaultsNomadAddr(t *testing.T) {
	p := &NomadDeployPlugin{jobFile: "nomad.hcl"}

	_, err := p.Deploy(context.Background(), "ghcr.io/acme/api:v1", "staging", nil)

	require.ErrorIs(t, err, ErrNomadDeploymentNotImplemented)
	assert.Equal(t, defaultNomadAddr, p.resolvedNomadAddr())
}

// TestNomadDeployPlugin_Deploy_JobOnlyIsAccepted covers the inline-job branch
// (`job` set, `jobFile` empty), the alternative to a job file: it must reach
// the not-implemented fail-closed error, never the "job or job file must be
// specified" validation error, proving job-only input is accepted by
// validation even though execution itself is still unimplemented.
func TestNomadDeployPlugin_Deploy_JobOnlyIsAccepted(t *testing.T) {
	p := &NomadDeployPlugin{job: `job "api" {}`}

	_, err := p.Deploy(context.Background(), "ghcr.io/acme/api:v1", "prod", nil)

	require.ErrorIs(t, err, ErrNomadDeploymentNotImplemented)
}

// TestNomadDeployPlugin_StageNomadJob_NoDaggerClientIsNoop covers the
// --list-steps shape: providers resolve without any Dagger connection, so a
// Deployer produced there must not fail merely because no client exists.
func TestNomadDeployPlugin_StageNomadJob_NoDaggerClientIsNoop(t *testing.T) {
	p := &NomadDeployPlugin{jobFile: "nomad.hcl"}

	assert.NoError(t, p.stageNomadJob(context.Background(), defaultNomadAddr, "staging", nil))
}

// TestNomadDeployPlugin_StageNomadJob_UnavailableClientIsNoop covers the
// accessor-returns-error branch (PluginContext.GetDaggerClient's documented
// "dagger client not available").
func TestNomadDeployPlugin_StageNomadJob_UnavailableClientIsNoop(t *testing.T) {
	p := &NomadDeployPlugin{
		jobFile:      "nomad.hcl",
		daggerClient: func() (*dagger.Client, error) { return nil, errors.New("dagger client not available") },
	}

	assert.NoError(t, p.stageNomadJob(context.Background(), defaultNomadAddr, "staging", nil))
}

// TestNomadDeployPlugin_Deploy_WithUnavailableDaggerClient proves the whole
// Deploy path stays reachable (validation and staging both tolerate a
// missing engine) when no Dagger client is wired — it must still reach the
// SAME not-implemented fail-closed error, not a different one caused by the
// missing client, and it must never fabricate a deployment reference.
func TestNomadDeployPlugin_Deploy_WithUnavailableDaggerClient(t *testing.T) {
	p := &NomadDeployPlugin{
		jobFile:      "nomad.hcl",
		daggerClient: func() (*dagger.Client, error) { return nil, errors.New("dagger client not available") },
	}

	got, err := p.Deploy(context.Background(), "ghcr.io/acme/api:v1", "staging", nil)

	require.ErrorIs(t, err, ErrNomadDeploymentNotImplemented)
	assert.Empty(t, got)
}

func TestRegistry_LoadBuiltinPlugins_ReportsLoadFailure(t *testing.T) {
	loader := NewMockPluginLoader()
	loader.ListBuiltinsFunc = func() []string { return []string{"missing"} }
	loader.LoadBuiltinFunc = func(context.Context, string) (Plugin, error) {
		return nil, errors.New("builtin plugin missing not found")
	}

	err := NewRegistry(loader).LoadBuiltinPlugins(context.Background(), NewMockPluginContext())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestRegistry_LoadBuiltinPlugins_ReportsNilFactoryResult(t *testing.T) {
	loader := NewMockPluginLoader()
	loader.ListBuiltinsFunc = func() []string { return []string{"nilly"} }
	loader.LoadBuiltinFunc = func(context.Context, string) (Plugin, error) { return nil, nil }

	err := NewRegistry(loader).LoadBuiltinPlugins(context.Background(), NewMockPluginContext())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nilly")
}
