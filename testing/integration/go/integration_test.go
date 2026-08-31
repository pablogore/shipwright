//go:build integration

package golang_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// TestGoBuilder_Build_RealEngine exercises GoBuilder.Build end to end
// against a real Dagger engine — the integration-tagged real-container case
// required by tasks.md's Phase 3 work-unit table (row 3). Per
// shipwright-testing-strategy, any test reaching a real Dagger container
// belongs at the integration level, never as a plain unit test, and MUST
// be guarded by the `integration` build tag so `go test ./...` stays fast
// and skips it cleanly.
func TestGoBuilder_Build_RealEngine(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := t.TempDir()
	mainGo := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	goMod := "module capabilitiestest\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	src := client.Host().Directory(tmpDir)
	builder := &golang.GoBuilder{
		Client: daggerkit.NewDaggerAdapter(client),
		Config: shipwright.BuildConfig{GoVersion: "1.26.1", BinaryName: "capabilitiestest"},
	}

	out, err := builder.Build(ctx, src)
	if err != nil {
		t.Fatalf("GoBuilder.Build() error = %v, want nil", err)
	}
	if out == nil {
		t.Fatal("GoBuilder.Build() returned a nil Directory on success")
	}

	entries, err := out.Entries(ctx)
	if err != nil {
		t.Fatalf("failed to list build output directory entries: %v", err)
	}

	found := false
	for _, e := range entries {
		if e == "capabilitiestest" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("build output directory entries = %v, want it to contain %q", entries, "capabilitiestest")
	}
}

// TestGoBuilder_Build_NilSource_RealClient covers the nil-source guard
// clause with a real, connected Dagger client — cheap even under a real
// engine because the guard returns before any container is built.
// Complements TestGoBuilder_Build_NilClient (pure unit test, no engine) so
// both branches of the guard chain have coverage.
func TestGoBuilder_Build_NilSource_RealClient(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	builder := &golang.GoBuilder{Client: daggerkit.NewDaggerAdapter(client)}

	_, err = builder.Build(ctx, nil)
	if err == nil {
		t.Fatal("GoBuilder.Build(nil source) error = nil, want error")
	}
}

// TestGoUnitTester_Test_RealEngine_PassesWithinThreshold exercises
// GoUnitTester.Test against a real Dagger engine and is a deliberate
// regression guard, not just coverage: an earlier version of this
// implementation set CGO_ENABLED=0 while running `go test -race`
// (mirroring the legacy pipeline's own — never actually exercised —
// internal/pipelines/shared.RunTestsWithCoverage), which fails closed with
// "-race requires cgo" on every real invocation. No unit test can prove a
// container's env vars are wired correctly; only a real engine run can.
func TestGoUnitTester_Test_RealEngine_PassesWithinThreshold(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := t.TempDir()
	mainGo := `package main

func Add(a, b int) int {
	return a + b
}

func main() {}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	mainTestGo := `package main

import "testing"

func TestAdd(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Fatal("Add(2, 3) != 5")
	}
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(mainTestGo), 0o644); err != nil {
		t.Fatalf("failed to write main_test.go: %v", err)
	}

	// "go 1.21", well below defaultGoVersion's golang:1.25.5 toolchain
	// image, so this fixture never hits a "go.mod requires go >= X"
	// mismatch regardless of which default version this package targets.
	goMod := "module capabilitiestest\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	src := client.Host().Directory(tmpDir)
	tester := &golang.GoUnitTester{
		Client: daggerkit.NewDaggerAdapter(client),
		Config: shipwright.TestConfig{Coverage: 1},
	}

	out, err := tester.Test(ctx, src)
	if err != nil {
		t.Fatalf("GoUnitTester.Test() error = %v, want nil", err)
	}
	if out == nil {
		t.Fatal("GoUnitTester.Test() returned a nil File on success")
	}
}

// TestContainerPublisher_Publish_BinaryNameMismatch_RealEngine covers PR
// #176 review finding #7: ContainerPublisher.Publish's Config.BinaryName
// and the paired Builder step's BuildConfig.BinaryName are independently
// configured manifest fields with no cross-validation. Before this fix, a
// mismatch surfaced only as chmod's opaque "no such file or directory";
// Publish must now fail with an actionable message naming the missing
// path.
func TestContainerPublisher_Publish_BinaryNameMismatch_RealEngine(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "my-service"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to write fixture binary: %v", err)
	}

	build := client.Host().Directory(tmpDir)
	publisher := &golang.ContainerPublisher{
		Client: daggerkit.NewDaggerAdapter(client),
		// Deliberately omitted: BinaryName. Publish falls back to "app",
		// but the build directory only contains "my-service" (as a real
		// GoBuilder run configured with a non-default binaryName would
		// produce), reproducing the cross-field mismatch.
	}

	_, err = publisher.Publish(ctx, build, "ghcr.io/acme/api:v1", nil)
	if err == nil {
		t.Fatal("ContainerPublisher.Publish() error = nil, want error for a binaryName mismatch")
	}
	if !strings.Contains(err.Error(), "expected binary at") || !strings.Contains(err.Error(), "not found in container") {
		t.Fatalf("ContainerPublisher.Publish() error = %v, want it to name the missing entrypoint path", err)
	}
}

// TestGoRuntimeUpgrader_Upgrade_Workspace_RealEngine is the integration-
// tagged real-engine variant tasks.md 3.5 requires: a real, connected
// Dagger client reading and mutating a go.work multi-module workspace end
// to end, proving daggerkit's real adapter — not the mocks every other
// runtimeupgrader_test.go case uses — round-trips Entries/File/WithNewFile
// correctly for the Phase 3 traversal loop (go.work itself, every use'd
// module's go.mod, and .go-version). Its post-mutation validation step
// also pulls a real "golang:"+targetVersion image and runs `go build
// ./...` (and `go mod tidy` when Tidy is set) inside it, so this test is
// also the proof that runtime-upgrade is NOT offline/hermetic: it requires
// a reachable Dagger engine and, through it, network access to a
// container registry and (when Tidy is set) a module proxy — unlike
// runtime-inspect, which never needs either.
func TestGoRuntimeUpgrader_Upgrade_Workspace_RealEngine(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := writeWorkspaceFixture(t)

	src := client.Host().Directory(tmpDir)
	upgrader := &golang.GoRuntimeUpgrader{Client: daggerkit.NewDaggerAdapter(client)}

	out, err := upgrader.Upgrade(ctx, src, "1.27.0")
	if err != nil {
		t.Fatalf("GoRuntimeUpgrader.Upgrade() error = %v, want nil", err)
	}
	if out == nil {
		t.Fatal("GoRuntimeUpgrader.Upgrade() returned a nil Directory on success")
	}

	assertWorkspaceUpgraded(ctx, t, daggerkit.NewDaggerDirectoryAdapter(out))
}

// writeWorkspaceFixture writes a minimal go.work workspace (two modules
// plus a root .go-version) to a fresh temp directory and returns its path.
func writeWorkspaceFixture(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	goWork := "go 1.26.7\n\nuse (\n\t./modA\n\t./modB\n)\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.work"), []byte(goWork), 0o644); err != nil {
		t.Fatalf("failed to write go.work: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".go-version"), []byte("1.26.7\n"), 0o644); err != nil {
		t.Fatalf("failed to write .go-version: %v", err)
	}

	for _, mod := range []string{"modA", "modB"} {
		modDir := filepath.Join(tmpDir, mod)
		if err := os.MkdirAll(modDir, 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", mod, err)
		}
		goMod := "module example.com/workspacetest/" + mod + "\n\ngo 1.26.7\n"
		if err := os.WriteFile(filepath.Join(modDir, "go.mod"), []byte(goMod), 0o644); err != nil {
			t.Fatalf("failed to write %s/go.mod: %v", mod, err)
		}
	}

	return tmpDir
}

// assertWorkspaceUpgraded reads outDir's mutated go.work, .go-version,
// every module's go.mod, and the upgrade report back through daggerkit,
// proving Upgrade's multi-module traversal loop actually ran against a
// real Dagger engine (mocks cover this logic elsewhere; this is the
// engine-fidelity proof tasks.md 3.5 requires).
func assertWorkspaceUpgraded(ctx context.Context, t *testing.T, outDir daggerkit.DaggerDirectory) {
	t.Helper()

	workContents, err := outDir.File("go.work").Contents(ctx)
	if err != nil {
		t.Fatalf("failed to read mutated go.work: %v", err)
	}
	if !strings.Contains(workContents, "go 1.27.0") {
		t.Fatalf("mutated go.work = %q, want it to contain %q", workContents, "go 1.27.0")
	}

	versionContents, err := outDir.File(".go-version").Contents(ctx)
	if err != nil {
		t.Fatalf("failed to read mutated .go-version: %v", err)
	}
	if strings.TrimSpace(versionContents) != "1.27.0" {
		t.Fatalf("mutated .go-version = %q, want %q", versionContents, "1.27.0")
	}

	for _, mod := range []string{"modA", "modB"} {
		modContents, err := outDir.File(mod + "/go.mod").Contents(ctx)
		if err != nil {
			t.Fatalf("failed to read mutated %s/go.mod: %v", mod, err)
		}
		if !strings.Contains(modContents, "go 1.27.0") {
			t.Fatalf("mutated %s/go.mod = %q, want it to contain %q", mod, modContents, "go 1.27.0")
		}
	}

	reportContents, err := outDir.File(".shipwright/runtime-upgrade-report.json").Contents(ctx)
	if err != nil {
		t.Fatalf("failed to read runtime-upgrade-report.json: %v", err)
	}
	var report shipwright.UpgradeReport
	if err := json.Unmarshal([]byte(reportContents), &report); err != nil {
		t.Fatalf("failed to unmarshal runtime-upgrade-report.json: %v\n%s", err, reportContents)
	}
	if report.TargetVersion != "1.27.0" {
		t.Fatalf("report.TargetVersion = %q, want %q", report.TargetVersion, "1.27.0")
	}
	if report.Validation != "build" {
		t.Fatalf("report.Validation = %q, want %q", report.Validation, "build")
	}
	if len(report.Modules) != 2 {
		t.Fatalf("report.Modules = %v, want exactly 2 entries", report.Modules)
	}
}

// TestGoRuntimeUpgrader_Upgrade_ValidationFailure_RealEngine is tasks.md
// 4.6's integration test: a real, connected Dagger engine actually runs
// `go build ./...` inside a "golang:"+targetVersion container after
// mutating the workspace's toolchain directives (design.md D-6/D-7). A
// workspace whose source compiles cleanly cannot, on its own, distinguish
// "go build ran and passed" from "go build never ran at all" — only a
// fixture whose source is guaranteed to fail `go build ./...` can prove the
// validation step actually executes against a real engine, per spec:
// "Post-mutation validation failure is not silently returned".
func TestGoRuntimeUpgrader_Upgrade_ValidationFailure_RealEngine(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := t.TempDir()
	brokenGo := `package main

func main() {
	undefinedSymbol()
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(brokenGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	goMod := "module example.com/upgradebrokentest\n\ngo 1.26.7\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	src := client.Host().Directory(tmpDir)
	upgrader := &golang.GoRuntimeUpgrader{Client: daggerkit.NewDaggerAdapter(client)}

	out, err := upgrader.Upgrade(ctx, src, "1.27.0")
	if err == nil {
		t.Fatal("GoRuntimeUpgrader.Upgrade() error = nil, want a post-mutation build validation failure")
	}
	if out != nil {
		t.Fatalf("GoRuntimeUpgrader.Upgrade() directory = %v, want nil on validation failure", out)
	}

	var validationErr *golang.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("GoRuntimeUpgrader.Upgrade() error = %v, want a *golang.ValidationError", err)
	}
	if validationErr.Validation != "build" {
		t.Fatalf("ValidationError.Validation = %q, want %q", validationErr.Validation, "build")
	}
	if validationErr.Failed != "." {
		t.Fatalf("ValidationError.Failed = %q, want %q", validationErr.Failed, ".")
	}
}
