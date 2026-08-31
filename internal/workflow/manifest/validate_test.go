// ValidateExecutable (stage 3.5, execution-only fail-close gate) tests —
// tasks.md 5.1, workflow-manifest spec "Policies Are Declared As
// Structured, Enforceable Schema Fields" and "Approval Gates Are Declared
// As Metadata Only", workflow-execution spec "Execution Controls —
// Concurrency, Failure Strategy, Timeout, Retry".
package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/internal/workflow/manifest"
)

// TestValidateExecutable_RejectsDeclaredApprovals proves any declared
// approvals block fails execution validation (no opt-out), and that
// multiple declared environments are inspected in sorted name order so the
// reported field path is deterministic regardless of Go's randomized map
// iteration (design.md D2).
func TestValidateExecutable_RejectsDeclaredApprovals(t *testing.T) {
	src := identityHeader + `
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: "1"}
  environments:
    staging:
      approvals: {required: [qa-team]}
    production:
      approvals: {required: [platform-team]}
`
	m, err := manifest.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("manifest.Parse() error = %v, want nil", err)
	}

	err = manifest.ValidateExecutable(m)
	if err == nil {
		t.Fatal("ValidateExecutable() with declared approvals must return an error, got nil")
	}

	var uce *manifest.UnenforceableControlError
	if !errors.As(err, &uce) {
		t.Fatalf("ValidateExecutable() error = %v, want *manifest.UnenforceableControlError", err)
	}
	// "production" sorts before "staging" — the sorted-order guarantee.
	if uce.Field != "spec.environments.production.approvals" {
		t.Fatalf("UnenforceableControlError.Field = %q, want spec.environments.production.approvals (sorted order)", uce.Field)
	}
}

// TestValidateExecutable_RejectsPoliciesFields proves ANY non-empty
// spec.policies.* declaration fails validation, including a `true` value
// matching today's unconditional behavior for forbidCycles/requireVersion.
func TestValidateExecutable_RejectsPoliciesFields(t *testing.T) {
	tests := []struct {
		name      string
		policy    string
		wantField string
	}{
		{
			name:      "secrets.forbidPlaintext",
			policy:    "secrets: {forbidPlaintext: true}",
			wantField: "spec.policies.secrets.forbidPlaintext",
		},
		{
			name:      "providers.requireVersion",
			policy:    "providers: {requireVersion: true}",
			wantField: "spec.policies.providers.requireVersion",
		},
		{
			name:      "dependencies.forbidCycles",
			policy:    "dependencies: {forbidCycles: true}",
			wantField: "spec.policies.dependencies.forbidCycles",
		},
		{
			name:      "artifacts.immutable",
			policy:    "artifacts: {immutable: true}",
			wantField: "spec.policies.artifacts.immutable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := identityHeader + `
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: "1"}
  policies:
    ` + tt.policy + `
`
			m, err := manifest.Parse(strings.NewReader(src))
			if err != nil {
				t.Fatalf("manifest.Parse() error = %v, want nil", err)
			}

			err = manifest.ValidateExecutable(m)
			if err == nil {
				t.Fatalf("ValidateExecutable() with %s declared must return an error, got nil", tt.name)
			}

			var uce *manifest.UnenforceableControlError
			if !errors.As(err, &uce) {
				t.Fatalf("ValidateExecutable() error = %v, want *manifest.UnenforceableControlError", err)
			}
			if uce.Field != tt.wantField {
				t.Fatalf("UnenforceableControlError.Field = %q, want %q", uce.Field, tt.wantField)
			}
		})
	}
}

// TestValidateExecutable_RejectsMaxParallelAboveOne proves maxParallel > 1
// fails validation while 0 and 1 (sequential execution, indistinguishable
// from each other) are accepted.
func TestValidateExecutable_RejectsMaxParallelAboveOne(t *testing.T) {
	tests := []struct {
		name        string
		maxParallel string
		wantErr     bool
	}{
		{name: "omitted", maxParallel: "", wantErr: false},
		{name: "zero", maxParallel: "0", wantErr: false},
		{name: "one", maxParallel: "1", wantErr: false},
		{name: "two", maxParallel: "2", wantErr: true},
		{name: "four", maxParallel: "4", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execBlock := ""
			if tt.maxParallel != "" {
				execBlock = "\n  execution:\n    concurrency: {maxParallel: " + tt.maxParallel + "}\n"
			}
			src := identityHeader + execBlock + `
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: "1"}
`
			m, err := manifest.Parse(strings.NewReader(src))
			if err != nil {
				t.Fatalf("manifest.Parse() error = %v, want nil", err)
			}

			err = manifest.ValidateExecutable(m)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateExecutable() with maxParallel=%s must return an error, got nil", tt.maxParallel)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateExecutable() with maxParallel=%s error = %v, want nil", tt.maxParallel, err)
			}
			if tt.wantErr {
				var uce *manifest.UnenforceableControlError
				if !errors.As(err, &uce) {
					t.Fatalf("ValidateExecutable() error = %v, want *manifest.UnenforceableControlError", err)
				}
				if uce.Field != "spec.execution.concurrency.maxParallel" {
					t.Fatalf("UnenforceableControlError.Field = %q, want spec.execution.concurrency.maxParallel", uce.Field)
				}
			}
		})
	}
}

// TestValidateExecutable_AcceptsCleanManifest proves a manifest declaring
// none of the unenforceable controls passes validation.
func TestValidateExecutable_AcceptsCleanManifest(t *testing.T) {
	src := identityHeader + `
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: "1"}
`
	m, err := manifest.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("manifest.Parse() error = %v, want nil", err)
	}

	if err := manifest.ValidateExecutable(m); err != nil {
		t.Fatalf("ValidateExecutable() error = %v, want nil", err)
	}
}
