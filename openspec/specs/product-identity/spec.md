# Product Identity Specification

## Purpose

Defines the canonical public identity of the product as **Shipwright** — Go
module path, binary name, environment-variable prefix, default config
filename, CI/CD and release artifact naming, documentation, and the GitHub
repository identity — replacing the legacy `syntegrity-dagger` identity with
zero functional or behavioral change to the underlying Dagger-wrapping CLI.

## Requirements

### Requirement: Go Module Path and Imports

The system MUST declare its Go module path as `github.com/pablogore/shipwright`
in `go.mod`, and every internal Go import MUST reference this path.

#### Scenario: Module declaration

- GIVEN the repository's `go.mod`
- WHEN it is inspected
- THEN it declares `module github.com/pablogore/shipwright`
- AND no line references `github.com/getsyntegrity/syntegrity-dagger` or
  `github.com/pablogore/syntegrity-dagger`

#### Scenario: Internal package imports

- GIVEN any `.go` file importing an internal package
- WHEN the import path is inspected
- THEN it is prefixed with `github.com/pablogore/shipwright`

### Requirement: CLI Binary Name and Help/Version Output

The system MUST build a binary named `shipwright`, and its `--help` and
`--version` output MUST present only the Shipwright identity.

#### Scenario: Binary artifact name

- GIVEN the build command is run
- WHEN the resulting artifact is inspected
- THEN it is named `shipwright`, not `syntegrity-dagger`

#### Scenario: Help and version text

- GIVEN the `shipwright` binary
- WHEN invoked with `--help` or `--version`
- THEN output names the command `shipwright` and contains no
  `syntegrity-dagger` or `Syntegrity Dagger` string

### Requirement: Default Config Filename

The system MUST load default configuration from `.shipwright.yml` and MUST
NOT reference `.syntegrity-dagger.yml`.

#### Scenario: Default config lookup

- GIVEN no explicit config path is supplied
- WHEN the CLI starts
- THEN it looks for `.shipwright.yml` in the working directory
- AND no code path references `.syntegrity-dagger.yml`

### Requirement: Environment Variable Prefix

The system MUST read configuration overrides only from `SHIPWRIGHT_`-prefixed
variables (e.g. `SHIPWRIGHT_TOKEN`, `SHIPWRIGHT_VERSION`,
`SHIPWRIGHT_PIPELINE_COVERAGE`, `SHIPWRIGHT_PIPELINE_GO_VERSION`,
`SHIPWRIGHT_ENVIRONMENT`).

#### Scenario: Prefix recognized, old prefix not

- GIVEN `SHIPWRIGHT_TOKEN` is set in the environment
- WHEN the CLI loads configuration
- THEN the value is applied
- AND `SYNTEGRITY_DAGGER_TOKEN` is not read as a fallback

### Requirement: CI/CD Workflow and Composite Action Directory

CI/CD workflows MUST reference the renamed composite action directory
`.github/actions/shipwright/`, and `ci.yml`, `release.yml`, `dependabot.yml`,
`CODEOWNERS`, and `rulesets/README.md` MUST carry only the Shipwright
identity.

#### Scenario: Composite action path renamed

- GIVEN a workflow step using the composite action
- WHEN its `uses:` path is inspected
- THEN it references `.github/actions/shipwright/`
- AND `.github/actions/syntegrity-dagger/` no longer exists

### Requirement: GoReleaser Artifact and Install-URL Naming

Release configuration MUST produce artifacts, binary names, and install-URL
templates under the `shipwright` identity.

#### Scenario: Release artifact naming

- GIVEN `.goreleaser.yml`
- WHEN a release build is inspected
- THEN produced binaries/archives are named using `shipwright`
- AND install-URL templates reference `pablogore/shipwright`

### Requirement: Documentation Presents Shipwright Exclusively

`README.md` (including badge URLs), files under `docs/`, `CHANGELOG.md`, and
`examples/**` MUST present the product exclusively as Shipwright, including
rewritten historical CHANGELOG entries.

#### Scenario: README and badges

- GIVEN `README.md`
- WHEN inspected
- THEN it names the product Shipwright and any badge URL points to
  `pablogore/shipwright`

#### Scenario: CHANGELOG rewritten

- GIVEN `CHANGELOG.md`
- WHEN inspected
- THEN every entry references Shipwright, not Syntegrity Dagger

### Requirement: Test and Fixture Identifiers Updated

Test files, mocks, and fixtures MUST use Shipwright identifiers wherever they
encode product identity (module paths, env var names, config filenames)
without altering test intent or assertions.

#### Scenario: Fixture identifiers updated

- GIVEN a fixture referencing the config filename or env prefix
- WHEN inspected
- THEN it uses `.shipwright.yml` / `SHIPWRIGHT_*`
- AND asserts behavior equivalent to before the rename

### Requirement: Zero Functional Change (Non-Regression)

The rename MUST NOT alter runtime behavior, control flow, package layout, or
the `dagger.io/dagger` SDK integration.

#### Scenario: Build and test parity

- GIVEN the renamed codebase
- WHEN `go build` and `go test -race ./...` run
- THEN both succeed and coverage stays at or above the existing threshold

#### Scenario: Dagger SDK untouched

- GIVEN any file importing `dagger.io/dagger`
- WHEN inspected
- THEN the import and its usage are unchanged from before the rename

### Requirement: Zero Residual Old-Identity References

A case-insensitive repository-wide search for the product-identity pattern
`syntegrity[-_ ]?dagger` (covering `syntegrity-dagger`, `syntegrity_dagger`,
`SyntegrityDagger`, `SYNTEGRITY_DAGGER`, and `Syntegrity Dagger`) MUST return
zero hits, except the two documented product-identity exclusions below. A
separate, narrower search for the bare company name `syntegrity` (without
`dagger`) MAY return hits, but only where they fall under "Company/Org
References (Preserved)" in Out of Scope.

#### Scenario: Clean sweep with documented exceptions

- GIVEN the fully rebranded repository
- WHEN a case-insensitive search for `syntegrity[-_ ]?dagger` is run
- THEN the only matches are the Makefile's pre-existing
  `gitlab.com/syntegrity` coverage-filter grep and the unrelated root-level
  `1export` file

#### Scenario: Company identity left untouched

- GIVEN the fully rebranded repository
- WHEN a case-insensitive search for bare `syntegrity` is run
- THEN every remaining match belongs to the real external Syntegrity company
  identity (the `getsyntegrity` GitHub org, its `go-kit-logger` dependency,
  the `getsyntegrity.com` email domain, or company-owned example values) and
  none of them refer to this product

### Requirement: GitHub Repository Identity

The product's canonical remote repository MUST be `pablogore/shipwright`,
reachable via `go get` and release URLs.

#### Scenario: Repository renamed as an operational step

- GIVEN the code-level rename is complete
- WHEN the GitHub repository is renamed from `pablogore/syntegrity-dagger` to
  `pablogore/shipwright` as an operational (non-code) step
- THEN `github.com/pablogore/shipwright` resolves and matches the `go.mod`
  module path

## Out of Scope (Non-Requirements)

- `dagger.io/dagger` SDK imports — genuine third-party dependency, never
  touched.
- The Makefile's `grep -E "gitlab.com/syntegrity"` coverage filter — a
  pre-existing dead reference, unrelated to this product's identity.
- The root-level stray `1export` file — an unrelated shell-env dump.
- Backward-compatibility aliases or deprecation shims for the old identity —
  none are introduced; this is a clean rename.
- **Company/Org References (Preserved)** — the real external Syntegrity
  company/org identity, distinct from this product's old name, is never
  touched: the `github.com/getsyntegrity/go-kit-logger` dependency (`go.mod`,
  `go.sum`, every importing `.go` file), the unrelated `eventengine` code
  examples in `AGENTS.md`, the `getsyntegrity.com` email domain and
  `"Syntegrity CI"` git-author default in `internal/pipelines/shared/{ssh,https}_cloner.go`,
  the `$HOME/.ssh/syntegrity` default key path in `ssh_cloner.go`, and the
  `ghcr.io/syntegrity` example registry namespace in
  `examples/configs/tenant-svc.yml`.
