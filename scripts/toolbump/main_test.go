package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// mkRepoRoot builds a t.TempDir() fixture containing exactly the entries a
// real repo root has for the "go" toolchain: a go.work file and a .dagger
// directory -- the same fixture scripts/gobump/main_test.go#280 used for
// requireRepoRoot, now driving the generalized requireRootMarkers instead.
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

// -- requireRootMarkers: task 3.1's generalization of requireRepoRoot ------

func TestRequireRootMarkersValidRootPasses(t *testing.T) {
	root := mkRepoRoot(t)

	if err := requireRootMarkers(root, []string{"go.work", ".dagger"}); err != nil {
		t.Fatalf("expected a valid repo root to pass, got: %v", err)
	}
}

func TestRequireRootMarkersSubdirectoryHardErrors(t *testing.T) {
	root := mkRepoRoot(t)
	sub := filepath.Join(root, "scripts", "toolbump")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir subdirectory fixture: %v", err)
	}

	err := requireRootMarkers(sub, []string{"go.work", ".dagger"})
	if err == nil {
		t.Fatal("expected a subdirectory cwd to hard-error, got nil")
	}

	// Zero side effects: the subdirectory guard must not have created,
	// removed, or modified anything under sub.
	entries, readErr := os.ReadDir(sub)
	if readErr != nil {
		t.Fatalf("re-read subdirectory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("expected zero side effects in %s, found: %v", sub, entries)
	}
}

func TestRequireRootMarkersMissingFirstMarkerHardErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".dagger"), 0o755); err != nil {
		t.Fatalf("mkdir .dagger fixture: %v", err)
	}

	if err := requireRootMarkers(root, []string{"go.work", ".dagger"}); err == nil {
		t.Fatal("expected a missing first marker to hard-error, got nil")
	}
}

func TestRequireRootMarkersMissingSecondMarkerHardErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.26.7\n"), 0o644); err != nil {
		t.Fatalf("write go.work fixture: %v", err)
	}

	if err := requireRootMarkers(root, []string{"go.work", ".dagger"}); err == nil {
		t.Fatal("expected a missing second marker to hard-error, got nil")
	}
}

func TestRequireRootMarkersEmptyDirectoryHardErrors(t *testing.T) {
	root := t.TempDir()

	if err := requireRootMarkers(root, []string{"go.work", ".dagger"}); err == nil {
		t.Fatal("expected an empty directory (neither marker present) to hard-error, got nil")
	}
}

func TestRequireRootMarkersEmptyMarkerListTriviallyPasses(t *testing.T) {
	root := t.TempDir()

	// A toolchain that declares zero RootMarkers has nothing to check --
	// requireRootMarkers must not fabricate a failure out of an empty list.
	if err := requireRootMarkers(root, nil); err != nil {
		t.Fatalf("expected zero markers to trivially pass, got: %v", err)
	}
}

// -- printDeliveryManifest: tasks 3.2/3.3 -----------------------------------

func TestPrintDeliveryManifestGoIsByteIdenticalToLegacy(t *testing.T) {
	// branch/commit/pr_title must match scripts/gobump/main.go#280's
	// hardcoded "go"/"Go" literals and bump-go-version-pr.sh#280's own
	// "chore/go-<target>" / "chore(go): bump Go toolchain to <target>"
	// text exactly (design.md's "Go delivery text unchanged" scenario).
	var buf bytes.Buffer
	if err := printDeliveryManifest(&buf, "go", "1.26.7"); err != nil {
		t.Fatalf("printDeliveryManifest: unexpected error: %v", err)
	}

	want := "branch=chore/go-1.26.7\n" +
		"commit=chore(go): bump Go toolchain to 1.26.7\n" +
		"pr_title=chore(go): bump Go toolchain to 1.26.7\n"
	if got := buf.String(); got != want {
		t.Fatalf("manifest = %q, want %q", got, want)
	}
}

func TestPrintDeliveryManifestMalformedTargetRefusesAndPrintsNothing(t *testing.T) {
	cases := []string{"1.2", "1.2.3 --base main", "1.2.3; rm -rf /", "latest", ""}

	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			var buf bytes.Buffer
			err := printDeliveryManifest(&buf, "go", target)
			if err == nil {
				t.Fatalf("expected target %q to be refused, got nil error", target)
			}
			if buf.Len() != 0 {
				t.Fatalf("expected nothing printed on refusal, got %q", buf.String())
			}
		})
	}
}

func TestPrintDeliveryManifestMalformedIDRefusesAndPrintsNothing(t *testing.T) {
	cases := []string{"Go", "go-lang", "go_lang", "go; rm -rf /", ""}

	for _, id := range cases {
		t.Run(id, func(t *testing.T) {
			var buf bytes.Buffer
			err := printDeliveryManifest(&buf, id, "1.26.7")
			if err == nil {
				t.Fatalf("expected id %q to be refused, got nil error", id)
			}
			if buf.Len() != 0 {
				t.Fatalf("expected nothing printed on refusal, got %q", buf.String())
			}
		})
	}
}

func TestPrintDeliveryManifestUnregisteredIDRefusesAndPrintsNothing(t *testing.T) {
	var buf bytes.Buffer
	err := printDeliveryManifest(&buf, "java", "1.26.7")
	if err == nil {
		t.Fatal("expected an unregistered toolchain id to be refused, got nil")
	}
	if buf.Len() != 0 {
		t.Fatalf("expected nothing printed on refusal, got %q", buf.String())
	}
}
