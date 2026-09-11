// Package gotoolchain is the Go core.Toolchain descriptor (issue #284,
// design.md D-2/D-3): it turns scripts/gobump#280's two hardcoded
// GoRuntimeUpgrader.Upgrade calls (workspace "." and single-module
// ".dagger") and its literal-replacement Sites registry into data a
// generic driver (internal/toolbump/core) can run without any
// Go-specific knowledge — the CLI selects "go" by ID, core.Run does the
// rest.
//
// Package name is "gotoolchain", not "golang" (providers/go's own
// package name, which this package imports and wraps unchanged, design.md
// D-1) or the directory's own "go" (a reserved word — illegal as a Go
// package identifier). Every call site that needs both packages would
// otherwise need an import alias for one of them; a distinct name avoids
// that entirely.
package gotoolchain

import (
	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/toolbump/core"
	"github.com/pablogore/shipwright/pkg/shipwright"
	golang "github.com/pablogore/shipwright/providers/go"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// hostExcludes is passed to every Host().Directory call a driver run
// makes for each Target below (design.md D-7 layer 1), moved verbatim
// from scripts/gobump/main.go#280's own hostExcludes var — see that
// file's own doc comment for why ".git" is listed alongside ".git/**": a
// live dry run showed excluding only ".git/**" leaves an empty top-level
// .git directory entry that trips Go's VCS auto-stamping into shelling
// out to `git status` against an incomplete tree, which then fails
// GoRuntimeUpgrader's read-only `go build ./...` validation step.
var hostExcludes = []string{"**/testdata/**", ".git", ".git/**", "coverage/**", "dist/**"}

// Toolchain is the Go core.Toolchain, registered under "go" by
// toolchains/registry.go (design.md D-2). A package-level singleton, not
// a constructor: Go has exactly one instance in this repo, and every
// field below is data #280's main.go used to hardcode, not runtime
// configuration a caller chooses between.
var Toolchain core.Toolchain = goToolchain{}

// goToolchain implements core.Toolchain for Go. It carries no
// per-instance state — every method returns fixed data or constructs a
// fresh *golang.GoRuntimeUpgrader — so Toolchain above can be one shared
// value rather than something callers construct themselves.
type goToolchain struct{}

// ID names this toolchain for delivery/CLI selection (design.md D-2);
// "go" already matches core.ValidateToolchain's required
// lowercase-alphanumeric format.
func (goToolchain) ID() string { return "go" }

// Display is the human-readable name used in commit/PR text (design.md
// D-3).
func (goToolchain) Display() string { return "Go" }

// RootMarkers lists the two paths scripts/gobump/main.go#280's own
// requireRepoRoot checked before doing anything else: go.work (the root
// workspace file) and .dagger (the standalone single-module directory).
// Enforced by the future CLI (Phase 3), never by this package or core.
func (goToolchain) RootMarkers() []string { return []string{"go.work", ".dagger"} }

// Sites returns Go's literal-replacement registry (issue #280, design.md
// D-5): every plain-string Go version pin GoRuntimeUpgrader's
// go.mod/go.work/.go-version handling cannot reach. Relocated verbatim
// from scripts/gobump/sites.Sites — extracted out of internal/toolbump/
// core in Phase 1 for exactly this reason (a Site registry is
// toolchain-specific data, not part of the generic engine). See sites'
// own doc comment below for the full audit history and per-site
// rationale, none of which changes here.
func (goToolchain) Sites() []core.Site { return sites }

// Targets returns Go's two hardcoded mutation sites from #280's
// main.go/run (design.md D-2 rows 1-2), in the same order main.go ran
// them: the go.work workspace root (".") first, then the .dagger
// single-module directory, deliberately excluded from go.work
// participation (design.md D-3). Both share hostExcludes above, the same
// exclude list #280 passed to every Host().Directory call.
func (goToolchain) Targets() []core.Target {
	return []core.Target{
		{Source: ".", Export: ".", Excludes: hostExcludes},
		{Source: ".dagger", Export: ".dagger", Excludes: hostExcludes},
	}
}

// Upgrader constructs the same *golang.GoRuntimeUpgrader configuration
// #280's main.go/run hardcoded for BOTH of its calls (WorkspaceRoot ".",
// Tidy false, AllowDowngrade false) — target is unused because Go's
// Upgrade behavior does not vary by which Target it mutates; the
// parameter exists to satisfy core.Toolchain's contract for a toolchain
// whose Upgrader construction DOES vary per Target.
func (goToolchain) Upgrader(client *dagger.Client, _ core.Target) shipwright.RuntimeUpgrader {
	return &golang.GoRuntimeUpgrader{
		Client:         daggerkit.NewDaggerAdapter(client),
		WorkspaceRoot:  ".",
		Tidy:           false,
		AllowDowngrade: false,
	}
}

// sites is Go's production literal-replacement registry (issue #280,
// design.md D-5): every plain-string Go version pin that
// GoRuntimeUpgrader's go.mod/go.work/.go-version handling cannot reach.
// Relocated verbatim from scripts/gobump/sites.Sites (which remains
// untouched at its original location until Phase 4 deletes it) —
// populated from a fresh, full-repo audit and cross-checked against
// D-7's exclusions: nothing under providers/go/testdata/runtime/**, and
// .dagger/release_test.go and testing/integration/go/integration_test.go
// stay deliberately unregistered (they assert behavior, not the repo's
// pin).
//
// Decision (tasks.md 1.2, carried forward unchanged): the illustrative
// doc-comment literals in internal/pipelines/shared/builder.go,
// internal/pipelines/shared/tests.go, and
// internal/pipelines/test/gotester.go ARE registered. They are cheap to
// keep in sync, each one currently restates the real fallback default
// used a few lines below it (not an arbitrary format example), and a
// removed/reformatted comment is caught loudly by E2 rather than
// silently going stale.
//
// Two categories found during the original census that were deliberately
// left OUT of this registry, because their literals do not track "the"
// current pin:
//   - "or later"/"or superior" minimum-version floors (docs/PRODUCTION_DEPLOYMENT.md's
//     "Go 1.21 or later", docs/LOCAL_USAGE.md's "Go 1.25.5 o superior
//     instalado") state a support floor, not the exact toolchain pin; a
//     bump does not necessarily need to raise them in lockstep.
//   - docs/MERGE_CONFLICT_RESOLUTION.md's two GO_VERSION literals
//     (1.25.1 / 1.25.2) are a worked git-conflict-marker example that
//     deliberately shows two DIFFERENT values; forcing both to the same
//     target would corrupt the example it's illustrating, not fix a pin.
//   - docs/CONFIGURATION.md's "Example" table column (`1.24.0`),
//     docs/INTEGRATION_GUIDE.md's `go-version: ['1.25.5', '1.26.1']` test
//     matrix, internal/pipelines/options.go's and
//     internal/config/validation.go's format examples (`1.24.2`, `1.25`)
//     are illustrative syntax samples that do not equal the repo's actual
//     current pin and are not meant to track it.
var sites = []core.Site{
	// -- Workflow GO_VERSION env vars ------------------------------------
	{Path: ".github/workflows/ci.yml", Anchor: "GO_VERSION: '{{V}}'", Occurrences: 1},
	{Path: ".github/workflows/release-provider-go.yml", Anchor: "GO_VERSION: '{{V}}'", Occurrences: 1},
	{Path: ".github/workflows/release-provider-rust.yml", Anchor: "GO_VERSION: '{{V}}'", Occurrences: 1},

	// -- Go source fallback/default literals -----------------------------
	{Path: "providers/go/gobuilder.go", Anchor: `const defaultGoVersion = "{{V}}"`, Occurrences: 1},
	{Path: "internal/config/config.go", Anchor: `DefaultGoVersion       = "{{V}}"`, Occurrences: 1},
	{Path: "internal/app/step_handlers.go", Anchor: `goVersion = "{{V}}"`, Occurrences: 1},
	{Path: "internal/executors/docker_executor.go", Anchor: `goVersion = "{{V}}"`, Occurrences: 1},
	{Path: "internal/pipelines/test/gotester.go", Anchor: "default to {{V}} if not set", Occurrences: 1},
	{Path: "internal/pipelines/test/gotester.go", Anchor: `goVersion = "{{V}}"`, Occurrences: 1},
	{Path: "internal/pipelines/shared/tests.go", Anchor: `(e.g., "{{V}}").`, Occurrences: 1},
	{Path: "internal/pipelines/shared/tests.go", Anchor: `goVersion = "{{V}}"`, Occurrences: 1},
	{Path: "internal/pipelines/shared/builder.go", Anchor: `(e.g., "1.21", "{{V}}").`, Occurrences: 1},

	// -- Docs ------------------------------------------------------------
	{Path: "docs/CONFIGURATION.md", Anchor: "go_version: {{V}}", Occurrences: 4},
	{Path: "docs/CONFIGURATION.md", Anchor: "`SHIPWRIGHT_GO_VERSION` | Go version to use | `{{V}}` | `1.24.0` |", Occurrences: 1},
	{Path: "docs/CONFIGURATION.md", Anchor: "base_image: golang:{{V}}-alpine", Occurrences: 1},
	{Path: "docs/LOCAL_USAGE.md", Anchor: `go_version: "{{V}}"`, Occurrences: 1},
	{Path: "docs/LOCAL_USAGE.md", Anchor: "SHIPWRIGHT_PIPELINE_GO_VERSION={{V}}", Occurrences: 1},
	{Path: "docs/WORKFLOW_GUIDE.md", Anchor: `goVersion: "{{V}}"`, Occurrences: 2},
	{Path: "docs/INTEGRATION_GUIDE.md", Anchor: `go_version: "{{V}}"`, Occurrences: 1},
	{Path: "docs/PIPELINE_DEVELOPMENT.md", Anchor: `From("golang:{{V}}-alpine").`, Occurrences: 1},
	{Path: "docs/PRD.md", Anchor: "Go `{{V}}` per `go.mod`/`.go-version`", Occurrences: 1},
	{Path: "docs/ANALYSIS_AND_RECOMMENDATIONS.md", Anchor: "GO_VERSION: '{{V}}'", Occurrences: 2},
	{Path: "docs/ANALYSIS_AND_RECOMMENDATIONS.md", Anchor: "go-version: '{{V}}'", Occurrences: 2},
	{Path: "docs/ANALYSIS_AND_RECOMMENDATIONS.md", Anchor: "golang:{{V}}", Occurrences: 4},
	{Path: "docs/PRODUCTION_DEPLOYMENT.md", Anchor: "image: golang:{{V}}", Occurrences: 1},
}
