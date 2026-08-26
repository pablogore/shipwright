# Workflow Execution Specification

## Purpose

Defines the DAG engine that consumes a `workflow-manifest` document:
dependency-graph construction from `needs[]`, cycle detection, topological
execution ordering, capability→provider resolution, variable/secret
interpolation resolution, and the execution controls (concurrency, failure
strategy, timeouts, retries, conditional execution). States explicitly what
this engine does **not** do: it does not enforce approval gates.

**Terminology note:** "capability" here means the Go/Dagger contract type
(e.g. `Builder`) that a manifest step's `capability` field names — see
`workflow-manifest` and `public-module-api`. Never this OpenSpec domain's
own name.

## Requirements

### Requirement: Dependency Graph Built From `needs[]`

The engine MUST construct its execution graph from each step's `needs[]`
list, never from `spec.steps[]` declaration order.

#### Scenario: Execution order follows needs, not declaration order

- GIVEN a manifest declaring step B before step A, with B's `needs: [A]`
- WHEN the workflow executes
- THEN A completes before B starts, regardless of declaration order

### Requirement: DAG Acyclicity Enforcement Is Precise

`policies.dependencies.forbidCycles` MUST be enforced: a manifest whose
`needs[]` graph contains a cycle (self-edge, mutual pair, or longer cycle)
MUST be rejected before execution starts, with an error identifying the
cycle. A valid diamond fan-in/fan-out graph — two steps that both depend on
one prior step and both feed into one later step — MUST NOT be rejected.

#### Scenario: Cyclic manifest rejected

- GIVEN a manifest where step A `needs: [B]` and step B `needs: [A]`
- WHEN the manifest is validated
- THEN validation fails with a cycle-detection error and no step executes

#### Scenario: Valid diamond fan-in graph executes successfully

- GIVEN steps `test-unit` and `test-vuln` both `needs: [build]`, and step
  `artifact` `needs: [test-unit, test-vuln]`
- WHEN the workflow executes
- THEN `build` runs first, `test-unit`/`test-vuln` run after it (and MAY
  run concurrently), and `artifact` runs only after both complete — no
  cycle error is raised

### Requirement: Topological, Dependency-Respecting Execution Ordering

A step MUST NOT start until every step in its `needs[]` has completed
successfully. Steps with no dependency relationship between them MAY
execute concurrently.

#### Scenario: Step waits for all of its declared dependencies

- GIVEN step C with `needs: [A, B]`
- WHEN A completes but B has not yet completed
- THEN C has not started

### Requirement: Capability→Provider Resolution Verifies Interface Satisfaction Only

Resolving a step's `uses.provider` or `uses.module` MUST verify only that
the resolved implementation satisfies the declared `capability`'s Go/Dagger
interface. The engine MUST NOT contain provider-specific logic keyed by
provider name.

#### Scenario: Swapping providers requires no engine change

- GIVEN a step `capability: build` resolved via `uses.provider: maven`
- WHEN `uses.provider` is changed to `gradle` and the new provider
  implements `Builder`
- THEN the workflow resolves and executes with no change to engine code

#### Scenario: Provider not satisfying the declared capability fails closed

- GIVEN a step's resolved `uses` implementation does not implement the
  declared `capability`'s interface
- WHEN the engine resolves the step
- THEN resolution fails with an explicit error — it never silently skips
  the step

### Requirement: Variable And Step-Output Interpolation

`${{ variables.x }}` and `${{ steps.<id>.output }}` MUST resolve to their
corresponding values before a step that references them executes. Every
capability returns exactly one typed result, so a step's result has no
named sub-field — its kind is fixed by the producing capability.

#### Scenario: Downstream step receives an upstream step's output

- GIVEN step `build` (a Builder capability) produces its single typed
  output, and step `deploy` declares `with: {image: "${{ steps.build.output
  }}"}`
- WHEN `deploy` executes
- THEN it receives the actual value produced by `build`, not the literal
  token

### Requirement: Secret Interpolation Resolves To Typed Handles, Never Plaintext

`${{ secrets.x }}` MUST resolve end-to-end to a `*dagger.Secret`-typed
handle. It MUST NOT be string-substituted into a plaintext Go `string` at
any point in the resolution path. The interpolation mechanism MUST NOT
evaluate arbitrary expressions.

#### Scenario: Secret value never appears as a plaintext string

- GIVEN a manifest references `${{ secrets.registry-token }}`
- WHEN the engine resolves it
- THEN the value never appears as a Go `string` type at any point in the
  resolution path — only as a `*dagger.Secret`

#### Scenario: Non-whitelisted interpolation expression rejected at resolution

- GIVEN a manifest field containing an interpolation token outside the
  fixed `variables.`/`secrets.`/`steps.<id>.output` grammar
- WHEN the engine attempts to resolve it
- THEN resolution fails — the engine never evaluates it as an expression

### Requirement: Approval Gates Are Declared Metadata, Not Enforced By The Engine

The engine MUST NOT implement blocking, queueing, or "wait for approval"
logic for `spec.environments.<name>.approvals`. It MUST treat the declared
approval metadata as pass-through data available to any caller, and MUST
execute a step according to normal DAG ordering regardless of whether an
external approval has been recorded.

#### Scenario: Deploy step executes without an external approval signal

- GIVEN a manifest declares a `production` environment with required
  approvals for its `deploy` step, and no external approval has been
  recorded
- WHEN the workflow executes and `deploy`'s `needs[]` are satisfied
- THEN `deploy` executes according to normal DAG ordering — the engine
  does not block, queue, or wait for an approval state

#### Scenario: Approval metadata passes through unchanged

- GIVEN the same manifest
- WHEN a caller queries the environment's approvals metadata
- THEN the engine returns the declared metadata unchanged, without
  interpreting or gating on it

### Requirement: Execution Controls — Concurrency, Failure Strategy, Timeout, Retry

The engine MUST honor `spec.execution.maxParallel`, its failure strategy
(fail-fast or continue), per-step `timeout`, and per-step `retries`.

#### Scenario: maxParallel limits concurrent steps

- GIVEN `spec.execution.maxParallel: 1` and two independent, ready steps
- WHEN the workflow executes
- THEN the two steps run sequentially, never concurrently

#### Scenario: Fail-fast skips downstream dependents of a failed step

- GIVEN `spec.execution.failFast: true` and step A fails
- WHEN the engine continues processing the graph
- THEN any not-yet-started step whose `needs[]` includes A is skipped, not
  executed

### Requirement: Conditional Execution Via `when`

A step declaring a `when` condition MUST NOT execute when the declared
value is not present in the corresponding predicate list. `when` is a
structured YAML predicate map over the same restricted references as
interpolation (for example `when: {branch: [main, develop]}`), evaluated
by exact match — never a string expression with operators.

#### Scenario: Step with a non-matching condition is skipped

- GIVEN a step declares `when: {branch: [main]}` and the current branch is
  `develop`
- WHEN the workflow executes
- THEN the step is skipped because `develop` is not in the declared list

#### Scenario: Step with a matching condition executes

- GIVEN a step declares `when: {branch: [main, develop]}` and the current
  branch is `develop`
- WHEN the workflow executes
- THEN the step executes because `develop` is in the declared list

### Requirement: Declared Provider/Secret Policies Enforced At Validation Time

`providers.requireVersion` and `secrets.forbidPlaintext` MUST be enforced
before execution starts, mirroring `forbidCycles`: a manifest that violates
either MUST fail validation with an error naming the violated policy.

#### Scenario: Missing provider version rejected under requireVersion

- GIVEN `spec.policies.providers.requireVersion: true` and a step's `uses`
  omits `version`
- WHEN the manifest is validated
- THEN validation fails, naming the missing version

### Requirement: External `module:` Providers Satisfy The Same Dagger Contract As In-Repo Providers

A provider referenced via `uses.module` MUST satisfy the same Dagger
Interface contract as an in-repo `uses.provider`; the engine applies
identical resolution and interface-satisfaction checks to both.

#### Scenario: External module resolves identically to a local provider

- GIVEN two steps with the same `capability: build`, one using
  `uses.provider: maven` and the other `uses.module:
  github.com/acme/custom-builder`
- WHEN both are resolved
- THEN both resolve through the same interface-satisfaction check, with no
  special-cased path for the external module

### Requirement: Manifest-Driven Entrypoint Replaces The Preset CLI Path In The Same Change

The manifest-driven execution entrypoint MUST land no later than the
removal of the `go-service` preset (see `composition-model`'s "No Named
Capability-Set Preset Ships"). The repository MUST NOT reach a merged state
where the CLI has neither the preset flag nor a working manifest-driven
entrypoint.

#### Scenario: CLI always has a working execution path

- GIVEN the repository state after the `go-service` preset and its CLI
  flag are removed
- WHEN the CLI is invoked with a workflow manifest
- THEN the manifest-driven entrypoint executes it successfully — there is
  no point in the merged history where the CLI can run neither path

#### Scenario: Git-based source resolves via clone

- GIVEN a manifest with `spec.source.repo: "https://github.com/org/repo.git"` and `spec.source.ref: "main"`
- WHEN the CLI is invoked with the manifest
- THEN `resolveWorkflowSource` clones the repository and returns a valid
  `*dagger.Directory` — the "not implemented" error is no longer raised

#### Scenario: SSH-based source resolves via clone

- GIVEN a manifest with `spec.source.repo: "git@github.com:org/repo.git"` and `spec.source.ref: "main"`
- WHEN the CLI is invoked with the manifest
- THEN `resolveWorkflowSource` clones via SSH protocol and returns a
  valid `*dagger.Directory`

#### Scenario: Missing ref with repo fails closed

- GIVEN a manifest with `spec.source.repo: "https://github.com/org/repo.git"` but no `spec.source.ref`
- WHEN the CLI is invoked with the manifest
- THEN `resolveWorkflowSource` returns an error — source.ref is required when source.repo is set

## Out of Scope

A remote policy-as-code integration, an approval-workflow UI, or a
notification system. CI-system integration (GitHub Actions / GitLab CI
triggers) — this is a local/programmatic execution engine. A
package/module registry service for `module:` providers. Concrete provider
adapters beyond the minimum needed to demonstrate the engine.
