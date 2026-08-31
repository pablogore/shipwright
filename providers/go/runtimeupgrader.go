package golang

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"dagger.io/dagger"
	"golang.org/x/mod/modfile"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// runtimeUpgradeReportPath is where Upgrade writes its JSON report inside
// the returned Directory (design.md D-2).
const runtimeUpgradeReportPath = ".shipwright/runtime-upgrade-report.json"

// runtimeUpgradeContainerRoot is where the mutated workspace Directory is
// mounted for post-mutation validation (design.md D-7 steps 3-6): building
// a "golang:"+targetVersion container, running `go mod tidy` (when
// u.Tidy) and `go build ./...` (D-6 — go vet is explicitly rejected) per
// module, then exporting the container's own workdir back out as the
// returned Directory, carrying tidy's go.sum.
const runtimeUpgradeContainerRoot = "/workspace"

// runtimeValidationKind is recorded in UpgradeReport.Validation on success
// (design.md D-6): "go build ./..." is the only claim Upgrade proves, and
// the report says so explicitly rather than letting a consumer assume `go
// vet` also ran. A failed validation instead names the specific stage that
// failed via classifyValidationFailure, since "build" alone doesn't tell a
// caller whether the toolchain image, `go mod tidy`, or the build itself
// was the one that needed network access it didn't have.
const runtimeValidationKind = "build"

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
	// Tidy controls whether validateAndFinalize's container-based `go mod
	// tidy` step runs before `go build ./...` for each module (design.md
	// D-7, default true). `go build ./...` always runs regardless of
	// Tidy. This is the network-conditional half of validation: `go mod
	// tidy` (when Tidy is true) may reach the module proxy to resolve
	// dependency versions, same as `go build ./...` may on a cold module
	// cache.
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
// for a workspace, or a single go.mod; .go-version when present),
// validates the mutation in a "golang:"+targetVersion container via `go
// mod tidy` + `go build ./...` per module (design.md D-6/D-7, tasks.md
// Phase 4), and writes a shipwright.UpgradeReport to
// runtimeUpgradeReportPath in the returned Directory. Any detected
// ambiguity or path-escape aborts before the first WithNewFile: (nil,
// err), never a partially mutated Directory. A post-mutation validation
// failure also returns (nil, err) — a *ValidationError naming which module
// failed and which already succeeded — never a directory presented as
// upgraded (spec: "Post-mutation validation failure is not silently
// returned").
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
		return u.upgradeWorkspace(ctx, dir, input, ws, targetVersion, root)
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

	return u.validateAndFinalize(ctx, result, dir, root, targetVersion, []string{"."}, map[string][]byte{".": mutatedMod}, []shipwright.ModuleDrift{drift})
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
func (u *GoRuntimeUpgrader) upgradeWorkspace(ctx context.Context, dir daggerkit.DaggerDirectory, input WorkspaceInput, ws *Workspace, targetVersion, root string) (*dagger.Directory, error) {
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
	mutatedByPath := make(map[string][]byte, len(modulePaths))
	for _, modPath := range modulePaths {
		mutatedMod, err := mutateGoMod(input.Modules[modPath], targetVersion)
		if err != nil {
			return nil, err
		}
		mutatedByPath[modPath] = mutatedMod

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

	return u.validateAndFinalize(ctx, result, dir, root, targetVersion, modulePaths, mutatedByPath, drifts)
}

// ValidationError is returned by GoRuntimeUpgrader.Upgrade when
// post-mutation validation (design.md D-6: `go build ./...`, never `go
// vet`) fails for a module. It names Validation (the stage that was run),
// Failed (the module whose build failed), and Succeeded (every module
// already validated before the failure), so a caller never has to guess
// which module broke the workspace — spec: "the report names which module
// failed and which succeeded". Upgrade returns (nil, err) alongside it:
// no directory is ever presented as a successfully upgraded workspace
// (spec: "Post-mutation validation failure is not silently returned").
type ValidationError struct {
	Validation string
	Failed     string
	Succeeded  []string
	Err        error
}

func (e *ValidationError) Error() string {
	succeeded := "none"
	if len(e.Succeeded) > 0 {
		succeeded = strings.Join(e.Succeeded, ", ")
	}
	return fmt.Sprintf("runtimeupgrader: %s validation failed for module %q (already succeeded: %s): %v", e.Validation, e.Failed, succeeded, e.Err)
}

// Unwrap exposes the underlying container error (e.g. Dagger's own
// ExecError, carrying the failing command's stderr), matching the wrap
// convention every other capability in this package already uses.
func (e *ValidationError) Unwrap() error { return e.Err }

// classifyValidationFailure names which stage of the single per-module
// Sync failed: "tidy" for `go mod tidy` (only ever runs when u.Tidy is
// true), "build" for `go build ./...` (always runs), or "toolchain" for
// the "golang:"+targetVersion image pull (Container().From, unconditional
// — the container chain is lazy, so an image that can't be resolved only
// surfaces here, at Sync, without ever reaching a WithExec).
//
// The primary signal is Dagger's own *dagger.ExecError.Cmd field, checked
// via errors.As: it carries the failing command's argv directly, which is
// the only reliable signal against a live engine — this SDK version's own
// ExecError.Error() can be as bare as "exit code: 1 [traceparent:...]",
// with no command or output in the message at all. When err isn't a
// *dagger.ExecError (an image-pull failure never reaches a WithExec, so
// never produces one), the message text is checked instead, matching
// process-error output that does name the command — the shape both the
// mocked test fixtures and some Dagger/buildkit versions produce. An error
// matching neither signal never reached a WithExec, so it's classified as
// the toolchain stage.
func classifyValidationFailure(err error) string {
	var execErr *dagger.ExecError
	if errors.As(err, &execErr) {
		switch strings.Join(execErr.Cmd, " ") {
		case "go mod tidy":
			return "tidy"
		case "go build ./...":
			return "build"
		}
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "go mod tidy"):
		return "tidy"
	case strings.Contains(msg, "go build ./..."):
		return "build"
	default:
		return "toolchain"
	}
}

// validateAndFinalize is design.md D-7 steps 3-6, shared by both the
// single-module and go.work workspace mutation paths: mount dir (already
// fully mutated by the caller) into a "golang:"+targetVersion container,
// run `go mod tidy` (when u.Tidy) then `go build ./...` per module in
// modulePaths order, and export the container's own workdir as the
// returned Directory once every module has validated cleanly. Sync runs
// once per module (not once for the whole chain) specifically so a
// failure names the exact module that broke and every module already
// proven to build (tasks.md 4.1/4.2) — a single trailing Sync could only
// report "something failed somewhere". mutatedByPath carries each
// module's already-mutated go.mod bytes (no extra read needed) as the
// "before" side of the per-module require-list delta recorded in the
// final report (design.md D-7's require-list delta fields, tasks.md 4.4).
// originalDir is the untouched pre-mutation source Directory (Upgrade
// itself never writes go.sum), used by recordModuleDelta as the "before"
// side of each module's go.sum byte comparison.
func (u *GoRuntimeUpgrader) validateAndFinalize(ctx context.Context, dir, originalDir daggerkit.DaggerDirectory, root, targetVersion string, modulePaths []string, mutatedByPath map[string][]byte, drifts []shipwright.ModuleDrift) (*dagger.Directory, error) {
	driftByPath := make(map[string]*shipwright.ModuleDrift, len(drifts))
	for i := range drifts {
		driftByPath[drifts[i].Path] = &drifts[i]
	}

	container := u.Client.Container().From("golang:"+targetVersion).WithMountedDirectory(runtimeUpgradeContainerRoot, dir)

	succeeded := make([]string, 0, len(modulePaths))
	for _, modPath := range modulePaths {
		workdir := moduleContainerWorkdir(modPath)
		container = container.WithWorkdir(workdir)
		if u.Tidy {
			container = container.WithExec([]string{"go", "mod", "tidy"}, daggerkit.DaggerContainerWithExecOpts{})
		}
		container = container.WithExec([]string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{})

		synced, err := container.Sync(ctx)
		if err != nil {
			return nil, &ValidationError{Validation: classifyValidationFailure(err), Failed: modPath, Succeeded: succeeded, Err: err}
		}
		container = synced

		if err := recordModuleDelta(ctx, container, originalDir, workdir, modPath, mutatedByPath[modPath], driftByPath[modPath]); err != nil {
			return nil, err
		}

		succeeded = append(succeeded, modPath)
	}

	report := shipwright.UpgradeReport{
		WorkspaceRoot: root,
		TargetVersion: targetVersion,
		Validation:    runtimeValidationKind,
		Modules:       drifts,
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("runtimeupgrader: failed to marshal upgrade report: %w", err)
	}

	finalDir := container.Directory(runtimeUpgradeContainerRoot)
	return finalDir.WithNewFile(runtimeUpgradeReportPath, string(data)).GetRealDirectory(), nil
}

// moduleContainerWorkdir returns modPath's absolute path inside the
// validation container: the mount root itself for the single-module "."
// case, or a subdirectory of it for a go.work-referenced module.
func moduleContainerWorkdir(modPath string) string {
	if modPath == "." {
		return runtimeUpgradeContainerRoot
	}
	return path.Join(runtimeUpgradeContainerRoot, modPath)
}

// recordModuleDelta fills in drift's AddedModules/RemovedModules fields
// (design.md D-7) by diffing beforeModBytes' require directives (already
// in memory — the bytes Upgrade itself wrote into the container, before
// `go mod tidy` ran) against workdir/go.mod's require directives as `go
// mod tidy` left them inside the container. Reports only the added/
// removed require module paths — never the raw go.sum diff, which can run
// to thousands of unreviewable lines (design.md D-7).
//
// GoSumChanged is computed independently of that require-list delta: a
// literal byte comparison of modPath's go.sum content in originalDir (the
// untouched pre-mutation source — Upgrade itself never writes go.sum)
// against workdir/go.sum as `go mod tidy` left it inside the container.
// A require-path diff alone misses `go mod tidy` bumping an *existing*
// dependency's version, which changes go.sum's hash entries (and the
// require directive's version) without changing the require path set —
// the byte comparison catches that case where the old path-diff proxy
// silently reported no change. go.sum absent on either side (e.g. a
// module with zero external dependencies) is treated as empty for
// comparison purposes, never a hard failure.
func recordModuleDelta(ctx context.Context, container daggerkit.DaggerContainer, originalDir daggerkit.DaggerDirectory, workdir, modPath string, beforeModBytes []byte, drift *shipwright.ModuleDrift) error {
	if drift == nil {
		return fmt.Errorf("runtimeupgrader: no drift entry recorded for module %s", modPath)
	}

	before, err := requireModulePaths(beforeModBytes, path.Join(modPath, "go.mod"))
	if err != nil {
		return fmt.Errorf("runtimeupgrader: failed to parse pre-validation %s/go.mod: %w", modPath, err)
	}

	afterContents, err := container.File(path.Join(workdir, "go.mod")).Contents(ctx)
	if err != nil {
		return fmt.Errorf("runtimeupgrader: failed to read post-validation %s/go.mod: %w", modPath, err)
	}
	after, err := requireModulePaths([]byte(afterContents), path.Join(modPath, "go.mod"))
	if err != nil {
		return fmt.Errorf("runtimeupgrader: failed to parse post-validation %s/go.mod: %w", modPath, err)
	}

	var added, removed []string
	for p := range after {
		if !before[p] {
			added = append(added, p)
		}
	}
	for p := range before {
		if !after[p] {
			removed = append(removed, p)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)

	drift.AddedModules = added
	drift.RemovedModules = removed

	beforeSum, err := readOptionalFileContents(ctx, originalDir.File(path.Join(modPath, "go.sum")))
	if err != nil {
		return fmt.Errorf("runtimeupgrader: failed to read pre-validation %s/go.sum: %w", modPath, err)
	}
	afterSum, err := readOptionalFileContents(ctx, container.File(path.Join(workdir, "go.sum")))
	if err != nil {
		return fmt.Errorf("runtimeupgrader: failed to read post-validation %s/go.sum: %w", modPath, err)
	}
	drift.GoSumChanged = !bytes.Equal(beforeSum, afterSum)

	return nil
}

// readOptionalFileContents reads f's full content, treating a "file does
// not exist" error as a legitimate absent state ((nil, nil), equivalent to
// empty for a later byte comparison) rather than a hard failure — go.sum
// is not guaranteed to exist either before or after `go mod tidy` runs
// (e.g. a module with zero external dependencies never has one). Any
// other error is propagated unchanged.
func readOptionalFileContents(ctx context.Context, f daggerkit.DaggerFile) ([]byte, error) {
	contents, err := f.Contents(ctx)
	if err != nil {
		if isFileNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return []byte(contents), nil
}

// isFileNotFoundError reports whether err represents a missing file rather
// than some other read failure. Dagger's SDK has no typed "not exist"
// sentinel for File.Contents (unlike os.IsNotExist), so this matches the
// engine's own "no such file or directory" message — the same substring a
// missing-path read surfaces through buildkit.
func isFileNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such file or directory")
}

// requireModulePaths parses modBytes and returns the set of module paths
// named by its require directives.
func requireModulePaths(modBytes []byte, fileName string) (map[string]bool, error) {
	mf, err := modfile.Parse(fileName, modBytes, nil)
	if err != nil {
		return nil, err
	}
	paths := make(map[string]bool, len(mf.Require))
	for _, r := range mf.Require {
		paths[r.Mod.Path] = true
	}
	return paths, nil
}
