# Proposal: Public Versioned Module API + Composition Model + Declarative Workflow Orchestration

> **Revision note — REVISION 2, SUPERSEDES the merged SPEC-002 + SPEC-003 proposal.** After the user reviewed `design.md`, scope expanded a third time. Three things changed: the `go-service` named preset is removed outright, the public `Pipeline` type is renamed, and a declarative workflow/DAG orchestration layer becomes in-scope work for this change. Scope, size, review burden, and correctness risk all grow again as a direct, accepted consequence.

**Guiding principle (updated):** "Capabilities compose into workflows; workflows form execution graphs. The SDK ships capabilities, never named pipelines."

The prior wording — "capabilities compose into pipelines" — is withdrawn. It kept the word `Pipeline` at the center of the mental model, and that is exactly what let a named preset survive the last design round.

## Intent

Two problems, one contract.

**Problem 1 — the original one (unchanged).** Shipwright's public surface is one monolithic `Pipeline` interface, documented in `docs/API.md` and `docs/ARCHITECTURE.md` as the canonical extension point, with implementations named per end-to-end stack (`go-service`, `infra`). That scales multiplicatively: N build tools × M deploy targets × K runtimes forces N·M·K named pipelines, each written, tested, versioned and documented separately, and no consumer can assemble an unanticipated combination.

**Problem 2 — found in design review, and the reason for this revision.** The approved `design.md` states the principle "the SDK ships capabilities, not pipelines" and then contradicts it in the migration layer:

| Location | Text | Why it is a defect |
|---|---|---|
| `design.md:123` (Data Flow) | `main.go --pipeline go-service ──► pipelines.Registry (capability-set presets)` | Reintroduces a registry of *named presets* — the exact anti-pattern this change exists to delete |
| `design.md` (Migration Sequence) | "`pipelines.Registry` registers capability-set factories, keeping `--pipeline go-service` CLI UX intact, so there is no release-level migration" | Justifies the preset by compatibility |
| `tasks.md:74` (WU6 / task 6.1) | "capability-set factories preserve `--pipeline go-service` CLI UX, no release-level migration" | Encodes the preset as an acceptance test |

`go-service` is a named pipeline preset. Preserving it under a CLI-UX-compatibility argument is not defensible here, because **the repository has zero external consumers** — an already-established fact of this change. There is nothing to stay compatible with. The compatibility argument was carrying the anti-pattern, not a real requirement.

**Why the DAG layer now.** Deleting the preset removes the only mechanism a user had for saying "do these steps in this order." Composition via chained Go calls replaces it for Go consumers only. A declarative manifest is what makes composition usable from any language and reviewable as data — and building it after the contract freezes means freezing a contract that never met its real consumer.

**Success looks like:** a consumer declares a workflow in YAML — steps naming a *capability* (the contract) and a *provider* (the implementation), wired by explicit dependency edges into a graph — and Shipwright executes it, without the engine knowing what "Maven" or "Tomcat" is, and without any named preset existing anywhere.

## Scope

### In Scope

Deliverables 1–8 are unchanged from Revision 1. Deliverables 9–11 are new.

| # | Deliverable | Status |
|---|---|---|
| 1 | Public-API contract fixing four properties: **versioned**, **composable**, **cross-language consumable**, **representable by Dagger's public type system** | Unchanged |
| 2 | Capability decomposition as a contract property: small orthogonal capabilities (Build, Deploy, Run, Test, Artifact) | Unchanged |
| 3 | The composition mechanism decision itself, applied in code | Unchanged |
| 4 | **Hard supersede** of `internal/pipelines/pipeline.go` and `internal/interfaces/interfaces.go`; **deletion** of `internal/pipelines/common/interfaces.go` | Unchanged |
| 5 | Migration of `go-service` and the DI/plugin wiring onto the new contract | **Amended by D3** |
| 6 | Versioning + **stable-from-first-release** compatibility policy | Unchanged |
| 7 | Dagger-module wiring sufficient to generate and verify **cross-language SDK bindings in a second language** | Unchanged |
| 8 | Minimum doc correction: `docs/API.md` / `docs/ARCHITECTURE.md` stop presenting `Pipeline` as canonical | Unchanged |
| 9 | **Removal of the `go-service` named preset**: no `--pipeline go-service` CLI path, no registry of capability-set presets, no bundling identity naming "the Go pipeline" | **NEW (D3)** |
| 10 | **Rename of the public `Pipeline` composition type** to a name that does not invite named-preset regressions | **NEW (D4)** |
| 11 | **Declarative workflow layer**: YAML manifest schema + DAG parser/validator (cycle detection, dependency resolution, topological ordering) + provider-resolution layer (capability→provider, including external `module:` references) + execution engine wired to the composition contract | **NEW (D5)** |

**Mandatory constraint (verbatim, canonical wording) — binds every public contract element, including the workflow layer and any external provider it resolves:**

> Dagger type-system constraint: Any public composition contract defined by this change MUST be representable by Dagger's module type system and consumable through generated cross-language SDK bindings. Language-specific implementation mechanisms, including Go generics, MUST NOT be required as part of the public contract.

### Out of Scope (Non-Goals)

The DAG layer's illustrative examples name providers (`maven`, `docker`, `tomcat`, `cargo`, `kubernetes`) to **prove the schema shape**. They are not commitments to ship those adapters.

| Non-goal | Boundary |
|---|---|
| Concrete capability adapters beyond the minimum needed to demonstrate the DAG engine — no production-grade Java/Gradle/Maven/Ant/Tomcat/K8s/SSH adapter library | Follow-up change (boundary unchanged, now explicitly restated against the YAML examples) |
| Full policy engine: remote policy-as-code integration, approval-workflow UI, notification systems | Follow-up. Only `forbidCycles`, `requireVersion`, `forbidPlaintext` ship, and they ship **enforced**, not documented |
| CI-system integration (GitHub Actions / GitLab CI triggers) | Never in this change. This is a local/programmatic execution engine, not a hosted CI service |
| A package/module registry service for `module:`-style external providers | Follow-up. Resolution assumes local/vendored providers or an existing mechanism — no new registry service |
| Full module productionization: registry publication, CI `dagger call` wiring, migrating `main.go` off `flag` | Follow-up change |
| Full rewrite of `docs/API.md` / `docs/ARCHITECTURE.md` | Fast-follow (minimum correction is in scope) |
| Compatibility guarantees for internal, non-public Go packages | Never — the guarantee covers the public contract only |

## Decisions

### D1 — Legacy `Pipeline` interfaces: hard supersede *(unchanged)*

| Interface | Disposition | Consequence |
|---|---|---|
| `internal/pipelines/common/interfaces.go` | **Deleted** | Dead code, no consumers found |
| `internal/pipelines/pipeline.go` | **Retired / replaced** | `go-service` decomposes into standalone capabilities |
| `internal/interfaces/interfaces.go` | **Retired / replaced** | DI container, step registry, hook manager, plugin layer migrate |

### D2 — "Versioned" means a stable contract from the first release *(unchanged)*

| Element | Position |
|---|---|
| Guarantee | SemVer-style backward-compatibility guarantee **effective from the first release**. No v0 escape hatch |
| Breaking change | Explicit major bump plus a written migration note |
| Version identity | Machine-readable marker at the contract boundary |
| Separation | Contract version / CLI binary SemVer / `dagger.json` engine pin MUST stay three distinct axes |
| Scope | Public contract only. Internal packages carry no guarantee |

*Honest tradeoff (unchanged, do not soften):* zero external consumers today, so we guarantee a shape nobody has used in anger, constraining iteration speed. Accepted because a "may break anytime" contract cannot be adopted cross-team or cross-language, and because the absence of a policy is what produced three divergent `Pipeline` interfaces. **Mitigation:** keep the guaranteed surface deliberately minimal.

### D3 — NEW: the `go-service` named preset is removed, not migrated

| Item | Position |
|---|---|
| `--pipeline go-service` CLI flag | **Removed.** Not preserved for compatibility |
| `pipelines.Registry` as a registry of capability-set presets | **Removed.** No preset registry ships |
| `go-service` decomposition output | Standalone, independently composable capability implementations with **no bundling identity** |
| Naming rule | Nothing may name "the Go pipeline" as a thing. Implementations name what they *do* (build Go, test Go, publish a container image), never a stack bundle |

Exact implementation names are a design-phase concern. Illustrative only: `GoBuilder`, `GoTester`, `DockerArtifactor`.

*Why the compatibility argument fails:* it is real only when a consumer exists. Zero external consumers is already established for this change, so the flag protects nothing and costs the change its stated principle. This supersedes the Revision-1 position that `go-service` merely "migrates" — migrating a preset preserves the preset.

### D4 — NEW: rename the public `Pipeline` composition type

| Item | Position |
|---|---|
| Current type | The design's `Pipeline` struct is **already** a generic composition result, built by explicit `.WithBuild().WithTest()...` calls — not a named preset |
| Decision | Rename it anyway (e.g. `Plan`, or `Compose`; the design phase picks the exact name) |
| Rationale | **Preventive, not corrective.** The word "Pipeline" in the public surface invites `GoPipeline` / `JavaPipeline` / `RustPipeline` regressions. The last design round shows the pull is real, not hypothetical |

This distinction was surfaced explicitly — the type is not itself defective — and the rename was chosen regardless. Recorded here so the spec and design phases do not re-litigate it as a bug report.

### D5 — NEW: declarative workflow manifest + DAG execution engine

**The core architectural rule, first-class:**

> **`capability` is the contract. `uses` / `provider` is the implementation.**

A step declares `capability: build` (satisfying the `Builder` interface this change already defines) and `uses: {provider: maven, version: "1"}` (resolving to a concrete `Builder`). Shipwright's engine never needs to know what "Maven" is — it verifies only that the resolved provider implements the declared capability. Substitutability is the already-decided Dagger Interfaces mechanism (D-A in `design.md`), and the Dagger type-system constraint above extends unchanged to third-party providers referenced as `module: github.com/acme/custom-builder, version: v3.2.1`.

**Manifest shape (illustrative and non-normative — exact field names are a spec/design decision):**

| Section | Purpose |
|---|---|
| `apiVersion` / `kind` / `metadata` | Versioned document identity (`shipwright.dev/v1`, `kind: Workflow`), plus name / description / labels |
| `spec.source` | Pluggable source provider (repo, ref, auth secret reference) |
| `spec.variables` | Named, interpolatable values (`${{ variables.x }}` style) |
| `spec.secrets` | Named secret references resolved from env/external sources. **Referenced by name, never embedded as plaintext** — ties directly to the already-decided `*dagger.Secret` retyping |
| `spec.steps[]` | `id`, `capability` (contract), `uses` (provider + pinned version, or external `module:` reference), `needs[]` (**explicit DAG edges, not an implicit linear list**), `when` (conditional execution, e.g. branch gating), `with` (provider-specific config under the provider's own schema), `outputs` (named results other steps reference via `needs`-scoped interpolation, e.g. `${{ build.artifact }}`) |
| `spec.execution` | `maxParallel` concurrency, fail-fast strategy, timeout, per-step retry overrides |
| `spec.environments` | Named environments with approval gates (required reviewers) — e.g. a `production` environment gating a `deploy` step on platform-team approval |
| `spec.policies` | Workflow-wide enforcement: `secrets.forbidPlaintext`, `providers.requireVersion`, `dependencies.forbidCycles`, `artifacts.immutable`. **Enforced rules, not documentation** |

`needs[]` is what makes this a graph rather than a list: parallel unit-tests and a vulnerability scan can both depend on `build` and both feed `artifact` (fan-out / fan-in).

**Consistency check against the existing decisions:** the illustrative manifest uses `capability: test` for both a unit-test step and a vulnerability-scan step, differing only by provider. That matches the already-decided mapping where `Test` / `Lint` / `Vuln` become three independent `Tester` implementations. No new capability is introduced — the five stay five.

## Capabilities

### New Capabilities

- `public-module-api`: the public, versioned, cross-language module contract — its four properties, capability decomposition (Build, Deploy, Run, Test, Artifact), the Dagger type-system constraint, the versioning/stability policy.
- `composition-model`: the programmatic composition mechanism, how capabilities combine, the renamed composition type (D4), retirement of the legacy `Pipeline` interfaces, and the removal of named presets (D3).
- `workflow-manifest`: **NEW.** The declarative contract — document identity and versioning, the `capability`-versus-`uses` separation, variable and secret referencing rules, provider version pinning, and the validation rules a manifest must satisfy (including acyclicity).
- `workflow-execution`: **NEW.** The engine — provider resolution (capability→implementation, local and external `module:`), topological ordering, concurrency, failure strategy, timeouts and retries, conditional execution, output interpolation, approval gates, and policy enforcement.

Splitting the workflow layer into contract (`workflow-manifest`) and mechanism (`workflow-execution`) mirrors the `public-module-api` / `composition-model` split already used, so reviewers can verify the declared schema independently of the engine that runs it.

### Modified Capabilities

- None. `product-identity` is the only existing domain spec and remains unaffected.

## Approach

1. Specify the four properties and capability orthogonality as testable requirements *(unchanged)*.
2. Design phase selects the composition mechanism under the verbatim Dagger constraint *(unchanged — Dagger Interfaces/Objects already chosen)*.
3. Specify the stable-from-day-one versioning policy (D2) and the supersede path (D1) *(unchanged)*.
4. **Rename the composition type (D4)** in the Layer 1 contract before anything depends on the old name.
5. **Decompose `go-service` into standalone capability implementations with no bundling identity, and delete the preset path and the preset registry (D3)** — do not build a factory that reconstitutes it.
6. **Specify the manifest schema as a versioned document contract**, with the `capability`/`uses` separation as its central invariant.
7. **Build the DAG parser and validator** — cycle detection, dependency resolution, topological ordering — with the three shipped policies enforced by failing tests, not comments.
8. **Build provider resolution and the execution engine** on top of the already-designed capability contract; the engine resolves a provider and checks only that it satisfies the declared capability.
9. Wire the Dagger module and verify the contract from a second language *(unchanged)*.
10. Migrate the DI container and plugin layer; delete the dead interfaces; regenerate mocks *(unchanged)*.
11. Apply the minimum doc correction *(unchanged)*.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `openspec/specs/{public-module-api,composition-model,workflow-manifest,workflow-execution}/` | New | The four contracts (created at archive) |
| `dagger.json`, `.dagger/` | New | Module wiring + second-language binding generation |
| Workflow manifest schema + parser/validator | **New** | DAG parsing, cycle detection, topological ordering, policy enforcement |
| Provider resolution + execution engine | **New** | Capability→provider lookup incl. external `module:` refs; graph execution, concurrency, retries, gates |
| `internal/pipelines/common/interfaces.go` | **Removed** | Dead code |
| `internal/pipelines/pipeline.go` | **Removed/Replaced** | Superseded by the capability contract |
| `internal/pipelines/registry.go`, `options.go` | **Removed/Replaced** | Preset registry deleted (D3), not re-typed |
| `main.go` | Modified | `--pipeline go-service` path **removed**; entrypoint targets a workflow manifest instead |
| `internal/interfaces/interfaces.go` | Modified/Replaced | `Pipeline` shape retired; `Container`/`StepRegistry`/`HookManager` re-typed |
| `internal/pipelines/go-service/*` | **Decomposed** | Becomes standalone capability implementations with no bundle identity |
| `internal/app/{container,pipeline_executor,step_registry,hook_manager}.go` | Modified | **Largest legacy blast radius** — DI wiring is typed against the retired interface |
| `internal/plugins/interfaces.go` | Modified | `PluginContext.GetPipeline()` returns a retired type |
| `mocks/**`, `internal/*/mocks.go` | Regenerated | `go.uber.org/mock` output tracks the retired interfaces |
| `docs/API.md`, `docs/ARCHITECTURE.md` | Modified (minimum) | Stop presenting `Pipeline` as canonical |

**Dagger compatibility note:** `dagger.io/dagger` v0.21.8 remains the client dependency. `dagger.json` adds an engine pin that must stay compatible with it, and binding generation adds a Dagger CLI dependency to the dev/CI toolchain. Verify both before merge.

## Risks

**This change is now materially larger than the already-High-risk Revision 1.** Stated plainly, as with the prior scope merge: review burden, changed-line count, and correctness risk all increase again.

| Risk | Likelihood | Mitigation |
|---|---|---|
| **The 8-work-unit / ~3,800–4,800-line plan is now an undercount.** Three new deliverables (preset removal, rename, full DAG layer) land on top of it | **High** | The tasks phase MUST re-forecast from scratch and will very likely need **more** chained PR slices than 8, not fewer. Stacked-to-main chain strategy already selected; the sequence grows |
| **Three scope expansions in one change** (contract+mechanism merge, then preset removal + rename, then the DAG layer) | **High** | Accepted user decision. Four separate spec files keep the pieces independently reviewable; the DAG layer sequences strictly after the contract stabilizes |
| **String-interpolation injection / secret leakage** in `${{ variables.x }}` and `${{ secrets.x }}` resolution | **High — security-relevant** | Secrets MUST resolve as `*dagger.Secret` handles, **never string-substituted into plaintext**. `forbidPlaintext` is an enforced policy with a failing test. Interpolation must not permit arbitrary expression evaluation. Design phase owns the resolution boundary explicitly |
| **DAG cycle-detection correctness.** A missed cycle is a hang or a silent partial execution; a false positive rejects valid fan-in graphs | **High** | `forbidCycles` is enforced, not assumed. RED tests first, covering self-edges, mutual pairs, long cycles, diamond fan-in (valid), and disconnected components |
| **Version-space collision:** manifest-level `uses.version` versus the already-decided contract-level `ContractVersion` / `COMPATIBILITY.md` axis | **High** | **Unresolved — design phase MUST reconcile:** are these the same version space or orthogonal? Revision 1 already declared three distinct axes; provider versioning is a candidate fourth and must be named as such or explicitly folded in |
| Migration blast radius breaks the DI container, plugin layer, or the decomposed capabilities at runtime | **High** | Strict TDD (`tdd: true`); `go test -race ./...` green per slice; 90% coverage gate holds |
| Removing `--pipeline go-service` leaves no working entrypoint until the manifest path lands | **Med** | Sequence the manifest entrypoint before or with the preset deletion; no slice may merge with the CLI unable to run anything |
| Scope creep inside the DAG layer (full policy engine, adapter library, CI integration) | **Med** | The Non-Goals table above is the boundary. Only three policies ship, and only enough adapters to demonstrate the engine |
| External `module:` provider resolution has no registry to resolve against | **Med** | Explicit non-goal. Resolution assumes local/vendored providers or an existing mechanism |
| Chosen mechanism not representable in Dagger's type system | Med | Verbatim constraint is a binding acceptance check; validate against generated bindings, not documentation |
| Stable-from-day-one locks in an unproven shape — now including a brand-new manifest schema | Med | Keep the guaranteed surface minimal; the manifest carries its own `apiVersion` so the schema can evolve independently of the Go contract |
| Cross-language binding proof adds unbudgeted toolchain/CI cost | Med | Accepted as a raised acceptance bar; a documented local `dagger` invocation suffices (CI wiring is out of scope) |
| **Terminology collision:** "capability" now means the Build/Test/… contract, an OpenSpec spec domain, *and* a manifest step field | Med | Spec phase MUST disambiguate in prose on first use in each document; do not rely on context |
| `dagger.json` engine pin conflicts with the pinned v0.21.8 client | Low | Verify pin compatibility before merge |
| Doc drift: `docs/*` keep teaching a deleted interface | **High** | Minimum correction is in scope and is a success criterion; full rewrite is an explicit fast-follow |

## Rollback Plan

Not purely additive — this change deletes live abstractions and a CLI path.

- **Chained PRs (expected):** revert slices in **reverse merge order**. Reverting the migration slice without the deletion slice leaves the tree uncompilable.
- **Single PR:** `git revert -m 1 <sha>`.
- **New in this revision:** reverting the preset deletion restores `--pipeline go-service`, so any revert crossing that slice must also revert the manifest entrypoint, or the CLI ends up with neither path. Treat preset-deletion and manifest-entrypoint as one rollback boundary.
- No state, data, config-file, or release migration, and no external consumers to notify — rollback is code-only. The manifest layer is greenfield, so there are no existing YAML files to migrate.
- Verification after any revert: `go build -o shipwright .` and `go test -race ./...` both green.

## Dependencies

- `dagger.json` engine-version pin compatible with `dagger.io/dagger` v0.21.8.
- Dagger CLI in the dev/CI toolchain for binding generation.
- A second-language SDK toolchain (TypeScript or Python) to verify generated bindings.
- A YAML parser for the manifest layer (design phase selects; prefer an already-vendored dependency).

## Success Criteria

Criteria carried from Revision 1:

- [ ] The contract states all four properties as testable requirements, and the Dagger type-system constraint appears verbatim and binds every public element
- [ ] No public contract element requires Go generics or any other language-specific mechanism
- [ ] Capabilities (Build, Deploy, Run, Test, Artifact) are orthogonal and independently meaningful
- [ ] The composition mechanism is decided, documented with rationale, and implemented
- [ ] The versioning policy states a guarantee effective from the first release, its scope, and the breaking-change rule
- [ ] `internal/pipelines/common/interfaces.go` is deleted; `pipeline.go` and `internal/interfaces/interfaces.go` no longer define a `Pipeline` extension point
- [ ] The DI container, plugin layer, and all generated mocks compile and pass against the new contract
- [ ] Generated Dagger SDK bindings exist in TypeScript or Python and consuming them is demonstrated
- [ ] `docs/API.md` and `docs/ARCHITECTURE.md` no longer present `Pipeline` as the canonical public surface
- [ ] `go build -o shipwright .` and `go test -race ./...` green; coverage ≥ 90%

New in Revision 2:

- [ ] **No named pipeline preset exists anywhere**: no `--pipeline go-service` flag, no preset registry, and no type or identifier naming a stack bundle. A grep for the preset name finds only historical notes
- [ ] `go-service` has become standalone capability implementations, each independently composable and usable without its former siblings
- [ ] The public composition type no longer contains the word "Pipeline"
- [ ] A workflow manifest declares steps by `capability` + `uses`, and the engine resolves providers **without any knowledge of the specific tool named**
- [ ] `needs[]` produces real fan-out/fan-in graph execution, not linear ordering — proven by a test with two parallel steps sharing one dependency and one dependent
- [ ] **`forbidCycles`, `requireVersion`, and `forbidPlaintext` are enforced** — each has a test that fails when the rule is violated
- [ ] Secrets referenced from a manifest resolve as `*dagger.Secret` handles; no code path substitutes a secret into a plaintext string
- [ ] The relationship between `uses.version` and `ContractVersion` is explicitly resolved and documented in `COMPATIBILITY.md`
- [ ] An approval-gated environment blocks its dependent step until approval is recorded

---

*Deviation note:* this proposal exceeds the skill's 450-word budget, as `design.md` and `tasks.md` did before it. Cause: it is a superseding revision that must record a verified defect with evidence, three new decisions, a full manifest schema shape, and an expanded non-goals boundary — while remaining a self-contained contract for the spec and design phases. Content is compressed into tables; completeness was prioritized over the word budget.
