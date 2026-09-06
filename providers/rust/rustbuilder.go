// Package rust provides Shipwright's Rust-toolchain capability
// implementations as their own standalone Go module
// (github.com/pablogore/shipwright/providers/rust), implementing
// Shipwright's Layer 1 capability interfaces (pkg/shipwright) and nothing
// else. Every type in this flat package implements exactly one capability
// and carries no shared "stack" identity, mirroring providers/go's own
// design.md D-F rationale: a preset registry keyed by a stack name, or a
// nested rustservice/ subdirectory, would make the path itself a bundling
// identity. See naming_test.go for the enforcing golden test.
//
// This package imports nothing from internal/**, enforced by
// internalimport_test.go — the public contract (pkg/shipwright) is
// sufficient on its own to implement every capability here, from outside
// the core module.
package rust

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
)

// defaultRustVersion is the Rust toolchain (rustc/cargo) version used
// whenever a caller leaves the toolchain version unspecified. Kept in step
// with a recent stable Rust release, mirroring providers/go's
// defaultGoVersion convention.
//
// 1.91.0, not an older release: CI observed cargo-tarpaulin==0.37.2
// (RustUnitTester's pinned coverage tool) failing to build twice as this
// value was raised — first under 1.83.0 ("feature `edition2024` is
// required", stabilized in 1.85.0), then again under 1.85.0 itself
// ("rustc 1.85.0 is not supported by the following packages:
// cargo-platform@0.3.3 requires rustc 1.91", the highest MSRV among
// tarpaulin's own locked transitive dependencies). 1.91.0 is that highest
// observed requirement, not a guess.
const defaultRustVersion = "1.91.0"

// defaultBinaryName mirrors providers/go's defaultBinaryName default output
// binary name.
const defaultBinaryName = "app"

// defaultCargoProfile is the cargo build profile used whenever a caller
// leaves BuildConfig.BuildMode unspecified. "release" mirrors the legacy
// go-service pipeline's own default of always producing an optimized
// binary.
const defaultCargoProfile = "release"

// RustBuilder builds a Rust source Directory (a Cargo package or workspace)
// into a Directory containing the compiled binary. Structural mirror of
// providers/go's GoBuilder, adapted for cargo.
//
// Behavioral judgment call, same as GoBuilder: RustBuilder always performs
// the binary-compile path; producing and publishing a container image from
// the resulting Directory is ContainerPublisher's job (Artifactor.Publish).
//nolint:revive // stutters with package rust by design: this is a deliberate structural mirror of providers/go's GoBuilder (see doc comment above), and every rust.Rust* type follows the same cross-provider naming symmetry
type RustBuilder struct {
	// Client is the Dagger client used to construct the build container.
	Client daggerkit.DaggerClient
	// Config configures the output binary name (BinaryName) and cargo
	// build profile (BuildMode, e.g. "release", "debug", or a custom
	// named profile).
	Config shipwright.BuildConfig
	// RustVersion selects the Rust toolchain image tag. Kept as its own
	// field rather than overloading Config.GoVersion, because
	// shipwright.BuildConfig has no Rust-specific version field
	// (pkg/shipwright/config.go) — the same reasoning providers/go's
	// GoUnitTester applies to its own GoVersion field. Defaults to
	// defaultRustVersion when left empty.
	RustVersion string
	// ManifestPath selects a Cargo.toml other than the source root's own,
	// mirroring `cargo build --manifest-path`. Needed to build a single
	// member out of a large workspace (e.g. ego-rs's
	// examples/reference-app/Cargo.toml) without cd'ing into it.
	ManifestPath string
	// Package restricts the build to one workspace member via `cargo build
	// --package`, e.g. ego-rs's "reference-app" among its 19 members.
	Package string
	// Bin selects one binary target within the package via `cargo build
	// --bin`, needed when a package defines more than one (e.g.
	// examples/reference-app/src/bin/server.rs).
	Bin string
	// Locked maps to `--locked`, forbidding Cargo from updating
	// Cargo.lock so the same lockfile always produces the same dependency
	// graph in CI.
	Locked bool
}

// Compile-time conformance assertion: RustBuilder must satisfy Layer 1's
// Builder interface (pkg/shipwright).
var _ shipwright.Builder = (*RustBuilder)(nil)

// Build compiles source into a binary and returns a Directory containing
// only the compiled artifact (rooted at "/", holding the binary file).
func (b *RustBuilder) Build(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
	if b.Client == nil {
		return nil, errors.New("rustbuilder: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("rustbuilder: source directory is nil")
	}

	rustVersion := resolveRustVersion(b.RustVersion)
	profile := resolveCargoProfile(b.Config.BuildMode)
	subdir := targetSubdir(profile)

	// Unlike GoBuilder's defaultBinaryName ("app"), a hardcoded fallback
	// here can't ever be correct: cargo always names its build output after
	// the crate's actual package name (Cargo.toml's [package].name), never
	// a caller-chosen "-o" path, so a Config.BinaryName-less manifest must
	// infer the real name from the source itself rather than guess "app"
	// and have the build's cp step fail against almost every real crate.
	sourceDir := daggerkit.NewDaggerDirectoryAdapter(source)

	binaryName := b.Config.BinaryName
	switch {
	case binaryName != "":
		// explicit Config.BinaryName always wins.
	case b.Bin != "":
		// --bin names the exact binary target cargo will produce.
		binaryName = b.Bin
	case b.Package != "":
		// No --bin given: cargo names a package's single default binary
		// target after the package itself in the common case. A package
		// with several bin targets, or a bin name that differs from its
		// package name (examples/reference-app/src/bin/server.rs), needs
		// an explicit BinaryName or Bin instead.
		binaryName = b.Package
	default:
		manifestPath := b.ManifestPath
		if manifestPath == "" {
			manifestPath = "Cargo.toml"
		}
		cargoToml, err := sourceDir.File(manifestPath).Contents(ctx)
		if err != nil {
			return nil, fmt.Errorf("rustbuilder: binaryName not set and failed to read %s to infer it: %w", manifestPath, err)
		}
		binaryName, err = parseCargoPackageName(cargoToml)
		if err != nil {
			return nil, fmt.Errorf("rustbuilder: binaryName not set and failed to infer it from %s: %w", manifestPath, err)
		}
	}

	container := b.Client.Container().
		From("rust:"+rustVersion).
		WithMountedCache(cargoRegistryMountPath, b.Client.CacheVolume(cargoRegistryCacheKey)).
		WithMountedDirectory("/app", sourceDir).
		WithWorkdir("/app").
		WithMountedCache("/app/target", b.Client.CacheVolume(rustBuilderTargetCacheKey)).
		WithExec(b.cargoBuildArgs(profile))

	// Unlike `go build -o`, cargo does not accept an arbitrary output
	// path/name on stable toolchains (`--out-dir` remains unstable) — the
	// produced binary's name and location are fixed by Cargo.toml's
	// package name under target/<profile>/. BinaryName is therefore
	// expected to match that package name; this explicit copy step moves
	// the fixed-location artifact into the same /output/<binaryName>
	// convention GoBuilder produces directly via `-o`.
	outPath := "/output/" + binaryName
	container = container.
		WithExec([]string{"mkdir", "-p", "/output"}).
		WithExec([]string{"cp", "/app/target/" + subdir + "/" + binaryName, outPath})

	built, err := container.Sync(ctx)
	if err != nil {
		return nil, wrapExecError("rustbuilder: failed to build rust binary", err)
	}

	return built.Directory("/output").GetRealDirectory(), nil
}

// cargoBuildArgs returns the `cargo build` invocation for profile, given
// b's manifest/package/bin/locked configuration. cargo treats "release" and
// "debug"/"dev" as its two built-in profiles (`cargo build --release` and
// plain `cargo build` respectively); any other name is a custom profile
// (stable since Rust 1.57) invoked via `--profile`.
func (b *RustBuilder) cargoBuildArgs(profile string) []string {
	args := []string{"cargo", "build"}

	if b.ManifestPath != "" {
		args = append(args, "--manifest-path", b.ManifestPath)
	}
	if b.Package != "" {
		args = append(args, "--package", b.Package)
	}
	if b.Bin != "" {
		args = append(args, "--bin", b.Bin)
	}
	if b.Locked {
		args = append(args, "--locked")
	}

	switch profile {
	case "release":
		args = append(args, "--release")
	case "debug", "dev":
		// plain `cargo build` already builds the debug profile.
	default:
		args = append(args, "--profile", profile)
	}

	return args
}

// resolveRustVersion returns rustVersion, or defaultRustVersion when
// rustVersion is empty. Extracted as a pure helper so it is unit-testable
// without a Dagger client.
func resolveRustVersion(rustVersion string) string {
	if rustVersion == "" {
		return defaultRustVersion
	}
	return rustVersion
}

// resolveBinaryName returns cfgName, or defaultBinaryName when cfgName is
// empty. Extracted as a pure helper so it is unit-testable without a Dagger
// client.
func resolveBinaryName(cfgName string) string {
	if cfgName == "" {
		return defaultBinaryName
	}
	return cfgName
}

// resolveCargoProfile returns cfgMode, or defaultCargoProfile when cfgMode
// is empty. Extracted as a pure helper so it is unit-testable without a
// Dagger client.
func resolveCargoProfile(cfgMode string) string {
	if cfgMode == "" {
		return defaultCargoProfile
	}
	return cfgMode
}

// parseCargoPackageName extracts the `name` key from a Cargo.toml's
// [package] table, the crate name cargo uses to name its build output under
// target/<profile>/. A minimal line scanner rather than a full TOML parser:
// providers/rust has no TOML dependency (go.mod), and only ever needs this
// one key from one well-known table.
func parseCargoPackageName(cargoToml string) (string, error) {
	inPackageSection := false
	for _, line := range strings.Split(cargoToml, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inPackageSection = trimmed == "[package]"
			continue
		}
		if !inPackageSection {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok || strings.TrimSpace(key) != "name" {
			continue
		}
		name := strings.Trim(strings.TrimSpace(value), `"'`)
		if name != "" {
			return name, nil
		}
	}
	return "", errors.New("no [package] name found in Cargo.toml")
}

// targetSubdir returns the target/ subdirectory cargo places profile's
// build output under. cargo special-cases "dev" to the "debug" directory
// (its historical name); every other profile, including custom ones, uses
// its own name as the directory.
func targetSubdir(profile string) string {
	if profile == "dev" {
		return "debug"
	}
	return profile
}
