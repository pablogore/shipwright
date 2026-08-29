package golang

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// runtimeUpgradeReportPath is where Upgrade writes its JSON report inside
// the returned Directory (design.md D-2).
const runtimeUpgradeReportPath = ".shipwright/runtime-upgrade-report.json"

// GoRuntimeUpgrader mutates a Go workspace's declarative toolchain-version
// sources (go.mod's go/toolchain directives, go.work's own go directive,
// .go-version) to TargetVersion and returns the mutated Directory.
// Discovery-driven (design.md D-4/D-9): it only mutates a declarative
// location that already exists in source, never fabricating one that
// wasn't there. Because *dagger.Directory is an immutable value, analysis
// (parse + A1-A7 conflict/path-escape detection) always completes before
// the first WithNewFile call, so a failed run returns (nil, err) — never a
// partially mutated Directory.
//
// A workspace with a go.work at its root traverses every use'd module
// (tasks.md Phase 3, design.md D-7's mutation loop) via mutateGoWork and a
// per-module mutateGoMod call, recording each module's outcome in the
// returned report's Modules slice. A workspace with only a go.mod at its
// root (no go.work) uses the single-module path below.
type GoRuntimeUpgrader struct {
	// Client is the Dagger client. Kept for parity with every other
	// capability provider in this package, and nil-checked the same way,
	// even though Phase 2's Upgrade never drives a container.
	Client daggerkit.DaggerClient
	// WorkspaceRoot is recorded in the report as the workspace root the
	// caller intends (default "."). The source Directory passed to
	// Upgrade is always read at its own top level; a caller using a
	// non-default WorkspaceRoot is responsible for scoping source to it
	// before calling Upgrade.
	WorkspaceRoot string
	// Tidy controls whether a later phase's container-based `go mod tidy`
	// + `go build ./...` validation step runs after mutation (design.md
	// D-7, default true). Bound from `with` at registration now so the
	// manifest schema is stable across phases; Upgrade does not yet read
	// it — Phase 2 never drives a container (tasks.md Phase 4 wires the
	// tidy/build sequencing this field controls).
	Tidy bool
	// AllowDowngrade permits targetVersion to be lower than the
	// workspace's current version (design.md D-5, A4). Downgrades are
	// rejected by default.
	AllowDowngrade bool
}

// Compile-time conformance assertion: GoRuntimeUpgrader must satisfy
// Layer 1's RuntimeUpgrader interface (pkg/shipwright).
var _ shipwright.RuntimeUpgrader = (*GoRuntimeUpgrader)(nil)

// Upgrade reads source's tier-1 version sources, validates targetVersion
// and the workspace's own internal consistency against the A1-A7 ambiguity
// rules (design.md D-5, plus the A7 path-escape guard), then mutates every
// present declarative location (go.work and every use'd module's go.mod
// for a workspace, or a single go.mod; .go-version when present) and
// writes a shipwright.UpgradeReport to runtimeUpgradeReportPath in the
// returned Directory. Any detected ambiguity or path-escape aborts before
// the first WithNewFile: (nil, err), never a partially mutated Directory.
func (u *GoRuntimeUpgrader) Upgrade(ctx context.Context, source *dagger.Directory, targetVersion string) (*dagger.Directory, error) {
	if u.Client == nil {
		return nil, errors.New("runtimeupgrader: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("runtimeupgrader: source directory is nil")
	}

	root := u.WorkspaceRoot
	if root == "" {
		root = "."
	}

	dir := newDaggerDirectory(source)

	// Threat matrix: path traversal via go.work use. Reads only go.work's
	// own bytes (never a module's go.mod) and validates every use
	// directive's path before anything else in the workspace is read or
	// mutated (tasks.md 3.2/3.3).
	if err := validateWorkspaceModulePaths(ctx, dir); err != nil {
		return nil, err
	}

	input, err := gatherWorkspaceInput(ctx, dir)
	if err != nil {
		return nil, err
	}

	ws, err := parseWorkspace(input)
	if err != nil {
		return nil, err
	}

	if conflict := detectConflicts(ws, ConflictOptions{TargetVersion: targetVersion, AllowDowngrade: u.AllowDowngrade}); conflict != nil {
		return nil, conflict
	}

	if len(input.GoWork) > 0 {
		return u.upgradeWorkspace(dir, input, ws, targetVersion, root)
	}

	modBytes, ok := input.Modules["."]
	if !ok {
		// Unreachable in practice: detectConflicts already returns A6 when
		// no go.work and no module is present. Defensive fail-closed only.
		return nil, &AmbiguousToolchainError{Code: CodeA6, Sites: []string{"neither go.work nor go.mod found at the workspace root"}}
	}

	mutatedMod, err := mutateGoMod(modBytes, targetVersion)
	if err != nil {
		return nil, err
	}

	drift := shipwright.ModuleDrift{
		Path:      ".",
		UpdatedGo: targetVersion,
	}
	if len(ws.Modules) == 1 {
		drift.PreviousGo = ws.Modules[0].Go
		drift.PreviousToolchain = ws.Modules[0].Toolchain
		if ws.Modules[0].Toolchain != "" {
			drift.UpdatedToolchain = "go" + targetVersion
		}
	}

	result := dir.WithNewFile("go.mod", string(mutatedMod))

	if ws.HasGoVersion {
		result = result.WithNewFile(".go-version", string(mutateGoVersion(targetVersion)))
	}

	return writeUpgradeReport(result, root, targetVersion, []shipwright.ModuleDrift{drift})
}

// validateWorkspaceModulePaths guards design.md's Threat Matrix "Path
// traversal via go.work use" row. It reads the workspace's entries and,
// if a go.work is present at the root, its raw bytes only — never any
// module's go.mod — and rejects the whole Upgrade call the moment any use
// directive resolves outside the workspace root (CodeA7), before
// gatherWorkspaceInput or any mutation ever runs. A workspace with no
// go.work at its root is a no-op here (single-module path, unaffected). A
// malformed go.work is left for parseWorkspace to report as A5 once it
// parses the same bytes below — this guard only classifies path
// traversal, not syntax.
func validateWorkspaceModulePaths(ctx context.Context, dir daggerkit.DaggerDirectory) error {
	entries, err := dir.Entries(ctx)
	if err != nil {
		return fmt.Errorf("runtimeupgrader: failed to list workspace entries: %w", err)
	}

	hasGoWork := false
	for _, e := range entries {
		if e == "go.work" {
			hasGoWork = true
			break
		}
	}
	if !hasGoWork {
		return nil
	}

	content, err := readContents(ctx, dir, "go.work")
	if err != nil {
		return fmt.Errorf("runtimeupgrader: failed to read go.work: %w", err)
	}

	modulePaths, err := goWorkModulePaths([]byte(content))
	if err != nil {
		// Malformed go.work is reported as an A5 conflict once
		// parseWorkspace parses it below; nothing to validate here.
		return nil
	}

	if conflict := validateModulePaths(modulePaths); conflict != nil {
		return conflict
	}
	return nil
}

// upgradeWorkspace mutates a go.work workspace's own go directive plus
// every use'd module's go.mod and, when present, .go-version — recording
// a per-module outcome in the report's Modules slice (tasks.md 3.4).
// Called only after validateWorkspaceModulePaths and detectConflicts have
// both already passed: every module path here is known to stay within the
// workspace root, and the workspace's tier-1 sources are known consistent.
func (u *GoRuntimeUpgrader) upgradeWorkspace(dir daggerkit.DaggerDirectory, input WorkspaceInput, ws *Workspace, targetVersion, root string) (*dagger.Directory, error) {
	mutatedWork, err := mutateGoWork(input.GoWork, targetVersion)
	if err != nil {
		return nil, err
	}
	result := dir.WithNewFile("go.work", string(mutatedWork))

	if ws.HasGoVersion {
		result = result.WithNewFile(".go-version", string(mutateGoVersion(targetVersion)))
	}

	modulePaths := make([]string, 0, len(input.Modules))
	for p := range input.Modules {
		modulePaths = append(modulePaths, p)
	}
	sort.Strings(modulePaths)

	previous := make(map[string]ModuleFile, len(ws.Modules))
	for _, m := range ws.Modules {
		previous[m.Path] = m
	}

	drifts := make([]shipwright.ModuleDrift, 0, len(modulePaths))
	for _, modPath := range modulePaths {
		mutatedMod, err := mutateGoMod(input.Modules[modPath], targetVersion)
		if err != nil {
			return nil, err
		}

		drift := shipwright.ModuleDrift{Path: modPath, UpdatedGo: targetVersion}
		if mf, ok := previous[modPath]; ok {
			drift.PreviousGo = mf.Go
			drift.PreviousToolchain = mf.Toolchain
			if mf.Toolchain != "" {
				drift.UpdatedToolchain = "go" + targetVersion
			}
		}
		drifts = append(drifts, drift)

		result = result.WithNewFile(path.Join(modPath, "go.mod"), string(mutatedMod))
	}

	return writeUpgradeReport(result, root, targetVersion, drifts)
}

// writeUpgradeReport marshals a shipwright.UpgradeReport and writes it to
// runtimeUpgradeReportPath inside dir, shared by both the single-module and
// go.work workspace mutation paths.
func writeUpgradeReport(dir daggerkit.DaggerDirectory, root, targetVersion string, modules []shipwright.ModuleDrift) (*dagger.Directory, error) {
	report := shipwright.UpgradeReport{
		WorkspaceRoot: root,
		TargetVersion: targetVersion,
		Modules:       modules,
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("runtimeupgrader: failed to marshal upgrade report: %w", err)
	}
	return dir.WithNewFile(runtimeUpgradeReportPath, string(data)).GetRealDirectory(), nil
}
