# Tasks: Public Versioned Module API + Composition Model + Declarative Workflow Orchestration

> **REPLACES the prior WU1–WU8 tasks.md.** Scope grew 6→12 slices (declarative
> workflow manifest + DAG execution engine added). Re-forecast from scratch
> per design REVISION 2's explicit instruction.

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | ~7,500–10,500 (additions+deletions, 12 slices; excludes generated mock regen and TS bindings from authored-risk count) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | 12 work units = 12 chained PRs, mirroring design.md's Migration Sequence table exactly |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main (pre-selected by user for this change) |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

Grounding (measured, current tree): `internal/app/container.go` 724L,
`main.go` 548L, `internal/pipelines/go-service/pipeline.go` 530L,
`internal/plugins/mocks.go` 521L (hand-rolled, not regen-only),
`internal/app/mocks.go` 575L, `internal/app/pipeline_executor.go` 328L,
`internal/app/step_registry.go` 284L, `internal/interfaces/interfaces.go`
270L, `internal/pipelines/options.go` 269L, `internal/plugins/nomad_deploy.go`
258L, `internal/app/hook_manager.go` 179L, `go-service/options.go` 134L,
`internal/executors/docker_executor.go` 131L, `pipeline.go` 82L,
`common/interfaces.go` 83L, `registry.go` 62L. `pkg/shipwright/`,
`.dagger/`, `internal/workflow/**`, `internal/capabilities/**` do not exist
yet — all net-new. Slice 10 (DI/plugin re-type) is the largest single unit
(~1,800+ lines touched across 4 files); slice 11 (deletion) removes
~1,650+ lines outright. Prior estimate is confirmed an undercount.

### Suggested Work Units

| # | Goal | PR | Focused test | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Layer 1 contract `pkg/shipwright/**` | 1 | `go test ./pkg/shipwright/...` | N/A — compile-time contract, nothing calls it yet | delete `pkg/shipwright/**` |
| 2 | Module wiring + `Plan` (`dagger.json`, `.dagger/**`) | 2 | `make dagger-test` | `dagger call` against `.dagger/`, real engine | delete `dagger.json`+`.dagger/**`; root untouched |
| 3 | `internal/capabilities/**` | 3 | `go test ./internal/capabilities/...` | `-short`-guarded real-container case | delete `internal/capabilities/**`; go-service untouched |
| 4 | `internal/workflow/manifest/**` | 4 | `go test ./internal/workflow/manifest/...` | N/A — pure parsing | delete package; additive |
| 5 | `internal/workflow/interp/**` | 5 | `go test ./internal/workflow/interp/...` | N/A — no eval, no I/O | delete package; additive |
| 6 | `internal/workflow/graph/**` | 6 | `go test ./internal/workflow/graph/...` | N/A — pure algorithm | delete package; additive |
| 7 | `internal/workflow/providers/**` | 7 | `go test ./internal/workflow/providers/...` | N/A — fake provider | delete package; additive |
| 8 | `internal/workflow/engine/**` + examples | 8 | `go test ./internal/workflow/engine/...` | `-short` end-to-end fake-capability run | delete package+examples; additive |
| 9 | CLI `--workflow` entrypoint (`main.go`) | 9 | `go test ./...` (root) | `shipwright --workflow examples/workflow/diamond.yaml` | pair with PR 11 (see 9.5) |
| 10 | DI/plugin re-type onto Layer 1 (largest) | 10 | `go test -race ./internal/app/... ./internal/plugins/... ./internal/executors/...` | `--pipeline go-service`, still green via shim | must precede PR 11 |
| 11 | Preset+shim deletion (rollback-paired with 9) | 11 | `go build -o shipwright . && go test -race ./...` | `shipwright --workflow .shipwright/workflow.yaml` | revert 9+11 together |
| 12 | Cross-language proof + docs | 12 | `tsc --noEmit` | `dagger call` from `examples/crosslang-ts/` | delete `examples/crosslang-ts/**` |

## Phase 1 — Layer 1 contract (`public-module-api`)

- [x] 1.1 RED: 5 capability interfaces compile with zero Go generics; signatures use only Dagger core types+scalars.
- [x] 1.2 GREEN: `pkg/shipwright/capabilities.go`, `config.go` (per-capability structs, D-D), `version.go` (`ContractVersion`).
- [x] 1.3 RED/GREEN: guaranteed-surface golden `testdata/api.golden`; review `-update` diff, never blind-accept.

## Phase 2 — Module wiring + `Plan` (`public-module-api`/`composition-model`)

- [x] 2.1 SPIKE: prove/refute interface-typed `Plan` chaining state survives Dagger v0.21.8 serialization. VERDICT: GO, with a signature caveat — see apply-progress for the full transcript. Interface-typed Object field state round-trips correctly (verified via a real chained call producing an actual artifact, not a hardcoded result). Dagger v0.21.8's Go SDK codegen cannot compile a client proxy for an interface method returning a lazy-chainable Dagger core type (`*dagger.Directory`/`*dagger.File`/`*dagger.Container`) together with `error`; `Builder.Build`/`Tester.Test`/`Runner.Run` drop the `error` return accordingly (Dagger's own lazy-chainable idiom). Scalar-returning methods (`(string, error)`) are unaffected.
- [x] 2.2 CONTINGENCY: not triggered — 2.1 confirmed GO, not refutation.
- [x] 2.3 RISK: not triggered — `dagger init --engine-version` resolved to `v0.21.8` verbatim (installed CLI is exactly v0.21.8), matching root `go.mod`'s existing pin with no bump needed.
- [x] 2.4 GREEN: `dagger.json`, `.dagger/**` (Dagger Interfaces, `Shipwright`/`Plan` Objects, thin adapters only) — signatures adjusted per the 2.1 spike finding.
- [x] 2.5 RED/GREEN: pin-parity test — root `go.mod` `dagger.io/dagger` == `dagger.json` `engineVersion` (`internal/daggerpin`).
- [x] 2.6 GREEN: `make dagger-test`; confirmed root `go build ./...`/`go test -race ./...` never traverse `.dagger/`.

## Phase 3 — Capability implementations (`composition-model`)

- [x] 3.1 RED: naming golden — no exported identifier in `internal/capabilities` names a stack bundle.
- [x] 3.2 GREEN: `GoBuilder` (from Build/buildBinary/buildDocker).
- [x] 3.3 GREEN: `GoUnitTester`, `GoLinter`, `GoVulnScanner` (three independent Testers — orthogonality win).
- [x] 3.4 GREEN: `ContainerPublisher` (from Package/Tag/Push).
- [x] 3.5 RED/GREEN: `var _ Builder = (*GoBuilder)(nil)` etc. per implementation; original `go-service` left in place.

## Phase 4 — Manifest schema + parser (`workflow-manifest`)

- [x] 4.1 RED: document-identity validation (missing `apiVersion`/`kind`/`metadata.name`).
- [x] 4.2 RED: structure validation (empty/duplicate step id, capability outside the five, missing `uses`, empty `uses.version`).
- [x] 4.3 RED: schema golden drift test (`testdata/schema.golden`); any accepted-field-set change forces an `apiVersion` decision in the same PR.
- [x] 4.4 SECURITY RED: oversized manifest rejected by an `io.LimitReader` cap (start 1 MiB; tune against 4.5's fixture).
- [x] 4.5 SECURITY RED: YAML alias-amplification ("billion laughs") fixture completes within a bounded time/memory budget — do not rely on yaml.v3's internal alias limits.
- [x] 4.6 GREEN: typed structs + `gopkg.in/yaml.v3` `KnownFields(true)` decode, stages 1–3; secrets referenced by name only, no inline plaintext accepted.

## Phase 5 — Interpolation + typed values (`workflow-execution`) [SECURITY-CRITICAL]

- [x] 5.1 RED: every rejected grammar form — operator, function call, nested placeholder, unknown namespace, trailing path segment — stage-4 parse error, never literal-text fallback.
- [x] 5.2 SECURITY RED: compile-level assertion that no exported accessor on `Value` returns a secret as `string`.
- [x] 5.3 SECURITY RED: `secrets.*` reference in a non-secret-typed field rejected (stage-7 `forbidPlaintext`). PARTIAL BY DESIGN — see apply-progress: full enforcement needs provider `WithSchema` (Phase 7); this WU ships `Reference.StaticKind()`, the primitive Phase 7's check will call, with its own RED/GREEN tests.
- [x] 5.4 SECURITY RED: secret+literal concatenation (`"Bearer ${{ secrets.tok }}"`) rejected — concatenation would require a string form.
- [x] 5.5 GREEN: hand-written scanner, closed grammar (`variables.`/`secrets.`/`steps.<id>.output`); `Value{kind,str,secret}` — `KindSecret` has no string accessor.

## Phase 6 — Graph + Kahn + kind checks (`workflow-execution`)

- [x] 6.1 RED: self-edge rejected.
- [x] 6.2 RED: mutual pair rejected.
- [x] 6.3 RED: long cycle (4+) rejected.
- [x] 6.4 RED: diamond fan-in ACCEPTED (`b`,`c` need `a`; `d` needs `b`,`c`) — no cycle false-positive.
- [x] 6.5 RED: disconnected components (multiple roots) accepted.
- [x] 6.6 RED: unknown `needs` id rejected, error DISTINCT from a cycle error.
- [x] 6.7 RED: duplicate step ids rejected.
- [x] 6.8 RED: data reference without a declared `needs` edge rejected (prevents unordered read of another step's output).
- [x] 6.9 RED: output-kind/input-kind mismatch rejected before anything runs — PARTIAL BY DESIGN, see apply-progress: only the statically-knowable case (a `secrets.*` reference used as a step's `input`) is checked; `with`-field kind compatibility and `steps.<id>.output` kind checks are deferred to Phase 7 (require provider `WithSchema`).
- [x] 6.10 GREEN: Kahn's algorithm (in-degree waves); cycle error enumerates residual-in-degree ids.

## Phase 7 — Provider registry + resolution (`workflow-execution`)

- [x] 7.1 RED: resolution hit/miss per capability (5 Register\*/Resolve\* pairs).
- [x] 7.2 RED: unsupported provider version rejected.
- [x] 7.3 SECURITY RED: unregistered `module:` reference fails closed at stage 6, naming the module path — proves the "compile-time-only, no `plugin.Open`" boundary (D-I).
- [x] 7.4 SECURITY RED: static assertion — no manifest-reachable code path calls `plugin.Open`.
- [x] 7.5 RED: `with` value kind mismatch against provider `WithSchema` rejected (stage 7).
- [x] 7.6 GREEN: typed `Registry`, `Ref{Name,Module,Version}`, `WithSchema`; register slice-3 capabilities.

## Phase 8 — Execution engine (`workflow-execution`)

- [x] 8.1 RED: wave order deterministic, manifest-declaration order within a wave.
- [x] 8.2 RED: fail-fast stops later waves, names the failing step id; skips not-yet-started dependents.
- [x] 8.3 RED: per-step `context.WithTimeout` fires.
- [x] 8.4 RED: bounded per-step retry.
- [x] 8.5 SCOPE NOTE (own task, do not over-build): `maxParallel` validated/recorded but NOT used to widen execution — serial is a correct schedule for any `maxParallel >= 1`; `maxParallel <= 0` fails at stage 3. Concurrent widening within a wave is explicitly deferred. GAP FOUND: `maxParallel <= 0` is confirmed NOT enforced anywhere in `internal/workflow/manifest` stage 3 as of this WU (grep-verified, no reference to MaxParallel outside schema.go's field decl and parse_test.go's decode assertion) — flagged for `sdd-verify`, not fixed here per this task's own instruction not to reach back into WU4's package.
- [x] 8.6 RED (absence-of-behavior, flag for sdd-verify): approval metadata under `spec.environments.<name>.approvals` does NOT block/queue/gate — engine executes per normal DAG ordering with no recorded approval. Supersedes the original proposal's blocking criterion (D-M).
- [x] 8.7 RED: `when` accepts only a structured predicate map (`when: {branch: [main]}`), evaluated by exact match — per design.md D-L. Verified `specs/workflow-execution/spec.md` already shows the structured form; no spec/design drift found (already reconciled in a prior WU).
- [x] 8.8 GREEN: wave scheduler over `graph.Graph` + `providers.Registry`, invoking Layer 1 interfaces directly — never through `Plan`.
- [x] 8.9 GREEN: `examples/workflow/*.yaml` including the diamond fan-in case.

## Phase 9 — CLI manifest entrypoint (`workflow-execution`)

- [x] 9.1 RED: `--workflow` with a missing manifest fails closed naming the expected path — no implicit legacy fallback. `manifest.ParseFile`'s own `os.Open` error already names the exact path (`"manifest: open %s: %w"`); `loadWorkflowManifest` adds no fallback. Proven by `TestCLI_Run_WorkflowMissingManifestFailsClosed`, through the real `CLI.Run` dispatcher.
- [x] 9.2 RED: `--step <id>` executes only `<id>`'s `needs`-transitive closure, topological order — via WU8's `engine.Closure` (`internal/workflow/engine/subgraph.go`), never reimplemented. Proven by `TestSelectWorkflowGraph_StepClosure` (diamond example, `unit`'s closure = `{build, unit}` only) and `TestSelectWorkflowGraph_UnknownStepFailsClosed`.
- [x] 9.3 RED: `--list-steps` lists manifest step ids + capability + resolved provider (name/version), resolved via `providers.Registry` with an empty `Values` map (safe — `checkWithSchema` only rejects fields actually present). Proven by `TestCLI_Run_WorkflowListSteps`, `TestResolveStepInfos_DiamondExample`, and `TestCLI_Run_WorkflowListSteps_UnregisteredProviderFailsClosed`.
- [x] 9.4 GREEN: wired `--workflow <path>` (default `.shipwright/workflow.yaml`) through `manifest.ParseFile` → `graph.Build` → `providers.RegisterDefaults` → `engine.OptionsFromSpec`/`engine.Execute` (`runWorkflowCLI`/`executeWorkflow` in `main.go`) — no engine logic reimplemented, main.go only assembles `engine.Config` and calls `engine.Execute`. `executeWorkflow`/`workflowDaggerClient`/`resolveWorkflowSecrets`/`resolveWorkflowSource` require a live Dagger client and are NOT unit-tested in this WU (0% coverage, matching this file's own pre-existing pattern for every other real-infra function — `executePipelineWithExecutor`, `executeStepLocally`, `runHealthChecks`, etc. were already 0% before this WU); verified up to (not including) real Dagger execution via `TestLoadWorkflowManifest_DiamondExample` against the real `examples/workflow/diamond.yaml`. **GAP FLAGGED for sdd-verify**: `spec.source.repo`/`ref` (git-based source) is not implemented — only `spec.source.path` is wired; `resolveWorkflowSource` fails closed with an explicit error naming the gap, deliberately out of this WU's "wire main.go to the already-built engine package" scope.
- [x] 9.5 SEQUENCING (non-negotiable, per D-N): `--pipeline`, `--list-pipelines`, `--only-build`, `--only-test`, `--skip-push`, and every other legacy flag are untouched byte-for-byte in this WU (only additive code — the entire legacy `Run()` dispatch after the new `if flags.workflowSet { return ... }` guard is unmodified). Confirmed via `go build -o shipwright . && shipwright --pipeline go-service --help` (exit 0, full legacy flag set printed unchanged) and `TestCLI_parseFlags_WorkflowModeDetection`'s "stays legacy" case. Both `--pipeline go-service` and `--workflow ...` work simultaneously after this WU — no merged state runs neither.

## Phase 10 — DI + plugin re-type onto Layer 1 (`composition-model`) [LARGEST]

- [x] 10.1 RED: `PluginContext.GetCapabilities()`/`GetConfig()` replace `GetPipeline()`/`GetPipelineConfig()`.
- [x] 10.2 RED: `Container`/`StepRegistry`/`HookManager` compile against Layer 1; `Artifact`→`StepArtifact` (collision).
- [x] 10.3 GREEN: re-type `internal/app/{container,pipeline_executor,step_registry,hook_manager}.go`, `internal/interfaces/interfaces.go`, `internal/plugins/{interfaces,context}.go`, `internal/executors/{selector,docker_executor}.go`.
- [x] 10.4 GREEN: keep a thin deprecated `pipelines.Pipeline` shim so `--pipeline` still runs during this slice.
- [x] 10.5 GREEN: regenerate `mocks/**`, `internal/{app,plugins,executors}/mocks.go`.
- [x] 10.6 SEQUENCING: must merge before Phase 11 — deleting the shim before this re-type leaves the tree uncompilable.

## Phase 11 — Preset + shim deletion (`composition-model`) [rollback-paired with 9]

- [x] 11.1 RED: no preset registry/factory map/CLI flag keyed by a stack name (e.g. `go-service`) exists anywhere. `internal/pipelines/no_preset_test.go` (`TestNoPresetRegistryOrStackNamedPipelineSurface`) asserts absence of `internal/pipelines/{pipeline.go,registry.go,common,go-service,infra}`; RED-confirmed against the pre-deletion tree, GREEN after 11.3.
- [x] 11.2 RED: `--pipeline`/`--list-pipelines`/`--only-build`/`--only-test`/`--skip-push` absent from `main.go`. `main_test.go` (`TestCLI_parseFlags_PresetFlagsRemoved`) asserts `flag.Parse` rejects all five as unknown; RED-confirmed pre-deletion, GREEN after 11.4.
- [x] 11.3 GREEN: deleted `internal/pipelines/{pipeline,registry}.go`, `common/interfaces.go`, `go-service/**`. **Deviation (see Deviation Note below): `options.go` was deliberately NOT deleted** — its `Config` struct is live, load-bearing plumbing shared by `internal/plugins`, `internal/executors`, and `internal/app/container.go`'s `BuildCapabilities`, none of which this work unit is authorized to re-type. **Judgment call (unanticipated reference):** also deleted `internal/pipelines/infra/**` — a second stack-named preset pipeline, unreachable once the preset registry is gone, and it depended on exactly the `pipelines.Pipeline`/`pipelines.HookFunc` types `pipeline.go`'s deletion removes. Also deleted `tests/go_pipeline_test.go` (ginkgo suite testing the deleted go-service package) and `mocks/pipeline_mock.go` (generated from the deleted `pipeline.go`).
- [x] 11.4 GREEN: removed `--pipeline`/`--list-pipelines`/`--only-build`/`--only-test`/`--skip-push` and the `PipelineAdapter`/`PipelineRegistry` shim from `main.go`/`internal/app/container.go`; removed `interfaces.Pipeline`/`PipelineRegistry`/`PipelineProvider` from `internal/interfaces/interfaces.go`; removed the now-uncompilable `App.RunPipeline`/`RunPipelineStep`/`ListPipelines`/`GetPipelineInfo` from `internal/app/app.go`; regenerated `mocks/interfaces_mock.go` (zero drift on the other 3 generated files) and hand-updated `mocks/gen.go`, `internal/{app,plugins}/mocks.go`.
- [x] 11.5 VERIFY: built `/tmp/shipwright-wu11` and ran `--workflow examples/workflow/diamond.yaml`: `--list-steps` exits 0 listing all 4 real steps; a full run reaches real `engine.Execute` against a live Dagger client (Docker detected) and fails only at the actual container-build step with a sandbox Dagger-engine-session error (same class of pre-existing sandbox limitation WU10 documented, not a regression). `--pipeline go-service --help` now fails closed with `flag provided but not defined: -pipeline`, confirming no merged state runs neither.

### Phase 11 review remediation — plugin → `providers.Registry` bridge

Added after a second independent review of PR #157 found that deleting `App.RunPipeline` left `App.loadAndInitializePlugins`/`cleanupPlugins` unreachable AND left the plugin extension mechanism itself structurally dead (`HookManager`/`StepRegistry` have no live production consumer; `engine.Execute` consults neither).

- [x] 11.6 RED: `internal/plugins/provider_bridge_test.go` — a plugin must reach the run's `*providers.Registry` (`PluginContext.GetProviderRegistry`) and register a real `shipwright.Deployer` resolvable through WU7's `ResolveDeployer`; `NomadDeployPlugin` must satisfy `shipwright.Deployer`; `LoadBuiltinPlugins` must never call `LoadFromFile`/`LoadFromConfig`. RED-confirmed (undefined symbols) before implementation.
- [x] 11.7 RED: `plugin_lifecycle_test.go` — `runWithPluginLifecycle` must run `CleanupPlugins` via `defer` so it fires on the run-failure path AND after a partial load failure, and a cleanup error must never mask the run's real outcome.
- [x] 11.8 GREEN: new extension point — `PluginContext.GetProviderRegistry() *providers.Registry`, `PluginLoader.ListBuiltins()`, `PluginRegistry.LoadBuiltinPlugins()` (builtin-only, never config/`.so`); `NewPluginContext` gains a `providerRegistry` parameter.
- [x] 11.9 GREEN: `NomadDeployPlugin` rewritten onto `shipwright.Deployer` (`Deploy(ctx, artifactRef, environment, creds)`), registering itself via `Registry.RegisterDeployer(Ref{Name: "nomad-deploy", Version: "1"}, WithSchema{...}, factory)`. Its `HookManager.RegisterHook`/`StepRegistry.RegisterStep` registrations and the `nomadDeployStepHandler`/`buildImageRefFromConfig` surface were REMOVED (repo-wide grep confirmed `PipelineExecutor` — the sole executor of hooks/registry steps — has no production caller after 11.4; only `examples/**` reaches it). Credentials cross into the container only as `*dagger.Secret` via `WithSecretVariable`.
- [x] 11.10 GREEN: `App.LoadAndInitializePlugins(ctx, providerRegistry, daggerClient)` / `App.CleanupPlugins(ctx)` exported and wired into BOTH `executeWorkflow` (live client) and `listWorkflowSteps` (nil client, so listing resolves the same provider set a run does), through `runWithPluginLifecycle`'s deferred cleanup. Stale `executePipelineWithExecutor above` reference in `workflowDaggerClient`'s doc comment corrected.
- [x] 11.11 VERIFY: ad-hoc manifest with `capability: deploy`, `uses.provider: nomad-deploy` → `--list-steps` exits 0 resolving the plugin-contributed provider; a full run logs `Nomad deploy provider registered` → `Deploying to Nomad` → `deployment=nomad://staging/ghcr.io/acme/api:v1` → `Workflow completed successfully`. A deliberately invalid variant (missing `artifactRef`) logs `Cleaning up Nomad deploy plugin` BEFORE the failure, proving the deferred cleanup. `examples/workflow/diamond.yaml` still fails only at the pre-existing sandbox Dagger-session error (confirmed identical on a stashed baseline build).

## Phase 12 — Cross-language proof + docs (`public-module-api`)

- [x] 12.1 GREEN: `examples/crosslang-ts/` — TypeScript `Builder` against generated bindings. `ExampleBuilder` (`@object()` class) implements `ShipwrightBuilder` (generated from `.dagger/capabilities.go`'s `Builder` interface); `id(): Promise<ID>` added via TypeScript declaration merging (type-only, zero runtime member — a real `id()` method broke the module runtime's own identity marshaling, see 12.2 evidence). Deliberately trivial: returns `source` unchanged.
- [x] 12.2 RED/GREEN: type-check clean + one documented local `dagger call`. `npx tsc --noEmit -p tsconfig.json` exits 0. Real invocation against the live engine: `dagger shell -c 'shipwright | plan $(host | directory <src>) | with-build $(example-builder) | execute'` — succeeds end-to-end (all steps green, `execute` returns `""`, matching `Plan.Execute`'s documented semantics when only `WithBuild` is chained). Two confirmed, documented v0.21.8 TS-SDK runtime constraints (see apply-progress): (1) a locally `new`-constructed instance cannot be passed inline as an Interface-typed argument within the same function body — it must be exposed as its own Dagger Function and composed across two engine calls; (2) a real (non-type-only) `id()` method is consumed by the runtime's own object-identity/introspection machinery and a stub implementation breaks marshaling — declaration merging avoids this by adding the member to the type only.
- [x] 12.3 GREEN: `COMPATIBILITY.md` — guaranteed surface (verified against `pkg/shipwright/testdata/api.golden`'s actual current content, not reconstructed from memory), five version axes per design.md D-E, explicit provider-version exclusion (`uses.version` orthogonal to `ContractVersion`).
- [x] 12.4 GREEN: minimum correction to `docs/API.md`/`docs/ARCHITECTURE.md` — added a prominent status notice to both (the entire pre-migration `Pipeline`/preset content is retained as explicitly-marked historical reference, not rewritten — a full rewrite is an out-of-scope fast-follow per the proposal's non-goals); added a "CLI Entrypoint (current)" section to `docs/ARCHITECTURE.md` documenting `--workflow`/`--step`/`--list-steps` as current and `--pipeline`/`--list-pipelines`/`--only-build`/`--only-test`/`--skip-push` as removed.

## Deviation Note

Exceeds the skill's 530-word budget. Cause: 12-slice migration (was 6),
each slice individually named as its own PR-worthy work unit per
design.md's own "Migration Sequence (also the PR-slice seams)" table, plus
mandatory explicit callouts (9 DAG invariants as separate lines, 4 distinct
security tasks, 1 absence-of-behavior test, 1 spec/design drift flag, 2 hard
sequencing constraints) that the launch instructions required not be buried
in prose. Completeness prioritized over the word budget, consistent with
this change's own design.md deviation note.

### WU11 deviation note (recorded during apply, not at planning time)

Two deviations from this task list's/design.md's literal text, both forced
by a full-repo dead-code sweep before deletion (per this work unit's own
launch instructions):

1. **`internal/pipelines/options.go` was NOT deleted**, despite tasks.md
   11.3 and design.md's File Changes table naming it. Its `Config` struct
   is live, load-bearing plumbing consumed by `internal/plugins`,
   `internal/executors`, and `internal/app/container.go`'s
   `BuildCapabilities` (the Layer-1-based DI wiring WU10 built and this
   work unit is explicitly barred from re-typing). Deleting it would have
   required re-typing all three packages — well beyond "remove the preset
   registry and its flags." `pipelines.Config` is not itself a preset
   registry/factory map keyed by a stack name, so keeping it does not
   violate task 11.1's invariant.
2. **`internal/pipelines/infra/**` was deleted, though named nowhere in
   tasks.md or design.md.** It is a second stack-named preset pipeline
   (`SyntegrityInfraPipeline`), reachable only through the same
   `PipelineRegistry`/`--pipeline infra` this work unit deletes, and it
   depends on exactly the `pipelines.Pipeline`/`pipelines.HookFunc` types
   `pipeline.go`'s deletion removes — leaving it in place would not
   compile. `tests/go_pipeline_test.go` (a ginkgo suite exercising the
   deleted `go-service` package) and `mocks/pipeline_mock.go` (generated
   from the deleted `pipeline.go`) were deleted for the same reason.

Scope also grew past tasks.md 11.3/11.4's literal file list because the
entire legacy `--pipeline` CLI dispatch tree in `main.go`
(`executePipeline`, `executeSingleStep`, `executePipelineWithExecutor`,
`executeStepWithExecutor`, `executePipelineLocally`, `executeStepLocally`,
`listAvailablePipelines`, the legacy `listAvailableSteps`,
`isCIEnvironment`) and `App.RunPipeline`/`RunPipelineStep`/
`ListPipelines`/`GetPipelineInfo` (`internal/app/app.go`) call
`Container.GetPipeline`/`GetPipelineRegistry` directly or transitively;
once those methods were removed with the `PipelineAdapter`/
`PipelineRegistry` shim, this entire tree became uncompilable dead code
and had to go too — `--workflow` is now main.go's only dispatch path
(`Run()` always calls `runWorkflowCLI`), matching the launch instructions'
own framing ("leaving `--workflow` as the sole CLI entrypoint"). Flagged
for `sdd-verify`: `App.loadAndInitializePlugins`/`cleanupPlugins`
(Layer-1 capability/plugin wiring) are left in place per the "do not
touch" boundary but now have zero production callers — a candidate for
reintegration with `--workflow` or removal in a later work unit.
