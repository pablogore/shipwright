# Shipwright

[![Release Pipeline](https://github.com/pablogore/shipwright/actions/workflows/release.yml/badge.svg?branch=main)](https://github.com/pablogore/shipwright/actions/workflows/release.yml)

Shipwright is a Dagger-powered software delivery engine: it defines CI/CD
pipelines as code, once, so they can run the same way on your laptop and in
any CI provider.

## Why Shipwright?

CI/CD logic tends to spread out and drift: provider YAML (GitHub Actions,
GitLab CI, Jenkins) accumulates business delivery logic, local scripts
reimplement a looser version of the same thing for developer convenience,
and every repository re-derives its own version of "lint, test, build,
scan, package, publish." Shipwright's premise is that this logic should live
in one place, as code, executed consistently — with CI providers reduced to
thin triggers instead of the source of truth for delivery behavior.

## Status

Shipwright is under active architectural evolution. It started as a
Go-specific pipeline library and is moving toward a provider-neutral,
polyglot delivery engine built on [Dagger](https://dagger.io). Some of what
follows already works today; some of it is the direction the project is
heading. They are marked accordingly — do not assume a "planned" item is
already usable.

**Available today**
- A compiled Go CLI/binary (`shipwright`) whose sole entrypoint is a
  declarative workflow manifest engine (`--workflow`, `shipwright.dev/v1`
  schema) that composes registered providers per step. Providers registered
  today include five Go providers (setup/test, lint, vulnerability scan,
  build, container publish) plus a full Rust equivalent set (`rust`,
  `rust-test`, `rust-integration-test`, `clippy`, `cargo-audit`,
  `rust-container`) and toolchain-drift `runtime-inspect`/`runtime-upgrade`
  capabilities.
- A Docker/Dagger-based execution path for workflow steps.
- Rust provider support (`providers/rust`): a Go-implemented provider
  package — builder, unit/integration testers, linter, vulnerability
  scanner, container publisher — mirroring `providers/go`'s shape and
  consumed through the workflow manifest engine above. Proven standalone
  (`GOWORK=off`, no workspace, no `replace` directive) on every push via
  `make provider-rust-standalone`, and via a dedicated git-tag release
  workflow.
- A public, versionable Dagger Module API at the repository root (`dagger
  call`; see `.dagger/capabilities.go` and `COMPATIBILITY.md`). The five
  core capabilities (`Builder`/`Tester`/`Artifactor`/`Deployer`/`Runner`)
  are wired into a chainable `Plan`/`Execute` composition today, versioned
  via `ContractVersion` (currently `1.0.0`). Two further capabilities,
  `RuntimeInspector`/`RuntimeUpgrader`, are also declared as Dagger
  Interfaces but not yet wired into `Plan`'s composition chain.
- A GitHub Actions composite action that wraps the CLI.
- A plugin/hook registration system at the infrastructure level.

**Planned / evolving**
- One unified pipeline/step abstraction (two structurally similar `Pipeline`
  interfaces exist internally today, bridged by an adapter).
- Typed artifacts between steps (today steps communicate through struct
  fields and host paths).
- Reusable step composition — configuring, disabling, replacing, or
  inserting steps without forking Shipwright.
- An additional language toolchain (Java) — not implemented today (Rust
  ships today, see above).
- Wiring `RuntimeInspector`/`RuntimeUpgrader` into the Dagger Module API's
  `Plan`/`Execute` composition chain.
- Build-once/promote artifact handling and an explicit Git-lifecycle
  (feature/develop/release/main/hotfix) model.
- GitLab CI and Jenkins integration with parity to the GitHub Actions path.

See [docs/PRD.md](docs/PRD.md) for the full product vision, current-state
detail, and roadmap.

## How it works today

Shipwright ships as a single binary whose only entrypoint is a declarative
workflow manifest (`shipwright.dev/v1` schema). It executes steps against
a Dagger-provisioned environment.

```bash
# Build from source
git clone https://github.com/pablogore/shipwright.git
cd shipwright
make build

# Run every step in a workflow manifest
./shipwright --workflow path/to/workflow.yaml

# Run a single step (and its needs-transitive dependencies)
./shipwright --workflow path/to/workflow.yaml --step test

# List the steps declared in a manifest instead of executing them
./shipwright --workflow path/to/workflow.yaml --list-steps

# Select which branch predicate conditional steps evaluate against
./shipwright --workflow path/to/workflow.yaml --branch main
```

A missing or invalid manifest fails closed with an explicit error — there is
no fallback pipeline to run instead. `--workflow`, `--step`, `--list-steps`,
and `--branch` are the flags that actually affect a workflow run; see
`shipwright --help` for the full flag set, but note that several flags in
that list (`--executor`, `--local`, `--env`, `--coverage`, `--git-ref`,
`--git-auth`, `--config`/`.shipwright.yml`) are parsed but currently have no
effect on `--workflow` execution — everything a workflow needs (source,
secrets, variables, per-step options) is declared in the manifest itself.

### Building from source

```bash
git clone https://github.com/pablogore/shipwright.git
cd shipwright
go mod download
make build   # builds ./shipwright
make test    # go test -race ./...
make lint    # golangci-lint
```

Requires Go 1.26 (see `go.mod` / `.go-version`) and Docker, since workflow
steps run via Dagger.

Compiled release binaries for Linux/macOS/Windows (amd64/arm64) are
published on the [GitHub Releases](https://github.com/pablogore/shipwright/releases)
page.

### GitHub Actions

A composite action wraps the CLI so provider YAML stays a thin trigger:

```yaml
- uses: actions/checkout@v4
- uses: ./.github/actions/shipwright
  with:
    workflow: .shipwright/workflow.yaml
    step: test
    branch: develop
```

See [examples/github-actions](examples/github-actions/) for complete
workflow examples.

## Architecture direction

```
GitHub Actions / GitLab CI / Jenkins / Local
                    |
                    v
                Shipwright
                    |
        +-----------+-----------+
        |           |           |
    Lifecycle*  Pipeline    Toolchain*
        |           |           |
        +-----------+-----------+
                    |
                  Dagger
                    |
                    v
          reproducible execution
```

`*` marks target components that do not exist as standalone abstractions
yet — `Lifecycle` and `Toolchain` are goals of the ongoing architectural
evolution, not shipped concepts. `Pipeline` exists today but as two
overlapping internal interfaces rather than one unified model. `Dagger` is
already the real execution substrate for the workflow manifest engine's
container-based steps.

## Current capabilities

Verified against the repository:

- Declarative workflow manifests (`--workflow`, `shipwright.dev/v1` schema)
  composing registered Go and Rust providers per step: test with coverage
  threshold, `golangci-lint`/`clippy` linting, `govulncheck`/`cargo-audit`
  vulnerability scanning, binary and/or container image build, and
  toolchain-drift `runtime-inspect`/`runtime-upgrade`.
- A public, versionable Dagger Module API at the repository root (`dagger
  call`, `.dagger/capabilities.go`) exposing `Builder`/`Tester`/
  `Artifactor`/`Deployer`/`Runner` as chainable Dagger Interfaces via
  `Plan`/`Execute` — see `COMPATIBILITY.md` for the exact guaranteed
  surface.
- Dagger-provisioned execution for workflow steps.
- Plugin registry/loader and a hook manager at the infrastructure layer
  (one built-in plugin, `nomad-deploy`); pipelines do not yet invoke
  before/after hooks.
- GitHub Actions composite action and example workflows.
- GoReleaser-based multi-platform release builds.

## Legacy / internal historical implementation

`internal/pipelines/` still contains the original `go-service` and `infra`
pipeline implementations (setup/test/lint/scan/build/package/tag/push logic
predating the workflow manifest engine). **They are not invocable from the
current CLI** — the `--pipeline` flag and preset registry that used to
dispatch to them were removed, and `main.go`'s only entrypoint is
`--workflow`. The code remains in the tree as history/reference, not as a
supported delivery path.

## Roadmap

High-level themes (see [docs/PRD.md](docs/PRD.md) §22 for detail):

- Wire `RuntimeInspector`/`RuntimeUpgrader` into the Dagger Module API's
  `Plan`/`Execute` composition chain (both capabilities exist today but
  are not yet chained)
- Unified pipeline/step model (retire the duplicate `Pipeline` interfaces)
- Typed artifacts between steps
- Pipeline composition: configure, disable, add, replace, insert steps
- Additional polyglot toolchain: Java (Rust ships today via `providers/rust`)
- Explicit Git-lifecycle engine (feature/develop/release/main/hotfix)
- Build-once, promote: immutable release artifacts
- Provider-neutral integrations (GitLab CI, Jenkins) with GitHub Actions
  parity
- Reproducibility: eliminate mutable `latest` dependencies, pin toolchain
  versions
- Structured, queryable execution observability

## Development

```bash
make build     # build the shipwright binary
make test      # go test -race ./...
make lint      # golangci-lint
make coverage  # coverage report with threshold validation
```

Project layout:

```
shipwright/
├── main.go                 # CLI entry point
├── internal/
│   ├── app/                # DI container, executors, plugin/hook wiring
│   ├── config/             # configuration loading and validation
│   ├── executors/          # native and Docker/Dagger execution
│   ├── interfaces/         # shared interfaces
│   ├── pipelines/          # legacy pipeline implementations (go-service, infra) -- not invocable from the CLI
│   └── plugins/            # plugin/hook system
├── examples/                # usage examples (GitHub Actions, Jenkins, local)
└── docs/                    # documentation
```

## Documentation

- [Product Requirements Document](docs/PRD.md) — canonical product vision,
  current-state detail, target architecture, and roadmap
- [Architecture Guide](docs/ARCHITECTURE.md)
- [Local Usage Guide](docs/LOCAL_USAGE.md)
- [Pipeline Development](docs/PIPELINE_DEVELOPMENT.md)
- [Configuration Reference](docs/CONFIGURATION.md)
- [API Reference](docs/API.md)
- [Release Process](docs/RELEASE_PROCESS.md)
- [Examples](examples/)

## Support

- [GitHub Issues](https://github.com/pablogore/shipwright/issues)
- [GitHub Discussions](https://github.com/pablogore/shipwright/discussions)
