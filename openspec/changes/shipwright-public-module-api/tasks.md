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

- [ ] 7.1 RED: resolution hit/miss per capability (5 Register\*/Resolve\* pairs).
- [ ] 7.2 RED: unsupported provider version rejected.
- [ ] 7.3 SECURITY RED: unregistered `module:` reference fails closed at stage 6, naming the module path — proves the "compile-time-only, no `plugin.Open`" boundary (D-I).
- [ ] 7.4 SECURITY RED: static assertion — no manifest-reachable code path calls `plugin.Open`.
- [ ] 7.5 RED: `with` value kind mismatch against provider `WithSchema` rejected (stage 7).
- [ ] 7.6 GREEN: typed `Registry`, `Ref{Name,Module,Version}`, `WithSchema`; register slice-3 capabilities.

## Phase 8 — Execution engine (`workflow-execution`)

- [ ] 8.1 RED: wave order deterministic, manifest-declaration order within a wave.
- [ ] 8.2 RED: fail-fast stops later waves, names the failing step id; skips not-yet-started dependents.
- [ ] 8.3 RED: per-step `context.WithTimeout` fires.
- [ ] 8.4 RED: bounded per-step retry.
- [ ] 8.5 SCOPE NOTE (own task, do not over-build): `maxParallel` validated/recorded but NOT used to widen execution — serial is a correct schedule for any `maxParallel >= 1`; `maxParallel <= 0` fails at stage 3. Concurrent widening within a wave is explicitly deferred.
- [ ] 8.6 RED (absence-of-behavior, flag for sdd-verify): approval metadata under `spec.environments.<name>.approvals` does NOT block/queue/gate — engine executes per normal DAG ordering with no recorded approval. Supersedes the original proposal's blocking criterion (D-M).
- [ ] 8.7 RED: `when` accepts only a structured predicate map (`when: {branch: [main]}`), evaluated by exact match — per design.md D-L, canonical over the `workflow-execution` spec's own scenario text, which illustrates a string-expression form; flag this spec/design drift to `sdd-verify`.
- [ ] 8.8 GREEN: wave scheduler over `graph.Graph` + `providers.Registry`, invoking Layer 1 interfaces directly — never through `Plan`.
- [ ] 8.9 GREEN: `examples/workflow/*.yaml` including the diamond fan-in case.

## Phase 9 — CLI manifest entrypoint (`workflow-execution`)

- [ ] 9.1 RED: `--workflow` with a missing manifest fails closed naming the expected path — no implicit legacy fallback.
- [ ] 9.2 RED: `--step <id>` executes only `<id>`'s `needs`-transitive closure, topological order.
- [ ] 9.3 RED: `--list-steps` lists manifest step ids + capability + resolved provider.
- [ ] 9.4 GREEN: wire `--workflow` (default `.shipwright/workflow.yaml`) through parse→interp→graph→providers→engine.
- [ ] 9.5 SEQUENCING (non-negotiable, per D-N): merges strictly before Phase 11; `--pipeline` stays untouched here so both paths work simultaneously.

## Phase 10 — DI + plugin re-type onto Layer 1 (`composition-model`) [LARGEST]

- [ ] 10.1 RED: `PluginContext.GetCapabilities()`/`GetConfig()` replace `GetPipeline()`/`GetPipelineConfig()`.
- [ ] 10.2 RED: `Container`/`StepRegistry`/`HookManager` compile against Layer 1; `Artifact`→`StepArtifact` (collision).
- [ ] 10.3 GREEN: re-type `internal/app/{container,pipeline_executor,step_registry,hook_manager}.go`, `internal/interfaces/interfaces.go`, `internal/plugins/{interfaces,context}.go`, `internal/executors/{selector,docker_executor}.go`.
- [ ] 10.4 GREEN: keep a thin deprecated `pipelines.Pipeline` shim so `--pipeline` still runs during this slice.
- [ ] 10.5 GREEN: regenerate `mocks/**`, `internal/{app,plugins,executors}/mocks.go`.
- [ ] 10.6 SEQUENCING: must merge before Phase 11 — deleting the shim before this re-type leaves the tree uncompilable.

## Phase 11 — Preset + shim deletion (`composition-model`) [rollback-paired with 9]

- [ ] 11.1 RED: no preset registry/factory map/CLI flag keyed by a stack name (e.g. `go-service`) exists anywhere.
- [ ] 11.2 RED: `--pipeline`/`--list-pipelines`/`--only-build`/`--only-test`/`--skip-push` absent from `main.go`.
- [ ] 11.3 GREEN: delete `internal/pipelines/{pipeline,registry,options}.go`, `common/interfaces.go` (confirmed dead), `go-service/**`.
- [ ] 11.4 GREEN: remove preset flags + shim from `main.go`; regenerate `mocks/**`.
- [ ] 11.5 VERIFY: `shipwright --workflow ...` proves a working path exists post-deletion — no merged state ever ran neither.

## Phase 12 — Cross-language proof + docs (`public-module-api`)

- [ ] 12.1 GREEN: `examples/crosslang-ts/` — TypeScript `Builder` against generated bindings.
- [ ] 12.2 RED/GREEN: type-check clean + one documented local `dagger call`.
- [ ] 12.3 GREEN: `COMPATIBILITY.md` — guaranteed surface, five version axes, explicit provider-version exclusion.
- [ ] 12.4 GREEN: minimum correction to `docs/API.md`/`docs/ARCHITECTURE.md` (stop presenting `Pipeline`); document flag removals.

## Deviation Note

Exceeds the skill's 530-word budget. Cause: 12-slice migration (was 6),
each slice individually named as its own PR-worthy work unit per
design.md's own "Migration Sequence (also the PR-slice seams)" table, plus
mandatory explicit callouts (9 DAG invariants as separate lines, 4 distinct
security tasks, 1 absence-of-behavior test, 1 spec/design drift flag, 2 hard
sequencing constraints) that the launch instructions required not be buried
in prose. Completeness prioritized over the word budget, consistent with
this change's own design.md deviation note.
