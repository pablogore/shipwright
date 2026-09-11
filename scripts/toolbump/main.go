// Command toolbump is the toolchain-agnostic version bump CLI (issue #284,
// design.md D-1/D-2/D-3), successor to scripts/gobump#280's Go-only main.go.
// It selects a Toolchain from internal/toolbump/toolchains.Registry by
// -toolchain (default "go") and delegates the entire plan-then-mutate
// sequence to internal/toolbump/core.Run, which owns exactly the ordering
// scripts/gobump#280's main.go/run() hardcoded for Go alone: Plan -> all
// Upgrades -> all Exports -> Apply, over the selected Toolchain's Targets().
//
// This package makes no filesystem or Dagger-client decision core.Run
// doesn't already own: requireRootMarkers is the only pure, independently
// unit-tested logic here (Threat Matrix: git repository selection), now
// driven by the selected Toolchain's own RootMarkers() data instead of a
// hardcoded go.work/.dagger pair.
//
// -print-delivery additionally emits a newline key=value manifest
// (branch=/commit=/pr_title=) for scripts/bump-toolchain-pr.sh to parse via
// `read`, never `eval` (design.md D-4) -- deriving delivery naming from
// ID()/Display() instead of a literal "go" (design.md D-3). Because this
// manifest becomes shell-adjacent input, -print-delivery re-validates both
// -toolchain and -target against the same strict formats
// core.ValidateToolchain and core.Plan already enforce deeper in a real
// run (design.md's "New hard guard"): a malformed value is rejected here,
// with nothing printed to stdout, before Registry is even consulted.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pablogore/shipwright/internal/toolbump/core"
	"github.com/pablogore/shipwright/internal/toolbump/toolchains"
)

// idPattern and targetPattern duplicate internal/toolbump/core's own
// unexported checks (idPattern in toolchain.go, targetPattern in sites.go).
// core.Run and core.Plan already enforce both deeper in a real run; this
// package's own copies exist solely so -print-delivery -- a manifest a
// shell script parses and uses as literal commit/branch/PR text -- can
// refuse a malformed -toolchain or -target before printing anything,
// without exporting either pattern out of core for the sole benefit of a
// pre-check duplicate of what Run/Plan will reject anyway.
var (
	idPattern     = regexp.MustCompile(`^[a-z0-9]+$`)
	targetPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

func main() {
	toolchainID := flag.String("toolchain", "go", "toolchain id to bump, e.g. go")
	target := flag.String("target", "", "target version, e.g. 1.26.7")
	printDelivery := flag.Bool("print-delivery", false, "print the branch/commit/pr_title manifest for the given -toolchain/-target and exit, without running the bump")
	flag.Parse()

	if *printDelivery {
		if err := printDeliveryManifest(os.Stdout, *toolchainID, *target); err != nil {
			fmt.Fprintf(os.Stderr, "toolbump: %v\n", err)
			os.Exit(1)
		}
		return
	}

	tc, ok := toolchains.Registry[*toolchainID]
	if !ok {
		fmt.Fprintf(os.Stderr, "toolbump: unknown toolchain %q (known: %s)\n", *toolchainID, knownToolchainIDs())
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "toolbump: cannot determine working directory: %v\n", err)
		os.Exit(1)
	}

	// Threat matrix: git repository selection. Refuse fast, before
	// core.Run does anything else, if cwd is missing any of tc's own
	// RootMarkers -- no caller-supplied path, cwd is the only source of
	// truth (mirrors scripts/gobump#280's requireRepoRoot, now
	// toolchain-owned data instead of a hardcoded go.work/.dagger pair).
	if err := requireRootMarkers(cwd, tc.RootMarkers()); err != nil {
		fmt.Fprintf(os.Stderr, "toolbump: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := core.Run(ctx, tc, cwd, *target); err != nil {
		fmt.Fprintf(os.Stderr, "toolbump: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("toolbump: bumped every site to %s\n", *target)
}

// requireRootMarkers returns a descriptive error unless cwd contains every
// one of markers as a direct child -- generalizing scripts/gobump#280's
// requireRepoRoot (which hardcoded exactly go.work + .dagger) over an
// arbitrary Toolchain's own RootMarkers() data. Checked before core.Run
// does anything else (Threat Matrix: git repository selection), so a
// wrong-cwd invocation hard-errors with zero side effects -- no Dagger
// client connect, no Directory read, no file write, not even a stat past
// the markers checked here.
func requireRootMarkers(cwd string, markers []string) error {
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(cwd, marker)); err != nil {
			return fmt.Errorf("must be run from the repository root: %s not found in %s", marker, cwd)
		}
	}
	return nil
}

// printDeliveryManifest writes a newline key=value manifest
// (branch=/commit=/pr_title=) for id/target to w, for
// scripts/bump-toolchain-pr.sh to parse via `read`, never `eval`
// (design.md D-4). id and target are re-validated against the same strict
// formats core.ValidateToolchain/core.Plan enforce deeper in a real run
// (design.md's "New hard guard"): a malformed value is rejected here, with
// nothing written to w, before Registry is even consulted.
//
// branch/commit derive from id/Display() the same way scripts/gobump#280's
// hardcoded "go"/"Go" literals + bump-go-version-pr.sh's own
// "chore/go-<target>" / "chore(go): bump Go toolchain to <target>" text
// did, so Go's own manifest is byte-identical to that legacy output
// (design.md's "Go delivery text unchanged" scenario).
func printDeliveryManifest(w io.Writer, id, target string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("toolchain id %q must match %s", id, idPattern.String())
	}
	if !targetPattern.MatchString(target) {
		return fmt.Errorf("target version %q must match %s", target, targetPattern.String())
	}

	tc, ok := toolchains.Registry[id]
	if !ok {
		return fmt.Errorf("unknown toolchain %q (known: %s)", id, knownToolchainIDs())
	}

	branch := fmt.Sprintf("chore/%s-%s", tc.ID(), target)
	commit := fmt.Sprintf("chore(%s): bump %s toolchain to %s", tc.ID(), tc.Display(), target)

	if _, err := fmt.Fprintf(w, "branch=%s\n", branch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "commit=%s\n", commit); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "pr_title=%s\n", commit); err != nil {
		return err
	}
	return nil
}

// knownToolchainIDs lists Registry's keys for an error message naming what
// IS valid, sorted for deterministic output across runs.
func knownToolchainIDs() string {
	ids := make([]string, 0, len(toolchains.Registry))
	for id := range toolchains.Registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return strings.Join(ids, ", ")
}
