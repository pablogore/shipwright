package engine_test

import (
	"context"
	"errors"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/workflow/engine"
	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
	"github.com/pablogore/shipwright/internal/workflow/providers"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// Dispatch must invoke ALL FIVE Layer 1 capability interfaces directly —
// Deployer and Runner have no in-repo WU3/WU7 provider yet, so these are
// the only tests proving the engine's own dispatch path for them.
func TestExecute_DispatchesDeployer(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	reg := providers.NewRegistry()
	reg.RegisterDeployer(providers.Ref{Name: "nomad", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Deployer {
		return fakeDeployer{DeployFunc: func(_ context.Context, artifactRef, environment string, _ *dagger.Secret) (string, error) {
			rec.record("deploy")
			return artifactRef + "@" + environment, nil
		}}
	})

	steps := []manifest.Step{{
		ID: "deploy", Capability: "deploy",
		Uses: manifest.UsesSpec{Provider: "nomad", Version: "1"},
		With: map[string]any{"artifactRef": "ghcr.io/acme/api:1", "environment": "production"},
	}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}
	if got := rec.snapshot(); len(got) != 1 || got[0] != "deploy" {
		t.Fatalf("invocation order = %v, want [deploy]", got)
	}
}

func TestExecute_DispatchesRunner(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	reg := providers.NewRegistry()
	reg.RegisterRunner(providers.Ref{Name: "local", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Runner {
		return fakeRunner{RunFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.Container, error) {
			rec.record("run")
			return nil, nil
		}}
	})

	steps := []manifest.Step{{ID: "run", Capability: "run", Uses: manifest.UsesSpec{Provider: "local", Version: "1"}}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}
	if got := rec.snapshot(); len(got) != 1 || got[0] != "run" {
		t.Fatalf("invocation order = %v, want [run]", got)
	}
}

// TestExecute_DispatchesRuntimeInspector guards dispatchRuntimeInspect's
// straight-line resolve->call->wrap shape (runtime-toolchain-upgrade,
// design.md D-4b, tasks.md 1.19/1.20): no blocking code, output.kind is
// outputText (mirroring Artifactor/Deployer's own string result), and the
// step's produced report string is exactly what the resolved
// RuntimeInspector.Inspect returned.
func TestExecute_DispatchesRuntimeInspector(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	reg := providers.NewRegistry()
	reg.RegisterRuntimeInspector(providers.Ref{Name: "go-runtime", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.RuntimeInspector {
		return fakeRuntimeInspector{InspectFunc: func(ctx context.Context, source *dagger.Directory) (string, error) {
			rec.record("inspect")
			return `{"workspaceRoot":"."}`, nil
		}}
	})

	steps := []manifest.Step{{ID: "inspect", Capability: "runtime-inspect", Uses: manifest.UsesSpec{Provider: "go-runtime", Version: "1"}}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}
	if got := rec.snapshot(); len(got) != 1 || got[0] != "inspect" {
		t.Fatalf("invocation order = %v, want [inspect]", got)
	}
}

// TestExecute_DispatchesRuntimeUpgrader guards dispatchRuntimeUpgrade's
// straight-line resolve->call->wrap shape (runtime-toolchain-upgrade,
// design.md D-9): no blocking code, the required targetVersion with-field
// is extracted via stringWith and forwarded to Upgrade as its own method
// parameter (not baked into the provider factory, unlike
// workspaceRoot/tidy/allowDowngrade), and a missing targetVersion fails the
// step instead of silently defaulting.
func TestExecute_DispatchesRuntimeUpgrader(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	reg := providers.NewRegistry()
	var gotTargetVersion string
	reg.RegisterRuntimeUpgrader(providers.Ref{Name: "go-runtime", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.RuntimeUpgrader {
		return fakeRuntimeUpgrader{UpgradeFunc: func(_ context.Context, source *dagger.Directory, targetVersion string) (*dagger.Directory, error) {
			rec.record("upgrade")
			gotTargetVersion = targetVersion
			return source, nil
		}}
	})

	steps := []manifest.Step{{
		ID: "upgrade", Capability: "runtime-upgrade",
		Uses: manifest.UsesSpec{Provider: "go-runtime", Version: "1"},
		With: map[string]any{"targetVersion": "1.27.0"},
	}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}
	if got := rec.snapshot(); len(got) != 1 || got[0] != "upgrade" {
		t.Fatalf("invocation order = %v, want [upgrade]", got)
	}
	if gotTargetVersion != "1.27.0" {
		t.Fatalf("Upgrade() targetVersion = %q, want %q", gotTargetVersion, "1.27.0")
	}
}

// TestExecute_RuntimeUpgraderMissingTargetVersionFieldRejected proves
// targetVersion's requiredness (design.md D-9's with-schema table) is
// actually enforced at dispatch time, not merely documented: a
// runtime-upgrade step with no targetVersion with-field fails the step
// rather than calling Upgrade with an empty string. Mirrors
// TestExecute_ArtifactorMissingRefFieldRejected's exact assertion shape.
func TestExecute_RuntimeUpgraderMissingTargetVersionFieldRejected(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	reg.RegisterRuntimeUpgrader(providers.Ref{Name: "go-runtime", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.RuntimeUpgrader {
		return fakeRuntimeUpgrader{}
	})
	steps := []manifest.Step{{ID: "upgrade", Capability: "runtime-upgrade", Uses: manifest.UsesSpec{Provider: "go-runtime", Version: "1"}}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	_, err = engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want *engine.MissingWithFieldError")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}
	var missing *engine.MissingWithFieldError
	if !errors.As(stepFailed.Err, &missing) {
		t.Fatalf("StepFailedError.Err = %v (%T), want *engine.MissingWithFieldError", stepFailed.Err, stepFailed.Err)
	}
	if missing.Field != "targetVersion" {
		t.Fatalf("MissingWithFieldError.Field = %q, want %q", missing.Field, "targetVersion")
	}
}

func TestExecute_UnknownCapabilityRejected(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	steps := []manifest.Step{{ID: "mystery", Capability: "teleport", Uses: manifest.UsesSpec{Provider: "x", Version: "1"}}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	_, err = engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want *engine.UnknownCapabilityError")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}
	var unknownCap *engine.UnknownCapabilityError
	if !errors.As(stepFailed.Err, &unknownCap) {
		t.Fatalf("StepFailedError.Err = %v (%T), want *engine.UnknownCapabilityError", stepFailed.Err, stepFailed.Err)
	}
}

func TestExecute_ArtifactorMissingRefFieldRejected(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	reg.RegisterArtifactor(providers.Ref{Name: "container", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Artifactor {
		return fakeArtifactor{}
	})
	steps := []manifest.Step{{ID: "publish", Capability: "artifact", Uses: manifest.UsesSpec{Provider: "container", Version: "1"}}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	_, err = engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want *engine.MissingWithFieldError")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}
	var missing *engine.MissingWithFieldError
	if !errors.As(stepFailed.Err, &missing) {
		t.Fatalf("StepFailedError.Err = %v (%T), want *engine.MissingWithFieldError", stepFailed.Err, stepFailed.Err)
	}
	if missing.Field != "ref" {
		t.Fatalf("MissingWithFieldError.Field = %q, want %q", missing.Field, "ref")
	}
}

func TestExecute_DeployerMissingFieldsRejected(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	reg.RegisterDeployer(providers.Ref{Name: "nomad", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Deployer {
		return fakeDeployer{}
	})
	steps := []manifest.Step{{
		ID: "deploy", Capability: "deploy",
		Uses: manifest.UsesSpec{Provider: "nomad", Version: "1"},
		With: map[string]any{"artifactRef": "ref"}, // environment missing
	}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	_, err = engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want *engine.MissingWithFieldError")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}
	var missing *engine.MissingWithFieldError
	if !errors.As(stepFailed.Err, &missing) {
		t.Fatalf("StepFailedError.Err = %v (%T), want *engine.MissingWithFieldError", stepFailed.Err, stepFailed.Err)
	}
	if missing.Field != "environment" {
		t.Fatalf("MissingWithFieldError.Field = %q, want %q", missing.Field, "environment")
	}
}

func TestExecute_TesterAndArtifactorPropagateCapabilityErrors(t *testing.T) {
	t.Parallel()

	t.Run("tester error", func(t *testing.T) {
		t.Parallel()
		reg := providers.NewRegistry()
		reg.RegisterTester(providers.Ref{Name: "go-test", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
			return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
				return nil, errors.New("tests failed")
			}}
		})
		steps := []manifest.Step{{ID: "unit", Capability: "test", Uses: manifest.UsesSpec{Provider: "go-test", Version: "1"}}}
		g, err := graph.Build(steps)
		if err != nil {
			t.Fatalf("graph.Build() error = %v, want nil", err)
		}
		cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

		_, err = engine.Execute(context.Background(), cfg)
		if err == nil {
			t.Fatal("Execute() error = nil, want the tester's own error to propagate")
		}
	})

	t.Run("artifactor error", func(t *testing.T) {
		t.Parallel()
		reg := providers.NewRegistry()
		reg.RegisterArtifactor(providers.Ref{Name: "container", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Artifactor {
			return fakeArtifactor{PublishFunc: func(_ context.Context, _ *dagger.Directory, _ string, _ *dagger.Secret) (string, error) {
				return "", errors.New("publish failed")
			}}
		})
		steps := []manifest.Step{{
			ID: "publish", Capability: "artifact",
			Uses: manifest.UsesSpec{Provider: "container", Version: "1"},
			With: map[string]any{"ref": "img"},
		}}
		g, err := graph.Build(steps)
		if err != nil {
			t.Fatalf("graph.Build() error = %v, want nil", err)
		}
		cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

		_, err = engine.Execute(context.Background(), cfg)
		if err == nil {
			t.Fatal("Execute() error = nil, want the artifactor's own error to propagate")
		}
	})
}

// resolveInput's own error paths: an "input" field that is not exactly
// one steps.<id>.output reference, and one that references a step whose
// output is unavailable because it was skipped (a non-matching "when").
func TestExecute_InputFieldErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("literal text is not a valid input reference", func(t *testing.T) {
		t.Parallel()
		reg := providers.NewRegistry()
		reg.RegisterRunner(providers.Ref{Name: "local", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Runner {
			return fakeRunner{}
		})
		steps := []manifest.Step{{ID: "run", Capability: "run", Uses: manifest.UsesSpec{Provider: "local", Version: "1"}, Input: "not-a-reference"}}
		g, err := graph.Build(steps)
		if err != nil {
			t.Fatalf("graph.Build() error = %v, want nil", err)
		}
		cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

		_, err = engine.Execute(context.Background(), cfg)
		if err == nil {
			t.Fatal("Execute() error = nil, want *engine.InvalidInputReferenceError")
		}
		var stepFailed *engine.StepFailedError
		if !errors.As(err, &stepFailed) {
			t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
		}
		var invalid *engine.InvalidInputReferenceError
		if !errors.As(stepFailed.Err, &invalid) {
			t.Fatalf("StepFailedError.Err = %v (%T), want *engine.InvalidInputReferenceError", stepFailed.Err, stepFailed.Err)
		}
	})

	t.Run("input referencing a skipped step's output is rejected", func(t *testing.T) {
		t.Parallel()
		reg := providers.NewRegistry()
		reg.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
			return fakeBuilder{}
		})
		reg.RegisterRunner(providers.Ref{Name: "local", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Runner {
			return fakeRunner{}
		})
		steps := []manifest.Step{
			{ID: "build", Capability: "build", Uses: manifest.UsesSpec{Provider: "go", Version: "1"}, When: map[string][]string{"branch": {"never-matches"}}},
			{ID: "run", Capability: "run", Uses: manifest.UsesSpec{Provider: "local", Version: "1"}, Needs: []string{"build"}, Input: "${{ steps.build.output }}"},
		}
		g, err := graph.Build(steps)
		if err != nil {
			t.Fatalf("graph.Build() error = %v, want nil", err)
		}
		cfg := engine.Config{Steps: steps, Graph: g, Registry: reg, Predicates: map[string]string{"branch": "main"}}

		_, err = engine.Execute(context.Background(), cfg)
		if err == nil {
			t.Fatal("Execute() error = nil, want *engine.MissingStepOutputError")
		}
		var stepFailed *engine.StepFailedError
		if !errors.As(err, &stepFailed) {
			t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
		}
		var missingOut *engine.MissingStepOutputError
		if !errors.As(stepFailed.Err, &missingOut) {
			t.Fatalf("StepFailedError.Err = %v (%T), want *engine.MissingStepOutputError", stepFailed.Err, stepFailed.Err)
		}
	})
}

// bindWithValue must handle every yaml.v3-decoded scalar shape an
// untyped "with" value can take, and reject anything else.
func TestExecute_WithValueTypeCoercion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		with    map[string]any
		wantErr bool
	}{
		{name: "bool", with: map[string]any{"ref": "img", "coverage": true}},
		{name: "int", with: map[string]any{"ref": "img", "coverage": 42}},
		{name: "int64", with: map[string]any{"ref": "img", "coverage": int64(42)}},
		{name: "float64", with: map[string]any{"ref": "img", "coverage": float64(42.5)}},
		{name: "unsupported type", with: map[string]any{"ref": "img", "coverage": []string{"nope"}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := providers.NewRegistry()
			reg.RegisterArtifactor(providers.Ref{Name: "container", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Artifactor {
				return fakeArtifactor{}
			})
			steps := []manifest.Step{{ID: "publish", Capability: "artifact", Uses: manifest.UsesSpec{Provider: "container", Version: "1"}, With: tt.with}}
			g, err := graph.Build(steps)
			if err != nil {
				t.Fatalf("graph.Build() error = %v, want nil", err)
			}
			cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

			_, err = engine.Execute(context.Background(), cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// A "with" field referencing a step's non-text output (e.g. a Directory)
// is a kind mismatch, not a missing-output error.
func TestExecute_WithFieldReferencingNonTextOutputIsRejected(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	reg.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{}
	})
	reg.RegisterArtifactor(providers.Ref{Name: "container", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Artifactor {
		return fakeArtifactor{}
	})
	steps := []manifest.Step{
		{ID: "build", Capability: "build", Uses: manifest.UsesSpec{Provider: "go", Version: "1"}},
		{
			ID: "publish", Capability: "artifact",
			Uses:  manifest.UsesSpec{Provider: "container", Version: "1"},
			Needs: []string{"build"}, With: map[string]any{"ref": "${{ steps.build.output }}"},
		},
	}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	_, err = engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want *engine.OutputKindMismatchError")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}
	var kindErr *engine.OutputKindMismatchError
	if !errors.As(stepFailed.Err, &kindErr) {
		t.Fatalf("StepFailedError.Err = %v (%T), want *engine.OutputKindMismatchError", stepFailed.Err, stepFailed.Err)
	}
}

// Execute defensively rejects a Graph whose waves reference a step id not
// present in cfg.Steps, mirroring internal/workflow/graph's own "does not
// assume an earlier stage already ran" discipline.
func TestExecute_GraphReferencingUnknownStepIsRejected(t *testing.T) {
	t.Parallel()

	g := &graph.Graph{
		Nodes: map[string]graph.Node{"ghost": {ID: "ghost"}},
		Waves: [][]string{{"ghost"}},
	}
	cfg := engine.Config{Steps: nil, Graph: g, Registry: providers.NewRegistry()}

	_, err := engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want an error naming the unknown step")
	}
}
