# Shipwright — Product Requirements Document

Status: living document. This PRD describes product intent and invariants. It
is not an implementation spec — atomic engineering specs for individual
capabilities are created separately, outside this document.

## 1. Executive Summary

Shipwright is a Dagger-powered software delivery engine. It began as a
Go-specific CI/CD pipeline library under a previous project identity and is
evolving into a reusable, provider-neutral, polyglot delivery engine: define
software delivery once, execute it consistently anywhere — on a laptop, in
GitHub Actions, in GitLab CI, in Jenkins, or in any future CI provider.

Today, Shipwright ships a working Go binary/CLI, a Dagger-backed `go-service`
pipeline, a native (Docker-free) local executor, a Docker/Dagger executor, a
plugin/hook system, and GitHub Actions integration. It does not yet provide a
public, versionable Dagger Module API, a unified pipeline/step/artifact model,
multi-language toolchains, or an explicit Git-lifecycle engine. This document
describes both what exists and what Shipwright is becoming, and draws a firm
line between the two.

## 2. Problem Statement

CI/CD logic tends to fragment across a repository and across repositories:

- Provider YAML (GitHub Actions, GitLab CI, Jenkins) accumulates business
  delivery logic — build order, test gating, packaging, promotion rules —
  that only that provider understands.
- Local development scripts (`run-local.sh`, Makefiles) reimplement a second,
  usually looser version of the same logic so contributors can iterate without
  waiting on CI.
- Each service or repository re-derives its own pipeline, so the same
  standard steps (lint, test, build, scan, package, publish) are copied,
  forked, and drift independently.
- None of this is portable across languages or CI providers: adding a new
  language or switching CI vendors means rewriting delivery logic, not
  reusing it.

Shipwright's premise is that this delivery logic should live in one place, as
code, executed the same way everywhere, with CI providers reduced to thin
triggers.

## 3. Product Vision

> Define software delivery once. Execute it consistently anywhere.

Shipwright provides reusable delivery primitives for local development, CI
validation, release preparation, artifact creation, artifact promotion, and
deployment — without coupling the delivery model to a specific CI provider,
programming language, or deployment platform. Dagger is the portable
execution substrate; CI providers become thin orchestration shells that
trigger Shipwright.

Conceptually:

```
Developer / CI Provider
        |
        v
   Shipwright
        |
        +-- Lifecycle
        +-- Pipeline
        +-- Steps
        +-- Toolchains
        +-- Artifacts
        +-- Policies
        +-- Delivery Context
        |
        v
      Dagger
        |
        v
 reproducible execution
```

This is not "a collection of CI scripts," not "a GitHub Actions framework,"
and not "a Go pipeline runner." It is a programmable software delivery model
executed through Dagger.

## 4. Goals

- Provide one delivery model that runs identically on a developer's machine
  and in any CI provider.
- Let CI provider configuration (YAML, Groovy, etc.) contain orchestration
  only — trigger conditions, secrets wiring, environment selection — never
  build/test/release business logic.
- Support multiple languages/toolchains through a common pipeline shape,
  starting with Go, with Rust and Java as the next targets.
- Let pipelines be assembled from reusable steps, and let consumers
  customize, replace, insert, or disable steps without forking Shipwright.
- Make step outputs explicit, typed artifacts rather than filesystem or
  environment-variable conventions.
- Guarantee that a released artifact is built once and promoted unchanged
  through environments (build-once, promote), not rebuilt per environment.
- Keep critical build inputs (toolchain versions, base images) pinnable and
  reproducible.
- Model Git workflow lifecycle (feature/develop/release/main/hotfix)
  independently from language-specific build mechanics.
- Expose enough structured execution metadata to answer "what ran, on what
  revision, with what result, producing what artifact" after the fact.

## 5. Non-Goals

- Shipwright is not a general-purpose workflow/orchestration engine (e.g. not
  a replacement for Airflow, Temporal, or Argo Workflows).
- Shipwright does not aim to replace Dagger; it is an opinionated delivery
  model built on top of Dagger, not an alternative execution engine.
- Shipwright does not aim to be a full internal developer platform (no
  service catalog, no deployment UI) — deployment execution is in scope,
  platform UX is not.
- This PRD does not commit to a big-bang rewrite. Existing Go-centric
  behavior keeps working while the target architecture is introduced
  incrementally (see §20).
- This PRD does not specify Go types, package layouts, or file paths for
  unbuilt capabilities — that is the job of atomic specs written later.

## 6. Design Principles

1. **Provider neutrality** — GitHub Actions, GitLab CI, Jenkins, Buildkite,
   etc. only trigger Shipwright. Desired shape: `GitHub Actions -> Shipwright
   lifecycle ci -> Dagger`. Business delivery logic does not live in
   provider YAML.
2. **Local/CI parity** — the same delivery operation is executable locally
   and in CI through one semantic implementation, not two independent ones
   (a local script and a separate provider workflow expressing different
   behavior).
3. **Polyglot architecture** — language support is implemented through
   toolchains/adapters (`Toolchain { Go, Rust, Java }`), not through
   per-language pipeline forks. A service pipeline is a service profile
   plus a language toolchain, not a language-named pipeline.
4. **Composition over duplication** — pipelines are assembled from reusable
   steps (e.g. `checkout, dependencies, lint, test, build, security,
   package, publish`) rather than each pipeline reimplementing every step.
5. **Explicit customization** — consumers can eventually configure a step,
   disable optional behavior, add a step, replace a step, or insert a step
   before/after another step, without forking Shipwright.
6. **Typed artifacts** — steps communicate through explicit outputs
   (`SourceArtifact, TestReport, CoverageReport, BinaryArtifact,
   ContainerArtifact, SBOMArtifact, ReleaseArtifact`) rather than filesystem
   conventions or ad hoc paths.
7. **Build once, promote** — release/deployment preserves artifact identity:
   `commit SHA -> build -> artifact -> staging / production`. Production
   does not rebuild source code.
8. **Reproducibility** — avoid mutable execution dependencies such as
   uncontrolled `latest` tags. Toolchains and critical build inputs should be
   pinnable and reproducible.
9. **Git lifecycle separated from build mechanics** — Git workflow semantics
   (feature, develop, release, main, hotfix) are modeled independently from
   language/build implementation; lifecycle policy is not hard-coded into
   language-specific build steps.
10. **Observability and debuggability** — delivery execution eventually
    exposes structured information: lifecycle, pipeline, step, duration,
    result, artifact, source revision.

## 7. Current State

Verified against the repository (module `github.com/pablogore/shipwright`,
Go `1.26.1` per `go.mod`/`.go-version`, Dagger SDK `dagger.io/dagger v0.21.8`).

**Entry point.** A single Go binary/CLI (`main.go`, no `cmd/` package)
providing flags such as `--pipeline`, `--step`, `--env`, `--executor`,
`--local`, `--config`, `--list-pipelines`, `--list-steps`, `--version`,
`--health`. There is no separate public Dagger Module — Shipwright is
consumed as a compiled binary, not as a `dagger call`-able module.

**Pipelines.** A `Registry` (`internal/pipelines`) holds two registered
pipelines:
- `go-service` (`internal/pipelines/go-service`) — the only fully
  implemented pipeline: setup (clone or use checked-out source), test (with
  coverage via `shared.RunTestsWithCoverage`), lint (`golangci-lint` in a
  container), vulnerability scan (`govulncheck`), build (binary and/or
  Docker image), package, tag, and push to a registry.
- `infra` — registered but mostly a stub: `Test`, `Build`, `Package` are
  no-ops; `Tag` and `Push` return "not implemented" errors. Its internal
  type name and `Name()` value predate the Shipwright rename and are
  intentionally preserved as a distinct, company-specific identity (not a
  general "infra" toolchain).

**Duplicate pipeline abstractions.** There are two structurally similar but
distinct `Pipeline` interfaces: `internal/pipelines.Pipeline` (implemented by
`go-service`/`infra`) and `internal/interfaces.Pipeline` (used by the app
container, step registry, and plugin system, with `ExecuteStep`/
`GetAvailableSteps`). A `PipelineAdapter` in `internal/app/container.go`
bridges the two. This is exactly the "multiple Pipeline abstractions"
architectural debt the target architecture (§8–§9) is meant to resolve.

**Execution model.** Two independent execution paths exist:
- A **native/local path**: `internal/app/local_executor.go` and
  `internal/executors/native_executor.go` run steps directly with the host's
  Go toolchain, without Dagger/Docker, supporting `setup, build, test, lint,
  security` (not `package`, `tag`, `push`, `release`, which are silently
  skipped locally).
- A **Docker/Dagger path**: `internal/executors/docker_executor.go` and the
  `go-service` pipeline's own methods use `dagger.io/dagger` directly for
  container-based execution.

  The CLI auto-detects which to use (`--executor`, `--local`, or CI-env
  detection via `CI`/`GITHUB_ACTIONS`/etc.), but the two paths are separate
  implementations of overlapping behavior, not one semantic pipeline
  executed through two backends. This is the "local vs CI executor
  duplication" the target architecture addresses.

**Configuration.** A flat `pipelines.Config` struct (env, git ref/branch/
protocol, registry credentials, coverage, `GoVersion`, `JavaVersion`, SSH
key, etc.) built from `interfaces.Configuration`, itself backed by `koanf`
plus an optional `.shipwright.yml` (parsed by `internal/config/yaml_parser.go`
into a separate, narrower `YAMLConfig` struct). `JavaVersion` exists as a
config field with a default (`"17"`); no Java toolchain or pipeline consumes
it today — it is unused scaffolding, not a supported capability.

**Plugins/hooks.** `internal/plugins` defines a `Plugin` interface
(`Initialize`/`Cleanup`, hook registration) and a registry/loader wired into
the container; one built-in plugin (`nomad-deploy`) exists. `Pipeline.
BeforeStep`/`AfterStep` on both `go-service` and `infra` currently return
`nil` (no hook logic implemented at the pipeline level yet), even though the
hook manager and plugin context exist.

**CI/release tooling.** GitHub Actions workflows (`ci.yml`, `release.yml`,
`cleanup-branches.yml`) enforce branch protection, run tests/build, and
drive a GoReleaser-based release (multi-OS/arch archives plus renamed raw
binaries uploaded to GitHub Releases). A composite action
(`.github/actions/shipwright/action.yml`) wraps the CLI for GitHub Actions
consumers — a working, if GitHub-specific, example of "provider triggers
Shipwright." Its `pipeline` input description still references retired
pipeline names (`go-kit`, `docker-go`) that no longer exist in the registry.

**Reproducibility gaps.** The `go-service` pipeline pulls
`golangci/golangci-lint:latest` and `alpine:latest`, and defaults an image
tag to `"latest"` when none is configured — the mutable-version pattern
principle 8 is meant to eliminate.

**No typed artifacts.** Steps communicate through struct fields (e.g.
`Pipeline.Image *dagger.Container`) and host-filesystem paths (e.g. binaries
exported to `bin/`), not through a defined artifact contract.

**No public reusable Dagger Module API, no explicit lifecycle engine, no
Rust/Java toolchain, no build-once/promote mechanism, and no Git Flow
lifecycle modeling.** These are target-architecture items (§8, §11, §12,
§13), not current capabilities.

## 8. Target Architecture

```
GitHub Actions / GitLab CI / Jenkins / Local
                    |
                    v
                Shipwright
                    |
        +-----------+-----------+
        |           |           |
    Lifecycle   Pipeline    Toolchain
        |           |           |
        +-----------+-----------+
                    |
                  Dagger
                    |
                    v
          reproducible execution
```

- **Lifecycle** *(target)* — interprets Git/delivery lifecycle intent
  (feature validation, CI integration, release candidate, promotion,
  hotfix) and selects the pipeline/policy to run.
- **Pipeline** *(evolving)* — an ordered/composable execution graph built
  from steps; today expressed as a fixed sequence per pipeline
  implementation, not yet a composable graph.
- **Toolchain** *(target, Go partially exists today as pipeline-embedded
  logic)* — language/build-system-specific behavior (compile, test, lint,
  package) factored out from the pipeline so a pipeline is "service profile
  + toolchain" rather than a language-named pipeline.
- **Dagger** *(current)* — the portable execution substrate already in use
  for the `go-service` pipeline's container-based steps.

Consolidating the two current `Pipeline` interfaces into one, and factoring
Go-specific behavior out of `go-service` into a `Toolchain`, are prerequisite
steps toward this diagram — not yet done.

## 9. Domain Model

These are conceptual product terms. None imply a specific Go type, package,
or file location — that is design/spec work, not PRD scope.

- **DeliveryContext** *(target; today approximated by the current flat
  `pipelines.Config`)* — repository, commit SHA, branch, tag, lifecycle,
  environment, version, CI metadata: the immutable facts about *what* is
  being delivered and *where* it came from.
- **Lifecycle** *(target)* — a named delivery intent (e.g. `ci`, `feature`,
  `release`, `hotfix`, `deploy`) that maps to a pipeline and policy set.
- **Pipeline** *(current, not yet composable)* — an ordered execution graph
  of steps for a given delivery intent.
- **Step** *(current, fixed per pipeline; target: reusable/composable)* —
  the smallest reusable execution unit (e.g. lint, test, build).
- **Toolchain** *(target)* — language/build-system-specific behavior
  (Go, Rust, Java, …) implementing the steps a pipeline calls.
- **Artifact** *(target)* — a typed output produced by a step (source,
  binary, container image, test report, coverage report, SBOM, release
  artifact), replacing today's struct-field/filesystem conventions.
- **Policy** *(target)* — rules controlling validation, release, and
  deployment behavior (e.g. minimum coverage, required scans, promotion
  gates).

## 10. Pipeline Composition

Target behavior: a pipeline is assembled from an ordered set of reusable
steps (e.g. `checkout, dependencies, lint, test, build, security, package,
publish`), where:

- A consumer can configure a step (e.g. coverage threshold) without
  touching Shipwright's source.
- A consumer can disable an optional step (e.g. skip security scanning for
  a throwaway branch build) through configuration.
- A consumer can add a new step, replace an existing step's implementation,
  or insert a step before/after another named step — all without forking
  the framework.

Today, each pipeline (`go-service`, `infra`) hard-codes its own step
sequence in Go; there is no generic composition mechanism, override
mechanism, or step-insertion mechanism. This section defines the target
behavior; the mechanism (interfaces, registries, config schema) is
implementation detail for a future spec.

## 11. Polyglot Toolchains

Initial target toolchains: **Go, Rust, Java**.

- **Go** — closest to existing capability today: build, test with coverage,
  lint (`golangci-lint`), vulnerability scan (`govulncheck`) are implemented,
  but embedded directly inside the `go-service` pipeline rather than
  factored into a standalone `Toolchain` a pipeline composes with.
- **Rust** — not implemented. No Cargo/Rust-specific code exists in the
  repository today. Planned.
- **Java** — not implemented as a toolchain. A `JavaVersion` configuration
  field exists with a default value, but nothing in the pipeline or
  executor layer consumes it. Planned.

This PRD does not prescribe how each toolchain integrates (base images,
package managers, caching strategy) — that is scoped to per-toolchain specs
once the toolchain abstraction itself exists.

## 12. Git Lifecycle

Target, Git-Flow-compatible model (product behavior, not implementation):

- **Feature** — `feature/*` branches run validation; merge to `develop`.
- **Develop** — `develop` runs CI and integration validation on every change.
- **Release** — `release/*` branches run full validation and produce an
  immutable release-candidate artifact.
- **Main** — `main` represents released state; deploying from `main`
  promotes an existing artifact, it does not rebuild one.
- **Hotfix** — `hotfix/*` branches run validation, release, and promotion,
  then synchronize the fix back into the ongoing development flow.

This PRD intentionally does not specify exact Git commands, merge
automation, or branch-protection mechanics — those are implementation
choices for later design work. Today, branch protection and release
triggering exist only as GitHub-specific workflow logic (`ci.yml`,
`release.yml`), not as a provider-neutral lifecycle concept Shipwright
understands natively.

## 13. Artifact Lifecycle

Target invariant: **build once, promote.**

```
commit SHA -> build -> artifact -> staging -> production
```

- An artifact produced from a given commit SHA is built exactly once.
- Promotion to staging or production moves that same artifact forward; it
  does not trigger a new build from source.
- Production deployment must be traceable back to the exact commit SHA and
  build that produced the running artifact.

Today, `go-service`'s `Package`/`Push` steps build and publish a container
image per pipeline run; there is no promotion mechanism that reuses a
previously built artifact across environments — each environment run
implies a fresh build unless a consumer wires that up themselves.

## 14. CI Provider Integration

Target: CI providers are thin adapters that trigger Shipwright and pass
context (branch, event type, secrets); they do not encode delivery logic.

Current state: GitHub Actions is the only integrated provider, via a
composite action (`.github/actions/shipwright/action.yml`) that shells out
to the `shipwright` CLI per stage. This is a working example of the target
shape (thin YAML, CLI does the work) but is GitHub-specific, and its
documented `pipeline` input values are stale (they reference pipelines that
no longer exist). GitLab CI and Jenkins are documented only as usage
examples (`examples/github-actions`, no GitLab/Jenkins composite
equivalents) — they are not integrated or verified providers today.

## 15. Local Development Experience

Target: the same semantic pipeline runs locally and in CI, differing only in
executor, not in behavior.

Current state: local execution exists (`--local`, auto-detection, native
executor) but supports a reduced step set (`setup, build, test, lint,
security`) and explicitly skips steps that require "cloud services"
(`push, release, tag, package`) rather than running an equivalent local
version of them. This is a deliberate current limitation, not a semantic
parity guarantee — it is the gap principle 2 (local/CI parity) targets.

## 16. Configuration and Customization

Current state: configuration flows from defaults, environment variables
(`SHIPWRIGHT_*` via `koanf`), and an optional `.shipwright.yml` file into a
flat config struct. A plugin/hook system exists at the infrastructure level
(registry, loader, plugin context, hook manager) but pipelines do not yet
call into it (`BeforeStep`/`AfterStep` return `nil` on both current
pipelines).

Target: consumers configure step behavior, disable optional steps, and add,
replace, or insert steps through a defined extension surface — without
forking Shipwright. The plugin/hook scaffolding is a plausible foundation
for this, but the pipeline-level wiring to make it effective does not exist
yet.

## 17. Reproducibility

Target: pinned toolchain/base-image versions and deterministic execution
where Dagger allows it; no dependency on a mutable `latest` tag for
anything that affects build output.

Current state: this is not met. The `go-service` pipeline pulls
`golangci/golangci-lint:latest` and `alpine:latest` directly, and defaults
an unset image tag to `"latest"`, illustrating why version pinning needs to
be a first-class, enforced concept rather than scattered defaults.

## 18. Security

High-level expectations (not implementation detail):

- Secrets (registry credentials, SSH keys, CI tokens) must never become
  ordinary pipeline configuration values that get logged, cached, or
  embedded in an artifact.
- Dagger's secret primitives (already used today for registry
  authentication in `go-service.Push`) are the intended mechanism for
  secret handling going forward, including for any new toolchain.
- Vulnerability and dependency scanning (`govulncheck` today for Go) should
  extend to each new toolchain as it is added, as a policy-gated step, not
  an optional afterthought.
- Artifacts and their metadata (e.g. SBOM) should be attributable to a
  specific commit SHA and build, supporting later audit/compliance needs.

## 19. Observability

Target: structured execution metadata sufficient to answer, after the
fact: which lifecycle, which pipeline, which step, how long it took, what
the result was, what artifact (if any) it produced, and from which source
revision.

Current state: execution emits structured logs via `go-kit-logger`
(step name, pipeline name, environment, coverage, etc.) but there is no
persisted, queryable execution record — observability today is "read the
logs of this run," not "query delivery history across runs."

## 20. Compatibility / Migration

Shipwright does not require a big-bang rewrite to reach the target
architecture:

- The existing `go-service` pipeline remains the reference implementation
  while a `Toolchain` abstraction is introduced; Go behavior is factored
  out of the pipeline incrementally, not replaced wholesale.
- The two existing `Pipeline` interfaces (`internal/pipelines.Pipeline`,
  `internal/interfaces.Pipeline`) can be consolidated behind the existing
  `PipelineAdapter` boundary before any consumer-facing API changes.
- The current CLI and `.shipwright.yml` remain the primary interface while
  a public Dagger Module API is evaluated as an additive surface, not a
  replacement.
- New toolchains (Rust, Java) are additive registrations in the existing
  pipeline registry pattern; they do not require redesigning `go-service`.
- Lifecycle and artifact-promotion concepts can be introduced as new,
  optional layers on top of the existing pipeline/step execution rather
  than replacing today's step sequence immediately.

## 21. Success Criteria

Product-level, measurable criteria for "the target architecture has
arrived":

- The same pipeline definition runs locally and in CI with no step silently
  skipped or behaviorally different between the two.
- Provider YAML (GitHub Actions, GitLab CI, etc.) contains only
  orchestration (trigger, secrets, environment selection) — no build/test/
  release business logic.
- A new language can be added by implementing a `Toolchain`, without
  forking or duplicating an existing pipeline.
- A consumer can configure, disable, replace, or insert a pipeline step
  through configuration alone, without forking Shipwright.
- Promoting a build to production does not rebuild source code; the
  promoted artifact is provably the same one validated earlier in the
  pipeline.
- Every artifact can be traced back to the exact source commit SHA that
  produced it.
- Any public Dagger Module / API surface Shipwright exposes is versioned
  and can evolve without breaking existing consumers unannounced.

## 22. Roadmap

Organized by capability, not by date. This is capability-level planning
only; it does not assign implementation details, file paths, or spec
numbers — those belong to atomic specs created separately.

- **Foundation** — consolidate the two `Pipeline` interfaces into one;
  factor Go-specific logic out of `go-service` into a `Toolchain`; replace
  ad hoc struct-field outputs with a minimal typed-artifact contract.
- **Composition** — a pipeline definition assembled from reusable, ordered
  steps; support for configuring, disabling, adding, replacing, and
  inserting steps without forking.
- **Polyglot** — Rust and Java toolchains implementing the same step
  contract Go does today.
- **Lifecycle** — a `Lifecycle` concept (`ci`, `feature`, `release`,
  `hotfix`, `deploy`) that selects pipeline/policy, decoupled from any one
  Git hosting provider's workflow syntax.
- **Release** — build-once/promote artifact handling; an immutable
  release-candidate artifact concept for `release/*`/`main`.
- **Provider Integration** — GitLab CI and Jenkins adapters with parity to
  the existing GitHub Actions composite action; provider adapters kept
  thin by construction.
- **Hardening** — reproducibility (eliminate mutable `latest` dependencies,
  pin toolchain/base-image versions); structured, queryable execution
  observability; secret-handling review across all toolchains.

## 23. Open Questions

- Should the public extension surface be a Go API, a Dagger Module
  (`dagger call`-able), configuration-only, or some combination? This
  affects how "replace/insert a step" is realized.
- Where does `Lifecycle` policy live — in Shipwright configuration, in a
  separate policy artifact, or inferred from Git branch naming conventions
  alone?
- How should build-once/promote work when the promotion target is a
  different infrastructure/runtime than the build target (e.g. built as a
  container, promoted to a non-container platform)?
- What is the minimal typed-artifact contract that is useful without
  over-specifying every toolchain's output shape up front?
- Should the existing plugin/hook system be the extension mechanism for
  step customization, or superseded by a purpose-built composition API?
- How much of today's GitHub-specific branch-protection/release logic
  (`ci.yml`, `release.yml`) should move into Shipwright itself versus stay
  provider-side as legitimate provider configuration?
