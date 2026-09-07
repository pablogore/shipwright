// Package engine_test — real concurrency coverage for runWave (design.md
// D-K's "worker pool drops into later" seam). Every scenario the change's
// spec required is here: genuine concurrent execution, MaxParallel as a hard
// upper bound, needs[] ordering never violated, deterministic Result.Outcomes
// order regardless of goroutine completion order, failFast under real
// concurrency (both true and false), timeout composed with concurrency, and
// per-step retry counted correctly per goroutine. All timing uses
// fake-provider time.Sleep only, never production code — see fakes_test.go.
package engine_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/workflow/engine"
	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
	"github.com/pablogore/shipwright/internal/workflow/providers"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// timeline records each step's [start, end) interval under a mutex, so
// concurrency tests can assert real overlap (proof of genuine concurrency)
// or its absence (proof a dependency/semaphore serialized two steps).
type timeline struct {
	mu        sync.Mutex
	intervals map[string][2]time.Time
}

func newTimeline() *timeline {
	return &timeline{intervals: make(map[string][2]time.Time)}
}

// track marks id's start now and returns a func to call at id's end.
func (tl *timeline) track(id string) func() {
	start := time.Now()
	return func() {
		end := time.Now()
		tl.mu.Lock()
		defer tl.mu.Unlock()
		tl.intervals[id] = [2]time.Time{start, end}
	}
}

func (tl *timeline) get(id string) (start, end time.Time) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	iv := tl.intervals[id]
	return iv[0], iv[1]
}

// overlaps reports whether a and b's recorded intervals intersect.
func (tl *timeline) overlaps(a, b string) bool {
	aStart, aEnd := tl.get(a)
	bStart, bEnd := tl.get(b)
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

// independentTestStep builds a "test" capability step with no needs, using
// id as both the step id and the registered provider name — independent
// same-wave steps are the structural precondition runWave's own doc comment
// relies on for lock-free outputs access.
func independentTestStep(id string) manifest.Step {
	return manifest.Step{ID: id, Capability: "test", Uses: manifest.UsesSpec{Provider: id, Version: "1"}}
}

// registerSleepyTester registers a Tester under Ref{Name: id} that sleeps d
// before succeeding, tracking its own [start,end) interval in tl.
func registerSleepyTester(reg *providers.Registry, tl *timeline, id string, d time.Duration) {
	reg.RegisterTester(providers.Ref{Name: id, Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
			done := tl.track(id)
			defer done()
			time.Sleep(d)
			return nil, nil
		}}
	})
}

func buildTestConfig(t *testing.T, steps []manifest.Step, reg *providers.Registry, opts engine.Options) engine.Config {
	t.Helper()
	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}
	return engine.Config{Steps: steps, Graph: g, Registry: reg, Options: opts}
}

// (1) Three independent same-wave steps genuinely overlap in wall-clock
// time, and the whole wave finishes far under their sequential sum.
func TestConcurrency_ThreeIndependentStepsRunConcurrently(t *testing.T) {
	t.Parallel()

	tl := newTimeline()
	reg := providers.NewRegistry()
	ids := []string{"a", "b", "c"}
	const sleep = 150 * time.Millisecond
	for _, id := range ids {
		registerSleepyTester(reg, tl, id, sleep)
	}
	steps := []manifest.Step{independentTestStep("a"), independentTestStep("b"), independentTestStep("c")}
	cfg := buildTestConfig(t, steps, reg, engine.Options{MaxParallel: 3})

	start := time.Now()
	res, err := engine.Execute(context.Background(), cfg)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}

	if elapsed >= 3*sleep {
		t.Fatalf("Execute() took %s, want well under the sequential sum %s (steps did not run concurrently)", elapsed, 3*sleep)
	}
	for _, pair := range [][2]string{{"a", "b"}, {"a", "c"}, {"b", "c"}} {
		if !tl.overlaps(pair[0], pair[1]) {
			t.Fatalf("steps %q and %q never overlapped, want genuine concurrent execution", pair[0], pair[1])
		}
	}
}

// (2) MaxParallel: 1 preserves strictly sequential execution — no two
// steps' intervals may overlap.
func TestConcurrency_MaxParallelOneIsSequential(t *testing.T) {
	t.Parallel()

	tl := newTimeline()
	reg := providers.NewRegistry()
	ids := []string{"a", "b", "c"}
	const sleep = 20 * time.Millisecond
	for _, id := range ids {
		registerSleepyTester(reg, tl, id, sleep)
	}
	steps := []manifest.Step{independentTestStep("a"), independentTestStep("b"), independentTestStep("c")}
	cfg := buildTestConfig(t, steps, reg, engine.Options{MaxParallel: 1})

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}

	for _, pair := range [][2]string{{"a", "b"}, {"a", "c"}, {"b", "c"}} {
		if tl.overlaps(pair[0], pair[1]) {
			t.Fatalf("steps %q and %q overlapped, want strictly sequential execution under MaxParallel: 1", pair[0], pair[1])
		}
	}
}

// (3) MaxParallel: 2 never has more than 2 steps active at once, across a
// wave of 5 independent steps.
func TestConcurrency_MaxParallelTwoNeverExceedsTwoActive(t *testing.T) {
	t.Parallel()

	var active, peak int32
	reg := providers.NewRegistry()
	ids := []string{"a", "b", "c", "d", "e"}
	steps := make([]manifest.Step, 0, len(ids))
	for _, id := range ids {
		steps = append(steps, independentTestStep(id))
		reg.RegisterTester(providers.Ref{Name: id, Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
			return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
				n := atomic.AddInt32(&active, 1)
				for {
					p := atomic.LoadInt32(&peak)
					if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
						break
					}
				}
				time.Sleep(30 * time.Millisecond)
				atomic.AddInt32(&active, -1)
				return nil, nil
			}}
		})
	}
	cfg := buildTestConfig(t, steps, reg, engine.Options{MaxParallel: 2})

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}
	if got := atomic.LoadInt32(&peak); got != 2 {
		t.Fatalf("peak concurrent active steps = %d, want exactly 2 (MaxParallel must be a hard, saturated bound)", got)
	}
}

// (4) An A -> B dependency never overlaps: B must not start until A's
// interval has closed, even though both run through the same concurrent
// engine.
func TestConcurrency_DependencyNeverOverlaps(t *testing.T) {
	t.Parallel()

	tl := newTimeline()
	reg := providers.NewRegistry()
	reg.RegisterBuilder(providers.Ref{Name: "a", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{BuildFunc: func(_ context.Context, source *dagger.Directory) (*dagger.Directory, error) {
			done := tl.track("a")
			defer done()
			time.Sleep(50 * time.Millisecond)
			return source, nil
		}}
	})
	registerSleepyTester(reg, tl, "b", 20*time.Millisecond)

	steps := []manifest.Step{
		{ID: "a", Capability: "build", Uses: manifest.UsesSpec{Provider: "a", Version: "1"}},
		{ID: "b", Capability: "test", Uses: manifest.UsesSpec{Provider: "b", Version: "1"}, Needs: []string{"a"}, Input: "${{ steps.a.output }}"},
	}
	cfg := buildTestConfig(t, steps, reg, engine.Options{MaxParallel: 4})

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}

	_, aEnd := tl.get("a")
	bStart, _ := tl.get("b")
	if bStart.Before(aEnd) {
		t.Fatalf("step %q started at %s, before its dependency %q ended at %s", "b", bStart, "a", aEnd)
	}
}

// (5) Diamond DAG: a runs alone, b/c run concurrently after a, d runs only
// after both b and c.
func TestConcurrency_DiamondDAG_ConcurrentMiddleWave(t *testing.T) {
	t.Parallel()

	tl := newTimeline()
	reg := providers.NewRegistry()
	reg.RegisterBuilder(providers.Ref{Name: "a", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{BuildFunc: func(_ context.Context, source *dagger.Directory) (*dagger.Directory, error) {
			done := tl.track("a")
			defer done()
			time.Sleep(20 * time.Millisecond)
			return source, nil
		}}
	})
	registerSleepyTester(reg, tl, "b", 80*time.Millisecond)
	registerSleepyTester(reg, tl, "c", 80*time.Millisecond)
	reg.RegisterArtifactor(providers.Ref{Name: "d", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Artifactor {
		return fakeArtifactor{PublishFunc: func(_ context.Context, _ *dagger.Directory, ref string, _ *dagger.Secret) (string, error) {
			done := tl.track("d")
			defer done()
			return ref, nil
		}}
	})

	steps := []manifest.Step{
		{ID: "a", Capability: "build", Uses: manifest.UsesSpec{Provider: "a", Version: "1"}},
		{ID: "b", Capability: "test", Uses: manifest.UsesSpec{Provider: "b", Version: "1"}, Needs: []string{"a"}, Input: "${{ steps.a.output }}"},
		{ID: "c", Capability: "test", Uses: manifest.UsesSpec{Provider: "c", Version: "1"}, Needs: []string{"a"}, Input: "${{ steps.a.output }}"},
		{ID: "d", Capability: "artifact", Uses: manifest.UsesSpec{Provider: "d", Version: "1"}, Needs: []string{"a", "b", "c"}, Input: "${{ steps.a.output }}", With: map[string]any{"ref": "x"}},
	}
	cfg := buildTestConfig(t, steps, reg, engine.Options{MaxParallel: 2})

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}

	if !tl.overlaps("b", "c") {
		t.Fatal("steps \"b\" and \"c\" never overlapped, want concurrent middle wave")
	}
	_, aEnd := tl.get("a")
	bStart, bEnd := tl.get("b")
	cStart, cEnd := tl.get("c")
	dStart, _ := tl.get("d")
	if bStart.Before(aEnd) || cStart.Before(aEnd) {
		t.Fatalf("\"b\"/\"c\" started before \"a\" ended (a ended %s, b started %s, c started %s)", aEnd, bStart, cStart)
	}
	if dStart.Before(bEnd) || dStart.Before(cEnd) {
		t.Fatalf("\"d\" started at %s before both \"b\" (%s) and \"c\" (%s) ended", dStart, bEnd, cEnd)
	}
}

// (6) Result.Outcomes preserves manifest-declaration order even when the
// first-declared step is the LAST to actually finish.
func TestConcurrency_DeclarationOrderPreservedRegardlessOfFinishOrder(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	tl := newTimeline()
	registerSleepyTester(reg, tl, "s1", 150*time.Millisecond)
	registerSleepyTester(reg, tl, "s2", 75*time.Millisecond)
	registerSleepyTester(reg, tl, "s3", 5*time.Millisecond)
	steps := []manifest.Step{independentTestStep("s1"), independentTestStep("s2"), independentTestStep("s3")}
	cfg := buildTestConfig(t, steps, reg, engine.Options{MaxParallel: 3})

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}

	want := []string{"s1", "s2", "s3"}
	if len(res.Outcomes) != len(want) {
		t.Fatalf("Outcomes = %v, want %d entries", res.Outcomes, len(want))
	}
	for i, id := range want {
		if res.Outcomes[i].StepID != id {
			t.Fatalf("Outcomes[%d].StepID = %q, want %q (Outcomes = %v)", i, res.Outcomes[i].StepID, id, res.Outcomes)
		}
	}

	_, s1End := tl.get("s1")
	_, s3End := tl.get("s3")
	if !s3End.Before(s1End) {
		t.Fatalf("test fixture invalid: \"s3\" (5ms) must finish before \"s1\" (150ms) for this test to prove anything")
	}
}

// (7) failFast: false lets independent siblings finish even after one
// step fails.
func TestConcurrency_FailFastFalseLetsSiblingsFinish(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "fail", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
			return nil, errors.New("boom")
		}}
	})
	tl := newTimeline()
	registerSleepyTester(reg, tl, "b", 50*time.Millisecond)
	registerSleepyTester(reg, tl, "c", 50*time.Millisecond)

	steps := []manifest.Step{independentTestStep("fail"), independentTestStep("b"), independentTestStep("c")}
	cfg := buildTestConfig(t, steps, reg, engine.Options{MaxParallel: 3, FailFast: false})

	res, err := engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want a *engine.StepFailedError")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}

	if len(res.Outcomes) != 3 {
		t.Fatalf("Outcomes = %v, want 3 entries (siblings must still finish under failFast: false)", res.Outcomes)
	}
	byID := make(map[string]engine.StepOutcome, 3)
	for _, o := range res.Outcomes {
		byID[o.StepID] = o
	}
	if byID["fail"].Status != engine.StatusFailed {
		t.Fatalf("Outcomes[\"fail\"].Status = %v, want StatusFailed", byID["fail"].Status)
	}
	if byID["b"].Status != engine.StatusSucceeded || byID["c"].Status != engine.StatusSucceeded {
		t.Fatalf("Outcomes[\"b\"/\"c\"] = %+v / %+v, want both StatusSucceeded (independent siblings must finish)", byID["b"], byID["c"])
	}
}

// (8) failFast: true, under real concurrency, prevents any NEW step from
// being dispatched once a failure is detected — an already-dispatched
// sibling still runs to completion (its result is kept), but a step the
// semaphore had not yet admitted must never start at all.
func TestConcurrency_FailFastTruePreventsFurtherScheduling(t *testing.T) {
	t.Parallel()

	var neverCalled atomic.Bool
	neverCalled.Store(true)

	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "fail", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
			return nil, errors.New("boom")
		}}
	})
	tl := newTimeline()
	registerSleepyTester(reg, tl, "b", 200*time.Millisecond)
	reg.RegisterTester(providers.Ref{Name: "c", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
			neverCalled.Store(false)
			return nil, nil
		}}
	})
	reg.RegisterTester(providers.Ref{Name: "d", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
			neverCalled.Store(false)
			return nil, nil
		}}
	})

	// MaxParallel: 2 saturates immediately on "fail"+"b" — "c" and "d" can
	// only be dispatched once a semaphore slot frees, which never happens
	// before failFast's cancellation fires ("fail" does zero work; "b"
	// sleeps 200ms).
	steps := []manifest.Step{independentTestStep("fail"), independentTestStep("b"), independentTestStep("c"), independentTestStep("d")}
	cfg := buildTestConfig(t, steps, reg, engine.Options{MaxParallel: 2, FailFast: true})

	res, err := engine.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("Execute() error = nil, want a *engine.StepFailedError")
	}
	var stepFailed *engine.StepFailedError
	if !errors.As(err, &stepFailed) {
		t.Fatalf("Execute() error = %v (%T), want *engine.StepFailedError", err, err)
	}
	if stepFailed.StepID != "fail" {
		t.Fatalf("StepFailedError.StepID = %q, want %q", stepFailed.StepID, "fail")
	}
	if !neverCalled.Load() {
		t.Fatal("\"c\" or \"d\" was dispatched, want NO new step scheduled once failFast fires")
	}

	byID := make(map[string]engine.StepOutcome, len(res.Outcomes))
	for _, o := range res.Outcomes {
		byID[o.StepID] = o
	}
	if _, ok := byID["c"]; ok {
		t.Fatalf("Outcomes contains %q, want it entirely absent (never dispatched)", "c")
	}
	if _, ok := byID["d"]; ok {
		t.Fatalf("Outcomes contains %q, want it entirely absent (never dispatched)", "d")
	}
	if byID["fail"].Status != engine.StatusFailed {
		t.Fatalf("Outcomes[\"fail\"].Status = %v, want StatusFailed", byID["fail"].Status)
	}
	if byID["b"].Status != engine.StatusSucceeded {
		t.Fatalf("Outcomes[\"b\"].Status = %v, want StatusSucceeded (already-dispatched sibling must finish, not be discarded)", byID["b"].Status)
	}
}

// (9) A concurrent timeout does not hang: every step that exceeds
// Options.Timeout is canceled and reported, never left blocked forever.
func TestConcurrency_TimeoutDoesNotHang(t *testing.T) {
	t.Parallel()

	reg := providers.NewRegistry()
	ids := []string{"t1", "t2", "t3"}
	for _, id := range ids {
		reg.RegisterTester(providers.Ref{Name: id, Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
			return fakeTester{TestFunc: func(ctx context.Context, _ *dagger.Directory) (*dagger.File, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			}}
		})
	}
	steps := []manifest.Step{independentTestStep("t1"), independentTestStep("t2"), independentTestStep("t3")}
	cfg := buildTestConfig(t, steps, reg, engine.Options{MaxParallel: 3, Timeout: 20 * time.Millisecond})

	start := time.Now()
	res, err := engine.Execute(context.Background(), cfg)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Execute() error = nil, want a timeout failure")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Execute() took %s, want well under 2s (no step may be left hanging)", elapsed)
	}
	if len(res.Failures) != 3 {
		t.Fatalf("Failures = %v, want all 3 steps to time out", res.Failures)
	}
	for _, o := range res.Outcomes {
		var timeoutErr *engine.StepTimeoutError
		if !errors.As(o.Err, &timeoutErr) {
			t.Fatalf("Outcomes[%q].Err = %v (%T), want *engine.StepTimeoutError", o.StepID, o.Err, o.Err)
		}
	}
}

// (10) Per-step retry counts are correct per goroutine under concurrent
// execution — no cross-step interference between two independently-retried
// steps.
func TestConcurrency_RetriesCountCorrectlyPerStep(t *testing.T) {
	t.Parallel()

	var callsR1, callsR2 int32
	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "r1", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
			if atomic.AddInt32(&callsR1, 1) < 3 {
				return nil, errors.New("transient")
			}
			return nil, nil
		}}
	})
	reg.RegisterTester(providers.Ref{Name: "r2", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{TestFunc: func(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
			if atomic.AddInt32(&callsR2, 1) < 3 {
				return nil, errors.New("transient")
			}
			return nil, nil
		}}
	})
	steps := []manifest.Step{independentTestStep("r1"), independentTestStep("r2")}
	cfg := buildTestConfig(t, steps, reg, engine.Options{MaxParallel: 2, Retries: 3})

	res, err := engine.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil (both steps should succeed on 3rd attempt)", err)
	}
	if res.Failed() {
		t.Fatalf("Execute() Failures = %v, want none", res.Failures)
	}
	if got := atomic.LoadInt32(&callsR1); got != 3 {
		t.Fatalf("\"r1\" was called %d times, want exactly 3", got)
	}
	if got := atomic.LoadInt32(&callsR2); got != 3 {
		t.Fatalf("\"r2\" was called %d times, want exactly 3", got)
	}
	for _, o := range res.Outcomes {
		if o.Attempts != 3 {
			t.Fatalf("Outcomes[%q].Attempts = %d, want 3", o.StepID, o.Attempts)
		}
	}
}

// TestConcurrency_WallClockDemo is the requested real validation demo:
// three same-wave steps taking ~150ms each run in ~150ms+overhead under
// MaxParallel: 3, versus ~450ms under MaxParallel: 1 — proving the engine,
// not just a unit assertion, actually collapses wall-clock time.
func TestConcurrency_WallClockDemo(t *testing.T) {
	t.Parallel()

	const perStep = 150 * time.Millisecond
	newCfg := func(maxParallel int) engine.Config {
		reg := providers.NewRegistry()
		tl := newTimeline()
		for _, id := range []string{"a", "b", "c"} {
			registerSleepyTester(reg, tl, id, perStep)
		}
		steps := []manifest.Step{independentTestStep("a"), independentTestStep("b"), independentTestStep("c")}
		return buildTestConfig(t, steps, reg, engine.Options{MaxParallel: maxParallel})
	}

	seqStart := time.Now()
	if _, err := engine.Execute(context.Background(), newCfg(1)); err != nil {
		t.Fatalf("Execute() sequential error = %v, want nil", err)
	}
	sequential := time.Since(seqStart)

	concStart := time.Now()
	if _, err := engine.Execute(context.Background(), newCfg(3)); err != nil {
		t.Fatalf("Execute() concurrent error = %v, want nil", err)
	}
	concurrent := time.Since(concStart)

	t.Logf("wall-clock demo: sequential (MaxParallel:1) = %s, concurrent (MaxParallel:3) = %s", sequential, concurrent)

	if sequential < 3*perStep {
		t.Fatalf("sequential run took %s, want at least %s (sum of all 3 steps)", sequential, 3*perStep)
	}
	if concurrent >= sequential {
		t.Fatalf("concurrent run (%s) was not faster than sequential (%s)", concurrent, sequential)
	}
	if concurrent >= 2*perStep {
		t.Fatalf("concurrent run took %s, want well under 2x a single step's %s", concurrent, perStep)
	}
}
