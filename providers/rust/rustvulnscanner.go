package rust

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// auditVulnerabilityCountRegexp captures the vulnerability count from
// cargo-audit's "error: N vulnerabilit{y,ies} found!" summary line.
var auditVulnerabilityCountRegexp = regexp.MustCompile(`(\d+) vulnerabilit(?:y|ies) found`)

// RustVulnScanner runs cargo-audit against a source Directory and returns
// its output as the report File. Structural mirror of providers/go's
// GoVulnScanner — one of three independent Tester implementations, none
// privileged, for the Rust toolchain.
//
// Design decision: cargo-audit over cargo-deny. cargo-deny is a broader
// policy tool (licenses, bans, duplicate versions, advisories); cargo-audit
// does one thing — scan Cargo.lock against the RustSec advisory database —
// making it the direct, single-purpose analog to govulncheck, which
// GoVulnScanner mirrors.
type RustVulnScanner struct {
	// Client is the Dagger client used to construct the scan container.
	Client *dagger.Client
	// RustVersion selects the Rust toolchain image used to install and run
	// cargo-audit. Defaults to defaultRustVersion when left empty.
	RustVersion string
}

// Compile-time conformance assertion: RustVulnScanner must satisfy Layer
// 1's Tester interface.
var _ shipwright.Tester = (*RustVulnScanner)(nil)

// Test installs and runs cargo-audit against the source Directory. It
// fails when known vulnerabilities are reported, otherwise returns the
// captured output as the report File.
//
// Same judgment call as GoVulnScanner: Dagger's ExecError embeds both
// stdout and stderr in err.Error() itself, so this implementation checks
// output+err.Error() instead of issuing a second container execution.
func (v *RustVulnScanner) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if v.Client == nil {
		return nil, errors.New("rustvulnscanner: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("rustvulnscanner: source directory is nil")
	}

	rustVersion := resolveRustVersion(v.RustVersion)

	container := v.Client.Container().
		From("rust:"+rustVersion).
		WithMountedDirectory("/app", source).
		WithWorkdir("/app").
		WithExec([]string{"cargo", "install", "cargo-audit", "--locked"})

	output, err := container.WithExec([]string{"cargo", "audit"}).Stdout(ctx)
	if err != nil {
		combined := output + err.Error()
		if auditVulnerabilitiesReported(combined) {
			return nil, fmt.Errorf("rustvulnscanner: security vulnerabilities detected:\n%s", combined)
		}
		return nil, fmt.Errorf("rustvulnscanner: cargo audit failed: %w", err)
	}

	if auditVulnerabilitiesReported(output) {
		return nil, fmt.Errorf("rustvulnscanner: security vulnerabilities detected:\n%s", output)
	}

	reportContainer := container.WithNewFile("/tmp/vuln-report.txt", output)
	return reportContainer.File("/tmp/vuln-report.txt"), nil
}

// auditVulnerabilitiesReported inspects cargo-audit's human-readable output
// for its vulnerability-count summary line and reports true only when the
// count is greater than zero.
func auditVulnerabilitiesReported(output string) bool {
	matches := auditVulnerabilityCountRegexp.FindStringSubmatch(output)
	if len(matches) < 2 {
		return false
	}

	count, err := strconv.Atoi(matches[1])
	if err != nil {
		return false
	}

	return count > 0
}
