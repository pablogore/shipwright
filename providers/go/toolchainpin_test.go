package golang

import (
	"os"
	"testing"

	"golang.org/x/mod/modfile"
)

// TestDefaultGoVersionMatchesGoMod guards design.md D-4's tier-3
// substitution: providers/go/gobuilder.go's defaultGoVersion is Shipwright's
// own builder-image default, structurally unreachable from runtime-inspect
// (it is a plain compile-time constant in package golang, not a property of
// an inspected workspace) — so instead of reporting it as workspace drift,
// this test enforces the invariant directly: defaultGoVersion must always
// equal this module's own go.mod go directive.
//
// This is expected to be RED at the moment this test is added: go.mod
// currently pins go 1.26.7, while defaultGoVersion is still 1.25.5.
func TestDefaultGoVersionMatchesGoMod(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}

	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		t.Fatalf("failed to parse go.mod: %v", err)
	}

	if f.Go == nil {
		t.Fatal("go.mod has no go directive")
	}

	if f.Go.Version != defaultGoVersion {
		t.Fatalf("defaultGoVersion = %q, want %q (providers/go/go.mod's go directive) — "+
			"gobuilder.go:44 must be bumped to match", defaultGoVersion, f.Go.Version)
	}
}
