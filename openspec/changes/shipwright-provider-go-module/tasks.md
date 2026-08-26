# Tasks: Extract `internal/capabilities/` into standalone `providers/go` module

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~310-370 slice 1, ~150-250 slice 1b (workflow YAML + test-only guard package), ~30-50 slice 2 (excludes generated `go.sum`/`go.work.sum`) |
| 400-line budget risk | Medium — each slice independently stays well under 400; slice 1b is `.github`-only + one small Go package, so it does not meaningfully raise any single PR's line count |
| Chained PRs recommended | Yes — design-mandated: 3 chained PR slices (D6). Slice 1b must merge before the first provider tag exists, so it sits between slice 1 and the tag; slice 2 cannot start earlier |
| Suggested split | PR 1 (slice 1) -> PR 1b (slice 1b, release automation) -> tag interlude (manual) -> PR 2 (slice 2) |
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
| 1b | Release automation (D6): fix `release.yml`'s 3 unfiltered `git describe` calls, `.goreleaser.yml` ignore_tags, new `release-provider-go.yml`, `internal/releaseguard/tags_test.go` guard | PR 1b (base `develop`, after PR 1 merges) | `go test -race ./internal/releaseguard/...` | N/A — GitHub Actions workflow only runs on a real `providers/go/v*` tag push; local guard test is the proof, no engine needed | Independent revert; touches only `.github/`, `.goreleaser.yml`, one test file |
| tag | Cut `providers/go/v0.1.0` on PR 1's merge sha, after PR 1b is merged | manual, not a PR | `GOPROXY=direct go list -m .../providers/go@v0.1.0` | N/A — VCS-only; push triggers slice 1b's workflow, which is the runtime harness for the tag itself | Immutable once published; superseded by `v0.1.1`, never deleted |
| 2 | Drop root `replace`, tag-based resolution, `go install` acceptance | PR 2 (base `develop`, after PR 1 + PR 1b + tag) | `go test -race ./...` (no-replace guard) | `go install github.com/pablogore/shipwright@<tag>` from clean `GOMODCACHE` | Independent revert to `replace` state |

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

## Phase 4: Slice 1b — Release guard RED (TDD)

- [ ] 4.1 RED `internal/releaseguard/tags_test.go`: table-test (a) no `git describe --tags` call in `release.yml` lacks `--match`, (b) `release.yml`'s tag globs don't match `providers/go/v0.1.0`, (c) `release-provider-go.yml`'s globs don't match `v1.2.3`, (d) the shape regex extracted from the workflow accepts `providers/go/v0.1.0` and rejects `providers/go/v2.0.0`, `providers/go/v01.0.0`, `v0.1.0` — fails closed: (a) on the unfiltered `git describe` calls in `release.yml`, (c)/(d) on the missing `release-provider-go.yml`

## Phase 5: Slice 1b — Release automation (GREEN)

- [ ] 5.1 Add `--match 'v[0-9]*'` to the three `git describe --tags --abbrev=0` calls in `.github/workflows/release.yml` (auto-bump ~L162, dispatch-bump ~L220, changelog range ~L245)
- [ ] 5.2 Add `git: ignore_tags: ['providers/*']` to `.goreleaser.yml`'s previous-tag lookup
- [ ] 5.3 Create `.github/workflows/release-provider-go.yml`: `on: push: tags: ['providers/go/v*']`, shape-validation step (`^providers/go/v(0|1)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$`, major restricted to `0|1` so it rejects >= 2 in the same regex, naming the `/vN` module-path rule), and module-identity step (`providers/go/go.mod`'s `module` line == `github.com/pablogore/shipwright/providers/go`)
- [ ] 5.4 Add isolation step to `release-provider-go.yml`: `cd providers/go && GOWORK=off go build ./... && GOWORK=off go test -race -short ./...` at the tag
- [ ] 5.5 Add path-scoped changelog step (`git log --pretty='- %s (%h)' <prev providers/go tag>..<tag> -- providers/go/`) and release step (`gh release create "$TAG" --title "providers/go $VERSION" --notes-file … --latest=false`) to `release-provider-go.yml`
- [ ] 5.6 Add proxy-visibility gate step to `release-provider-go.yml`: `curl -sf https://proxy.golang.org/github.com/pablogore/shipwright/providers/go/@v/$VERSION.info`
- [ ] 5.7 GREEN: re-run 4.1

## Phase 6: Slice 1b — Verification & Merge

- [ ] 6.1 `go test -race ./internal/releaseguard/...` green; `golangci-lint run` clean on the new package
- [ ] 6.2 Validate workflow YAML (`actionlint` or equivalent, if available) on `release-provider-go.yml` and the modified `release.yml`
- [ ] 6.3 Merge slice 1b to `develop` — branch pushed and PR opened per repo delivery convention (stacked-to-main chain, base `develop` after PR 1); actual merge requires human/reviewer action, not performed by this agent

## Phase 7: Tag interlude (manual)

- [ ] 7.1 `git tag providers/go/v0.1.0 <slice-1 merge sha> && git push origin providers/go/v0.1.0` — this push triggers slice 1b's `release-provider-go.yml`
- [ ] 7.2 Confirm the triggered workflow run is green: shape, identity, isolation build/test, changelog, `gh release create`, and the proxy-visibility gate all pass
- [ ] 7.3 Confirm resolution: `GOPROXY=direct go list -m github.com/pablogore/shipwright/providers/go@v0.1.0`; if repo visibility is unconfirmed, also `curl -s https://proxy.golang.org/github.com/pablogore/shipwright/@v/list` and report loudly if empty

## Phase 8: Slice 2 — No-`replace` guard (TDD)

- [ ] 8.1 RED: root test asserting `modfile.Parse(go.mod).Replace` is empty (fails: `replace` still present)
- [ ] 8.2 Delete `replace` from root `go.mod`; keep `require .../providers/go v0.1.0`
- [ ] 8.3 `GOWORK=off go mod tidy` at root; commit regenerated `go.sum`
- [ ] 8.4 GREEN: re-run 8.1

## Phase 9: Slice 2 — Acceptance & regression

- [ ] 9.1 `go install github.com/pablogore/shipwright@<tag>` from clean `GOMODCACHE`
- [ ] 9.2 Run `examples/workflow/diamond.yaml` before/after; confirm identical resolved providers
- [ ] 9.3 Confirm coverage >= 90% across both modules (`make coverage`)
- [ ] 9.4 Confirm `.dagger` guard fails when `use ./.dagger` is added (covered by 1.1's synthetic `t.TempDir()` cases)

## Threat Matrix

`N/A` for routing, shell command, subprocess, executable-file classification,
and process-integration boundary changes — none of those apply.

**VCS/PR automation: `Applicable` since D6.** The tag-reacting workflow only
reacts to a human-pushed ref (never creates/moves/deletes one), holds
job-level `permissions: contents: write` only, and runs `go build`/`go test`
of the tagged tree with no other code execution. RED coverage: Phase 4's task
4.1 (`internal/releaseguard/tags_test.go` assertions a-d) is the only RED task
required for this row.

---

*Deviation note:* exceeds the skill's 530-word budget. Cause: the design
mandates three chained slices (D6 added slice 1b) with a manual tag interlude
between slice 1b and slice 2, three new guard packages, and a 13-file move
plus a 3-file importer swap — matching the same complexity that already
forced `proposal.md` and `design.md` past their own budgets. Content stays
checklist-only with no narrative padding.
