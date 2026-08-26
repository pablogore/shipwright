// Stage 3 (structure) validation — tasks.md 4.2, workflow-manifest spec
// "capability Is The Contract, uses Is The Implementation" and "Explicit
// DAG Edges Via needs[]".
package manifest_test

import (
	"strings"
	"testing"

	"github.com/pablogore/shipwright/internal/workflow/manifest"
)

// identityHeader is a minimal valid document identity (stage 2), shared by
// every structure-level test so each test body only needs to add the
// spec.steps fixture under test.
const identityHeader = `
apiVersion: shipwright.dev/v1
kind: Workflow
metadata:
  name: example
spec:
  source: {path: .}
`

func TestParse_EmptyStepIDFailsValidation(t *testing.T) {
	src := identityHeader + `
  steps:
    - id: ""
      capability: build
      uses: {provider: go, version: "1"}
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with an empty step id must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "empty id") {
		t.Fatalf("Parse() error = %v, want an empty step id error", err)
	}
}

func TestParse_DuplicateStepIDFailsValidation(t *testing.T) {
	src := identityHeader + `
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: "1"}
    - id: build
      capability: test
      uses: {provider: go-test, version: "1"}
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with a duplicate step id must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate step id") {
		t.Fatalf("Parse() error = %v, want a duplicate step id error", err)
	}
}

func TestParse_CapabilityOutsideFiveFailsValidation(t *testing.T) {
	src := identityHeader + `
  steps:
    - id: build
      capability: compile
      uses: {provider: go, version: "1"}
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with a capability outside the five must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported capability") {
		t.Fatalf("Parse() error = %v, want an unsupported capability error", err)
	}
}

func TestParse_MissingUsesFailsValidation(t *testing.T) {
	src := identityHeader + `
  steps:
    - id: build
      capability: build
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with a step missing uses must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "missing uses") {
		t.Fatalf("Parse() error = %v, want a missing uses error", err)
	}
}

func TestParse_EmptyUsesVersionFailsValidation(t *testing.T) {
	src := identityHeader + `
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: ""}
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with an empty uses.version must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "empty uses.version") {
		t.Fatalf("Parse() error = %v, want an empty uses.version error", err)
	}
}

func TestParse_StepWithCapabilityAndUsesValidates(t *testing.T) {
	src := identityHeader + `
  steps:
    - id: build
      capability: build
      uses: {provider: maven, version: "1"}
`
	m, err := manifest.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(m.Spec.Steps) != 1 || m.Spec.Steps[0].ID != "build" {
		t.Fatalf("Spec.Steps = %+v, want one step with id build", m.Spec.Steps)
	}
}

// TestParse_DeclarationOrderDoesNotImplyDependency guards the
// workflow-manifest spec requirement that spec.steps[] list position never
// implies a needs[] edge: two steps declared consecutively with no needs[]
// between them must parse with no inferred dependency.
func TestParse_DeclarationOrderDoesNotImplyDependency(t *testing.T) {
	src := identityHeader + `
  steps:
    - id: a
      capability: build
      uses: {provider: go, version: "1"}
    - id: b
      capability: test
      uses: {provider: go-test, version: "1"}
`
	m, err := manifest.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(m.Spec.Steps[1].Needs) != 0 {
		t.Fatalf("Spec.Steps[1].Needs = %v, want no inferred dependency from declaration order", m.Spec.Steps[1].Needs)
	}
}

// TestParse_ExternalModuleVersionParsesIndependentlyOfContractVersion
// guards the workflow-manifest spec requirement that a `uses.module`
// reference's version is tracked as its own axis, distinct from
// ContractVersion — this package only proves the field parses; the actual
// independence-of-compatibility claim has no code path to violate here,
// since this package makes no ContractVersion comparison at all.
func TestParse_ExternalModuleVersionParsesIndependentlyOfContractVersion(t *testing.T) {
	src := identityHeader + `
  steps:
    - id: build
      capability: build
      uses: {module: "github.com/acme/custom-builder", version: "v3.2.1"}
`
	m, err := manifest.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	step := m.Spec.Steps[0]
	if step.Uses.Module != "github.com/acme/custom-builder" || step.Uses.Version != "v3.2.1" {
		t.Fatalf("Steps[0].Uses = %+v, want module github.com/acme/custom-builder at version v3.2.1", step.Uses)
	}
}

// TestParse_PolicyBlockParsesIntoStructuredValues guards the
// workflow-manifest spec requirement that spec.policies parses into typed
// fields consumable by a later enforcement layer.
func TestParse_PolicyBlockParsesIntoStructuredValues(t *testing.T) {
	src := identityHeader + `
  policies:
    dependencies: {forbidCycles: true}
    providers: {requireVersion: true}
`
	m, err := manifest.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if !m.Spec.Policies.Dependencies.ForbidCycles || !m.Spec.Policies.Providers.RequireVersion {
		t.Fatalf("Spec.Policies = %+v, want both forbidCycles and requireVersion true", m.Spec.Policies)
	}
}

// TestParse_DeclaredApprovalMetadataIsQueryable guards the
// workflow-manifest spec requirement that approval metadata parses as
// plain data with no attached blocking behavior (design.md D-M) — this
// package only proves the field is readable; enforcement absence is
// workflow-execution's concern (tasks.md 8.6).
func TestParse_DeclaredApprovalMetadataIsQueryable(t *testing.T) {
	src := identityHeader + `
  environments:
    production:
      approvals: {required: [platform-team]}
`
	m, err := manifest.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	approvals := m.Spec.Environments["production"].Approvals.Required
	if len(approvals) != 1 || approvals[0] != "platform-team" {
		t.Fatalf("Environments[production].Approvals.Required = %v, want [platform-team]", approvals)
	}
}

// --- source.ref validation (P1: contract enforcement) ---

func TestValidateSourceRef_BranchName(t *testing.T) {
	if err := manifest.ValidateSourceRef("main"); err != nil {
		t.Fatalf("ValidateSourceRef(%q) = %v, want nil", "main", err)
	}
}

func TestValidateSourceRef_BranchWithSlash(t *testing.T) {
	if err := manifest.ValidateSourceRef("feature/my-feature"); err != nil {
		t.Fatalf("ValidateSourceRef(%q) = %v, want nil", "feature/my-feature", err)
	}
}

func TestValidateSourceRef_TagName(t *testing.T) {
	if err := manifest.ValidateSourceRef("v1.2.3"); err != nil {
		t.Fatalf("ValidateSourceRef(%q) = %v, want nil", "v1.2.3", err)
	}
}

func TestValidateSourceRef_EmptyRefIsValid(t *testing.T) {
	if err := manifest.ValidateSourceRef(""); err != nil {
		t.Fatalf("ValidateSourceRef(%q) = %v, want nil", "", err)
	}
}

func TestValidateSourceRef_CommitSHATooLong(t *testing.T) {
	sha := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	if err := manifest.ValidateSourceRef(sha); err == nil {
		t.Fatalf("ValidateSourceRef(%q) = nil, want error", sha)
	}
}

func TestValidateSourceRef_CommitSHAShort(t *testing.T) {
	sha := "a1b2c3d"
	if err := manifest.ValidateSourceRef(sha); err == nil {
		t.Fatalf("ValidateSourceRef(%q) = nil, want error", sha)
	}
}

func TestValidateSourceRef_InvalidCharacters(t *testing.T) {
	if err := manifest.ValidateSourceRef("branch with spaces"); err == nil {
		t.Fatalf("ValidateSourceRef(%q) = nil, want error", "branch with spaces")
	}
}

func TestParse_SourceRefWithSHARejectsAtValidation(t *testing.T) {
	src := `
apiVersion: shipwright.dev/v1
kind: Workflow
metadata:
  name: example
spec:
  source:
    repo: "https://github.com/org/repo.git"
    ref: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: "1"}
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with a commit SHA as ref must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "looks like a commit SHA") {
		t.Fatalf("Parse() error = %v, want commit SHA rejection error", err)
	}
}
