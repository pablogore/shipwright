# Tasks: `pkg/graph.Impacted`

Source of truth for contract and rationale: `proposal.md`, `specs/impact-graph/spec.md`, `design.md`. This file only orders and slices the work. No task touches `internal/executors` or `internal/pipelines` — `pkg/graph` is a standalone, dependency-free package, so no task in this change carries the repo's "higher risk" flag.

## Phase 1 — RED: failing tests for every spec scenario

Ref: `design.md` §4 (scenario → test-case table), `strict_tdd: true`.

- [ ] 1.1 Scaffold `pkg/graph/impacted.go` (signature only, `panic("not implemented")` body) and `pkg/graph/impacted_test.go`
- [ ] 1.2 Table-driven tests: Transitive Reverse Reachability (direct, multi-level, fan-out, fan-in, disconnected-excluded)
- [ ] 1.3 Table-driven tests: Changed Set Self-Inclusion (changed leaf with no dependents, changed node unknown to `edges`)
- [ ] 1.4 Table-driven tests: Malformed-Input Tolerance (dangling edge) and Cycle Safety (self-edge, mutual pair, three-node cycle)
- [ ] 1.5 Table-driven tests: Duplicate Edge Collapse
- [ ] 1.6 Table-driven tests: Deterministic, Sorted Output (repeated-call equality, empty graph + empty changed set, nil-vs-empty-slice assertion)
- [ ] 1.7 Confirm all of 1.2–1.6 fail for the right reason (`panic("not implemented")`), not a compile error

## Phase 2 — GREEN: implementation

Ref: `design.md` §2 (algorithm), §3 (data structures).

- [ ] 2.1 Implement reverse-adjacency construction from `edges`
- [ ] 2.2 Implement the BFS traversal with a `visited` set seeded from `changed`
- [ ] 2.3 Implement deterministic output: convert `visited` to a slice, `sort.Strings`, always non-nil
- [ ] 2.4 Run `go test -race ./pkg/graph/...` — all Phase 1 tests green

## Phase 3 — Structural check + quality gates

Ref: spec.md Requirement "No Execution or Manifest Coupling"; `openspec/config.yaml` testing/quality_tools.

- [ ] 3.1 Verify `pkg/graph` imports nothing from `internal/` (`go list -deps ./pkg/graph/...` or equivalent) — add as a static assertion (e.g. an import-graph test), not just a manual check
- [ ] 3.2 `go build -o shipwright .`
- [ ] 3.3 `go vet ./pkg/graph/...` and `golangci-lint run ./pkg/graph/...`
- [ ] 3.4 `gofmt` / `goimports` clean
- [ ] 3.5 Coverage check: `go test -coverprofile=coverage/coverage.out -covermode=atomic ./pkg/graph/...` ≥ 90%

## Phase 4 — Release

Ref: `proposal.md` Scope #4, Success Criteria.

- [ ] 4.1 Open PR against `develop`, dual-language proposal/spec/design included
- [ ] 4.2 After merge, tag a release (or record the merge commit SHA) `verimand-platform` can pin in `go.mod`
- [ ] 4.3 Report completion back to `verimand-platform` issue #207 / `tasks.md` task S1.1, unblocking its PR3

## Gate

- [ ] Every scenario in `specs/impact-graph/spec.md` has a passing, named test
- [ ] `pkg/graph` has zero imports from `internal/` (Phase 3.1's static check, re-verified at PR review)
- [ ] `go build -o shipwright .` and `go test -race ./...` green for the whole repo, not just `pkg/graph`
