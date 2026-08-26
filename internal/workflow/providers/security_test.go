// SECURITY test — tasks.md 7.4: a static assertion (not a documentation
// comment) that no manifest-reachable code path can call plugin.Open.
// design.md D-I explicitly rejects reusing internal/plugins/loader.go's
// plugin.Open for manifest-declared providers (supply-chain / arbitrary
// code execution risk) — this test PROVES that boundary rather than
// merely asserting it in prose.
package providers_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// modulePrefix is this repository's module path (go.mod). Only imports
// under this prefix are followed transitively — third-party/stdlib
// dependencies are trusted per design.md's Threat Matrix ("the residual
// concern is ordinary Go dependency review, which go.mod/go.sum already
// covers"); this test's job is to prove OUR code never imports "plugin",
// not to re-audit every transitive dependency's own internals.
const modulePrefix = "github.com/pablogore/shipwright"

// pluginPackageImportPath is the standard library package whose Open
// function loads and executes arbitrary native code
// (internal/plugins/loader.go's own attack surface, design.md D-I).
const pluginPackageImportPath = "plugin"

// TestNoPluginOpenReachableFromProviderResolution is tasks.md 7.4's
// SECURITY RED-then-GREEN static assertion. It computes the transitive
// Go IMPORT graph rooted at this package (internal/workflow/providers,
// the provider-resolution entrypoint design.md D-I names) using only
// go/parser — no `go list` subprocess, no build tags, nothing that could
// itself be fooled by an unusual build configuration — and asserts:
//
//  1. the standard library "plugin" package never appears anywhere in
//     that transitive closure, and
//  2. internal/plugins (the package that literally calls plugin.Open)
//     never appears in that transitive closure either.
//
// Why this PROVES the property, not just documents it: Go requires a
// package to IMPORT a symbol's package before it can reference that
// symbol at all (there is no reflection-only way to invoke a stdlib
// function by name across package boundaries without importing it, short
// of unsafe/linkname tricks that are themselves a separate, detectable
// category of attack this repository does not use). If "plugin" is absent
// from provider resolution's entire import closure, no code reachable
// from provider resolution can contain a valid `plugin.Open(...)` call —
// full stop, mechanically, not by policy.
func TestNoPluginOpenReachableFromProviderResolution(t *testing.T) {
	t.Parallel()

	repoRoot := findRepoRoot(t)

	visited := map[string]bool{}
	var badImporters []string
	var pluginPackageFound bool
	var pluginsPackageFound bool

	var walk func(importPath string)
	walk = func(importPath string) {
		if visited[importPath] {
			return
		}
		visited[importPath] = true

		dir, ok := resolveModuleDir(repoRoot, importPath)
		if !ok {
			// Not a package under this module (e.g. it IS the "plugin"
			// stdlib import itself, or another external dependency) —
			// nothing to walk into.
			return
		}

		imports := packageImports(t, dir)
		for _, imp := range imports {
			if imp == pluginPackageImportPath {
				pluginPackageFound = true
				badImporters = append(badImporters, importPath)
			}
			if imp == modulePrefix+"/internal/plugins" {
				pluginsPackageFound = true
				badImporters = append(badImporters, importPath)
			}
			if imp == modulePrefix || len(imp) > len(modulePrefix) && imp[:len(modulePrefix)+1] == modulePrefix+"/" {
				walk(imp)
			}
		}
	}

	walk(modulePrefix + "/internal/workflow/providers")

	if len(visited) < 2 {
		t.Fatalf("import walk visited only %d package(s) (%v) — the walk is not actually traversing anything, this test would pass vacuously", len(visited), visited)
	}

	if pluginPackageFound {
		sort.Strings(badImporters)
		t.Fatalf("standard library %q package is reachable from provider resolution's import graph via: %v — this violates design.md D-I's compile-time-only, no plugin.Open boundary", pluginPackageImportPath, badImporters)
	}
	if pluginsPackageFound {
		sort.Strings(badImporters)
		t.Fatalf("internal/plugins (which calls plugin.Open) is reachable from provider resolution's import graph via: %v — this violates design.md D-I's compile-time-only, no plugin.Open boundary", badImporters)
	}
}

// resolveModuleDir maps a same-module import path to its on-disk
// directory. It reports ok=false for anything outside modulePrefix
// (stdlib, third-party) — those are not walked further, see the test's
// doc comment.
func resolveModuleDir(repoRoot, importPath string) (string, bool) {
	if importPath == modulePrefix {
		return repoRoot, true
	}
	if len(importPath) <= len(modulePrefix)+1 || importPath[:len(modulePrefix)+1] != modulePrefix+"/" {
		return "", false
	}
	rel := importPath[len(modulePrefix)+1:]
	return filepath.Join(repoRoot, filepath.FromSlash(rel)), true
}

// packageImports parses every .go file directly in dir (both production
// and _test.go files, deliberately — this walk is a security boundary
// proof, not a build-graph optimization, so it errs toward walking more,
// never less) using parser.ImportsOnly for speed, and returns the
// deduplicated set of import paths found.
func packageImports(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		// A directory named by an import path that does not exist on
		// disk is not this test's concern (build-constrained or
		// generated files) — treat as no imports rather than failing the
		// whole walk.
		return nil
	}

	seen := map[string]bool{}
	var out []string

	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("packageImports: failed to parse %s: %v", path, err)
		}

		for _, imp := range file.Imports {
			importPath := imp.Path.Value
			// imp.Path.Value includes the surrounding quotes.
			importPath = importPath[1 : len(importPath)-1]
			if seen[importPath] {
				continue
			}
			seen[importPath] = true
			out = append(out, importPath)
		}
	}

	return out
}

// findRepoRoot locates this module's root directory from this test
// file's own path (runtime.Caller), without shelling out to `go env
// GOMOD` or similar — this file lives at
// internal/workflow/providers/security_test.go, three directories below
// the repo root.
func findRepoRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("findRepoRoot: runtime.Caller(0) failed")
	}

	dir := filepath.Dir(thisFile) // internal/workflow/providers
	dir = filepath.Dir(dir)       // internal/workflow
	dir = filepath.Dir(dir)       // internal
	dir = filepath.Dir(dir)       // repo root

	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatalf("findRepoRoot: %s does not contain go.mod (computed from %s): %v", dir, thisFile, err)
	}

	return dir
}
