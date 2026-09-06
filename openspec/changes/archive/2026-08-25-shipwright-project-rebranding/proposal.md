# Proposal: Shipwright Project Rebranding

> SPEC-000 · "Same product. Same behavior. New identity: Shipwright."

## Intent

The product ships under an inconsistent identity: `go.mod` declares `github.com/getsyntegrity/syntegrity-dagger` while the real remote is `github.com/pablogore/syntegrity-dagger`, and 866 case-insensitive matches of the `syntegrity[-_ ]?dagger` name span 97 files (module path, binary, env vars, config filename, CI, docs, examples, tests). The name also collides conceptually with the third-party Dagger engine it wraps. Rebranding to **Shipwright** gives one unambiguous public identity with zero functional change.

A separate, real external identity must not be confused with the old product name: `getsyntegrity` is an actual GitHub org that publishes `go-kit-logger` (a genuine dependency, imported in 9+ files), owns the `getsyntegrity.com` email domain, and appears in unrelated `eventengine` code examples in `AGENTS.md`. Bare `syntegrity` (without `dagger`) in those contexts is the company's own identity, not this product's — it stays untouched.

## Scope

### In Scope

| Category | Change |
|---|---|
| Go module | `go.mod` → `github.com/pablogore/shipwright`; update imports in 56 `.go` files |
| CLI / binary | `Makefile` `BINARY_NAME`, `main.go` flag defaults, version string, help/usage text |
| Config & env | `SYNTEGRITY_DAGGER_*` → `SHIPWRIGHT_*`; `.syntegrity-dagger.yml` → `.shipwright.yml` |
| CI/CD & release | `.goreleaser.yml`, `ci.yml`, `release.yml`, `.github/actions/syntegrity-dagger/` (dir rename), `dependabot.yml`, `CODEOWNERS`, `rulesets/README.md` |
| Docs | `README.md` (incl. badge URL), 21 files under `docs/`, `CHANGELOG.md` |
| Examples | `examples/` GitHub Actions, Jenkins, local, configs, Go samples |
| Tests & mocks | `internal/**/*_test.go`, `internal/plugins/mocks*.go`, `tests/`, fixtures |
| Comments | Old-identity mentions across `internal/` |

**Resolved module path:** `github.com/pablogore/shipwright` (matches the actual git remote owner; the `getsyntegrity` line in `go.mod` is stale).

### Out of Scope

- `dagger.io/dagger` SDK imports (~19 files) — genuine third-party dependency. **Never touched.** No Dagger SDK version or pipeline-step compatibility impact.
- `Makefile` `grep -E "gitlab.com/syntegrity"` coverage filter (~lines 185, 236, 238–240, 274) — dead reference to a former GitLab module path that no longer matches `go.mod`; not this product's identity. Follow-up, not this change.
- Root-level stray `1export` file — accidental shell-env dump, unrelated. Untouched.
- Backward-compatibility aliases or deprecation shims for the old identity.
- Pipeline redesign, lifecycle engine, Git Flow, Rust/Java support, overrides system, Artifact Model, any new capability, or non-essential refactoring.
- **Real external company/org identity (`getsyntegrity`).** Never touched: `github.com/getsyntegrity/go-kit-logger` (`go.mod`, `go.sum`, all importing `.go` files), the unrelated `eventengine` examples in `AGENTS.md`, the `getsyntegrity.com` email domain and `"Syntegrity CI"` git-author default in `internal/pipelines/shared/{ssh,https}_cloner.go`, the `$HOME/.ssh/syntegrity` default key path in `ssh_cloner.go`, and the `ghcr.io/syntegrity` example registry namespace in `examples/configs/tenant-svc.yml`. This is the company's own identity, distinct from the product being renamed.

## Capabilities

### New Capabilities
- `product-identity`: canonical public naming contract — module path, binary name, env-var prefix, default config filename, and release/action identifiers.

### Modified Capabilities
- None. `openspec/specs/` holds no domain specs yet; no existing requirement changes.

## Approach

Mechanical, exhaustive find-and-replace over the enumerated categories, ordered: module path + imports → build/CLI → env/config → CI/release → docs/examples → tests/comments. Each variant form (`syntegrity-dagger`, `SYNTEGRITY_DAGGER`, `Syntegrity Dagger`) maps to one Shipwright form. Behavior, package layout, and control flow stay byte-equivalent apart from identifiers and strings. Green build + full test suite is the acceptance signal.

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| External consumers break (CI badge, install curl URLs, composite action env vars) | Med | **Decision made: clean rename, no aliases.** No concrete consumer evidence exists (no pkg.go.dev badge, no registry refs). If evidence surfaces, handle as a separate compat follow-up. |
| Missed residue in an unscanned file | Med | Final repo-wide case-insensitive sweep for `syntegrity` excluding the two documented exclusions |
| Over-replacing `dagger` and breaking SDK imports | Low | Replace only old-identity token forms; never bare `dagger` |
| Stale `gitlab.com/syntegrity` grep silently drops coverage | Low | Documented as pre-existing follow-up; verify coverage output unchanged |

## Rollback Plan

Single-purpose branch with no functional edits — revert the merge commit (`git revert -m 1 <sha>`) to restore the old identity wholesale. If a published release already carries the new name, cut a corrected tag; no data or state migration is involved.

## Dependencies

- Confirm the GitHub repository is renamed to `pablogore/shipwright` (or that the module path is reachable) before or with merge, so `go get` and release URLs resolve.

## Success Criteria

- [ ] Repo-wide case-insensitive search for the old identity returns zero hits outside the two documented exclusions
- [ ] `go build` and `go test -race ./...` pass; coverage gate unchanged
- [ ] Binary, `--help`, and version output present only "Shipwright"
- [ ] `SHIPWRIGHT_*` env vars and `.shipwright.yml` are the only documented config surface
- [ ] `dagger.io/dagger` imports and pipeline behavior are provably unchanged

## Atomicity Confirmation

This change is strictly atomic: it renames the product's public identity and nothing else. It introduces no capability, alters no behavior, changes no architecture, and performs no opportunistic refactoring. Any diff hunk that changes something other than an identity string, identifier, path, or filename is out of scope for SPEC-000.
