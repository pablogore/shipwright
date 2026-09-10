package main

import (
	"os"
	"path/filepath"
	"testing"
)

// mkRepoRoot builds a t.TempDir() fixture containing exactly the entries a
// real repo root has: a go.work file and a .dagger directory.
func mkRepoRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.26.7\n"), 0o644); err != nil {
		t.Fatalf("write go.work fixture: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, ".dagger"), 0o755); err != nil {
		t.Fatalf("mkdir .dagger fixture: %v", err)
	}
	return root
}

func TestRequireRepoRootValidRootPasses(t *testing.T) {
	root := mkRepoRoot(t)

	if err := requireRepoRoot(root); err != nil {
		t.Fatalf("expected a valid repo root to pass, got: %v", err)
	}
}

func TestRequireRepoRootSubdirectoryHardErrors(t *testing.T) {
	root := mkRepoRoot(t)
	sub := filepath.Join(root, "scripts", "gobump")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir subdirectory fixture: %v", err)
	}

	err := requireRepoRoot(sub)
	if err == nil {
		t.Fatal("expected a subdirectory cwd to hard-error, got nil")
	}

	// Zero side effects: the subdirectory guard must not have created,
	// removed, or modified anything under root.
	entries, readErr := os.ReadDir(sub)
	if readErr != nil {
		t.Fatalf("re-read subdirectory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("expected zero side effects in %s, found: %v", sub, entries)
	}
}

func TestRequireRepoRootMissingGoWorkHardErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".dagger"), 0o755); err != nil {
		t.Fatalf("mkdir .dagger fixture: %v", err)
	}

	if err := requireRepoRoot(root); err == nil {
		t.Fatal("expected a missing go.work to hard-error, got nil")
	}
}

func TestRequireRepoRootMissingDaggerDirHardErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.26.7\n"), 0o644); err != nil {
		t.Fatalf("write go.work fixture: %v", err)
	}

	if err := requireRepoRoot(root); err == nil {
		t.Fatal("expected a missing .dagger directory to hard-error, got nil")
	}
}

func TestRequireRepoRootGoWorkAsDirectoryHardErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "go.work"), 0o755); err != nil {
		t.Fatalf("mkdir go.work-as-directory fixture: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, ".dagger"), 0o755); err != nil {
		t.Fatalf("mkdir .dagger fixture: %v", err)
	}

	if err := requireRepoRoot(root); err == nil {
		t.Fatal("expected go.work being a directory to hard-error, got nil")
	}
}

func TestRequireRepoRootDaggerAsFileHardErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.26.7\n"), 0o644); err != nil {
		t.Fatalf("write go.work fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".dagger"), []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("write .dagger-as-file fixture: %v", err)
	}

	if err := requireRepoRoot(root); err == nil {
		t.Fatal("expected .dagger being a file to hard-error, got nil")
	}
}

func TestRequireRepoRootEmptyDirectoryHardErrors(t *testing.T) {
	root := t.TempDir()

	if err := requireRepoRoot(root); err == nil {
		t.Fatal("expected an empty directory (neither go.work nor .dagger) to hard-error, got nil")
	}
}
