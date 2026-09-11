package toolchains

import (
	"testing"

	gotoolchain "github.com/pablogore/shipwright/internal/toolbump/toolchains/go"
)

// TestRegistryContainsGo is the RED test for tasks.md 2.2: the future CLI
// (Phase 3) selects a core.Toolchain by looking up its -toolchain flag
// value in this map, so "go" must resolve to exactly the singleton
// toolchains/go exports — never a distinct copy.
func TestRegistryContainsGo(t *testing.T) {
	tc, ok := Registry["go"]
	if !ok {
		t.Fatal(`Registry["go"] not registered`)
	}
	if tc != gotoolchain.Toolchain {
		t.Fatalf(`Registry["go"] = %#v, want the toolchains/go.Toolchain singleton %#v`, tc, gotoolchain.Toolchain)
	}
}

// TestRegistryKeyedByOwnID proves the registry's map key always agrees
// with the registered Toolchain's own ID() (design.md's "Registry
// extensibility" requirement) — a future toolchain #2 registered under
// the wrong key would silently break -toolchain=<id> lookups without
// this guard ever going red.
func TestRegistryKeyedByOwnID(t *testing.T) {
	for key, tc := range Registry {
		if tc.ID() != key {
			t.Errorf("Registry[%q].ID() = %q, want %q (map key must match the toolchain's own ID)", key, tc.ID(), key)
		}
	}
}
