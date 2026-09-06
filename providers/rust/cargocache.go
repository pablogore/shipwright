package rust

// cargoRegistryMountPath is the official rust:<version> image's CARGO_HOME
// registry cache directory. Shared by every capability in this package that
// invokes cargo (build, test, clippy, cargo-audit/cargo-tarpaulin install)
// via one common cache volume, so a downloaded crate is fetched once per
// workflow run instead of once per step.
const cargoRegistryMountPath = "/usr/local/cargo/registry"

// cargoRegistryCacheKey names the shared registry cache volume all four
// Rust capability containers mount at cargoRegistryMountPath.
const cargoRegistryCacheKey = "shipwright-cargo-registry"

// Target-directory cache keys, one per capability that actually compiles
// project code (cargo build/test/clippy). Kept distinct rather than shared:
// RustBuilder's default profile is "release" while RustUnitTester/RustLinter
// always build in a test/dev-like profile, so a shared target cache would
// otherwise ping-pong between incompatible incremental-compilation states
// on every run. RustVulnScanner has no target cache — cargo-audit only
// inspects Cargo.lock and never compiles the project itself.
const (
	rustBuilderTargetCacheKey    = "shipwright-rust-builder-target"
	rustUnitTesterTargetCacheKey = "shipwright-rust-unittester-target"
	rustLinterTargetCacheKey     = "shipwright-rust-linter-target"
	// rustIntegrationTesterTargetCacheKey is its own key, not shared with
	// rustUnitTesterTargetCacheKey: RustIntegrationTester's ManifestPath
	// typically points at a wholly separate Cargo workspace (e.g. ego-rs's
	// integration-tests/Cargo.toml, deliberately excluded from the main
	// workspace) with its own dependency graph and feature set
	// (crash-test-failpoint), so sharing a target dir would invalidate
	// incremental-compilation state between the two on every run.
	rustIntegrationTesterTargetCacheKey = "shipwright-rust-integrationtester-target"
)
