// Package manifest defines Shipwright's declarative workflow manifest
// schema (design.md D-H, the workflow-manifest spec): the typed Go structs
// a YAML workflow document decodes into, plus stages 1-3 of the fixed
// seven-stage validation pipeline — size-capped read + decode (stage 1),
// document identity (stage 2), and structure (stage 3).
//
// Stages 4-7 (references, graph, provider resolution, value binding)
// belong to later packages (internal/workflow/interp,
// internal/workflow/graph, internal/workflow/providers) and are
// deliberately NOT implemented here — see Parse's doc comment.
//
// Schema drift enforcement mirrors pkg/shipwright's guaranteed-surface
// golden (design.md D-E): testdata/schema.golden records the accepted
// field set, and any change to it forces a deliberate, reviewed `-update`
// diff and an explicit apiVersion decision in the same PR (design.md D-H).
package manifest

// Manifest is the top-level document shape: APIVersion + Kind identify the
// schema version (stage 2, ValidateIdentity); Metadata names the workflow;
// Spec holds everything else.
type Manifest struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

// Metadata identifies the workflow document itself.
type Metadata struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
}

// Spec is the workflow's declarative body.
type Spec struct {
	Source       SourceSpec             `yaml:"source"`
	Variables    map[string]string      `yaml:"variables,omitempty"`
	Secrets      map[string]SecretSpec  `yaml:"secrets,omitempty"`
	Steps        []Step                 `yaml:"steps"`
	Execution    ExecutionSpec          `yaml:"execution,omitempty"`
	Environments map[string]Environment `yaml:"environments,omitempty"`
	Policies     Policies               `yaml:"policies,omitempty"`
}

// SourceSpec selects the workflow's input source: a local Path, or a Git
// Repo+Ref pinned pair with an optional AuthSecretRef. Which combination a
// given provider actually requires is a later-stage (references/graph)
// concern, not a structural one — this package parses the shape only.
type SourceSpec struct {
	Path          string `yaml:"path,omitempty"`
	Repo          string `yaml:"repo,omitempty"`
	Ref           string `yaml:"ref,omitempty"`
	AuthSecretRef string `yaml:"authSecretRef,omitempty"`
}

// SecretSpec is a NAME REFERENCE to a secret resolved from an external
// source (an environment variable today) — never an inline plaintext value
// (workflow-manifest spec, "Secrets Referenced By Name, Never Embedded As
// Plaintext"). This struct deliberately declares no field capable of
// holding a literal secret value: a manifest attempting
// `spec.secrets.<name>.value: "..."` fails at stage 1 decode
// (yaml.Decoder.KnownFields(true) rejects the unknown `value` field), not
// at a later validation stage — the schema shape itself is the
// enforcement mechanism, mirroring D-D's retyping of credentials to
// *dagger.Secret and the internal/pipelines/shared/docker.go
// client.SetSecret pattern this mechanism must reuse.
type SecretSpec struct {
	FromEnv string `yaml:"fromEnv"`
}

// Step is one node in the workflow DAG. Capability is the contract (one of
// public-module-api's five capability interfaces, named build/test/
// artifact/deploy/run); Uses is the implementation. Needs declares
// explicit DAG edges — a step's position in Spec.Steps never implies an
// edge (workflow-manifest spec, "Explicit DAG Edges Via needs[]"). Outputs
// is deliberately NOT a field here: every capability returns exactly one
// typed result, addressable as `${{ steps.<id>.output }}` (design.md D-H
// schema simplification — the proposal's field list was explicitly
// non-normative).
type Step struct {
	ID         string              `yaml:"id"`
	Capability string              `yaml:"capability"`
	Uses       UsesSpec            `yaml:"uses"`
	Needs      []string            `yaml:"needs,omitempty"`
	Input      string              `yaml:"input,omitempty"`
	With       map[string]any      `yaml:"with,omitempty"`
	When       map[string][]string `yaml:"when,omitempty"`
	Attempts   *int                `yaml:"attempts,omitempty"`
}

// UsesSpec names the step's implementation: either an in-repo Provider or
// an external Module (never both meaningfully at once — which is used is a
// stage-6 provider-resolution concern), both pinned to a required Version.
// Version is design.md D-E's provider-version axis — independent of, and
// not covered by, ContractVersion's compatibility guarantee.
type UsesSpec struct {
	Provider string `yaml:"provider,omitempty"`
	Module   string `yaml:"module,omitempty"`
	Version  string `yaml:"version"`
}

// ExecutionSpec declares scheduling intent. MaxParallel (via Concurrency)
// is validated and recorded as an upper bound (design.md D-K); the engine
// (Phase 8, out of scope here) is not required to use it to widen
// execution — sequential waves are a correct schedule for any
// maxParallel >= 1.
type ExecutionSpec struct {
	Concurrency ConcurrencySpec `yaml:"concurrency,omitempty"`
	FailFast    bool            `yaml:"failFast,omitempty"`
	Timeout     string          `yaml:"timeout,omitempty"`
}

// ConcurrencySpec bounds parallel execution within a wave.
type ConcurrencySpec struct {
	MaxParallel int `yaml:"maxParallel,omitempty"`
}

// Environment names a deployment target and its declared approval
// metadata. The engine never blocks, queues, or waits on approvals
// (design.md D-M); any DECLARED approvals block — even an empty one — is
// rejected outright by ValidateExecutable (see the Policies comment).
// Approvals is a pointer so ValidateExecutable can distinguish "omitted"
// (nil) from "declared but empty" (non-nil, Required == nil): a manifest
// author writing `approvals: {}` still asserts a control this engine does
// not implement, and must fail closed the same as a populated block.
type Environment struct {
	Approvals *ApprovalSpec `yaml:"approvals,omitempty"`
}

// ApprovalSpec is declared, queryable metadata only — see Environment's
// doc comment and design.md D-M.
type ApprovalSpec struct {
	Required []string `yaml:"required,omitempty"`
}

// Policies declares intent this schema parses but does not itself enforce
// (workflow-manifest spec, "Policies Are Declared As Structured,
// Enforceable Schema Fields"). ANY declaration — even an explicit `false`,
// and even one whose guarantee already holds unconditionally, like
// Providers.RequireVersion or Dependencies.ForbidCycles — is rejected by
// ValidateExecutable, an execution-only stage NOT run by Parse/ParseFile.
// Each flag below is a pointer so ValidateExecutable can tell "omitted"
// (nil) apart from "declared" (non-nil, at any value) — see each field's
// comment.
type Policies struct {
	Secrets      SecretsPolicy      `yaml:"secrets,omitempty"`
	Providers    ProvidersPolicy    `yaml:"providers,omitempty"`
	Dependencies DependenciesPolicy `yaml:"dependencies,omitempty"`
	Artifacts    ArtifactsPolicy    `yaml:"artifacts,omitempty"`
}

// SecretsPolicy declares whether a secret reference is forbidden in a
// non-secret-typed field. No layer enforces it; see the Policies comment.
// ForbidPlaintext is a pointer: declaring it `false` still asserts a
// guarantee this engine does not provide, and must fail closed too.
type SecretsPolicy struct {
	ForbidPlaintext *bool `yaml:"forbidPlaintext,omitempty"`
}

// ProvidersPolicy declares provider-version intent. An empty uses.version
// is already unconditionally rejected by ValidateStructure regardless of
// this flag; see the Policies doc comment for why the flag is rejected
// even when declared `false`.
type ProvidersPolicy struct {
	RequireVersion *bool `yaml:"requireVersion,omitempty"`
}

// DependenciesPolicy declares whether cycles are forbidden. Acyclicity is
// already enforced unconditionally at stage 5; see the Policies comment
// for why the flag is rejected even when declared `false`.
type DependenciesPolicy struct {
	ForbidCycles *bool `yaml:"forbidCycles,omitempty"`
}

// ArtifactsPolicy declares whether published artifacts must be immutable.
// No layer enforces it; see the Policies comment. Immutable is a pointer:
// declaring it `false` still asserts a guarantee this engine does not
// provide, and must fail closed too.
type ArtifactsPolicy struct {
	Immutable *bool `yaml:"immutable,omitempty"`
}
