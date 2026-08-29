# Proposal: Provider-Managed Runtime/Toolchain Upgrade (Go First)

## Intent

Toolchain versions drift silently across a repository's declarative sources.
Live proof here: `go.mod`, `go.work` and CI pin `go 1.26.7`, while
`providers/go/gobuilder.go` still pins `defaultGoVersion = "1.25.5"`. Nothing
detects or fixes this. Give each language provider ownership of discovering,
upgrading and verifying its own toolchain version — with zero
ecosystem-specific mutation logic in the core engine.

## Scope

### In Scope

- Two single-method capability kinds: `runtime-inspect` (read-only drift
  report) and `runtime-upgrade` (discover → mutate → validate), projected in
  Layer 1 (`pkg/shipwright`) and Layer 2 (`.dagger`).
- Manifest allowlist entries, provider registry pairs, engine dispatch for both.
- Go provider: `go.mod` `go`/`toolchain` directives, `go.work`, declarative
  build/runtime image pins, `go mod tidy` normalization, multi-module workspace
  awareness, post-mutation validation.
- Discovery-driven: mutate only locations that actually exist. Fail closed on
  absent, ambiguous, or conflicting version sources — never guess.
- Mutation output is a returned workspace directory plus a structured report.
  No network, no git, no push.

### Out of Scope

- Java/Rust/Python providers (Go only in this slice).
- A generic Renovate/Dependabot-style bot; application dependency (`go.sum`)
  upgrades.
- Maven/Gradle/Cargo/Poetry/npm coupling in core.
- SCM branch/PR creation (see D2); scheduled or webhook triggering.
- Mutating arbitrary source-code constants or CI workflow YAML.

## Decisions

**D1 — Manual gating: operator-invoked trigger, no engine blocking primitive.**
`engine/execute.go` contains zero blocking logic by explicit design, and
`workflow-execution` requires approval gates stay declared metadata. Building a
real gate is precedent-setting core work that competes with the actual
deliverable. "Manual, not automatic" is met by shipping no scheduler or webhook
trigger. Consequence: `workflow-execution` needs **no** MODIFIED delta.
Tradeoff: no in-engine enforcement; safety comes from D2 instead.

**D2 — SCM/PR adapter: fully out of scope, deferred to a follow-up change.**
No SCM code exists today; building one alongside the Go provider cannot fit the
review budget. "Never push to a protected branch" is satisfied *structurally* —
the capability has no code path that can reach a remote. A follow-up change
consumes the returned directory and report. Tradeoff: v1 stops before PR
creation.

## Capabilities

### New Capabilities

- `runtime-toolchain`: read-only toolchain drift inspection and provider-owned
  upgrade of declarative toolchain metadata.

### Modified Capabilities

- `workflow-manifest`: the `capability` allowlist is no longer exactly five
  (spec.md:42–46).
- `public-module-api`: the exported surface is no longer exactly five capability
  interfaces (spec.md:28).
- `workflow-execution`: **none** — the approval-gate requirement is preserved
  verbatim per D1.

## Approach

Preserve the repository's one-capability-one-method invariant: two interfaces,
not one four-method `RuntimeManager`. This makes read-only versus mutating a
*type-level* boundary — a manifest declaring only `runtime-inspect` cannot reach
mutation code. Plan/apply/verify staging lives inside the provider, keeping core
purely orchestrational. `sdd-design` owns the final signatures.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `pkg/shipwright/capabilities.go` | Modified | +2 interfaces and result types |
| `.dagger/capabilities.go` | Modified | Layer 2 projection |
| `internal/workflow/providers/registry.go`, `register.go` | Modified | 2 Register/Resolve pairs |
| `internal/workflow/manifest/schema.go`, `validate.go` | Modified | Allowlist 5 → 7 |
| `internal/workflow/engine/execute.go` | Modified | 2 dispatch cases, no blocking logic |
| `providers/go/` | New | Go toolchain inspector and upgrader |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| 400-line single-PR budget exceeded | High | `sdd-tasks` must forecast; if High, recommend a 2-slice chain (inspect first, upgrade second) |
| Capability count 5 → 7 touches 5 core files | Medium | Additive only; no existing behavior modified; both kinds follow the existing single-method pattern |
| Ambiguous or conflicting version sources drive a wrong mutation | Medium | Fail closed: report and abort, never guess or partially mutate |
| `go mod tidy` alters `go.sum` beyond toolchain intent | Medium | Tidy runs only after directive mutation; diff is reported |
| Mutated directory consumed without review | Low | No remote-write path exists in v1 (D2) |
| Dagger SDK / pipeline-step compatibility | Low | No new Dagger core types; both interfaces use the existing `*dagger.Directory` |

## Rollback Plan

Revert the commit. The change is additive: no existing capability, manifest, or
engine path changes behavior, and removing the two allowlist entries restores
the prior closed five-value set. No persisted state or migration is involved.

## Dependencies

- `golang.org/x/mod/modfile` for `go.mod` / `go.work` parsing.
- `internal/daggerpin/pin.go` as the existing version-comparison precedent.

## Success Criteria

- [ ] `runtime-inspect` on this repository reports the live `1.26.7` vs `1.25.5`
      drift while mutating nothing.
- [ ] `runtime-upgrade` updates `go.mod` `go`/`toolchain` and `go.work` across
      all workspace modules.
- [ ] Ambiguous or conflicting version sources abort with an explicit error and
      no partial mutation.
- [ ] The engine gains no blocking/approval code (D1) and no SCM/git dependency
      (D2).
- [ ] `go test -race ./...` passes, coverage ≥ 90%, `golangci-lint run` clean.
