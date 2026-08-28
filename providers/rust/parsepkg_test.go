package rust_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// parsedPackage is a minimal stand-in for the deprecated go/ast.Package
// (SA1019: deprecated since Go 1.22, use go/types instead) — the golden
// tests here only ever need the parsed files, never Package's
// Name/Scope/Imports.
type parsedPackage struct {
	Files map[string]*ast.File
}

// parsePackageDir parses every top-level .go file in dir whose base name
// passes filter (a nil filter accepts every .go file), grouping the result
// by each file's declared package name. This replaces the deprecated
// parser.ParseDir/ast.Package pairing (SA1019: ParseDir has been deprecated
// since Go 1.25) with an explicit directory walk plus parser.ParseFile.
func parsePackageDir(fset *token.FileSet, dir string, filter func(name string) bool, mode parser.Mode) (map[string]*parsedPackage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	pkgs := make(map[string]*parsedPackage)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		if filter != nil && !filter(name) {
			continue
		}

		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, mode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		pkgName := file.Name.Name
		pkg, ok := pkgs[pkgName]
		if !ok {
			pkg = &parsedPackage{Files: make(map[string]*ast.File)}
			pkgs[pkgName] = pkg
		}
		pkg.Files[name] = file
	}

	return pkgs, nil
}

// nonTestGoFile is a parsePackageDir filter that excludes _test.go files,
// so a golden test built on it reflects only the package's public
// production source, mirroring providers/go's own filter.
func nonTestGoFile(name string) bool {
	return !strings.HasSuffix(name, "_test.go")
}
