# Exploration: Provider-managed runtime/toolchain upgrade capability

## Current State

Shipwright's core has exactly five capability interfaces (`pkg/shipwright/capabilities.go`): `Builder.Build`, `Tester.Test`, `Artifactor.Publish`, `Deployer.Deploy`, `Runner.Run` — every one is a single-method interface using only Dagger core types. There is no multi-phase (inspect/plan/apply/verify) interface shape anywhere in the codebase.

Provider registration (`internal/workflow/providers/registry.go`, `register.go`) is a flat `Name+Module+Version` table with five `Register*/Resolve*` pairs, each carrying a `WithSchema` for typed `with` validation. Providers are compiled-in only — `security_test.go` statically proves `plugin` (i.e. `plugin.Open`) is unreachable from this package's import graph (design.md D-I anti-supply-chain boundary).

The manifest schema (`internal/workflow/manifest/schema.go`, `validate.go`) enforces `capability` as a closed 5-value allowlist (`build`/`test`/`artifact`/`deploy`/`run`), 1:1 with the five interfaces. **There is no `manual` field on `Step` at all.**

`Environment.Approvals`/`ApprovalSpec` exists in the schema but is confirmed dead metadata: `engine/execute.go`'s package doc states verbatim "This package contains NO blocking, queueing, or 'wait for approval' logic anywhere," and `openspec/specs/workflow-execution/spec.md` (line 10) states the engine "does not enforce approval gates."

No SCM/branch/PR-creation code exists anywhere in the repo — grepped for `CreatePullRequest`/`CreateBranch`/`PullRequest`/`octokit`/`go-github`, zero matches. Only GitHub touchpoint today is credential *sourcing* (`GITHUB_TOKEN` in `internal/pipelines/shared/credentials.go`), not PR/branch creation.

`internal/daggerpin/pin.go` is the closest "parse + compare version across files, fail on mismatch" precedent, but it's a root-module test-only guard (`TestEngineVersionMatchesGoModDaggerVersion`), never wired into the workflow engine as a provider capability.

Any new capability kind also needs a Layer 2 duplication in `.dagger/capabilities.go` (Dagger Interface projection, per design.md D-A/D-B's dual-layer convention).

Live drift already exists today: root `go.mod`, `providers/go/go.mod`, `providers/rust/go.mod`, `go.work` all pin `go 1.26.7`; CI's `.github/workflows/ci.yml` pins `GO_VERSION: '1.26.7'` (5 sites); but `providers/go/gobuilder.go`'s `defaultGoVersion = "1.25.5"` (the actual build-container image version) is already out of sync — a concrete, present-day example of the exact drift category `runtime.verify` is meant to catch. No `toolchain` directive exists anywhere in the repo.

## Affected Areas

- `pkg/shipwright/capabilities.go` — needs a genuinely new interface shape.
- `internal/workflow/providers/registry.go`, `register.go` — new Register*/Resolve* pair or different registration shape.
- `internal/workflow/manifest/schema.go`, `validate.go` — allowlist extension, and (if kept) a new `manual` Step field with real semantics.
- `internal/workflow/engine/execute.go` — new dispatch case, and (if manual gating is real) new blocking logic contradicting the package's current explicit "no blocking logic" design.
- `.dagger/capabilities.go` — Layer 2 projection.
- `providers/go/` (new file) — first concrete implementation, likely reusing `golang.org/x/mod/modfile`.
- `openspec/specs/workflow-execution/spec.md` — its explicit "does not enforce approval gates" line needs a formal `MODIFIED Requirements` delta if gating becomes real.

## Approaches

1. **Single new `RuntimeManager` 4-method capability + Layer 1/Layer 2 dual projection** — Pros: matches existing dual-layer convention, closest to proposal's YAML. Cons: breaks the "one capability = one method" pattern everywhere else; needs schema/validate.go + spec delta. Effort: Medium-High.
2. **Compose from 4 separate existing-shape steps chained via `needs[]`** — Pros: zero new interface shape, reuses registry/dispatch pattern verbatim. Cons: 4 new capability-kind allowlist entries (or phase-branching in core, which risks the "no ecosystem/operation-specific logic in core" principle); more verbose manifest, diverges from proposal's one-step example. Effort: Medium.
3. **Provider-internal 4-stage orchestration behind one `Upgrade` method** — Pros: smallest core diff, matches proposal's one-step YAML. Cons: loses manifest-visible plan-before-apply gate and the "drift detection independent from mutation" requirement without a second method. Effort: Low-Medium.

## Recommendation

None of the three is a clean fit without `sdd-propose` making two explicit scope decisions first, because the raw proposal's two foundational assumptions are false against this codebase: (a) no manual/approval-gating mechanism exists to reuse — it would need to be designed and built; (b) no SCM/PR capability exists to sequence after the language provider — 100% new work. Recommend Approach 2 for capability shape (introduces zero new interface conventions) unless `sdd-design` deliberately wants to set a new multi-phase precedent (Approach 1).

## Risks

- Manual/approval gating doesn't exist — silently absorbing "build the gate" into "first Go slice" changes the size/risk class materially.
- A 6th capability kind touches 5 core files simultaneously (`pkg/shipwright`, `.dagger`, `providers/registry.go`, `manifest/validate.go`, `engine/execute.go`) for a feature meant to be "core owns only orchestration" — needs careful proposal framing.
- SCM/PR flow has zero existing implementation; if left implicit, it silently doubles scope with a second new GitHub-specific adapter contract.
- `providers/go/gobuilder.go`'s `defaultGoVersion` is already drifted from `go.mod`'s `1.26.7` — a live regression target for `runtime.verify` design.
- `openspec/specs/workflow-execution/spec.md`'s explicit "does not enforce approval gates" assertion needs a formal delta, not a silent behavior change, if gating becomes real.

## Ready for Proposal

Yes, contingent on `sdd-propose` explicitly resolving: (1) real engine-level manual gating vs. deferred/external trigger for v1, and (2) SCM/PR adapter in-scope-as-stub vs. fully out-of-scope. Both are currently unaddressed and both materially change task-list size.
