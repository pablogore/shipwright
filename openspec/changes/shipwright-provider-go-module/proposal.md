# Proposal: Extract `internal/capabilities/` into a standalone `providers/go` module

**This change is a fire test, not a refactor.** It asks one question about the contract just merged by `shipwright-public-module-api` (PRs #147–#158): *is `pkg/shipwright` sufficient to implement a real provider from outside the core module?* Exploration already answered it — all five capability implementations and all eight test files import only stdlib, `dagger.io/dagger`, and `pkg/shipwright`, with **zero `internal/**` imports**. This change converts that observation into an enforced, compiler-checked boundary.

## Intent

| | |
|---|---|
| **Problem** | The public contract's sufficiency is asserted by documentation, not proven by the build. Nothing stops a future capability from reaching into `internal/**` and quietly making the contract insufficient. |
| **Why now** | The contract is frozen and stable-from-first-release. Proving it against a real consumer is cheapest immediately after freeze and before other providers multiply the cost. |
| **Success** | `providers/go` builds and tests as its own module against the public contract alone. The `diamond.yaml` end-to-end proof runs identically. Location changed; behavior did not. |
| **Failure is informative** | If this needs *any* change to `pkg/shipwright/**`, the fire test **failed** and the missing abstraction MUST be reported loudly, not patched around. |

## Scope

### In Scope

| # | Deliverable |
|---|---|
| 1 | Move the five source files + eight test files from `internal/capabilities/` to `providers/go/` (move, never duplicate) |
| 2 | `providers/go/go.mod` — own module, requiring only `github.com/pablogore/shipwright` (for `pkg/shipwright`) + `dagger.io/dagger` |
| 3 | Root `go.mod` gains `require github.com/pablogore/shipwright/providers/go vX.Y.Z` at a **real tag** — not a bare local `replace` (D4) |
| 4 | Root `go.work` (`use .`, `use ./providers/go`) so `./...` spans the new module — plus an enforced guard test (D1) |
| 5 | `internal/workflow/providers/register.go` import swap, **atomic with the move** (D3) |
| 6 | New spec requirement on `public-module-api` making external implementability permanent (D5, see Capabilities) |
| 7 | Git tag `providers/go/vX.Y.Z` published as part of shipping this change, so `go install` resolves (D4) — **non-optional** |

### Out of Scope (Non-Goals)

| Non-goal | Boundary |
|---|---|
| Any other provider (container/docker, govulncheck-as-module, nomad, rust) | Follow-up changes, **one at a time**, only after this pattern is proven end-to-end |
| Making `internal/workflow/providers` (`Registry`, `RegisterBuilder`, `WithSchema`, `Values`) public | The real blocker for genuinely out-of-tree third-party providers. Separate, later architectural question. **This change proves an in-repo module boundary only — it does not open the door to external repos.** |
| An ongoing independent **release cadence** for `providers/go` | Narrowed by D4. The **first** tag `providers/go/vX.Y.Z` is now in scope, because `go install` cannot resolve the module without it. What stays out: a recurring release process, a provider changelog, and whether CI/goreleaser automates *future* provider tags |
| **Any change to `pkg/shipwright/**`** | Hard constraint. A required change here means the fire test failed (see Intent) |
| Changing `.dagger/`'s isolation | Preserved exactly. `.dagger/` MUST never appear in `go.work` |

## Decisions

### D1 — `go.work` **is added**, diverging from the `.dagger` precedent

| | |
|---|---|
| **Decision** | Check in a root `go.work` with `use .` and `use ./providers/go`. Never `use ./.dagger`. |
| **Why not mirror `dagger-test`** | The two modules are not analogous. Root **imports** `providers/go` (via `RegisterDefaults`) but never imports `.dagger`. And `.dagger`'s tests need a live engine (`dagger run`) — which is *why* they are excluded from `make test`; `providers/go`'s tests are plain `go test`. |
| **The decisive argument** | Without `go.work`, root `go build ./...` still compiles `providers/go`'s *source* (it is a dependency), so compile errors are caught either way. What is silently lost is its **eight test files** — including the AST golden `naming_test.go` — and `govulncheck` coverage. A change whose entire purpose is proving a contract holds cannot ship its evidence outside CI. |
| **Cost, stated honestly** | `go.work` is a new top-level file that changes resolution semantics for every contributor and alters `go mod tidy` behavior in workspace mode. |
| **Layering that limits the blast radius** | The root `go.mod` `require` handles *resolution* (so `GOWORK=off` and goreleaser checkout builds still work). `go.work` does exactly one job: make `./...` span the new module's own packages and tests. **Amended by D4** — that `require` must point at a real tag, not a `replace`. |
| **Guard (mandatory)** | The accidental-`use ./.dagger` risk is human error, not automatic — a `use` directive is never recursive. It is mitigated by an **enforced root-level test** asserting `.dagger` never appears in `go.work`, not by a comment. |

### D2 — Module path `github.com/pablogore/shipwright/providers/go`

| | |
|---|---|
| **Path** | `github.com/pablogore/shipwright/providers/go`, directory `providers/go/`. In-repo nested module; no external repo split. |
| **Why this path** | Directory name mirrors the manifest provider family (`"go"`, `"go-test"`) and reads consistently with future `providers/rust`, `providers/nomad`. |
| **Known wart** | `package go` is a **syntax error** — `go` is a Go keyword. The package clause MUST differ from the last path element (e.g. `package golang`), and the single importer aliases it: `import golang ".../providers/go"`. |
| **Tradeoff considered** | Renaming to `providers/golang` removes the mismatch but breaks the 1:1 directory↔provider-family naming for the sake of one alias in one file. Rejected. |
| **Verify during apply** | Confirm `revive` (default rules, `.golangci.yml`) does not flag the package-name/import-path mismatch. |

### D3 — The `RegisterDefaults` import swap ships **atomically** with the extraction

Non-negotiable, and not really a fork: deleting `internal/capabilities/` without updating `register.go` leaves the tree uncompilable. Exploration surfaced no reason to defer. Note `register.go` also imports `internal/workflow/interp` and stays in the root module — dependency direction is core→provider, never the reverse.

### D4 — `go install` is a real distribution path, so `providers/go` ships as a **tagged, resolvable version**

Revision 1 filed this as a Low-impact hypothetical risk. It is not one. `go install github.com/pablogore/shipwright@latest` is a real, intended distribution path for this repo, which promotes it from a footnote to a resolved design constraint.

| | |
|---|---|
| **Decision** | The committed root `go.mod` resolves `providers/go` through a `require` on a **real published version**, backed by a git tag. A bare local `replace` MUST NOT be the committed resolution mechanism. |
| **Why `replace` cannot work** | `go install pkg@version` runs with **no main module**: it ignores `go.work` and any `go.mod` in the current or parent directories, and fetches the target module from its source/proxy. A relative `replace ... => ./providers/go` has no checkout to resolve against. It is also rejected outright — the target module's `go.mod` must not contain directives that would make it behave differently than if it were the main module, and `replace`/`exclude` are named. |
| **Sharpened by the current tree** | Root `go.mod` has **zero** `replace` directives today, so `go install` works right now. Adding one would be *this change* breaking it — not a pre-existing gap. |
| **Mechanism (Go's own multi-module convention, not invented)** | A nested module in one repository is published with a **path-prefixed tag**: module `github.com/pablogore/shipwright/providers/go` releases as git tag `providers/go/v0.1.0`, distinct from the root module's own `v0.1.0`. Once that tag exists, `go install`/`go get` and the module proxy resolve it through ordinary module resolution for a public repo. |
| **Relation to D1** | D1 stands unchanged. `go.work` is a workspace-local, development-time mechanism; it is ignored by `go install pkg@version` and by anyone consuming this module as a dependency, so it never participates in published resolution. `replace` MAY remain for local development convenience — the **committed, tagged state must be `go install`-resolvable without it**. |
| **Ordering consequence (for design)** | `providers/go` requires the root module while the root requires `providers/go`. Go permits module-level cycles (package imports stay acyclic), but it forces a tagging order that design/tasks must make concrete. |
| **Deliberately left open** | The exact first version number (`v0.1.0` is the expected default, consistent with the independently-versioned-provider principle in `COMPATIBILITY.md` Axis 5) and whether the tag is cut manually or automated by goreleaser/CI. **The tag requirement itself is decided and non-optional; only its mechanics are deferred.** |

### D5 — External implementability becomes a **permanent spec requirement**, not a one-time success criterion

The fire test must not expire when this change merges. The `public-module-api` spec gains a standing requirement, so a future capability that reaches into `internal/**` fails the spec rather than passing unnoticed. The success-criteria checklist below verifies this change; the spec requirement governs every change after it. Both, not either.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `public-module-api`: add a **permanent requirement** (D5) that the public contract MUST be sufficient to implement every shipped capability from a **separate Go module importing nothing from `internal/**`**. This is the point of the change — a standing, testable property of the contract that outlives this change, not a one-time observation recorded only in a checklist. No other spec changes: provider names, resolution, and execution behavior are byte-identical.

## Approach

1. Add `providers/go/go.mod`; `git mv` the five sources + eight tests.
2. Add the root `require`; add `go.work`; add the `.dagger` guard test. (`replace` is permitted here only as a *local development* aid while the tag does not yet exist — D4.)
3. Swap `register.go`'s import (atomic with step 1) and delete `internal/capabilities/`.
4. Verify no `pkg/shipwright/**` change was needed — if one was, **stop and report the failed fire test**.
5. Re-run the `examples/workflow/diamond.yaml` end-to-end proof as the behavioral regression check.
6. **Cut the `providers/go/vX.Y.Z` tag** and settle the committed root `go.mod` on that version, with no `replace` remaining (D4). Design/tasks own the ordering, the version number, and manual-vs-automated tagging.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `internal/capabilities/**` | **Removed** | Moved to `providers/go/` — no duplicate copies |
| `providers/go/**` | **New** | 5 sources + 8 tests + `go.mod`/`go.sum` |
| `internal/workflow/providers/register.go` | Modified | Import-path swap only; all five registrations identical |
| `go.mod` (root) | Modified | `require` on the new module at a **published tag**; no committed `replace` (D4) |
| `go.work` | **New** | `use .` + `use ./providers/go` — development-time only, never part of published resolution |
| Git tag `providers/go/vX.Y.Z` | **New** | Path-prefixed nested-module tag; required for `go install` to resolve (D4) |
| `.dagger/`, `Makefile` `dagger-test` | Unchanged | Isolation preserved; new guard test protects it |
| `.github/workflows/ci.yml` | Unchanged | `./...` in build/test/security stages now spans `providers/go` via `go.work` — no new step |
| `pkg/shipwright/**` | **Must be unchanged** | Hard constraint |
| `COMPATIBILITY.md` | Optional footnote | Axis 5 already covers the exclusion correctly |

**Dagger compatibility:** `dagger.io/dagger` v0.21.8 unchanged; `providers/go/go.mod` MUST pin the same version to avoid two client versions in one build. No pipeline-step or manifest-schema impact.

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Someone later adds `use ./.dagger`, collapsing the isolation `design.md` D-B established | Med | Enforced guard test (D1), not a comment |
| ~~`go install` cannot resolve the untagged nested module~~ — **resolved into D4, no longer an open risk** | — | Was ranked Low on the assumption that no `go install` path existed. That assumption was wrong. Now a design constraint: `require` at a real tag, no committed `replace` |
| The `providers/go/vX.Y.Z` tag is forgotten, or cut in the wrong order relative to the root release | **High** — it is a manual step outside the normal `git merge` flow, and the two modules require each other | Success criterion below asserts a clean `go install` from a published state; design/tasks must specify the tagging order explicitly rather than leaving it to release day |
| Two `dagger.io/dagger` versions drift between the modules | Med | Pin identically; `go.work` surfaces mismatches at build time |
| Contributors unfamiliar with workspace-mode `go mod tidy` semantics | Med | Document the two-file layering (`replace` = resolution, `go.work` = `./...` span) in the PR description |
| Package-name/import-path mismatch (D2) trips a linter | Low | Verify against `.golangci.yml` during apply |
| **The fire test actually fails** — something turns out to need `internal/**` or a contract change | Low (exploration found zero such imports) | Report loudly as a contract defect. Do NOT widen `pkg/shipwright` quietly to make the extraction pass |

## Rollback Plan

Structural, not behavioral — but not purely additive (it deletes `internal/capabilities/`).

- Extraction + `register.go` swap + `go.work` are **one rollback boundary**. Reverting the move without the import swap leaves the tree uncompilable.
- Single PR: `git revert -m 1 <sha>`. Chained PRs: revert in **reverse merge order**.
- No state, data, config, or release migration; no external consumers. Manifest files are untouched.
- **One-way door introduced by D4:** a published tag is immutable on the Go module proxy. A bad `providers/go` tag is superseded by a new patch tag, never deleted. This is the only part of the change that cannot be reverted by `git revert`.
- Post-revert verification: `go build -o shipwright .` and `go test -race ./...` green, and `diamond.yaml` still resolves.

## Dependencies

- Go ≥ 1.18 for workspaces (repo is on 1.26.1 — satisfied).
- No new third-party dependencies.

## Success Criteria

- [ ] `providers/go/go.mod` exists and depends only on the `pkg/shipwright`-owning module + `dagger.io/dagger` + stdlib — verified via `go list -m all` from inside `providers/go/`
- [ ] `go build ./...` and `go test -race ./...` succeed from within `providers/go/` as a standalone module
- [ ] Root `go build ./...` / `go test -race ./...` succeed, **do** traverse `providers/go` (D1), and still **do not** traverse `.dagger/`
- [ ] A test fails if `.dagger` is ever added to `go.work`
- [ ] `internal/capabilities/**` is deleted — a repo-wide search finds exactly one copy of each of the five types
- [ ] `RegisterDefaults` registers all five under the identical names: `"go"`, `"go-test"`, `"golangci-lint"`, `"govulncheck"`, `"container"`
- [ ] `examples/workflow/diamond.yaml` resolves and runs identically to pre-extraction
- [ ] **`git diff` touches zero files under `pkg/shipwright/`**
- [ ] Coverage ≥ 90% holds across both modules
- [ ] **Committed root `go.mod` contains no `replace` directive** — `go install github.com/pablogore/shipwright@latest` still works (D4)
- [ ] Git tag `providers/go/vX.Y.Z` exists, and `go install github.com/pablogore/shipwright@<root-tag>` resolves `providers/go` from a clean environment with no local checkout

---

*Revision 2 (targeted amendment):* the user confirmed `go install github.com/pablogore/shipwright@latest` is a real intended distribution path. That single answer promoted a Low-ranked footnote risk into **D4** (tagged nested module, no committed `replace`) and added a non-optional tagging step to scope. **D5** was recorded explicitly to remove any ambiguity that external implementability is a standing spec requirement, not just this change's checklist item. D1–D3 are unchanged; D1's layering row was amended only on its `replace` half.

*Deviation note:* exceeds the skill's 450-word budget. Cause: the change turns on decisions the orchestrator explicitly required to be justified rather than defaulted (D1–D5), and D1 diverges from an established in-repo precedent, which cannot be recorded credibly without its reasoning. Content is compressed into tables.
