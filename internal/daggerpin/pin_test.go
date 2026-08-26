// Package daggerpin guards design.md D-B's pin-coupling decision: the root
// module's dagger.io/dagger client version and dagger.json's engineVersion
// live in separate Go modules (root vs .dagger/) and never link at compile
// time, so drift between them can only be caught by an explicit test. This
// is a plain root-module unit test — it reads dagger.json and go.mod as
// text/JSON, it never requires a running Dagger engine.
package daggerpin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed to resolve this test file's path")
	}
	// this file lives at <repoRoot>/internal/daggerpin/pin_test.go
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func TestEngineVersionMatchesGoModDaggerVersion(t *testing.T) {
	root := repoRoot(t)

	engineVersion, err := EngineVersion(filepath.Join(root, "dagger.json"))
	if err != nil {
		t.Fatalf("EngineVersion() error = %v, want nil", err)
	}

	goModVersion, err := GoModDaggerVersion(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("GoModDaggerVersion() error = %v, want nil", err)
	}

	if engineVersion != goModVersion {
		t.Fatalf(
			"dagger.json engineVersion = %q, root go.mod dagger.io/dagger = %q — D-B requires these to match; if dagger init refused this pin, bump both together (design.md D-B)",
			engineVersion, goModVersion,
		)
	}
}

func TestEngineVersion_MissingFileFailsClosed(t *testing.T) {
	_, err := EngineVersion(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err == nil {
		t.Fatal("EngineVersion() with a missing dagger.json must return an error, got nil")
	}
}

func TestGoModDaggerVersion_MissingRequireFailsClosed(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(modPath, []byte("module example.com/nodagger\n\ngo 1.26.1\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v, want nil", modPath, err)
	}

	_, err := GoModDaggerVersion(modPath)
	if err == nil {
		t.Fatal("GoModDaggerVersion() must return an error when go.mod has no dagger.io/dagger requirement")
	}
}

// TestGoModDaggerVersion_ReplaceDirectiveWins guards against the gap found in
// PR #148 review: a `replace dagger.io/dagger => ...` directive changes what
// actually gets built, so the pin-parity guard must report the REPLACED
// version, not the bare `require` version — otherwise the guard silently
// stops detecting drift the moment a replace directive is introduced
// (design.md D-B: "Drift fails RED instead of surviving review").
func TestGoModDaggerVersion_ReplaceDirectiveWins(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "go.mod")
	content := "module example.com/replaced\n\n" +
		"go 1.26.1\n\n" +
		"require dagger.io/dagger v0.21.8\n\n" +
		"replace dagger.io/dagger => dagger.io/dagger v0.22.0\n"
	if err := os.WriteFile(modPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v, want nil", modPath, err)
	}

	got, err := GoModDaggerVersion(modPath)
	if err != nil {
		t.Fatalf("GoModDaggerVersion() error = %v, want nil", err)
	}

	const want = "v0.22.0"
	if got != want {
		t.Fatalf("GoModDaggerVersion() = %q, want %q (the replace directive's version, not the require version v0.21.8)", got, want)
	}
}

// TestGoModDaggerVersion_LocalPathReplaceFailsClosed guards against a local
// filesystem replace directive (`replace dagger.io/dagger => ../dagger`)
// silently producing an empty effective version. modfile's parsed New.Version
// is "" for a local path replace (there is no version to compare), so
// returning it as-is would make the pin-parity test report a confusing
// "v0.21.8 != ”" mismatch instead of naming the actual, unrelated problem:
// a local replace has no comparable version at all.
func TestGoModDaggerVersion_LocalPathReplaceFailsClosed(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "go.mod")
	content := "module example.com/localreplace\n\n" +
		"go 1.26.1\n\n" +
		"require dagger.io/dagger v0.21.8\n\n" +
		"replace dagger.io/dagger => ../dagger\n"
	if err := os.WriteFile(modPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v, want nil", modPath, err)
	}

	_, err := GoModDaggerVersion(modPath)
	if err == nil {
		t.Fatal("GoModDaggerVersion() error = nil, want a fail-closed error naming the local replace path")
	}

	const wantSubstring = "../dagger"
	if !strings.Contains(err.Error(), wantSubstring) {
		t.Fatalf("GoModDaggerVersion() error = %q, want it to name the local replace path %q", err.Error(), wantSubstring)
	}
}
