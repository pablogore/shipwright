package shipwright_test

import (
	"bytes"
	"flag"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// update regenerates testdata/api.golden from the current package source.
// Per shipwright-testing-tdd's golden-file rule, a diff produced by
// `-update` MUST be read and deliberately accepted, never blind-committed.
var update = flag.Bool("update", false, "update pkg/shipwright/testdata/api.golden from current source")

// TestGuaranteedSurface_MatchesGolden renders every exported top-level
// const and type declaration in pkg/shipwright and compares it against
// testdata/api.golden. Any change to this guaranteed surface (design.md
// D-E: the five capability interfaces, the config structs, and
// ContractVersion) is a deliberate, reviewable decision — this test exists
// so that decision cannot happen silently.
func TestGuaranteedSurface_MatchesGolden(t *testing.T) {
	got := renderExportedSurface(t)

	goldenPath := filepath.Join("testdata", "api.golden")

	if *update {
		if err := os.WriteFile(goldenPath, []byte(got), 0o600); err != nil {
			t.Fatalf("failed to update golden file %s: %v", goldenPath, err)
		}
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s (run with -update once the surface is intentional): %v", goldenPath, err)
	}

	if got != string(want) {
		t.Fatalf(
			"guaranteed surface drifted from %s\nrun `go test ./pkg/shipwright/... -run TestGuaranteedSurface_MatchesGolden -update` "+
				"and review the diff yourself before accepting it\n--- got ---\n%s\n--- want ---\n%s",
			goldenPath, got, want,
		)
	}
}

// renderExportedSurface parses this package's non-test source files and
// prints every exported top-level type and const declaration, sorted
// deterministically, into a single string.
func renderExportedSurface(t *testing.T) string {
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

	pkg, ok := pkgs["shipwright"]
	if !ok {
		t.Fatalf("package %q not found while parsing %s", "shipwright", pkgDir)
	}

	var decls []string

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			switch genDecl.Tok {
			case token.TYPE:
				decls = append(decls, renderExportedTypeSpecs(t, fset, genDecl)...)
			case token.CONST:
				decls = append(decls, renderExportedValueSpecs(t, fset, genDecl, "const")...)
			}
		}
	}

	sort.Strings(decls)

	var b strings.Builder
	for _, d := range decls {
		b.WriteString(d)
		b.WriteString("\n\n")
	}

	return b.String()
}

func renderExportedTypeSpecs(t *testing.T, fset *token.FileSet, genDecl *ast.GenDecl) []string {
	t.Helper()

	var rendered []string

	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok || !typeSpec.Name.IsExported() {
			continue
		}

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, typeSpec); err != nil {
			t.Fatalf("failed to print type spec %s: %v", typeSpec.Name.Name, err)
		}

		rendered = append(rendered, "type "+strings.TrimSpace(buf.String()))
	}

	return rendered
}

func renderExportedValueSpecs(t *testing.T, fset *token.FileSet, genDecl *ast.GenDecl, keyword string) []string {
	t.Helper()

	var rendered []string

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		exportedAny := false
		for _, name := range valueSpec.Names {
			if name.IsExported() {
				exportedAny = true
				break
			}
		}
		if !exportedAny {
			continue
		}

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, valueSpec); err != nil {
			t.Fatalf("failed to print value spec: %v", err)
		}

		rendered = append(rendered, keyword+" "+strings.TrimSpace(buf.String()))
	}

	return rendered
}

// nonTestGoFile is a parser.ParseDir filter that excludes _test.go files,
// so the golden test reflects only the package's public production source.
func nonTestGoFile(info os.FileInfo) bool {
	return !strings.HasSuffix(info.Name(), "_test.go")
}
