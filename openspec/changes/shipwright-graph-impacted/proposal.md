# Proposal: `pkg/graph.Impacted` — reverse-reachability for changed-node impact analysis

## Intent

Verimand (`github.com/getsyntegrity/verimand-platform`, issue [#207](https://github.com/getsyntegrity/verimand-docs/issues/207)) needs to compute, from a module dependency graph and a set of changed nodes, every node that transitively depends on a changed node — so its CI pipeline can run only what a change actually affects, and safely fall back to running everything when it cannot prove otherwise.

That computation is a generic graph algorithm (reverse reachability from a seed set), not a Shipwright execution concern. It has no business living in `internal/workflow/graph`, which is manifest-`Step`-specific and internal to Shipwright's own engine — and `internal/` packages are not importable from another Go module regardless. Verimand needs to import this algorithm as an ordinary dependency, exactly as it already imports `pkg/shipwright`'s capability interfaces.

**Success looks like:** a new public package, `pkg/graph`, exporting one pure function — `Impacted(edges map[string][]string, changed []string) []string` — with no dependency on `manifest`, no I/O, no side effects, and a test suite exhaustive enough that both Shipwright and Verimand can trust it as a dependency-free primitive.

## Scope

### In Scope

| # | Deliverable |
|---|---|
| 1 | New public package `pkg/graph`, sibling to `pkg/shipwright` |
| 2 | `Impacted(edges map[string][]string, changed []string) []string` — pure, deterministic, no I/O |
| 3 | Exhaustive test suite (empty graph, empty changed set, single node, direct/multi-level dependent, fan-out, fan-in, disconnected nodes, duplicate edges, unknown changed node, deterministic output, cycles/malformed graph) |
| 4 | A tagged release (or pinned commit) Verimand can add to its `go.mod` |

### Out of Scope (Non-Goals)

| Non-goal | Boundary |
|---|---|
| Any change to `internal/workflow/graph` or the manifest/execution engine | `Impacted` is a standalone algorithm; the existing DAG engine is untouched |
| Module/dependency scanning (`go list -m`, `go list -json`) | Verimand's own concern (`internal/ciimpact.BuildModuleGraph`); this package only consumes an already-built edge map |
| Any Verimand-specific type or naming (`ModuleGraph`, `ImpactPlan`, ...) | `Impacted` takes opaque string node IDs only — no coupling to a specific caller's domain model |
| CLI wiring, manifest generation from impact results | Verimand's V7 (`design.md` §2.3 in the `207-affected-module-detection` change), not this package |

## Decisions

### D1 — New capability domain: `impact-graph`, not `workflow-execution`

| Item | Position |
|---|---|
| Candidate | `workflow-execution` (existing domain — DAG-adjacent) |
| Decision | New domain: `impact-graph` |
| Rationale | `workflow-execution` owns provider resolution, topological ordering, and execution of `manifest.Step` graphs — it is coupled to the manifest schema by design. `Impacted` is explicitly decoupled from `manifest` (opaque string nodes, no `Step` knowledge) so it can be imported by a caller — Verimand — that has never heard of a Shipwright manifest. Folding it into `workflow-execution` would either leak manifest coupling into a caller that must not have it, or force `workflow-execution` to carry a manifest-free code path that has no other reason to exist there. |

### D2 — Direction convention matches `Node.Needs`

| Item | Position |
|---|---|
| Convention | `edges[X]` = the set of nodes `X` directly depends on (dependency direction, same as the existing `Node.Needs`/Build-graph convention) |
| Rationale | `Impacted` computes the reverse of this relation internally. Keeping the public input in the same direction as every other graph representation in this codebase avoids forcing every caller (including Verimand) to invert edges before calling in |

## Capabilities

### New Capabilities

- `impact-graph`: reverse-reachability over an opaque node graph — the `Impacted` contract, its determinism/purity guarantees, and its malformed-input handling (cycles, dangling edges, unknown nodes).

### Modified Capabilities

- None. This is additive; no existing capability's contract changes.

## Approach

1. Specify `Impacted`'s contract as testable requirements: purity, determinism (sorted output), self-inclusion of `changed`, dangling-edge tolerance, cycle termination (RFC 2119 keywords, Given/When/Then scenarios per this repo's spec rules).
2. Design the algorithm: build a reverse adjacency map from `edges`, then BFS/DFS from every node in `changed` with a visited-set guard (cycle-safe by construction, not by detection-and-reject).
3. RED tests first (`strict_tdd: true`) covering the full list in Scope #3, including the malformed-graph cases called out as highest risk below.
4. Implement `pkg/graph/impacted.go` to green.
5. Tag a release (or record the merge commit) Verimand's `go.mod` can pin to.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `pkg/graph/` | New | The package, `impacted.go` + `impacted_test.go` |
| `openspec/specs/impact-graph/` | New | The capability contract (created at archive) |
| `go.mod` (this repo) | Unaffected | No new dependency — pure stdlib |
| Everything else | Unaffected | No existing package imports or is imported by `pkg/graph` |

**Dagger compatibility note:** `Impacted` has no I/O and no Dagger type-system exposure — it is a plain Go function, not a capability/provider surface, and is not called through Dagger's module system. No Dagger SDK or pipeline-step compatibility impact.

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Cycle handling is wrong (infinite loop or missed node) | Med | Visited-set guard is structural, not a special case; RED tests include self-edges, mutual pairs, long cycles, and fan-in diamonds before implementation |
| Verimand's `ImpactedFunc` type (`func(edges map[string][]string, changed []string) []string`) drifts from this package's exported signature | Low | Signature is copied verbatim from `verimand-platform`'s `design.md` §2.2, which was itself authored against this exact contract |
| Blocking dependency: Verimand's PR3 (issue #207) cannot start until this ships and is pinned | High (by design) | This proposal exists specifically to unblock it — sequence release before announcing completion back to Verimand |

## Rollback Plan

Purely additive — a new package with no existing consumer inside this repo.

- Revert the merge commit; no other code references `pkg/graph`, so nothing else breaks.
- No state, data, config, or release migration.
- Verification after revert: `go build -o shipwright .` and `go test -race ./...` both green.

## Dependencies

- None (stdlib only).

## Success Criteria

- [ ] `pkg/graph.Impacted(edges map[string][]string, changed []string) []string` exists, exported, and matches the signature in `verimand-platform`'s `design.md` §2.2 exactly
- [ ] Output is sorted and deduplicated on every call (determinism)
- [ ] Every node in `changed` appears in the output, including nodes absent from `edges`' key set
- [ ] A node with a dangling edge (naming a node absent from `edges`) does not panic
- [ ] A cyclic or malformed graph terminates and does not panic (self-edges, mutual pairs, longer cycles)
- [ ] Duplicate edges collapse to a single logical edge
- [ ] Fan-out and fan-in cases both covered by a passing test
- [ ] `go build -o shipwright .` and `go test -race ./...` green; coverage ≥ 90%
- [ ] A tagged release or pinned commit exists that `verimand-platform` can add to `go.mod`
