package graph

import (
	"fmt"

	"github.com/pablogore/shipwright/internal/workflow/manifest"
)

// CycleError reports the set of step ids that remain unresolved after
// Kahn's algorithm drains every zero-in-degree node (tasks.md 6.10,
// design.md D-J): "after Kahn drains, any node with residual in-degree >
// 0 is in or downstream of a cycle." StepIDs is ordered by manifest
// declaration order for a deterministic, reproducible error message.
type CycleError struct{ StepIDs []string }

func (e *CycleError) Error() string {
	return fmt.Sprintf("graph: cycle detected among steps %v", e.StepIDs)
}

// kahn implements Kahn's algorithm (design.md D-J): repeatedly remove
// every currently zero-in-degree node as one "wave", decrementing the
// in-degree of each of its dependents, until no more progress can be
// made. This is the ONLY cycle-detection mechanism in this package —
// design.md rejected DFS three-colour marking specifically because
// naive visited-set DFS variants produce false positives on legitimate
// diamond fan-in (tasks.md 6.4), which in-degree counting handles
// natively. It also produces the topological wave structure the
// execution engine (design.md D-K) consumes directly: Waves[i] is every
// step that can run in round i.
//
// steps supplies manifest declaration order (nodes is a map and has
// none); design.md D-K requires wave membership to be ordered by
// declaration order, and the residual cycle report to be deterministic
// for the same reason.
func kahn(steps []manifest.Step, nodes map[string]Node) ([][]string, error) {
	declOrder := make([]string, 0, len(steps))
	inDegree := make(map[string]int, len(nodes))
	children := make(map[string][]string, len(nodes))

	for _, s := range steps {
		declOrder = append(declOrder, s.ID)
		inDegree[s.ID] = len(nodes[s.ID].Needs)
	}
	for id, n := range nodes {
		for _, dep := range n.Needs {
			children[dep] = append(children[dep], id)
		}
	}

	processed := make(map[string]bool, len(nodes))
	var waves [][]string

	for len(processed) < len(nodes) {
		var wave []string
		for _, id := range declOrder {
			if !processed[id] && inDegree[id] == 0 {
				wave = append(wave, id)
			}
		}
		if len(wave) == 0 {
			// No zero-in-degree node remains: every unprocessed node
			// is in, or downstream of, a cycle. Stop — nothing further
			// can ever become ready.
			break
		}

		for _, id := range wave {
			processed[id] = true
		}
		for _, id := range wave {
			for _, child := range children[id] {
				inDegree[child]--
			}
		}

		waves = append(waves, wave)
	}

	if len(processed) < len(nodes) {
		var stuck []string
		for _, id := range declOrder {
			if !processed[id] {
				stuck = append(stuck, id)
			}
		}
		return nil, &CycleError{StepIDs: stuck}
	}

	return waves, nil
}
