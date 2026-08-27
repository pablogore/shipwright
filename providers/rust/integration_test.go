package rust_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/rust"
)

// TestRustBuilder_Build_RealEngine exercises RustBuilder.Build end to end
// against a real Dagger engine, mirroring providers/go's
// TestGoBuilder_Build_RealEngine. Per shipwright-testing-strategy, any test
// reaching a real Dagger container belongs at the integration level, never
// as a plain unit test, and MUST be guarded by testing.Short() so
// `go test -short ./...` stays fast and skips it cleanly.
func TestRustBuilder_Build_RealEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-container RustBuilder integration test in -short mode")
	}

	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := t.TempDir()
	cargoToml := `[package]
name = "capabilitiestest"
version = "0.1.0"
edition = "2021"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoToml), 0o644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create src directory: %v", err)
	}
	mainRs := "fn main() {}\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "src", "main.rs"), []byte(mainRs), 0o644); err != nil {
		t.Fatalf("failed to write main.rs: %v", err)
	}

	src := client.Host().Directory(tmpDir)
	builder := &rust.RustBuilder{
		Client:      client,
		Config:      shipwright.BuildConfig{BinaryName: "capabilitiestest"},
		RustVersion: "1.83.0",
	}

	out, err := builder.Build(ctx, src)
	if err != nil {
		t.Fatalf("RustBuilder.Build() error = %v, want nil", err)
	}
	if out == nil {
		t.Fatal("RustBuilder.Build() returned a nil Directory on success")
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

// TestRustBuilder_Build_RealEngine_CompileErrorIncludesStderr proves
// RustBuilder.Build's failure path now wraps *dagger.ExecError via
// wrapExecError instead of a bare `%w`, so a real compile failure's actual
// rustc diagnostics (only ever on stderr) reach the returned error instead
// of being swallowed by dagger.ExecError.Error()'s generic "process ...
// did not complete successfully" message.
func TestRustBuilder_Build_RealEngine_CompileErrorIncludesStderr(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-container RustBuilder integration test in -short mode")
	}

	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := t.TempDir()
	cargoToml := `[package]
name = "brokencrate"
version = "0.1.0"
edition = "2021"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoToml), 0o644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create src directory: %v", err)
	}
	// Deliberate syntax error: cargo build must fail with a real rustc
	// diagnostic on stderr.
	brokenMainRs := "fn main() { let x: i32 = \"not an int\"; }\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "src", "main.rs"), []byte(brokenMainRs), 0o644); err != nil {
		t.Fatalf("failed to write main.rs: %v", err)
	}

	src := client.Host().Directory(tmpDir)
	builder := &rust.RustBuilder{
		Client:      client,
		Config:      shipwright.BuildConfig{BinaryName: "brokencrate"},
		RustVersion: "1.83.0",
	}

	_, err = builder.Build(ctx, src)
	if err == nil {
		t.Fatal("RustBuilder.Build() error = nil, want error for a crate that fails to compile")
	}
	if !strings.Contains(err.Error(), "stderr:") {
		t.Fatalf("RustBuilder.Build() error = %v, want it to include wrapped stderr diagnostics", err)
	}
}

// TestRustBuilder_Build_RealEngine_RepeatedBuildReusesCache is a smoke test
// for the cargo registry/target Dagger cache volumes RustBuilder.Build now
// mounts: it proves a second build of the same source, against the same
// client (so the cache volumes persist across the two calls), still
// succeeds and produces the expected binary. It does NOT assert a timing
// improvement — Dagger cache-volume speedups aren't reliably measurable in
// a single test run — only that mounting WithMountedCache doesn't change
// Build's observable success/output behavior.
func TestRustBuilder_Build_RealEngine_RepeatedBuildReusesCache(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-container RustBuilder integration test in -short mode")
	}

	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := t.TempDir()
	cargoToml := `[package]
name = "cachetest"
version = "0.1.0"
edition = "2021"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoToml), 0o644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create src directory: %v", err)
	}
	mainRs := "fn main() {}\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "src", "main.rs"), []byte(mainRs), 0o644); err != nil {
		t.Fatalf("failed to write main.rs: %v", err)
	}

	src := client.Host().Directory(tmpDir)
	builder := &rust.RustBuilder{
		Client:      client,
		Config:      shipwright.BuildConfig{BinaryName: "cachetest"},
		RustVersion: "1.83.0",
	}

	for i := 0; i < 2; i++ {
		out, err := builder.Build(ctx, src)
		if err != nil {
			t.Fatalf("RustBuilder.Build() call #%d error = %v, want nil", i+1, err)
		}
		entries, err := out.Entries(ctx)
		if err != nil {
			t.Fatalf("failed to list build output directory entries on call #%d: %v", i+1, err)
		}
		found := false
		for _, e := range entries {
			if e == "cachetest" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("build output directory entries on call #%d = %v, want it to contain %q", i+1, entries, "cachetest")
		}
	}
}

// TestRustBuilder_Build_NilSource_RealClient covers the nil-source guard
// clause with a real, connected Dagger client — cheap even under a real
// engine because the guard returns before any container is built.
// Complements TestRustBuilder_Build_NilClient (pure unit test, no engine)
// so both branches of the guard chain have coverage.
func TestRustBuilder_Build_NilSource_RealClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-engine test in -short mode")
	}

	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	builder := &rust.RustBuilder{Client: client}

	_, err = builder.Build(ctx, nil)
	if err == nil {
		t.Fatal("RustBuilder.Build(nil source) error = nil, want error")
	}
}

// TestRustUnitTester_Test_RealEngine_PassesWithinThreshold exercises
// RustUnitTester.Test against a real Dagger engine, mirroring
// providers/go's TestGoUnitTester_Test_RealEngine_PassesWithinThreshold. No
// unit test can prove a container's toolchain/coverage wiring is correct;
// only a real engine run can.
func TestRustUnitTester_Test_RealEngine_PassesWithinThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-container RustUnitTester integration test in -short mode")
	}

	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := t.TempDir()
	cargoToml := `[package]
name = "capabilitiestest"
version = "0.1.0"
edition = "2021"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoToml), 0o644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create src directory: %v", err)
	}

	libRs := `pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_add() {
        assert_eq!(add(2, 3), 5);
    }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "src", "lib.rs"), []byte(libRs), 0o644); err != nil {
		t.Fatalf("failed to write lib.rs: %v", err)
	}

	src := client.Host().Directory(tmpDir)
	tester := &rust.RustUnitTester{
		Client: client,
		Config: shipwright.TestConfig{Coverage: 1},
	}

	out, err := tester.Test(ctx, src)
	if err != nil {
		t.Fatalf("RustUnitTester.Test() error = %v, want nil", err)
	}
	if out == nil {
		t.Fatal("RustUnitTester.Test() returned a nil File on success")
	}
}

// TestRustLinter_Test_RealEngine_ReportIncludesClippyDiagnostics proves
// RustLinter.Test's report now contains clippy's actual diagnostics.
// clippy, like rustc, writes its findings to stderr — capturing only Stdout
// (the pre-fix behavior) produced a report with no diagnostic detail at
// all, clean or not.
func TestRustLinter_Test_RealEngine_ReportIncludesClippyDiagnostics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-container RustLinter integration test in -short mode")
	}

	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := t.TempDir()
	cargoToml := `[package]
name = "clippytest"
version = "0.1.0"
edition = "2021"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoToml), 0o644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create src directory: %v", err)
	}
	mainRs := "fn main() {}\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "src", "main.rs"), []byte(mainRs), 0o644); err != nil {
		t.Fatalf("failed to write main.rs: %v", err)
	}

	src := client.Host().Directory(tmpDir)
	linter := &rust.RustLinter{Client: client, RustVersion: "1.83.0"}

	out, err := linter.Test(ctx, src)
	if err != nil {
		t.Fatalf("RustLinter.Test() error = %v, want nil", err)
	}
	contents, err := out.Contents(ctx)
	if err != nil {
		t.Fatalf("failed to read lint report contents: %v", err)
	}
	// clippy's compilation progress ("Compiling clippytest ...", "Finished
	// ...") is only ever written to stderr, never stdout — its presence in
	// the report proves stderr was actually captured.
	if !strings.Contains(contents, "Compiling") && !strings.Contains(contents, "Finished") {
		t.Fatalf("lint report contents = %q, want it to include clippy's stderr diagnostics", contents)
	}
}
