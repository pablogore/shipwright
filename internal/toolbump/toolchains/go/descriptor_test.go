package gotoolchain

import (
	"testing"

	"github.com/pablogore/shipwright/internal/toolbump/core"
)

// TestGoToolchainIdentity is the RED test for tasks.md 2.1's ID()/
// Display() pair: "go" already matches core.ValidateToolchain's required
// lowercase-alphanumeric format, and "Go" is the human-readable name
// #280's commit/PR text would need if it were driven through this
// descriptor.
func TestGoToolchainIdentity(t *testing.T) {
	if got, want := Toolchain.ID(), "go"; got != want {
		t.Errorf("ID() = %q, want %q", got, want)
	}
	if got, want := Toolchain.Display(), "Go"; got != want {
		t.Errorf("Display() = %q, want %q", got, want)
	}
}

// TestGoToolchainSitesWellFormed proves every entry in the relocated
// registry compiles cleanly (the standing guard scripts/gobump/sites's
// own TestSitesWellFormed gave the pre-move production Sites var — see
// this package's own doc comment for why that test could not simply move
// here in Phase 1).
func TestGoToolchainSitesWellFormed(t *testing.T) {
	sites := Toolchain.Sites()
	if len(sites) == 0 {
		t.Fatal("Sites() must not be empty")
	}
	for _, s := range sites {
		if _, err := core.Compile(s); err != nil {
			t.Errorf("Sites() entry %q does not compile: %v", s.Path, err)
		}
		if s.Occurrences < 1 {
			t.Errorf("Sites() entry %q has Occurrences=%d, want >= 1", s.Path, s.Occurrences)
		}
	}
}
