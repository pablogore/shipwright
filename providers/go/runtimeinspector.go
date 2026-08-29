package golang

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"

	"dagger.io/dagger"
	"golang.org/x/mod/modfile"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// newDaggerDirectory wraps a real *dagger.Directory as a
// daggerkit.DaggerDirectory. It is a package-level seam, not a direct call
// to daggerkit.NewDaggerDirectoryAdapter, so runtimeinspector_test.go can
// inject a daggerkit.MockDaggerDirectory instead: GoRuntimeInspector is
// this module's first capability that reads a Directory's own content
// (Entries/File.Contents) rather than only mounting it into a container,
// and a bare *dagger.Directory has no live Dagger session to mock against
// in a unit test.
var newDaggerDirectory = daggerkit.NewDaggerDirectoryAdapter

// GoRuntimeInspector reports drift across a Go workspace's tier-1
// declarative toolchain-version sources — go.mod's go/toolchain
// directives, go.work, and .go-version (design.md D-4) — without mutating
// anything. It never calls WithNewFile, exec.Command, or any host/network
// primitive (spec: Read-Only Drift Inspection).
type GoRuntimeInspector struct {
	// Client is the Dagger client. Kept for parity with every other
	// capability provider in this package, and nil-checked the same way,
	// even though Inspect only ever reads the source Directory's own
	// content rather than driving a container.
	Client daggerkit.DaggerClient
	// WorkspaceRoot is recorded in the report as the workspace root the
	// caller intends (default "."). The source Directory passed to
	// Inspect is always read at its own top level; a caller using a
	// non-default WorkspaceRoot is responsible for scoping source to it
	// before calling Inspect.
	WorkspaceRoot string
	// ExpectedVersion, if set, is echoed into the report for the
	// caller's own comparison. Inspect does not compare it against the
	// discovered sources itself.
	ExpectedVersion string
	// FailOnDrift makes Inspect return an error when the tier-1 sources
	// disagree (design.md D-4b), instead of always succeeding with the
	// conflict recorded in the report.
	FailOnDrift bool
}

// Compile-time conformance assertion: GoRuntimeInspector must satisfy
// Layer 1's RuntimeInspector interface (pkg/shipwright).
var _ shipwright.RuntimeInspector = (*GoRuntimeInspector)(nil)

// Inspect reads source's tier-1 version sources (go.work, .go-version,
// every go.work-referenced module's go.mod, or a single root go.mod when
// go.work is absent) and returns a shipwright.DriftReport JSON string. A
// declarative location absent from the workspace has no entry in the
// report's Sources map — never a fabricated default. FailOnDrift turns a
// detected ambiguity into a returned error instead of a report-only
// success.
func (i *GoRuntimeInspector) Inspect(ctx context.Context, source *dagger.Directory) (string, error) {
	if i.Client == nil {
		return "", errors.New("runtimeinspector: dagger client is not configured")
	}
	if source == nil {
		return "", errors.New("runtimeinspector: source directory is nil")
	}

	root := i.WorkspaceRoot
	if root == "" {
		root = "."
	}

	dir := newDaggerDirectory(source)

	input, err := gatherWorkspaceInput(ctx, dir)
	if err != nil {
		return "", err
	}

	ws, parseErr := parseWorkspace(input)

	var conflict *AmbiguousToolchainError
	if parseErr != nil {
		conflict, _ = parseErr.(*AmbiguousToolchainError)
	} else {
		conflict = detectConflicts(ws, ConflictOptions{})
	}

	report := buildDriftReport(root, i.ExpectedVersion, ws, conflict)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("runtimeinspector: failed to marshal drift report: %w", err)
	}

	if i.FailOnDrift && conflict != nil {
		return "", conflict
	}

	return string(data), nil
}

// gatherWorkspaceInput reads dir's present tier-1 sources into a
// WorkspaceInput. Presence is determined via dir.Entries at the
// workspace's own top level (never via a Contents error, which would
// conflate a genuinely missing file with a malformed one).
func gatherWorkspaceInput(ctx context.Context, dir daggerkit.DaggerDirectory) (WorkspaceInput, error) {
	input := WorkspaceInput{Modules: map[string][]byte{}}

	entries, err := dir.Entries(ctx)
	if err != nil {
		return input, fmt.Errorf("runtimeinspector: failed to list workspace entries: %w", err)
	}
	present := make(map[string]bool, len(entries))
	for _, e := range entries {
		present[e] = true
	}

	if present["go.work"] {
		content, err := readContents(ctx, dir, "go.work")
		if err != nil {
			return input, fmt.Errorf("runtimeinspector: failed to read go.work: %w", err)
		}
		input.GoWork = []byte(content)
	}

	if present[".go-version"] {
		content, err := readContents(ctx, dir, ".go-version")
		if err != nil {
			return input, fmt.Errorf("runtimeinspector: failed to read .go-version: %w", err)
		}
		input.GoVersion = []byte(content)
	}

	if len(input.GoWork) > 0 {
		modulePaths, err := goWorkModulePaths(input.GoWork)
		if err != nil {
			// A malformed go.work is reported as an A5 conflict by
			// parseWorkspace itself once GoWork bytes reach it below;
			// skip module discovery here rather than failing outright.
			return input, nil
		}
		for _, p := range modulePaths {
			modPath := path.Join(p, "go.mod")
			content, err := readContents(ctx, dir, modPath)
			if err != nil {
				return input, fmt.Errorf("runtimeinspector: failed to read %s: %w", modPath, err)
			}
			input.Modules[p] = []byte(content)
		}
	} else if present["go.mod"] {
		content, err := readContents(ctx, dir, "go.mod")
		if err != nil {
			return input, fmt.Errorf("runtimeinspector: failed to read go.mod: %w", err)
		}
		input.Modules["."] = []byte(content)
	}

	return input, nil
}

// readContents reads path's full content via dir.
func readContents(ctx context.Context, dir daggerkit.DaggerDirectory, path string) (string, error) {
	return dir.File(path).Contents(ctx)
}

// goWorkModulePaths returns every use directive's cleaned, workspace-root-
// relative path declared in go.work bytes.
func goWorkModulePaths(data []byte) ([]string, error) {
	wf, err := modfile.ParseWork("go.work", data, nil)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(wf.Use))
	for _, u := range wf.Use {
		paths = append(paths, path.Clean(u.Path))
	}
	sort.Strings(paths)
	return paths, nil
}

// buildDriftReport projects a parsed Workspace (nil when go.work or a
// go.mod failed to parse) and its detected conflict, if any, into a
// shipwright.DriftReport.
func buildDriftReport(root, expectedVersion string, ws *Workspace, conflict *AmbiguousToolchainError) shipwright.DriftReport {
	report := shipwright.DriftReport{
		WorkspaceRoot:   root,
		ExpectedVersion: expectedVersion,
		Sources:         map[string]string{},
	}

	if ws != nil {
		if ws.HasGoWork {
			report.Sources["go.work"] = ws.GoWorkGo
		}
		if ws.HasGoVersion {
			report.Sources[".go-version"] = ws.GoVersion
		}
		for _, m := range ws.Modules {
			report.Modules = append(report.Modules, shipwright.ModuleVersion{
				Path:      m.Path,
				Go:        m.Go,
				Toolchain: m.Toolchain,
			})
		}
	}

	if conflict != nil {
		report.Conflict = shipwright.ConflictState{
			Ambiguous: true,
			Code:      conflict.Code,
			Sites:     conflict.Sites,
		}
	}

	return report
}
