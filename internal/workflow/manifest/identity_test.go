// Stage 2 (document identity) validation — tasks.md 4.1, workflow-manifest
// spec "Versioned Document Identity".
package manifest_test

import (
	"strings"
	"testing"

	"github.com/pablogore/shipwright/internal/workflow/manifest"
)

func TestParse_MissingAPIVersionFailsValidation(t *testing.T) {
	src := `
kind: Workflow
metadata:
  name: example
spec:
  source: {path: .}
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with missing apiVersion must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "apiVersion") {
		t.Fatalf("Parse() error = %v, want an error naming the missing apiVersion field", err)
	}
}

func TestParse_MissingKindFailsValidation(t *testing.T) {
	src := `
apiVersion: shipwright.dev/v1
metadata:
  name: example
spec:
  source: {path: .}
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with missing kind must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "kind") {
		t.Fatalf("Parse() error = %v, want an error naming the missing kind field", err)
	}
}

func TestParse_MissingMetadataNameFailsValidation(t *testing.T) {
	src := `
apiVersion: shipwright.dev/v1
kind: Workflow
metadata: {}
spec:
  source: {path: .}
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with missing metadata.name must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "metadata.name") {
		t.Fatalf("Parse() error = %v, want an error naming the missing metadata.name field", err)
	}
}

func TestParse_UnsupportedAPIVersionFailsValidation(t *testing.T) {
	src := `
apiVersion: shipwright.dev/v2
kind: Workflow
metadata:
  name: example
spec:
  source: {path: .}
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with an unsupported apiVersion must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported apiVersion") {
		t.Fatalf("Parse() error = %v, want an unsupported apiVersion error", err)
	}
}

func TestParse_UnsupportedKindFailsValidation(t *testing.T) {
	src := `
apiVersion: shipwright.dev/v1
kind: Pipeline
metadata:
  name: example
spec:
  source: {path: .}
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with an unsupported kind must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported kind") {
		t.Fatalf("Parse() error = %v, want an unsupported kind error", err)
	}
}

func TestParse_WellFormedIdentityValidates(t *testing.T) {
	src := `
apiVersion: shipwright.dev/v1
kind: Workflow
metadata:
  name: example
spec:
  source: {path: .}
`
	m, err := manifest.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if m.Metadata.Name != "example" {
		t.Fatalf("Metadata.Name = %q, want %q", m.Metadata.Name, "example")
	}
}
