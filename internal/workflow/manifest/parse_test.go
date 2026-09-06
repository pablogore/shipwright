// Stage 1 (size-capped read + decode) tests — tasks.md 4.6, plus
// end-to-end ParseFile and the full design.md Interfaces/Contracts example
// manifest, proving the typed schema round-trips the documented shape.
package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/internal/workflow/manifest"
)

func TestParse_MalformedYAMLFailsWithSpecificError(t *testing.T) {
	src := "apiVersion: [this is not: closed properly\n"

	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with malformed YAML must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "manifest: decode:") {
		t.Fatalf("Parse() error = %v, want a manifest: decode: prefixed error", err)
	}
}

// TestParse_UnknownFieldRejected proves yaml.Decoder.KnownFields(true) is
// wired in: a field absent from the typed schema fails decode rather than
// being silently ignored (design.md D-H, tasks.md 4.6).
func TestParse_UnknownFieldRejected(t *testing.T) {
	src := identityHeader + `
  bogusField: true
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with an unknown top-level spec field must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "bogusField") {
		t.Fatalf("Parse() error = %v, want an error naming the unknown field bogusField", err)
	}
}

func TestParseFile_ReadsAndValidatesFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workflow.yaml")
	if err := os.WriteFile(path, []byte(identityHeader), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v, want nil", path, err)
	}

	m, err := manifest.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%q) error = %v, want nil", path, err)
	}
	if m.Metadata.Name != "example" {
		t.Fatalf("ParseFile(%q).Metadata.Name = %q, want %q", path, m.Metadata.Name, "example")
	}
}

func TestParseFile_MissingFileFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")

	_, err := manifest.ParseFile(path)
	if err == nil {
		t.Fatal("ParseFile() with a missing file must return an error, got nil")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("ParseFile() error = %v, want an error naming the missing path %s", err, path)
	}
}

// TestParse_DesignExampleManifestValidates round-trips design.md's own
// Interfaces/Contracts example (the go-service-release workflow, diamond
// fan-in included) end to end through stages 1-3, proving the typed
// schema actually represents the documented shape rather than a
// paraphrase of it.
func TestParse_DesignExampleManifestValidates(t *testing.T) {
	src := `
apiVersion: shipwright.dev/v1
kind: Workflow
metadata:
  name: go-service-release
spec:
  source:
    path: .
  variables:
    imageRef: ghcr.io/acme/api
  secrets:
    registry: {fromEnv: REGISTRY_PASSWORD}
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: "1"}
      with: {goVersion: "1.26.1"}
    - id: unit
      capability: test
      uses: {provider: go-test, version: "1"}
      needs: [build]
      input: ${{ steps.build.output }}
    - id: vuln
      capability: test
      uses: {provider: govulncheck, version: "1"}
      needs: [build]
      input: ${{ steps.build.output }}
    - id: publish
      capability: artifact
      uses: {provider: container, version: "1"}
      needs: [unit, vuln]
      input: ${{ steps.build.output }}
      with:
        ref: ${{ variables.imageRef }}
        creds: ${{ secrets.registry }}
      when: {branch: [main]}
  execution:
    concurrency: {maxParallel: 4}
    failFast: true
    timeout: 30m
  environments:
    production:
      approvals: {required: [platform-team]}
  policies:
    dependencies: {forbidCycles: true}
    providers: {requireVersion: true}
    secrets: {forbidPlaintext: true}
`
	m, err := manifest.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if len(m.Spec.Steps) != 4 {
		t.Fatalf("len(Spec.Steps) = %d, want 4", len(m.Spec.Steps))
	}

	publish := m.Spec.Steps[3]
	if publish.ID != "publish" {
		t.Fatalf("Spec.Steps[3].ID = %q, want publish", publish.ID)
	}
	if len(publish.Needs) != 2 || publish.Needs[0] != "unit" || publish.Needs[1] != "vuln" {
		t.Fatalf("publish.Needs = %v, want [unit vuln] — the diamond fan-in", publish.Needs)
	}
	if branch := publish.When["branch"]; len(branch) != 1 || branch[0] != "main" {
		t.Fatalf(`publish.When["branch"] = %v, want [main]`, branch)
	}
	if m.Spec.Execution.Concurrency.MaxParallel != 4 {
		t.Fatalf("Spec.Execution.Concurrency.MaxParallel = %d, want 4", m.Spec.Execution.Concurrency.MaxParallel)
	}
}
