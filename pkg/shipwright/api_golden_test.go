package shipwright_test

import (
	"bytes"
	"flag"
	"fmt"
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
	pkg, err := parsePackageDir(fset, pkgDir)
	if err != nil {
		t.Fatalf("failed to parse package directory %s: %v", pkgDir, err)
	}

	return renderPackageSurface(t, fset, pkg)
}

// parsedPackage is a minimal stand-in for the deprecated go/ast.Package
// (SA1019: deprecated since Go 1.22, use go/types instead) — renderPackageSurface
// only ever needs the parsed files, never Package's Name/Scope/Imports.
type parsedPackage struct {
	Files map[string]*ast.File
}

// parsePackageDir parses every non-test .go file directly in dir, replacing
// the deprecated parser.ParseDir/ast.Package pairing (SA1019: ParseDir has
// been deprecated since Go 1.25) with an explicit directory walk plus
// parser.ParseFile.
func parsePackageDir(fset *token.FileSet, dir string) (*parsedPackage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	files := make(map[string]*ast.File)
	for _, entry := range entries {
		if entry.IsDir() || !nonTestGoFile(entry.Name()) {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		files[entry.Name()] = file
	}

	return &parsedPackage{Files: files}, nil
}

// renderPackageSurface walks every top-level declaration in pkg and prints
// the exported ones deterministically. Extracted from renderExportedSurface
// so the rendering logic itself — not just the real pkg/shipwright
// directory — can be exercised directly against synthetic fixtures.
func renderPackageSurface(t *testing.T, fset *token.FileSet, pkg *parsedPackage) string {
	t.Helper()

	var decls []string

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				switch d.Tok {
				case token.TYPE:
					decls = append(decls, renderExportedTypeSpecs(t, fset, d)...)
				case token.CONST:
					decls = append(decls, renderExportedValueSpecs(t, fset, d, "const")...)
				case token.VAR:
					decls = append(decls, renderExportedValueSpecs(t, fset, d, "var")...)
				}
			case *ast.FuncDecl:
				if r := renderExportedFuncDecl(t, fset, d); r != "" {
					decls = append(decls, r)
				}
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

// renderExportedFuncDecl prints an exported top-level function or method's
// signature (never its body — the golden file records the guaranteed
// surface shape, not implementation). Returns "" for an unexported func,
// which the caller must skip rather than append.
func renderExportedFuncDecl(t *testing.T, fset *token.FileSet, funcDecl *ast.FuncDecl) string {
	t.Helper()

	if !funcDecl.Name.IsExported() {
		return ""
	}

	sigOnly := *funcDecl
	sigOnly.Body = nil
	sigOnly.Doc = nil

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, &sigOnly); err != nil {
		t.Fatalf("failed to print func decl %s: %v", funcDecl.Name.Name, err)
	}

	return strings.TrimSpace(buf.String())
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

		filtered := filterExportedValueSpec(valueSpec)
		if filtered == nil {
			continue
		}

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, filtered); err != nil {
			t.Fatalf("failed to print value spec: %v", err)
		}

		rendered = append(rendered, keyword+" "+strings.TrimSpace(buf.String()))
	}

	return rendered
}

// filterExportedValueSpec returns a copy of valueSpec containing only its
// exported names — and, when Values is one-to-one with Names, only the
// correspondingly-indexed values — or nil if none of its names are
// exported. Without this, a mixed grouped declaration like
// `const Foo, internalBar = "x", "y"` would leak the unexported name
// (and its value) into the guaranteed-surface golden file the moment any
// one name in the group is exported.
func filterExportedValueSpec(valueSpec *ast.ValueSpec) *ast.ValueSpec {
	parallelValues := len(valueSpec.Values) == len(valueSpec.Names)

	var names []*ast.Ident
	var values []ast.Expr

	for i, name := range valueSpec.Names {
		if !name.IsExported() {
			continue
		}
		names = append(names, name)
		if parallelValues {
			values = append(values, valueSpec.Values[i])
		}
	}

	if len(names) == 0 {
		return nil
	}

	filtered := *valueSpec
	filtered.Names = names
	if parallelValues {
		filtered.Values = values
	}
	return &filtered
}

// nonTestGoFile excludes _test.go files, so the golden test reflects only
// the package's public production source.
func nonTestGoFile(name string) bool {
	return !strings.HasSuffix(name, "_test.go")
}

// parseFixturePackage parses a synthetic single-file "shipwright" package
// from source, so renderPackageSurface's rendering logic can be exercised
// directly without depending on the real pkg/shipwright directory contents.
func parseFixturePackage(t *testing.T, fset *token.FileSet, source string) *parsedPackage {
	t.Helper()

	file, err := parser.ParseFile(fset, "fixture.go", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse fixture source: %v", err)
	}

	return &parsedPackage{
		Files: map[string]*ast.File{"fixture.go": file},
	}
}

// TestRenderPackageSurface_IncludesExportedVar guards against the golden
// test silently missing an exported top-level var — renderPackageSurface's
// GenDecl switch previously handled only token.TYPE and token.CONST.
func TestRenderPackageSurface_IncludesExportedVar(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	pkg := parseFixturePackage(t, fset, `package shipwright

var ExportedVar = 1

var unexportedVar = 2
`)

	got := renderPackageSurface(t, fset, pkg)

	if !strings.Contains(got, "ExportedVar") {
		t.Fatalf("rendered surface missing exported var ExportedVar:\n%s", got)
	}
	if strings.Contains(got, "unexportedVar") {
		t.Fatalf("rendered surface leaked unexported var unexportedVar:\n%s", got)
	}
}

// TestRenderPackageSurface_IncludesExportedFunc guards against the golden
// test silently missing an exported top-level function or method —
// *ast.FuncDecl previously wasn't handled at all (only *ast.GenDecl was).
func TestRenderPackageSurface_IncludesExportedFunc(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	pkg := parseFixturePackage(t, fset, `package shipwright

func ExportedFunc(x int) string {
	return "unused body must not leak into the golden file"
}

func unexportedFunc() {}
`)

	got := renderPackageSurface(t, fset, pkg)

	if !strings.Contains(got, "func ExportedFunc(x int) string") {
		t.Fatalf("rendered surface missing exported func signature:\n%s", got)
	}
	if strings.Contains(got, "unused body must not leak") {
		t.Fatalf("rendered surface leaked function body, not just its signature:\n%s", got)
	}
	if strings.Contains(got, "unexportedFunc") {
		t.Fatalf("rendered surface leaked unexported func unexportedFunc:\n%s", got)
	}
}

// TestRenderPackageSurface_MixedConstGroupExcludesUnexportedName guards
// against a mixed grouped declaration leaking its unexported name (and
// value) into the golden file the moment one name in the group is
// exported — the pre-fix behavior rendered the whole ValueSpec verbatim.
func TestRenderPackageSurface_MixedConstGroupExcludesUnexportedName(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	pkg := parseFixturePackage(t, fset, `package shipwright

const Foo, internalBar = "x", "y"
`)

	got := renderPackageSurface(t, fset, pkg)

	if !strings.Contains(got, "Foo") || !strings.Contains(got, `"x"`) {
		t.Fatalf("rendered surface missing exported const Foo = \"x\":\n%s", got)
	}
	if strings.Contains(got, "internalBar") {
		t.Fatalf("rendered surface leaked unexported name internalBar from a mixed group:\n%s", got)
	}
	if strings.Contains(got, `"y"`) {
		t.Fatalf("rendered surface leaked unexported value \"y\" from a mixed group:\n%s", got)
	}
}
