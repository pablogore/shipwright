package providers

import "testing"

// Unit tests for changelog.go's unexported pure text-processing helpers.
// In-package (not providers_test) for the same reason as
// internal_test.go's stringField/floatField tests: these are internal
// convenience helpers, not part of ChangelogRunner's public contract.
//
// Assertions mirror the former internal/app.ChangelogStepHandler's own
// test suite (internal/app/changelog_step_handler_test.go, now deleted
// along with the handler it tested) as closely as the renamed unexported
// symbols allow — the text-processing logic itself was ported unchanged.

func TestClassifyChangelogCommits_Breaking(t *testing.T) {
	c := classifyChangelogCommits([]string{
		"feat!: drop support for legacy config format",
		"chore: BREAKING CHANGE tokens are no longer accepted",
	})

	if len(c.Added) != 0 {
		t.Fatalf("Added = %v, want empty", c.Added)
	}
	if len(c.Fixed) != 0 {
		t.Fatalf("Fixed = %v, want empty", c.Fixed)
	}
	if len(c.Changed) != 2 {
		t.Fatalf("Changed = %v, want 2 entries", c.Changed)
	}
	if want := "**BREAKING:**"; !contains(c.Changed[0], want) || !contains(c.Changed[0], "drop support for legacy config format") {
		t.Fatalf("Changed[0] = %q, want it to contain %q and the description", c.Changed[0], want)
	}
	if !contains(c.Changed[1], "**BREAKING:**") {
		t.Fatalf("Changed[1] = %q, want it to contain **BREAKING:**", c.Changed[1])
	}
}

func TestClassifyChangelogCommits_Feat(t *testing.T) {
	c := classifyChangelogCommits([]string{
		"feat: add changelog runner",
		"feat(cli): add --dry-run flag",
	})

	if len(c.Added) != 2 {
		t.Fatalf("Added = %v, want 2 entries", c.Added)
	}
	if c.Added[0] != "add changelog runner" {
		t.Fatalf("Added[0] = %q, want %q", c.Added[0], "add changelog runner")
	}
	if c.Added[1] != "add --dry-run flag" {
		t.Fatalf("Added[1] = %q, want %q", c.Added[1], "add --dry-run flag")
	}
	if len(c.Changed) != 0 || len(c.Fixed) != 0 {
		t.Fatalf("Changed/Fixed = %v/%v, want both empty", c.Changed, c.Fixed)
	}
}

func TestClassifyChangelogCommits_Fix(t *testing.T) {
	c := classifyChangelogCommits([]string{
		"fix: correct nil pointer in provider registry",
		"fix(config): handle missing go.mod gracefully",
	})

	if len(c.Fixed) != 2 {
		t.Fatalf("Fixed = %v, want 2 entries", c.Fixed)
	}
	if c.Fixed[0] != "correct nil pointer in provider registry" {
		t.Fatalf("Fixed[0] = %q, want %q", c.Fixed[0], "correct nil pointer in provider registry")
	}
	if c.Fixed[1] != "handle missing go.mod gracefully" {
		t.Fatalf("Fixed[1] = %q, want %q", c.Fixed[1], "handle missing go.mod gracefully")
	}
	if len(c.Added) != 0 || len(c.Changed) != 0 {
		t.Fatalf("Added/Changed = %v/%v, want both empty", c.Added, c.Changed)
	}
}

func TestClassifyChangelogCommits_Other(t *testing.T) {
	c := classifyChangelogCommits([]string{
		"docs: update README",
		"chore: bump dependencies",
		"refactor internal helpers",
	})

	if len(c.Changed) != 3 {
		t.Fatalf("Changed = %v, want 3 entries", c.Changed)
	}
	if len(c.Added) != 0 || len(c.Fixed) != 0 {
		t.Fatalf("Added/Fixed = %v/%v, want both empty", c.Added, c.Fixed)
	}
}

func TestClassifyChangelogCommits_IgnoresBlankSubjects(t *testing.T) {
	c := classifyChangelogCommits([]string{"", "   ", "feat: real change"})

	if len(c.Added) != 1 || c.Added[0] != "real change" {
		t.Fatalf("Added = %v, want [\"real change\"]", c.Added)
	}
}

func TestClassifyChangelogCommits_Empty(t *testing.T) {
	c := classifyChangelogCommits(nil)

	if len(c.Added) != 0 || len(c.Changed) != 0 || len(c.Fixed) != 0 {
		t.Fatalf("classifyChangelogCommits(nil) = %+v, want all empty", c)
	}
}

func TestChangelogCommitSubjects(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty output", in: "", want: nil},
		{name: "whitespace only", in: "   \n  ", want: nil},
		{name: "single line", in: "feat: first", want: []string{"feat: first"}},
		{name: "multiple lines", in: "feat: first\nfix: second", want: []string{"feat: first", "fix: second"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := changelogCommitSubjects(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("changelogCommitSubjects(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("changelogCommitSubjects(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildChangelogSection_AllCategories(t *testing.T) {
	c := changelogCommitClassification{
		Added:   []string{"feat: a"},
		Changed: []string{"**BREAKING:** feat!: b", "docs: c"},
		Fixed:   []string{"fix: d"},
	}

	section := buildChangelogSection(c)
	want := "### Added\n- feat: a\n\n### Changed\n- **BREAKING:** feat!: b\n- docs: c\n\n### Fixed\n- fix: d"
	if section != want {
		t.Fatalf("buildChangelogSection() = %q, want %q", section, want)
	}
}

func TestBuildChangelogSection_OnlyPopulatedCategories(t *testing.T) {
	c := changelogCommitClassification{Fixed: []string{"fix: only this"}}

	section := buildChangelogSection(c)
	want := "### Fixed\n- fix: only this"
	if section != want {
		t.Fatalf("buildChangelogSection() = %q, want %q", section, want)
	}
}

func TestBuildChangelogSection_Empty(t *testing.T) {
	section := buildChangelogSection(changelogCommitClassification{})
	if section != "" {
		t.Fatalf("buildChangelogSection(empty) = %q, want empty", section)
	}
}

func TestPrependUnreleasedSection_ExistingHeading(t *testing.T) {
	existing := "# Changelog\n\n## [Unreleased]\n\n### Added\n- old entry\n\n## [0.0.1] - 2024-01-10\n"
	section := "### Added\n- new entry"

	updated := prependUnreleasedSection(existing, section)

	if !contains(updated, "## [Unreleased]\n\n### Added\n- new entry") {
		t.Fatalf("prependUnreleasedSection() = %q, want new entry directly under the heading", updated)
	}
	if !contains(updated, "- old entry") {
		t.Fatalf("prependUnreleasedSection() = %q, want pre-existing entry preserved", updated)
	}
	if !contains(updated, "## [0.0.1] - 2024-01-10") {
		t.Fatalf("prependUnreleasedSection() = %q, want prior release section preserved", updated)
	}
}

func TestPrependUnreleasedSection_MissingHeading(t *testing.T) {
	existing := "# Changelog\n\nSome intro text.\n"
	section := "### Fixed\n- fixed thing"

	updated := prependUnreleasedSection(existing, section)

	if !contains(updated, "## [Unreleased]") {
		t.Fatalf("prependUnreleasedSection() = %q, want a created Unreleased heading", updated)
	}
	if !contains(updated, "### Fixed\n- fixed thing") {
		t.Fatalf("prependUnreleasedSection() = %q, want the new section", updated)
	}
	if !contains(updated, "Some intro text.") {
		t.Fatalf("prependUnreleasedSection() = %q, want pre-existing intro text preserved", updated)
	}
}

func TestPrependUnreleasedSection_EmptyExisting(t *testing.T) {
	updated := prependUnreleasedSection("", "### Added\n- first entry")
	want := "## [Unreleased]\n\n### Added\n- first entry\n"
	if updated != want {
		t.Fatalf("prependUnreleasedSection(\"\", ...) = %q, want %q", updated, want)
	}
}

func TestPrependUnreleasedSection_EmptySectionIsNoop(t *testing.T) {
	existing := "# Changelog\n\n## [Unreleased]\n\n### Added\n- old entry\n"
	updated := prependUnreleasedSection(existing, "")
	if updated != existing {
		t.Fatalf("prependUnreleasedSection(existing, \"\") = %q, want unchanged %q", updated, existing)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
