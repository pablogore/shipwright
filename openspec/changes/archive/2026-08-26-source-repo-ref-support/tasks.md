# Tasks: Wire `spec.source.repo` / `spec.source.ref` Into Workflow Source Resolution

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 80–120 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Core implementation + tests | PR 1 | `go test -race -run TestResolveWorkflowSource ./...` | Invalid-repo clone proves delegation path without network | `main.go` resolveWorkflowSource + caller; fully reversible |

## Phase 1: Signature Update

- [x] 1.1 Add `ctx context.Context` as first parameter to `resolveWorkflowSource` in `main.go:563` — new signature: `func resolveWorkflowSource(ctx context.Context, client *dagger.Client, spec manifest.SourceSpec) (*dagger.Directory, error)`
- [x] 1.2 Add `"strings"` to the import block in `main.go`
- [x] 1.3 Update the single caller `runWorkflowEngine` at `main.go:484` to pass `ctx` — change `resolveWorkflowSource(client, m.Spec.Source)` to `resolveWorkflowSource(ctx, client, m.Spec.Source)`

## Phase 2: Core Implementation

- [x] 2.1 Replace the `spec.Repo != ""` stub in `resolveWorkflowSource` with protocol detection: `protocol := "https"; if strings.HasPrefix(spec.Repo, "git@") { protocol = "ssh" }`
- [x] 2.2 Add ref defaulting: `ref := spec.Ref; if ref == "" { ref = "main" }`
- [x] 2.3 Build `shared.GitCloneOpts{Repo: spec.Repo, Branch: ref, Name: "workflow-source"}` and call `shared.CloneRepo(ctx, client, opts, protocol)`, returning its result
- [x] 2.4 Verify `"github.com/pablogore/shipwright/internal/pipelines/shared"` is in the import block (add if missing)

## Phase 3: Tests (RED first, then GREEN)

- [x] 3.1 RED: Add `TestResolveWorkflowSource_HTTPSClone` in `main_test.go` — `spec.Repo = "https://github.com/org/repo.git"`, `spec.Ref = ""`, expect error containing `"main"` branch (proves HTTPS path + ref defaulting are exercised; real clone fails on invalid repo)
- [x] 3.2 RED: Add `TestResolveWorkflowSource_SSHClone` in `main_test.go` — `spec.Repo = "git@github.com:org/repo.git"`, expect error (proves SSH detection; SSH clone fails without key)
- [x] 3.3 RED: Add `TestResolveWorkflowSource_ExplicitRefPreserved` in `main_test.go` — `spec.Ref = "develop"`, expect error containing `"develop"` (proves explicit ref is forwarded)
- [x] 3.4 RED: Add `TestResolveWorkflowSource_EmptyRefDefaultsToMain` in `main_test.go` — `spec.Ref = ""`, expect error containing `"main"` (proves defaulting before clone)
- [x] 3.5 RED: Add `TestResolveWorkflowSource_PathFallback` in `main_test.go` — `spec.Repo = ""`, `spec.Path = "./src"`, expect `client.Host().Directory("./src")` with no error (proves existing path logic unchanged)
- [x] 3.6 GREEN: Run `go test -race -run TestResolveWorkflowSource ./...` — all five tests pass

## Phase 4: Cleanup

- [x] 4.1 Remove the old `spec.source.repo is not supported` error comment from `resolveWorkflowSource` doc comment — replace with description of the two code paths (repo → clone, path → local)
- [x] 4.2 Run `go vet ./...` and `go build ./...` to confirm no compilation issues
