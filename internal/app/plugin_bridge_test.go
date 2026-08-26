package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/internal/config"
	"github.com/pablogore/shipwright/internal/workflow/providers"
)

// nomadDeployRef is the providers.Ref the one compile-time-registered
// builtin plugin (registerPluginComponents' "nomad-deploy") registers its
// shipwright.Deployer under.
var nomadDeployRef = providers.Ref{Name: "nomad-deploy", Version: "1"}

func startedContainer(t *testing.T) *Container {
	t.Helper()

	cfg, err := config.NewConfigurationWrapper()
	require.NoError(t, err)

	container := NewContainer(t.Context(), cfg)
	require.NoError(t, container.Start(t.Context()))

	return container
}

// TestApp_LoadAndInitializePlugins_RegistersBuiltinProvidersIntoRegistry is
// the RED test for the wiring gap the second PR review found: before this
// work unit the plugin load path had no production caller AND no way to
// contribute anything engine.Execute could resolve. This asserts the whole
// bridge end to end at the app layer — builtin plugin loaded, initialized,
// and its Deployer resolvable from the caller's own registry.
func TestApp_LoadAndInitializePlugins_RegistersBuiltinProvidersIntoRegistry(t *testing.T) {
	a := NewApp(startedContainer(t))
	reg := providers.NewRegistry()

	require.NoError(t, a.LoadAndInitializePlugins(t.Context(), reg, nil))

	deployer, err := reg.ResolveDeployer(nomadDeployRef, providers.Values{})
	require.NoError(t, err)
	assert.NotNil(t, deployer)
}

// TestApp_LoadAndInitializePlugins_NilRegistryFailsClosed keeps the caller
// honest: loading plugins with nowhere to register their capabilities is a
// programming error, not a silent no-op that looks like success.
func TestApp_LoadAndInitializePlugins_NilRegistryFailsClosed(t *testing.T) {
	a := NewApp(startedContainer(t))

	err := a.LoadAndInitializePlugins(t.Context(), nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider registry")
}

// TestApp_LoadAndInitializePlugins_IsIdempotent proves a second call in the
// same process (for example --list-steps followed by a run in one test
// binary) does not fail on "already registered".
func TestApp_LoadAndInitializePlugins_IsIdempotent(t *testing.T) {
	a := NewApp(startedContainer(t))

	require.NoError(t, a.LoadAndInitializePlugins(t.Context(), providers.NewRegistry(), nil))
	require.NoError(t, a.LoadAndInitializePlugins(t.Context(), providers.NewRegistry(), nil))
}

func TestApp_CleanupPlugins_Succeeds(t *testing.T) {
	a := NewApp(startedContainer(t))
	require.NoError(t, a.LoadAndInitializePlugins(t.Context(), providers.NewRegistry(), nil))

	assert.NoError(t, a.CleanupPlugins(t.Context()))
}

// TestApp_CleanupPlugins_WithoutLoadIsSafe covers executeWorkflow's deferred
// cleanup firing after a load failure that registered nothing.
func TestApp_CleanupPlugins_WithoutLoadIsSafe(t *testing.T) {
	a := NewApp(startedContainer(t))

	assert.NoError(t, a.CleanupPlugins(t.Context()))
}

// unstartedContainer has none of its components registered, so every
// container.Get lookup fails — the shape both methods' "failed to get ..."
// error paths guard against.
func unstartedContainer(t *testing.T) *Container {
	t.Helper()

	cfg, err := config.NewConfigurationWrapper()
	require.NoError(t, err)

	return NewContainer(t.Context(), cfg)
}

func TestApp_LoadAndInitializePlugins_MissingComponentFailsClosed(t *testing.T) {
	a := NewApp(unstartedContainer(t))

	err := a.LoadAndInitializePlugins(t.Context(), providers.NewRegistry(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "plugin registry")
}

func TestApp_CleanupPlugins_MissingComponentFailsClosed(t *testing.T) {
	a := NewApp(unstartedContainer(t))

	err := a.CleanupPlugins(t.Context())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "plugin registry")
}
