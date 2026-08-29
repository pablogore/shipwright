package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/internal/pipelines/shared"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
	"github.com/pablogore/shipwright/internal/workflow/providers"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// fakeDeployer/fakeRunner are bare, no-op shipwright.Deployer/Runner
// fixtures — no in-repo Deployer/Runner provider exists yet
// (pkg/shipwright.DeployConfig/RunConfig are still empty stubs, WU3/WU7),
// so these exist solely to prove resolveCapabilityRef's deploy/run
// dispatch branches without asserting anything about a concrete provider.
type fakeDeployer struct{}

func (fakeDeployer) Deploy(context.Context, string, string, *dagger.Secret) (string, error) {
	return "", nil
}

type fakeRunner struct{}

func (fakeRunner) Run(context.Context, *dagger.Directory) (*dagger.Container, error) {
	return nil, nil
}

// providersRegistryForTest builds a *providers.Registry with the real
// in-repo default providers registered (RegisterDefaults, WU7), the exact
// same registry-construction call listWorkflowSteps/executeWorkflow make
// in main.go — a nil Dagger client is safe here because these tests only
// resolve (never call a capability method), per RegisterDefaults' own doc
// comment.
func providersRegistryForTest(t *testing.T) *providers.Registry {
	t.Helper()
	reg := providers.NewRegistry()
	providers.RegisterDefaults(reg, nil)
	return reg
}

func TestCLI_Run_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "invalid flag should return error",
			args:    []string{"-invalid-flag"},
			wantErr: true,
		},
		{
			name:    "version flag should not error",
			args:    []string{"-version"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI()
			err := cli.Run(tt.args)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCLI_parseFlags_ErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid flags should not error",
			args:    []string{"-workflow", diamondManifestPath},
			wantErr: false,
		},
		{
			name:    "invalid flag should error",
			args:    []string{"-unknown-flag"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI()
			flags, err := cli.parseFlags(tt.args)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, flags)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, flags)
			}
		})
	}
}

func TestMain_ErrorOutput(t *testing.T) {
	// Test that main properly handles errors without panicking.
	// We can't call main() itself in a test (it calls os.Exit), so this
	// exercises the same flag-parsing path main() drives, with invalid args.
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	os.Args = []string{cliName, "-invalid-flag"}

	cli := NewCLI()
	flags, err := cli.parseFlags(os.Args[1:])
	require.Error(t, err)
	assert.Nil(t, flags)
}

func TestCLIIdentityConstants(t *testing.T) {
	// The CLI flagset name drives the "Usage of <name>:" line printed by
	// --help, and the version/init log messages are shown by --version and
	// on successful startup. Both MUST present the Shipwright identity and
	// MUST NOT contain any trace of the pre-rebrand product identity.
	assert.Equal(t, "shipwright", cliName)
	assert.Equal(t, "Shipwright version", versionLogMessage)
	assert.Equal(t, "Shipwright initialized successfully", initLogMessage)

	assert.NotContains(t, cliName, "syntegrity")
	assert.NotContains(t, versionLogMessage, "Syntegrity")
	assert.NotContains(t, initLogMessage, "Syntegrity")
}

const diamondManifestPath = "examples/workflow/diamond.yaml"

// TestCLI_parseFlags_WorkflowPath is the flag-parsing-level proof that
// --workflow is the CLI's sole entrypoint after this work unit (design.md
// D-N, tasks.md 11.1/11.2/11.4): there is no more mode switch between a
// legacy --pipeline path and a manifest-driven path — flags.workflow always
// holds the manifest path (its registered default, or an explicit
// override), and --step always parses as a manifest step id.
func TestCLI_parseFlags_WorkflowPath(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantWorkflow string
		wantStep     string
	}{
		{
			name:         "bare invocation uses the default manifest path",
			args:         []string{},
			wantWorkflow: defaultWorkflowManifestPath,
			wantStep:     "",
		},
		{
			name:         "explicit --workflow overrides the default path",
			args:         []string{"-workflow", diamondManifestPath, "-step", "unit"},
			wantWorkflow: diamondManifestPath,
			wantStep:     "unit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI()
			flags, err := cli.parseFlags(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.wantWorkflow, flags.workflow)
			assert.Equal(t, tt.wantStep, flags.step)
		})
	}
}

// TestCLI_Run_WorkflowMissingManifestFailsClosed is task 9.1's RED test:
// --workflow naming a manifest that does not exist MUST fail closed,
// naming the exact expected path, with NO implicit fallback to
// --pipeline go-service (or any other legacy) behavior. Exercised through
// the real CLI.Run dispatcher, not a lower-level helper, so the assertion
// covers the actual --workflow flag wiring end to end.
func TestCLI_Run_WorkflowMissingManifestFailsClosed(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.yaml")

	cli := NewCLI()
	err := cli.Run([]string{"-workflow", missing})

	require.Error(t, err)
	assert.Contains(t, err.Error(), missing,
		"error must name the exact expected manifest path — no implicit legacy fallback")
}

// TestCLI_Run_WorkflowListSteps is task 9.3's RED/GREEN test: --list-steps
// combined with --workflow lists the manifest's step ids, capabilities,
// and resolved providers (design.md D-N) — never the legacy
// --pipeline-preset step listing. Runs the real diamond.yaml example
// (examples/workflow/diamond.yaml, WU8) through the actual CLI.Run
// dispatcher; providers.RegisterDefaults resolves against a nil Dagger
// client (see listWorkflowSteps' doc comment in main.go), so this needs no
// live Dagger engine.
func TestCLI_Run_WorkflowListSteps(t *testing.T) {
	cli := NewCLI()
	err := cli.Run([]string{"-workflow", diamondManifestPath, "-list-steps"})
	require.NoError(t, err)
}

// TestCLI_Run_WorkflowListSteps_UnregisteredProviderFailsClosed proves
// --list-steps' provider resolution is real, not cosmetic: a manifest
// step naming a provider no in-repo factory registers fails closed with
// an error naming the step.
func TestCLI_Run_WorkflowListSteps_UnregisteredProviderFailsClosed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unregistered-provider.yaml")
	manifestYAML := `apiVersion: shipwright.dev/v1
kind: Workflow
metadata:
  name: unregistered-provider-example
spec:
  source:
    path: .
  steps:
    - id: build
      capability: build
      uses:
        provider: does-not-exist
        version: "1"
`
	require.NoError(t, os.WriteFile(path, []byte(manifestYAML), 0o600))

	cli := NewCLI()
	err := cli.Run([]string{"-workflow", path, "-list-steps"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "build")
}

// TestSelectWorkflowGraph_FullRunWhenNoStepGiven proves selectWorkflowGraph
// returns g unchanged for a full workflow run (--step not given) — the
// exact same *graph.Graph selectWorkflowGraph would filter down for --step.
func TestSelectWorkflowGraph_FullRunWhenNoStepGiven(t *testing.T) {
	_, g, err := loadWorkflowManifest(diamondManifestPath)
	require.NoError(t, err)

	got, err := selectWorkflowGraph(g, "")
	require.NoError(t, err)
	assert.Same(t, g, got)
}

// TestSelectWorkflowGraph_StepClosure is task 9.2's RED test: --step <id>
// executes only <id>'s needs-transitive closure, in topological order —
// computed by engine.Closure (WU8's subgraph.go), never reimplemented
// here. "unit" needs only "build" in the diamond example, so its closure
// must contain exactly {build, unit}, in that wave order, excluding the
// sibling "vuln" step and the downstream "publish" step.
func TestSelectWorkflowGraph_StepClosure(t *testing.T) {
	_, g, err := loadWorkflowManifest(diamondManifestPath)
	require.NoError(t, err)

	closure, err := selectWorkflowGraph(g, "unit")
	require.NoError(t, err)

	assert.Equal(t, [][]string{{"build"}, {"unit"}}, closure.Waves,
		"closure must be exactly build's wave then unit's wave, in topological order")

	_, hasVuln := closure.Nodes["vuln"]
	_, hasPublish := closure.Nodes["publish"]
	assert.False(t, hasVuln, "closure must exclude unit's sibling step vuln")
	assert.False(t, hasPublish, "closure must exclude publish, which depends on unit, not the reverse")

	_, hasBuild := closure.Nodes["build"]
	_, hasUnit := closure.Nodes["unit"]
	assert.True(t, hasBuild)
	assert.True(t, hasUnit)
}

// TestSelectWorkflowGraph_UnknownStepFailsClosed proves --step naming a
// step id absent from the manifest fails closed via
// engine.UnknownStepError, wrapped with the "workflow:" prefix every other
// workflow-mode error in main.go uses.
func TestSelectWorkflowGraph_UnknownStepFailsClosed(t *testing.T) {
	_, g, err := loadWorkflowManifest(diamondManifestPath)
	require.NoError(t, err)

	_, err = selectWorkflowGraph(g, "does-not-exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does-not-exist")
}

// TestLoadWorkflowManifest_DiamondExample is the runtime-harness proof the
// work-unit table names ("shipwright --workflow
// examples/workflow/diamond.yaml"), exercised up to (but not including)
// actual Dagger execution — see this work unit's own report for exactly
// what remains manual. It proves manifest.ParseFile -> graph.Build (the
// first two stages of runWorkflowCLI's real dispatch) succeed end to end
// against the real example file WU8 shipped, producing the same
// build -> {unit, vuln} -> publish wave shape WU8's own end-to-end engine
// test already proved executes correctly.
func TestLoadWorkflowManifest_DiamondExample(t *testing.T) {
	m, g, err := loadWorkflowManifest(diamondManifestPath)
	require.NoError(t, err)

	assert.Equal(t, "go-service-release", m.Metadata.Name)
	assert.Len(t, m.Spec.Steps, 4)
	assert.Equal(t, [][]string{{"build"}, {"unit", "vuln"}, {"publish"}}, g.Waves)
}

// TestResolveStepInfos_DiamondExample proves resolveStepInfos (the
// production function listWorkflowSteps calls) reports every diamond
// example step's id, capability, and resolved provider name/version.
func TestResolveStepInfos_DiamondExample(t *testing.T) {
	m, _, err := loadWorkflowManifest(diamondManifestPath)
	require.NoError(t, err)

	reg := providersRegistryForTest(t)
	infos, err := resolveStepInfos(m.Spec.Steps, reg)
	require.NoError(t, err)

	require.Len(t, infos, 4)
	assert.Equal(t, workflowStepInfo{StepID: "build", Capability: "build", ProviderName: "go", ProviderVersion: "1"}, infos[0])
	assert.Equal(t, workflowStepInfo{StepID: "unit", Capability: "test", ProviderName: "go-test", ProviderVersion: "1"}, infos[1])
	assert.Equal(t, workflowStepInfo{StepID: "vuln", Capability: "test", ProviderName: "govulncheck", ProviderVersion: "1"}, infos[2])
	assert.Equal(t, workflowStepInfo{StepID: "publish", Capability: "artifact", ProviderName: "container", ProviderVersion: "1"}, infos[3])
}

// TestResolveCapabilityRef_AllSevenCapabilities covers every one of
// resolveCapabilityRef's seven capability branches (runtime-toolchain-upgrade,
// design.md D-4b/D-8 adds runtime-inspect as the sixth, D-9 adds
// runtime-upgrade as the seventh) — the diamond example only exercises
// build/test/artifact, so deploy/run need their own registered fixtures (no
// in-repo Deployer/Runner provider exists yet, WU3/WU7 — see
// providers/register.go's own doc comment — so this test registers a bare
// fake for each, exactly to prove the dispatch branch itself, not any
// concrete provider). runtime-inspect/runtime-upgrade both use the real
// go-runtime provider RegisterDefaults already registers
// (providersRegistryForTest), same as build/test/artifact above.
func TestResolveCapabilityRef_AllSevenCapabilities(t *testing.T) {
	reg := providersRegistryForTest(t)
	reg.RegisterDeployer(providers.Ref{Name: "fake-deploy", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Deployer {
		return fakeDeployer{}
	})
	reg.RegisterRunner(providers.Ref{Name: "fake-run", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Runner {
		return fakeRunner{}
	})

	tests := []struct {
		capability string
		ref        providers.Ref
	}{
		{"build", providers.Ref{Name: "go", Version: "1"}},
		{"test", providers.Ref{Name: "go-test", Version: "1"}},
		{"artifact", providers.Ref{Name: "container", Version: "1"}},
		{"deploy", providers.Ref{Name: "fake-deploy", Version: "1"}},
		{"run", providers.Ref{Name: "fake-run", Version: "1"}},
		{"runtime-inspect", providers.Ref{Name: "go-runtime", Version: "1"}},
		{"runtime-upgrade", providers.Ref{Name: "go-runtime", Version: "1"}},
	}

	for _, tt := range tests {
		t.Run(tt.capability, func(t *testing.T) {
			err := resolveCapabilityRef(reg, tt.capability, tt.ref)
			require.NoError(t, err)
		})
	}
}

// TestResolveCapabilityRef_UnknownCapabilityFailsClosed covers
// resolveCapabilityRef's defensive default branch: manifest.
// ValidateStructure (stage 3, already run by manifest.ParseFile) rejects
// any capability outside the five known values before resolveCapabilityRef
// would ever see one through the normal parse -> list pipeline, but this
// function does not assume that already happened (mirrors
// internal/workflow/engine's dispatch's own defensive UnknownCapabilityError
// case).
func TestResolveCapabilityRef_UnknownCapabilityFailsClosed(t *testing.T) {
	reg := providersRegistryForTest(t)

	err := resolveCapabilityRef(reg, "not-a-real-capability", providers.Ref{Name: "go", Version: "1"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not-a-real-capability")
}

// --- resolveWorkflowSource tests (source-repo-ref-support) ---

// cloneCall captures the arguments passed to a mock cloneFn.
type cloneCall struct {
	opts     shared.GitCloneOpts
	protocol string
}

// mockCloneFunc returns a cloneRepoFunc that records every call and
// returns (nil, nil) — no caller needs a non-nil directory or error result.
// No Dagger client or network access is needed.
func mockCloneFunc(out *[]cloneCall) cloneRepoFunc {
	return func(_ context.Context, _ *dagger.Client, opts shared.GitCloneOpts, protocol string) (*dagger.Directory, error) {
		if out != nil {
			*out = append(*out, cloneCall{opts: opts, protocol: protocol})
		}
		return nil, nil
	}
}

// TestResolveWorkflowSource_HTTPSProtocolSelectsHTTPS proves that a repo
// URL not starting with "git@" or "ssh://" selects protocol "https".
func TestResolveWorkflowSource_HTTPSProtocolSelectsHTTPS(t *testing.T) {
	var calls []cloneCall
	cloneFn := mockCloneFunc(&calls)

	spec := manifest.SourceSpec{
		Repo: "https://github.com/org/repo.git",
		Ref:  "main",
	}

	_, err := resolveWorkflowSource(context.Background(), nil, spec, cloneFn)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, "https", calls[0].protocol,
		"non-SSH URL must select HTTPS protocol")
}

// TestResolveWorkflowSource_SSHProtocolSelectsSSH proves that a repo URL
// starting with "git@" selects protocol "ssh".
func TestResolveWorkflowSource_SSHProtocolSelectsSSH(t *testing.T) {
	var calls []cloneCall
	cloneFn := mockCloneFunc(&calls)

	spec := manifest.SourceSpec{
		Repo: "git@github.com:org/repo.git",
		Ref:  "main",
	}

	_, err := resolveWorkflowSource(context.Background(), nil, spec, cloneFn)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, "ssh", calls[0].protocol,
		"git@ URL must select SSH protocol")
}

// TestResolveWorkflowSource_SSHSchemeSelectsSSH proves that a repo URL
// using the ssh:// scheme selects protocol "ssh", not "https".
func TestResolveWorkflowSource_SSHSchemeSelectsSSH(t *testing.T) {
	var calls []cloneCall
	cloneFn := mockCloneFunc(&calls)

	spec := manifest.SourceSpec{
		Repo: "ssh://git@github.com/org/repo.git",
		Ref:  "main",
	}

	_, err := resolveWorkflowSource(context.Background(), nil, spec, cloneFn)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, "ssh", calls[0].protocol,
		"ssh:// URL must select SSH protocol")
}

// TestResolveWorkflowSource_ExplicitRefPreserved proves that when spec.Ref
// is set, it is forwarded as opts.Branch unchanged.
func TestResolveWorkflowSource_ExplicitRefPreserved(t *testing.T) {
	var calls []cloneCall
	cloneFn := mockCloneFunc(&calls)

	spec := manifest.SourceSpec{
		Repo: "https://github.com/org/repo.git",
		Ref:  "develop",
	}

	_, err := resolveWorkflowSource(context.Background(), nil, spec, cloneFn)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, "develop", calls[0].opts.Branch,
		"explicit ref must be forwarded as Branch")
}

// TestResolveWorkflowSource_EmptyRefFailsClosed proves that an empty
// spec.Ref is rejected with an explicit error when source.repo is set.
// A remote workflow source requires an explicit pinned ref.
func TestResolveWorkflowSource_EmptyRefFailsClosed(t *testing.T) {
	spec := manifest.SourceSpec{
		Repo: "https://github.com/org/repo.git",
		Ref:  "",
	}

	_, err := resolveWorkflowSource(context.Background(), nil, spec, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source.ref is required")
}

// TestResolveWorkflowSource_RepoForwarded proves the repo URL is forwarded
// as opts.Repo unchanged.
func TestResolveWorkflowSource_RepoForwarded(t *testing.T) {
	var calls []cloneCall
	cloneFn := mockCloneFunc(&calls)

	spec := manifest.SourceSpec{
		Repo: "git@github.com:org/repo.git",
		Ref:  "v1.0.0",
	}

	_, err := resolveWorkflowSource(context.Background(), nil, spec, cloneFn)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, "git@github.com:org/repo.git", calls[0].opts.Repo,
		"repo URL must be forwarded unchanged")
}

// TestResolveWorkflowSource_AuthSecretRefFailsClosed proves that a
// non-empty authSecretRef is rejected with an explicit error, not silently
// ignored. The cloner currently uses global credentials; letting
// authSecretRef through would be misleading.
func TestResolveWorkflowSource_AuthSecretRefFailsClosed(t *testing.T) {
	spec := manifest.SourceSpec{
		Repo:          "https://github.com/org/repo.git",
		Ref:           "main",
		AuthSecretRef: "github-prod",
	}

	_, err := resolveWorkflowSource(context.Background(), nil, spec, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authSecretRef is not supported yet")
}

// TestResolveWorkflowSource_PathFallback proves the existing path-based
// code path is unchanged: when spec.Repo is empty, the function returns
// client.Host().Directory(path) with no clone attempt.
func TestResolveWorkflowSource_PathFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skip Dagger integration in short mode")
	}

	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(io.Discard))
	require.NoError(t, err)
	defer client.Close()

	spec := manifest.SourceSpec{
		Repo: "",
		Path: ".",
	}

	dir, err := resolveWorkflowSource(ctx, client, spec, nil)
	require.NoError(t, err)
	assert.NotNil(t, dir, "path fallback must return a non-nil Directory")
}

// TestCLI_parseFlags_PresetFlagsRemoved is task 11.2's RED test (design.md
// D-N, tasks.md 11.2): --pipeline, --list-pipelines, --only-build,
// --only-test, and --skip-push must be absent from main.go's flag set after
// this work unit — flag.FlagSet.Parse rejects each as an unknown flag. It
// fails while any of the five is still registered, and passes once all are
// removed (tasks.md 11.4).
func TestCLI_parseFlags_PresetFlagsRemoved(t *testing.T) {
	removedFlags := []string{
		"-pipeline=go-service", "-list-pipelines", "-only-build", "-only-test", "-skip-push",
	}

	for _, flagArg := range removedFlags {
		t.Run(flagArg, func(t *testing.T) {
			cli := NewCLI()
			flags, err := cli.parseFlags([]string{flagArg})

			require.Error(t, err, "%s must no longer be a registered flag (design.md D-N)", flagArg)
			assert.Nil(t, flags)
		})
	}
}
