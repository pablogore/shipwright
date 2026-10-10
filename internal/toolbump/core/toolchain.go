// Toolchain descriptor contract (issue #284, design.md D-2). main.go's
// scripts/gobump#280 hardcoded which directories to mutate, which
// RuntimeUpgrader to construct, and which literal-replacement Sites to
// apply, for exactly one toolchain (Go). Toolchain turns those four
// hardcoded decisions into data a generic driver (run.go) can walk without
// knowing which toolchain it is running for — the CLI selects a
// Toolchain by ID, run.go does the rest.
//
// Deliberately NOT a new upgrade interface: Upgrader returns
// pkg/shipwright.RuntimeUpgrader verbatim (design.md D-1). The two
// "gaps" that might look like signature gaps at first — workspace vs
// single-module mode, and more than one mutation site per toolchain —
// are not signature gaps at all: both of Go's calls already use the
// identical RuntimeUpgrader.Upgrade signature and differ only in which
// source Directory is mounted, and N mutation sites is N Target entries,
// which is plan cardinality (data), not a method shape.
package core

import (
	"fmt"
	"regexp"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// idPattern is the format every Toolchain.ID() MUST satisfy: lowercase
// letters and digits only. ID() feeds delivery naming — a future
// bump-toolchain-pr.sh branch name (chore/<id>-<target>) and CLI
// -toolchain flag value — so the same strict, shell-safe format E6
// already enforces on a target version is enforced here on id, before
// that id is ever used to name anything.
var idPattern = regexp.MustCompile(`^[a-z0-9]+$`)

// Target is one mutation site a driver run applies a Toolchain's
// RuntimeUpgrader to: Source is the host-relative directory to mount and
// mutate (for example "." or ".dagger"), Export is the host-relative
// directory the mutated result is written back to (usually equal to
// Source), and Excludes lists the Host().Directory exclude globs that
// keep testdata/.git/coverage/dist content out of what Dagger ever reads
// (design.md D-7 layer 1).
type Target struct {
	Source   string
	Export   string
	Excludes []string
}

// Toolchain describes everything a driver run (run.go) needs to bump one
// toolchain's version across its repo, without the driver knowing which
// toolchain it is.
type Toolchain interface {
	// ID names this toolchain for delivery/CLI selection. It MUST match
	// idPattern; ValidateToolchain rejects any Toolchain whose ID()
	// does not, before a driver run does anything else.
	ID() string
	// Display is the human-readable name used in commit/PR text (for
	// example "Go").
	Display() string
	// RootMarkers lists the repo-relative paths whose presence proves a
	// CLI invocation's cwd is a valid root for this toolchain (for
	// example ["go.work", ".dagger"]). Enforced by the CLI, not by
	// anything in this package.
	RootMarkers() []string
	// Sites returns this toolchain's literal-replacement registry — the
	// plain-string version pins its RuntimeUpgrader cannot reach.
	Sites() []Site
	// Targets returns every mutation site a driver run applies
	// Upgrader's RuntimeUpgrader to, in the order they are upgraded and
	// exported. The count and contents are entirely toolchain-owned
	// data; run.go never hardcodes how many there are.
	Targets() []Target
	// Upgrader constructs the shipwright.RuntimeUpgrader that mutates
	// target's mounted source. Receiving client lets an implementation
	// wrap it in whatever adapter its concrete RuntimeUpgrader needs
	// (for example providers/go's daggerkit.DaggerAdapter).
	Upgrader(client *dagger.Client, target Target) shipwright.RuntimeUpgrader
}

// ValidateToolchain checks t's static contract ahead of any Plan or
// Dagger use: ID() must match idPattern. A driver run calls this first,
// before Plan and before any Dagger connection, so a misconfigured
// Toolchain is rejected with zero side effects — the same "fail before
// touching anything" guarantee E6 already gives a malformed target
// version.
func ValidateToolchain(t Toolchain) error {
	id := t.ID()
	if !idPattern.MatchString(id) {
		return fmt.Errorf("toolbump: toolchain id %q must match %s", id, idPattern.String())
	}
	return nil
}
