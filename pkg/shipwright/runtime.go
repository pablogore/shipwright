package shipwright

// DriftReport is the JSON payload RuntimeInspector.Inspect returns as a
// plain string (design.md D-1: report structs may never appear directly in
// a capability interface signature — only context.Context, error, string,
// and the four Dagger core types may). It is the JSON contract, not a
// method parameter.
//
// Spec requirement "Read-Only Drift Inspection" (runtime-toolchain):
// contains the version(s) discovered at each declarative location, the
// configured target/expected version (if any), and an explicit
// conflict/ambiguity state. A declarative location that is absent from the
// inspected workspace is omitted from Sources entirely — never fabricated
// with a default or assumed value.
type DriftReport struct {
	// WorkspaceRoot is the inspected workspace-relative root ("." by
	// default).
	WorkspaceRoot string `json:"workspaceRoot"`
	// ExpectedVersion echoes the caller-configured expected version, if
	// any ("" when not configured). It is informational only: Inspect
	// does not compare it against the discovered sources itself, leaving
	// that judgment to the report's consumer.
	ExpectedVersion string `json:"expectedVersion,omitempty"`
	// Sources maps each present tier-1 declarative location's fixed name
	// ("go.work", ".go-version") to the version it declares. A location
	// that does not exist in the workspace has no entry here.
	Sources map[string]string `json:"sources"`
	// Modules lists every discovered module's own go.mod version
	// directives. A single-module workspace (no go.work) has exactly one
	// entry with Path ".".
	Modules []ModuleVersion `json:"modules,omitempty"`
	// Conflict is the explicit ambiguity state (spec: "the report marks
	// the conflict state explicitly, naming both sources and versions").
	Conflict ConflictState `json:"conflict"`
}

// ModuleVersion is one workspace module's go.mod version directives, as
// reported by RuntimeInspector.Inspect.
type ModuleVersion struct {
	// Path is the module's directory relative to the workspace root ("."
	// for a single go.mod at the workspace root itself).
	Path string `json:"path"`
	// Go is the module's go directive value ("" if absent).
	Go string `json:"go,omitempty"`
	// Toolchain is the module's toolchain directive value ("" if absent).
	Toolchain string `json:"toolchain,omitempty"`
}

// ConflictState is DriftReport's explicit ambiguity marker. Ambiguous is
// false and Code/Sites are empty whenever every present tier-1 source
// agrees; no single "winning" version is ever inferred when they do not
// (spec: "Ambiguous sources are reported, never guessed").
type ConflictState struct {
	// Ambiguous is true when two or more declarative locations disagree
	// with no resolvable precedence.
	Ambiguous bool `json:"ambiguous"`
	// Code is the ambiguity rule that fired (design.md D-5: A1-A6), empty
	// when Ambiguous is false.
	Code string `json:"code,omitempty"`
	// Sites names every conflicting source and the version it declares,
	// empty when Ambiguous is false.
	Sites []string `json:"sites,omitempty"`
}
