// Package graph provides a pure, dependency-free reverse-reachability
// algorithm over an opaque node graph. It has no dependency on
// internal/workflow/manifest or any other Shipwright execution concept — see
// openspec/changes/shipwright-graph-impacted/design.md for why it is
// deliberately not internal/workflow/graph's shared abstraction.
package graph

import "sort"

// Impacted returns every node transitively dependent on any node in changed,
// including changed itself, given edges in dependency direction (edges[X] =
// the nodes X directly depends on — same direction as this repo's existing
// Node.Needs/Build-graph convention). The output is sorted, deduplicated,
// and deterministic. See specs/impact-graph/spec.md for the full contract.
func Impacted(edges map[string][]string, changed []string) []string {
	reverse := make(map[string][]string, len(edges))
	for node, deps := range edges {
		for _, dep := range deps {
			reverse[dep] = append(reverse[dep], node)
		}
	}

	visited := make(map[string]struct{}, len(changed))
	queue := make([]string, 0, len(changed))
	for _, node := range changed {
		if _, ok := visited[node]; !ok {
			visited[node] = struct{}{}
			queue = append(queue, node)
		}
	}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		for _, dependent := range reverse[node] {
			if _, ok := visited[dependent]; ok {
				continue
			}
			visited[dependent] = struct{}{}
			queue = append(queue, dependent)
		}
	}

	result := make([]string, 0, len(visited))
	for node := range visited {
		result = append(result, node)
	}
	sort.Strings(result)
	return result
}
