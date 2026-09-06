# Design: Provider-Managed Runtime/Toolchain Upgrade (Go First)

## Technical Approach

Two single-method capability kinds, per the proposal's Approach. All parsing
and mutation is pure Go over file *bytes* (`golang.org/x/mod/modfile`) executed
**before** any Dagger call; the mutated workspace is materialized as a new
immutable `*dagger.Directory`. Core gains two dispatch cases and two registry
pairs — no ecosystem logic, no blocking primitive (D1), no remote path (D2).

Canonical capability strings, fixed here for `sdd-spec` convergence:
**`runtime-inspect`** and **`runtime-upgrade`** (hyphenated, matching the
repo's existing hyphenated provider names `go-test`/`rust-integration-test`).

## Architecture Decisions

### D-1: Report types never appear in an interface signature

`pkg/shipwright/capabilities_test.go`'s `allowedSignatureType` reflection test
enforces design.md D-A: capability methods may use **only** `context.Context`,
`error`, `string`, and the four Dagger core types. A `DriftReport` struct
parameter/return would fail that existing test.

| Option | Verdict |
|---|---|
| Return concrete `*DriftReport` | **Rejected** — violates D-A, breaks a live test |
| Return `*dagger.File` (JSON report file) | Rejected — Layer 2 codegen cannot return a lazy core type *with* `error`; also unreferenceable from a `with` field |
| Return `string` (JSON payload) | **Chosen** for `Inspect` |
| Return mutated `*dagger.Directory`, report written *into* it | **Chosen** for `Upgrade` |

`Inspect`'s `(string, error)` is the shape the `.dagger` package doc records as
compiling cleanly under Dagger v0.21.8 codegen, and it maps to the engine's
`outputText` kind, so `${{ steps.<id>.output }}` can feed a downstream `with`
field. `Upgrade` returns a Directory (engine `outputDirectory`), which the
deferred D2 SCM change consumes directly. The report structs are still exported
concrete Go types in `pkg/shipwright` — they are the JSON contract, just never
a method parameter.

### D-2: `Upgrade` returns the Directory; the report rides inside it

A Dagger Interface method returns at most `(T, error)`, so directory + report
cannot both be returned. `Upgrade` writes
`.shipwright/runtime-upgrade-report.json` into the returned Directory.
Alternative (a second `LastReport()` method) rejected: it breaks the
one-capability-one-method invariant the proposal's Approach exists to preserve.

### D-3: Layer 2 declares the interfaces but does NOT wire them into `Plan`

`Plan` is a fixed five-slot linear `build→test→artifact→deploy→run` chain, and
the engine bypasses it entirely (D-G). A toolchain upgrade is not a stage of
that pipeline; two extra slots would force `Execute` to invent an ordering with
no defensible answer. Saves ~40 lines and a semantic mess. No Layer1/Layer2
interface-set parity test exists (verified), so this is safe.

Layer 2's `Upgrade` **drops `error`**, matching `Builder.Build`/`Tester.Test`:
the package doc records that v0.21.8 codegen cannot compile a lazy-chainable
core return type alongside `error`. `Inspect`'s `(string, error)` is kept.

### D-4: Version sources are split into three tiers by how *declarative* they are

Measured live drift in this repository — **six distinct versions**, materially
worse than the exploration recorded (it saw only the `gobuilder.go` case and
read CI as uniformly 1.26.7, missing `.go-version` and the three `release*.yml`
files entirely):

| Source | Version | Tier |
|---|---|---|
| `go.work`, `go.mod`, `providers/go/go.mod`, `providers/rust/go.mod` | 1.26.7 | 1 |
| `.go-version` | **1.26.1** | 1 |
| `.dagger/go.mod` | **1.26.5** | 3 |
| `.github/workflows/ci.yml` `GO_VERSION` | 1.26.7 | 2 |
| `.github/workflows/release.yml` `GO_VERSION` | **1.26.1** | 2 |
| `.github/workflows/release-provider-{go,rust}.yml` `GO_VERSION` | **1.26.1** | 2 |
| `providers/go/gobuilder.go` `defaultGoVersion` | **1.25.5** | 3 |

**Tier 1 — in scope, inspected and mutated.** `go.mod` `go`/`toolchain`,
`go.work`, and **`.go-version`**. `.go-version` earns inclusion on the same
criterion `go.mod` does: a fixed filename at the workspace root with a
single-line, tool-agnostic format (the goenv/asdf/mise convention). No
guessing, no per-repo assumption, ~30 lines to support, and it is a live drift
site *today*. It joins the A1–A3 unanimity check as a first-class source.

**Tier 2 — out of scope: CI workflow YAML.** `GO_VERSION` in
`.github/workflows/*.yml` is a naming *convention*, not a contract — the key
name, the file set, and the consuming action are all repo-specific. Locating it
generically is guessing, and guessing wrong means mutating an unrelated
workflow variable. A follow-up change can add an explicitly-configured
`ciVersionPins: [{file, key}]` list; v1 does not guess.

**Tier 3 — out of scope, structurally unreachable.**
- `providers/go/gobuilder.go`'s `defaultGoVersion` is **Shipwright's own**
  builder-image default, not a property of the inspected workspace. Reporting
  it as workspace drift is a category error, and against a third-party repo it
  is meaningless-to-misleading. No AST parsing is needed either way: the
  inspector lives in **package `golang`**, so the constant is a plain
  compile-time reference.
- `.dagger/go.mod` is deliberately absent from `go.work`'s `use` list, and the
  archived `shipwright-provider-go-module` design ships a guard test that
  *fails* if `use ./.dagger` is ever added (D-B module isolation). It is
  therefore invisible to workspace traversal by design. Its 1.26.5 drift is
  real but must be fixed by a separate, deliberate decision — not silently by
  this capability.

**Substitution, not deferral** for tier 3's first item: this change ships
`providers/go/toolchainpin_test.go` in the `internal/daggerpin` mould —
`TestDefaultGoVersionMatchesGoMod` parses `providers/go/go.mod`'s `go`
directive and asserts it equals `defaultGoVersion`. ~30 lines, RED today
(1.26.7 vs 1.25.5), runs locally in milliseconds, and enforces the invariant
permanently instead of reporting it into a JSON nobody reads.

> **`sdd-spec` convergence risk**: do NOT write an ADDED requirement that
> `runtime-inspect` reports a build-image or CI-YAML pin. Proposal success
> criterion #1 is satisfied by the guard test instead. **Do** write
> `.go-version` as an inspected and mutated source.

### D-4b: `verify` semantics without a third capability kind

Read-only inspection has two distinct uses: *report* the drift, and *fail the
build on* drift. Those are different engine outcomes, but not different
capabilities — a third `runtime-verify` kind would cost another allowlist
entry, registry pair, dispatch case, Layer 2 interface, and `main.go` branch
for a one-bit behavioral difference.

**Choice**: one `runtime-inspect` kind with a `failOnDrift` bool `with`-field
(default `false`). `failOnDrift: true` makes `Inspect` return an error when the
tier-1 sources disagree, which the engine already turns into a failed step and,
under `failFast`, a failed workflow. `failOnDrift: false` always succeeds and
emits the report string for a downstream step to consume.

```yaml
- id: verify-toolchain          # gate semantics
  capability: runtime-inspect
  with: { failOnDrift: true }
```

Rejected: a separate `runtime-verify` kind (~90 authored lines across six files
for one boolean); making failure the unconditional default (removes the
report-only mode the proposal's D2 handoff needs).

### D-5: `modfile` confirmed; the ambiguity rules

`golang.org/x/mod/modfile` is the correct dependency — the same package
`cmd/go` uses, already the repo's precedent (`internal/daggerpin/pin.go`,
`modfile.ParseWork` in the archived provider-go-module design). It supplies
`Parse`/`ParseWork`, `File.Go`/`File.Toolchain`/`WorkFile.Use`,
`AddGoStmt`/`AddToolchainStmt`/`DropToolchainStmt`, and `Format()` which
preserves comments and layout. `AddGoStmt` validates against
`modfile.GoVersionRE`, giving free fail-closed target validation. No
alternative considered viable.

**Note**: `golang.org/x/mod` is *not* currently a dependency of
`providers/go/go.mod` (only of the root module). It becomes a new direct
requirement there, pinned to root's version.

"Ambiguous" is exactly these, all detected **before** any mutation:

| Code | Condition | Behavior |
|---|---|---|
| A1 | Two workspace modules declare different `go` directives | abort |
| A2 | Two modules declare different `toolchain` directives | abort |
| A3 | `go.work`'s `go`, or `.go-version`, disagrees with the unanimous module `go` | abort |
| A4 | `targetVersion` < current (semver, `golang.org/x/mod/semver`) | abort unless `allowDowngrade: true` |
| A5 | Malformed target or existing directive | abort |
| A6 | Neither `go.work` nor `go.mod` found at workspace root | abort |

Abort returns a typed `*AmbiguousToolchainError` listing every conflicting site
and `(nil, err)`. **No partial mutation is structurally impossible to violate**:
analysis completes before the first `WithNewFile`, and a `dagger.Directory` is
an immutable value — a failed run simply never returns one.

### D-6: Validation is `go build ./...` per module — `go vet` explicitly rejected

The upgrade's claim is "this workspace still compiles under the new toolchain",
and `go build ./...` is exactly that claim. `go vet` was considered and
rejected: its checks evolve per release, so it widens the failure surface with
failures unrelated to the bump, it duplicates the repo's own CI stage, and it
doubles container time. The report records `validation: "build"` so no consumer
is misled about what was proven. Per-module, because `./...` provably does not
cross a `go.work` module boundary (verified in the archived design's Open
Question).

### D-7: `go mod tidy` sequencing

Ordered, single container chain:

1. Host-side pure Go: parse every `go.mod`/`go.work`, detect A1–A6, mutate bytes.
2. Materialize: one `WithNewFile` per changed file → new Directory.
3. `From("golang:" + targetVersion)`, mount the mutated Directory.
4. Per module: `cd <mod> && go mod tidy` (skipped when `tidy: false`).
5. Per module: `go build ./...` (must follow 4 — an untidied `go.sum` breaks it).
6. Export the container workdir as the returned Directory, carrying tidy's `go.sum`.

The report records, per module, `goSumChanged: bool` plus the added/removed
`require` module paths — **not** the raw `go.sum` diff, which can run to
thousands of unreviewable lines. Validation failure ⇒ abort, no Directory
returned.

### D-8: `main.go` is a sixth core dispatch site the proposal missed

`resolveCapabilityRef` (`main.go`, exercised by `main_test.go:337`'s "five
capability branches") is a second five-way capability switch driving
`--list-steps`. Without two new cases, a `runtime-inspect` step executes fine
but fails `--list-steps` with an unknown-capability error. Add to Affected
Areas: +2 cases, ~10 production lines.

### D-9: `providers/go/daggerkit` must grow a read/write surface

`DaggerDirectory` currently exposes only `GetRealDirectory()`, and `DaggerFile`
has no `Contents()` — the existing five providers never read a file. Reading
`go.mod` and writing the mutated tree needs `Directory.File(string)`,
`File.Contents(ctx)`, `Directory.Entries(ctx)`, and
`Directory.WithNewFile(path, contents)`, plus adapters and mocks. Doing it via
`WithExec` instead was rejected: it makes the parse/mutate core untestable
without a real engine, violating the testing-tdd skill's double-selection rule.
This is unbudgeted work the proposal did not surface (~95 lines).

## Data Flow

```
manifest step (capability: runtime-upgrade, with: {targetVersion})
   │
   ▼
engine/execute.go  dispatch ──▶ providers.ResolveRuntimeUpgrader
   │                                      │
   │                                      ▼
   │                          providers/go.GoRuntimeUpgrader
   ▼                                      │
 *dagger.Directory (source) ──────────────┤
                                          │ 1. Contents() go.work + .go-version
                                          │    + every use'd module's go.mod
                                          │ 2. parseWorkspace → detectConflicts (A1..A6)
                                          │ 3. mutate bytes (modfile.Format)
                                          │ 4. WithNewFile ×N  → mutated Directory
                                          │ 5. container: go mod tidy, go build ./...
                                          ▼
                       *dagger.Directory + .shipwright/runtime-upgrade-report.json
                                          │
                              (D2 follow-up change consumes this)
```

Nothing crosses the process boundary on the host: no `os.WriteFile`, no
`exec.Command`, no network, no git.

## File Changes

| File | Action | Description |
|---|---|---|
| `pkg/shipwright/capabilities.go` | Modify | +`RuntimeInspector`, +`RuntimeUpgrader` |
| `pkg/shipwright/runtime.go` | Create | `DriftReport`, `UpgradeReport`, `ModuleDrift` structs |
| `pkg/shipwright/testdata/api.golden` | Regenerate | **`-update` is a REQUIRED task step** |
| `.dagger/capabilities.go` | Modify | Layer 2 projection; `Upgrade` drops `error` (D-3) |
| `internal/workflow/manifest/validate.go` | Modify | Allowlist 5→7 + error message |
| `internal/workflow/providers/registry.go` | Modify | 2 tables + `Register`/`Resolve` pairs |
| `internal/workflow/providers/register.go` | Modify | Register `go-runtime` under both kinds |
| `internal/workflow/engine/execute.go` | Modify | 2 dispatch cases + with-field consts |
| `main.go` | Modify | `resolveCapabilityRef` +2 cases (D-8) |
| `providers/go/toolchain.go` | Create | Pure parse/conflict/mutate core — no Dagger |
| `providers/go/runtimeinspector.go` | Create | `GoRuntimeInspector.Inspect` |
| `providers/go/runtimeupgrader.go` | Create | `GoRuntimeUpgrader.Upgrade` |
| `providers/go/daggerkit/{interfaces,adapter,mocks}.go` | Modify | D-9 read/write surface |
| `providers/go/go.mod`, `go.sum` | Modify | +`golang.org/x/mod` |
| `providers/go/toolchainpin_test.go` | Create | D-4 guard, RED today |
| `providers/go/testdata/runtime/**` | Create | 8 workspace fixtures |

## Interfaces / Contracts

```go
// pkg/shipwright/capabilities.go — Layer 1
type RuntimeInspector interface {
    Inspect(ctx context.Context, source *dagger.Directory) (string, error)
}

type RuntimeUpgrader interface {
    Upgrade(ctx context.Context, source *dagger.Directory, targetVersion string) (*dagger.Directory, error)
}
```

`targetVersion` is a method parameter, not a struct field, precisely so the
type system forbids a zero-value default — an upgrade always carries an
explicit target. Everything else (`workspaceRoot`, `tidy`, `allowDowngrade`) is
provider-struct config bound from `with` at registration, exactly as
`RustBuilder.ManifestPath` already is.

```go
// .dagger/capabilities.go — Layer 2
type RuntimeInspector interface {
    dagger.DaggerObject
    Inspect(ctx context.Context, source *dagger.Directory) (string, error)
}

type RuntimeUpgrader interface {
    dagger.DaggerObject
    Upgrade(ctx context.Context, source *dagger.Directory, targetVersion string) *dagger.Directory
}
```

Manifest `with` schemas (must match `sdd-spec` exactly):

| Capability | Field | Kind | Required |
|---|---|---|---|
| `runtime-inspect` | `workspaceRoot` | String | no (default `.`) |
| `runtime-inspect` | `expectedVersion` | String | no |
| `runtime-inspect` | `failOnDrift` | Bool | no (default false, D-4b) |
| `runtime-upgrade` | `targetVersion` | String | **yes** |
| `runtime-upgrade` | `workspaceRoot` | String | no (default `.`) |
| `runtime-upgrade` | `tidy` | Bool | no (default true) |
| `runtime-upgrade` | `allowDowngrade` | Bool | no (default false) |

Engine confirmation, per proposal D1: `dispatchRuntimeInspect` and
`dispatchRuntimeUpgrade` are straight-line `Resolve* → call → wrap result`
functions, structurally identical to `dispatchBuild`. **No blocking, queueing,
waiting, approval, or scheduling code is added anywhere** — `execute.go`'s
package-doc claim stays literally true, and `workflow-execution` needs no delta.

## Testing Strategy

| Layer | What to test | Approach |
|---|---|---|
| Unit (pure) | `parseWorkspace`, `detectConflicts` A1–A6, `mutateGoMod`, `mutateGoWork` | Table-driven over `testdata/runtime/*` fixtures; no Dagger, no engine. **The TDD mass lives here.** |
| Unit (mocked) | `Inspect`/`Upgrade` happy path, nil-client, nil-source | Extended `providers/go/daggerkit` mocks (double-selection order rule 1) |
| Unit (guard) | `defaultGoVersion` == `go.mod`'s `go` | D-4; `daggerpin` mould; RED today |
| Unit (contract) | `capabilities_test.go` interface map | **Must add both new interfaces**, else D-A is unenforced for them |
| Unit (golden) | `pkg/shipwright/testdata/api.golden` | **REQUIRED `-update` run + reviewed diff** — CI fails the moment the interfaces compile |
| Unit (core) | allowlist, registry pairs, engine dispatch, `main.go` cases | Extend existing `validate_test.go`, `registry_test.go`, `fakes_test.go`, `dispatch_test.go`, `main_test.go` |
| Integration | One real-engine `Upgrade` over a temp workspace | `integration` build tag |

Fixtures: `single-module`, `workspace-3-modules`, `divergent-go` (A1),
`divergent-toolchain` (A2), `work-go-mismatch` (A3), `goversion-file-mismatch`
(A3, mirroring this repo's live `.go-version` 1.26.1 vs `go.mod` 1.26.7),
`downgrade` (A4), `malformed` (A5), `path-escape` (`use ../evil`).

One fixture is a byte-for-byte copy of this repository's own tier-1 sources, so
`detectConflicts` is proven RED against the real drift before any mutation code
exists.

## Threat Matrix

| Row | Applicability | Safe behavior | RED test |
|---|---|---|---|
| Path traversal via `go.work` `use` | **Applicable** | Reject absolute paths, and any path escaping the workspace root after `filepath.Clean` | `use ../../etc` fixture aborts |
| Arbitrary file write outside returned dir | **Applicable** | All writes via `Directory.WithNewFile` on an immutable value; zero host writes | Static import guard: no `os.WriteFile`/`os.Create`/`os.Remove` in the runtime files |
| Command construction from config values | **Applicable** | `targetVersion` validated against `modfile.GoVersionRE` **before** reaching `"golang:"+v` or argv; argv-array `WithExec` only, never `sh -c` | `"1.26.7; rm -rf /"` and `"--flag"` rejected at parse |
| Credential handling | N/A | Neither capability accepts a `*dagger.Secret`; no network/registry/git |  |
| Host subprocess | N/A | Every command runs in a Dagger container; no `exec.Command` |  |
| VCS/PR automation | N/A | D2 — no SCM code path exists |  |
| Plugin / dynamic load | N/A | Compiled-in registration only (D-I); `security_test.go` already proves it |  |

## Scope Questions Raised During Design

Raised mid-design; resolved here against the accepted proposal rather than
silently absorbed.

| Raised | Resolution |
|---|---|
| A third `runtime.verify` kind alongside `inspect`/`upgrade` | **Absorbed, not added** — D-4b's `failOnDrift` bool delivers gate semantics for ~6 lines instead of ~90 across six files. |
| Dot-separated names (`runtime.upgrade`) instead of hyphens | **Deferred as a mechanical rename.** `sdd-spec` is running in parallel against a proposal that literally writes `runtime-inspect`/`runtime-upgrade`; diverging now guarantees a mismatch. Nothing in the validator or interpolation grammar forbids a dot — this is a one-line allowlist edit whenever it is decided. |
| A `manual: true` step field | **Out of scope — proposal D1, accepted.** No `manual` field exists on `Step`, and `execute.go`'s package doc asserts zero blocking/queueing logic. "Manual, not automatic" is met by shipping no scheduler or webhook trigger, not by an engine gate. |
| Branch/PR creation after upgrade | **Out of scope — proposal D2, accepted.** This design's handoff point is exactly `Upgrade`'s returned `*dagger.Directory` plus its embedded report; the follow-up SCM change consumes both. No remote-write path exists in v1, which is the structural safety property standing in for the missing gate. |
| Fan-out to rust/java/python providers | **Already supported, zero extra work.** Both capability kinds are language-agnostic; `providers/rust` registers its own `RuntimeUpgrader` under the same kind with a different provider name, exactly as `rust` and `go` already coexist under `build`. Out of scope to *implement* here (proposal), but nothing in this design blocks it. |
| A four-phase `inspect→plan→apply→verify` provider pipeline | **Kept provider-internal, per the proposal's Approach.** Those four phases are the *internal* sequence of `Upgrade` (D-7 steps 1–6); exposing them as manifest steps would be exploration's rejected Approach 2 and would put phase-branching in core. |

## Reconciliation With `sdd-spec` (ran in parallel)

Capability strings converged exactly: `runtime-inspect` / `runtime-upgrade`.
Four points need an explicit decision before archive.

| # | `specs/runtime-toolchain/spec.md` | This design | Resolution |
|---|---|---|---|
| 1 | Go interface names assumed `RuntimeInspect`/`RuntimeUpgrade` (state.yaml flags design as owner) | `RuntimeInspector`/`RuntimeUpgrader` | **Design wins.** Agent-noun matches all five existing interfaces (`Builder`, `Tester`, `Artifactor`, `Deployer`, `Runner`). |
| 2 | L83: post-mutation validation "e.g. `go build`/`go vet` fails" | D-6: `go build ./...` only, `go vet` explicitly rejected | `e.g.` is illustrative, not normative, so no MUST is violated — but the mention should be trimmed to `go build` so it does not read as endorsing vet. |
| 3 | L71: on abort "the returned directory is byte-identical to the input" | `Upgrade` returns `(nil, err)` | Compatible (no mutation occurred, vacuously). `nil` is the safer contract: a caller that ignores the error gets a nil panic instead of silently shipping an untouched tree it believes was upgraded. Recommend rewording the scenario to "no file in the workspace was mutated". |
| 4 | L75–78 scenario contemplates "a CI pin file" | D-4 tier 2 puts CI workflow YAML out of scope | Satisfiable vacuously (absent ⇒ recorded absent, nothing fabricated). No ADDED requirement demands build-image or CI-YAML reporting, so D-4's substitution survives. `.go-version` is a superset the spec does not forbid. |

Also open from state.yaml: whether `public-module-api`'s "Composable,
Orthogonal Capabilities" requirement (which also enumerates the five capability
names) needs its own MODIFIED delta. **It does** — it enumerates the same
closed set that this change opens to seven.

## Migration / Rollout

No migration. Purely additive: no existing capability, manifest, or engine path
changes behavior. Rollback = revert; removing the two allowlist entries
restores the prior closed five-value set. One caveat: the `api.golden`
regeneration must revert with it.

## Review Budget Forecast

Authored lines (additions + deletions; generated `api.golden` excluded per the
Review Workload Guard):

| Area | Prod | Test |
|---|---|---|
| `pkg/shipwright` (2 interfaces + report structs) | 85 | 20 |
| `.dagger` projection | 25 | 25 |
| manifest allowlist | 4 | 20 |
| providers registry + register | 80 | 70 |
| engine dispatch | 50 | 90 |
| `main.go` (D-8) | 10 | 15 |
| `providers/go/daggerkit` (D-9) | 95 | — |
| `providers/go/toolchain.go` (parse/conflict/mutate, incl. `.go-version`) | 220 | 350 |
| `providers/go/runtimeinspector.go` (incl. `failOnDrift`) | 90 | 100 |
| `providers/go/runtimeupgrader.go` | 130 | 130 |
| D-4 guard test | — | 30 |
| go.mod/go.sum | 8 | — |
| **Subtotal** | **~797** | **~850** |

**Total ≈ 1,650 authored lines — 4.1× the 400-line budget.**

- `Decision needed before apply: Yes`
- `Chained PRs recommended: Yes`
- `400-line budget risk: High`

`delivery_strategy: single-pr` is **not achievable** for this design. The
orchestrator must either re-resolve to a chain or grant an explicit
`size:exception`. Recommended 4-slice chain, each independently shippable with
its own verification and rollback:

| Slice | Content | Authored |
|---|---|---|
| **1** | `runtime-inspect` end-to-end: Layer 1 + Layer 2 + allowlist + registry pair + engine + `main.go` case + read-only daggerkit + `toolchain.go` parse/conflict half (incl. `.go-version`) + `failOnDrift` + D-4 guard + tests. **Useful alone**: it detects this repo's live six-version drift with zero mutation risk. | ~520 |
| **2** | `runtime-upgrade` wiring + single-module `go.mod` + `.go-version` directive mutation + write-side daggerkit. | ~430 |
| **3** | `go.work` multi-module traversal + cross-module conflict detection (A1–A3) + path-escape guard. | ~370 |
| **4** | `go mod tidy` normalization, post-mutation `go build ./...` validation, go.sum-delta report fields, integration test. | ~290 |

Slice 1 carries the highest value-to-risk ratio and is the natural first PR.
Slice 2 alone ships a mutator whose only safety is fail-closed pre-analysis (no
post-mutation build check until slice 4) — acceptable because no remote-write
path exists (D2), but it must be stated in that PR's description.

## Open Questions

- [ ] **`sdd-spec` convergence on D-4.** If the parallel spec run wrote an
      ADDED requirement for build-image-pin reporting, it conflicts with this
      design and one of the two must yield. Design's position: the guard test.
- [ ] **`golang.org/x/mod` version pin in `providers/go/go.mod`.** Should match
      root's; confirm at apply time and consider extending
      `internal/daggerpin/pin_test.go`'s cross-module parity idea to it.
- [ ] **Provider name.** Design assumes one provider, `go-runtime`, registered
      under both capability kinds (mirroring how `golang.ContainerPublisher` is
      one type under one kind). Confirm against the naming convention in
      `register.go` at apply time.
- [ ] **Capability-string separator**: `runtime-inspect` (chosen, matches the
      accepted proposal and the repo's hyphenated provider names) vs
      `runtime.inspect` (reads better as a family). Mechanical rename; decide
      before slice 1 merges, not after.
- [ ] **`.dagger/go.mod` at 1.26.5 and the four `GO_VERSION: '1.26.1'` workflow
      pins are real drift this change deliberately does not fix** (D-4 tiers 2
      and 3). Worth a separate maintenance issue so they are corrected by an
      explicit decision rather than forgotten.

---

*Deviation note*: exceeds the skill's 800-word budget. Cause: the launch brief
required concrete Go signatures, exact `with` schemas, precise ambiguity rules,
a per-file line forecast, and an explicit slicing recommendation. Content is
compressed into tables; no narrative padding.
