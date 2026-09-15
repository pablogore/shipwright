# Impact Graph Specification

## Purpose

Defines the contract of `pkg/graph.Impacted` — a pure, dependency-free
reverse-reachability function over an opaque node graph. Given a dependency
edge map and a set of changed nodes, it MUST return every node transitively
dependent on any changed node, including the changed nodes themselves. This
contract has no knowledge of `manifest.Step`, providers, or any other
Shipwright execution concept — it exists so an external caller (Verimand,
`github.com/getsyntegrity/verimand-platform` issue #207) can import it as an
ordinary dependency-free algorithm.

**Terminology note:** "changed" here means "known to have changed by the
caller" — this contract does not detect changes itself (no git, no
filesystem, no I/O of any kind); it only propagates a caller-supplied set.

## Requirements

### Requirement: Dependency-Direction Convention

`edges[X]` MUST represent the set of nodes `X` directly depends on — the same
direction as this repository's existing `Node.Needs`/Build-graph convention.
`Impacted` MUST compute the reverse of this relation internally; callers MUST
NOT be required to pre-invert their edge map.

#### Scenario: A node is impacted by a change in something it depends on

- GIVEN an edge map where `edges["service"] = ["lib"]` (service depends on
  lib)
- WHEN `Impacted` is called with `changed = ["lib"]`
- THEN the output includes `"service"`

### Requirement: Changed Set Self-Inclusion

Every node present in the `changed` input MUST appear in the output, even if
it has no dependents and even if it is absent from `edges`' key set.

#### Scenario: A changed node with no dependents is still in the output

- GIVEN an edge map with no entry depending on `"leaf"`
- WHEN `Impacted` is called with `changed = ["leaf"]`
- THEN the output is exactly `["leaf"]`

#### Scenario: A changed node unknown to the edge map is still in the output

- GIVEN an edge map whose keys do not include `"ghost"`
- WHEN `Impacted` is called with `changed = ["ghost"]`
- THEN the output includes `"ghost"` (it is impacted by definition — the
  graph having no record of it does not remove it from the changed set)

### Requirement: Transitive Reverse Reachability

`Impacted` MUST include every node that depends on a changed node at any
depth, not only direct dependents, and MUST correctly handle fan-out (one
changed node with multiple direct dependents) and fan-in (multiple changed
nodes sharing a common transitive dependent).

#### Scenario: Direct dependent is impacted

- GIVEN `edges["a"] = ["b"]`
- WHEN `Impacted` is called with `changed = ["b"]`
- THEN the output includes `"a"`

#### Scenario: Multi-level (transitive) dependent is impacted

- GIVEN `edges["a"] = ["b"]` and `edges["b"] = ["c"]`
- WHEN `Impacted` is called with `changed = ["c"]`
- THEN the output includes `"a"` and `"b"`

#### Scenario: Fan-out — one changed node, multiple direct dependents

- GIVEN `edges["a"] = ["c"]` and `edges["b"] = ["c"]`
- WHEN `Impacted` is called with `changed = ["c"]`
- THEN the output includes both `"a"` and `"b"`

#### Scenario: Fan-in — multiple changed nodes converging on one dependent

- GIVEN `edges["shared"] = ["a", "b"]`
- WHEN `Impacted` is called with `changed = ["a", "b"]`
- THEN the output includes `"shared"` exactly once

#### Scenario: Disconnected node is not impacted

- GIVEN `edges["a"] = ["b"]` and an unrelated node `"z"` with no edge to or
  from `"b"`
- WHEN `Impacted` is called with `changed = ["b"]`
- THEN the output does not include `"z"`

### Requirement: Malformed-Input Tolerance

A dangling edge — one naming a node absent from `edges`' own key set — MUST
NOT cause a panic; that node simply has no further outgoing edges to
traverse.

#### Scenario: Dangling edge does not panic

- GIVEN `edges["a"] = ["missing"]` where `"missing"` is not itself a key in
  `edges`
- WHEN `Impacted` is called with any `changed` set
- THEN the call returns normally, without panicking

### Requirement: Cycle Safety

A cyclic or otherwise malformed graph MUST NOT cause an infinite loop or a
panic. Traversal MUST be guarded by a visited-node set.

#### Scenario: Self-edge terminates

- GIVEN `edges["a"] = ["a"]`
- WHEN `Impacted` is called with `changed = ["a"]`
- THEN the call terminates and the output is `["a"]`

#### Scenario: Mutual two-node cycle terminates

- GIVEN `edges["a"] = ["b"]` and `edges["b"] = ["a"]`
- WHEN `Impacted` is called with `changed = ["b"]`
- THEN the call terminates and the output includes both `"a"` and `"b"`

#### Scenario: Longer cycle terminates

- GIVEN a cycle `edges["a"] = ["b"]`, `edges["b"] = ["c"]`,
  `edges["c"] = ["a"]`
- WHEN `Impacted` is called with `changed = ["c"]`
- THEN the call terminates and the output includes `"a"`, `"b"`, and `"c"`

### Requirement: Duplicate Edge Collapse

Duplicate edges between the same two nodes MUST NOT affect the correctness of
the output — they collapse to a single logical edge.

#### Scenario: Duplicate edge does not duplicate output or break traversal

- GIVEN `edges["a"] = ["b", "b"]`
- WHEN `Impacted` is called with `changed = ["b"]`
- THEN the output includes `"a"` exactly once

### Requirement: Deterministic, Sorted Output

`Impacted` MUST return a sorted, deduplicated slice. The same input MUST
always produce the identical output slice, independent of Go's unordered map
iteration.

#### Scenario: Repeated calls with identical input produce identical output

- GIVEN a fixed edge map and a fixed `changed` set
- WHEN `Impacted` is called multiple times with the same input
- THEN every call returns the same slice, in the same lexicographically
  sorted order, with no duplicate entries

#### Scenario: Empty graph and empty changed set

- GIVEN an empty edge map
- WHEN `Impacted` is called with an empty `changed` slice
- THEN the output is an empty slice, not nil-vs-empty-ambiguous in a way that
  breaks caller equality checks against `[]string{}`

### Requirement: No Execution or Manifest Coupling

`Impacted`'s exported signature MUST take only opaque string node
identifiers (`map[string][]string`, `[]string`) and MUST NOT reference
`manifest.Step`, any provider type, or any other Shipwright execution
concept. `pkg/graph` MUST NOT import `internal/workflow/manifest` or any
`internal/` package.

#### Scenario: Package has no dependency on the manifest schema

- GIVEN `pkg/graph`'s import graph
- WHEN inspected
- THEN it does not import `internal/workflow/manifest` or any other
  `internal/` package

## Out of Scope

Module/dependency scanning that produces the `edges` input (e.g. `go list
-json`) is a caller responsibility (Verimand's `internal/ciimpact`), not part
of this contract. Manifest generation from an `Impacted` result is Verimand's
V7 concern (`design.md` §2.3 of the `207-affected-module-detection` change in
`verimand-platform`), not this package. Any change to
`internal/workflow/graph` or the existing manifest execution engine is
unaffected by and out of scope for this contract.
