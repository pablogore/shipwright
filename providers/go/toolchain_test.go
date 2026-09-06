package golang

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// loadFixture reads a testdata/runtime/<name> fixture directory into a
// WorkspaceInput: go.work and .go-version at the fixture root (if present),
// and every go.mod found anywhere under the fixture root, keyed by its
// directory relative to the fixture root ("." for a go.mod at the fixture
// root itself). This mirrors what GoRuntimeInspector gathers via
// daggerkit.DaggerDirectory.Entries/File in production, without any Dagger
// dependency here (testing-tdd double-selection rule: no engine needed for
// pure-Go parsing logic).
func loadFixture(t *testing.T, name string) WorkspaceInput {
	t.Helper()

	root := filepath.Join("testdata", "runtime", name)
	input := WorkspaceInput{Modules: map[string][]byte{}}

	if data, err := os.ReadFile(filepath.Join(root, "go.work")); err == nil {
		input.GoWork = data
	} else if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("failed to read %s/go.work: %v", name, err)
	}

	if data, err := os.ReadFile(filepath.Join(root, ".go-version")); err == nil {
		input.GoVersion = data
	} else if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("failed to read %s/.go-version: %v", name, err)
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, filepath.Dir(path))
		if relErr != nil {
			return relErr
		}
		input.Modules[rel] = data
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk fixture %s: %v", name, err)
	}

	return input
}

// TestParseWorkspace_DetectConflicts is the TDD mass for design.md D-5's
// ambiguity rules (A1-A6), table-driven over providers/go/testdata/runtime/*
// fixtures — pure Go, no Dagger, no engine.
func TestParseWorkspace_DetectConflicts(t *testing.T) {
	tests := []struct {
		name           string
		fixture        string
		targetVersion  string
		allowDowngrade bool
		wantCode       string // "" means no conflict expected
	}{
		{
			name:     "single module, no go.work, no .go-version: no conflict",
			fixture:  "single-module",
			wantCode: "",
		},
		{
			name:     "three-module workspace, all sources agree: no conflict",
			fixture:  "workspace-3-modules",
			wantCode: "",
		},
		{
			name:     "two modules declare different go directives",
			fixture:  "divergent-go",
			wantCode: CodeA1,
		},
		{
			name:     "two modules declare different toolchain directives",
			fixture:  "divergent-toolchain",
			wantCode: CodeA2,
		},
		{
			name:     "go.work's go directive disagrees with the unanimous module go",
			fixture:  "work-go-mismatch",
			wantCode: CodeA3,
		},
		{
			name:     ".go-version disagrees with the unanimous module go (mirrors this repo's real drift)",
			fixture:  "goversion-file-mismatch",
			wantCode: CodeA3,
		},
		{
			name:          "target version below current, downgrade not allowed",
			fixture:       "downgrade",
			targetVersion: "1.20.0",
			wantCode:      CodeA4,
		},
		{
			name:           "target version below current, downgrade explicitly allowed",
			fixture:        "downgrade",
			targetVersion:  "1.20.0",
			allowDowngrade: true,
			wantCode:       "",
		},
		{
			name:     "malformed go directive fails modfile's own validation",
			fixture:  "malformed",
			wantCode: CodeA5,
		},
		{
			name:     "this repository's own live tier-1 sources: .go-version vs go.mod/go.work mismatch",
			fixture:  "live-repo-drift",
			wantCode: CodeA3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := loadFixture(t, tt.fixture)

			ws, err := parseWorkspace(input)

			var ambiguous *AmbiguousToolchainError
			if err != nil {
				if !errors.As(err, &ambiguous) {
					t.Fatalf("parseWorkspace(%s) error = %v, want an *AmbiguousToolchainError", tt.fixture, err)
				}
			} else {
				ambiguous = detectConflicts(ws, ConflictOptions{
					TargetVersion:  tt.targetVersion,
					AllowDowngrade: tt.allowDowngrade,
				})
			}

			if tt.wantCode == "" {
				if ambiguous != nil {
					t.Fatalf("fixture %s: got conflict %v, want none", tt.fixture, ambiguous)
				}
				return
			}

			if ambiguous == nil {
				t.Fatalf("fixture %s: got no conflict, want code %s", tt.fixture, tt.wantCode)
			}
			if ambiguous.Code != tt.wantCode {
				t.Fatalf("fixture %s: got code %s, want %s (%v)", tt.fixture, ambiguous.Code, tt.wantCode, ambiguous)
			}
		})
	}
}

// TestDetectConflicts_A6_NoWorkspaceSources guards A6 directly: a Workspace
// with neither a go.work nor any go.mod is ambiguous by construction, before
// detectConflicts even looks at directive values.
func TestDetectConflicts_A6_NoWorkspaceSources(t *testing.T) {
	ws := &Workspace{}

	got := detectConflicts(ws, ConflictOptions{})

	if got == nil {
		t.Fatal("detectConflicts() with no go.work and no go.mod = nil, want A6")
	}
	if got.Code != CodeA6 {
		t.Fatalf("detectConflicts() code = %s, want %s", got.Code, CodeA6)
	}
}

// TestMutateGoMod is the TDD mass for design.md D-7/D-9's mutation step,
// table-driven over testdata/runtime/* fixtures — pure Go, no Dagger.
func TestMutateGoMod(t *testing.T) {
	tests := []struct {
		name          string
		fixture       string
		targetVersion string
		wantErrCode   string // "" means no error expected
	}{
		{
			name:          "single module: go directive updated",
			fixture:       "single-module",
			targetVersion: "1.27.0",
		},
		{
			name:          "downgrade fixture: mutateGoMod itself does not enforce A4 (detectConflicts's job, called by the caller first)",
			fixture:       "downgrade",
			targetVersion: "1.20.0",
		},
		{
			name:          "malformed go.mod: A5",
			fixture:       "malformed",
			targetVersion: "1.27.0",
			wantErrCode:   CodeA5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := loadFixture(t, tt.fixture)
			modBytes := input.Modules["."]

			got, err := mutateGoMod(modBytes, tt.targetVersion)

			if tt.wantErrCode != "" {
				var ambiguous *AmbiguousToolchainError
				if !errors.As(err, &ambiguous) {
					t.Fatalf("mutateGoMod(%s) error = %v, want an *AmbiguousToolchainError", tt.fixture, err)
				}
				if ambiguous.Code != tt.wantErrCode {
					t.Fatalf("mutateGoMod(%s) code = %s, want %s", tt.fixture, ambiguous.Code, tt.wantErrCode)
				}
				if got != nil {
					t.Fatalf("mutateGoMod(%s) bytes = %q, want nil on error", tt.fixture, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("mutateGoMod(%s) error = %v, want nil", tt.fixture, err)
			}

			mutated, parseErr := parseWorkspace(WorkspaceInput{Modules: map[string][]byte{".": got}})
			if parseErr != nil {
				t.Fatalf("mutateGoMod(%s) produced unparseable bytes: %v\n%s", tt.fixture, parseErr, got)
			}
			if mutated.Modules[0].Go != tt.targetVersion {
				t.Fatalf("mutateGoMod(%s) go directive = %s, want %s", tt.fixture, mutated.Modules[0].Go, tt.targetVersion)
			}
		})
	}
}

// TestMutateGoMod_UpdatesExistingToolchainDirective proves the toolchain
// directive is updated in place when the original go.mod already declares
// one — discovery-driven: mutateGoMod never introduces a toolchain
// directive that was not there before.
func TestMutateGoMod_UpdatesExistingToolchainDirective(t *testing.T) {
	original := []byte("module example.com/fixture/toolchain\n\ngo 1.26.7\n\ntoolchain go1.26.7\n")

	got, err := mutateGoMod(original, "1.27.0")
	if err != nil {
		t.Fatalf("mutateGoMod() error = %v, want nil", err)
	}

	mf, err := parseWorkspace(WorkspaceInput{Modules: map[string][]byte{".": got}})
	if err != nil {
		t.Fatalf("mutateGoMod() produced unparseable bytes: %v\n%s", err, got)
	}
	if mf.Modules[0].Toolchain != "go1.27.0" {
		t.Fatalf("mutateGoMod() toolchain = %s, want go1.27.0", mf.Modules[0].Toolchain)
	}
}

// TestMutateGoMod_ThreatMatrix_MaliciousTargetVersion is the RED test
// design.md's Threat Matrix row "Command construction from config values"
// requires: targetVersion is validated against modfile.GoVersionRE before
// any write, rejecting strings shaped like shell injection or flag
// injection attempts before they ever reach "golang:"+v or an argv slice.
func TestMutateGoMod_ThreatMatrix_MaliciousTargetVersion(t *testing.T) {
	malicious := []string{
		"1.26.7; rm -rf /",
		"--flag",
		"",
		"$(whoami)",
	}

	modBytes := loadFixture(t, "single-module").Modules["."]

	for _, target := range malicious {
		t.Run(target, func(t *testing.T) {
			got, err := mutateGoMod(modBytes, target)

			var ambiguous *AmbiguousToolchainError
			if !errors.As(err, &ambiguous) {
				t.Fatalf("mutateGoMod(%q) error = %v, want an *AmbiguousToolchainError", target, err)
			}
			if ambiguous.Code != CodeA5 {
				t.Fatalf("mutateGoMod(%q) code = %s, want %s", target, ambiguous.Code, CodeA5)
			}
			if got != nil {
				t.Fatalf("mutateGoMod(%q) bytes = %q, want nil on a rejected target", target, got)
			}
		})
	}
}

// TestMutateGoWork proves go.work's own go directive is updated the same
// way mutateGoMod updates go.mod's — pure Go, no wiring into Upgrade yet
// (tasks.md Phase 3).
func TestMutateGoWork(t *testing.T) {
	input := loadFixture(t, "workspace-3-modules")

	got, err := mutateGoWork(input.GoWork, "1.27.0")
	if err != nil {
		t.Fatalf("mutateGoWork() error = %v, want nil", err)
	}

	mutated, err := parseWorkspace(WorkspaceInput{GoWork: got})
	if err != nil {
		t.Fatalf("mutateGoWork() produced unparseable bytes: %v\n%s", err, got)
	}
	if mutated.GoWorkGo != "1.27.0" {
		t.Fatalf("mutateGoWork() go directive = %s, want 1.27.0", mutated.GoWorkGo)
	}
}

// TestMutateGoWork_MalformedFailsClosed mirrors TestMutateGoMod's A5 case
// for go.work.
func TestMutateGoWork_MalformedFailsClosed(t *testing.T) {
	got, err := mutateGoWork([]byte("go notaversion\n"), "1.27.0")

	var ambiguous *AmbiguousToolchainError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("mutateGoWork() error = %v, want an *AmbiguousToolchainError", err)
	}
	if ambiguous.Code != CodeA5 {
		t.Fatalf("mutateGoWork() code = %s, want %s", ambiguous.Code, CodeA5)
	}
	if got != nil {
		t.Fatalf("mutateGoWork() bytes = %q, want nil on error", got)
	}
}

// TestMutateGoWork_ThreatMatrix_MaliciousTargetVersion mirrors
// TestMutateGoMod_ThreatMatrix_MaliciousTargetVersion for go.work.
func TestMutateGoWork_ThreatMatrix_MaliciousTargetVersion(t *testing.T) {
	workBytes := loadFixture(t, "workspace-3-modules").GoWork

	for _, target := range []string{"1.26.7; rm -rf /", "--flag"} {
		t.Run(target, func(t *testing.T) {
			got, err := mutateGoWork(workBytes, target)

			var ambiguous *AmbiguousToolchainError
			if !errors.As(err, &ambiguous) {
				t.Fatalf("mutateGoWork(%q) error = %v, want an *AmbiguousToolchainError", target, err)
			}
			if ambiguous.Code != CodeA5 {
				t.Fatalf("mutateGoWork(%q) code = %s, want %s", target, ambiguous.Code, CodeA5)
			}
			if got != nil {
				t.Fatalf("mutateGoWork(%q) bytes = %q, want nil on a rejected target", target, got)
			}
		})
	}
}

// TestMutateGoVersion proves the trivial .go-version mutation: a single
// line, exactly targetVersion.
func TestMutateGoVersion(t *testing.T) {
	got := mutateGoVersion("1.27.0")

	if string(got) != "1.27.0\n" {
		t.Fatalf("mutateGoVersion() = %q, want %q", got, "1.27.0\n")
	}
}

// TestDetectConflicts_WorkspacePreMutation proves detectConflicts's
// existing A1-A3 rules (built in Phase 1) already generalize correctly to
// the multi-module go.work case when invoked exactly the way
// GoRuntimeUpgrader.Upgrade invokes it — with a non-empty TargetVersion —
// so Upgrade's pre-mutation abort path (tasks.md 3.1) needs no new
// conflict-detection logic, only to stop short-circuiting before this
// check runs for a go.work workspace. Pure Go, no Dagger.
func TestDetectConflicts_WorkspacePreMutation(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantCode string // "" means no conflict expected
	}{
		{
			name:     "three-module workspace, all sources agree: no conflict",
			fixture:  "workspace-3-modules",
			wantCode: "",
		},
		{
			name:     "two modules declare different go directives",
			fixture:  "divergent-go",
			wantCode: CodeA1,
		},
		{
			name:     "two modules declare different toolchain directives",
			fixture:  "divergent-toolchain",
			wantCode: CodeA2,
		},
		{
			name:     "go.work's go directive disagrees with the unanimous module go",
			fixture:  "work-go-mismatch",
			wantCode: CodeA3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := loadFixture(t, tt.fixture)

			ws, err := parseWorkspace(input)
			if err != nil {
				t.Fatalf("parseWorkspace(%s) error = %v, want nil", tt.fixture, err)
			}

			got := detectConflicts(ws, ConflictOptions{TargetVersion: "1.27.0"})

			if tt.wantCode == "" {
				if got != nil {
					t.Fatalf("fixture %s: got conflict %v, want none", tt.fixture, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("fixture %s: got no conflict, want code %s", tt.fixture, tt.wantCode)
			}
			if got.Code != tt.wantCode {
				t.Fatalf("fixture %s: got code %s, want %s (%v)", tt.fixture, got.Code, tt.wantCode, got)
			}
		})
	}
}

// TestValidateModulePaths is the pure-Go path-escape guard tasks.md
// 3.2/3.3 requires: design.md's Threat Matrix "Path traversal via go.work
// use" row — reject absolute paths, and any path escaping the workspace
// root after path.Clean.
func TestValidateModulePaths(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		wantCode string // "" means no conflict expected
	}{
		{
			name:     "within workspace: no conflict",
			paths:    []string{"modA", "modB/nested"},
			wantCode: "",
		},
		{
			name:     "workspace root itself is fine",
			paths:    []string{"."},
			wantCode: "",
		},
		{
			name:     "escapes workspace root via ../../etc (path-escape fixture shape)",
			paths:    []string{"../../etc"},
			wantCode: CodeA7,
		},
		{
			name:     "absolute path",
			paths:    []string{"/etc/passwd"},
			wantCode: CodeA7,
		},
		{
			name:     "one valid, one escaping: whole call rejected",
			paths:    []string{"modA", "../evil"},
			wantCode: CodeA7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateModulePaths(tt.paths)

			if tt.wantCode == "" {
				if got != nil {
					t.Fatalf("validateModulePaths(%v) = %v, want nil", tt.paths, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("validateModulePaths(%v) = nil, want code %s", tt.paths, tt.wantCode)
			}
			if got.Code != tt.wantCode {
				t.Fatalf("validateModulePaths(%v) code = %s, want %s", tt.paths, got.Code, tt.wantCode)
			}
		})
	}
}
