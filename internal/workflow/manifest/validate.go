package manifest

import (
	"errors"
	"fmt"
	"sort"
)

// allowedAPIVersions is the schema's apiVersion allowlist (stage 2,
// design.md D-H). Design.md D-E: the manifest schema version evolves
// independently of ContractVersion, additive-only within v1 — an
// apiVersion outside this allowlist fails closed rather than being
// silently treated as the current version.
var allowedAPIVersions = map[string]bool{
	"shipwright.dev/v1": true,
}

// workflowKind is the only supported Kind value at this manifest schema
// version.
const workflowKind = "Workflow"

// validCapabilities is the manifest's capability allowlist — the five
// original contract types public-module-api defines (pkg/shipwright.
// {Builder, Tester, Artifactor, Deployer, Runner}), named as their
// manifest-facing lowercase strings, plus runtime-inspect
// (pkg/shipwright.RuntimeInspector, design.md D-4b) and runtime-upgrade
// (pkg/shipwright.RuntimeUpgrader, design.md D-9) — both from the
// shipwright-runtime-toolchain-upgrade change, hyphenated, matching this
// repo's existing hyphenated provider-name convention rather than the
// original five's single-word style.
var validCapabilities = map[string]bool{
	"build":           true,
	"test":            true,
	"artifact":        true,
	"deploy":          true,
	"run":             true,
	"runtime-inspect": true,
	"runtime-upgrade": true,
}

// ValidateIdentity implements stage 2 of the fixed seven-stage validation
// pipeline (design.md D-H, workflow-manifest spec "Versioned Document
// Identity"): apiVersion and kind must be present and within their
// allowlists, and metadata.name must be present. Nothing about step
// structure is inspected here — that is ValidateStructure's job (stage 3).
func ValidateIdentity(m *Manifest) error {
	if m.APIVersion == "" {
		return errors.New("manifest: missing required field apiVersion")
	}
	if !allowedAPIVersions[m.APIVersion] {
		return fmt.Errorf("manifest: unsupported apiVersion %q", m.APIVersion)
	}

	if m.Kind == "" {
		return errors.New("manifest: missing required field kind")
	}
	if m.Kind != workflowKind {
		return fmt.Errorf("manifest: unsupported kind %q, want %q", m.Kind, workflowKind)
	}

	if m.Metadata.Name == "" {
		return errors.New("manifest: missing required field metadata.name")
	}

	return nil
}

// ValidateStructure implements stage 3 of the fixed seven-stage validation
// pipeline (design.md D-H, workflow-manifest spec "capability Is The
// Contract, uses Is The Implementation"): every step id is non-empty and
// unique, capability is one of the five contract types, uses is present
// (a provider or a module), and uses.version is non-empty.
//
// needs[]/when/with are parsed by the schema but their CONTENTS are not
// interpreted here — resolving needs against known step ids belongs to
// stage 4 (internal/workflow/interp) and stage 5 (internal/workflow/graph,
// cycle/reference validation), which this package deliberately does not
// implement.
func ValidateStructure(m *Manifest) error {
	if m.Spec.Execution.Concurrency.MaxParallel < 0 {
		return fmt.Errorf("manifest: spec.execution.concurrency.maxParallel must be >= 0, got %d",
			m.Spec.Execution.Concurrency.MaxParallel)
	}

	seen := make(map[string]bool, len(m.Spec.Steps))

	for i, step := range m.Spec.Steps {
		if err := validateStep(i, step, seen); err != nil {
			return err
		}
	}

	return nil
}

func validateStep(index int, step Step, seen map[string]bool) error {
	if step.ID == "" {
		return fmt.Errorf("manifest: step[%d] has an empty id", index)
	}
	if seen[step.ID] {
		return fmt.Errorf("manifest: duplicate step id %q", step.ID)
	}
	seen[step.ID] = true

	if !validCapabilities[step.Capability] {
		return fmt.Errorf(
			"manifest: step %q has unsupported capability %q (must be one of build, test, artifact, deploy, run, runtime-inspect, runtime-upgrade)",
			step.ID, step.Capability,
		)
	}

	if step.Uses.Provider == "" && step.Uses.Module == "" {
		return fmt.Errorf("manifest: step %q is missing uses (provider or module)", step.ID)
	}

	if step.Uses.Version == "" {
		return fmt.Errorf("manifest: step %q has an empty uses.version", step.ID)
	}

	return nil
}

// UnenforceableControlError reports that a manifest declares a control this
// engine parses but does not enforce at runtime (design.md D2). Field is
// the full dotted path of the declared value (e.g.
// "spec.policies.secrets.forbidPlaintext"); Detail explains why the
// declaration is rejected rather than silently accepted.
type UnenforceableControlError struct {
	Field  string
	Detail string
}

func (e *UnenforceableControlError) Error() string {
	return fmt.Sprintf("manifest: %s is declared but not enforced by this engine: %s", e.Field, e.Detail)
}

// ValidateExecutable implements the execution-only fail-close gate
// (design.md D1, workflow-manifest spec "Policies Are Declared As
// Structured, Enforceable Schema Fields" and "Approval Gates Are Declared
// As Metadata Only", workflow-execution spec "Execution Controls"). It is
// deliberately NOT called by Parse/ParseFile — those stages back both the
// read-only --list-steps path and the execute path, and a read-only
// inspection command must still be able to parse and display a manifest
// declaring these controls (see the scenarios above). Callers on the
// execution path only MUST invoke ValidateExecutable themselves, after
// confirming the command is not read-only, and before any Dagger
// connection is established.
//
// Rejection is first-offender, in a fixed order: declared approvals
// (environment names visited in sorted order, so the reported field is
// deterministic — Environments is a map, and Go's map iteration order is
// randomized) → spec.policies.* (struct declaration order: secrets,
// providers, dependencies, artifacts) → spec.execution.concurrency.
// maxParallel > 1. There is no override flag or opt-out for any of these.
func ValidateExecutable(m *Manifest) error {
	if err := validateNoApprovals(m); err != nil {
		return err
	}
	if err := validateNoPolicies(m); err != nil {
		return err
	}
	if m.Spec.Execution.Concurrency.MaxParallel > 1 {
		return &UnenforceableControlError{
			Field:  "spec.execution.concurrency.maxParallel",
			Detail: "the engine executes waves strictly sequentially and cannot honor an asserted parallelism greater than 1",
		}
	}

	return nil
}

// validateNoApprovals rejects any environment declaring a non-empty
// approvals block. Environment names are visited in sorted order so the
// first-offender error is deterministic (design.md D2).
func validateNoApprovals(m *Manifest) error {
	names := make([]string, 0, len(m.Spec.Environments))
	for name := range m.Spec.Environments {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if len(m.Spec.Environments[name].Approvals.Required) > 0 {
			return &UnenforceableControlError{
				Field:  fmt.Sprintf("spec.environments.%s.approvals", name),
				Detail: "the engine implements no blocking, queueing, or wait-for-approval logic; approvals are metadata only",
			}
		}
	}

	return nil
}

// validateNoPolicies rejects any non-empty spec.policies.* declaration, in
// struct declaration order (secrets, providers, dependencies, artifacts).
// This includes forbidCycles/requireVersion despite ValidateStructure
// already unconditionally enforcing the equivalent behavior for other
// reasons (schema.go's Policies doc comment) — declaring the flag itself
// is still rejected, for consistency: a manifest cannot assert an intent
// this schema does not let it selectively enable or disable.
func validateNoPolicies(m *Manifest) error {
	p := m.Spec.Policies

	if p.Secrets.ForbidPlaintext {
		return &UnenforceableControlError{
			Field:  "spec.policies.secrets.forbidPlaintext",
			Detail: "not enforced by any layer; declaring it asserts a guarantee this engine does not provide",
		}
	}
	if p.Providers.RequireVersion {
		return &UnenforceableControlError{
			Field:  "spec.policies.providers.requireVersion",
			Detail: "uses.version is already required unconditionally; this flag has no effect and cannot be selectively enabled",
		}
	}
	if p.Dependencies.ForbidCycles {
		return &UnenforceableControlError{
			Field:  "spec.policies.dependencies.forbidCycles",
			Detail: "acyclicity is already enforced unconditionally; this flag has no effect and cannot be selectively enabled",
		}
	}
	if p.Artifacts.Immutable {
		return &UnenforceableControlError{
			Field:  "spec.policies.artifacts.immutable",
			Detail: "not enforced by any layer; declaring it asserts a guarantee this engine does not provide",
		}
	}

	return nil
}
