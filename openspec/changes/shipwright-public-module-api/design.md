# Design: Public Versioned Module API + Composition Model + Declarative Workflow Orchestration

> **Revision note — DESIGN REVISION 2.** Amends, not replaces, the previous design. `D-A`–`D-E` stand, with `Pipeline` renamed to `Plan` throughout. Removed outright: the `pipelines.Registry` capability-set-preset claim and the `--pipeline go-service` "UX unchanged" migration argument — a verified self-contradiction against this change's own principle. Added: `D-F`–`D-N` covering the declarative workflow layer, which is now the primary CLI entrypoint. The Threat Matrix flips from `N/A` to applicable.

## Technical Approach

**Two contract layers (unchanged) plus a workflow layer that is a consumer, not a third contract.**

| Layer | Location | Role |
|---|---|---|
| 1 — capability contract | `pkg/shipwright/` (root module) | Five capabilities as plain exported Go interfaces, no generics. All internal code implements this |
| 2 — Dagger projection | `.dagger/` (own Go module) | Same five as **Dagger Interfaces** (`DaggerObject` embed); composition as Dagger Objects. Thin adapters, no logic |
| 3 — workflow layer | `internal/workflow/**` | Manifest schema, DAG, provider resolution, scheduler. **Consumes Layer 1; defines no new Go contract** |

The Layer 1/2 split is forced, not chosen: `DaggerObject` exists only inside the Dagger-generated package, and `.dagger/` as a separate Go module cannot import `github.com/pablogore/shipwright/internal/**`.

Layer 3 lives in `internal/` deliberately (see D-H). Its public contract is the **YAML document**, which is data — inherently language-neutral, so it satisfies "cross-language consumable" without any type projection.

## Architecture Decisions

### D-A: Dagger Interfaces for seams, Dagger Objects for state *(unchanged; `Pipeline` → `Plan`)*

| Option | Tradeoff | Verdict |
|---|---|---|
| Go generics in public signatures | No support evidence in exported Dagger Functions; forbidden verbatim by the proposal | Rejected (binding constraint) |
| Concrete Go interfaces only | Not representable in Dagger's type system; zero cross-language reach | Rejected |
| Concrete Dagger Objects only | Representable, but a `Deploy` taking one concrete artifact type forces a single producer — the combinatorial explosion returns at object level | Rejected |
| **Dagger Interfaces (seams) + Dagger Objects (state/results)** | Structural typing gives cross-language substitutability; Objects carry chaining state | **Chosen** |

**Signature rule (risk control):** capability interface methods use ONLY Dagger core types (`Directory`, `File`, `Container`, `Secret`) and scalars. Module-defined Objects never appear in an *interface method* signature — that corner of Dagger's type system is unverified.

```go
// .dagger/capabilities.go — the public Dagger surface
type Builder interface {
	DaggerObject
	Build(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error)
}
type Tester interface {
	DaggerObject
	Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error)
}
type Artifactor interface {
	DaggerObject
	Publish(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error)
}
type Deployer interface {
	DaggerObject
	Deploy(ctx context.Context, artifactRef, environment string, creds *dagger.Secret) (string, error)
}
type Runner interface {
	DaggerObject
	Run(ctx context.Context, build *dagger.Directory) (*dagger.Container, error)
}

type Shipwright struct{}
func (m *Shipwright) ContractVersion() string
func (m *Shipwright) Plan(source *dagger.Directory) *Plan

type Plan struct{ Source *dagger.Directory } // Dagger Object, interface-typed state
func (p *Plan) WithBuild(b Builder) *Plan    // + WithTest/WithArtifact/WithDeploy/WithRun
func (p *Plan) Execute(ctx context.Context) (string, error)
```

`Setup` disappears — source is an *input*, not a step. `BeforeStep`/`AfterStep` leave the contract; hooks stay host-side (`interfaces.HookManager`), which is what makes capabilities orthogonal. Generics are permitted only in unexported helpers.

**Fallback if interface-typed Object state does not survive Dagger serialization** (proven in slice 2, never assumed): collapse to one flat Function `Plan(ctx, source, build Builder, test Tester, artifact Artifactor, deploy Deployer, run Runner)`. Interfaces as Function *arguments* are documented; only the chained-state form carries risk. The capability contract is identical either way, and Layer 3 is unaffected because it never goes through `Plan` (D-G).

### D-B: Module wiring and version-pin decoupling *(unchanged)*

| Element | Decision |
|---|---|
| `dagger.json` (root) | `{name: shipwright, sdk: go, source: ".dagger", engineVersion: "v0.21.8"}` |
| Module source | `.dagger/` with its **own** Dagger-generated `go.mod` |
| Root `go.mod` | Unchanged — keeps `dagger.io/dagger v0.21.8` for `main.go` / `internal/**` |
| Isolation | `.dagger/` is both a nested module and a dot-directory, so root `go build ./...` and `go test -race ./...` never traverse it; its tests run via a new `make dagger-test` |
| Pin coupling | **Resolved, not deferred**: the two pins live in separate modules and never link. The residual risk is *drift*, so it becomes a test |

Pin-parity guard — a plain root-module unit test (no Dagger required) reads `dagger.json`'s `engineVersion` and root `go.mod`'s `dagger.io/dagger` version and asserts equality. If `dagger init` refuses `v0.21.8`, bump **both** sides in the same commit keeping that test green — never diverge.

### D-C: Second-language proof = TypeScript *(unchanged)*

| Option | Lift for this repo | Verdict |
|---|---|---|
| Python (`@interface` over `Protocol`) | Runs in-engine, no host toolchain; but the proof is runtime-only — a passing `dagger call` proves one path, not the contract shape | Rejected |
| **TypeScript (`@interface`)** | Runs in-engine, no host Node needed; generated bindings are typed `.ts`, so type-checking the implementation is a **compile-time** proof that the whole contract shape crossed the language boundary | **Chosen** |

Artifact: `examples/crosslang-ts/` with its own `dagger.json` (`sdk: typescript`) depending on the root module. It implements `Builder` in TypeScript and passes it to `shipwright.plan().withBuild(...)`. Acceptance = bindings generated + type-check clean + one successful `dagger call`, documented as a local invocation.

### D-D: `pipelines.Config` decomposes per capability *(unchanged)*

The monolith (Git + Registry + Build + Coverage + SSH + Go/Java) is replaced by per-capability structs in `pkg/shipwright/config.go`, so orthogonality is compiler-enforced rather than documented.

| New struct | Absorbs |
|---|---|
| `SourceConfig` | GitRepo, GitRef, GitProtocol, GitUserEmail, GitUserName, SSHPrivateKey |
| `BuildConfig` | GoVersion, JavaVersion, BuildMode, BinaryName |
| `TestConfig` | Coverage |
| `ArtifactConfig` | Registry, RegistryURL, RegistryUser, ImageName, ImageTag, BuildTag, CommitSHA, BranchName, Version |
| `DeployConfig`, `RunConfig` | Empty at this change — adapters deferred |
| **Dropped** | `Image`, `ImageContainer`, `ImageRef` — live Dagger handles are runtime state, not configuration |
| **Retyped** | `RegistryPass`, `RegistryToken`, `Token` → `*dagger.Secret` |

**Binding on Layer 3:** the manifest's `spec.secrets` mechanism MUST reuse this retyping and the existing `client.SetSecret` pattern from `internal/pipelines/shared/docker.go`. It MUST NOT introduce a second secret representation.

### D-E: Versioning mechanics *(amended — version spaces)*

| Element | Mechanism |
|---|---|
| Source of truth | `pkg/shipwright/version.go` → `const ContractVersion = "1.0.0"` |
| Machine-readable marker | Projected as `Shipwright.ContractVersion()`, readable via `dagger call contract-version` |
| Go module path | Bare at contract v1. A major bump requires `/v2` suffix + `ContractVersion` major + migration note, together |
| Policy file | New `COMPATIBILITY.md` |
| Enforcement | `pkg/shipwright/testdata/api.golden` + a golden surface test; any guaranteed-surface change fails RED |

**Guaranteed surface, deliberately minimal:** the five capability interfaces (both layers), `Shipwright.{Plan, ContractVersion}`, `Plan.{WithBuild, WithTest, WithArtifact, WithDeploy, WithRun, Execute}`, the `pkg/shipwright` config structs, and the `shipwright.dev/v1` manifest schema (as a *data* contract, guarded by its own golden — see D-H). Nothing else.

**Version spaces — five independent axes.** Revision 1 declared three; the workflow layer adds two. This closes the proposal's open version-space question:

| Axis | Carrier | Guarantee |
|---|---|---|
| Contract version | `ContractVersion` | Stable from first release, guaranteed surface only |
| Manifest schema version | `apiVersion: shipwright.dev/v1` in each document | Evolves independently of `ContractVersion`; additive-only within `v1` |
| CLI release SemVer | goreleaser + `CHANGELOG.md` | Ordinary release versioning |
| Engine pin | `dagger.json` `engineVersion` | Must equal the root `go.mod` client pin (D-B parity test) |
| **Provider version (`uses.version`)** | Manifest step, owned by the provider | **Not covered by any Shipwright guarantee** |

`uses.version` and `ContractVersion` are **orthogonal**, not the same space. The stable-from-first-release guarantee covers only the enumerated surface above; it does **not** extend to in-repo or third-party providers, their `with` schemas, or their version semantics. A provider may break its own users at any version without touching `ContractVersion`. `COMPATIBILITY.md` MUST state this exclusion explicitly.

### D-F: No named preset — capability implementations, no bundling identity

The previous design's "`pipelines.Registry` registers capability-set factories, keeping `--pipeline go-service` CLI UX intact" is **deleted**. It reintroduced the named-preset anti-pattern under a compatibility argument that is void with zero external consumers.

| Option | Tradeoff | Verdict |
|---|---|---|
| Capability-set factory registry keyed by preset name | Preserves CLI UX; **is** the anti-pattern under another name | Rejected (the verified defect) |
| One `internal/capabilities/goservice` package holding all five | No preset *type*, but the package path itself names the bundle | Rejected |
| **Flat `internal/capabilities` package, one file and one exported type per implementation** | No bundle identity in any type, file, or path; each registers individually | **Chosen** |

| Legacy `go-service` method | New implementation | Capability |
|---|---|---|
| `Setup` | Removed — source is an input `*dagger.Directory` | — |
| `Build`, `buildBinary`, `buildDocker` | `GoBuilder` | `Builder` |
| `Test` | `GoUnitTester` | `Tester` |
| `Lint` | `GoLinter` | `Tester` |
| `Vuln` | `GoVulnScanner` | `Tester` |
| `Package`, `Tag`, `Push` | `ContainerPublisher` | `Artifactor` |
| `BeforeStep`, `AfterStep` | Removed from contract — `interfaces.HookManager` owns hooks | — |

Three independent `Tester` implementations is the orthogonality win, and it is what makes `capability: test` with three different providers work in a manifest. `Deployer`/`Runner` ship contract-only. Naming rule enforced by test: a golden test over the exported identifiers of `internal/capabilities` fails if any identifier names a stack bundle.

### D-G: `Pipeline` → `Plan`, and `Plan` is not the DAG

| Option | Tradeoff | Verdict |
|---|---|---|
| Keep `Pipeline` | Already generic and composed, so not defective — but the word invites `GoPipeline`/`JavaPipeline` regressions, and the last round proves the pull is real | Rejected (preventive rename, user-final) |
| `Compose` | A verb; reads badly as a type and as `dagger call compose` | Rejected |
| **`Plan`** | Noun, short, no stack connotation, reads well chained and cross-language (`shipwright.plan().withBuild(...)`) | **Chosen** |

**`Plan` is the programmatic/cross-language composition surface. It is NOT the manifest's representation.** A five-slot chain cannot express `needs[]` edges, and forcing the graph through it would either flatten the DAG or bloat the guaranteed surface with graph primitives. Layer 3 therefore compiles a manifest into an internal `graph.Graph` of resolved capability instances and invokes the Layer 1 interfaces directly. The shared invariant across both front-ends is **the capability interface, not the composition object**.

Terminology guard: the engine's compiled artifact is a `Graph`, never a "plan". `Plan` is reserved for the Dagger Object.

### D-H: Manifest schema representation

| Option | Tradeoff | Verdict |
|---|---|---|
| JSON Schema file + a runtime validator dependency | Machine-checkable and publishable, but adds a dependency and duplicates the Go shape | Rejected (new dep; non-goal proximity) |
| `map[string]any` + hand-rolled key walking | No new types, but every field access is an untyped lookup — the error surface the security skill warns about | Rejected |
| **Typed Go structs + `gopkg.in/yaml.v3` with `KnownFields(true)`, in `internal/workflow/manifest`** | Already a direct dependency (`go.mod`), no custom unmarshalers, unknown fields fail closed, mirrors the existing `internal/config/yaml_parser.go` pattern | **Chosen** |

Schema drift enforcement mirrors D-E: `internal/workflow/manifest/testdata/schema.golden` records the accepted field set; any schema change fails RED and forces an explicit `apiVersion` decision in the same PR.

**Schema simplification (design decision, the proposal's field list was explicitly non-normative):** `spec.steps[].outputs` is **dropped**. Every capability returns exactly one typed result, so a step's result is addressable as `${{ steps.<id>.output }}` with its kind fixed by the capability. An aliasing layer would add naming surface for zero expressive power. A step's Directory-typed input is bound by `steps[].input` (default `spec.source`).

**Validation is a fixed seven-stage pipeline, and nothing executes until every stage admits the document** (no partial execution):

| # | Stage | Fails closed on |
|---|---|---|
| 1 | Size-capped read + decode | File over the read cap; malformed YAML; unknown field |
| 2 | Document identity | `apiVersion`/`kind` outside the allowlist |
| 3 | Structure | Empty or duplicate step `id`; `capability` outside the five; missing `uses`; `uses.version` empty (`requireVersion`) |
| 4 | References | `needs` naming an unknown step; interpolation reference to an undeclared variable/secret/step |
| 5 | Graph | Cycle; data reference without a matching `needs` edge; output-kind incompatible with the consumer's input kind |
| 6 | Provider resolution | No provider registered for `(capability, name)`; version unsupported |
| 7 | Value binding | Secret reference in a non-secret-typed field (`forbidPlaintext`); type mismatch in `with` |

### D-I: Provider resolution — typed registry, compile-time-only external modules

| Option | Tradeoff | Verdict |
|---|---|---|
| One `map[string]func(...) any` + type assertion at use | Single registry, but resolution errors surface at execution time and `any` defeats the capability typing | Rejected |
| Reuse `internal/plugins/loader.go` (`plugin.Open` on a `.so` path) | Existing machinery, but runs arbitrary native code in-process from a manifest-controlled path — the highest-severity surface in the repo | **Rejected (security)** |
| Fetch `module:` references at run time | Matches the illustrative `github.com/acme/custom-builder` syntax literally, but is a package-registry service — an explicit non-goal | Rejected |
| **Five typed register methods; `module:` resolves only to already-compiled, self-registered providers** | Fully typed resolution, fail-closed, zero new runtime attack surface | **Chosen** |

```go
// internal/workflow/providers/registry.go
type Registry struct{ /* per-capability maps, guarded */ }

func (r *Registry) RegisterBuilder(ref Ref, schema WithSchema, f func(Values) shipwright.Builder)
// + RegisterTester / RegisterArtifactor / RegisterDeployer / RegisterRunner

func (r *Registry) ResolveBuilder(ref Ref, v Values) (shipwright.Builder, error)
// + one Resolve* per capability

type Ref struct{ Name, Module, Version string } // Module == "" means in-repo
```

`WithSchema` is the provider's declaration of its `with` keys as `name → kind{string,int,bool,secret}`. It is what makes stage 7 checkable and what keeps providers receiving **typed values, never a shell string** (the security skill's `sh -c` rule).

**Scope boundary, stated explicitly:** `uses.module` is resolved *only* against providers that registered themselves at build time (a Go import in the binary's provider-registration file). There is no fetch, no download, no cache, no `.so` load, and no registry service. An unregistered `module:` reference fails closed at stage 6 with an error naming the module path. Broadening this is a follow-up change and would need its own security review.

### D-J: DAG construction and cycle detection — Kahn's algorithm

| Option | Tradeoff | Verdict |
|---|---|---|
| DFS three-colour marking | Correct for detection, but recursion depth on a user-supplied graph, and topological order needs a second pass | Rejected |
| Reachability matrix / transitive closure | Simple to reason about, but O(n³) and reports a cycle without naming its members | Rejected |
| New graph library dependency | Battle-tested, but a new dependency for ~60 lines | Rejected |
| **Kahn's algorithm (in-degree waves)** | Detects cycles and produces the topological order in one pass; diamond fan-in is handled natively by in-degree counting, which is exactly where naive visited-set DFS variants produce false positives | **Chosen** |

Cycle report: after Kahn drains, any node with residual in-degree > 0 is in or downstream of a cycle. The error enumerates those ids so the author can act.

Enforced invariants (each a RED test before implementation): self-edge; mutual pair; long cycle (4+); **diamond fan-in accepted** (`b` and `c` both need `a`, `d` needs both); disconnected components accepted (multiple roots); `needs` to an unknown id rejected with a *distinct* error from a cycle; duplicate ids rejected; data reference without a declared `needs` edge rejected (prevents an unordered read of another step's output); output-kind/input-kind mismatch rejected before anything runs.

`policies.dependencies.forbidCycles` is therefore an **enforced check with a failing test**, not documentation. It has no permissive setting in this change: acyclicity is unconditional, and the policy field records the intent for schema stability.

### D-K: Scheduling — sequential level waves, declared parallelism as an upper bound

| Option | Tradeoff | Verdict |
|---|---|---|
| Full bounded worker pool honouring `maxParallel` now | Matches the schema literally, but adds concurrent Dagger client use, cancellation propagation, partial-failure semantics, and race-detector surface — roughly doubling the engine's review burden | Rejected for this change |
| Ignore `spec.execution` entirely | Smallest, but silently discards a declared field | Rejected |
| **Execute Kahn's waves in order, sequentially within a wave in manifest-declaration order; `maxParallel` validated and recorded, not yet used to widen** | Sequential execution is a *correct* schedule of the same DAG for any `maxParallel ≥ 1`, so nothing is misreported; the wave boundary is the exact seam a worker pool drops into later | **Chosen** |

Stated plainly: a manifest declaring `maxParallel: 4` runs correctly but serially. `maxParallel` is an upper bound, not a requirement. `maxParallel: 0` or negative fails validation at stage 3.

Implemented now because each is small and independently testable: per-step timeout (`context.WithTimeout`), bounded per-step retry, fail-fast (on a step error, stop scheduling further waves and return an error naming the step id). Deferred with an explicit hook: concurrent widening within a wave.

### D-L: Interpolation — restricted placeholder grammar, secrets never become strings

| Option | Tradeoff | Verdict |
|---|---|---|
| `text/template` | Standard library, but supports function calls and reflective field traversal — a general evaluator | **Rejected (no eval)** |
| A CEL/`expr` expression library | Powerful conditions, but a new dependency *and* an arbitrary-expression evaluation surface | **Rejected (no eval)** |
| `os.Expand` | Tiny, but `$VAR` semantics leak the process environment into the manifest | Rejected |
| **Hand-written scanner over a fixed grammar producing typed values** | ~80 lines, no operators, no function calls, no nesting, no environment reach; every reference is resolvable statically at stage 4 | **Chosen** |

Grammar, closed and complete:

```
placeholder := "${{" ws ref ws "}}"
ref         := "variables." name
             | "secrets."   name
             | "steps."     name ".output"
name        := [A-Za-z_][A-Za-z0-9_-]*
```

Anything else — an operator, a function call, a nested placeholder, a third namespace, a trailing path segment — is a **parse error at stage 4**, not a fallback to literal text. The scanner emits `[]Token` where each token is a literal run or a resolved reference; there is no evaluation step to attack.

**The secret rule, mechanically enforced rather than documented:**

```go
type Kind int // KindString | KindInt | KindBool | KindSecret

type Value struct {
	kind   Kind
	str    string          // never set when kind == KindSecret
	secret *dagger.Secret  // only set when kind == KindSecret
}
```

- `secrets.*` resolves to a `Value{kind: KindSecret, secret: ...}` and to nothing else. There is no accessor that returns a secret as a `string`.
- A `secrets.*` reference in a field whose provider-declared kind is not `KindSecret` is a **stage-7 validation error** (`forbidPlaintext`), never a substitution. So no code path can produce a plaintext secret, because the string-producing path cannot hold a secret value at all.
- A placeholder mixing a secret reference with literal text in one field (`"Bearer ${{ secrets.tok }}"`) is rejected for the same reason: concatenation would require a string form.
- Plaintext exists at exactly one boundary, unchanged from today's code: the `client.SetSecret(name, value)` call that reads the configured environment variable. That value never enters the scanner, a manifest-derived string, or a log line. New value-carrying types implement `String()` omitting secrets, following `GitCredentials.String()`.

**Conditions are structured, not expressions.** `when` is a YAML predicate map over the same restricted references (for example `when: {branch: [main, develop]}`) evaluated by exact match. A string-expression `when` would reintroduce the evaluator this decision exists to avoid.

### D-M: Approval gates are metadata-only

`spec.environments.<name>.approvals` is **parsed, validated for well-formedness, surfaced in the workflow description and logs, and never consulted by the scheduler.** No blocking state machine, no approval store, no reviewer identity check, no gate ships in `workflow-execution`.

**Rationale:** an enforced gate needs durable approval state, an identity source, and a resume path — three subsystems that are each larger than the engine itself, and all three sit inside the "full policy engine / approval-workflow UI" non-goal. A half-enforced gate is worse than a declared one, because it would look like a control.

**Explicit divergence from the proposal, flagged for `sdd-spec` and `sdd-tasks`:** the proposal's success criterion *"An approval-gated environment blocks its dependent step until approval is recorded"* is **superseded** by this decision (user's explicit correction). The replacement criterion is: *a manifest declaring approvals parses, validates, and executes without blocking, and a RED test asserts the engine does not gate.*

### D-N: CLI entrypoint and the deletion-sequencing coupling

The manifest is the **primary** entrypoint. `main.go` keeps the stdlib `flag` package (migrating off `flag` remains a non-goal).

| Flag | Disposition |
|---|---|
| `--workflow <path>` | **New.** Manifest path. Default `.shipwright/workflow.yaml`; a missing file fails closed naming the expected path — never an implicit legacy fallback |
| `--step <id>` | **Retargeted** to a manifest step id. Executes the `needs`-transitive closure of `<id>` in topological order and stops — a subgraph run, computed by reachability over the graph already built. Preserves the existing per-step CI invocation pattern |
| `--list-steps` | **Retargeted** to list manifest step ids with capability and resolved provider |
| `--pipeline`, `--list-pipelines` | **Removed** — they name presets |
| `--only-build`, `--only-test`, `--skip-push` | **Removed** — they presuppose the preset's step names; `--step` replaces them |
| `--config .shipwright.yml`, `--env`, `--executor`, `--verbose`, `--version`, `--health`, `--local`, git flags | Unchanged |

**Sequencing coupling, made concrete (this was a proposal risk; it is now a slice-ordering rule):** the manifest entrypoint slice lands **strictly before** the preset-deletion slice, and for exactly one slice both `--workflow` and `--pipeline` work. No merge point ever leaves the CLI able to run nothing. The two slices form **one rollback boundary**: reverting the deletion without the entrypoint is fine (both paths return), but reverting the entrypoint without the deletion leaves no path — so a revert crossing that boundary must take both.

## Data Flow

```
 A. Manifest path (PRIMARY)                     B. Programmatic path
 ──────────────────────────                     ────────────────────
 shipwright --workflow wf.yaml                  dagger call | TS module | Go consumer
        │                                              │
        ▼                                              ▼
 manifest.Parse  (stages 1-3)                   Shipwright.Plan(source: Directory)
        │                                        .withBuild(Builder).withTest(Tester)...
        ▼                                              │
 interp.Scan     (stage 4, typed Values)               ▼
        │                                        Plan (Dagger Object,
        ▼                                              interface-typed state)
 graph.Build + Kahn  (stage 5)                         │ Execute()
        │                                              │
        ▼                                              │
 providers.Resolve   (stage 6)  ───────┐               │
        │                              │               │
        ▼                              ▼               ▼
 values.Bind         (stage 7)   ┌─────────────────────────────────┐
        │                        │  Layer 1  pkg/shipwright        │
        ▼                        │  Builder / Tester / Artifactor  │
 engine.Execute (waves) ────────►│  Deployer / Runner              │
                                 └───────────────┬─────────────────┘
                                                 ▼
                                 internal/capabilities/{GoBuilder,
                                   GoUnitTester, GoLinter,
                                   GoVulnScanner, ContainerPublisher}
                                                 ▼
                                 Dagger core types (Directory/File/Container/Secret)
```

Sequence — one manifest run (fail-closed admission before any side effect):

```
CLI      manifest   interp   graph   providers   engine   capability   Dagger
 │──parse──►│                                                          
 │          │──ok────►│  scan refs (no eval)                           
 │                    │──ok───►│ build + Kahn → order | CYCLE→abort     
 │                             │──ok────►│ resolve | UNKNOWN→abort      
 │                                       │──ok──►│ bind Values          
 │                                       │       │ SECRET-IN-STRING→abort
 │                                       │       │──wave 1──►│──►│ Directory/File
 │                                       │       │◄─result───│   │
 │                                       │       │──wave 2──►│──►│
 │◄──────────────── summary / first failing step id ─────────┘
```

Nothing between `parse` and `bind` touches the network, the filesystem beyond the manifest and its declared source, or a container.

## File Changes

| File | Action | Description |
|---|---|---|
| `pkg/shipwright/{capabilities,config,version}.go` | Create | Layer 1 contract, per-capability configs, `ContractVersion` |
| `pkg/shipwright/testdata/api.golden` | Create | Guaranteed-surface golden |
| `dagger.json`, `.dagger/**` | Create | Module wiring + Layer 2 adapters, `Shipwright`, `Plan` |
| `internal/capabilities/{gobuilder,gounittester,golinter,govulnscanner,containerpublisher}.go` | Create | Standalone implementations, no bundle identity (D-F) |
| `internal/workflow/manifest/{schema,parse,validate}.go` + `testdata/schema.golden` | Create | `workflow-manifest`: typed schema, decode, stages 1–3 |
| `internal/workflow/interp/{scan,value}.go` | Create | Restricted grammar scanner, typed `Value` (D-L) |
| `internal/workflow/graph/{build,kahn}.go` | Create | Adjacency, Kahn, cycle/kind validation (D-J) |
| `internal/workflow/providers/{registry,register.go}` | Create | Typed resolution; in-repo provider registration (D-I) |
| `internal/workflow/engine/{execute,subgraph}.go` | Create | Wave scheduler, timeout, retry, fail-fast, `--step` closure (D-K) |
| `main.go` | Modify | `--workflow` added; `--step`/`--list-steps` retargeted; preset flags removed (D-N) |
| `internal/pipelines/{pipeline,registry,options}.go` | Delete | Preset registry and legacy contract — not re-typed |
| `internal/pipelines/common/interfaces.go` | Delete | Dead code |
| `internal/pipelines/go-service/**` | Delete | Superseded by `internal/capabilities/**` |
| `internal/interfaces/interfaces.go` | Modify | `Pipeline` deleted; `Container`/`StepRegistry`/`HookManager` re-typed; `Artifact` → `StepArtifact` (name collision) |
| `internal/app/{container,pipeline_executor,step_registry,hook_manager}.go` | Modify | **Largest legacy blast radius** — DI wiring re-typed onto Layer 1 |
| `internal/plugins/{interfaces,context}.go` | Modify | `GetPipeline()` → `GetCapabilities()`; `GetPipelineConfig()` → `GetConfig()` |
| `internal/executors/{selector,docker_executor}.go` | Modify | Re-typed off `pipelines.Config` |
| `mocks/**`, `internal/*/mocks.go` | Regenerate | Track the retired interfaces |
| `examples/crosslang-ts/**` | Create | TypeScript proof (D-C) |
| `examples/workflow/*.yaml` | Create | Runnable manifest, including the diamond fan-in case |
| `COMPATIBILITY.md` | Create | Guaranteed surface, five version axes, provider-version exclusion |
| `docs/API.md`, `docs/ARCHITECTURE.md` | Modify (minimum) | Stop presenting `Pipeline` as canonical |
| `Makefile` | Modify | `make dagger-test` |

## Interfaces / Contracts

```yaml
apiVersion: shipwright.dev/v1
kind: Workflow
metadata:
  name: go-service-release
spec:
  source:
    path: .                        # must resolve inside the manifest's tree
  variables:
    imageRef: ghcr.io/acme/api
  secrets:
    registry: {fromEnv: REGISTRY_PASSWORD}
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: "1"}
      with: {goVersion: "1.26.1"}
    - id: unit                     # diamond: unit + vuln both need build
      capability: test
      uses: {provider: go-test, version: "1"}
      needs: [build]
      input: ${{ steps.build.output }}
    - id: vuln
      capability: test
      uses: {provider: govulncheck, version: "1"}
      needs: [build]
      input: ${{ steps.build.output }}
    - id: publish                  # fan-in
      capability: artifact
      uses: {provider: container, version: "1"}
      needs: [unit, vuln]
      input: ${{ steps.build.output }}
      with:
        ref: ${{ variables.imageRef }}
        creds: ${{ secrets.registry }}   # secret-typed key — string key would fail stage 7
      when: {branch: [main]}
  execution:
    concurrency: {maxParallel: 4}    # upper bound; serial execution is a valid schedule
    failFast: true
    timeout: 30m
  environments:
    production:
      approvals: {required: [platform-team]}   # metadata only (D-M)
  policies:
    dependencies: {forbidCycles: true}
    providers:    {requireVersion: true}
    secrets:      {forbidPlaintext: true}
```

## Migration Sequence (also the PR-slice seams)

| # | Slice | Touches | Blast radius |
|---|---|---|---|
| 1 | Layer 1 contract | `pkg/shipwright/**` + goldens | None — additive |
| 2 | Module wiring + Layer 2 (`Plan`) | `dagger.json`, `.dagger/**`, pin-parity test, `Makefile` | None to root code |
| 3 | Capability implementations | `internal/capabilities/**` (from `go-service` logic; original left in place) | Low — additive |
| 4 | Manifest schema + parser | `internal/workflow/manifest/**` + schema golden | None — additive |
| 5 | Interpolation + typed values | `internal/workflow/interp/**` | None — additive |
| 6 | Graph + Kahn + kind checks | `internal/workflow/graph/**` | None — additive |
| 7 | Provider registry + resolution | `internal/workflow/providers/**` (registers slice 3) | None — additive |
| 8 | Execution engine | `internal/workflow/engine/**`, `examples/workflow/*.yaml` | None — additive |
| 9 | **CLI manifest entrypoint** | `main.go` (`--workflow`, `--step`/`--list-steps` retarget) | Medium — **both CLI paths work after this slice** |
| 10 | DI + plugin re-type onto Layer 1 | `internal/app/**`, `internal/interfaces/interfaces.go`, `internal/plugins/{interfaces,context}.go`, `internal/executors/**`; legacy `pipelines.Pipeline` kept as a thin deprecated shim so `--pipeline` still runs | **Largest** |
| 11 | **Preset + shim deletion** | `internal/pipelines/{pipeline,registry,options}.go`, `common/interfaces.go`, `go-service/**`, preset flags in `main.go`, `mocks/**` regen | High — **rollback-paired with slice 9** |
| 12 | Cross-language proof + docs | `examples/crosslang-ts/**`, `docs/*`, `COMPATIBILITY.md` | Low |

Ordering rules that are not negotiable: **9 before 11** (never a CLI that runs nothing); **10 before 11** (deleting the shim before the DI re-type leaves the tree uncompilable); slices 4–8 are strictly additive and can be reviewed independently of the legacy tree. Rollback is reverse merge order, with 9+11 taken together.

## Testing Strategy

Strict TDD: every row is RED-first. Gates per slice: `go test -race ./...` green, `go build -o shipwright .` green, coverage ≥ 90% local / 70% CI, `golangci-lint run` with no function over gocyclo 15.

| Layer | What to test | Approach |
|---|---|---|
| Unit — Layer 1 | Contract shape, config decomposition, `ContractVersion`, pin parity, guaranteed-surface golden, `internal/capabilities` naming golden | In-package `_test.go`; capability interfaces are 1-method → hand-rolled stubs (double order 4) |
| Unit — manifest | Table-driven per validation stage: unknown field, bad `apiVersion`, duplicate/empty id, capability outside the five, missing `uses`, empty `uses.version`, `maxParallel ≤ 0`, schema golden drift | In-package, `testdata/*.yaml` fixtures |
| Unit — interp | Every rejected grammar form (operator, function call, nested placeholder, unknown namespace, extra path segment); `Value` has no string accessor for secrets; secret-in-string-field rejected; secret+literal concatenation rejected | Table-driven; a compile-level assertion that no exported accessor returns a secret as `string` |
| Unit — graph | Self-edge, mutual pair, long cycle, **diamond fan-in accepted**, disconnected components accepted, unknown `needs` id (distinct error), data reference without `needs`, output/input kind mismatch, `--step` closure correctness | Table-driven; cycle-error message asserts the offending ids |
| Unit — providers | Resolution hit/miss per capability, unsupported version, unregistered `module:` fails closed, `with` kind mismatch | Table-driven with a fake provider |
| Unit — engine | Wave order deterministic, fail-fast stops later waves, per-step timeout fires, retry bounded, approvals do NOT block, cancelled context propagates | Fake capability implementations recording invocation order |
| Unit — `.dagger` | Adapter conformance `var _ Builder = (*goBuilder)(nil)` per capability | `make dagger-test` |
| Integration (`testing.Short()`) | `dagger call` walking skeleton; interface-typed Object state round-trip (validates D-A, triggers its fallback if it fails); one real manifest end-to-end | In-package, short-guarded |
| Cross-language | TS module type-checks against generated bindings; one `dagger call` succeeds | `examples/crosslang-ts/` |
| Adversarial | Threat-matrix rows below, one test per applicable row | In-package, fixture-driven |

## Threat Matrix

**Applicable — the previous `N/A` verdict no longer holds.** This revision adds a host-facing input boundary: the CLI reads an externally-authored YAML manifest, selects a source directory from it, and resolves references that carry credentials.

| Boundary | Minimum adversarial cases | Applicability | Design response | Planned RED tests |
|---|---|---|---|---|
| Documentation-like paths | `requirements.txt`, executable Markdown, `README.sh` | **N/A** — the engine classifies nothing by filename; it acts only on step ids that a manifest declares explicitly | — | — |
| Git repository / source selection | `spec.source.path` with `..`, an absolute path, a symlink escaping the tree; `spec.source` git ref | **Applicable** | `path` MUST resolve (after `filepath.Abs` + symlink evaluation) inside the manifest's own directory tree; escapes fail closed before any Dagger call. Git sources reuse the existing `internal/pipelines/shared/credentials.go` cascade — no new auth mechanism | One test per escape form (`..`, absolute, symlink), each asserting a specific error, plus an accepted in-tree case |
| Commit state | staged, `commit -a`, empty index | **N/A** — no Git write operation exists anywhere in this change | — | — |
| Push state | tracking branch, first push, explicit refspec | **N/A** — image publication goes through `WithRegistryAuth`, not a Git push | — | — |
| PR commands | `--head`, environment prefix, composed commands | **N/A** — no VCS/PR automation | — | — |

Four additional boundaries are this change's real risk and carry RED tests as ordinary design requirements:

| # | Boundary | Design response | RED test |
|---|---|---|---|
| 1 | **Untrusted manifest YAML** — malformed input, deeply nested or alias-amplified documents ("billion laughs"), oversized files | The decoder is `gopkg.in/yaml.v3` (already a direct dependency) into typed structs with `KnownFields(true)` and **no custom unmarshalers**, so decoding executes nothing — the risk is resource consumption, not code execution. We do **not** rely on the library's internal alias limits: the manifest is read through an explicit `io.LimitReader` cap before decode, and nesting depth is bounded by the typed schema itself | Oversized file rejected by the cap; an alias-amplification fixture completes within a bounded time/memory budget; malformed YAML returns a specific error |
| 2 | **Interpolation injection / secret leakage** | The closed grammar in D-L has no operators, functions, nesting, or environment reach, so there is no expression to inject into. `Value` cannot hold a secret and a string simultaneously and exposes no string accessor for secrets, so a plaintext secret is unrepresentable rather than merely discouraged. `forbidPlaintext` is stage 7 | Secret in a string-typed `with` key rejected; secret+literal concatenation rejected; every non-grammar form rejected; no exported accessor returns a secret as `string` |
| 3 | **`uses.module` external providers — supply-chain** | Resolution never fetches, downloads, caches, or `plugin.Open`s anything (D-I explicitly rejects reusing `internal/plugins/loader.go` for manifest-declared providers). A `module:` reference can only name code that was already compiled into this binary and self-registered at build time, so a manifest cannot introduce code the operator did not build. The residual concern is ordinary Go dependency review, which `go.mod`/`go.sum` already covers | An unregistered `module:` reference fails closed naming the path; a test asserts no manifest-reachable code path calls `plugin.Open` |
| 4 | **Provider argument construction** | Providers receive `Values` typed by their declared `WithSchema` — never an interpolated shell string. Any container invocation uses argv-array `WithExec`, never `sh -c` with a manifest-sourced value (the existing `nomad_deploy.go` `fmt.Sprintf`-into-`sh -c` pattern is explicitly not to be copied) | A `with` value containing shell metacharacters (`'; rm -rf /`) reaches the provider as one inert argv element |

Additionally carried from Revision 1: credentials cross the public contract as `*dagger.Secret` only (a plaintext credential in the guaranteed surface fails the golden surface test), and step/capability routing MUST fail closed — an unknown step id or an absent capability returns an error, never a silent skip.

## Migration / Rollout

Code-only. No state, data, or release migration; the workflow layer is greenfield, so there are no existing manifests to migrate. Two user-visible CLI changes land in slice 11: `--pipeline`/`--list-pipelines`/`--only-build`/`--only-test`/`--skip-push` are removed, and `--workflow` becomes the entrypoint. There are no external consumers to notify; plugin-API breakage affects only in-repo `internal/plugins/nomad_deploy.go`. `docs/API.md` and `docs/ARCHITECTURE.md` receive the minimum correction in slice 12, and the same slice documents the flag removals.

## Open Questions

- [ ] Does Dagger v0.21.8 serialize interface-typed fields in Object chaining state? Proven or refuted in slice 2; refutation triggers the D-A fallback and leaves Layer 3 untouched.
- [ ] Does `dagger init` accept `engineVersion v0.21.8` verbatim? If not, bump both pins together in slice 2 with the parity test green.
- [ ] What is the right manifest read cap? Proposed 1 MiB, to be confirmed against the alias-amplification fixture in slice 4 — the number is a test constant, not a contract element.

---

*Deviation note:* this design exceeds the skill's 800-word budget (~3,400 words). Cause: it is a superseding revision that must preserve five existing decisions, remove a verified contradiction, and design nine new decisions spanning a manifest schema, a validation pipeline, provider resolution, a graph algorithm, a scheduler, a security-critical interpolation mechanism, and an entrypoint migration with a hard sequencing constraint — plus a Threat Matrix that flipped from `N/A` to applicable. Content is compressed into tables; completeness was prioritized over the word budget.
