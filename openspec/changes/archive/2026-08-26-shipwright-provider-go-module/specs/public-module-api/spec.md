# Delta for public-module-api

## ADDED Requirements

### Requirement: External Implementability Enforced by Automated Check

The public contract (`pkg/shipwright`) MUST remain sufficient — together with
Dagger's Go SDK and the standard library alone — to implement any shipped
capability as a standalone Go module. This property MUST be verified by an
automated, repeatable check (e.g. inspecting `go list -m all` from inside the
candidate module, or an equivalent import-graph check) that fails when a
capability implementation imports any `internal/**` package. Documentation or
manual review alone MUST NOT be treated as satisfying this requirement. This
requirement is permanent: it governs every capability shipped after this
change, not only the ones this change extracts.

#### Scenario: Standalone module builds against the contract alone

- GIVEN a capability implementation packaged as its own Go module
- WHEN `go list -m all` is run from inside that module
- THEN its only non-stdlib dependencies are the module owning
  `pkg/shipwright` and `dagger.io/dagger`

#### Scenario: Internal-package import fails the automated check

- GIVEN a capability implementation that imports any `internal/**` package
- WHEN the automated import-graph check runs against it
- THEN the check fails and reports the offending import

#### Scenario: Requirement applies to future capabilities

- GIVEN a new capability added after this change
- WHEN it is registered as a shipped capability
- THEN the same automated check MUST pass for it before it ships

### Requirement: Nested Provider Module Is Structurally Isolated

An extracted capability provider MUST live in its own Go module (for this
change: `providers/go`, module path
`github.com/pablogore/shipwright/providers/go`, package `golang`) importing
nothing from `internal/**`. The root `go.work` MUST include that module so
root `./...` (build, test, CI) spans it, and MUST NOT ever include `.dagger`.
An automated test MUST fail if `.dagger` is ever added to `go.work`.

#### Scenario: Provider module dependency graph is minimal

- GIVEN `providers/go/go.mod`
- WHEN its requirements are inspected
- THEN they list only the `pkg/shipwright`-owning module, `dagger.io/dagger`,
  and stdlib

#### Scenario: Root build and test span the provider module

- GIVEN `go.work` with `use .` and `use ./providers/go`
- WHEN `go build ./...` and `go test -race ./...` run from the repo root
- THEN both traverse `providers/go`

#### Scenario: `.dagger` isolation guard fails on violation

- GIVEN `go.work` modified to include `use ./.dagger`
- WHEN the isolation guard test runs
- THEN it fails

### Requirement: Extraction Preserves Registration, Distribution, and Behavior

`internal/workflow/providers/register.go` MUST register every extracted
capability under its pre-extraction provider name. The committed root
`go.mod` MUST resolve the nested provider module via a published,
path-prefixed git tag with no `replace` directive, so
`go install github.com/pablogore/shipwright@latest` continues to work
unchanged. An end-to-end manifest run MUST behave identically before and
after extraction. No file under `pkg/shipwright/**` MUST change as part of
an extraction.

#### Scenario: Provider names unchanged after extraction

- GIVEN `RegisterDefaults` after the `providers/go` extraction
- WHEN the registry is inspected
- THEN it registers capabilities under `"go"`, `"go-test"`,
  `"golangci-lint"`, `"govulncheck"`, and `"container"`, identically to
  before extraction

#### Scenario: `go install` succeeds with no committed `replace` directive

- GIVEN the committed root `go.mod` after extraction
- WHEN it is inspected
- THEN it contains no `replace` directive
- AND `go install github.com/pablogore/shipwright@latest` resolves and
  installs from a clean environment

#### Scenario: End-to-end manifest run is behavior-identical

- GIVEN `examples/workflow/diamond.yaml`
- WHEN it is run before and after the extraction
- THEN both runs resolve the same providers and produce equivalent results

#### Scenario: Public contract package is untouched by the extraction

- GIVEN the completed extraction
- WHEN `git diff` is inspected for the change
- THEN it touches zero files under `pkg/shipwright/`
