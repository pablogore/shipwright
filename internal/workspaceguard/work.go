// Package workspaceguard guards design.md D1's go.work isolation decision:
// the root module's workspace membership decides which packages
// `go build ./...` and `go test -race ./...` traverse, so this package
// makes that membership subject to an explicit, automated allowlist rather
// than a comment. It parses go.work exactly once, the same way
// internal/daggerpin parses go.mod/dagger.json, and never requires a
// running Dagger engine.
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
