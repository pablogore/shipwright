// Cycle-detection tests for internal/workflow/graph, exercised entirely
// through the public graph.Build entry point (design.md D-J, tasks.md
// 6.1-6.3, 6.10). Kahn's algorithm was chosen specifically because
// in-degree counting handles diamond fan-in correctly (see
// build_test.go's TestBuild_DiamondFanInAccepted), where naive
// visited-set DFS variants produce false positives on that exact shape.
package graph_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
)

// tasks.md 6.1: a step whose needs[] includes its own id is rejected.
func TestBuild_SelfEdgeRejected(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "a", Needs: []string{"a"}},
	}

	_, err := graph.Build(steps)
	var cycleErr *graph.CycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("Build() error = %v (%T), want *graph.CycleError", err, err)
	}
	if want := `graph: cycle detected among steps [a]`; cycleErr.Error() != want {
		t.Fatalf("CycleError.Error() = %q, want %q", cycleErr.Error(), want)
	}

	assertCycle(t, steps, []string{"a"})
}

// tasks.md 6.2: a mutual pair (a needs b, b needs a) is rejected.
func TestBuild_MutualPairRejected(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "a", Needs: []string{"b"}},
		{ID: "b", Needs: []string{"a"}},
	}

	assertCycle(t, steps, []string{"a", "b"})
}

// tasks.md 6.3: a longer cycle (4+ nodes) is rejected, and every member
// of the cycle is named in the error — not just "a cycle exists
// somewhere".
func TestBuild_LongCycleRejected(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "a", Needs: []string{"d"}},
		{ID: "b", Needs: []string{"a"}},
		{ID: "c", Needs: []string{"b"}},
		{ID: "d", Needs: []string{"c"}},
	}

	assertCycle(t, steps, []string{"a", "b", "c", "d"})
}

// tasks.md 6.10: a cycle that leaves part of the graph unaffected (a
// disconnected, acyclic step alongside a cyclic pair) still names only
// the actual residual in-degree>0 members — proves the cycle error
// enumerates the SPECIFIC offending ids (design.md D-J), not the whole
// graph.
func TestBuild_CycleErrorNamesOnlyResidualNodes(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "standalone"},
		{ID: "a", Needs: []string{"b"}},
		{ID: "b", Needs: []string{"a"}},
	}

	assertCycle(t, steps, []string{"a", "b"})
}

// tasks.md 6.10: a simple linear chain produces one step per wave, in
// dependency order — the sequential-schedule shape design.md D-K's
// execution engine will consume directly.
func TestBuild_LinearChainProducesSequentialWaves(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "c", Needs: []string{"b"}},
		{ID: "b", Needs: []string{"a"}},
		{ID: "a"},
	}

	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("Build() error = %v, want nil", err)
	}

	assertWaves(t, g.Waves, [][]string{{"a"}, {"b"}, {"c"}})
}

func assertCycle(t *testing.T, steps []manifest.Step, wantIDs []string) {
	t.Helper()

	_, err := graph.Build(steps)
	if err == nil {
		t.Fatal("Build() error = nil, want a cycle error")
	}

	var cycleErr *graph.CycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("Build() error = %v (%T), want *graph.CycleError", err, err)
	}

	if len(cycleErr.StepIDs) != len(wantIDs) {
		t.Fatalf("CycleError.StepIDs = %v, want %v", cycleErr.StepIDs, wantIDs)
	}
	got := make(map[string]bool, len(cycleErr.StepIDs))
	for _, id := range cycleErr.StepIDs {
		got[id] = true
	}
	for _, id := range wantIDs {
		if !got[id] {
			t.Fatalf("CycleError.StepIDs = %v, missing %q", cycleErr.StepIDs, id)
		}
	}

	wantMsg := fmt.Sprintf("graph: cycle detected among steps %v", cycleErr.StepIDs)
	if cycleErr.Error() != wantMsg {
		t.Fatalf("CycleError.Error() = %q, want %q", cycleErr.Error(), wantMsg)
	}
}
