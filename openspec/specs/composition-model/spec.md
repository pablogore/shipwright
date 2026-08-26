# Composition Model Specification

## Purpose

Defines the composition mechanism for combining Build, Deploy, Run, Test, and
Artifact capabilities into a composition result, retires the legacy
`Pipeline` interfaces that currently serve as Shipwright's public/DI-facing
contract, removes the `go-service` named capability-set preset, and renames
the public composition-result type away from `Pipeline`. Requirements
constrain properties and behavior of the chosen mechanism; they do not
select Dagger Interfaces/Objects, concrete Go interfaces, or internal-only
generics on the spec's behalf — that choice belongs to the design phase and
MUST satisfy every requirement below regardless of which mechanism is
picked.

**Terminology note:** "capability" here means the Go/Dagger contract type,
never this OpenSpec domain's own name nor a manifest step's `capability`
field (see `workflow-manifest`). "Composition type" below is a placeholder
for the renamed public type (e.g. `Plan`/`Compose`); the design phase picks
the exact name and this spec is satisfied by any name that is not, and does
not contain, "Pipeline".

## Requirements

### Requirement: Composition Mechanism Satisfies the Dagger Constraint

Whichever mechanism the design phase selects to compose capabilities, it MUST
be representable by Dagger's module type system and consumable through
generated cross-language SDK bindings, and MUST NOT require a Go generic type
parameter as part of the public contract.

#### Scenario: Composition result consumable cross-language

- GIVEN a pipeline assembled from two or more capabilities via the chosen
  composition mechanism
- WHEN the corresponding Dagger SDK binding is exercised in a second
  language (TypeScript or Python)
- THEN the composed pipeline executes with behavior equivalent to the
  Go-side composition

#### Scenario: Composition mechanism does not depend on Go generics

- GIVEN the exported composition function or type
- WHEN its public signature is inspected
- THEN it declares no Go generic type parameter

### Requirement: Capabilities Compose Without a Central Pipeline Type

The composition mechanism MUST allow assembling any subset of Build, Deploy,
Run, Test, and Artifact capabilities into a pipeline result without requiring
a pre-declared, named `Pipeline` struct per combination.

#### Scenario: Novel capability combination composes without new code

- GIVEN Build and Test capabilities already exist and are independently
  implemented
- WHEN a consumer composes them into a pipeline for the first time
- THEN no new named `Pipeline` type or registry entry needs to be authored
  for that combination

### Requirement: Legacy Pipeline Interfaces Retired

`internal/pipelines/pipeline.go` and `internal/interfaces/interfaces.go`
MUST no longer define a `Pipeline` type that serves as the SDK's extension
point. Both MUST be replaced by the new composition contract; coexistence
with the old shape as a parallel public surface is prohibited.

#### Scenario: Legacy Pipeline interfaces no longer define an extension point

- GIVEN `internal/pipelines/pipeline.go` and `internal/interfaces/interfaces.go`
  after migration
- WHEN their exported declarations are inspected
- THEN neither defines a `Pipeline` interface used as a public extension
  point

### Requirement: Dead Legacy Interface Deleted

`internal/pipelines/common/interfaces.go` MUST be deleted, since it is
confirmed dead code with no consumers.

#### Scenario: Dead interface file removed

- GIVEN the migrated repository
- WHEN `internal/pipelines/common/interfaces.go` is looked up
- THEN the file does not exist and no import references it

### Requirement: Existing Implementation and Wiring Migrated

The `go-service` pipeline implementation MUST be decomposed into standalone
capability implementations (per "No Named Capability-Set Preset Ships"
below, not migrated behind a compatibility flag), and the DI container
(`internal/app/container.go`, `pipeline_executor.go`, `step_registry.go`,
`hook_manager.go`) and the plugin layer (`internal/plugins/interfaces.go`)
MUST be migrated onto the new composition contract. All generated mocks
MUST be regenerated to match.
(Previously: described as "migrated," implying `--pipeline go-service`
compatibility was preserved. Corrected — see "No Named Capability-Set
Preset Ships": the CLI flag and preset registry are removed, not
preserved.)

#### Scenario: go-service decomposed and passing, no compatibility flag preserved

- GIVEN `internal/pipelines/go-service` after decomposition
- WHEN `go build -o shipwright .` and `go test -race ./...` run
- THEN both succeed with no reference to the retired `Pipeline` interfaces
  AND no `--pipeline go-service` CLI flag exists anywhere in `main.go`

#### Scenario: DI container and plugin layer compile against the new contract

- GIVEN `internal/app/container.go` and `internal/plugins/interfaces.go`
  after migration
- WHEN the package is built
- THEN `PluginContext.GetPipeline()` (or its replacement) returns a type
  defined by the new composition contract, not the retired
  `interfaces.Pipeline`

#### Scenario: Mocks regenerated

- GIVEN the new composition contract's interfaces
- WHEN `go.uber.org/mock`-based mocks are regenerated
- THEN generated mock files compile and satisfy the new contract with no
  reference to retired types

### Requirement: No Named Capability-Set Preset Ships

Shipwright MUST NOT ship a named capability-set preset — no `--pipeline
go-service` CLI flag, no registry of capability-set factories keyed by a
stack name, and no type or identifier that names a bundle of capabilities
as a single reusable unit. The `go-service` decomposition MUST produce
standalone, independently composable capability implementations with no
bundling identity; implementation names MUST describe what they do (e.g. a
Go builder, a Docker artifactor), never a stack bundle.

#### Scenario: No preset registry exists

- GIVEN the migrated repository
- WHEN it is searched for a registry of capability-set presets keyed by a
  stack name (e.g. `go-service`)
- THEN no such registry, factory map, or CLI flag exists

#### Scenario: go-service capabilities usable independently of each other

- GIVEN the capability implementations produced by decomposing `go-service`
- WHEN one of them (e.g. the Go builder) is composed without any of its
  former siblings
- THEN it composes and executes successfully with no reference to a
  `go-service` bundle identity

### Requirement: Composition Result Type Renamed Away From "Pipeline"

The public composition-result type (built via explicit `.With*()` calls,
per "Capabilities Compose Without a Central Pipeline Type" above) MUST NOT
be named `Pipeline`, and its exported name MUST NOT contain the substring
"Pipeline". The rename is preventive, not a correction of defective
behavior: the composition mechanism and its `.With*()` building pattern are
unchanged.

#### Scenario: Exported composition type name excludes "Pipeline"

- GIVEN the public composition-result type exported by the contract
- WHEN its exported Go identifier is inspected
- THEN it does not equal, and does not contain as a substring
  (case-insensitive), the word "Pipeline"

## Out of Scope

Selecting the composition mechanism's exact shape (Dagger Interfaces/Objects
vs. concrete Go interfaces vs. internal-only generics) is a design-phase
decision; this spec constrains its properties, not its implementation. Full
productionization (registry publication, CI `dagger call` wiring) is
deferred.
