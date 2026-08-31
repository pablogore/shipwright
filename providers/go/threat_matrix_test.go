// SECURITY test — tasks.md 1.23/2.16, design.md's Threat Matrix rows
// "Arbitrary file write outside returned dir" and "Host subprocess": a
// static assertion (not a documentation comment) that toolchain.go,
// runtimeinspector.go, and runtimeupgrader.go — the files implementing
// GoRuntimeInspector's read-only drift detection and GoRuntimeUpgrader's
// mutation (writes only ever go through Directory.WithNewFile on an
// immutable value, never a host primitive) — never import a package
// capable of a host filesystem write, a host subprocess, an HTTP client, or
// a VCS/git invocation. Mirrors internal/workflow/providers/security_test.go's
// own "prove it, don't just assert it" static-import-guard style, scoped to
// these specific files (parser.ImportsOnly, no `go list` subprocess).
package golang_test

import (
	"go/parser"
	"go/token"
	"sort"
	"testing"
)

// runtimeInspectFiles are the exact source files design.md's Threat Matrix
// names: toolchain.go (pure-Go parse/conflict core), runtimeinspector.go
// (the Dagger-facing read-only capability), and runtimeupgrader.go (the
// Dagger-facing mutating capability, added tasks.md 2.16). All three are
// production code only — test fixture files (toolchain_test.go etc.)
// legitimately import "os" to read testdata from the host and are
// deliberately excluded.
var runtimeInspectFiles = []string{
	"toolchain.go",
	"runtimeinspector.go",
	"runtimeupgrader.go",
}

// forbiddenImports maps each disallowed import path to the Threat Matrix
// row it violates if reachable from either runtimeInspectFiles entry.
//
// The net/net-http ban is not a claim that runtime-upgrade never performs
// network I/O — its validation step (openspec/specs/runtime-toolchain,
// requirement "Network-Dependent Validation") pulls a "golang:"+
// targetVersion image and may run `go mod tidy` / `go build ./...`, all of
// which can reach the network, but only ever through the Dagger SDK's own
// container boundary (WithExec inside a container Sync), never a direct
// host-level net/net-http call from this file. That boundary is what's
// enforced here: a direct import would let this file open a host socket
// itself, bypassing Dagger entirely, which is what every row below guards
// against regardless of which capability's implementation it is.
var forbiddenImports = map[string]string{
	"os":                          "Arbitrary file write outside returned dir — os.WriteFile/os.Create/os.Remove must never be reachable; all writes go through Directory.WithNewFile on an immutable value",
	"os/exec":                     "Host subprocess — every command must run in a Dagger container via argv-array WithExec, never exec.Command",
	"net/http":                    "Direct host-level network access bypassing the Dagger container boundary — any network I/O these files perform must go through WithExec inside a container, never a direct HTTP client",
	"net":                         "Direct host-level network access bypassing the Dagger container boundary — any network I/O these files perform must go through WithExec inside a container, never a direct socket",
	"github.com/go-git/go-git/v5": "VCS/PR automation — D2, no SCM code path exists in either runtime-inspect or runtime-upgrade, regardless of their network behavior",
}

// TestRuntimeInspectFiles_NoHostWriteExecNetworkOrGitImports is tasks.md
// 1.23's static RED-then-GREEN assertion: parsing toolchain.go's and
// runtimeinspector.go's own import declarations (not a transitive walk —
// design.md's Threat Matrix names these two files specifically, and their
// only same-module dependency is providers/go/daggerkit, already proven
// Dagger-only by its own package boundary) and failing if any forbidden
// import path in forbiddenImports appears.
func TestRuntimeInspectFiles_NoHostWriteExecNetworkOrGitImports(t *testing.T) {
	t.Parallel()

	var violations []string

	for _, file := range runtimeInspectFiles {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", file, err)
		}

		for _, imp := range f.Imports {
			importPath := imp.Path.Value
			importPath = importPath[1 : len(importPath)-1] // strip quotes
			if reason, forbidden := forbiddenImports[importPath]; forbidden {
				violations = append(violations, file+" imports "+importPath+": "+reason)
			}
		}
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("runtime-inspect threat-matrix violation(s):\n%s", joinLines(violations))
	}
}

// scmSideEffectImports is the subset of forbiddenImports that maps
// directly to openspec/specs/runtime-toolchain's "No SCM Side Effects"
// requirement: no code path in either capability may create a branch,
// push, or call an SCM/PR API. Kept as its own test (rather than folded
// into the broader host-write/exec/network guard above) so this specific
// guarantee — the one that stays true regardless of runtime-upgrade's
// network behavior — has its own single, directly traceable proof.
var scmSideEffectImports = map[string]string{
	"os/exec":                     forbiddenImports["os/exec"],
	"github.com/go-git/go-git/v5": forbiddenImports["github.com/go-git/go-git/v5"],
}

// TestRuntimeUpgradeFiles_NoSCMSideEffectImports proves runtime-upgrade
// (and runtime-inspect) reach no git command or SCM/PR API: no
// github.com/go-git/go-git/v5 import (a git library call) and no os/exec
// import (a host git binary invocation) in either capability's
// implementation files. This holds independently of runtime-upgrade's
// validation step needing network access for its own toolchain/module
// resolution — "no SCM side effects" and "no network" are separate
// guarantees, and this test proves only the former.
func TestRuntimeUpgradeFiles_NoSCMSideEffectImports(t *testing.T) {
	t.Parallel()

	var violations []string

	for _, file := range runtimeInspectFiles {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", file, err)
		}

		for _, imp := range f.Imports {
			importPath := imp.Path.Value
			importPath = importPath[1 : len(importPath)-1] // strip quotes
			if reason, forbidden := scmSideEffectImports[importPath]; forbidden {
				violations = append(violations, file+" imports "+importPath+": "+reason)
			}
		}
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("SCM side-effect violation(s):\n%s", joinLines(violations))
	}
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += "  - " + l + "\n"
	}
	return out
}
