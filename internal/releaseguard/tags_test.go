// Package releaseguard guards design.md D6's release-automation decision:
// each tag-reacting per-provider release workflow (providers/go, mirrored
// for providers/rust) must never collide with the root release line's v*
// tag namespace, and the root release workflow's `git describe --tags`
// calls must never resolve a provider's vX.Y.Z tag as if it were a root
// version (design.md D6's "blast radius on the root release line"). It
// parses the workflow YAMLs as text/YAML, the same way internal/daggerpin
// and internal/workspaceguard guard other design decisions, and never
// requires a running Dagger engine or a real GitHub Actions run.
package releaseguard

import (
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed to resolve this test file's path")
	}
	// this file lives at <repoRoot>/internal/releaseguard/tags_test.go
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func releaseWorkflowPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), ".github", "workflows", "release.yml")
}

func providerWorkflowPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), ".github", "workflows", "release-provider-go.yml")
}

func rustProviderWorkflowPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), ".github", "workflows", "release-provider-rust.yml")
}

// (a) no `git describe --tags` call in release.yml lacks --match.
func TestReleaseYAML_DescribeTagsCallsAreFiltered(t *testing.T) {
	unfiltered, err := UnfilteredDescribeCalls(releaseWorkflowPath(t))
	if err != nil {
		t.Fatalf("UnfilteredDescribeCalls() error = %v, want nil", err)
	}
	if len(unfiltered) != 0 {
		t.Fatalf("release.yml has %d `git describe --tags` call(s) without --match (design.md D6): %v", len(unfiltered), unfiltered)
	}
}

// (b) release.yml's tag globs don't match providers/go/v0.1.0 or
// providers/rust/v0.1.0.
func TestReleaseYAML_TagGlobsExcludeProviderNamespace(t *testing.T) {
	globs, err := PushTagGlobs(releaseWorkflowPath(t))
	if err != nil {
		t.Fatalf("PushTagGlobs(release.yml) error = %v, want nil", err)
	}
	if len(globs) == 0 {
		t.Fatal("release.yml declares no on.push.tags globs, want at least one (e.g. v*)")
	}
	for _, providerTag := range []string{"providers/go/v0.1.0", "providers/rust/v0.1.0"} {
		for _, g := range globs {
			if GlobMatches(g, providerTag) {
				t.Fatalf("release.yml's tag glob %q matches %q, want disjoint from the provider namespace (design.md D6)", g, providerTag)
			}
		}
	}
}

// (c) release-provider-go.yml's globs don't match v1.2.3.
func TestProviderWorkflow_TagGlobsExcludeRootNamespace(t *testing.T) {
	globs, err := PushTagGlobs(providerWorkflowPath(t))
	if err != nil {
		t.Fatalf("PushTagGlobs(release-provider-go.yml) error = %v, want nil", err)
	}
	if len(globs) == 0 {
		t.Fatal("release-provider-go.yml declares no on.push.tags globs, want at least one (e.g. providers/go/v*)")
	}
	for _, g := range globs {
		if GlobMatches(g, "v1.2.3") {
			t.Fatalf("release-provider-go.yml's tag glob %q matches %q, want disjoint from the root v* namespace (design.md D6)", g, "v1.2.3")
		}
	}
}

// (d) the shape regex extracted from release-provider-go.yml accepts a
// valid providers/go tag and rejects the shapes design.md D6 names.
func TestProviderWorkflow_ShapeRegex(t *testing.T) {
	pattern, err := ExtractShapeRegex(providerWorkflowPath(t))
	if err != nil {
		t.Fatalf("ExtractShapeRegex() error = %v, want nil", err)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("regexp.Compile(%q) error = %v, want nil", pattern, err)
	}

	tests := []struct {
		name string
		tag  string
		want bool
	}{
		{name: "valid v0.1.0", tag: "providers/go/v0.1.0", want: true},
		{name: "reject major >= 2 (module-path /vN rule)", tag: "providers/go/v2.0.0", want: false},
		{name: "reject leading zero", tag: "providers/go/v01.0.0", want: false},
		{name: "reject missing providers/go prefix", tag: "v0.1.0", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := re.MatchString(tt.tag); got != tt.want {
				t.Fatalf("regex %q MatchString(%q) = %v, want %v", pattern, tt.tag, got, tt.want)
			}
		})
	}
}

// (c) release-provider-rust.yml's globs don't match v1.2.3 -- mirrors
// TestProviderWorkflow_TagGlobsExcludeRootNamespace for the Rust provider
// (design.md D6, applied per-provider).
func TestRustProviderWorkflow_TagGlobsExcludeRootNamespace(t *testing.T) {
	globs, err := PushTagGlobs(rustProviderWorkflowPath(t))
	if err != nil {
		t.Fatalf("PushTagGlobs(release-provider-rust.yml) error = %v, want nil", err)
	}
	if len(globs) == 0 {
		t.Fatal("release-provider-rust.yml declares no on.push.tags globs, want at least one (e.g. providers/rust/v*)")
	}
	for _, g := range globs {
		if GlobMatches(g, "v1.2.3") {
			t.Fatalf("release-provider-rust.yml's tag glob %q matches %q, want disjoint from the root v* namespace (design.md D6)", g, "v1.2.3")
		}
	}
}

// (d) the shape regex extracted from release-provider-rust.yml accepts a
// valid providers/rust tag and rejects the shapes design.md D6 names --
// mirrors TestProviderWorkflow_ShapeRegex for the Rust provider.
func TestRustProviderWorkflow_ShapeRegex(t *testing.T) {
	pattern, err := ExtractShapeRegex(rustProviderWorkflowPath(t))
	if err != nil {
		t.Fatalf("ExtractShapeRegex() error = %v, want nil", err)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("regexp.Compile(%q) error = %v, want nil", pattern, err)
	}

	tests := []struct {
		name string
		tag  string
		want bool
	}{
		{name: "valid v0.1.0", tag: "providers/rust/v0.1.0", want: true},
		{name: "reject major >= 2 (module-path /vN rule)", tag: "providers/rust/v2.0.0", want: false},
		{name: "reject leading zero", tag: "providers/rust/v01.0.0", want: false},
		{name: "reject missing providers/rust prefix", tag: "v0.1.0", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := re.MatchString(tt.tag); got != tt.want {
				t.Fatalf("regex %q MatchString(%q) = %v, want %v", pattern, tt.tag, got, tt.want)
			}
		})
	}
}
