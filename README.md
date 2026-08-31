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
- A compiled Go CLI/binary (`shipwright`) with a working `go-service`
  pipeline: setup, test (with coverage), lint, vulnerability scan, build
  (binary and/or Docker image), package, tag, push.
- Native (Docker-free) local execution for a subset of steps, and a
  Docker/Dagger-based execution path for the full pipeline.
- A declarative workflow manifest engine (`--workflow`, `shipwright.dev/v1`
  schema) that composes registered providers per step, independently of the
  `go-service` pipeline above. Providers registered today include the
  original five Go providers plus a full Rust equivalent set (`rust`,
  `rust-test`, `rust-integration-test`, `clippy`, `cargo-audit`,
  `rust-container`) and toolchain-drift `runtime-inspect`/`runtime-upgrade`
  capabilities.
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
- YAML + environment-variable configuration (`.shipwright.yml`,
  `SHIPWRIGHT_*`).
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

Shipwright ships as a single binary. It auto-detects whether it's running
locally or in CI and picks a native (no Docker) or Dagger/Docker-based
executor accordingly.

```bash
# Build from source
git clone https://github.com/pablogore/shipwright.git
cd shipwright
make build

# Run the go-service pipeline (auto-detects local vs CI execution)
./shipwright --pipeline go-service

# Run a single step
./shipwright --pipeline go-service --step test

# Force local (native) execution, no Docker required
./shipwright --pipeline go-service --local

# Force the Docker/Dagger executor
./shipwright --pipeline go-service --executor docker

# List what's available
./shipwright --list-pipelines
./shipwright --list-steps --pipeline go-service
```

Locally, only `setup, build, test, lint, security` run natively; steps that
need registry/cloud access (`package, tag, push, release`) are skipped in
local mode rather than run against real infrastructure.

Configuration can come from an optional `.shipwright.yml`:

```yaml
pipeline:
  name: go-service
  environment: dev
  coverage: 90
  goVersion: "1.26.1"
  steps:
    - setup
    - build
    - test

registry:
  baseUrl: registry.example.com
  image: my-service
  user: ${REGISTRY_USERNAME}

security:
  enableVulnCheck: true
  enableLinting: true

git:
  protocol: ssh
```

or overridden with CLI flags (`--env`, `--coverage`, `--executor`,
`--skip-push`, `--only-build`, `--only-test`, `--verbose`, and more —
run `shipwright --help` for the full set).

### Building from source

```bash
git clone https://github.com/pablogore/shipwright.git
cd shipwright
go mod download
make build   # builds ./shipwright
make test    # go test -race ./...
make lint    # golangci-lint
```

Requires Go 1.26 (see `go.mod` / `.go-version`) and Docker if you intend to
use the Docker/Dagger executor.

Compiled release binaries for Linux/macOS/Windows (amd64/arm64) are
published on the [GitHub Releases](https://github.com/pablogore/shipwright/releases)
page.

### GitHub Actions

A composite action wraps the CLI so provider YAML stays a thin trigger:

```yaml
- uses: actions/checkout@v4
- uses: ./.github/actions/shipwright
  with:
    pipeline: go-service
    stage: test
    env: dev
    coverage: '90'
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
already the real execution substrate for the `go-service` pipeline's
container-based steps.

## Current capabilities

Verified against the repository:

- `go-service` pipeline: test with coverage threshold, `golangci-lint`
  linting, `govulncheck` vulnerability scanning, binary and/or Docker image
  build, packaging, tagging, and registry push.
- `infra` pipeline: registered, but only `Setup`/test-with-coverage is
  implemented — build/package are no-ops and tag/push are not implemented.
- Declarative workflow manifests (`--workflow`) composing registered Go and
  Rust providers per step, independently of the `go-service`/`infra`
  pipelines above.
- A public, versionable Dagger Module API at the repository root (`dagger
  call`, `.dagger/capabilities.go`) exposing `Builder`/`Tester`/
  `Artifactor`/`Deployer`/`Runner` as chainable Dagger Interfaces via
  `Plan`/`Execute` — see `COMPATIBILITY.md` for the exact guaranteed
  surface.
- Native local executor and Docker/Dagger executor, auto-selected or
  forced via `--executor`/`--local`.
- YAML config file + `SHIPWRIGHT_*` environment variable configuration.
- Plugin registry/loader and a hook manager at the infrastructure layer
  (one built-in plugin, `nomad-deploy`); pipelines do not yet invoke
  before/after hooks.
- GitHub Actions composite action and example workflows.
- GoReleaser-based multi-platform release builds.

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
│   ├── pipelines/          # pipeline implementations (go-service, infra)
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
