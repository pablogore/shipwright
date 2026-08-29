package golang

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

// Ambiguity codes, design.md D-5. Detected before any mutation is even
// considered: analysis always completes before the first WithNewFile.
const (
	CodeA1 = "A1" // two modules declare different go directives
	CodeA2 = "A2" // two modules declare different toolchain directives
	CodeA3 = "A3" // go.work's go, or .go-version, disagrees with the unanimous module go
	CodeA4 = "A4" // target version is a downgrade from current, and downgrade is not allowed
	CodeA5 = "A5" // malformed target version or existing directive
	CodeA6 = "A6" // neither go.work nor go.mod found at the workspace root
)

// AmbiguousToolchainError is returned whenever a workspace's tier-1 version
// sources (go.mod go/toolchain, go.work, .go-version) disagree, or a
// requested target version is malformed or a disallowed downgrade. It lists
// every conflicting site so a caller never has to guess which file to fix.
type AmbiguousToolchainError struct {
	Code  string
	Sites []string
}

func (e *AmbiguousToolchainError) Error() string {
	return fmt.Sprintf("toolchain: ambiguous version sources (%s): %s", e.Code, strings.Join(e.Sites, "; "))
}

// ModuleFile is one workspace module's parsed go.mod version directives.
type ModuleFile struct {
	// Path is the module's directory relative to the workspace root ("."
	// for a single go.mod at the workspace root itself).
	Path      string
	Go        string // go directive, e.g. "1.26.7" ("" if absent, which modfile does not allow for a valid go.mod, but is defensively handled)
	Toolchain string // toolchain directive, e.g. "go1.26.7" ("" if absent)
}

// Workspace is the parsed, unvalidated view of a workspace's tier-1 version
// sources (design.md D-4). Validation against the A1-A6 ambiguity rules is
// detectConflicts's job, not parseWorkspace's.
type Workspace struct {
	HasGoWork    bool
	GoWorkGo     string // go.work's go directive ("" if go.work is absent or has none)
	HasGoVersion bool
	GoVersion    string // .go-version's trimmed content ("" if absent)
	Modules      []ModuleFile
}

// WorkspaceInput carries the raw bytes of a workspace's tier-1 sources,
// already gathered by the caller (GoRuntimeInspector, via daggerkit) before
// any parsing happens — this package never touches Dagger or the host
// filesystem itself. Modules maps each module's directory, relative to the
// workspace root, to its go.mod bytes; "." is a go.mod at the workspace
// root itself (the single-module case).
type WorkspaceInput struct {
	GoWork    []byte // nil/empty if go.work is absent
	GoVersion []byte // nil/empty if .go-version is absent
	Modules   map[string][]byte
}

// parseWorkspace parses raw workspace bytes into a Workspace. It performs no
// ambiguity validation beyond what modfile itself enforces while parsing: a
// malformed go/toolchain directive line causes modfile.Parse/ParseWork to
// fail, which parseWorkspace reports as an A5 *AmbiguousToolchainError
// (design.md D-5, threat matrix "malformed").
func parseWorkspace(input WorkspaceInput) (*Workspace, error) {
	ws := &Workspace{}

	if len(input.GoWork) > 0 {
		wf, err := modfile.ParseWork("go.work", input.GoWork, nil)
		if err != nil {
			return nil, &AmbiguousToolchainError{
				Code:  CodeA5,
				Sites: []string{fmt.Sprintf("go.work: malformed: %v", err)},
			}
		}
		ws.HasGoWork = true
		if wf.Go != nil {
			ws.GoWorkGo = wf.Go.Version
		}
	}

	if len(input.GoVersion) > 0 {
		ws.HasGoVersion = true
		ws.GoVersion = strings.TrimSpace(string(input.GoVersion))
	}

	paths := make([]string, 0, len(input.Modules))
	for p := range input.Modules {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		modPath := p + "/go.mod"
		mf, err := modfile.Parse(modPath, input.Modules[p], nil)
		if err != nil {
			return nil, &AmbiguousToolchainError{
				Code:  CodeA5,
				Sites: []string{fmt.Sprintf("%s: malformed: %v", modPath, err)},
			}
		}

		m := ModuleFile{Path: p}
		if mf.Go != nil {
			m.Go = mf.Go.Version
		}
		if mf.Toolchain != nil {
			m.Toolchain = mf.Toolchain.Name
		}
		ws.Modules = append(ws.Modules, m)
	}

	return ws, nil
}

// ConflictOptions carries the optional pieces of detectConflicts input that
// only matter when a target version is actually being requested (the
// runtime-upgrade capability, Phase 2+) — runtime-inspect always leaves
// TargetVersion empty, which skips the A4/A5-on-target checks entirely and
// only evaluates the workspace's own internal consistency (A1-A3, A6).
type ConflictOptions struct {
	TargetVersion  string
	AllowDowngrade bool
}

// detectConflicts checks ws against the A1-A6 ambiguity rules (design.md
// D-5) and returns nil when the workspace's tier-1 sources are unanimous
// (and, when opts.TargetVersion is set, when the target is well-formed and
// not a disallowed downgrade).
func detectConflicts(ws *Workspace, opts ConflictOptions) *AmbiguousToolchainError {
	if !ws.HasGoWork && len(ws.Modules) == 0 {
		return &AmbiguousToolchainError{
			Code:  CodeA6,
			Sites: []string{"neither go.work nor go.mod found at the workspace root"},
		}
	}

	unanimousGo, err := detectModuleDivergence(ws)
	if err != nil {
		return err
	}

	if err := detectSourceMismatch(ws, unanimousGo); err != nil {
		return err
	}

	if opts.TargetVersion == "" {
		return nil
	}

	return validateTarget(opts.TargetVersion, unanimousGo, ws.GoWorkGo, opts.AllowDowngrade)
}

// detectModuleDivergence returns A1 if two or more modules declare
// different non-empty go directives, or A2 if two or more modules declare
// different non-empty toolchain directives. On success it returns the
// workspace's single agreed-upon go directive value (possibly "" if no
// module declares one at all).
func detectModuleDivergence(ws *Workspace) (string, *AmbiguousToolchainError) {
	getGo := func(m ModuleFile) string { return m.Go }
	getToolchain := func(m ModuleFile) string { return m.Toolchain }

	if hasDivergence(ws.Modules, getGo) {
		return "", &AmbiguousToolchainError{Code: CodeA1, Sites: divergentSites(ws.Modules, getGo, "go")}
	}
	if hasDivergence(ws.Modules, getToolchain) {
		return "", &AmbiguousToolchainError{Code: CodeA2, Sites: divergentSites(ws.Modules, getToolchain, "toolchain")}
	}

	var unanimousGo string
	for _, m := range ws.Modules {
		if m.Go != "" {
			unanimousGo = m.Go
			break
		}
	}

	return unanimousGo, nil
}

// hasDivergence reports whether two or more modules declare a different
// non-empty value for the field selected by get.
func hasDivergence(modules []ModuleFile, get func(ModuleFile) string) bool {
	seen := ""
	for _, m := range modules {
		v := get(m)
		if v == "" {
			continue
		}
		if seen == "" {
			seen = v
			continue
		}
		if v != seen {
			return true
		}
	}
	return false
}

// divergentSites lists every module whose value for the field selected by
// get is non-empty, labeled with directive for readability in the error.
func divergentSites(modules []ModuleFile, get func(ModuleFile) string, directive string) []string {
	var sites []string
	for _, m := range modules {
		if v := get(m); v != "" {
			sites = append(sites, fmt.Sprintf("%s/go.mod: %s %s", m.Path, directive, v))
		}
	}
	return sites
}

// detectSourceMismatch returns A3 if go.work's go directive, or
// .go-version's content, disagrees with the workspace's unanimous module go
// directive. unanimousGo may be "" (no module declares a go directive at
// all), in which case there is nothing to compare against.
func detectSourceMismatch(ws *Workspace, unanimousGo string) *AmbiguousToolchainError {
	if unanimousGo == "" {
		return nil
	}

	var sites []string

	if ws.HasGoWork && ws.GoWorkGo != "" && ws.GoWorkGo != unanimousGo {
		sites = append(sites, "go.work: go "+ws.GoWorkGo)
	}
	if ws.HasGoVersion && ws.GoVersion != "" && ws.GoVersion != unanimousGo {
		sites = append(sites, ".go-version: "+ws.GoVersion)
	}

	if len(sites) == 0 {
		return nil
	}

	sites = append(sites, fmt.Sprintf("go.mod: go %s (unanimous)", unanimousGo))
	return &AmbiguousToolchainError{Code: CodeA3, Sites: sites}
}

// validateTarget checks a requested upgrade target version: A5 if it is
// malformed (fails the same format modfile.AddGoStmt itself enforces), A4
// if it is a downgrade from the workspace's current version and downgrade
// is not explicitly allowed. current is whichever of unanimousGo/goWorkGo
// is non-empty (they cannot disagree here — detectSourceMismatch already
// ran and returned nil).
func validateTarget(target, unanimousGo, goWorkGo string, allowDowngrade bool) *AmbiguousToolchainError {
	if err := validTargetVersion(target); err != nil {
		return err
	}

	current := unanimousGo
	if current == "" {
		current = goWorkGo
	}
	if current == "" {
		return nil
	}

	if !allowDowngrade && semver.Compare("v"+target, "v"+current) < 0 {
		return &AmbiguousToolchainError{
			Code: CodeA4,
			Sites: []string{
				fmt.Sprintf("targetVersion %s is a downgrade from current %s", target, current),
			},
		}
	}

	return nil
}

// validTargetVersion reports whether target is a well-formed go directive
// value (the same format modfile.AddGoStmt itself enforces). Both
// mutateGoMod and mutateGoWork call this before touching any byte (threat
// matrix: command construction from config — "1.26.7; rm -rf /" and
// "--flag" are rejected here, at parse, never reaching "golang:"+v or an
// argv slice). This is defense in depth alongside validateTarget's own A5
// check above: mutateGoMod/mutateGoWork may be called directly (unit
// tests) without ever going through detectConflicts first.
func validTargetVersion(target string) *AmbiguousToolchainError {
	if !modfile.GoVersionRE.MatchString(target) {
		return &AmbiguousToolchainError{
			Code:  CodeA5,
			Sites: []string{fmt.Sprintf("targetVersion %q is not a valid go directive value", target)},
		}
	}
	return nil
}

// mutateGoMod parses modBytes and returns its bytes with the go directive
// set to targetVersion. When modBytes already declares a toolchain
// directive, it is updated to match ("go"+targetVersion) too — a
// toolchain directive is never added where none existed before
// (discovery-driven: mutate only what already exists, design.md D-4/D-9).
func mutateGoMod(modBytes []byte, targetVersion string) ([]byte, error) {
	if err := validTargetVersion(targetVersion); err != nil {
		return nil, err
	}

	mf, err := modfile.Parse("go.mod", modBytes, nil)
	if err != nil {
		return nil, &AmbiguousToolchainError{Code: CodeA5, Sites: []string{fmt.Sprintf("go.mod: malformed: %v", err)}}
	}

	if err := mf.AddGoStmt(targetVersion); err != nil {
		return nil, &AmbiguousToolchainError{Code: CodeA5, Sites: []string{fmt.Sprintf("go.mod: failed to set go directive: %v", err)}}
	}

	if mf.Toolchain != nil {
		if err := mf.AddToolchainStmt("go" + targetVersion); err != nil {
			return nil, &AmbiguousToolchainError{Code: CodeA5, Sites: []string{fmt.Sprintf("go.mod: failed to set toolchain directive: %v", err)}}
		}
	}

	mf.Cleanup()
	return mf.Format()
}

// mutateGoWork parses workBytes and returns its bytes with go.work's own
// go directive set to targetVersion, updating an existing toolchain
// directive the same discovery-driven way mutateGoMod does for go.mod.
//
// Not yet wired into GoRuntimeUpgrader.Upgrade — the go.work-referenced
// multi-module traversal loop that calls this is tasks.md Phase 3
// (design.md D-7's mutation loop). Built now, unit-tested directly against
// testdata/runtime fixtures, per design.md's own Testing Strategy table
// ("mutateGoMod, mutateGoWork" share toolchain.go's pure byte-level TDD
// mass, no Dagger, no engine needed for either).
func mutateGoWork(workBytes []byte, targetVersion string) ([]byte, error) {
	if err := validTargetVersion(targetVersion); err != nil {
		return nil, err
	}

	wf, err := modfile.ParseWork("go.work", workBytes, nil)
	if err != nil {
		return nil, &AmbiguousToolchainError{Code: CodeA5, Sites: []string{fmt.Sprintf("go.work: malformed: %v", err)}}
	}

	if err := wf.AddGoStmt(targetVersion); err != nil {
		return nil, &AmbiguousToolchainError{Code: CodeA5, Sites: []string{fmt.Sprintf("go.work: failed to set go directive: %v", err)}}
	}

	if wf.Toolchain != nil {
		if err := wf.AddToolchainStmt("go" + targetVersion); err != nil {
			return nil, &AmbiguousToolchainError{Code: CodeA5, Sites: []string{fmt.Sprintf("go.work: failed to set toolchain directive: %v", err)}}
		}
	}

	wf.Cleanup()
	return modfile.Format(wf.Syntax), nil
}

// mutateGoVersion returns .go-version's new content for targetVersion — a
// single line, the same tool-agnostic format goenv/asdf/mise use. No
// parsing of the previous content is needed: the file's only content is
// the version string itself, unlike go.mod/go.work's structured
// directives.
func mutateGoVersion(targetVersion string) []byte {
	return []byte(targetVersion + "\n")
}
