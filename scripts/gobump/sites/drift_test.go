package sites

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

// goDirectivePattern matches the "go X.Y.Z" directive line in a go.mod
// file. A minimal line scan, deliberately not golang.org/x/mod/modfile
// (design.md D-8): that package is already an indirect dependency of the
// root module, but parsing go.mod directly here would promote it to a
// direct one for a single line of text this package already knows how to
// match.
var goDirectivePattern = regexp.MustCompile(`(?m)^go\s+(\d+\.\d+(?:\.\d+)?)\s*$`)

// repoRoot resolves the repository root relative to this test file's own
// location, since `go test` runs with the package directory as its
// working directory (scripts/gobump/sites -> repo root is three levels
// up).
func repoRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine the caller's file to resolve the repo root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

// TestSitesMatchGoModDirective is the standing drift guard (design.md
// D-8): it reads the repository's real, on-disk root go.mod `go`
// directive and asserts every entry in the real production var Sites
// currently renders that exact version. A green run here means the
// literal-replacement registry and the module's own tier-1 pin have not
// drifted apart; the moment a version bump lands through any path other
// than this tool (or this registry falls behind a manual go.mod edit),
// this test goes red and names exactly which registered site disagrees.
func TestSitesMatchGoModDirective(t *testing.T) {
	root := repoRoot(t)

	modBytes, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("reading root go.mod: %v", err)
	}
	m := goDirectivePattern.FindSubmatch(modBytes)
	if m == nil {
		t.Fatal(`root go.mod has no recognizable "go X.Y.Z" directive`)
	}
	want := string(m[1])

	for _, s := range Sites {
		re, err := Compile(s)
		if err != nil {
			t.Errorf("Compile(%s): %v", s.Path, err)
			continue
		}

		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(s.Path)))
		if err != nil {
			t.Errorf("reading %s: %v", s.Path, err)
			continue
		}

		matches := re.FindAllSubmatch(content, -1)
		if len(matches) != s.Occurrences {
			t.Errorf("%s: anchor matched %d time(s), expected %d (registry Occurrences out of date)", s.Path, len(matches), s.Occurrences)
			continue
		}

		for i, mm := range matches {
			got := string(mm[1])
			if got != want {
				t.Errorf("%s (match %d): on-disk version %q does not match go.mod's %q -- drift detected", s.Path, i, got, want)
			}
		}
	}
}
