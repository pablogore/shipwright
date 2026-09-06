package engine_test

import (
	"errors"
	"testing"

	"github.com/pablogore/shipwright/internal/workflow/engine"
	"github.com/pablogore/shipwright/internal/workflow/graph"
)

// design.md D-N: Closure computes the needs-transitive closure of a step,
// in topological order, as a subgraph of the graph already built —
// exposed here as the underlying reachability computation Phase 9's
// --step flag will call, not wired to any CLI flag in this work unit.
func TestClosure_NeedsTransitiveClosure(t *testing.T) {
	t.Parallel()

	g, err := graph.Build(diamondSteps())
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}

	tests := []struct {
		name      string
		stepID    string
		wantWaves [][]string
	}{
		{name: "root only", stepID: "build", wantWaves: [][]string{{"build"}}},
		{name: "mid-level excludes sibling and publish", stepID: "unit", wantWaves: [][]string{{"build"}, {"unit"}}},
		{name: "full closure includes everything", stepID: "publish", wantWaves: [][]string{{"build"}, {"vuln", "unit"}, {"publish"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sub, err := engine.Closure(g, tt.stepID)
			if err != nil {
				t.Fatalf("Closure(%q) error = %v, want nil", tt.stepID, err)
			}
			if len(sub.Waves) != len(tt.wantWaves) {
				t.Fatalf("Closure(%q).Waves = %v, want %v", tt.stepID, sub.Waves, tt.wantWaves)
			}
			for i := range tt.wantWaves {
				if len(sub.Waves[i]) != len(tt.wantWaves[i]) {
					t.Fatalf("Closure(%q).Waves = %v, want %v", tt.stepID, sub.Waves, tt.wantWaves)
				}
				for j := range tt.wantWaves[i] {
					if sub.Waves[i][j] != tt.wantWaves[i][j] {
						t.Fatalf("Closure(%q).Waves = %v, want %v", tt.stepID, sub.Waves, tt.wantWaves)
					}
				}
			}
			if _, ok := sub.Nodes[tt.stepID]; !ok {
				t.Fatalf("Closure(%q).Nodes is missing %q itself", tt.stepID, tt.stepID)
			}
		})
	}
}

func TestClosure_UnknownStepRejected(t *testing.T) {
	t.Parallel()

	g, err := graph.Build(diamondSteps())
	if err != nil {
		t.Fatalf("graph.Build() error = %v, want nil", err)
	}

	_, err = engine.Closure(g, "does-not-exist")
	if err == nil {
		t.Fatal("Closure(unknown step) error = nil, want *engine.UnknownStepError")
	}
	var unknown *engine.UnknownStepError
	if !errors.As(err, &unknown) {
		t.Fatalf("Closure(unknown step) error = %v (%T), want *engine.UnknownStepError", err, err)
	}
}
