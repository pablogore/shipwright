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

// TestComputeEntrypoint covers the same bug providers/go's containerpublisher
// had: ContainerPublisher.Publish hardcoded "/app/"+defaultBinaryName as
// the entrypoint, ignoring any non-default Config.BinaryName a manifest set
// via the rust builder — so a manifest using `binaryName: my-service`
// published an image whose entrypoint pointed at "/app/app", a file that
// build never produced (the actual file is "/app/my-service").
// computeEntrypoint must vary with its input rather than always returning
// "/app/app".
func TestComputeEntrypoint(t *testing.T) {
	tests := []struct {
		name       string
		binaryName string
		want       string
	}{
		{name: "empty falls back to default", binaryName: "", want: "/app/app"},
		{name: "explicit binary name changes the entrypoint", binaryName: "my-service", want: "/app/my-service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeEntrypoint(tt.binaryName)
			if got != tt.want {
				t.Fatalf("computeEntrypoint(%q) = %q, want %q", tt.binaryName, got, tt.want)
			}
		})
	}

	if got := computeEntrypoint("my-service"); got == "/app/app" {
		t.Fatalf("computeEntrypoint(%q) = %q, want it to differ from the hardcoded default /app/app", "my-service", got)
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

func TestRegistryHost(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{name: "dotted registry host", ref: "ghcr.io/acme/api:v1", want: "ghcr.io"},
		{name: "host with port", ref: "localhost:5000/acme/api:v1", want: "localhost:5000"},
		{name: "bare docker hub name has no registry segment", ref: "acme/api:v1", want: "acme/api:v1"},
		{name: "no slash at all", ref: "api:v1", want: "api:v1"},
		{name: "localhost without port", ref: "localhost/acme/api:v1", want: "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := registryHost(tt.ref)
			if got != tt.want {
				t.Fatalf("registryHost(%q) = %q, want %q", tt.ref, got, tt.want)
			}
		})
	}
}
