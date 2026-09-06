package rust

import "strings"

// cargoTestArgsFor builds a `cargo test` invocation shared by every Tester
// in this package that runs cargo test against a (possibly non-root)
// manifest — RustUnitTester and RustIntegrationTester.
func cargoTestArgsFor(manifestPath, pkg string, locked, allFeatures bool, features []string) []string {
	return append([]string{"cargo", "test"}, cargoScopeArgsFor(manifestPath, pkg, locked, allFeatures, features)...)
}

// cargoScopeArgsFor builds the manifest/package/feature/lockfile selection
// args shared by `cargo test` and `cargo tarpaulin` — both accept the same
// flag set (tarpaulin 0.37.2 supports --manifest-path, --package/--packages,
// --features, --all-features, and --locked identically to cargo test).
// Extracted so RustUnitTester.enforceCoverageThreshold measures coverage
// over exactly the scope that was tested, instead of independently
// re-deriving (and risking silently diverging from) the same selection.
//
// Package and --workspace are mutually exclusive selection modes, so a set
// pkg drops --workspace entirely rather than combining both — mirrors
// ego-rs's own Makefile invocation `cargo test -p security-jwt --features
// test-kit`, which never passes --workspace alongside -p.
func cargoScopeArgsFor(manifestPath, pkg string, locked, allFeatures bool, features []string) []string {
	var args []string

	if manifestPath != "" {
		args = append(args, "--manifest-path", manifestPath)
	}

	if pkg != "" {
		args = append(args, "--package", pkg)
	} else {
		args = append(args, "--workspace")
	}

	if locked {
		args = append(args, "--locked")
	}

	if allFeatures {
		args = append(args, "--all-features")
	} else if len(features) > 0 {
		args = append(args, "--features", strings.Join(features, ","))
	}

	return args
}
