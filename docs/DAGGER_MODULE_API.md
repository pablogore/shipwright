# Dagger Module API Reference (Layer 2 — `.dagger/`)

This document is the reference for Shipwright's **Layer 2** contract: the
Dagger module wrapper in `.dagger/capabilities.go`. If you call Shipwright
through the Dagger CLI (`dagger call ...`) or from another Dagger module,
this is your document.

Layer 2 re-declares the same five capabilities as
[`docs/API.md`](API.md) (Layer 1 — `pkg/shipwright`) as Dagger Interfaces,
and adds `Plan`, a composition object that chains them together. Shared
concepts (what each capability does, config structs, the secrets invariant)
are documented once in `docs/API.md` and linked from here, not repeated.

For what is version-guaranteed and how the five independent version axes
work, see [`COMPATIBILITY.md`](../COMPATIBILITY.md) (repo root). This
document does not restate that policy.

## Quick Path

```bash
dagger call \
  plan --source=. \
  with-build --build=<builder-module-ref> \
  with-test --test=<tester-module-ref> \
  with-artifact --artifact=<artifactor-module-ref> --ref=myapp:latest --creds=env:REGISTRY_TOKEN \
  execute
```

`Plan` starts empty with only a `Source`; each `With*` call chains one
capability and returns the same `Plan` for further chaining; `Execute` runs
everything that was chained, in order, and returns the last artifact or
deployment reference produced.

## Composition Surface

Signatures below are verbatim from
[`.dagger/capabilities.go`](../.dagger/capabilities.go).

### Shipwright.Plan

```go
func (m *Shipwright) Plan(source *dagger.Directory) *Plan
```

Starts a composition with only a `Source`; no capability is chained yet.

### Shipwright.ContractVersion

```go
func (m *Shipwright) ContractVersion() string
```

Mirrors `pkg/shipwright.ContractVersion` (see [`docs/API.md`](API.md#contractversion)).
Layer 2 cannot import Layer 1 — `.dagger/` is its own Go module with its own
`go.mod` — so this is a hand-kept duplicated literal, not a shared Go
constant. A mismatch between the two literals fails
`pkg/shipwright`'s `TestContractVersion_MatchesDaggerLayer2Literal`.

### Plan.WithBuild

```go
func (p *Plan) WithBuild(b Builder) *Plan
```

Chains a `Builder` into the `Plan`'s state.

### Plan.WithTest

```go
func (p *Plan) WithTest(t Tester) *Plan
```

Chains a `Tester` into the `Plan`'s state.

### Plan.WithArtifact

```go
func (p *Plan) WithArtifact(a Artifactor, ref string, creds *dagger.Secret) *Plan
```

Chains an `Artifactor` and the publish parameters its `Publish` method
requires into the `Plan`'s state.

### Plan.WithDeploy

```go
func (p *Plan) WithDeploy(d Deployer, environment string, creds *dagger.Secret) *Plan
```

Chains a `Deployer` and the deploy parameters its `Deploy` method requires
into the `Plan`'s state.

### Plan.WithRun

```go
func (p *Plan) WithRun(r Runner) *Plan
```

Chains a `Runner` into the `Plan`'s state.

### Plan.Execute

```go
func (p *Plan) Execute(ctx context.Context) (string, error)
```

Runs every chained capability in sequence over the build output:

1. **Build** (if chained) transforms `Source` into the build output, then is
   unconditionally synced so a build failure surfaces here rather than
   silently later.
2. **Test** (if chained) runs against the build output.
3. **Artifact** (if chained) publishes the build output and its resolved
   reference becomes the running `result`.
4. **Deploy** (if chained) deploys that `result`. **Fail-closed guard**:
   `Execute` rejects `Deploy` if `Artifact` was not chained first, returning
   an error (`"deploy requires an artifact to be chained first (WithArtifact
   before WithDeploy)"`) rather than deploying an empty artifact reference.
5. **Run** (if chained) runs the build output as a container.

`Execute` returns the last artifact or deployment reference produced, or an
empty string if neither `Artifact` nor `Deploy` was chained.

## The Dropped `error` Return

`Builder.Build`, `Tester.Test`, and `Runner.Run` on Layer 2 return only the
bare Dagger core type — no `error` — unlike their Layer 1 counterparts in
`pkg/shipwright`:

```go
// Layer 2 (.dagger/capabilities.go)
type Builder interface {
	dagger.DaggerObject
	Build(ctx context.Context, source *dagger.Directory) *dagger.Directory
}
```

```go
// Layer 1 (pkg/shipwright/capabilities.go), for comparison
type Builder interface {
	Build(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error)
}
```

This is **not a typo or an inconsistency**. It is a documented constraint of
the Dagger v0.21.8 Go SDK's codegen: the generated client-proxy code cannot
compile for a Dagger Interface method that returns a lazy-chainable Dagger
core type (`*dagger.Directory`, `*dagger.File`, `*dagger.Container`)
together with an explicit `error` — the generated `dagger.gen.go` emits a
single-value return for a signature that demands two, which fails to
compile. This is documented in the package doc of
[`.dagger/capabilities.go`](../.dagger/capabilities.go) (lines 15–31).

`Artifactor.Publish` and `Deployer.Deploy` are unaffected — they return
`(string, error)`, a non-lazy-chainable scalar pair, which compiles cleanly
on both layers.

**Where the error actually surfaces**: at the terminal/scalar call.
`Plan.Execute` unconditionally `Sync`s each lazy Dagger value it receives
from `Build`, `Test`, and `Run` before proceeding, and returns any resulting
error from `Execute` itself. A caller composing these interfaces directly
(outside `Plan`) must call `.Sync(ctx)` (or otherwise force evaluation) on
the returned `*dagger.Directory`/`*dagger.File`/`*dagger.Container` to
observe a failure.

## Boundaries

- **Guarantee and versioning policy**: see
  [`COMPATIBILITY.md`](../COMPATIBILITY.md). Not restated here.
- **Capability semantics, config structs, secrets invariant**: see
  [`docs/API.md`](API.md) (Layer 1). Not repeated here.
