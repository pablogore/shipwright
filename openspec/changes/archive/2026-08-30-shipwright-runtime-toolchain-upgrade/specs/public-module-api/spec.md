# Delta for Public Module API

## MODIFIED Requirements

### Requirement: Versioned Contract Stable From First Release

The public contract MUST carry a documented SemVer-style
backward-compatibility guarantee effective from its first release. A
breaking change MUST require an explicit major-version bump and a
written migration note. The contract MUST expose a machine-readable
version marker at its own boundary, distinct from the CLI binary
release version (goreleaser/CHANGELOG) and the `dagger.json`
engine-version pin. The guarantee MUST apply only to the public
contract surface; internal packages carry no compatibility guarantee.
The guaranteed surface is exactly the seven capability interfaces
(Build, Test, Artifact, Deploy, Run, RuntimeInspector, RuntimeUpgrader)
and the `Shipwright`/composition-type surface already enumerated for
this contract (see `composition-model`); a workflow manifest's
`uses.version` (a provider's own version) is a separate,
non-overlapping version axis and is NOT covered by this guarantee,
even when the provider is referenced through `module:`.
(Previously: the guaranteed surface was exactly the five capability
interfaces — Build, Test, Artifact, Deploy, Run — with no
runtime-toolchain capability kinds.)

#### Scenario: Provider version is independent of the contract version

- GIVEN a workflow manifest step declaring `uses: {provider: maven,
  version: "2"}`
- WHEN the contract's `ContractVersion` is inspected
- THEN the two values are independent — changing `uses.version` never
  implies a `ContractVersion` change, and the compatibility guarantee
  does not extend to the referenced provider

#### Scenario: Version marker present and distinct

- GIVEN the public module contract
- WHEN its version marker is inspected
- THEN it resolves independently of the CLI binary's SemVer tag and
  the `dagger.json` engine pin

#### Scenario: Breaking change requires major bump and migration note

- GIVEN a change to the public contract that alters an existing
  exported capability signature
- WHEN the change is classified
- THEN it MUST bump the contract's major version and MUST ship a
  written migration note

#### Scenario: Internal package change carries no guarantee

- GIVEN a change to a non-exported, internal-only package
- WHEN the change is classified against the compatibility policy
- THEN it is exempt from the version-bump requirement

#### Scenario: Guaranteed surface count grows from five to seven

- GIVEN the public contract's guaranteed surface is inspected after
  `RuntimeInspector` and `RuntimeUpgrader` are added
- WHEN the interface count is checked
- THEN it is exactly seven — the original five plus the two
  runtime-toolchain interfaces
- AND a breaking change to either new interface is subject to the
  same major-bump-and-migration-note requirement as the original five

### Requirement: Composable, Orthogonal Capabilities

The public contract MUST decompose into small, orthogonal capabilities
— Build, Deploy, Run, Test, Artifact, RuntimeInspector,
RuntimeUpgrader. Each capability MUST be independently meaningful and
replaceable, and MUST NOT require knowledge of any sibling capability.
No single named type MUST be the SDK's central abstraction; a
composition result (see `composition-model`) or a declarative workflow
(see `workflow-manifest`/`workflow-execution`) MAY exist only as the
*result* of composing capabilities, never as a pre-declared,
per-combination type.
(Previously: the enumerated set was exactly the five capabilities —
Build, Deploy, Run, Test, Artifact — with no runtime-toolchain
capability kinds.)

#### Scenario: Capability composed without a concrete composition-result reference

- GIVEN a consumer imports the public capability package
- WHEN it composes a Build capability with a Deploy capability without
  importing or referencing any concrete, pre-declared composition-result
  struct new to that combination
- THEN the composition succeeds and produces a valid composition result

#### Scenario: Capability usable in isolation

- GIVEN only the Build capability is imported
- WHEN it is invoked without any Deploy, Run, Test, or Artifact capability
  present
- THEN it executes successfully with no compile-time or run-time dependency
  on the other capabilities

#### Scenario: RuntimeInspector and RuntimeUpgrader are independently usable

- GIVEN only the RuntimeInspector capability is imported
- WHEN it is invoked without RuntimeUpgrader, or any of Build, Deploy,
  Run, Test, or Artifact, present
- THEN it executes successfully with no compile-time or run-time
  dependency on any other capability
- AND the same independence holds for RuntimeUpgrader used without
  RuntimeInspector
