# Tasks: Extract `internal/capabilities/` into standalone `providers/go` module

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~310-370 slice 1, ~30-50 slice 2 (excludes generated `go.sum`/`go.work.sum`) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes — design-mandated: the tag cannot exist before slice 1 merges, so slice 2 cannot start earlier |
| Suggested split | PR 1 (slice 1) -> tag interlude (manual) -> PR 2 (slice 2) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Extract `providers/go` module, guards, importer swap | PR 1 | `go test -race ./... && GOWORK=off go build ./...` | `examples/workflow/diamond.yaml` run, pre/post compare | `git revert -m 1` (atomic: move + swap + `go.work`) |
| tag | Cut `providers/go/v0.1.0` on PR 1's merge sha | manual, not a PR | `GOPROXY=direct go list -m .../providers/go@v0.1.0` | N/A — VCS-only, no code executed | Immutable once published; superseded by `v0.1.1`, never deleted |
| 2 | Drop root `replace`, tag-based resolution, `go install` acceptance | PR 2 (base `develop`, after PR 1 + tag) | `go test -race ./...` (no-replace guard) | `go install github.com/pablogore/shipwright@<tag>` from clean `GOMODCACHE` | Independent revert to `replace` state |

## Phase 1: Slice 1 — Guards RED (TDD)

- [x] 1.1 RED `internal/workspaceguard/work_test.go`: assert `go.work` use-set == `{".", "./providers/go"}`, reject any `.dagger`/`dagger` path segment, fail closed on missing/unparseable `go.work`
- [x] 1.2 RED `providers/go/internalimport_test.go`: `parser.ParseDir` incl. `_test.go`, fail on any `internal/**` import
- [x] 1.3 RED `internal/daggerpin/pin_test.go`: add `TestProvidersGoDaggerVersionMatchesRoot` against `providers/go/go.mod`

## Phase 2: Slice 1 — Extraction (GREEN)

- [x] 2.1 Create `providers/go/go.mod` (`go 1.26.1`; require `dagger.io/dagger v0.21.8` + shipwright pre-extraction pseudo-version, D4)
- [x] 2.2 Implement `internal/workspaceguard/work.go` via `modfile.ParseWork` (GREEN 1.1)
- [x] 2.3 Create `go.work` (`use .` + `use ./providers/go`, `go 1.26.1`)
- [x] 2.4 `git mv` 5 sources + 8 tests, `internal/capabilities/` -> `providers/go/`; edit package clause/doc to `package golang`
- [x] 2.5 Edit `naming_test.go`: `pkgs["capabilities"]` -> `pkgs["golang"]` + doc refs
- [x] 2.6 `cd providers/go && GOWORK=off go mod tidy` (GREEN 1.2, 1.3)
- [x] 2.7 Root `go.mod`: add `require .../providers/go` + temporary `replace => ./providers/go` (D4 sanctioned interim)
- [x] 2.8 Swap importers: `internal/workflow/providers/register.go`, `internal/app/container.go`, `internal/app/container_capabilities_test.go` (import alias `golang`, type refs)
- [x] 2.9 Delete `internal/capabilities/`; confirm exactly one copy of each of the 5 types remains
- [x] 2.10 Update `COMPATIBILITY.md` line 46: `internal/capabilities/**` -> `providers/go/**`

## Phase 3: Slice 1 — Verification & Merge

- [x] 3.1 `go test -race ./...` and `GOWORK=off go build ./...` both green
- [x] 3.2 Verify `git diff --stat -- pkg/shipwright/` is empty
- [x] 3.3 Verify CI `setup` stage (`go mod download`/`verify`) under workspace mode; prefix `GOWORK=off` only if it fails
- [ ] 3.4 Merge slice 1 to `develop` — branch pushed and PR opened per repo delivery convention (stacked-to-main chain); actual merge requires human/reviewer action, not performed by this agent

## Phase 4: Tag interlude (manual)

- [ ] 4.1 `git tag providers/go/v0.1.0 <slice-1 merge sha> && git push origin providers/go/v0.1.0`
- [ ] 4.2 Confirm resolution: `GOPROXY=direct go list -m github.com/pablogore/shipwright/providers/go@v0.1.0`
- [ ] 4.3 If repo visibility is unconfirmed: `curl -s https://proxy.golang.org/github.com/pablogore/shipwright/@v/list`; report loudly if empty

## Phase 5: Slice 2 — No-`replace` guard (TDD)

- [ ] 5.1 RED: root test asserting `modfile.Parse(go.mod).Replace` is empty (fails: `replace` still present)
- [ ] 5.2 Delete `replace` from root `go.mod`; keep `require .../providers/go v0.1.0`
- [ ] 5.3 `GOWORK=off go mod tidy` at root; commit regenerated `go.sum`
- [ ] 5.4 GREEN: re-run 5.1

## Phase 6: Slice 2 — Acceptance & regression

- [ ] 6.1 `go install github.com/pablogore/shipwright@<tag>` from clean `GOMODCACHE`
- [ ] 6.2 Run `examples/workflow/diamond.yaml` before/after; confirm identical resolved providers
- [ ] 6.3 Confirm coverage >= 90% across both modules (`make coverage`)
- [ ] 6.4 Confirm `.dagger` guard fails when `use ./.dagger` is added (covered by 1.1's synthetic `t.TempDir()` cases)

## Threat Matrix

N/A — design.md records no routing, shell command, subprocess, executable-file
classification, or process-integration boundary changes. No additional RED
tasks required beyond Phase 1.

---

*Deviation note:* exceeds the skill's 530-word budget. Cause: the design
mandates two chained slices with a manual tag interlude, two new guard
packages, and a 13-file move plus a 3-file importer swap — matching the same
complexity that already forced `proposal.md` and `design.md` past their own
budgets. Content stays checklist-only with no narrative padding.
