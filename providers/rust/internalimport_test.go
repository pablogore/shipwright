// Package rust_test enforces design.md D5's boundary rule, extended to
// providers/rust: the public contract (pkg/shipwright) must remain
// sufficient, on its own, to implement every shipped capability from a
// separate Go module. The Go compiler does not enforce this here --
// internal/** visibility is path-prefix based, not module based, so
// github.com/pablogore/shipwright/providers/rust sitting under
// github.com/pablogore/shipwright/ means an
// `import ".../internal/workflow/interp"` from this module compiles fine
// even across the module boundary. This test is the only thing that
// enforces the property.
package rust_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestNoImportReachesIntoInternal mirrors providers/go's D5 guard: no
// import in providers/rust, production or test, may reach into any
// internal/** package.
func TestNoImportReachesIntoInternal(t *testing.T) {
	dir := providersRustDir(t)

	fset := token.NewFileSet()
	// parsePackageDir (parsepkg_test.go) replaces the deprecated
	// parser.ParseDir/ast.Package pairing (SA1019).
	pkgs, err := parsePackageDir(fset, dir, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("failed to parse package directory %s: %v", dir, err)
	}

	if _, ok := pkgs["rust"]; !ok {
		t.Fatalf(
			"no %q package found in %s -- expected the providers/rust module's "+
				"production package to exist", "rust", dir,
		)
	}

	for pkgName, pkg := range pkgs {
		for filename, file := range pkg.Files {
			for _, imp := range file.Imports {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Fatalf("failed to unquote import %s in %s: %v", imp.Path.Value, filename, err)
				}
				if hasInternalSegment(path) {
					t.Errorf(
						"package %q, file %s imports %q, which reaches into an internal/** "+
							"package -- design.md D5 requires providers/rust to build against "+
							"pkg/shipwright, dagger.io/dagger, and stdlib alone",
						pkgName, filepath.Base(filename), path,
					)
				}
			}
		}
	}
}

// providersRustDir locates this module's own directory via runtime.Caller.
func providersRustDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test source location via runtime.Caller")
	}
	return filepath.Dir(thisFile)
}

func hasInternalSegment(importPath string) bool {
	for _, seg := range strings.Split(importPath, "/") {
		if seg == "internal" {
			return true
		}
	}
	return false
}
