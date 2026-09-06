// SECURITY tests — tasks.md 4.4 (oversized manifest cap) and 4.5
// (alias-amplification / "billion laughs" bounded budget), plus the
// secrets-by-name-only rule that stage 1 decode enforces structurally
// (design.md D-H threat matrix items 1 and 2, workflow-manifest spec
// "Secrets Referenced By Name, Never Embedded As Plaintext").
package manifest_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pablogore/shipwright/internal/workflow/manifest"
)

// TestParse_OversizedManifestRejected proves the io.LimitReader cap
// (design.md D-H threat matrix item 1) rejects a manifest over
// MaxManifestBytes with a clear, specific error — never a silent
// truncation and never a panic.
func TestParse_OversizedManifestRejected(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), manifest.MaxManifestBytes+1)

	_, err := manifest.Parse(bytes.NewReader(oversized))
	if err == nil {
		t.Fatal("Parse() with an oversized manifest must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("Parse() error = %v, want an exceeds-maximum-size error", err)
	}
}

// TestParse_BillionLaughsFixtureBounded proves an alias-amplification
// ("billion laughs") document completes within a bounded time budget
// rather than hanging or exhausting memory (tasks.md 4.5). Any outcome —
// success or a specific decode error — is acceptable; what this test
// guards is boundedness itself, not a particular error message. This
// deliberately does not depend solely on yaml.v3's internal alias-ratio
// protection: MaxManifestBytes already bounds how much raw source (and
// therefore how many anchors) this fixture can contain, and the typed
// schema bounds where an unmarshaled alias tree can actually expand to —
// design.md's explicit instruction not to rely on the library's internal
// limit ALONE.
func TestParse_BillionLaughsFixtureBounded(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "billion-laughs.yaml"))
	if err != nil {
		t.Fatalf("failed to read billion-laughs fixture: %v", err)
	}

	if len(data) >= manifest.MaxManifestBytes {
		t.Fatalf("billion-laughs fixture is %d bytes, want well under MaxManifestBytes (%d) — the attack's premise is a small source", len(data), manifest.MaxManifestBytes)
	}

	done := make(chan error, 1)
	go func() {
		_, parseErr := manifest.Parse(bytes.NewReader(data))
		done <- parseErr
	}()

	const budget = 5 * time.Second
	select {
	case parseErr := <-done:
		t.Logf("billion-laughs fixture returned within the %s budget, err = %v", budget, parseErr)
	case <-time.After(budget):
		t.Fatalf("Parse() did not complete within the %s bounded budget for the billion-laughs fixture — possible alias-amplification DoS", budget)
	}
}

// TestParse_InlineSecretPlaintextValueRejected proves
// spec.secrets.<name>.value (a literal plaintext value, not a reference)
// is rejected — the SecretSpec struct declares no `value` field, so
// yaml.Decoder.KnownFields(true) rejects it at stage 1 decode, before any
// later stage runs (workflow-manifest spec, "Inline plaintext secret
// value rejected").
func TestParse_InlineSecretPlaintextValueRejected(t *testing.T) {
	src := identityHeader + `
  secrets:
    registry-token:
      value: "s3cr3t"
`
	_, err := manifest.Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse() with an inline plaintext spec.secrets.<name>.value must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "value") {
		t.Fatalf("Parse() error = %v, want an error naming the unknown value field", err)
	}
}

// TestParse_NamedSecretReferenceValidates proves the accepted shape —
// spec.secrets.<name>: {fromEnv: ENV_VAR} — parses and holds no plaintext
// (workflow-manifest spec, "Named secret reference without plaintext
// validates").
func TestParse_NamedSecretReferenceValidates(t *testing.T) {
	src := identityHeader + `
  secrets:
    registry-token: {fromEnv: REGISTRY_TOKEN}
`
	m, err := manifest.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if m.Spec.Secrets["registry-token"].FromEnv != "REGISTRY_TOKEN" {
		t.Fatalf("Secrets[registry-token].FromEnv = %q, want REGISTRY_TOKEN", m.Spec.Secrets["registry-token"].FromEnv)
	}
}
