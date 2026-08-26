// Package engine_test exercises the wave-scheduling execution engine
// (design.md D-K, workflow-execution spec): deterministic wave order
// (tasks.md 8.1), fail-fast (8.2), per-step timeout (8.3), bounded per-step
// retry (8.4), maxParallel recorded-not-widened (8.5), approvals never
// blocking (8.6, absence-of-behavior), and structured `when` predicate
// matching (8.7). Every test uses hand-rolled fake capability
// implementations (fakes_test.go) — never a real Dagger container or
// engine connection, per design.md's own testing strategy table ("Fake
// capability implementations recording invocation order").
package engine_test

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/workflow/engine"
	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
	"github.com/pablogore/shipwright/internal/workflow/providers"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// diamondSteps returns a build -> {vuln, unit} -> publish diamond fan-in
// fixture (tasks.md 8.9's own shape), DELIBERATELY declared in a
// non-alphabetical order (vuln before unit, and interleaved with build) so
// a passing order-assertion actually proves Execute walks cfg.Graph.Waves
// (which mirrors manifest declaration order per internal/workflow/graph's
// kahn()), never any map iteration order.
func diamondSteps() []manifest.Step {
	return []manifest.Step{
		{
			ID: "vuln", Capability: "test",
			Uses:  manifest.UsesSpec{Provider: "govulncheck", Version: "1"},
			Needs: []string{"build"}, Input: "${{ steps.build.output }}",
		},
		{
			ID: "build", Capability: "build",
			Uses: manifest.UsesSpec{Provider: "go", Version: "1"},
		},
		{
			ID: "unit", Capability: "test",
			Uses:  manifest.UsesSpec{Provider: "go-test", Version: "1"},
			Needs: []string{"build"}, Input: "${{ steps.build.output }}",
		},
		{
			ID: "publish", Capability: "artifact",
			Uses:  manifest.UsesSpec{Provider: "container", Version: "1"},
			Needs: []string{"build", "unit", "vuln"}, Input: "${{ steps.build.output }}",
			With: map[string]any{"ref": "ghcr.io/acme/api"},
		},
	}
}

// newDiamondRegistry registers fake Builder/Tester(x2)/Artifactor
// providers under the exact names diamondSteps() references, each
// recording its own step id into rec (via the closures passed by the
// caller) in the order the engine invokes them.
func newDiamondRegistry(rec *recorder) *providers.Registry {
	reg := providers.NewRegistry()

	reg.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{BuildFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
			rec.record("build")
			return source, nil
		}}
	})
	reg.RegisterTester(providers.Ref{Name: "govulncheck", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
			rec.record("vuln")
			return nil, nil
		}}
	})
	reg.RegisterTester(providers.Ref{Name: "go-test", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
			rec.record("unit")
			return nil, nil
		}}
	})
	reg.RegisterArtifactor(providers.Ref{Name: "container", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Artifactor {
		return fakeArtifactor{PublishFunc: func(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error) {
			rec.record("publish")
			return ref, nil
		}}
	})

	return reg
}

func buildDiamondConfig(t *testing.T, rec *recorder, opts engine.Options) engine.Config {
	t.Helper()
	steps := diamondSteps()
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build(diamondSteps()) error = %v, want nil", err)
	}
	return engine.Config{
		Steps:    steps,
		Graph:    g,
		Registry: newDiamondRegistry(rec),
		Options:  opts,
	}
}

// tasks.md 8.1: within a single wave, execution follows manifest
// declaration order — vuln is declared before unit, so it must run first,
// even though "unit" < "vuln" alphabetically and neither declaration
// position matches any natural map key order.
func TestExecute_WaveOrderIsManifestDeclarationOrder(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := buildDiamondConfig(t, rec, engine.Options{})

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}

	want := []string{"build", "vuln", "unit", "publish"}
	got := rec.snapshot()
	if len(got) != len(want) {
		t.Fatalf("invocation order = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("invocation order = %v, want %v", got, want)
		}
	}

	for i, o := range res.Outcomes {
		if o.Status != engine.StatusSucceeded {
			t.Fatalf("Outcomes[%d] = %+v, want StatusSucceeded", i, o)
		}
	}
}

// tasks.md 8.5 (scope note): a manifest-declared maxParallel > 1 must NOT
// widen execution — the same fixture with MaxParallel: 4 must produce the
// EXACT same serial, declaration-ordered invocation sequence as the
// default (MaxParallel: 0) case above.
func TestExecute_MaxParallelDoesNotWidenExecution(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := buildDiamondConfig(t, rec, engine.Options{MaxParallel: 4})

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}

	want := []string{"build", "vuln", "unit", "publish"}
	got := rec.snapshot()
	if len(got) != len(want) {
		t.Fatalf("invocation order = %v, want %v (maxParallel must not widen execution)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("invocation order = %v, want %v (maxParallel must not widen execution)", got, want)
		}
	}
}

// tasks.md 8.2: fail-fast stops later waves, names the failing step id,
// and never starts a not-yet-started dependent — it simply never gets
// scheduled (no explicit skip bookkeeping required, per the task's own
// wording).
func TestExecute_FailFastStopsLaterWaves(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	reg := providers.NewRegistry()
	reg.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{BuildFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
			rec.record("build")
			return nil, errors.New("boom: build failed")
		}}
	})
	reg.RegisterTester(providers.Ref{Name: "go-test", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
			rec.record("unit")
			return nil, nil
		}}
	})
	reg.RegisterTester(providers.Ref{Name: "govulncheck", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
			rec.record("vuln")
			return nil, nil
		}}
	})
	reg.RegisterArtifactor(providers.Ref{Name: "container", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Artifactor {
		return fakeArtifactor{PublishFunc: func(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error) {
			rec.record("publish")
			return ref, nil
		}}
	})

	steps := diamondSteps()
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build(diamondSteps()) error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg, Options: engine.Options{FailFast: true}}

	res, err := engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want a *engine.StepFailedError")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}
	if stepFailed.StepID != "build" {
		t.Fatalf("StepFailedError.StepID = %q, want %q", stepFailed.StepID, "build")
	}

	if got := rec.snapshot(); len(got) != 1 || got[0] != "build" {
		t.Fatalf("invocation order = %v, want [build] only (unit/vuln/publish must never start)", got)
	}
	if len(res.Failures) != 1 || res.Failures[0] != "build" {
		t.Fatalf("Result.Failures = %v, want [build]", res.Failures)
	}
}

// tasks.md 8.3: a step exceeding its configured timeout is canceled via
// context.WithTimeout and reported as a timeout failure, never left
// hanging — proven with a fake that genuinely blocks on <-ctx.Done().
func TestExecute_PerStepTimeoutFires(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	reg.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{BuildFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}}
	})

	steps := []manifest.Step{{ID: "build", Capability: "build", Uses: manifest.UsesSpec{Provider: "go", Version: "1"}}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg, Options: engine.Options{Timeout: 20 * time.Millisecond}}

	start := time.Now()
	_, err = engine.Execute(context.Background(), cfg)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Execute() error = nil, want a timeout failure")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}
	var timeoutErr *engine.StepTimeoutError
	if !errors.As(stepFailed.Err, &timeoutErr) {
		t.Fatalf("StepFailedError.Err = %v (%T), want *engine.StepTimeoutError", stepFailed.Err, stepFailed.Err)
	}
	if timeoutErr.StepID != "build" {
		t.Fatalf("StepTimeoutError.StepID = %q, want %q", timeoutErr.StepID, "build")
	}
	// A generous upper bound proves the step was actually canceled near
	// its configured budget, not left hanging until some unrelated
	// external deadline.
	if elapsed > 2*time.Second {
		t.Fatalf("Execute() took %s, want well under 2s (step must be canceled, not left hanging)", elapsed)
	}
}

// tasks.md 8.4: a step's configured retry count is honored — attempted N
// times before being reported as a final failure, and no more than N.
func TestExecute_BoundedRetry_SucceedsWithinBudget(t *testing.T) {
	t.Parallel()

	var calls int32
	reg := providers.NewRegistry()
	reg.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{BuildFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
			n := atomic.AddInt32(&calls, 1)
			if n < 3 {
				return nil, errors.New("transient failure")
			}
			return source, nil
		}}
	})

	steps := []manifest.Step{{ID: "build", Capability: "build", Uses: manifest.UsesSpec{Provider: "go", Version: "1"}}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg, Options: engine.Options{Retries: 3}}

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil (should succeed on 3rd attempt)", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("Build() was called %d times, want exactly 3", got)
	}
	if len(res.Outcomes) != 1 || res.Outcomes[0].Attempts != 3 || res.Outcomes[0].Status != engine.StatusSucceeded {
		t.Fatalf("Outcomes = %+v, want one StatusSucceeded outcome with Attempts=3", res.Outcomes)
	}
}

func TestExecute_BoundedRetry_FailsAfterBudgetExhausted(t *testing.T) {
	t.Parallel()

	var calls int32
	reg := providers.NewRegistry()
	reg.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{BuildFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
			atomic.AddInt32(&calls, 1)
			return nil, errors.New("permanent failure")
		}}
	})

	steps := []manifest.Step{{ID: "build", Capability: "build", Uses: manifest.UsesSpec{Provider: "go", Version: "1"}}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg, Options: engine.Options{Retries: 2}}

	res, err := engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want a final failure after the retry budget is exhausted")
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("Build() was called %d times, want exactly 2 (never unbounded)", got)
	}
	if len(res.Outcomes) != 1 || res.Outcomes[0].Attempts != 2 || res.Outcomes[0].Status != engine.StatusFailed {
		t.Fatalf("Outcomes = %+v, want one StatusFailed outcome with Attempts=2", res.Outcomes)
	}
}

// A Retries value < 1 must still mean exactly one attempt — never zero
// attempts and never unbounded.
func TestExecute_RetriesZeroMeansOneAttempt(t *testing.T) {
	t.Parallel()

	var calls int32
	reg := providers.NewRegistry()
	reg.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{BuildFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
			atomic.AddInt32(&calls, 1)
			return source, nil
		}}
	})
	steps := []manifest.Step{{ID: "build", Capability: "build", Uses: manifest.UsesSpec{Provider: "go", Version: "1"}}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg, Options: engine.Options{}}

	if _, err := engine.Execute(context.Background(), cfg); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("Build() was called %d times, want exactly 1", got)
	}
}

// tasks.md 8.6 (absence-of-behavior): a manifest declaring approvals under
// spec.environments.<name>.approvals never blocks, queues, or gates
// execution — engine.Config does not even expose an Environments field, so
// there is structurally no code path for this test to accidentally
// exercise a blocking implementation through; it proves the engine
// executes to completion using a manifest parsed with approvals declared.
func TestExecute_ApprovalsNeverBlockExecution(t *testing.T) {
	t.Parallel()

	src := []byte(`
apiVersion: shipwright.dev/v1
kind: Workflow
metadata:
  name: approvals-do-not-block
spec:
  source: {path: "."}
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: "1"}
  environments:
    production:
      approvals:
        required: [platform-team]
`)
	m, err := manifest.Parse(bytes.NewReader(src))
	if err != nil {
		t.Fatalf("manifest.Parse() error = %v, want nil", err)
	}
	if len(m.Spec.Environments["production"].Approvals.Required) != 1 {
		t.Fatalf("manifest fixture is wrong: approvals not parsed as expected")
	}

	rec := &recorder{}
	reg := providers.NewRegistry()
	reg.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{BuildFunc: func(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
			rec.record("build")
			return source, nil
		}}
	})

	g, err := graph.Build(m.Spec.Steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: m.Spec.Steps, Graph: g, Registry: reg}

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil — approvals must never block execution", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}
	if got := rec.snapshot(); len(got) != 1 || got[0] != "build" {
		t.Fatalf("invocation order = %v, want [build] — the step ran normally, with no approval wait", got)
	}
}

// tasks.md 8.7: `when` accepts only a structured predicate map, evaluated
// by exact match — a non-matching predicate skips the step, a matching one
// executes it.
func TestExecute_WhenStructuredPredicateMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		predicates map[string]string
		wantRan    bool
	}{
		{name: "matching branch executes", predicates: map[string]string{"branch": "main"}, wantRan: true},
		{name: "non-matching branch is skipped", predicates: map[string]string{"branch": "develop"}, wantRan: false},
		{name: "missing predicate key is skipped", predicates: map[string]string{}, wantRan: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := &recorder{}
			reg := providers.NewRegistry()
			reg.RegisterArtifactor(providers.Ref{Name: "container", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Artifactor {
				return fakeArtifactor{PublishFunc: func(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error) {
					rec.record("publish")
					return ref, nil
				}}
			})

			steps := []manifest.Step{{
				ID: "publish", Capability: "artifact",
				Uses: manifest.UsesSpec{Provider: "container", Version: "1"},
				With: map[string]any{"ref": "ghcr.io/acme/api"},
				When: map[string][]string{"branch": {"main"}},
			}}
			g, err := graph.Build(steps)
			if err != nil {
				t.Fatalf("graph.Build() error = %v, want nil", err)
			}
			cfg := engine.Config{Steps: steps, Graph: g, Registry: reg, Predicates: tt.predicates}

			res, err := engine.Execute(context.Background(), cfg)
			if err != nil {
				t.Fatalf("Execute() error = %v, want nil", err)
			}

			ran := len(rec.snapshot()) == 1
			if ran != tt.wantRan {
				t.Fatalf("step ran = %v, want %v", ran, tt.wantRan)
			}
			wantStatus := engine.StatusSkipped
			if tt.wantRan {
				wantStatus = engine.StatusSucceeded
			}
			if len(res.Outcomes) != 1 || res.Outcomes[0].Status != wantStatus {
				t.Fatalf("Outcomes = %+v, want a single %s outcome", res.Outcomes, wantStatus)
			}
		})
	}
}

// Closing WU7's loop: a with-field value whose resolved Kind mismatches
// the resolved provider's declared WithSchema must fail via the SAME
// *providers.WithSchemaMismatchError WU7 shipped — proving the engine
// actually feeds real interpolated Values into Registry.Resolve*, not a
// bypassed or re-implemented check.
func TestExecute_WithSchemaMismatchPropagatesFromProviderRegistry(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	reg.RegisterBuilder(
		providers.Ref{Name: "go", Version: "1"},
		providers.WithSchema{"goVersion": interp.KindString},
		func(providers.Values) shipwright.Builder { return fakeBuilder{} },
	)

	steps := []manifest.Step{{
		ID: "build", Capability: "build",
		Uses: manifest.UsesSpec{Provider: "go", Version: "1"},
		With: map[string]any{"goVersion": true}, // bool, schema wants string
	}}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	_, err = engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want a with-schema mismatch failure")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}
	var mismatch *providers.WithSchemaMismatchError
	if !errors.As(stepFailed.Err, &mismatch) {
		t.Fatalf("StepFailedError.Err = %v (%T), want *providers.WithSchemaMismatchError", stepFailed.Err, stepFailed.Err)
	}
}

// Closing the WU5/WU6 loop: a step's "input" field referencing another
// step's output whose actual produced kind is not Directory (a Tester
// produces a File, never a Directory) must fail with
// *engine.OutputKindMismatchError — this kind check is genuinely not
// knowable until the referenced step has actually run and its capability
// (hence its real output kind) is known.
func TestExecute_InputReferencingNonDirectoryOutputIsRejected(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "go-test", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{}
	})
	reg.RegisterRunner(providers.Ref{Name: "run", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Runner {
		return fakeRunner{}
	})

	steps := []manifest.Step{
		{ID: "unit", Capability: "test", Uses: manifest.UsesSpec{Provider: "go-test", Version: "1"}},
		{ID: "run", Capability: "run", Uses: manifest.UsesSpec{Provider: "run", Version: "1"}, Needs: []string{"unit"}, Input: "${{ steps.unit.output }}"},
	}
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	cfg := engine.Config{Steps: steps, Graph: g, Registry: reg}

	_, err = engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want an output-kind-mismatch failure")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}
	var kindErr *engine.OutputKindMismatchError
	if !errors.As(stepFailed.Err, &kindErr) {
		t.Fatalf("StepFailedError.Err = %v (%T), want *engine.OutputKindMismatchError", stepFailed.Err, stepFailed.Err)
	}
	if kindErr.ReferencedStepID != "unit" || kindErr.Want != "directory" {
		t.Fatalf("OutputKindMismatchError = %+v, want ReferencedStepID=unit Want=directory", kindErr)
	}
}

// Undeclared variables.*/secrets.* references fail closed with a named
// error rather than silently resolving to an empty value.
func TestExecute_UndeclaredVariableAndSecretRejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		with map[string]any
	}{
		{name: "undeclared variable", with: map[string]any{"ref": "${{ variables.missing }}"}},
		{name: "undeclared secret", with: map[string]any{"ref": "img", "creds": "${{ secrets.missing }}"}},
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
			if err == nil {
				t.Fatal("Execute() error = nil, want an undeclared-reference failure")
			}
		})
	}
}

// OptionsFromSpec is where spec.execution.concurrency.maxParallel is
// "validated and recorded" (tasks.md 8.5) and spec.execution.timeout is
// parsed into a time.Duration.
func TestOptionsFromSpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    manifest.ExecutionSpec
		want    engine.Options
		wantErr bool
	}{
		{
			name: "full spec",
			spec: manifest.ExecutionSpec{
				Concurrency: manifest.ConcurrencySpec{MaxParallel: 4},
				FailFast:    true,
				Timeout:     "30m",
			},
			want: engine.Options{MaxParallel: 4, FailFast: true, Timeout: 30 * time.Minute},
		},
		{
			name: "zero value spec",
			spec: manifest.ExecutionSpec{},
			want: engine.Options{},
		},
		{
			name:    "malformed timeout",
			spec:    manifest.ExecutionSpec{Timeout: "not-a-duration"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := engine.OptionsFromSpec(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("OptionsFromSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("OptionsFromSpec() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
