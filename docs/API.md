  # API Reference (Layer 1 — `pkg/shipwright`)

This document is the reference for Shipwright's **Layer 1** contract: the
plain Go capability interfaces and config structs in `pkg/shipwright`. If
you import `github.com/pablogore/shipwright/pkg/shipwright` directly from
Go code, this is your document.

If you instead call the Dagger module (`dagger call ...` against
`.dagger/`), see [`DAGGER_MODULE_API.md`](DAGGER_MODULE_API.md) — the
**Layer 2** composition surface, which wraps these same capabilities for
Dagger's type system.

For what is version-guaranteed and how the five independent version axes
work, see [`COMPATIBILITY.md`](../COMPATIBILITY.md) (repo root). This
document does not restate that policy.

## Quick Path

A Go embedder composes capabilities directly — no `Plan`, no chaining
helpers, just calling interface methods against your own implementations:

```go
package main

import (
	"context"
	"log"

	"dagger.io/dagger"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

func main() {
	ctx := context.Background()

	client, err := dagger.Connect(ctx)
	if err != nil {
		log.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	source := client.Host().Directory(".")

	var builder shipwright.Builder = myBuilder{} // your Builder implementation

	build, err := builder.Build(ctx, source)
	if err != nil {
		log.Fatalf("build failed: %v", err)
	}

	if _, err := build.Sync(ctx); err != nil {
		log.Fatalf("build result failed: %v", err)
	}
}
```

Each capability interface (`Builder`, `Tester`, `Artifactor`, `Deployer`,
`Runner`) takes and returns plain Dagger core types
(`*dagger.Directory`, `*dagger.File`, `*dagger.Container`,
`*dagger.Secret`) plus Go scalars, `context.Context`, and `error`. There is
no shared base type and no required composition helper at this layer —
capabilities compose because their inputs and outputs line up, not because
of an interface contract between them.

## Surface Inventory

This table lists every entry in
[`pkg/shipwright/testdata/api.golden`](../pkg/shipwright/testdata/api.golden)
— the machine-enforced, textual-diff enumeration of the guaranteed Layer 1
surface — in the golden's own sorted order. Every row below has exactly one
corresponding doc section. If a future change updates `api.golden` (via a
reviewed `-update` diff), this table must gain or lose a matching row in the
same PR.

Regenerate the golden after an intentional surface change:

```bash
go test ./pkg/shipwright/... -run TestGuaranteedSurface_MatchesGolden -update
```

| # | Golden Entry | Kind | Doc Section |
|---|---|---|---|
| 1 | `ContractVersion` | const | [ContractVersion](#contractversion) |
| 2 | `ArtifactConfig` | struct | [ArtifactConfig](#artifactconfig) |
| 3 | `Artifactor` | interface | [Artifactor](#artifactor) |
| 4 | `BuildConfig` | struct | [BuildConfig](#buildconfig) |
| 5 | `Builder` | interface | [Builder](#builder) |
| 6 | `DeployConfig` | struct | [DeployConfig](#deployconfig) |
| 7 | `Deployer` | interface | [Deployer](#deployer) |
| 8 | `RunConfig` | struct | [RunConfig](#runconfig) |
| 9 | `Runner` | interface | [Runner](#runner) |
| 10 | `SourceConfig` | struct | [SourceConfig](#sourceconfig) |
| 11 | `TestConfig` | struct | [TestConfig](#testconfig) |
| 12 | `Tester` | interface | [Tester](#tester) |

> **Maintenance note**: `api.golden` currently enumerates 12 entries (1
> const, 5 interfaces, 6 structs). Any accepted `-update` diff to the golden
> — an addition, removal, or rename — MUST land a matching row change in
> this table in the same PR that changes the golden.

## Capabilities

Capabilities are independent — none depends on a sibling, and none shares a
base interface. Signatures below are verbatim from
[`pkg/shipwright/capabilities.go`](../pkg/shipwright/capabilities.go).

### Builder

```go
type Builder interface {
	Build(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error)
}
```

Builds a source `Directory` into a build-output `Directory`. Has no
knowledge of `Test`, `Artifact`, `Deploy`, or `Run`.

### Tester

```go
type Tester interface {
	Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error)
}
```

Runs tests against a build-output `Directory` and returns a report `File`.
Multiple independent `Tester` implementations may exist for the same input
(unit, lint, vulnerability scan, ...); none is privileged.

### Artifactor

```go
type Artifactor interface {
	Publish(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error)
}
```

Publishes a build-output `Directory` as a versioned artifact and returns its
resolved reference (for example an image reference).

### Deployer

```go
type Deployer interface {
	Deploy(ctx context.Context, artifactRef, environment string, creds *dagger.Secret) (string, error)
}
```

Deploys a previously published artifact reference into a named environment
and returns a deployment result reference. Note that `Deploy` takes
`artifactRef`, `environment`, and `creds` as method parameters — not through
`DeployConfig` (see [DeployConfig](#deployconfig) below).

### Runner

```go
type Runner interface {
	Run(ctx context.Context, build *dagger.Directory) (*dagger.Container, error)
}
```

Runs a build-output `Directory` as a live `Container`, for example to
execute it locally or expose it for interactive inspection.

## ContractVersion

```go
const ContractVersion = "1.0.0"
```

The current Layer 1 contract version. Bumping it is a deliberate,
guarantee-relevant act — see `COMPATIBILITY.md`'s "Bumping the Contract
Version" section for the full procedure, including the hand-kept literal
mirror in `.dagger/capabilities.go` and the test that keeps the two in sync.

## Configuration Structs

Signatures below are verbatim from
[`pkg/shipwright/config.go`](../pkg/shipwright/config.go).

### SourceConfig

Configures how a `Builder` obtains its source input (git checkout,
credentials). It carries no `Build`/`Test`/`Artifact`/`Deploy`/`Run` field —
orthogonality here is compiler-enforced, not documented.

```go
type SourceConfig struct {
	GitRepo       string
	GitRef        string
	GitProtocol   string
	GitUserEmail  string
	GitUserName   string
	SSHPrivateKey *dagger.Secret
}
```

### BuildConfig

Configures a `Builder` implementation.

```go
type BuildConfig struct {
	GoVersion   string
	JavaVersion string
	BuildMode   string
	BinaryName  string
}
```

`BuildMode` selects how a concrete `Builder` produces its output (e.g.
`"binary"`, `"docker"`, `"both"`) — it is left as a plain string here; each
`Builder` implementation owns its own enum.

### TestConfig

Configures a `Tester` implementation.

```go
type TestConfig struct {
	Coverage float64
}
```

### ArtifactConfig

Configures an `Artifactor` implementation.

```go
type ArtifactConfig struct {
	Registry      string
	RegistryURL   string
	RegistryUser  string
	RegistryPass  *dagger.Secret
	RegistryToken *dagger.Secret
	ImageName     string
	ImageTag      string
	BuildTag      string
	CommitSHA     string
	BranchName    string
	Version       string
	Token         *dagger.Secret
}
```

### DeployConfig

```go
type DeployConfig struct{}
```

**Currently empty.** No fields are declared. Concrete deploy adapters
(Kubernetes, Nomad, SSH, ...) are deferred to a follow-up change. This is
not an omission in this document — `api.golden` enumerates `DeployConfig`
as an empty struct today, and this section states that fact rather than
inventing fields. `Deployer.Deploy` already takes the parameters it needs
(`artifactRef`, `environment`, `creds`) directly through its method
signature, not through this struct.

### RunConfig

```go
type RunConfig struct{}
```

**Currently empty.** No fields are declared. Concrete run adapters are
deferred to a follow-up change, the same as `DeployConfig` above.

## Secrets Invariant

Every credential-bearing field or parameter in the Layer 1 contract crosses
as `*dagger.Secret`, never a plaintext `string`:

- `SourceConfig.SSHPrivateKey`
- `ArtifactConfig.RegistryPass`
- `ArtifactConfig.RegistryToken`
- `ArtifactConfig.Token`
- every `creds *dagger.Secret` method parameter (`Artifactor.Publish`,
  `Deployer.Deploy`)

This is enforced by convention and code review, not by the Go type system
alone rejecting a plain `string` at every call site — treat any new
credential-shaped field that is typed as `string` as a defect.

## Boundaries

- **Guarantee and versioning policy**: see
  [`COMPATIBILITY.md`](../COMPATIBILITY.md) for what is covered, the five
  independent version axes, and the contract-bump procedure. Not restated
  here.
- **Layer 2 (Dagger module) signatures**: `Builder.Build`, `Tester.Test`,
  and `Runner.Run` look different on the Dagger-module side — they drop the
  `error` return present here. This is a deliberate codegen constraint, not
  an inconsistency; see
  [`DAGGER_MODULE_API.md`](DAGGER_MODULE_API.md#the-dropped-error-return)
  for the full explanation.
