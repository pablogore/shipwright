package core

import (
	"context"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// fakeToolchain is a minimal, non-production Toolchain test double used
// only within this package's own tests to exercise ValidateToolchain and
// Run's ordering/abort guarantees (tasks.md 1.5/1.7) without any real
// toolchain data or a live Dagger engine. It is not a second/fixture
// Toolchain descriptor — design.md's "Paper Walkthrough" for a
// hypothetical toolchain #2 is explicitly out of cycle-2 scope, and this
// type never leaves this package's _test.go files.
type fakeToolchain struct {
	id      string
	sites   []Site
	targets []Target
	upgrade func(client *dagger.Client, target Target) shipwright.RuntimeUpgrader

	sitesCalled, targetsCalled, upgraderCalled bool
}

func (f *fakeToolchain) ID() string            { return f.id }
func (f *fakeToolchain) Display() string       { return "Fake" }
func (f *fakeToolchain) RootMarkers() []string { return nil }

func (f *fakeToolchain) Sites() []Site {
	f.sitesCalled = true
	return f.sites
}

func (f *fakeToolchain) Targets() []Target {
	f.targetsCalled = true
	return f.targets
}

func (f *fakeToolchain) Upgrader(client *dagger.Client, target Target) shipwright.RuntimeUpgrader {
	f.upgraderCalled = true
	return f.upgrade(client, target)
}

func TestValidateToolchainRejectsInvalidID(t *testing.T) {
	cases := []string{"Go", "go-lang", "go_lang", "go lang", "", "GO", "1.2"}

	for _, id := range cases {
		t.Run(id, func(t *testing.T) {
			tc := &fakeToolchain{id: id}
			if err := ValidateToolchain(tc); err == nil {
				t.Fatalf("expected toolchain id %q to be rejected", id)
			}
		})
	}
}

func TestValidateToolchainAcceptsValidID(t *testing.T) {
	for _, id := range []string{"go", "go2", "rust"} {
		tc := &fakeToolchain{id: id}
		if err := ValidateToolchain(tc); err != nil {
			t.Errorf("expected toolchain id %q to be accepted, got: %v", id, err)
		}
	}
}

// TestRunRejectsInvalidIDBeforeAnyRunStarts is the RED test for spec's
// "Invalid id rejected" scenario (capability toolchain-bump-plugin):
// registration/rejection happens before Run touches Sites, Targets, or
// Upgrader at all — proven here by asserting none of the three were ever
// called once Run rejects the id.
func TestRunRejectsInvalidIDBeforeAnyRunStarts(t *testing.T) {
	tc := &fakeToolchain{id: "Invalid-ID!"}

	err := Run(context.Background(), tc, t.TempDir(), "1.27.0")
	if err == nil {
		t.Fatal("expected an error for an invalid toolchain id")
	}
	if tc.sitesCalled || tc.targetsCalled || tc.upgraderCalled {
		t.Fatalf("Run touched the toolchain descriptor before rejecting its id: sitesCalled=%v targetsCalled=%v upgraderCalled=%v",
			tc.sitesCalled, tc.targetsCalled, tc.upgraderCalled)
	}
}
