package golang_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go"
)

// TestGoBuilder_Build_RealEngine exercises GoBuilder.Build end to end
// against a real Dagger engine — the -short-guarded real-container case
// required by tasks.md's Phase 3 work-unit table (row 3). Per
// shipwright-testing-strategy, any test reaching a real Dagger container
// belongs at the integration level, never as a plain unit test, and MUST
// be guarded by testing.Short() so `go test -short ./...` stays fast and
// skips it cleanly.
func TestGoBuilder_Build_RealEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-container GoBuilder integration test in -short mode")
	}

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
		Client: client,
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
	if testing.Short() {
		t.Skip("skipping real-engine test in -short mode")
	}

	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	builder := &golang.GoBuilder{Client: client}

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
	if testing.Short() {
		t.Skip("skipping real-container GoUnitTester integration test in -short mode")
	}

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
		Client: client,
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
