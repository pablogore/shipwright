# Tasks: Provider-Managed Runtime/Toolchain Upgrade (Go First)

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | ~1,650 authored (design.md "Review Budget Forecast") |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | 4-slice chain, ~520 / ~430 / ~370 / ~290 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

```text
Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High
```

**Divergence note**: design's own slice count/boundaries (4 slices, inspect-first) were re-verified against exact current file state (`providers/go/daggerkit`, `main.go`, `providers/go/go.mod`) and confirmed correct — no re-slicing needed. `chain_strategy: stacked-to-main` is realized in this repo as **stacked PRs merging to `develop` in order**: `shipwright-chained-pr`'s hard rule forbids cutting chain branches from `main` (Git Flow), and `develop→main` is a separate automated promotion gate (commit `f972335`), not a manual PR target. "main" in the generic guard vocabulary maps to this repo's trunk, `develop`.

### Suggested Work Units (chain boundaries)

| Slice | Branch (base `develop`) | Content | Est. lines | Focused test | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|---|
| 1 | `feature/runtime-inspect-go` | `runtime-inspect` end-to-end: parse/conflict core (A1-A6, incl. `.go-version`), Inspect wiring, read-side daggerkit, D-4 guard, allowlist 5→6 | ~520 | `go test ./providers/go/... ./internal/workflow/... ./pkg/shipwright/...` | `go run . --list-steps` against a manifest with a `runtime-inspect` step | Revert commit; removes 1 allowlist entry, 1 registry pair, 1 dispatch case, 1 `main.go` case — zero mutation code exists yet |
| 2 | `feature/runtime-upgrade-single-module` | `runtime-upgrade` wiring, single-module `go.mod`/`.go-version` mutation, write-side daggerkit, allowlist 6→7 | ~430 | `go test ./providers/go/... ./internal/workflow/...` | `go run . --list-steps` with a `runtime-upgrade` step against a fixture workspace | Revert commit; no multi-module or `go build`/`tidy` validation exists yet to leave in a half-wired state |
| 3 | `feature/runtime-upgrade-workspace` | `go.work` multi-module mutation loop, A1-A3 wiring into `Upgrade`, path-escape guard | ~370 | `go test -run TestUpgrade_Workspace ./providers/go/...` | integration-tagged real-engine `Upgrade` over `testdata/runtime/workspace-3-modules` | Revert commit; single-module path (slice 2) keeps working unmutated |
| 4 | `feature/runtime-upgrade-tidy-validate` | `go mod tidy`, `go build ./...` post-mutation validation, `go.sum` delta report, integration test | ~290 | `go test -race ./providers/go/...` | `make test-integration` (real Dagger container, `integration` build tag) | Revert commit; slice 3's mutation still functions without post-validation (documented as an accepted interim gap in that PR's description) |

Every slice independently passes `go build ./...`, `go test -race ./...`, `golangci-lint run` at its own checkpoint before the next slice branches from `develop`.

---

## Phase 1 (Slice 1 — PR1 `feature/runtime-inspect-go`): `runtime-inspect` end-to-end

- [x] 1.1 Create `providers/go/testdata/runtime/{single-module,workspace-3-modules,divergent-go,divergent-toolchain,work-go-mismatch,goversion-file-mismatch,downgrade,malformed,path-escape,live-repo-drift}/` fixtures (all 9, reused across slices 2-3); `live-repo-drift` is a byte-copy of this repo's own `go.work`/`go.mod`/`.go-version`.
- [x] 1.2 RED `providers/go/toolchainpin_test.go::TestDefaultGoVersionMatchesGoMod` — parses `providers/go/go.mod`'s `go` directive, asserts equal to `defaultGoVersion` (`gobuilder.go:44`); fails today (1.26.7 vs 1.25.5) — spec req: runtime-toolchain (D-4 tier-3 substitution).
- [x] 1.3 `go get golang.org/x/mod@<root's pinned version>` in `providers/go` (`go.mod`/`go.sum`) — confirm version parity with root module (design Open Question).
- [x] 1.4 RED `providers/go/toolchain_test.go` — table-driven over `testdata/runtime/*` fixtures for `parseWorkspace`/`detectConflicts` A1-A6 (incl. `goversion-file-mismatch` proving RED against this repo's real A3 drift) — spec req: Read-Only Drift Inspection, Ambiguous sources scenario.
- [x] 1.5 GREEN `providers/go/toolchain.go` — `parseWorkspace`, `detectConflicts` (A1-A6), `*AmbiguousToolchainError`; pure Go over bytes via `golang.org/x/mod/modfile`, no Dagger, no `os.WriteFile`/`exec.Command` (threat matrix: command construction, arbitrary write) — static-guard test asserting no such imports in this file.
- [x] 1.6 RED `providers/go/daggerkit/adapter_test.go` — mocked `DaggerDirectory.File`/`Entries`, `DaggerFile.Contents` (double-selection order rule 1, daggerkit mocks first).
- [x] 1.7 GREEN `providers/go/daggerkit/{interfaces,adapter,mocks}.go` — add `DaggerDirectory.File(string) DaggerFile`, `DaggerDirectory.Entries(context.Context) ([]string, error)`, `DaggerFile.Contents(context.Context) (string, error)` (D-9 read-side; `DaggerDirectory.WithNewFile` deferred to Phase 2).
- [x] 1.8 RED `providers/go/runtimeinspector_test.go` — `GoRuntimeInspector.Inspect` happy path, nil-client/nil-source, `failOnDrift` true/false (D-4b) — spec req: Read-Only Drift Inspection scenarios.
- [x] 1.9 GREEN `providers/go/runtimeinspector.go` — `GoRuntimeInspector.Inspect(ctx, source) (string, error)`; builds `DriftReport` JSON via `pkg/shipwright/runtime.go` (create: `DriftReport`, `ConflictState` types); `failOnDrift` `with`-field returns error on conflict.
- [x] 1.10 RED extend `pkg/shipwright/capabilities_test.go`'s reflection map with `RuntimeInspector` (D-A enforcement).
- [x] 1.11 GREEN `pkg/shipwright/capabilities.go` — add `RuntimeInspector interface { Inspect(ctx, source *dagger.Directory) (string, error) }`.
- [x] 1.12 GREEN `.dagger/capabilities.go` — Layer 2 `RuntimeInspector` projection (`dagger.DaggerObject` + `Inspect(...) (string, error)`, keeps `error` per D-3).
- [x] 1.13 REQUIRED `go run ./pkg/shipwright -update` (or repo's documented golden-update command) to regenerate `pkg/shipwright/testdata/api.golden` — **review the diff**, do not blind-commit (interface count 5→6).
- [x] 1.14 RED extend `internal/workflow/manifest/validate_test.go` — `runtime-inspect` in allowlist, capability-outside-allowlist still fails — spec req: workflow-manifest MODIFIED (allowlist scenarios).
- [x] 1.15 GREEN `internal/workflow/manifest/validate.go` — add `runtime-inspect` to the allowlist (5→6) + `with` schema (`workspaceRoot`, `expectedVersion`, `failOnDrift`).
- [x] 1.16 RED extend `internal/workflow/providers/registry_test.go` — `RegisterRuntimeInspector`/`ResolveRuntimeInspector` pair.
- [x] 1.17 GREEN `internal/workflow/providers/registry.go` — add `inspectors *table[shipwright.RuntimeInspector]` + Register/Resolve pair.
- [x] 1.18 GREEN `internal/workflow/providers/register.go` — register `go-runtime` provider under `runtime-inspect` (confirm provider name at apply time per design Open Question).
- [x] 1.19 RED extend `internal/workflow/engine/dispatch_test.go` — `dispatchRuntimeInspect` straight-line resolve→call→wrap, no blocking code added (D1 confirmation).
- [x] 1.20 GREEN `internal/workflow/engine/execute.go` — `dispatchRuntimeInspect` dispatch case + `with`-field consts.
- [x] 1.21 RED extend `main_test.go` — `resolveCapabilityRef` gains a `runtime-inspect` case (D-8; was "five capability branches").
- [x] 1.22 GREEN `main.go::resolveCapabilityRef` — add `case "runtime-inspect"` (~5 lines).
- [x] 1.23 Threat-matrix RED: static import guard test confirms no `os.WriteFile`/`exec.Command`/HTTP client/git command reachable from `providers/go/toolchain.go` or `runtimeinspector.go` (No Network/Git/SCM requirement).
- [x] 1.24 Verify: `go build ./...`, `go test -race ./...`, `golangci-lint run` all green; confirm `--list-steps` lists a `runtime-inspect` step without error.

## Phase 2 (Slice 2 — PR2 `feature/runtime-upgrade-single-module`): single-module `runtime-upgrade`

- [x] 2.1 RED `providers/go/toolchain_test.go` — extend with `mutateGoMod`/`mutateGoWork`/`.go-version` mutation over `single-module`, `downgrade` (A4), `malformed` (A5) fixtures.
- [x] 2.2 GREEN `providers/go/toolchain.go` — `mutateGoMod`, mutate `.go-version`; `targetVersion` validated via `modfile.GoVersionRE` **before** any write (threat matrix: command construction from config) — RED test: `"1.26.7; rm -rf /"`, `"--flag"` rejected at parse.
- [x] 2.3 RED `providers/go/daggerkit/adapter_test.go` — mocked `DaggerDirectory.WithNewFile`.
- [x] 2.4 GREEN `providers/go/daggerkit/{interfaces,adapter,mocks}.go` — add `DaggerDirectory.WithNewFile(path, contents string) DaggerDirectory` (D-9 write-side).
- [x] 2.5 RED `pkg/shipwright/capabilities_test.go` reflection map — add `RuntimeUpgrader`.
- [x] 2.6 GREEN `pkg/shipwright/capabilities.go` — `RuntimeUpgrader interface { Upgrade(ctx, source *dagger.Directory, targetVersion string) (*dagger.Directory, error) }`.
- [x] 2.7 GREEN `.dagger/capabilities.go` — Layer 2 `RuntimeUpgrader` projection; `Upgrade` **drops `error`** (D-3, matches `Builder.Build`).
- [x] 2.8 REQUIRED regenerate `pkg/shipwright/testdata/api.golden` (`-update`, reviewed diff) — interface count 6→7.
- [x] 2.9 RED `providers/go/runtimeupgrader_test.go` — single-module happy path, ambiguous-abort (`(nil, err)`, no directory), missing-location-skipped — spec req: Discovery-Driven Upgrade scenarios.
- [x] 2.10 GREEN `providers/go/runtimeupgrader.go` — `GoRuntimeUpgrader.Upgrade`; single-module `go.mod` path only (no `go.work` traversal yet); writes `.shipwright/runtime-upgrade-report.json` into returned Directory (D-2); create `pkg/shipwright/runtime.go` additions: `UpgradeReport`, `ModuleDrift`.
- [x] 2.11 RED/GREEN `internal/workflow/manifest/validate.go` + `validate_test.go` — allowlist 6→7, `runtime-upgrade` `with` schema (`targetVersion` required, `workspaceRoot`, `tidy`, `allowDowngrade`).
- [x] 2.12 RED/GREEN `internal/workflow/providers/registry.go` + `registry_test.go` — `upgraders` table + Register/Resolve pair.
- [x] 2.13 GREEN `internal/workflow/providers/register.go` — register `go-runtime` under `runtime-upgrade`.
- [x] 2.14 RED/GREEN `internal/workflow/engine/execute.go` + `dispatch_test.go` — `dispatchRuntimeUpgrade` case, no blocking logic (D1).
- [x] 2.15 RED/GREEN `main.go::resolveCapabilityRef` + `main_test.go` — add `case "runtime-upgrade"` (D-8's second case).
- [x] 2.16 Threat-matrix RED: static import guard extended to `runtimeupgrader.go` (no host writes; only `Directory.WithNewFile`).
- [x] 2.17 Verify: `go build ./...`, `go test -race ./providers/go/... ./internal/workflow/...`, `golangci-lint run` green; `--list-steps` lists both capabilities.

## Phase 3 (Slice 3 — PR3 `feature/runtime-upgrade-workspace`): multi-module `go.work` upgrade

- [x] 3.1 RED `providers/go/toolchain_test.go` — wire A1/A2/A3 `detectConflicts` results (built in Phase 1) into `Upgrade`'s pre-mutation abort path, over `workspace-3-modules`, `divergent-go`, `divergent-toolchain`, `work-go-mismatch` fixtures — spec req: Multi-Module Workspace Consistency.
- [x] 3.2 RED `providers/go/runtimeupgrader_test.go::TestUpgrade_PathEscape` (placed here, not `toolchain_test.go`, because it needs a mock-based "no `WithNewFile` call happened" assertion via daggerkit's mocked `DaggerDirectory` — see the test's own doc comment) — `use ../../etc` fixture aborts before any `WithNewFile` (threat matrix: path traversal via `go.work` `use`) — reject absolute paths and any path escaping workspace root after `filepath.Clean`.
- [x] 3.3 GREEN `providers/go/toolchain.go` — path-escape guard function used by the mutation loop.
- [x] 3.4 GREEN `providers/go/runtimeupgrader.go` — extend `Upgrade` to loop over every `go.work`-referenced module, mutate each, and record a per-module outcome in `UpgradeReport.Modules []ModuleDrift`.
- [x] 3.5 RED `providers/go/runtimeupgrader_test.go::TestUpgrade_Workspace` — all-three-modules-updated scenario; integration-tagged real-engine variant added under `testing/integration/` (`integration` build tag).
- [x] 3.6 Verify: `go test -run TestUpgrade_Workspace ./providers/go/...`, `go test -race ./...`, `golangci-lint run` green; single-module path (Phase 2) still passes unmodified.

## Phase 4 (Slice 4 — PR4 `feature/runtime-upgrade-tidy-validate`): tidy + build validation

- [ ] 4.1 RED `providers/go/runtimeupgrader_test.go` — post-mutation validation failure returns `(nil, err)` with a report describing the failure, never a directory presented as upgraded — spec req: Post-mutation validation failure scenario.
- [ ] 4.2 RED `providers/go/runtimeupgrader_test.go` — one-module-fails-validation-fails-the-whole-operation, report names which module failed/succeeded — spec req: One module's validation failure scenario.
- [ ] 4.3 GREEN `providers/go/runtimeupgrader.go` — container chain per D-7: `From("golang:"+targetVersion)`, mount mutated Directory, per module `go mod tidy` (skip if `tidy: false`), per module `go build ./...` (D-6, no `go vet`), export container workdir as returned Directory.
- [ ] 4.4 GREEN `pkg/shipwright/runtime.go` — extend `UpgradeReport`/`ModuleDrift` with `GoSumChanged bool`, `AddedModules`/`RemovedModules []string` (per-module `go.sum` delta, never the raw diff).
- [ ] 4.5 RED/GREEN `providers/go/runtimeupgrader_test.go` — `validation: "build"` recorded in the report so no consumer is misled (D-6).
- [ ] 4.6 Integration test: real-engine `Upgrade` over a temp workspace under the `integration` build tag, asserting `go build ./...` actually ran.
- [ ] 4.7 Verify: `go test -race ./...`, `make test-integration`, `golangci-lint run`, `make coverage` ≥ 90% local floor all green.

---

*Deviation note*: exceeds the skill's 530-word budget. Cause: the launch brief requires per-slice branch names, base, line estimates, focused test/runtime-harness/rollback evidence per work unit, every applicable threat-matrix row as an explicit RED task, and the two confirmed proposal gaps (`main.go` D-8, `providers/go/daggerkit` D-9) placed in their correct slice — compressed into checklist items, no narrative padding.
