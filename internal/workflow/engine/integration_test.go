package engine_test

import (
	"context"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/workflow/engine"
	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
	"github.com/pablogore/shipwright/internal/workflow/providers"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// TestEndToEnd_DiamondExampleManifest is the runtime harness the work-unit
// table requires ("-short end-to-end fake-capability run"): parse ->
// graph -> resolve -> interpolate -> execute, driven entirely by
// examples/workflow/diamond.yaml (tasks.md 8.9) and hand-rolled fake
// capabilities — never a real Dagger container or engine connection, which
// is exactly what makes this fast enough to not strictly need a -short
// guard; it is guarded anyway per the harness column's explicit
// instruction.
//
// diamond.yaml declares maxParallel: 4 (its own doc comment example), which
// the engine now genuinely honors: "unit" and "vuln" both belong to wave 2
// and share no needs[] edge, so they dispatch concurrently and their
// relative START order is a real goroutine-scheduling race — this test
// must not assert one over the other. What stays guaranteed, and is what
// this test actually proves, is Result.Outcomes' order (always
// wave-then-declaration order, never completion order — runWave's doc
// comment) and the wave boundary itself: "build" must precede both of them
// and "publish" must follow both, since those are real needs[] edges.
func TestEndToEnd_DiamondExampleManifest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end workflow harness in -short mode")
	}
	t.Parallel()

	m, err := manifest.ParseFile("../../../examples/workflow/diamond.yaml")
	if err != nil {
		t.Fatalf("manifest.ParseFile(diamond.yaml) error = %v, want nil", err)
	}

	g, err := graph.Build(m.Spec.Steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	wantWaves := [][]string{{"build"}, {"unit", "vuln"}, {"publish"}}
	if len(g.Waves) != len(wantWaves) {
		t.Fatalf("graph.Build().Waves = %v, want %v", g.Waves, wantWaves)
	}

	rec := &recorder{}
	reg := providers.NewRegistry()
	reg.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{
		"goVersion": interp.KindString,
	}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{BuildFunc: func(_ context.Context, source *dagger.Directory) (*dagger.Directory, error) {
			rec.record("build")
			return source, nil
		}}
	})
	reg.RegisterTester(providers.Ref{Name: "go-test", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
			rec.record("unit")
			return nil, nil
		}}
	})
	reg.RegisterTester(providers.Ref{Name: "govulncheck", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
			rec.record("vuln")
			return nil, nil
		}}
	})
	reg.RegisterArtifactor(providers.Ref{Name: "container", Version: "1"}, providers.WithSchema{
		"ref":   interp.KindString,
		"creds": interp.KindSecret,
	}, func(providers.Values) shipwright.Artifactor {
		return fakeArtifactor{PublishFunc: func(_ context.Context, _ *dagger.Directory, ref string, _ *dagger.Secret) (string, error) {
			rec.record("publish")
			return ref, nil
		}}
	})

	opts, err := engine.OptionsFromSpec(m.Spec.Execution)
	if err != nil {
		t.Fatalf("engine.OptionsFromSpec() error = %v, want nil", err)
	}

	cfg := engine.Config{
		Steps:      m.Spec.Steps,
		Graph:      g,
		Registry:   reg,
		Variables:  m.Spec.Variables,
		Secrets:    map[string]*dagger.Secret{"registry": nil},
		Predicates: map[string]string{"branch": "main"},
		Options:    opts,
	}

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("engine.Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("engine.Execute() Failures = %v, want none", res.Failures)
	}

	// Invocation order: "build" first, "publish" last, "unit"/"vuln" as a
	// set (concurrent, unordered relative to each other) in between —
	// maxParallel: 4 is genuinely honored for wave 2's two independent
	// steps.
	got := rec.snapshot()
	if len(got) != 4 {
		t.Fatalf("invocation order = %v, want 4 entries", got)
	}
	if got[0] != "build" {
		t.Fatalf("invocation order = %v, want \"build\" first", got)
	}
	if got[3] != "publish" {
		t.Fatalf("invocation order = %v, want \"publish\" last", got)
	}
	middle := map[string]bool{got[1]: true, got[2]: true}
	if !middle["unit"] || !middle["vuln"] {
		t.Fatalf("invocation order = %v, want {\"unit\", \"vuln\"} concurrently in positions 1-2", got)
	}

	// Result.Outcomes stays deterministic regardless of goroutine
	// completion order (runWave's doc comment): wave order, then
	// manifest-declaration order.
	wantOutcomeOrder := []string{"build", "unit", "vuln", "publish"}
	if len(res.Outcomes) != len(wantOutcomeOrder) {
		t.Fatalf("Outcomes = %v, want %d entries", res.Outcomes, len(wantOutcomeOrder))
	}
	for i, wantID := range wantOutcomeOrder {
		if res.Outcomes[i].StepID != wantID {
			t.Fatalf("Outcomes[%d].StepID = %q, want %q (Outcomes order = %v)", i, res.Outcomes[i].StepID, wantID, res.Outcomes)
		}
	}
}
