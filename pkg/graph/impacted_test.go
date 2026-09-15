package graph_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pablogore/shipwright/pkg/graph"
)

func TestImpacted_TransitiveReverseReachability(t *testing.T) {
	t.Run("direct dependent is impacted", func(t *testing.T) {
		edges := map[string][]string{"a": {"b"}}
		got := graph.Impacted(edges, []string{"b"})
		assertSameElements(t, []string{"a", "b"}, got)
	})

	t.Run("multi-level transitive dependent is impacted", func(t *testing.T) {
		edges := map[string][]string{"a": {"b"}, "b": {"c"}}
		got := graph.Impacted(edges, []string{"c"})
		assertSameElements(t, []string{"a", "b", "c"}, got)
	})

	t.Run("fan-out: one changed node, multiple direct dependents", func(t *testing.T) {
		edges := map[string][]string{"a": {"c"}, "b": {"c"}}
		got := graph.Impacted(edges, []string{"c"})
		assertSameElements(t, []string{"a", "b", "c"}, got)
	})

	t.Run("fan-in: multiple changed nodes converge on one dependent", func(t *testing.T) {
		edges := map[string][]string{"shared": {"a", "b"}}
		got := graph.Impacted(edges, []string{"a", "b"})
		assertSameElements(t, []string{"shared", "a", "b"}, got)
	})

	t.Run("disconnected node is not impacted", func(t *testing.T) {
		edges := map[string][]string{"a": {"b"}, "z": {"y"}}
		got := graph.Impacted(edges, []string{"b"})
		assert.NotContains(t, got, "z")
		assert.NotContains(t, got, "y")
		assertSameElements(t, []string{"a", "b"}, got)
	})
}

func TestImpacted_ChangedSetSelfInclusion(t *testing.T) {
	t.Run("changed leaf with no dependents is still in the output", func(t *testing.T) {
		edges := map[string][]string{"a": {"b"}}
		got := graph.Impacted(edges, []string{"leaf"})
		assertSameElements(t, []string{"leaf"}, got)
	})

	t.Run("changed node unknown to the edge map is still in the output", func(t *testing.T) {
		edges := map[string][]string{"a": {"b"}}
		got := graph.Impacted(edges, []string{"ghost"})
		assertSameElements(t, []string{"ghost"}, got)
	})
}

func TestImpacted_MalformedInputTolerance(t *testing.T) {
	t.Run("dangling edge does not panic", func(t *testing.T) {
		edges := map[string][]string{"a": {"missing"}}
		assert.NotPanics(t, func() {
			got := graph.Impacted(edges, []string{"missing"})
			assertSameElements(t, []string{"a", "missing"}, got)
		})
	})
}

func TestImpacted_CycleSafety(t *testing.T) {
	t.Run("self-edge terminates", func(t *testing.T) {
		edges := map[string][]string{"a": {"a"}}
		var got []string
		assert.NotPanics(t, func() {
			got = graph.Impacted(edges, []string{"a"})
		})
		assertSameElements(t, []string{"a"}, got)
	})

	t.Run("mutual two-node cycle terminates", func(t *testing.T) {
		edges := map[string][]string{"a": {"b"}, "b": {"a"}}
		var got []string
		assert.NotPanics(t, func() {
			got = graph.Impacted(edges, []string{"b"})
		})
		assertSameElements(t, []string{"a", "b"}, got)
	})

	t.Run("three-node cycle terminates", func(t *testing.T) {
		edges := map[string][]string{"a": {"b"}, "b": {"c"}, "c": {"a"}}
		var got []string
		assert.NotPanics(t, func() {
			got = graph.Impacted(edges, []string{"c"})
		})
		assertSameElements(t, []string{"a", "b", "c"}, got)
	})
}

func TestImpacted_DuplicateEdgeCollapse(t *testing.T) {
	t.Run("duplicate edge does not duplicate output", func(t *testing.T) {
		edges := map[string][]string{"a": {"b", "b"}}
		got := graph.Impacted(edges, []string{"b"})
		assertSameElements(t, []string{"a", "b"}, got)
		assert.Len(t, got, 2)
	})
}

func TestImpacted_DeterministicSortedOutput(t *testing.T) {
	t.Run("repeated calls with identical input produce identical output", func(t *testing.T) {
		edges := map[string][]string{"a": {"c"}, "b": {"c"}, "c": {"d"}}
		changed := []string{"d"}

		first := graph.Impacted(edges, changed)
		for i := 0; i < 10; i++ {
			got := graph.Impacted(edges, changed)
			assert.Equal(t, first, got)
			assert.True(t, sort.StringsAreSorted(got), "output must be sorted: %v", got)
		}
	})

	t.Run("empty graph and empty changed set returns an empty, non-nil slice", func(t *testing.T) {
		got := graph.Impacted(map[string][]string{}, []string{})
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}

func TestImpacted_DependencyDirectionConvention(t *testing.T) {
	t.Run("edges[X] means X depends on the listed nodes, not the reverse", func(t *testing.T) {
		// service depends on lib (edges["service"] = ["lib"]). A change to
		// lib impacts service, not the other way around.
		edges := map[string][]string{"service": {"lib"}}
		got := graph.Impacted(edges, []string{"lib"})
		assertSameElements(t, []string{"service", "lib"}, got)
	})
}

func assertSameElements(t *testing.T, want, got []string) {
	t.Helper()
	wantSorted := append([]string(nil), want...)
	gotSorted := append([]string(nil), got...)
	sort.Strings(wantSorted)
	sort.Strings(gotSorted)
	assert.Equal(t, wantSorted, gotSorted)
}
