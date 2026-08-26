package golang_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// forbiddenBundleSubstrings enumerates naming patterns design.md D-F
// explicitly rejects for providers/go (formerly internal/capabilities): an
// exported identifier that names a stack bundle instead of a single,
// standalone capability implementation (e.g. GoService, GoStack). Matching
// is case-insensitive.
var forbiddenBundleSubstrings = []string{
	"goservice",
	"gostack",
	"stack",
	"bundle",
	"preset",
}

// TestNoExportedIdentifierNamesAStackBundle is the naming golden test
// required by tasks.md 3.1 (originally in internal/capabilities, moved
// here unchanged by the providers/go extraction): no exported identifier
// in this package may name a stack bundle. Every implementation must
// describe what it does (GoBuilder, ContainerPublisher, ...), never a
// bundled stack identity — design.md D-F rejects both a preset registry
// keyed by a stack name and a `goservice/` subdirectory, because the path
// itself would be a bundling identity. This test guards the
// exported-identifier half of that rule.
func TestNoExportedIdentifierNamesAStackBundle(t *testing.T) {
	pkg := parseGolangPackage(t)

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			for _, name := range exportedDeclNames(decl) {
				assertNotBundleName(t, name)
			}
		}
	}
}

func assertNotBundleName(t *testing.T, name string) {
	t.Helper()
	lower := strings.ToLower(name)
	for _, forbidden := range forbiddenBundleSubstrings {
		if strings.Contains(lower, forbidden) {
			t.Errorf(
				"exported identifier %q names a stack bundle (contains %q) -- design.md D-F requires standalone, "+
					"non-bundled capability implementations, never a preset/stack identity",
				name, forbidden,
			)
		}
	}
}

// parseGolangPackage parses the non-test production source of providers/go
// (package golang). Deliberately fails closed (t.Fatal) if the package does
// not exist yet — that was the expected RED state before the extraction's
// tasks.md 2.1-2.9 added production files here.
func parseGolangPackage(t *testing.T) *ast.Package {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test source location via runtime.Caller")
	}
	pkgDir := filepath.Dir(thisFile)

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, nonTestGoFile, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse package directory %s: %v", pkgDir, err)
	}

	pkg, ok := pkgs["golang"]
	if !ok {
		t.Fatalf(
			"package %q not found while parsing %s -- providers/go must exist with at least one "+
				"non-test production file",
			"golang", pkgDir,
		)
	}
	return pkg
}

// nonTestGoFile is a parser.ParseDir filter that excludes _test.go files,
// so this golden test reflects only the package's public production
// source, mirroring pkg/shipwright/api_golden_test.go's own filter.
func nonTestGoFile(info os.FileInfo) bool {
	return !strings.HasSuffix(info.Name(), "_test.go")
}

// exportedDeclNames extracts every exported top-level identifier
// introduced by decl: type names, const/var names, and function/method
// names. Struct field names are deliberately out of scope — design.md D-F
// speaks to "the exported identifiers of providers/go", meaning top-level
// package declarations, not the fields of an already-standalone type.
func exportedDeclNames(decl ast.Decl) []string {
	var names []string

	switch d := decl.(type) {
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if s.Name.IsExported() {
					names = append(names, s.Name.Name)
				}
			case *ast.ValueSpec:
				for _, n := range s.Names {
					if n.IsExported() {
						names = append(names, n.Name)
					}
				}
			}
		}
	case *ast.FuncDecl:
		if d.Name.IsExported() {
			names = append(names, d.Name.Name)
		}
	}

	return names
}
