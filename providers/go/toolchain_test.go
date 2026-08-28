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
