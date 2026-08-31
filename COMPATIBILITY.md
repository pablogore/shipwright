# Shipwright Compatibility Policy

This document is the policy file referenced by `design.md` D-E
(`openspec/changes/shipwright-public-module-api/design.md`). It defines what
Shipwright guarantees to stay stable, what it does not, and the five
independent version spaces a consumer needs to reason about.

## Guaranteed Surface

Shipwright's stability guarantee covers exactly the following, deliberately
minimal, surface — nothing else:

- **The five composition capability interfaces, on both contract layers**:
  - Layer 1 (`pkg/shipwright`, plain Go — `Builder`, `Tester`, `Artifactor`,
    `Deployer`, `Runner`)
  - Layer 2 (`.dagger/capabilities.go`, the Dagger projection of the same
    five as Dagger Interfaces)
- **The two runtime-maintenance capability interfaces, `RuntimeInspector` and
  `RuntimeUpgrader`** (`pkg/shipwright`, added by the
  shipwright-runtime-toolchain-upgrade change): guaranteed on Layer 1 and
  machine-enforced by the same `api.golden` below. Layer 2 also declares
  both as Dagger Interfaces (`.dagger/capabilities.go`), but they are not
  yet wired into `Plan`'s `WithBuild`/`WithTest`/.../`Execute` composition
  chain (design.md D-3) and are not covered by a Layer 2 golden — treat
  their Layer 2 shape as unguaranteed until that wiring lands.
- **`Shipwright.{Plan, ContractVersion}`** — the Layer 2 module entrypoint's
  two functions
- **`Plan.{WithBuild, WithTest, WithArtifact, WithDeploy, WithRun, Execute}`**
  — the composition surface (`.dagger/capabilities.go`)
- **The `pkg/shipwright` config structs** — `SourceConfig`, `BuildConfig`,
  `TestConfig`, `ArtifactConfig`, `DeployConfig`, `RunConfig`
- **The `pkg/shipwright` runtime report structs** — `DriftReport` (with its
  nested `ModuleVersion`/`ConflictState`) and `UpgradeReport` (with its
  nested `ModuleDrift`), returned by `RuntimeInspector.Inspect`/
  `RuntimeUpgrader.Upgrade` as JSON, never as a typed return value
  (design.md D-1)
- **The `shipwright.dev/v1` manifest schema**, as a *data* contract — the
  YAML document shape accepted by `--workflow`, guarded by its own golden
  (`internal/workflow/manifest/testdata/schema.golden`), independently of
  the Go/Dagger contract above

This exact enumeration is machine-enforced for the Go/Dagger surface by
`pkg/shipwright/testdata/api.golden` and its golden test
(`pkg/shipwright/api_golden_test.go`): any change to the guaranteed surface
must appear in a reviewed `-update` diff to that golden. `api.golden`
currently enumerates, in full: `ContractVersion`, the `Artifactor`/
`Builder`/`Deployer`/`Runner`/`RuntimeInspector`/`RuntimeUpgrader`/`Tester`
interfaces, and the `ArtifactConfig`/`BuildConfig`/`ConflictState`/
`DeployConfig`/`DriftReport`/`ModuleDrift`/`ModuleVersion`/`RunConfig`/
`SourceConfig`/`TestConfig`/`UpgradeReport` structs — i.e. Layer 1 in full.
The golden is a **textual diff, not a semantic type check**: it forces
every exported-surface change into a diff a human must read and
deliberately accept, never blind-accept.
It does not, on its own, reject a plaintext credential field; that guarantee
comes from the credential-typing discipline documented in `design.md`'s
Threat Matrix (all credential fields are `*dagger.Secret`, never a plain
`string`).

**Nothing outside this list carries a compatibility guarantee.** Internal,
non-exported packages (`internal/**`), example code (`examples/**`), and any
implementation detail of `providers/go/**` may change at any time without a
`ContractVersion` bump.

## Five Independent Version Axes

Revision 1 of this change's design declared three version spaces. The
declarative workflow layer added two more. These five are genuinely
independent — a change to one never implies a change to another, and no
two are ever conflated:

| # | Axis | Carrier | Guarantee |
|---|---|---|---|
| 1 | **Contract version** | `pkg/shipwright.ContractVersion` (also readable via `dagger call contract-version` on the Layer 2 module) | Stable from first release, covering only the guaranteed surface enumerated above |
| 2 | **Manifest schema version** | `apiVersion: shipwright.dev/v1` in each workflow document | Evolves independently of `ContractVersion`; additive-only within `v1` — a breaking schema change requires a new `apiVersion` |
| 3 | **CLI release SemVer** | goreleaser + `CHANGELOG.md` | Ordinary release versioning for the `shipwright` binary itself |
| 4 | **Engine pin** | `dagger.json` `engineVersion` | Must equal the root `go.mod` `dagger.io/dagger` client pin — enforced by a pin-parity unit test (`internal/daggerpin`); the two pins live in separate Go modules (root and `.dagger/`) and never link directly, so drift is the only residual risk, and that risk is a test |
| 5 | **Provider version (`uses.version`)** | A manifest step's `uses` block, owned by the provider that registers it | **Not covered by any Shipwright guarantee** |

## Provider-Version Exclusion (Explicit)

`uses.version` and `ContractVersion` are **orthogonal axes, not the same
version space.** The stable-from-first-release guarantee above covers only
the enumerated guaranteed surface; it does **not** extend to:

- in-repo providers registered through `internal/workflow/providers`
  (e.g. the `go`, `go-test`, `govulncheck`, `container`, `nomad-deploy`
  providers),
- third-party or plugin-contributed providers (e.g. providers registered by
  a `Plugin` through `PluginContext.GetProviderRegistry()`),
- any provider's declared `WithSchema` (its `with` key shapes), or
- a provider's own version semantics for `uses.version`.

A provider **may break its own users at any version without touching
`ContractVersion`.** A manifest author pins `uses.version` precisely because
Shipwright's own contract guarantee stops at the provider boundary — the
provider, not Shipwright, owns backward compatibility for its `with` schema
and behavior at each of its own versions.

## Non-Guarantees (Explicit)

- Approval gates (`spec.environments.<name>.approvals`), `spec.policies.*`,
  and `maxParallel > 1` were previously described here as silently
  accepted/ignored controls. As of PR #202
  (`manifest.ValidateExecutable`, `internal/workflow/manifest/validate.go`),
  that is no longer true: a manifest declaring any of them now **fails
  closed at execute time** with a typed
  `*manifest.UnenforceableControlError`, instead of running with the
  control silently unenforced. Specifically, `ValidateExecutable` — called
  from `main.go` immediately before the execution path connects to Dagger,
  but deliberately not from `Parse`/`ParseFile`, so a read-only command
  like `--list-steps` still parses and displays these fields — rejects, in
  fixed first-offender order:
  1. any `spec.environments.<name>.approvals` block present at all
     (visited in sorted environment-name order), regardless of whether
     `required` is populated;
  2. any explicitly-set `spec.policies.*` flag — `secrets.forbidPlaintext`,
     `providers.requireVersion`, `dependencies.forbidCycles`,
     `artifacts.immutable` — in that struct order, even when set to
     `false`, since an explicit value still asserts a control this engine
     does not selectively implement;
  3. `spec.execution.concurrency.maxParallel > 1`, since the engine
     executes waves strictly sequentially and cannot honor an asserted
     parallelism greater than 1.

  This is a behavioral property, not a versioned surface — the exact set of
  rejected controls can change without a `ContractVersion` bump — but it is
  called out here because "declares a value this engine cannot enforce" now
  means "manifest is rejected," not "value is silently ignored."
- Anything in `internal/**` — the DI container, the plugin loader's
  `LoadFromFile`/`LoadFromConfig` path, the step/hook registries, and the
  legacy `internal/pipelines/options.go` config struct — carries no
  compatibility guarantee at all.

## Bumping the Contract Version

A breaking change to the guaranteed surface above MUST:

1. Bump the major segment of `ContractVersion` in `pkg/shipwright/version.go`
   (and its hand-kept duplicate literal in
   `.dagger/capabilities.go`'s `Shipwright.ContractVersion()` — the two
   layers cannot share a Go constant, since `.dagger/` is a separate Go
   module that cannot import `pkg/shipwright`; a mismatch fails
   `pkg/shipwright`'s `TestContractVersion_MatchesDaggerLayer2Literal`).
2. Bump the Go module path to `/v2` (or higher) at contract v1 → v2 and
   above.
3. Ship a written migration note alongside the release.

A breaking change to the manifest schema instead bumps `apiVersion` (e.g.
`shipwright.dev/v2`) and is independent of the above — see Axis 2.
