package rust

import (
	"errors"
	"strings"
	"testing"

	"dagger.io/dagger"
)

// Unit tests for unexported pure helpers shared across this package's
// capability implementations. In-package (not rust_test) because these
// helpers are deliberately unexported — each is an internal convenience,
// not part of any capability's public contract. Mirrors providers/go's
// internal_test.go.

func TestResolveRustVersion(t *testing.T) {
	tests := []struct {
		name       string
		cfgVersion string
		want       string
	}{
		{name: "empty falls back to default", cfgVersion: "", want: defaultRustVersion},
		{name: "explicit version is preserved", cfgVersion: "1.83.0", want: "1.83.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveRustVersion(tt.cfgVersion)
			if got != tt.want {
				t.Fatalf("resolveRustVersion(%q) = %q, want %q", tt.cfgVersion, got, tt.want)
			}
		})
	}
}

func TestResolveBinaryName(t *testing.T) {
	tests := []struct {
		name    string
		cfgName string
		want    string
	}{
		{name: "empty falls back to default", cfgName: "", want: defaultBinaryName},
		{name: "explicit name is preserved", cfgName: "myservice", want: "myservice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveBinaryName(tt.cfgName)
			if got != tt.want {
				t.Fatalf("resolveBinaryName(%q) = %q, want %q", tt.cfgName, got, tt.want)
			}
		})
	}
}

func TestRustBuilder_CargoBuildArgs(t *testing.T) {
	tests := []struct {
		name    string
		builder RustBuilder
		profile string
		want    []string
	}{
		{
			name:    "default release profile",
			builder: RustBuilder{},
			profile: "release",
			want:    []string{"cargo", "build", "--release"},
		},
		{
			name:    "debug profile adds no extra flag",
			builder: RustBuilder{},
			profile: "debug",
			want:    []string{"cargo", "build"},
		},
		{
			name:    "custom profile uses --profile",
			builder: RustBuilder{},
			profile: "bench-profile",
			want:    []string{"cargo", "build", "--profile", "bench-profile"},
		},
		{
			name:    "package is passed ahead of the profile flag",
			builder: RustBuilder{Package: "reference-app"},
			profile: "release",
			want:    []string{"cargo", "build", "--package", "reference-app", "--release"},
		},
		{
			name:    "bin narrows within the selected package",
			builder: RustBuilder{Package: "reference-app", Bin: "server"},
			profile: "release",
			want:    []string{"cargo", "build", "--package", "reference-app", "--bin", "server", "--release"},
		},
		{
			name:    "locked adds --locked before the profile flag",
			builder: RustBuilder{Locked: true},
			profile: "release",
			want:    []string{"cargo", "build", "--locked", "--release"},
		},
		{
			name:    "manifestPath is passed ahead of package/bin selection",
			builder: RustBuilder{ManifestPath: "examples/reference-app/Cargo.toml", Package: "reference-app"},
			profile: "release",
			want:    []string{"cargo", "build", "--manifest-path", "examples/reference-app/Cargo.toml", "--package", "reference-app", "--release"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.builder.cargoBuildArgs(tt.profile)
			if len(got) != len(tt.want) {
				t.Fatalf("cargoBuildArgs(%q) = %v, want %v", tt.profile, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("cargoBuildArgs(%q) = %v, want %v", tt.profile, got, tt.want)
				}
			}
		})
	}
}

func TestResolveDockerSocketPath(t *testing.T) {
	tests := []struct {
		name    string
		cfgPath string
		want    string
	}{
		{name: "empty falls back to default", cfgPath: "", want: defaultDockerSocketPath},
		{name: "explicit path is preserved", cfgPath: "/custom/docker.sock", want: "/custom/docker.sock"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDockerSocketPath(tt.cfgPath)
			if got != tt.want {
				t.Fatalf("resolveDockerSocketPath(%q) = %q, want %q", tt.cfgPath, got, tt.want)
			}
		})
	}
}

func TestRustIntegrationTester_CargoTestArgs(t *testing.T) {
	tests := []struct {
		name   string
		tester RustIntegrationTester
		want   []string
	}{
		{
			name:   "manifestPath selects the isolated integration workspace",
			tester: RustIntegrationTester{ManifestPath: "integration-tests/Cargo.toml"},
			want:   []string{"cargo", "test", "--manifest-path", "integration-tests/Cargo.toml", "--workspace"},
		},
		{
			name: "package, locked, and features combine as expected",
			tester: RustIntegrationTester{
				ManifestPath: "integration-tests/Cargo.toml",
				Package:      "ego-integration",
				Locked:       true,
				Features:     []string{"crash-test-failpoint"},
			},
			want: []string{
				"cargo", "test",
				"--manifest-path", "integration-tests/Cargo.toml",
				"--package", "ego-integration",
				"--locked",
				"--features", "crash-test-failpoint",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tester.cargoTestArgs()
			if len(got) != len(tt.want) {
				t.Fatalf("cargoTestArgs() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("cargoTestArgs() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestParseCargoPackageName(t *testing.T) {
	tests := []struct {
		name      string
		cargoToml string
		want      string
		wantErr   bool
	}{
		{
			name: "simple package name",
			cargoToml: `[package]
name = "my-crate"
version = "0.1.0"
edition = "2021"
`,
			want: "my-crate",
		},
		{
			name: "package name after other keys",
			cargoToml: `[package]
edition = "2021"
name = "otherservice"
version = "0.1.0"
`,
			want: "otherservice",
		},
		{
			name: "ignores name keys outside [package]",
			cargoToml: `[dependencies]
name = "not-the-package-name"

[package]
name = "realname"
`,
			want: "realname",
		},
		{
			name: "single-quoted name",
			cargoToml: `[package]
name = 'quoted'
`,
			want: "quoted",
		},
		{
			name:      "no package section",
			cargoToml: "[dependencies]\nserde = \"1\"\n",
			wantErr:   true,
		},
		{
			name:      "empty input",
			cargoToml: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCargoPackageName(tt.cargoToml)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseCargoPackageName(%q) = %q, want error", tt.cargoToml, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCargoPackageName(%q) unexpected error: %v", tt.cargoToml, err)
			}
			if got != tt.want {
				t.Fatalf("parseCargoPackageName(%q) = %q, want %q", tt.cargoToml, got, tt.want)
			}
		})
	}
}

func TestResolveCargoProfile(t *testing.T) {
	tests := []struct {
		name    string
		cfgMode string
		want    string
	}{
		{name: "empty falls back to release", cfgMode: "", want: "release"},
		{name: "explicit release is preserved", cfgMode: "release", want: "release"},
		{name: "explicit debug is preserved", cfgMode: "debug", want: "debug"},
		{name: "explicit custom profile is preserved", cfgMode: "bench-profile", want: "bench-profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveCargoProfile(tt.cfgMode)
			if got != tt.want {
				t.Fatalf("resolveCargoProfile(%q) = %q, want %q", tt.cfgMode, got, tt.want)
			}
		})
	}
}

func TestTargetSubdir(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		want    string
	}{
		{name: "release profile maps to release dir", profile: "release", want: "release"},
		{name: "debug profile maps to debug dir", profile: "debug", want: "debug"},
		{name: "dev profile maps to debug dir", profile: "dev", want: "debug"},
		{name: "custom profile maps to its own dir", profile: "bench-profile", want: "bench-profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := targetSubdir(tt.profile)
			if got != tt.want {
				t.Fatalf("targetSubdir(%q) = %q, want %q", tt.profile, got, tt.want)
			}
		})
	}
}

// TestWrapExecError covers the CI blind spot from
// TestRustUnitTester_Test_RealEngine_PassesWithinThreshold's exit-code-101
// failure: a bare `%w`-wrapped dagger.ExecError never surfaces stderr, so
// wrapExecError must expand it into the outer message. The *dagger.ExecError
// case is constructed via a keyed literal that only sets its exported
// fields (Stderr/ExitCode is all this package can set from outside
// dagger.io/dagger), leaving its unexported `original` field nil — proof
// that wrapExecError reads only exported fields (never Error()/Message(),
// which would panic against that nil field) is what keeps this safe to run
// without a live Dagger client.
func TestWrapExecError(t *testing.T) {
	t.Run("plain error wraps normally", func(t *testing.T) {
		base := errors.New("boom")
		got := wrapExecError("rustunittester: tests failed", base)
		if got == nil {
			t.Fatal("wrapExecError() = nil, want non-nil error")
		}
		if !strings.Contains(got.Error(), "rustunittester: tests failed") || !strings.Contains(got.Error(), "boom") {
			t.Fatalf("wrapExecError() = %q, want it to contain the prefix and the base error", got.Error())
		}
		if !errors.Is(got, base) {
			t.Fatal("wrapExecError() does not unwrap to the original error via errors.Is")
		}
	})

	t.Run("ExecError expands exit code and stderr", func(t *testing.T) {
		execErr := &dagger.ExecError{
			ExitCode: 101,
			Stderr:   "thread 'main' panicked: ptrace(2): Operation not permitted (os error 1)\n",
		}
		got := wrapExecError("rustunittester: failed to compute coverage", execErr)
		if got == nil {
			t.Fatal("wrapExecError() = nil, want non-nil error")
		}
		msg := got.Error()
		if !strings.Contains(msg, "101") {
			t.Fatalf("wrapExecError() = %q, want it to contain the exit code 101", msg)
		}
		if !strings.Contains(msg, "ptrace") {
			t.Fatalf("wrapExecError() = %q, want it to contain the captured stderr", msg)
		}
	})
}

func TestRustUnitTester_CargoTestArgs(t *testing.T) {
	tests := []struct {
		name   string
		tester RustUnitTester
		want   []string
	}{
		{
			name:   "default runs the whole workspace with no feature flags",
			tester: RustUnitTester{},
			want:   []string{"cargo", "test", "--workspace"},
		},
		{
			name:   "package drops --workspace in favor of --package",
			tester: RustUnitTester{Package: "security-jwt"},
			want:   []string{"cargo", "test", "--package", "security-jwt"},
		},
		{
			name:   "features are joined and passed without --all-features",
			tester: RustUnitTester{Features: []string{"test-kit", "other-feature"}},
			want:   []string{"cargo", "test", "--workspace", "--features", "test-kit,other-feature"},
		},
		{
			name:   "allFeatures wins over an explicit features list",
			tester: RustUnitTester{Features: []string{"test-kit"}, AllFeatures: true},
			want:   []string{"cargo", "test", "--workspace", "--all-features"},
		},
		{
			name:   "locked adds --locked before feature flags",
			tester: RustUnitTester{Locked: true},
			want:   []string{"cargo", "test", "--workspace", "--locked"},
		},
		{
			name:   "manifestPath is passed ahead of workspace/package selection",
			tester: RustUnitTester{ManifestPath: "integration-tests/Cargo.toml", Package: "integration-tests"},
			want:   []string{"cargo", "test", "--manifest-path", "integration-tests/Cargo.toml", "--package", "integration-tests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tester.cargoTestArgs()
			if len(got) != len(tt.want) {
				t.Fatalf("cargoTestArgs() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("cargoTestArgs() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestParseTarpaulinCoverage(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    float64
		wantErr bool
	}{
		{
			name:   "well-formed tarpaulin output",
			output: "Coverage Results:\n|| Tested/Total Lines:\n|| src/lib.rs: 10/12\n||\n87.50% coverage, 10/12 lines covered\n",
			want:   87.50,
		},
		{
			name:    "malformed output has no coverage line",
			output:  "no coverage data here",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTarpaulinCoverage(tt.output)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseTarpaulinCoverage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("parseTarpaulinCoverage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuditVulnerabilitiesReported(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{name: "clean scan", output: "Scanning Cargo.lock for vulnerabilities (42 crate dependencies)\n", want: false},
		{name: "single vulnerability", output: "error: 1 vulnerability found!\n", want: true},
		{name: "multiple vulnerabilities", output: "error: 3 vulnerabilities found!\n", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auditVulnerabilitiesReported(tt.output)
			if got != tt.want {
				t.Fatalf("auditVulnerabilitiesReported(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

// TestAuditCombinedOutput covers the bug behind rustvulnscanner.go's
// `combined := output + err.Error()`: dagger.ExecError.Error() never embeds
// stdout/stderr (only wrapExecError's own doc comment explains why — it
// defers to an unexported `original` field the dagger package alone
// populates), so a real cargo-audit failure's CVE details never reached the
// combined string audited for "N vulnerabilities found!". Constructed the
// same way as TestWrapExecError's synthetic *dagger.ExecError, for the same
// reason: only its exported fields (Stdout/Stderr/ExitCode) are settable
// from outside dagger.io/dagger.
func TestAuditCombinedOutput(t *testing.T) {
	t.Run("plain error falls back to Error()", func(t *testing.T) {
		base := errors.New("boom")
		got := auditCombinedOutput("partial output", base)
		if !strings.Contains(got, "partial output") || !strings.Contains(got, "boom") {
			t.Fatalf("auditCombinedOutput() = %q, want it to contain the output and the base error", got)
		}
	})

	t.Run("ExecError contributes its own stdout and stderr", func(t *testing.T) {
		execErr := &dagger.ExecError{
			ExitCode: 1,
			Stdout:   "Scanning Cargo.lock for vulnerabilities (42 crate dependencies)\nerror: 1 vulnerability found!\n",
			Stderr:   "warning: unmaintained crate\n",
		}
		got := auditCombinedOutput("", execErr)
		if !strings.Contains(got, "1 vulnerability found!") {
			t.Fatalf("auditCombinedOutput() = %q, want it to contain the ExecError's Stdout", got)
		}
		if !strings.Contains(got, "unmaintained crate") {
			t.Fatalf("auditCombinedOutput() = %q, want it to contain the ExecError's Stderr", got)
		}
	})
}
