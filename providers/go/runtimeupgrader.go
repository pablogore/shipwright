package golang

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// runtimeUpgradeReportPath is where Upgrade writes its JSON report inside
// the returned Directory (design.md D-2).
const runtimeUpgradeReportPath = ".shipwright/runtime-upgrade-report.json"

// GoRuntimeUpgrader mutates a single-module Go workspace's declarative
// toolchain-version sources (go.mod's go/toolchain directives,
// .go-version) to TargetVersion and returns the mutated Directory.
// Discovery-driven (design.md D-4/D-9): it only mutates a declarative
// location that already exists in source, never fabricating one that
// wasn't there. Because *dagger.Directory is an immutable value, analysis
// (parse + A1-A6 conflict detection) always completes before the first
// WithNewFile call, so a failed run returns (nil, err) — never a partially
// mutated Directory.
//
// Phase 2 scope: single-module go.mod path only. A workspace with a
// go.work at its root is explicitly rejected — go.work-referenced
// multi-module traversal (mutateGoWork wired into this flow) is a later
// work unit (tasks.md Phase 3).
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
// and the workspace's own internal consistency against the A1-A6 ambiguity
// rules (design.md D-5), then mutates every present declarative location
// (go.mod, and .go-version when present) and writes a
// shipwright.UpgradeReport to runtimeUpgradeReportPath in the returned
// Directory. Any detected ambiguity, or an unsupported go.work workspace,
// aborts before the first WithNewFile: (nil, err), never a partially
// mutated Directory.
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

	input, err := gatherWorkspaceInput(ctx, dir)
	if err != nil {
		return nil, err
	}

	if len(input.GoWork) > 0 {
		return nil, errors.New("runtimeupgrader: workspace go.work multi-module upgrade is not yet supported")
	}

	ws, err := parseWorkspace(input)
	if err != nil {
		return nil, err
	}

	if conflict := detectConflicts(ws, ConflictOptions{TargetVersion: targetVersion, AllowDowngrade: u.AllowDowngrade}); conflict != nil {
		return nil, conflict
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

	report := shipwright.UpgradeReport{
		WorkspaceRoot: root,
		TargetVersion: targetVersion,
		Modules:       []shipwright.ModuleDrift{drift},
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("runtimeupgrader: failed to marshal upgrade report: %w", err)
	}
	result = result.WithNewFile(runtimeUpgradeReportPath, string(data))

	return result.GetRealDirectory(), nil
}
