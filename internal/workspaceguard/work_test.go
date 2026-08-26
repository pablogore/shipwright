// Package workspaceguard guards design.md D1's go.work isolation decision:
// go.work declares which modules participate in the workspace and how
// dependency resolution works, but it does NOT make `go build ./...` or
// `go test -race ./...` from root traverse nested modules — each module
// must be tested/built explicitly (e.g., `cd providers/go && go test ./...`).
// This package enforces which modules may appear in the workspace via an
// explicit, automated allowlist rather than a comment. This is a plain
// root-module unit test — it reads go.work and CI/Makefile as text,
// mirroring internal/daggerpin's pin_test.go, and never requires a
// running Dagger engine.
package workspaceguard

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
	// this file lives at <repoRoot>/internal/workspaceguard/work_test.go
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// wantUse is the exact, allowlisted `use` set design.md D1 commits to: only
// the root module and providers/go, nothing else -- an addition of any
// third member (not only .dagger) must fail this test, not only a
// .dagger-specific one.
var wantUse = []string{".", "./providers/go"}

func TestGoWorkUseSetIsExactAllowlist(t *testing.T) {
	root := repoRoot(t)

	got, err := UsePaths(filepath.Join(root, "go.work"))
	if err != nil {
		t.Fatalf("UsePaths() error = %v, want nil", err)
	}

	if len(got) != len(wantUse) {
		t.Fatalf("go.work use set = %v, want exactly %v (design.md D1 allowlist)", got, wantUse)
	}
	for i, want := range wantUse {
		if got[i] != want {
			t.Fatalf("go.work use set = %v, want exactly %v (design.md D1 allowlist)", got, wantUse)
		}
	}
}

func TestGoWorkRejectsDaggerUse(t *testing.T) {
	if use, found := DaggerUse([]string{".", "./providers/go"}); found {
		t.Fatalf("DaggerUse(%q) = true, want false for the committed allowlist", use)
	}

	tests := []struct {
		name string
		use  string
	}{
		{name: "bare .dagger", use: "./.dagger"},
		{name: "bare dagger", use: "./dagger"},
		{name: "nested .dagger segment", use: "./sub/.dagger"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			use, found := DaggerUse([]string{".", "./providers/go", tt.use})
			if !found {
				t.Fatalf("DaggerUse() = false, want true for %q -- design.md D-B isolation must fail closed", tt.use)
			}
			if use != tt.use {
				t.Fatalf("DaggerUse() reported %q, want %q", use, tt.use)
			}
		})
	}
}

func TestUsePaths_MissingFileFailsClosed(t *testing.T) {
	_, err := UsePaths(filepath.Join(t.TempDir(), "does-not-exist.work"))
	if err == nil {
		t.Fatal("UsePaths() with a missing go.work must return an error, got nil")
	}
}

func TestUsePaths_UnparseableFileFailsClosed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.work")
	if err := os.WriteFile(path, []byte("not a valid go.work file {{{"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v, want nil", path, err)
	}

	_, err := UsePaths(path)
	if err == nil {
		t.Fatal("UsePaths() with an unparseable go.work must return an error, got nil")
	}
}

// TestCIMakefileExplicitlyTestsProvidersGo verifies that the Makefile and CI
// workflow explicitly test/build providers/go as a separate module. Since
// `go test ./...` from root never traverses nested modules (see design.md
// Open Questions), CI would silently skip providers/go tests without these
// explicit invocations. This test makes it impossible to forget them.
func TestCIMakefileExplicitlyTestsProvidersGo(t *testing.T) {
	root := repoRoot(t)

	t.Run("Makefile test target includes providers/go", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(root, "Makefile"))
		if err != nil {
			t.Fatalf("ReadFile(Makefile) error = %v", err)
		}
		content := string(data)

		// The test target must explicitly cd into providers/go and run tests
		if !strings.Contains(content, "cd providers/go") {
			t.Error("Makefile does not include 'cd providers/go' — " +
				"go test ./... from root never traverses nested modules")
		}
	})

	t.Run("CI test step includes providers/go", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
		if err != nil {
			t.Fatalf("ReadFile(ci.yml) error = %v", err)
		}
		content := string(data)

		if !strings.Contains(content, "cd providers/go") {
			t.Error("CI workflow does not include 'cd providers/go' — " +
				"CI would silently skip providers/go tests")
		}
	})
}
