---
name: shipwright-testing-strategy
description: "Trigger: test strategy, coverage threshold, test level, unit vs integration, quality gate, local/CI parity. Define the mandatory testing standards and the test level for a change in Shipwright."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
  ported-from: "ego-rs skills/testing-strategy/SKILL.md"
---

## Activation Contract

Load this skill when:
- Deciding whether a change needs a unit test or a `-short`-skippable integration test
- Reviewing whether a PR meets the coverage and complexity gates
- Adding tests to a package that has none
- Arguing about test level or test placement

Pair with `shipwright-testing-tdd` for the concrete RED→GREEN workflow, and with `shipwright-testing` for the hard placement rules and existing test-pattern decision gate.

## Scope

These standards govern **new tests and tests you modify**. The existing suite is not migrated retroactively.

- New test → full compliance.
- Test you are already editing → bring it into compliance, without expanding the diff beyond what you touched.
- Untouched test → leave it. Cleanup is separate, deliberate work.

The coverage gate is repo-wide (a property of `go test -coverprofile`), not a property of an individual test.

## Quality Standards (Mandatory)

### Test-Driven Development

- Write the test BEFORE the implementation. The test is the specification.
- Implementation exists to make the test pass. Refactor with the test as the net.
- A task is not started until its RED test exists and fails for the right reason.

### Coverage

- Local gate: `make coverage` → `go test -coverprofile=coverage.out -covermode=atomic` against `COVERAGE_THRESHOLD=90` (see `Makefile`).
- CI gate: `make coverage-threshold` / the pipeline's coverage step enforces `COVERAGE_THRESHOLD_CI=70` — lower than local on purpose, but still a hard floor.
- `make coverage-100` is an opt-in stricter mode (`COVERAGE_THRESHOLD_100=100`); not required unless a package explicitly adopts it.
- Coverage is a floor, not a target. A number over untested error paths is a failure dressed as a pass.
- `mocks/`, `examples/`, `app/`, and `config/` are excluded from the coverage calculation (see the `grep -v` filters in `Makefile`'s `coverage` target) — don't rely on those directories to move the number.

### Cyclomatic Complexity

- Maximum 15 per function, configured as `gocyclo` in `.golangci.yml`.
- **Not currently wired into CI as a blocking step** — check it explicitly with `golangci-lint run` (or `make vet` for the narrower `go vet` checks) before opening a PR. Don't assume CI will catch a function over the limit.
- Over the limit means extract functions. Refactoring is mandatory, not optional, once you notice it.

### Level Separation — Local/CI Executor Parity

Shipwright has no Rust-style workspace/crate boundaries, so its test levels aren't a unit/integration/end-to-end pyramid with fixed shares. Its actual defining invariant is **local/CI executor parity**: a step run through `NativeExecutor` locally and the same step run through `DockerExecutor`/`CicdExecutor` in CI must produce equivalent results (see `internal/executors/selector.go`). That parity requirement *is* Shipwright's level-separation rule — see `shipwright-testing`'s "Local/CI semantic parity is a hard requirement" rule for the mechanics.

| Level | Location | What belongs |
|---|---|---|
| Unit | in-package `_test.go`, mocked `Executor`/`PluginContext` boundaries | Pure functions, config parsing, decision/selection logic |
| Integration (`-short`-skippable) | in-package `_test.go`, guarded by `testing.Short()` | Real Dagger container execution, real external commands (`go`, `golangci-lint`, `govulncheck`, `dagger`) |
| Parity case | wherever the step's test already lives | The same behavior asserted through both the `NativeExecutor` and `DockerExecutor`/`CicdExecutor` paths, via the `Executor` interface |

A unit test that reaches a real container, a real external command, or the network is misclassified — it moves to the integration level and gets a `testing.Short()` guard, not a feature flag or an `-ignore`-style skip.

## Decision Gates

| Question | Answer |
|---|---|
| Touches only in-memory state or mocked `Executor`/`PluginContext` | Unit — in-package `_test.go` |
| Runs a real external command or real Dagger container | Integration — same package, guarded by `testing.Short()` |
| The step exists through both `NativeExecutor` and `DockerExecutor`/`CicdExecutor` | Add or update the parity case (see `shipwright-testing`) |
| PR touches `internal/executors`, `internal/plugins`, `internal/config`, or `internal/pipelines` | Coverage gate applies — 90% local (`make coverage`), 70% CI floor (`make coverage-threshold`) |
| A function crosses gocyclo complexity 15 | Extract functions before merge; verify with `golangci-lint run` |

## Running Tests

```bash
go test ./...                # everything
go test -race ./...          # race detector — matches `make test` and CI
go test -short ./...         # skip integration/container tests
make coverage                 # coverage report against the 90% local threshold
make coverage-threshold       # CI-equivalent check against the 70% floor
make coverage-100             # opt-in 100% mode
make vet                      # go vet
golangci-lint run              # gocyclo (15) plus the rest of .golangci.yml — not yet a CI-blocking step
```

## Test Naming

- Unit: `Test<Function>_<Scenario>` — `TestValidateGoVersion_MalformedVersion`
- Table-driven cases: name `tt.name` by scenario, never by input mechanics
- Integration/parity: descriptive behavior — `TestBuildStep_NativeAndDockerExecutorsAgree`

## Output Contract

Report, for every change:
1. Which level(s) the new tests live at (unit / `-short` integration), and why.
2. That `go test -race ./...` passes with 0 failures.
3. Coverage impact if the change touches a package under the coverage gate.
4. Any function that crossed gocyclo complexity 15 and how it was split.

## References

- `shipwright-testing` — hard placement rules, existing test-pattern decision gate, examples
- `shipwright-testing-tdd` — the RED→GREEN→REFACTOR workflow and test-double patterns
- `internal/executors/selector.go` — the local/CI parity source of truth
