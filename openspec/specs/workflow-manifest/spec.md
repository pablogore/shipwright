# Workflow Manifest Specification

## Purpose

Defines the declarative YAML contract for a Shipwright workflow: document
identity and versioning, the separation between a step's `capability` (the
contract) and its `uses` (the implementation), variable and secret
referencing rules, provider version pinning, and the schema-level validation
rules a manifest must satisfy — including acyclicity as a *declared* policy
(enforcement is `workflow-execution`'s responsibility, not this schema's).

**Terminology note — "capability" means three things in this codebase; this
document uses only one of them:** (1) the Go/Dagger contract type
(`Builder`, `Tester`, ...) defined by `public-module-api`; (2) this file's
own OpenSpec domain name; (3) a manifest step's `capability` YAML field,
which names one of the five contract types from (1). Every bare use of
"capability" below means (3) unless stated otherwise.

## Requirements

### Requirement: Versioned Document Identity

A workflow manifest MUST declare `apiVersion`, `kind: Workflow`, and a
`metadata.name`. A manifest missing any of these three fields MUST fail
schema validation before any step is inspected.

#### Scenario: Manifest missing apiVersion fails validation

- GIVEN a YAML document with `kind: Workflow` but no `apiVersion`
- WHEN it is validated against the manifest schema
- THEN validation fails with an error naming the missing field

#### Scenario: Well-formed manifest identity validates

- GIVEN a YAML document with `apiVersion: shipwright.dev/v1`,
  `kind: Workflow`, and `metadata.name: example`
- WHEN it is validated
- THEN document-identity validation succeeds

### Requirement: `capability` Is The Contract, `uses` Is The Implementation

Each step MUST declare exactly one `capability` field naming one of
the seven capability interfaces defined by `public-module-api` (Build,
Test, Artifact, Deploy, Run, RuntimeInspector, RuntimeUpgrader), and
exactly one `uses` field identifying either a local provider
(`provider` + `version`) or an external provider (`module` +
`version`). A step missing `uses`, or naming a `capability` outside
the seven defined interfaces, MUST fail schema validation.
(Previously: the allowlist was exactly five values —
build/test/artifact/deploy/run — with no runtime-toolchain capability
kinds.)

#### Scenario: Step with capability and uses validates

- GIVEN a step `{id: build, capability: build, uses: {provider:
  maven, version: "1"}}`
- WHEN the manifest is validated
- THEN the step validates successfully

#### Scenario: Step missing uses fails validation

- GIVEN a step declaring `capability: build` with no `uses` field
- WHEN the manifest is validated
- THEN validation fails, naming the missing `uses` field

#### Scenario: Step declaring runtime-inspect validates

- GIVEN a step `{id: check-go, capability: runtime-inspect, uses:
  {provider: go, version: "1"}}`
- WHEN the manifest is validated
- THEN the step validates successfully

#### Scenario: Step declaring runtime-upgrade validates

- GIVEN a step `{id: bump-go, capability: runtime-upgrade, uses:
  {provider: go, version: "1"}, with: {target: "1.26.7"}}`
- WHEN the manifest is validated
- THEN the step validates successfully

#### Scenario: Capability outside the seven-value allowlist fails validation

- GIVEN a step declaring `capability: deploy-runtime` (not one of the
  seven allowed values)
- WHEN the manifest is validated
- THEN validation fails, naming the invalid `capability` value

### Requirement: Explicit DAG Edges Via `needs[]`

A step's dependencies MUST be declared explicitly through its `needs[]`
list. The manifest schema MUST NOT infer any ordering or dependency from a
step's position in `spec.steps[]`.

#### Scenario: Declaration order does not imply dependency

- GIVEN two steps declared consecutively in `spec.steps[]` with no
  `needs[]` relationship between them
- WHEN the manifest is parsed
- THEN no dependency edge is inferred between them from their list position

### Requirement: Secrets Referenced By Name, Never Embedded As Plaintext

`spec.secrets` entries MUST be name references resolved from an
environment or external source. The schema MUST NOT accept a literal secret
value inline under `spec.secrets`.

#### Scenario: Inline plaintext secret value rejected

- GIVEN a manifest with `spec.secrets.registry-token.value: "s3cr3t"` (a
  literal value, not a reference)
- WHEN the manifest is validated
- THEN validation fails

#### Scenario: Named secret reference without plaintext validates

- GIVEN a manifest with `spec.secrets.registry-token: {fromEnv:
  REGISTRY_TOKEN}` and a step field `${{ secrets.registry-token }}`
- WHEN the manifest is validated
- THEN validation succeeds and no plaintext value is present anywhere in
  the document

### Requirement: Provider Version Space Is Independent Of `ContractVersion`

`uses.version` (a provider's own version) MUST be tracked as a version axis
distinct from the public contract's `ContractVersion`. The manifest schema
MUST NOT couple a provider's version to the contract's compatibility
guarantee, and the guarantee defined in `public-module-api` does not extend
to a provider referenced via `uses`/`module`.

#### Scenario: External module version is independent of ContractVersion

- GIVEN a step `uses: {module: "github.com/acme/custom-builder", version:
  "v3.2.1"}`
- WHEN the manifest is validated against a contract at a given
  `ContractVersion`
- THEN validation does not depend on, and makes no claim about, the
  compatibility of `v3.2.1` with `ContractVersion`

### Requirement: Policies Are Declared As Structured, Enforceable Schema Fields

`spec.policies` MUST accept `secrets.forbidPlaintext`,
`providers.requireVersion`, `dependencies.forbidCycles`, and
`artifacts.immutable` as structured fields consumable by an execution
engine. This schema declares the fields; it does not itself enforce them —
enforcement is `workflow-execution`'s responsibility.

#### Scenario: Policy block parses into structured values

- GIVEN `spec.policies: {dependencies: {forbidCycles: true}, providers:
  {requireVersion: true}}`
- WHEN the manifest is parsed
- THEN both policy values are available as typed fields to a consuming
  engine

### Requirement: Approval Gates Are Declared As Metadata Only

`spec.environments.<name>.approvals` MUST be representable as a structured
object with a `required` field holding a list of reviewer names (e.g.
`approvals: {required: [platform-team]}`). The schema MUST NOT define any
execution or blocking semantics for this field — it is data, readable by
any caller or external system.

#### Scenario: Declared approval metadata is queryable, not executable

- GIVEN a manifest declaring `spec.environments.production.approvals:
  {required: [platform-team]}`
- WHEN the manifest is parsed
- THEN the `required` list is available as plain data to any caller, and
  the schema attaches no blocking behavior to it

### Requirement: Interpolation Tokens Use A Fixed Grammar, No Arbitrary Expressions

`${{ variables.x }}`, `${{ secrets.x }}`, and `${{ steps.<id>.output }}`
are the only supported interpolation forms. Every capability returns
exactly one typed result, so a step's result has no named sub-field. The
schema MUST reject a token containing an expression outside this fixed
grammar (arithmetic, function calls, or shell metacharacters).

#### Scenario: Fixed-grammar token accepted

- GIVEN a field value `${{ variables.registry }}`
- WHEN the manifest is validated
- THEN the token is accepted as a variable reference

#### Scenario: Arbitrary-expression token rejected

- GIVEN a field value `${{ 1 + 1 }}` or `${{ os.Exec("rm -rf /") }}`
- WHEN the manifest is validated
- THEN validation fails — the schema never accepts an expression outside
  the fixed `variables.`/`secrets.`/`steps.<id>.output` grammar

## Out of Scope

Concrete provider adapters (`maven`, `docker`, `tomcat`, ...) prove schema
shape only, not a commitment to ship them. A package/module registry
service for `module:` references is deferred — resolution assumes
local/vendored providers or an existing mechanism. Execution semantics
(graph traversal, provider resolution, interpolation resolution, approval
enforcement) belong to `workflow-execution`, not this schema.
