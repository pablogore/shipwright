package engine

import (
	"fmt"

	"github.com/pablogore/shipwright/internal/workflow/graph"
)

// UnknownStepError reports a Closure request naming a step id absent from
// g.Nodes.
type UnknownStepError struct{ StepID string }

func (e *UnknownStepError) Error() string {
	return fmt.Sprintf("engine: unknown step %q", e.StepID)
}

// Closure computes the needs-transitive closure of stepID over g: a new
// *graph.Graph containing stepID and every step it depends on, directly or
// transitively, with Waves filtered to preserve g's own topological order
// and per-wave manifest-declaration order (design.md D-N: "--step <id>
// retargeted to a manifest step id — executes the needs-transitive closure
// of <id> in topological order and stops — a subgraph run, computed by
// reachability over the graph already built").
//
// This function exposes ONLY the underlying reachability computation.
// Wiring it to the CLI's --step flag is Phase 9's job (design.md D-N); this
// package deliberately stops at the computed *graph.Graph so Phase 9 can
// pass it straight to Execute unchanged.
//
// Closure never re-runs cycle detection: g is assumed to already be the
// product of graph.Build, which never returns a Graph containing a cycle
// (an acyclic input has no cycle to reintroduce by filtering a subset of
// its own edges), so the reachability walk below is unconditionally
// termination-safe.
func Closure(g *graph.Graph, stepID string) (*graph.Graph, error) {
	if _, ok := g.Nodes[stepID]; !ok {
		return nil, &UnknownStepError{StepID: stepID}
	}

	reachable := make(map[string]bool, len(g.Nodes))
	visitReachable(g, stepID, reachable)

	nodes := make(map[string]graph.Node, len(reachable))
	for id := range reachable {
		nodes[id] = g.Nodes[id]
	}

	var waves [][]string
	for _, wave := range g.Waves {
		filtered := filterReachable(wave, reachable)
		if len(filtered) > 0 {
			waves = append(waves, filtered)
		}
	}

	return &graph.Graph{Nodes: nodes, Waves: waves}, nil
}

// visitReachable marks id and every step it (transitively) needs as
// reachable, via a plain recursive walk over g.Nodes[*].Needs.
func visitReachable(g *graph.Graph, id string, reachable map[string]bool) {
	if reachable[id] {
		return
	}
	reachable[id] = true
	for _, dep := range g.Nodes[id].Needs {
		visitReachable(g, dep, reachable)
	}
}

// filterReachable returns the subset of wave present in reachable,
// preserving wave's own (manifest-declaration) order.
func filterReachable(wave []string, reachable map[string]bool) []string {
	var out []string
	for _, id := range wave {
		if reachable[id] {
			out = append(out, id)
		}
	}
	return out
}
