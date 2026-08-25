---
name: shipwright-testing-tdd
description: "Trigger: TDD, red green refactor, write a failing test first, test double, table-driven test, mock, fixture, arrange act assert. Apply the TDD workflow and the concrete test patterns used in Shipwright."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
  ported-from: "ego-rs skills/testing-tdd/SKILL.md"
---

## Activation Contract

Load this skill when:
- Implementing any task under TDD
- Choosing a test double (existing hand-rolled fake vs. gomock mock vs. new stub)
- Writing a table-driven test or a fixture
- Reviewing a test for shape, independence, or hidden implementation coupling

Pair with `shipwright-testing-strategy` for the level and coverage gates.

## Scope

This workflow governs **new tests and tests you modify**. The existing suite is not rewritten retroactively — an untouched test stays as it is, and cleanup is separate, deliberate work with its own review.

## Quick Reference

| Standard | Value |
|---|---|
| TDD | Test before implementation, always |
| Coverage | `make coverage` → 90% local floor; `make coverage-threshold` → 70% CI floor |
| Complexity | gocyclo ≤ 15 (`.golangci.yml`), checked via `golangci-lint run` — not yet CI-enforced |
| Race detector | `go test -race ./...` — CI-enforced |
| Mock framework | `go.uber.org/mock` (mockgen) for generated mocks in `mocks/`; hand-rolled function-field fakes for `internal/executors`, `internal/plugins` |
| Real infrastructure | Real Dagger containers or external commands, always guarded by `testing.Short()` |

```bash
go test ./...
go test -race ./...
go test -short ./...
go test -run TestValidateGoVersion ./internal/config
make coverage
golangci-lint run
```

## TDD Workflow

### Step 1 — RED: write the failing test

The test must fail for the RIGHT reason. A test that fails to compile because the function doesn't exist yet is a valid RED; a test that fails because of a typo in the assertion is not.

```go
func TestValidateGoVersion_RejectsEmpty(t *testing.T) {
    err := ValidateGoVersion("")
    if err == nil {
        t.Fatal("ValidateGoVersion(\"\") must return an error")
    }
}
```

Never write a unit test against a real resource:

```go
// WRONG — real Dagger container in a unit test
client, _ := dagger.Connect(ctx)
container := client.Container().From("golang:1.26.1-alpine")

// WRONG — real external command outside a -short guard
out, _ := exec.Command("golangci-lint", "run").Output()

// WRONG — real wall clock
expired := time.Since(issuedAt) > time.Hour
```

Use a double instead:

```go
// Existing hand-rolled fake first — real interface, isolated instance
exec := &executors.MockExecutor{
    ExecuteStepFunc: func(ctx context.Context, step string) error { return nil },
}

// gomock when no hand-rolled fake exists for the interface
ctrl := gomock.NewController(t)
cloner := mocks.NewMockCloner(ctrl)
cloner.EXPECT().Clone(gomock.Any(), "repo-url").Return(nil)
```

### Step 2 — GREEN: minimal implementation

Write the least code that turns the test green. No speculative branches, no unused parameters, no "we will need this later" hooks. Untested branches added now are the ones that break in production later.

### Step 3 — REFACTOR: split anything over complexity 15

```bash
golangci-lint run --enable-only gocyclo
```

```go
// Before — one function carrying the whole decision tree
func (r *Runner) ExecuteStep(ctx context.Context, step Step) error {
    // executor selection, cache check, plugin hooks, retries, error mapping...
}

// After — each step independently testable, each under the limit
func (r *Runner) ExecuteStep(ctx context.Context, step Step) error {
    exec := r.selectExecutor(step)
    if err := r.runHooks(ctx, step); err != nil {
        return err
    }
    return exec.ExecuteStep(ctx, step.Name)
}
```

The refactor step is not optional and the tests must stay green through it.

## Choosing a Test Double

Apply in order. Stop at the first that fits.

| Order | Double | When |
|---|---|---|
| 1 | Existing hand-rolled fake (`internal/executors/mocks.go`, `internal/plugins/mocks.go`) | The interface already has one — extend it, don't hand-roll a new one |
| 2 | Existing gomock mock (`mocks/*.go`) | A generated mock already covers the interface |
| 3 | New gomock mock via `mockgen` | The interface has no existing mock and has 3+ methods or is reused across packages |
| 4 | Hand-rolled stub | A one- or two-method interface used once; gomock is overkill |
| 5 | No double — move the test | It needs a real Dagger container or external command → it is not a unit test |

Step 5 is not a suggestion. A unit test that reaches a real container, a real external command, or the network is **misclassified**, and the fix is never a permanent `t.Skip()` or an environment-conditional guard — it moves to an in-package integration test behind `testing.Short()` (see `shipwright-testing-strategy`).

## Test Patterns

### Table-driven (Arrange–Act–Assert)

```go
func TestExecuteStep(t *testing.T) {
    tests := []struct {
        name    string
        step    Step
        wantErr bool
    }{
        {name: "known step succeeds", step: Step{Name: "build"}},
        {name: "unknown step fails", step: Step{Name: "nope"}, wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            exec := executors.NewMockExecutor()
            err := exec.ExecuteStep(context.Background(), tt.step.Name)
            if (err != nil) != tt.wantErr {
                t.Fatalf("ExecuteStep() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Cover every outcome

Enumerate the variants the function can return. One test per outcome, or one table-driven test — never a single happy-path assertion.

### Test the error paths

Error paths are where coverage gates are usually gamed. Assert the specific error — via `errors.Is`/`errors.As` or a distinct message check — not just `err != nil`:

```go
err := ValidateGoVersion("vNext")
if err == nil || !strings.Contains(err.Error(), "malformed") {
    t.Fatalf("ValidateGoVersion(\"vNext\") error = %v, want malformed-version error", err)
}
```

### Assert on observable effects

```go
func TestExecuteStep_RecordsAttempt(t *testing.T) {
    var recorded []string
    exec := &executors.MockExecutor{
        ExecuteStepFunc: func(ctx context.Context, step string) error {
            recorded = append(recorded, step)
            return nil
        },
    }

    _ = exec.ExecuteStep(context.Background(), "build")

    if len(recorded) != 1 || recorded[0] != "build" {
        t.Fatalf("recorded = %v, want [build]", recorded)
    }
}
```

### Builders over literal construction

A hand-built struct literal breaks every test whenever a field is added. Once a type is constructed the same way in three or more tests, factor a small builder or a `newTest<Type>(opts ...func(*T))` helper so new fields don't fan out across the suite.

## Common Mistakes

**Testing implementation, not behavior**

```go
// BAD — reaches into private state
if len(runner.internalQueue) != 1 { ... }

// GOOD — asserts through the public contract
got, err := runner.ExecuteStep(ctx, step)
```

**Interdependent tests.** Shared mutable state or ordering assumptions across `t.Run` cases. Every subtest constructs its own fixture; only add `t.Parallel()` deliberately, once independence is verified.

**Swallowed errors**

```go
_ = exec.ExecuteStep(ctx, step)          // BAD
err := exec.ExecuteStep(ctx, step)
if err != nil { t.Fatalf(...) }          // GOOD
```

**Permanent `t.Skip()` as an escape hatch.** Reserved for a known-flaky test with a tracking issue. Needing a real container or command means it belongs behind `testing.Short()` as an integration test, not silenced in place.

**Blind-accepting golden files.** Running with `-update` and committing without reading the diff turns a regression into a committed expectation (see `shipwright-testing`'s golden file rule).

## Coverage Checklist

Before opening a PR:

- [ ] Every new exported function has a test
- [ ] Every error path asserts the specific error, not just `err != nil`
- [ ] Edge cases: empty, zero, nil, cancelled/timed-out context
- [ ] `go test -race ./...` passes with 0 failures
- [ ] `go test -short ./...` still passes (integration tests skip cleanly)
- [ ] No function exceeds gocyclo complexity 15
- [ ] Golden files, if touched, were reviewed via `-update` diff, not blind-accepted

## Output Contract

- Tests written before implementation, RED verified before GREEN.
- Report which double was chosen at each boundary and why (hand-rolled fake / gomock / stub).
- Report any new mock added under `mocks/` (via `mockgen`) or an `internal/*/mocks.go`, and why no existing one fit.

## References

- `shipwright-testing-strategy` — levels, coverage, and complexity gates
- `shipwright-testing` — hard placement rules and existing examples
- `internal/executors/mocks.go`, `internal/plugins/mocks.go` — the hand-rolled fake pattern
- `mocks/` — gomock-generated mocks
