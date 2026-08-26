package manifest

import (
	"fmt"
	"regexp"
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
// contract types public-module-api defines (pkg/shipwright.{Builder,
// Tester, Artifactor, Deployer, Runner}), named as their manifest-facing
// lowercase strings.
var validCapabilities = map[string]bool{
	"build":    true,
	"test":     true,
	"artifact": true,
	"deploy":   true,
	"run":      true,
}

// ValidateIdentity implements stage 2 of the fixed seven-stage validation
// pipeline (design.md D-H, workflow-manifest spec "Versioned Document
// Identity"): apiVersion and kind must be present and within their
// allowlists, and metadata.name must be present. Nothing about step
// structure is inspected here — that is ValidateStructure's job (stage 3).
func ValidateIdentity(m *Manifest) error {
	if m.APIVersion == "" {
		return fmt.Errorf("manifest: missing required field apiVersion")
	}
	if !allowedAPIVersions[m.APIVersion] {
		return fmt.Errorf("manifest: unsupported apiVersion %q", m.APIVersion)
	}

	if m.Kind == "" {
		return fmt.Errorf("manifest: missing required field kind")
	}
	if m.Kind != workflowKind {
		return fmt.Errorf("manifest: unsupported kind %q, want %q", m.Kind, workflowKind)
	}

	if m.Metadata.Name == "" {
		return fmt.Errorf("manifest: missing required field metadata.name")
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
	if err := ValidateSourceRef(m.Spec.Source.Ref); err != nil {
		return err
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
			"manifest: step %q has unsupported capability %q (must be one of build, test, artifact, deploy, run)",
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

// refPattern matches valid Git branch and tag names per git-check-ref-format
// rules (simplified). Commit SHAs (40-char hex strings) are explicitly
// excluded to enforce the branch-or-tag-only contract.
//
// The pattern allows: letters, digits, dots, hyphens, underscores, slashes,
// and the special refs like HEAD — but rejects anything that looks like a
// raw SHA. This is intentionally conservative: a ref that passes this
// validation is guaranteed to be a valid branch/tag name for git clone --branch.
var refPattern = regexp.MustCompile(`^[a-zA-Z0-9._/\-@{~]+$`)

// shaPattern detects 7+ hex characters that look like a commit SHA prefix.
var shaPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// ValidateSourceRef validates that ref is a valid branch or tag name,
// NOT a commit SHA. When source.repo is set, ref must be non-empty
// (resolveWorkflowSource defaults to "main"). An empty ref is valid
// for path-based sources (source.repo is empty).
//
// The git clone --branch flag only accepts branch or tag names. Passing a
// commit SHA with --depth=1 produces undefined behavior depending on Git
// version. This validation makes the restriction explicit at parse time.
func ValidateSourceRef(ref string) error {
	if ref == "" {
		return nil
	}
	if shaPattern.MatchString(ref) {
		return fmt.Errorf("manifest: source.ref %q looks like a commit SHA; only branch or tag names are supported", ref)
	}
	if !refPattern.MatchString(ref) {
		return fmt.Errorf("manifest: source.ref %q contains invalid characters; only branch or tag names are allowed", ref)
	}
	return nil
}
