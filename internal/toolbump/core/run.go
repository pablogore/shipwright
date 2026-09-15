// Plan-then-mutate driver (issue #284, design.md D-2/D-6). Run
// generalizes scripts/gobump#280's main.go/run() sequence — Plan every
// literal-replacement site, connect to Dagger only once that succeeds,
// upgrade and export every mutation Target, then apply the already-
// computed edits last — over an arbitrary Toolchain instead of two
// hardcoded Go invocations.
package core

import (
	"context"
	"fmt"

	"dagger.io/dagger"
)

// connectFunc, newHostDirectory, exportDirectory, and closeClient are
// unexported, swappable function variables wrapping the only four points
// where Run touches the real Dagger SDK. Run's real callers never
// override them — they exist so a test can substitute a Dagger-free
// fake and verify Run's ordering and abort-on-Plan-failure guarantees
// (tasks.md 1.7) without a live Dagger engine, matching the "pure
// plan/driver unit tests" this phase's tests are scoped to.
var (
	connectFunc = func(ctx context.Context) (*dagger.Client, error) {
		return dagger.Connect(ctx)
	}
	newHostDirectory = func(client *dagger.Client, t Target) *dagger.Directory {
		return client.Host().Directory(t.Source, dagger.HostDirectoryOpts{Exclude: t.Excludes})
	}
	exportDirectory = func(ctx context.Context, dir *dagger.Directory, t Target) error {
		_, err := dir.Export(ctx, t.Export)
		return err
	}
	closeClient = func(client *dagger.Client) error {
		return client.Close()
	}
)

// Run performs a full toolchain version bump for t: validate t's static
// contract, compute t.Sites()'s plan, connect to Dagger only if that
// plan succeeded, run every Target's Upgrade, export every result back
// to the host, and finally Apply the edits the plan already computed.
//
// Ordering is fixed and toolchain-agnostic (design.md's "Generic
// multi-target driver" requirement): Plan -> all Upgrades -> all Exports
// -> Apply, over t.Targets(), with no hardcoded count and no
// toolchain-specific branching. A Plan failure returns before Dagger is
// ever connected and before any host file is mutated; an Upgrade or
// Export failure for any one Target aborts before Apply runs, so
// Apply — the only step that can mutate a literal-replacement site — is
// never reached unless every Target already succeeded.
func Run(ctx context.Context, t Toolchain, cwd, target string) (err error) {
	if err := ValidateToolchain(t); err != nil {
		return err
	}

	edits, err := Plan(cwd, t.Sites(), target)
	if err != nil {
		return fmt.Errorf("toolbump: %s: site plan failed, no site was mutated: %w", t.ID(), err)
	}

	client, err := connectFunc(ctx)
	if err != nil {
		return fmt.Errorf("toolbump: %s: failed to connect to Dagger: %w", t.ID(), err)
	}
	defer func() {
		if closeErr := closeClient(client); closeErr != nil && err == nil {
			err = fmt.Errorf("toolbump: %s: failed to close Dagger client: %w", t.ID(), closeErr)
		}
	}()

	targets := t.Targets()
	results := make([]*dagger.Directory, len(targets))
	for i, tgt := range targets {
		src := newHostDirectory(client, tgt)
		result, upErr := t.Upgrader(client, tgt).Upgrade(ctx, src, target)
		if upErr != nil {
			return fmt.Errorf("toolbump: %s: upgrade of %s failed: %w", t.ID(), tgt.Source, upErr)
		}
		results[i] = result
	}

	// Every Upgrade already succeeded; only now does Run start exporting
	// results back to the host (design.md's Plan -> all Upgrades -> all
	// Exports -> Apply ordering) — never Wipe, mirroring #280's export
	// semantics: anything a Target's Excludes kept out of Dagger is left
	// untouched on disk, not deleted.
	for i, tgt := range targets {
		if expErr := exportDirectory(ctx, results[i], tgt); expErr != nil {
			return fmt.Errorf("toolbump: %s: export of %s failed: %w", t.ID(), tgt.Export, expErr)
		}
	}

	// Sites last, only after every Target is already known-good on
	// disk — Apply's own contract (sites.go) is to run only with edits
	// from a Plan that returned no error for the same site set, which
	// is exactly what the first step above already guaranteed.
	return Apply(cwd, edits)
}
