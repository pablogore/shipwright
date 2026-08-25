---
name: shipwright-testing
description: "Trigger: Go tests, go test coverage, executor/plugin mocks, golden files, local/CI parity. Apply focused Go testing patterns for Shipwright."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## Activation Contract

Load this skill when writing or reviewing Go tests in this repo, adding coverage for `internal/executors`, `internal/plugins`, `internal/config`, or `internal/pipelines`, or validating that a Dagger-executed step behaves the same locally and in CI.

## Hard Rules

- Prefer table-driven tests for multiple cases; use `t.Run(tt.name, ...)`.
- Test behavior and state transitions, not implementation trivia.
- Use `t.TempDir()` for filesystem tests; never rely on a real home directory or the developer's working tree.
- Keep integration tests skippable with `testing.Short()` when they run external commands, real Dagger containers, or slow flows.
- Use small mocks/interfaces around execution boundaries — this repo already does this for executors (`internal/executors/mocks.go`) and plugins (`internal/plugins/mocks.go`); extend those rather than hand-rolling new fakes.
- Golden files must be deterministic; update only through the repo's `-update` path and rerun tests without `-update`.
- **Local/CI semantic parity is a hard requirement.** Shipwright's whole premise is that a step run through the `NativeExecutor` locally and the same step run through the `DockerExecutor`/`CicdExecutor` in CI produce equivalent results (see `internal/executors/selector.go`). A test that only exercises one executor path for a step that has both is incomplete — cover the executor selection logic itself when adding a new step, and prefer testing behavior against the `Executor` interface so both implementations are exercised.
- **No mutable tool versions unless explicitly justified.** Go version, base container images (`golang:1.26.1-alpine`, etc.), and linter/vuln-scanner versions are pinned in pipeline config and CI (see `docs/PIPELINE_DEVELOPMENT.md`, `docs/LOCAL_USAGE.md`). Do not write tests that assert against `latest` tags or unpinned tool output; if a test needs a new pinned version, update the pin explicitly and say why in the commit/PR, don't let a test silently start passing against a moving target.

## Decision Gates

| Target | Test pattern |
|---|---|
| Pure function or parser (`internal/config`) | Table-driven unit test. |
| Error behavior | Explicit success and failure cases. |
| File operations | `t.TempDir()` plus focused assertions. |
| Executor behavior (`internal/executors`) | Test against the `Executor` interface via mocks; add a parity case when both native and container paths exist for the step. |
| Plugin hook/step (`internal/plugins`) | Use `PluginContext` mocks; assert hook registration and step execution independently. |
| Dagger-driven step output | Golden file test where output is deterministic; skip real container execution under `-short`. |
| Real external command (`go`, `golangci-lint`, `govulncheck`, `dagger`) | Integration test; skip in `-short`. |

## Execution Steps

1. Identify behavior under test and the smallest public boundary that proves it (usually an interface in `internal/interfaces/`).
2. Choose the test pattern from the decision gate.
3. Name cases by scenario, not input mechanics.
4. Assert outputs, errors, state, and side effects explicitly — including that secrets are never asserted against as plain strings (see `shipwright-security`).
5. Run the narrow package test first, then the relevant broader suite.
6. When the change affects a step available through more than one executor, add or update the parity case and note the local/CI parity check explicitly in the PR test plan.
7. For golden updates: run with `-update`, inspect the diff, then rerun without `-update`.

## Output Contract

Report test files changed, scenarios covered, commands executed, golden files updated, local/CI parity verified or explicitly N/A with reason, and any skipped integration scope.

## Examples

### Table-driven test

```go
func TestValidateGoVersion(t *testing.T) {
    tests := []struct {
        name    string
        version string
        wantErr bool
    }{
        {name: "valid semver", version: "1.26.1"},
        {name: "valid major.minor", version: "1.26"},
        {name: "empty version", version: "", wantErr: true},
        {name: "malformed version", version: "vNext", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateGoVersion(tt.version)
            if (err != nil) != tt.wantErr {
                t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Executor parity via interface

```go
func TestBuildStep_AcrossExecutors(t *testing.T) {
    executors := map[string]Executor{
        "native": NewNativeExecutor(),
        "docker": NewMockDockerExecutor(t), // wraps a real/fake Dagger container
    }

    for name, exec := range executors {
        t.Run(name, func(t *testing.T) {
            got, err := exec.Build(context.Background())
            if err != nil {
                t.Fatalf("Build() error = %v", err)
            }
            if !got.Success {
                t.Fatalf("Build() success = false, want true")
            }
        })
    }
}
```

### Golden file pattern

```go
var update = flag.Bool("update", false, "update golden files")

func assertGolden(t *testing.T, path string, got string) {
    t.Helper()
    if *update {
        if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
            t.Fatal(err)
        }
    }
    want, err := os.ReadFile(path)
    if err != nil {
        t.Fatal(err)
    }
    if got != string(want) {
        t.Fatalf("golden mismatch for %s", path)
    }
}
```

## Commands

```bash
go test ./...
go test -v ./internal/executors/...
go test -run TestValidateGoVersion ./internal/config
go test -cover ./...
go test ./internal/config -update
go test -short ./...
```
