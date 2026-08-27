package rust

import "strings"

// cargoTestArgsFor builds a `cargo test` invocation shared by every Tester
// in this package that runs cargo test against a (possibly non-root)
// manifest — RustUnitTester and RustIntegrationTester. Package and
// --workspace are mutually exclusive selection modes, so a set pkg drops
// --workspace entirely rather than combining both — mirrors ego-rs's own
// Makefile invocation `cargo test -p security-jwt --features test-kit`,
// which never passes --workspace alongside -p.
func cargoTestArgsFor(manifestPath, pkg string, locked, allFeatures bool, features []string) []string {
	args := []string{"cargo", "test"}

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
