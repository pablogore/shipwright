// Schema drift golden test — tasks.md 4.3, design.md D-H: "Schema drift
// enforcement mirrors D-E: internal/workflow/manifest/testdata/schema.golden
// records the accepted field set; any schema change fails RED and forces
// an explicit apiVersion decision in the same PR." Pattern mirrored from
// pkg/shipwright/api_golden_test.go (guaranteed-surface golden) and
// internal/daggerpin's file-as-text guards, adapted to render only this
// package's schema.go type declarations rather than the whole package
// surface — the manifest's *accepted field set*, not its exported
// functions.
package manifest_test

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

// update regenerates testdata/schema.golden from the current schema.go
// source. Per shipwright-testing-tdd's golden-file rule, a diff produced
// by `-update` MUST be read and deliberately accepted, never
// blind-committed.
var update = flag.Bool("update", false, "update testdata/schema.golden from the current schema.go source")

// TestManifestSchema_MatchesGolden renders every exported struct
// declaration in schema.go (the manifest's accepted field set) and
// compares it against testdata/schema.golden. Any change to this set —
// adding, removing, or retyping a field — is a deliberate, reviewable
// decision that must also weigh an apiVersion bump (design.md D-H); this
// test exists so that decision cannot happen silently.
func TestManifestSchema_MatchesGolden(t *testing.T) {
	got := renderSchemaSurface(t)

	goldenPath := filepath.Join("testdata", "schema.golden")

	if *update {
		if err := os.WriteFile(goldenPath, []byte(got), 0o600); err != nil {
			t.Fatalf("failed to update golden file %s: %v", goldenPath, err)
		}
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s (run with -update once the schema change is intentional): %v", goldenPath, err)
	}

	if got != string(want) {
		t.Fatalf(
			"manifest schema drifted from %s\nrun `go test ./internal/workflow/manifest/... -run TestManifestSchema_MatchesGolden -update` "+
				"and review the diff yourself before accepting it — any accepted-field-set change forces an apiVersion decision in the same PR\n"+
				"--- got ---\n%s\n--- want ---\n%s",
			goldenPath, got, want,
		)
	}
}

// renderSchemaSurface parses schema.go (only — not the whole package, so
// Parse/ParseFile/ValidateIdentity/ValidateStructure never leak into a
// golden meant to capture field shape, not the parser's API) and prints
// every exported top-level type declaration, sorted deterministically.
func renderSchemaSurface(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test source location via runtime.Caller")
	}
	schemaPath := filepath.Join(filepath.Dir(thisFile), "schema.go")

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, schemaPath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse %s: %v", schemaPath, err)
	}

	var decls []string
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		decls = append(decls, renderExportedTypeSpecs(t, fset, genDecl)...)
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
