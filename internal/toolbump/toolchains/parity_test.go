package toolchains

import (
	"reflect"
	"testing"

	"github.com/pablogore/shipwright/internal/toolbump/core"
	golang "github.com/pablogore/shipwright/providers/go"
)

// legacyHostExcludes duplicates scripts/gobump/main.go#280's own
// hostExcludes var verbatim (main.go's CLI wiring is untouched this
// phase — hard constraint). It is the expected value every assertion
// below compares against, so a future edit to main.go's real list (or a
// silent drift in the descriptor's copy) goes red here.
var legacyHostExcludes = []string{"**/testdata/**", ".git", ".git/**", "coverage/**", "dist/**"}

// TestRegistryGoTargetsMatchLegacyMainGo is the RED test for tasks.md
// 2.3's parity requirement, half one of three: Registry["go"]'s
// Targets() must resolve to exactly #280's two hardcoded
// Host().Directory calls, in the same order main.go/run issued them —
// workspace "." first, then ".dagger" — with the identical exclude list.
// A future CLI (Phase 3) that looks up Registry["go"] and drives it
// through core.Run mounts, exports, and excludes exactly what #280's
// main.go always did.
func TestRegistryGoTargetsMatchLegacyMainGo(t *testing.T) {
	tc, ok := Registry["go"]
	if !ok {
		t.Fatal(`Registry["go"] not registered`)
	}

	want := []core.Target{
		{Source: ".", Export: ".", Excludes: legacyHostExcludes},
		{Source: ".dagger", Export: ".dagger", Excludes: legacyHostExcludes},
	}
	if got := tc.Targets(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Registry[\"go\"].Targets() = %#v, want %#v (must match scripts/gobump/main.go#280's two hardcoded Upgrade calls)", got, want)
	}
}

// TestRegistryGoUpgraderMatchesLegacyConfig is parity half two: both of
// #280's hardcoded calls constructed an identical *golang.GoRuntimeUpgrader
// (WorkspaceRoot ".", Tidy false, AllowDowngrade false) regardless of
// which of the two directories it mutated. Registry["go"]'s Upgrader
// must reproduce that same configuration for every Target it returns —
// the "zero output diff" spec scenario, proven at construction time
// since scripts/toolbump's CLI does not exist yet (Phase 3) to invoke
// end-to-end.
func TestRegistryGoUpgraderMatchesLegacyConfig(t *testing.T) {
	tc := Registry["go"]

	for _, target := range tc.Targets() {
		up := tc.Upgrader(nil, target)
		gu, ok := up.(*golang.GoRuntimeUpgrader)
		if !ok {
			t.Fatalf("Upgrader(%s) = %T, want *golang.GoRuntimeUpgrader", target.Source, up)
		}
		if gu.WorkspaceRoot != "." || gu.Tidy != false || gu.AllowDowngrade != false {
			t.Fatalf("Upgrader(%s) config = %+v, want WorkspaceRoot=\".\" Tidy=false AllowDowngrade=false (scripts/gobump/main.go#280 parity)", target.Source, gu)
		}
	}
}

// TestRegistryGoRootMarkersMatchLegacyRequireRepoRoot is parity half
// three: #280's requireRepoRoot guard checked exactly go.work and
// .dagger, in that order, before doing anything else. Registry["go"]'s
// RootMarkers must name the same two paths so a future CLI's root guard
// (Phase 3) rejects the identical set of wrong-cwd invocations #280 did.
func TestRegistryGoRootMarkersMatchLegacyRequireRepoRoot(t *testing.T) {
	tc := Registry["go"]

	want := []string{"go.work", ".dagger"}
	if got := tc.RootMarkers(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Registry[\"go\"].RootMarkers() = %v, want %v", got, want)
	}
}
