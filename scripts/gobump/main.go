// Command gobump is the coordinated Go SDK/toolchain version bump CLI
// (issue #280, design.md D-1/D-2/D-3/D-6). It wires the pure
// scripts/gobump/sites registry (literal-replacement sites design.md
// D-4/D-5 cannot reach through GoRuntimeUpgrader) together with
// GoRuntimeUpgrader (providers/go, unchanged) to mutate every Go-module
// and literal-replacement site to a single target version, in the
// plan-then-mutate order design.md D-6 requires:
//
//  1. sites.Plan first — pure, cheapest, zero side effects. If the
//     registry is broken (moved/renamed anchor, missing file, ...), the
//     run exits before a Dagger client is even connected: no Upgrade
//     call, no Export, no write.
//  2. Hardcoded GoRuntimeUpgrader.Upgrade call #1, workspace mode:
//     Host().Directory(".") — the repo-root go.work workspace (root
//     go.mod, providers/go/go.mod, providers/rust/go.mod, go.work,
//     .go-version).
//  3. Hardcoded GoRuntimeUpgrader.Upgrade call #2, single-module mode:
//     Host().Directory(".dagger") — .dagger/go.mod, deliberately excluded
//     from go.work (design.md D-3), handled by the same Upgrade method's
//     existing single-module branch unchanged.
//  4. Export both resulting directories back to the host, never Wipe, and
//     only after BOTH Upgrade calls have succeeded.
//  5. sites.Apply last, using the edits Plan already computed in step 1.
//
// scripts/gobump/main.go itself makes no filesystem or Dagger-client
// decision that isn't already covered by this order: requireRepoRoot is
// the only pure, independently unit-tested logic here (Threat Matrix: git
// repository selection). Everything past it requires a live Dagger
// engine and is exercised by Phase 8's integration dry run, not a unit
// test in this package.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"dagger.io/dagger"

	golang "github.com/pablogore/shipwright/providers/go"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
	"github.com/pablogore/shipwright/scripts/gobump/sites"
)

// hostExcludes is passed to every Host().Directory call this package
// makes (design.md D-7 layer 1): excluded bytes are never part of the
// Directory Dagger reads, so on-disk testdata/.git/coverage/dist content
// is unreachable by construction, not by policy — the same guarantee
// D-7 layer 2 (sites' own E5 check) gives the literal-replacement
// registry independently.
//
// ".git" is listed alongside ".git/**" — a minimal, empirically required
// addition beyond design.md D-7's literal 4-entry list. A live dry run
// (tasks.md 4/Phase 8) showed ".git/**" alone excludes only .git's
// *contents*, leaving an empty top-level .git directory entry in the
// mounted Directory. Go's default VCS auto-stamping detects that empty
// .git as a real repository marker and shells out to `git status`, which
// fails with "exit status 128" against the incomplete directory — causing
// GoRuntimeUpgrader's unchanged, read-only `go build ./...` validation
// step (runtimeupgrader.go) to fail on every run. Excluding the bare
// ".git" entry too keeps it out of the Directory entirely, matching
// design.md D-7's own stated intent ("excluded bytes are never *in* the
// Directory") rather than changing it.
var hostExcludes = []string{"**/testdata/**", ".git", ".git/**", "coverage/**", "dist/**"}

func main() {
	target := flag.String("target", "", "target Go version, e.g. 1.26.7")
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobump: cannot determine working directory: %v\n", err)
		os.Exit(1)
	}

	// Threat matrix: git repository selection. Refuse fast, before any
	// Dagger client connection is made, if cwd is not the repo root — no
	// git -C, no caller-supplied repo path, cwd is the only source of
	// truth (design.md Threat Matrix, tasks.md 4.1).
	if err := requireRepoRoot(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "gobump: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Step 1 of plan-then-mutate (design.md D-6): resolves every
	// registered literal-replacement site and aggregates every failure.
	// Zero bytes written by Plan itself, and — because this runs before
	// dagger.Connect — no Dagger client is even created if the plan
	// fails, so no Upgrade/Export can possibly run after it either.
	edits, err := sites.Plan(cwd, sites.Sites, *target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobump: site plan failed, no site was mutated: %v\n", err)
		os.Exit(1)
	}

	client, err := dagger.Connect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobump: failed to connect to Dagger: %v\n", err)
		os.Exit(1)
	}
	// Not deferred: every error path below calls os.Exit, which skips
	// deferred calls entirely (gocritic: exitAfterDefer). run() closes the
	// client explicitly on its own return so both the success and the
	// error paths still close it before main() translates run's error
	// into an os.Exit(1).
	if err := run(ctx, client, cwd, *target, edits); err != nil {
		client.Close()
		fmt.Fprintf(os.Stderr, "gobump: %v\n", err)
		os.Exit(1)
	}
	if err := client.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "gobump: failed to close Dagger client: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("gobump: bumped every site to %s\n", *target)
}

// run performs every step of plan-then-mutate (design.md D-6) that
// requires a live Dagger client: both hardcoded GoRuntimeUpgrader.Upgrade
// calls, exporting their results back to the host, and applying edits
// last. Split out of main so main itself never defers client.Close after
// an os.Exit-reachable error path (gocritic: exitAfterDefer) — every
// return here goes back through main, which closes the client exactly
// once on every path.
func run(ctx context.Context, client *dagger.Client, cwd, target string, edits []sites.Edit) error {
	adapter := daggerkit.NewDaggerAdapter(client)

	// Step 2: workspace-mode Upgrade (design.md D-2 row 1) — root
	// go.mod, providers/go/go.mod, providers/rust/go.mod, go.work, and
	// .go-version, discovered and mutated by GoRuntimeUpgrader's own
	// go.work-aware path. WorkspaceRoot is report metadata only (see
	// runtimeupgrader.go's Upgrade); it does not affect which files are
	// read or written.
	workspaceUpgrader := &golang.GoRuntimeUpgrader{
		Client:         adapter,
		WorkspaceRoot:  ".",
		Tidy:           false,
		AllowDowngrade: false,
	}
	workspaceSrc := client.Host().Directory(".", dagger.HostDirectoryOpts{Exclude: hostExcludes})
	workspaceResult, err := workspaceUpgrader.Upgrade(ctx, workspaceSrc, target)
	if err != nil {
		return fmt.Errorf("go.work workspace upgrade failed: %w", err)
	}

	// Step 3: single-module Upgrade (design.md D-2 row 2 / D-3) —
	// .dagger/go.mod, deliberately excluded from go.work participation.
	// Because the Host().Directory source is already scoped to ".dagger",
	// Upgrade's own root/toolchain detection sees no go.work at this
	// Directory's top level and takes its existing single-module branch
	// unchanged; no new API on GoRuntimeUpgrader (design.md D-3).
	daggerUpgrader := &golang.GoRuntimeUpgrader{
		Client:         adapter,
		WorkspaceRoot:  ".",
		Tidy:           false,
		AllowDowngrade: false,
	}
	daggerSrc := client.Host().Directory(".dagger", dagger.HostDirectoryOpts{Exclude: hostExcludes})
	daggerResult, err := daggerUpgrader.Upgrade(ctx, daggerSrc, target)
	if err != nil {
		return fmt.Errorf(".dagger module upgrade failed: %w", err)
	}

	// Step 4: export both mutated directories back to the host, only
	// after BOTH Upgrade calls have succeeded. Never Wipe — the zero
	// value already means "merge with existing host contents", so
	// anything Host().Directory excluded above (testdata/.git/coverage/
	// dist) is left untouched on disk rather than deleted.
	if _, err := workspaceResult.Export(ctx, "."); err != nil {
		return fmt.Errorf("failed to export the go.work workspace: %w", err)
	}
	if _, err := daggerResult.Export(ctx, ".dagger"); err != nil {
		return fmt.Errorf("failed to export the .dagger module: %w", err)
	}

	// Step 5: literal-replacement sites, last — only after every
	// Go-module site is already known-good on disk. Apply's contract
	// (sites.go) is to be called only with edits from a Plan that
	// returned no error for the same site set, which is exactly what
	// step 1 already guaranteed.
	if err := sites.Apply(cwd, edits); err != nil {
		return err
	}

	return nil
}

// requireRepoRoot returns a descriptive error unless cwd is the
// repository root: it must directly contain both go.work (the root Go
// workspace file) and .dagger (the standalone Dagger module directory).
// Checked before any Dagger client connection is made (Threat Matrix:
// git repository selection, tasks.md 4.1/4.2) so a wrong-cwd invocation
// hard-errors with zero side effects — no client connect, no Directory
// read, no file write, not even a stat past the two checked here.
func requireRepoRoot(cwd string) error {
	goWorkPath := filepath.Join(cwd, "go.work")
	info, err := os.Stat(goWorkPath)
	if err != nil {
		return fmt.Errorf("must be run from the repository root: go.work not found in %s", cwd)
	}
	if info.IsDir() {
		return fmt.Errorf("must be run from the repository root: %s is a directory, not the go.work file", goWorkPath)
	}

	daggerDirPath := filepath.Join(cwd, ".dagger")
	info, err = os.Stat(daggerDirPath)
	if err != nil {
		return fmt.Errorf("must be run from the repository root: .dagger not found in %s", cwd)
	}
	if !info.IsDir() {
		return fmt.Errorf("must be run from the repository root: %s is a file, not the .dagger directory", daggerDirPath)
	}

	return nil
}
