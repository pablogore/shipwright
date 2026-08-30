# Runtime Toolchain Specification

## Purpose

Defines two capability kinds — `runtime-inspect` (read-only) and
`runtime-upgrade` (mutating) — that let a language provider discover,
report, and (on request) correct drift in its own declarative
toolchain-version metadata (e.g. Go's `go.mod`/`go.work` directives),
with zero ecosystem-specific logic in Shipwright's core engine. Both
are single-method interfaces, matching `public-module-api`'s
one-capability-one-method invariant: a manifest declaring only
`runtime-inspect` has no code path capable of mutation.

## Requirements

### Requirement: Read-Only Drift Inspection

The `runtime-inspect` capability MUST accept a workspace
`*dagger.Directory` and return a structured drift report containing:
the version(s) discovered at each declarative location, the
configured target/expected version (if any), and an explicit
conflict/ambiguity state. `runtime-inspect` MUST NOT mutate the input
directory, write any file, or cause any side effect beyond the
returned report.

#### Scenario: Inspection produces zero mutation

- GIVEN a workspace directory with drifted toolchain versions across
  locations
- WHEN `runtime-inspect` runs against it
- THEN no file content in the workspace differs from the input
- AND no output other than the structured report is produced

#### Scenario: Ambiguous sources are reported, never guessed

- GIVEN a workspace where two declarative locations disagree with no
  resolvable precedence
- WHEN `runtime-inspect` runs
- THEN the report marks the conflict state explicitly, naming both
  sources and versions
- AND no single "winning" version is inferred

#### Scenario: Missing declarative location is omitted, not fabricated

- GIVEN a workspace with no `go.work` file
- WHEN `runtime-inspect` runs
- THEN the report contains no `go.work` entry
- AND no default or assumed value is fabricated for it

### Requirement: Discovery-Driven, Provider-Owned Upgrade

The `runtime-upgrade` capability MUST accept a workspace
`*dagger.Directory` and a target version, and return a mutated
`*dagger.Directory` plus a structured report. It MUST mutate only
declarative locations that actually exist in the input workspace
(discovery-driven) and MUST validate the result before returning it.

#### Scenario: Only existing locations are mutated

- GIVEN a workspace with `go.mod` but no `go.work`
- WHEN `runtime-upgrade` runs with a target version
- THEN `go.mod`'s `go`/`toolchain` directives are updated
- AND no `go.work` file is created

#### Scenario: Ambiguous sources abort with zero mutation

- GIVEN a workspace where declarative sources conflict with no
  resolvable precedence
- WHEN `runtime-upgrade` runs
- THEN it returns an error identifying the conflict
- AND no file in the workspace was mutated — the returned error carries
  no directory, so the caller receives no output to consume

#### Scenario: Missing declared location is skipped

- GIVEN a workspace with no CI pin file
- WHEN `runtime-upgrade` runs
- THEN the report records that location as absent
- AND no file is fabricated at that location

#### Scenario: Post-mutation validation failure is not silently returned

- GIVEN a mutation that would leave the workspace unable to pass its
  own post-mutation validation (e.g. `go build ./...` fails)
- WHEN `runtime-upgrade` runs
- THEN it returns an error and a report describing the validation
  failure
- AND it MUST NOT return a directory presented as successfully
  upgraded

### Requirement: No Network, Git, Or SCM Side Effects

Both `runtime-inspect` and `runtime-upgrade` MUST NOT perform any
network call, git operation, or SCM/PR side effect (branch creation,
push, pull-request creation). Their only interaction with the outside
world is the `*dagger.Directory` input and the
`*dagger.Directory`/report returned.

#### Scenario: No code path reaches network or SCM operations

- GIVEN the implementations of `runtime-inspect` and `runtime-upgrade`
- WHEN their call graphs are inspected
- THEN neither reaches an HTTP client, a git command, or an SCM/PR API
  call

### Requirement: Multi-Module Workspace Consistency

When a `go.work` file spans multiple `go.mod` modules,
`runtime-upgrade` MUST report per-module outcomes and MUST NOT leave
the workspace in a partially-mutated, unreported state: if any module
fails post-mutation validation, the operation's overall result MUST
reflect that failure.

#### Scenario: Multi-module workspace upgrades consistently

- GIVEN a `go.work` referencing three `go.mod` modules, all sharing
  the same drifted version
- WHEN `runtime-upgrade` runs with a target version
- THEN all three modules' directives are updated
- AND the report lists a per-module outcome for all three

#### Scenario: One module's validation failure fails the whole operation

- GIVEN a `go.work` referencing two modules, where one fails
  post-mutation validation
- WHEN `runtime-upgrade` runs
- THEN the overall result is a failure
- AND the report names which module failed and which succeeded
- AND the returned directory is not presented as a clean,
  fully-upgraded result
