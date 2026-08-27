package rust_test

import (
	"context"
	"os"
	"path/filepath"
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
