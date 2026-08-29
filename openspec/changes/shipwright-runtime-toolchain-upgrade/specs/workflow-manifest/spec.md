# Delta for Workflow Manifest

## MODIFIED Requirements

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
