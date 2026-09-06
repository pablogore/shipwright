package shipwright_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// repoRootForContractVersionParity resolves the repository root the same
// way api_golden_test.go and internal/daggerpin/pin_test.go do: via
// runtime.Caller(0), since this file lives at
// <repoRoot>/pkg/shipwright/contract_version_parity_test.go.
func repoRootForContractVersionParity(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed to resolve this test file's path")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// TestContractVersion_MatchesDaggerLayer2Literal guards design.md D-E's
// deliberately duplicated ContractVersion: .dagger/capabilities.go cannot
// import pkg/shipwright (separate Go module — .dagger/ is Dagger-generated
// and D-A forbids importing internal/**), so Shipwright.ContractVersion()
// hand-copies pkg/shipwright.ContractVersion as a bare string literal, with
// its own doc comment admitting it must be kept in sync "by hand ... until
// a cross-module parity guard exists." Without this test, a bump to
// pkg/shipwright.ContractVersion leaves the Layer 2 literal silently stale
// and CI stays green through the drift (PR #148 review finding 4).
//
// This is a root-module, file-as-text check: it parses
// .dagger/capabilities.go's SOURCE TEXT via go/parser and asserts the
// literal returned by ContractVersion() equals pkg/shipwright.ContractVersion
// — it never imports .dagger/ as a package (impossible; separate module) and
// never invokes a live Dagger engine. This mirrors exactly how
// internal/daggerpin's pin-parity test reads dagger.json as a file from the
// root module, and how api_golden_test.go parses this package's own source
// via go/parser — the established pattern in this codebase for "must not
// drift silently, forces a reviewable failure" (design.md D-B, D-E).
func TestContractVersion_MatchesDaggerLayer2Literal(t *testing.T) {
	root := repoRootForContractVersionParity(t)
	capsPath := filepath.Join(root, ".dagger", "capabilities.go")

	got, err := daggerLayer2ContractVersionLiteral(capsPath)
	if err != nil {
		t.Fatalf("daggerLayer2ContractVersionLiteral(%q) error = %v, want nil", capsPath, err)
	}

	if got != shipwright.ContractVersion {
		t.Fatalf(
			"Shipwright.ContractVersion() literal in %s = %q, want %q (pkg/shipwright.ContractVersion) — "+
				"D-E requires these to stay equal; bump both together",
			capsPath, got, shipwright.ContractVersion,
		)
	}
}

// daggerLayer2ContractVersionLiteral parses capsPath (expected to be
// .dagger/capabilities.go) and extracts the string literal returned by the
// `func (m *Shipwright) ContractVersion() string` method, without importing
// or building the .dagger/ module.
func daggerLayer2ContractVersionLiteral(capsPath string) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, capsPath, nil, 0)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", capsPath, err)
	}

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != "ContractVersion" || !hasShipwrightPointerReceiver(funcDecl) {
			continue
		}

		lit, ok := singleStringReturnLiteral(funcDecl)
		if !ok {
			return "", fmt.Errorf("%s: func (*Shipwright) ContractVersion() does not return a single string literal", capsPath)
		}
		return lit, nil
	}

	return "", fmt.Errorf("%s: no ContractVersion() method found on Shipwright", capsPath)
}

// hasShipwrightPointerReceiver reports whether funcDecl is a method on
// *Shipwright — the exact receiver capabilities.go's ContractVersion uses.
func hasShipwrightPointerReceiver(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) != 1 {
		return false
	}

	star, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	ident, ok := star.X.(*ast.Ident)
	return ok && ident.Name == "Shipwright"
}

// singleStringReturnLiteral returns the unquoted string value when
// funcDecl's body is exactly one `return "<literal>"` statement, and false
// otherwise (e.g. a computed expression, multiple statements, or a
// non-string literal) — this test only ever expects to see a bare literal,
// and anything else should fail loudly rather than parse silently.
func singleStringReturnLiteral(funcDecl *ast.FuncDecl) (string, bool) {
	if funcDecl.Body == nil || len(funcDecl.Body.List) != 1 {
		return "", false
	}

	ret, ok := funcDecl.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return "", false
	}

	lit, ok := ret.Results[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}

	unquoted, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}

	return unquoted, true
}
