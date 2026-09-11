package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// fakeUpgrader is a shipwright.RuntimeUpgrader test double that records
// its call instead of touching Dagger, letting these tests verify Run's
// Upgrade/Export/Apply ordering (tasks.md 1.7) without a live Dagger
// engine.
type fakeUpgrader struct {
	name string
	log  *[]string
}

func (u *fakeUpgrader) Upgrade(_ context.Context, source *dagger.Directory, _ string) (*dagger.Directory, error) {
	*u.log = append(*u.log, "upgrade:"+u.name)
	return source, nil
}

// withFakeDaggerSeams swaps Run's Dagger-touching function variables
// (run.go) for no-op fakes that record into log instead of calling the
// real Dagger SDK, restoring the real ones on test cleanup. Run's real
// callers (a future CLI, Phase 3) never override these — see run.go's own
// doc comment on the var block for why they exist.
func withFakeDaggerSeams(t *testing.T, log *[]string) {
	t.Helper()

	origConnect, origHost, origExport, origClose := connectFunc, newHostDirectory, exportDirectory, closeClient

	connectFunc = func(context.Context) (*dagger.Client, error) {
		*log = append(*log, "connect")
		return &dagger.Client{}, nil
	}
	newHostDirectory = func(*dagger.Client, Target) *dagger.Directory {
		return nil
	}
	exportDirectory = func(_ context.Context, _ *dagger.Directory, tgt Target) error {
		*log = append(*log, "export:"+tgt.Export)
		return nil
	}
	closeClient = func(*dagger.Client) error {
		*log = append(*log, "close")
		return nil
	}

	t.Cleanup(func() {
		connectFunc, newHostDirectory, exportDirectory, closeClient = origConnect, origHost, origExport, origClose
	})
}

// TestRunOrdersUpgradeThenExportPerTargetBeforeApply is the RED test for
// spec's "N-target orchestration" scenario (capability toolchain-bump-
// core): a 2-target fake descriptor runs Upgrade once per target (in
// order), then Export once per target (in order), and only then Apply —
// matching design.md's fixed Plan -> all Upgrades -> all Exports -> Apply
// ordering, with target count driven entirely by t.Targets(), never
// hardcoded in Run.
func TestRunOrdersUpgradeThenExportPerTargetBeforeApply(t *testing.T) {
	root := t.TempDir()
	const sitePath = "VERSION"
	if err := os.WriteFile(filepath.Join(root, sitePath), []byte(`v = "1.26.1"`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	var log []string
	withFakeDaggerSeams(t, &log)

	tc := &fakeToolchain{
		id:    "fake",
		sites: []Site{{Path: sitePath, Anchor: `v = "{{V}}"`, Occurrences: 1}},
		targets: []Target{
			{Source: "workspace", Export: "workspace"},
			{Source: "module", Export: "module"},
		},
		upgrade: func(_ *dagger.Client, target Target) shipwright.RuntimeUpgrader {
			return &fakeUpgrader{name: target.Source, log: &log}
		},
	}

	if err := Run(context.Background(), tc, root, "1.27.0"); err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	want := []string{"connect", "upgrade:workspace", "upgrade:module", "export:workspace", "export:module", "close"}
	if len(log) != len(want) {
		t.Fatalf("call order mismatch:\n got:  %v\n want: %v", log, want)
	}
	for i := range want {
		if log[i] != want[i] {
			t.Fatalf("call order mismatch at index %d:\n got:  %v\n want: %v", i, log, want)
		}
	}

	got, err := os.ReadFile(filepath.Join(root, sitePath))
	if err != nil {
		t.Fatalf("reading %s: %v", sitePath, err)
	}
	if string(got) != `v = "1.27.0"` {
		t.Fatalf("Apply did not run last (or at all): %q", got)
	}
}

// TestRunAbortsBeforeDaggerConnectOnPlanFailure is the RED test for
// spec's "No mutation on plan failure" scenario (capability
// toolchain-bump-core): when Plan fails for a site, Run returns before
// connecting to Dagger (connectFunc is never invoked) and before any
// host file is mutated (Upgrader is never constructed).
func TestRunAbortsBeforeDaggerConnectOnPlanFailure(t *testing.T) {
	root := t.TempDir()

	var log []string
	withFakeDaggerSeams(t, &log)

	tc := &fakeToolchain{
		id:      "fake",
		sites:   []Site{{Path: "does/not/exist", Anchor: `v = "{{V}}"`, Occurrences: 1}}, // E1: missing file
		targets: []Target{{Source: "workspace", Export: "workspace"}},
		upgrade: func(*dagger.Client, Target) shipwright.RuntimeUpgrader {
			t.Fatal("Upgrader must not be constructed when Plan already failed")
			return nil
		},
	}

	err := Run(context.Background(), tc, root, "1.27.0")
	if err == nil {
		t.Fatal("expected an error when Plan fails, got nil")
	}
	if len(log) != 0 {
		t.Fatalf("expected zero Dagger interaction after a Plan failure, got: %v", log)
	}
	if tc.targetsCalled || tc.upgraderCalled {
		t.Fatalf("Run touched Targets/Upgrader despite a Plan failure: targetsCalled=%v upgraderCalled=%v", tc.targetsCalled, tc.upgraderCalled)
	}
}
