package golang

import "testing"

// Unit tests for unexported pure helpers shared across this package's
// capability implementations. In-package (not capabilities_test) because
// these helpers are deliberately unexported — each is an internal
// convenience, not part of any capability's public contract.

func TestResolveGoVersion(t *testing.T) {
	tests := []struct {
		name       string
		cfgVersion string
		want       string
	}{
		{name: "empty falls back to default", cfgVersion: "", want: defaultGoVersion},
		{name: "explicit version is preserved", cfgVersion: "1.26.1", want: "1.26.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveGoVersion(tt.cfgVersion)
			if got != tt.want {
				t.Fatalf("resolveGoVersion(%q) = %q, want %q", tt.cfgVersion, got, tt.want)
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

func TestParseCoveragePercentage(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    float64
		wantErr bool
	}{
		{
			name:   "well-formed cover output",
			output: "github.com/x/y/main.go:10:\tmain\t\t100.0%\ntotal:\t\t\t\t(statements)\t87.50%\n",
			want:   87.50,
		},
		{
			name:    "malformed output has no total line",
			output:  "no coverage data here",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCoveragePercentage(tt.output)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseCoveragePercentage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("parseCoveragePercentage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVulnerabilitiesReported(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{name: "clean scan", output: "No vulnerabilities found.", want: false},
		{name: "affected code", output: "Your code is affected by 1 vulnerability", want: true},
		{name: "multiple affected", output: "Your code is affected by 3 vulnerabilities", want: true},
		{name: "vulnerabilities found phrasing", output: "Vulnerabilities found in dependencies", want: true},
		{
			// Regression case for the false-positive bug found via this
			// package's real-engine integration test: govulncheck's own
			// clean-scan summary literally contains "Your code is
			// affected by 0 vulnerabilities" — a bare substring match on
			// "Your code is affected" (the legacy pipeline's check)
			// would misreport this as a finding.
			name:   "zero affected is not a finding",
			output: "=== Symbol Results ===\n\nNo vulnerabilities found.\n\nYour code is affected by 0 vulnerabilities.",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := vulnerabilitiesReported(tt.output)
			if got != tt.want {
				t.Fatalf("vulnerabilitiesReported(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}
