package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixture writes content to root/rel, creating parent directories as
// needed, and returns the absolute path written.
func writeFixture(t *testing.T, root, rel, content string) string {
	t.Helper()

	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
	return full
}

// -- E1: missing file / directory / symlink Path -----------------------

func TestPlanE1MissingFileHardErrors(t *testing.T) {
	root := t.TempDir()
	site := Site{Path: "does/not/exist.go", Anchor: `v = "{{V}}"`, Occurrences: 1}

	edits, err := Plan(root, []Site{site}, "1.27.0")
	if err == nil {
		t.Fatal("expected a hard error for a missing file, got nil")
	}
	if !strings.Contains(err.Error(), "[E1]") {
		t.Fatalf("expected an E1-classified error, got: %v", err)
	}
	if edits != nil {
		t.Fatalf("expected no edits on error, got %v", edits)
	}
}

func TestPlanE1DirectoryPathHardErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	site := Site{Path: "docs", Anchor: `v = "{{V}}"`, Occurrences: 1}

	_, err := Plan(root, []Site{site}, "1.27.0")
	if err == nil {
		t.Fatal("expected a hard error for a directory Path, got nil")
	}
	if !strings.Contains(err.Error(), "[E1]") {
		t.Fatalf("expected an E1-classified error, got: %v", err)
	}
}

func TestPlanE1SymlinkPathHardErrors(t *testing.T) {
	root := t.TempDir()
	realFile := writeFixture(t, root, "real.go", `const v = "1.26.1"`)
	link := filepath.Join(root, "link.go")
	if err := os.Symlink(realFile, link); err != nil {
		t.Skipf("symlink not supported on this platform: %v", err)
	}
	site := Site{Path: "link.go", Anchor: `const v = "{{V}}"`, Occurrences: 1}

	_, err := Plan(root, []Site{site}, "1.27.0")
	if err == nil {
		t.Fatal("expected a hard error for a symlink Path, got nil")
	}
	if !strings.Contains(err.Error(), "[E1]") {
		t.Fatalf("expected an E1-classified error, got: %v", err)
	}
}

// -- E2: zero regex matches ----------------------------------------------

func TestPlanE2ZeroMatchesHardErrors(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "gobuilder.go", `const defaultVersion = "1.26.1" // anchor text renamed`)
	site := Site{Path: "gobuilder.go", Anchor: `const defaultGoVersion = "{{V}}"`, Occurrences: 1}

	edits, err := Plan(root, []Site{site}, "1.27.0")
	if err == nil {
		t.Fatal("expected a hard error when the anchor matches zero times, got nil")
	}
	if !strings.Contains(err.Error(), "[E2]") {
		t.Fatalf("expected an E2-classified error, got: %v", err)
	}
	if edits != nil {
		t.Fatalf("expected no edits on error, got %v", edits)
	}
}

// -- E3: match-count mismatch vs Occurrences ------------------------------

func TestPlanE3OccurrenceMismatchHardErrors(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "ci.yml", "GO_VERSION: '1.26.1'\nGO_VERSION: '1.26.1'\n")
	site := Site{Path: "ci.yml", Anchor: `GO_VERSION: '{{V}}'`, Occurrences: 1}

	_, err := Plan(root, []Site{site}, "1.27.0")
	if err == nil {
		t.Fatal("expected a hard error on match-count mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "[E3]") {
		t.Fatalf("expected an E3-classified error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "matched 2 time(s), expected 1") {
		t.Fatalf("expected the error to name the actual and expected counts, got: %v", err)
	}
}

// -- E4: malformed anchor template ---------------------------------------

func TestCompileE4MalformedAnchor(t *testing.T) {
	cases := []struct {
		name   string
		anchor string
	}{
		{"zero placeholders", `const defaultGoVersion = "1.26.1"`},
		{"two placeholders", `{{V}} through {{V}}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile(Site{Path: "gobuilder.go", Anchor: tc.anchor, Occurrences: 1})
			if err == nil {
				t.Fatal("expected a hard error for a malformed anchor, got nil")
			}
			if !strings.Contains(err.Error(), "[E4]") {
				t.Fatalf("expected an E4-classified error, got: %v", err)
			}
		})
	}
}

// -- E5: path safety -------------------------------------------------------

func TestCompileE5PathSafety(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"testdata segment", "providers/go/testdata/runtime/live/go.mod"},
		{"absolute path", "/etc/passwd"},
		{"escapes repo root", "../../etc/passwd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile(Site{Path: tc.path, Anchor: `v = "{{V}}"`, Occurrences: 1})
			if err == nil {
				t.Fatal("expected a hard error for an unsafe path, got nil")
			}
			if !strings.Contains(err.Error(), "[E5]") {
				t.Fatalf("expected an E5-classified error, got: %v", err)
			}
		})
	}
}

func TestPlanE5PathSafetyBeforeAnyFileRead(t *testing.T) {
	root := t.TempDir()
	site := Site{Path: "../outside.go", Anchor: `v = "{{V}}"`, Occurrences: 1}

	_, err := Plan(root, []Site{site}, "1.27.0")
	if err == nil {
		t.Fatal("expected a hard error for a path escaping the repo root, got nil")
	}
	if !strings.Contains(err.Error(), "[E5]") {
		t.Fatalf("expected an E5-classified error, got: %v", err)
	}
}

// -- E6: target version format validation ---------------------------------

func TestPlanE6TargetFormatRejectedBeforeAnySite(t *testing.T) {
	cases := []string{
		"1.2.3 --base main",
		"1.2",
		"1.2.3; rm -rf /",
		"latest",
		"",
	}

	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			root := t.TempDir()
			// A site that would succeed if Plan ever reached it, proving
			// E6 is checked strictly before any site is touched.
			writeFixture(t, root, "gobuilder.go", `const defaultGoVersion = "1.26.1"`)
			site := Site{Path: "gobuilder.go", Anchor: `const defaultGoVersion = "{{V}}"`, Occurrences: 1}

			edits, err := Plan(root, []Site{site}, target)
			if err == nil {
				t.Fatalf("expected target %q to be rejected, got nil error", target)
			}
			if !strings.Contains(err.Error(), "[E6]") {
				t.Fatalf("expected an E6-classified error, got: %v", err)
			}
			if edits != nil {
				t.Fatalf("expected no edits on E6, got %v", edits)
			}
		})
	}
}

// -- Aggregation: independent site failures are all named in one error ----

func TestPlanAggregatesAllSiteFailuresInOneError(t *testing.T) {
	root := t.TempDir()

	// Site A: E1, missing entirely.
	siteA := Site{Path: "missing.go", Anchor: `v = "{{V}}"`, Occurrences: 1}

	// Site B: E2, anchor no longer present.
	writeFixture(t, root, "moved.go", `const other = "value"`)
	siteB := Site{Path: "moved.go", Anchor: `const defaultGoVersion = "{{V}}"`, Occurrences: 1}

	// Site C: E3, occurrence count mismatch.
	writeFixture(t, root, "workflow.yml", "GO_VERSION: '1.26.1'\n")
	siteC := Site{Path: "workflow.yml", Anchor: `GO_VERSION: '{{V}}'`, Occurrences: 2}

	// A fourth, healthy site to prove it never gets applied either.
	writeFixture(t, root, "healthy.go", `const defaultGoVersion = "1.26.1"`)
	siteHealthy := Site{Path: "healthy.go", Anchor: `const defaultGoVersion = "{{V}}"`, Occurrences: 1}

	edits, err := Plan(root, []Site{siteA, siteB, siteC, siteHealthy}, "1.27.0")
	if err == nil {
		t.Fatal("expected an aggregated error, got nil")
	}
	if edits != nil {
		t.Fatalf("expected zero edits on any failure, got %v", edits)
	}

	for _, wantPath := range []string{"missing.go", "moved.go", "workflow.yml"} {
		if !strings.Contains(err.Error(), wantPath) {
			t.Errorf("aggregated error does not name failing path %q: %v", wantPath, err)
		}
	}
	for _, code := range []string{"[E1]", "[E2]", "[E3]"} {
		if !strings.Contains(err.Error(), code) {
			t.Errorf("aggregated error does not contain %s: %v", code, err)
		}
	}

	// The healthy site's file must be byte-identical: zero partial writes.
	got, readErr := os.ReadFile(filepath.Join(root, "healthy.go"))
	if readErr != nil {
		t.Fatalf("reading healthy.go: %v", readErr)
	}
	if string(got) != `const defaultGoVersion = "1.26.1"` {
		t.Fatalf("healthy.go was mutated despite the aggregated Plan failure: %q", got)
	}
}

// -- Success path: mixed current versions converge, bytes outside the ----
// -- matched span are untouched (golden compare) --------------------------

func TestPlanAndApplySuccessMixedVersionsConverge(t *testing.T) {
	root := t.TempDir()

	goBuilder := "package golang\n\nconst defaultGoVersion = \"1.26.1\"\n"
	ciWorkflow := "env:\n  GO_VERSION: '1.26.5'\n  CACHE_VERSION: 'v1'\n"
	docs := "Go `1.25.5` per `go.mod`.\nGo `1.25.5` per `.go-version`.\n"

	writeFixture(t, root, "gobuilder.go", goBuilder)
	writeFixture(t, root, "ci.yml", ciWorkflow)
	writeFixture(t, root, "PRD.md", docs)

	sitesList := []Site{
		{Path: "gobuilder.go", Anchor: `const defaultGoVersion = "{{V}}"`, Occurrences: 1},
		{Path: "ci.yml", Anchor: `GO_VERSION: '{{V}}'`, Occurrences: 1},
		{Path: "PRD.md", Anchor: "Go `{{V}}`", Occurrences: 2},
	}

	edits, err := Plan(root, sitesList, "1.27.0")
	if err != nil {
		t.Fatalf("Plan: unexpected error: %v", err)
	}
	if len(edits) != len(sitesList) {
		t.Fatalf("expected %d edits, got %d", len(sitesList), len(edits))
	}

	if err := Apply(root, edits); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	wantGoBuilder := "package golang\n\nconst defaultGoVersion = \"1.27.0\"\n"
	wantCiWorkflow := "env:\n  GO_VERSION: '1.27.0'\n  CACHE_VERSION: 'v1'\n"
	wantDocs := "Go `1.27.0` per `go.mod`.\nGo `1.27.0` per `.go-version`.\n"

	for rel, want := range map[string]string{
		"gobuilder.go": wantGoBuilder,
		"ci.yml":       wantCiWorkflow,
		"PRD.md":       wantDocs,
	} {
		got, readErr := os.ReadFile(filepath.Join(root, rel))
		if readErr != nil {
			t.Fatalf("reading %s: %v", rel, readErr)
		}
		if string(got) != want {
			t.Errorf("%s: full file bytes do not golden-match after Apply.\n got:  %q\n want: %q", rel, got, want)
		}
	}
}

func TestApplyDoesNotAlterUnrelatedFiles(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "target.go", `const defaultGoVersion = "1.26.1"`)
	untouched := writeFixture(t, root, "bystander.go", `const other = "1.26.1"`)

	sitesList := []Site{{Path: "target.go", Anchor: `const defaultGoVersion = "{{V}}"`, Occurrences: 1}}

	edits, err := Plan(root, sitesList, "1.27.0")
	if err != nil {
		t.Fatalf("Plan: unexpected error: %v", err)
	}
	if err := Apply(root, edits); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	got, err := os.ReadFile(untouched)
	if err != nil {
		t.Fatalf("reading bystander.go: %v", err)
	}
	if string(got) != `const other = "1.26.1"` {
		t.Fatalf("bystander.go was mutated by an Apply call that never planned it: %q", got)
	}
}

// TestApplyPreservesFileMode is the RED test for tasks.md 3.9 (Threat
// Matrix: documentation-like paths): Apply must never widen or narrow a
// registered site's existing file mode -- it reads the current mode via
// os.Stat and reuses it verbatim on write, never hardcoding 0o644 or
// setting +x on a file that wasn't already executable (design.md's own
// "Apply preserves mode, never sets +x" response to this boundary).
func TestApplyPreservesFileMode(t *testing.T) {
	root := t.TempDir()
	path := writeFixture(t, root, "hooks/pre-commit", `const defaultGoVersion = "1.26.1"`)
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod fixture to 0o755: %v", err)
	}

	site := Site{Path: "hooks/pre-commit", Anchor: `const defaultGoVersion = "{{V}}"`, Occurrences: 1}
	edits, err := Plan(root, []Site{site}, "1.27.0")
	if err != nil {
		t.Fatalf("Plan: unexpected error: %v", err)
	}
	if err := Apply(root, edits); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after Apply: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("Apply must preserve the pre-existing file mode: got %o, want %o", got, 0o755)
	}
}
