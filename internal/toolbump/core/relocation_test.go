package core

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRootForRelocation resolves the repository root relative to this test
// file's own location (internal/toolbump/core -> repo root is three levels
// up), the same resolution drift_test.go used at its original
// scripts/gobump/sites location (also three levels from the repo root).
func repoRootForRelocation(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine the caller's file to resolve the repo root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

// significantCodeLines reads path and returns every line that is neither
// blank nor a comment-only line, trimmed of surrounding whitespace — the
// lines that actually define behavior, independent of doc comment
// wording. When dropSitesVar is true, it additionally drops the pre-move
// file's own `var Sites = []Site{...}` block (and only that block) —
// tasks.md 1.1 deliberately extracts that toolchain-specific registry out
// of the generic engine; every other line must still line up exactly.
func significantCodeLines(t *testing.T, path string, dropSitesVar bool) []string {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	defer f.Close()

	var lines []string
	inSitesVar := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if dropSitesVar {
			if line == "var Sites = []Site{" {
				inSitesVar = true
				continue
			}
			if inSitesVar {
				if line == "}" {
					inSitesVar = false
				}
				continue
			}
		}

		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanning %s: %v", path, err)
	}
	return lines
}

// dropLine returns lines with every exact occurrence of target removed.
func dropLine(lines []string, target string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l == target {
			continue
		}
		out = append(out, l)
	}
	return out
}

// TestRelocationIsImportPathOnlyDiff is the standing regression guard for
// spec's "Import-only diff" scenario (sdd/generalized-toolchain-upgrade,
// capability toolchain-bump-core): every significant (non-comment,
// non-blank) code line in the pre-move scripts/gobump/sites/sites.go MUST
// still appear, unchanged and in order, in this package's sites.go, except
// for the package declaration itself and the production var Sites block —
// which tasks.md 1.1 deliberately extracts out of the generic engine (it
// now belongs with the toolchain descriptor that owns the registry, not
// the engine that runs it).
//
// A green run means the relocation altered nothing about how a Site is
// classified or mutated; a red run names exactly which line diverged.
//
// This guard is only meaningful while both copies coexist during the
// PR1-PR3 migration (tasks.md 4.4 removes scripts/gobump/ entirely, at
// which point this test — and the pre-move copy it compares against —
// should be removed together).
func TestRelocationIsImportPathOnlyDiff(t *testing.T) {
	root := repoRootForRelocation(t)
	oldPath := filepath.Join(root, "scripts", "gobump", "sites", "sites.go")
	newPath := filepath.Join(root, "internal", "toolbump", "core", "sites.go")

	if _, err := os.Stat(oldPath); err != nil {
		t.Skipf("pre-move scripts/gobump/sites/sites.go no longer exists (%v) — this guard only applies while both copies coexist", err)
	}

	oldLines := dropLine(significantCodeLines(t, oldPath, true), "package sites")
	newLines := dropLine(significantCodeLines(t, newPath, false), "package core")

	if len(oldLines) != len(newLines) {
		t.Fatalf("relocated sites.go has a non-import-path diff: %d significant code lines in the original vs %d in the relocated copy", len(oldLines), len(newLines))
	}
	for i := range oldLines {
		if oldLines[i] != newLines[i] {
			t.Fatalf("relocated sites.go diverges from the original at significant line %d:\n original:  %s\n relocated: %s", i, oldLines[i], newLines[i])
		}
	}
}
