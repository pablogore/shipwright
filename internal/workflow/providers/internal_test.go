package providers

import (
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/workflow/interp"
)

// Unit tests for unexported pure helpers (register.go's stringField and
// floatField) and the exported error types' Error() formatting. In-package
// (not providers_test) because stringField/floatField are deliberately
// unexported convenience helpers used only while building a WU3
// capability's Config from a manifest's `with` Values — following the
// same pattern as internal/capabilities/internal_test.go.
//
// Note on branch reachability: through the PUBLIC Registry.Resolve* path,
// checkWithSchema (registry.go) already rejects any with-field whose Kind
// does not match its schema-declared Kind before a factory ever runs, so
// stringField/floatField's "value present but wrong kind" defensive
// branches are unreachable via that path — they are tested directly here
// instead, both because they are a genuine defense-in-depth guard (a
// factory could be registered with a schema that does NOT declare a field
// it reads, deliberately or by mistake) and to keep this package's
// coverage honest rather than leaving a false sense of unreachable-code
// safety.

func TestStringField(t *testing.T) {
	tests := []struct {
		name string
		v    Values
		key  string
		want string
	}{
		{name: "key absent returns empty", v: Values{}, key: "goVersion", want: ""},
		{name: "string value returned as-is", v: Values{"goVersion": interp.NewString("1.26.1")}, key: "goVersion", want: "1.26.1"},
		{name: "secret value (no string form) returns empty", v: Values{"goVersion": interp.NewSecret(&dagger.Secret{})}, key: "goVersion", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringField(tt.v, tt.key); got != tt.want {
				t.Fatalf("stringField(%v, %q) = %q, want %q", tt.v, tt.key, got, tt.want)
			}
		})
	}
}

func TestFloatField(t *testing.T) {
	tests := []struct {
		name string
		v    Values
		key  string
		want float64
	}{
		{name: "key absent returns zero", v: Values{}, key: "coverage", want: 0},
		{name: "parseable numeric string returned", v: Values{"coverage": interp.NewInt(90)}, key: "coverage", want: 90},
		{name: "secret value (no string form) returns zero", v: Values{"coverage": interp.NewSecret(&dagger.Secret{})}, key: "coverage", want: 0},
		{name: "unparseable string returns zero", v: Values{"coverage": interp.NewString("not-a-number")}, key: "coverage", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := floatField(tt.v, tt.key); got != tt.want {
				t.Fatalf("floatField(%v, %q) = %v, want %v", tt.v, tt.key, got, tt.want)
			}
		})
	}
}

func TestUnregisteredProviderError_Error(t *testing.T) {
	inRepo := &UnregisteredProviderError{Capability: "build", Ref: Ref{Name: "gradle", Version: "2"}}
	if got, want := inRepo.Error(), `providers: unregistered provider "gradle" for capability "build" (version "2")`; got != want {
		t.Fatalf("UnregisteredProviderError.Error() = %q, want %q", got, want)
	}

	module := &UnregisteredProviderError{Capability: "build", Ref: Ref{Name: "custom", Module: "github.com/acme/custom-builder", Version: "1"}}
	got := module.Error()
	if !containsModulePath(got, "github.com/acme/custom-builder") {
		t.Fatalf("UnregisteredProviderError.Error() = %q, must name the module path", got)
	}
}

func TestUnsupportedVersionError_Error(t *testing.T) {
	err := &UnsupportedVersionError{
		Capability:        "build",
		Name:              "go",
		RequestedVersion:  "2",
		SupportedVersions: []string{"1"},
	}
	want := `providers: "go" does not support capability "build" version "2" (supported: [1])`
	if got := err.Error(); got != want {
		t.Fatalf("UnsupportedVersionError.Error() = %q, want %q", got, want)
	}

	moduleErr := &UnsupportedVersionError{
		Capability:        "build",
		Module:            "github.com/acme/custom-builder",
		RequestedVersion:  "3",
		SupportedVersions: []string{"1", "2"},
	}
	if got := moduleErr.Error(); !containsModulePath(got, "github.com/acme/custom-builder") {
		t.Fatalf("UnsupportedVersionError.Error() = %q, must name the module path when Module is set", got)
	}
}

func TestWithSchemaMismatchError_Error(t *testing.T) {
	err := &WithSchemaMismatchError{Capability: "artifact", Field: "ref", Want: interp.KindString, Got: interp.KindSecret}
	want := `providers: capability "artifact" with field "ref" has kind secret, provider requires string`
	if got := err.Error(); got != want {
		t.Fatalf("WithSchemaMismatchError.Error() = %q, want %q", got, want)
	}
}

// containsModulePath mirrors register_test.go's helper of the same name —
// duplicated deliberately: this file is package providers (in-package),
// register_test.go is package providers_test (black-box), and a test
// helper cannot cross that package boundary.
func containsModulePath(msg, path string) bool {
	for i := 0; i+len(path) <= len(msg); i++ {
		if msg[i:i+len(path)] == path {
			return true
		}
	}
	return false
}
