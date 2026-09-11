// Package core implements the literal-replacement half of the coordinated
// Go SDK/toolchain version bump (issue #280, design.md D-4/D-5/D-6): an
// explicit, enumerated registry of every plain-string version pin the
// GoRuntimeUpgrader provider cannot reach (workflow GO_VERSION env vars, Go
// source fallback constants, and documentation), plus a plan-then-mutate
// engine that resolves every registered Site before writing anything.
//
// Compile classifies malformed anchors (E4) and unsafe paths (E5) without
// touching the filesystem. Plan validates the target version (E6), then
// reads and matches every Site, aggregating every failure (E1-E3) into one
// returned error that names every failing path — it never writes a file.
// Apply performs the writes; it is the caller's contract to invoke it only
// after a Plan for the same site set returned no error.
//
// Relocated from scripts/gobump/sites (issue #284, design.md's core-engine
// relocation): its own production registry (var Sites) is intentionally
// NOT here — a Site registry is toolchain-specific data, not part of the
// generic engine, so it now lives with the Toolchain descriptor that owns
// it (internal/toolbump/toolchains/go for Go's registry).
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Site declares one anchored literal-replacement location whose version
// substring must track a toolchain's canonical pin.
type Site struct {
	// Path is repo-root relative. It must not be absolute, must not
	// escape the repo root, and must not contain a "testdata" path
	// segment (D-7 layer 2).
	Path string
	// Anchor is the exact literal text surrounding the version
	// substring, with the version itself replaced by the placeholder
	// "{{V}}". Anchor must contain exactly one "{{V}}" occurrence.
	Anchor string
	// Occurrences is the exact number of times Anchor (with its version
	// substring resolved) is expected to match inside Path. A count of
	// zero is always an error (E2); any other mismatch is E3.
	Occurrences int
}

// Edit is a planned, not-yet-written mutation: Path's full new file
// contents after every one of its Site's matched version substrings has
// been replaced by the target version. Edit is only ever produced by Plan.
type Edit struct {
	Path string
	New  []byte
}

// placeholder is the literal token an Anchor template uses to mark where
// the version substring lives.
const placeholder = "{{V}}"

// versionPattern matches the version substring a Site anchors around: X.Y
// or X.Y.Z, digits only.
const versionPattern = `\d+\.\d+(?:\.\d+)?`

// targetPattern is the strict format required of a bump's target version:
// exactly X.Y.Z, digits only. Anything else — flags, spaces, shell
// metacharacters such as the shell-injection-shaped "1.2.3 --base main" —
// is rejected here (E6) before it ever reaches a Plan/Apply call, a file
// write, or a shell-adjacent invocation.
var targetPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// siteError is a single classified Plan/Compile failure. The code is one
// of E1-E6 (design.md D-5); path is empty for target-format failures (E6),
// which are not associated with any one site.
type siteError struct {
	code string
	path string
	msg  string
}

func (e *siteError) Error() string {
	if e.path == "" {
		return fmt.Sprintf("[%s] %s", e.code, e.msg)
	}
	return fmt.Sprintf("[%s] %s: %s", e.code, e.path, e.msg)
}

// Compile builds the anchored regexp for Site s. It classifies E4
// (malformed anchor: zero or more than one "{{V}}" placeholder) and E5
// (unsafe path: absolute, escapes the repo root, or contains a "testdata"
// path segment) without touching the filesystem — path safety is knowable
// from Site.Path alone, so Compile is the natural place to check it, ahead
// of Plan's file-level checks.
func Compile(s Site) (*regexp.Regexp, error) {
	if err := checkPathSafety(s.Path); err != nil {
		return nil, err
	}

	n := strings.Count(s.Anchor, placeholder)
	if n != 1 {
		return nil, &siteError{
			code: "E4",
			path: s.Path,
			msg:  fmt.Sprintf("anchor must contain exactly one %q placeholder, found %d", placeholder, n),
		}
	}

	parts := strings.SplitN(s.Anchor, placeholder, 2)
	pattern := regexp.QuoteMeta(parts[0]) + "(" + versionPattern + ")" + regexp.QuoteMeta(parts[1])
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, &siteError{code: "E4", path: s.Path, msg: fmt.Sprintf("compiled anchor is not a valid regexp: %v", err)}
	}
	return re, nil
}

// checkPathSafety classifies E5: an absolute Path, a Path that escapes the
// repo root via a ".." segment, or a Path with a literal "testdata"
// segment anywhere in it.
func checkPathSafety(path string) error {
	if path == "" {
		return &siteError{code: "E5", path: path, msg: "path must not be empty"}
	}
	if filepath.IsAbs(path) {
		return &siteError{code: "E5", path: path, msg: "path must be repo-relative, not absolute"}
	}

	for _, seg := range strings.Split(filepath.ToSlash(path), "/") {
		switch seg {
		case "testdata":
			return &siteError{code: "E5", path: path, msg: `path must not contain a "testdata" segment`}
		case "..":
			return &siteError{code: "E5", path: path, msg: "path must not escape the repo root"}
		}
	}
	return nil
}

// Plan validates target's format (E6) before touching any site, then
// resolves every registered site: Compile the anchor, read the file,
// classify E1 (missing/non-regular/symlink path), E2 (zero matches), or E3
// (match count mismatch). Every site's failure is aggregated into a single
// returned error naming every failing path — Plan never writes a file.
func Plan(root string, sitesList []Site, target string) ([]Edit, error) {
	if !targetPattern.MatchString(target) {
		return nil, &siteError{code: "E6", msg: fmt.Sprintf("target version %q must match %s", target, targetPattern.String())}
	}

	var errs []error
	edits := make([]Edit, 0, len(sitesList))

	for _, s := range sitesList {
		edit, err := planSite(root, s, target)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		edits = append(edits, edit)
	}

	if len(errs) > 0 {
		return nil, aggregateErrors(errs)
	}
	return edits, nil
}

// planSite resolves a single Site against root, returning either a fully
// computed Edit or a classified E1/E2/E3 error. It never writes a file.
func planSite(root string, s Site, target string) (Edit, error) {
	re, err := Compile(s)
	if err != nil {
		return Edit{}, err
	}

	full := filepath.Join(root, filepath.FromSlash(s.Path))

	info, err := os.Lstat(full)
	if err != nil {
		return Edit{}, &siteError{code: "E1", path: s.Path, msg: fmt.Sprintf("cannot stat: %v", err)}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Edit{}, &siteError{code: "E1", path: s.Path, msg: "path is a symlink, not a regular file"}
	}
	if !info.Mode().IsRegular() {
		return Edit{}, &siteError{code: "E1", path: s.Path, msg: "path is not a regular file"}
	}

	content, err := os.ReadFile(full)
	if err != nil {
		return Edit{}, &siteError{code: "E1", path: s.Path, msg: fmt.Sprintf("cannot read: %v", err)}
	}

	matches := re.FindAllSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return Edit{}, &siteError{code: "E2", path: s.Path, msg: "anchor matched zero times (moved, renamed, or reformatted since registration)"}
	}
	if len(matches) != s.Occurrences {
		return Edit{}, &siteError{code: "E3", path: s.Path, msg: fmt.Sprintf("anchor matched %d time(s), expected %d", len(matches), s.Occurrences)}
	}

	newContent := make([]byte, 0, len(content))
	last := 0
	for _, m := range matches {
		// m[2], m[3] bound capture group 1: the version substring.
		newContent = append(newContent, content[last:m[2]]...)
		newContent = append(newContent, target...)
		last = m[3]
	}
	newContent = append(newContent, content[last:]...)

	return Edit{Path: s.Path, New: newContent}, nil
}

// aggregateErrors joins every site failure into one error whose message
// names every failing path, so a caller sees the full set of problems in
// one Plan call instead of stopping at the first.
func aggregateErrors(errs []error) error {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		msgs = append(msgs, e.Error())
	}
	return fmt.Errorf("toolbump: %d site(s) failed:\n%s", len(errs), strings.Join(msgs, "\n"))
}

// Apply writes every Edit's New content to its Path under root, preserving
// the existing file's mode. Apply performs no validation of its own — this
// package does not enforce call order — so it is the caller's contract to
// invoke Apply only with the edits returned by a Plan call that returned a
// nil error for the same site set, never with a partial or hand-built edit
// list.
func Apply(root string, edits []Edit) error {
	for _, e := range edits {
		full := filepath.Join(root, filepath.FromSlash(e.Path))

		mode := os.FileMode(0o644)
		if info, err := os.Stat(full); err == nil {
			mode = info.Mode()
		}

		if err := os.WriteFile(full, e.New, mode); err != nil {
			return fmt.Errorf("toolbump: apply %s: %w", e.Path, err)
		}
	}
	return nil
}
