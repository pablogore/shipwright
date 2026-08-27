// Package workspaceguard guards design.md D1's go.work isolation decision:
// go.work declares which modules participate in the workspace and how
// dependency resolution works, but it does NOT make `go build ./...` or
// `go test -race ./...` from root traverse nested modules — each module
// must be tested/built explicitly (e.g., `cd providers/go && go test ./...`).
// This package enforces which modules may appear in the workspace via an
// explicit, automated allowlist rather than a comment. It parses go.work
// exactly once, the same way internal/daggerpin parses go.mod/dagger.json,
// and never requires a running Dagger engine.
package workspaceguard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

// UsePaths reads path (a go.work file) and returns the literal `use`
// directive paths it declares, in file order and exactly as written. It
// does not clean or normalize them, because the allowlist comparison
// (design.md D1) is against the literal committed content.
func UsePaths(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("workspaceguard: read %s: %w", path, err)
	}

	f, err := modfile.ParseWork(path, raw, nil)
	if err != nil {
		return nil, fmt.Errorf("workspaceguard: parse %s: %w", path, err)
	}

	uses := make([]string, 0, len(f.Use))
	for _, u := range f.Use {
		uses = append(uses, u.Path)
	}

	return uses, nil
}

// ReplaceDirectives reads path (a go.mod file) and returns the set of
// replace directives it declares, each as "old => new" (with version or
// local path). An empty slice means no replace directives exist.
func ReplaceDirectives(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("workspaceguard: read %s: %w", path, err)
	}

	f, err := modfile.Parse(path, raw, nil)
	if err != nil {
		return nil, fmt.Errorf("workspaceguard: parse %s: %w", path, err)
	}

	var reps []string
	for _, r := range f.Replace {
		if r.New.Version != "" {
			reps = append(reps, r.Old.String()+" => "+r.New.Path+" "+r.New.Version)
		} else {
			reps = append(reps, r.Old.String()+" => "+r.New.Path)
		}
	}

	return reps, nil
}

// DaggerUse reports the first `use` path (if any) whose cleaned path has a
// segment equal to ".dagger" or "dagger" -- the isolation design.md's D-B
// established for the .dagger/ Dagger-engine module, which D1 must never
// collapse by adding it to the workspace.
func DaggerUse(usePaths []string) (string, bool) {
	for _, use := range usePaths {
		cleaned := filepath.ToSlash(filepath.Clean(use))
		for _, seg := range strings.Split(cleaned, "/") {
			if seg == ".dagger" || seg == "dagger" {
				return use, true
			}
		}
	}

	return "", false
}
