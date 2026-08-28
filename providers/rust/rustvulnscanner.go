package rust

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
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
	Client daggerkit.DaggerClient
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
func (v *RustVulnScanner) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if v.Client == nil {
		return nil, errors.New("rustvulnscanner: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("rustvulnscanner: source directory is nil")
	}

	rustVersion := resolveRustVersion(v.RustVersion)
	sourceDir := daggerkit.NewDaggerDirectoryAdapter(source)

	container := v.Client.Container().
		From("rust:"+rustVersion).
		WithMountedCache(cargoRegistryMountPath, v.Client.CacheVolume(cargoRegistryCacheKey)).
		WithMountedDirectory("/app", sourceDir).
		WithWorkdir("/app").
		WithExec([]string{"cargo", "install", "cargo-audit", "--locked"})

	output, err := container.WithExec([]string{"cargo", "audit"}).Stdout(ctx)
	if err != nil {
		combined := auditCombinedOutput(output, err)
		if auditVulnerabilitiesReported(combined) {
			return nil, fmt.Errorf("rustvulnscanner: security vulnerabilities detected:\n%s", combined)
		}
		return nil, wrapExecError("rustvulnscanner: cargo audit failed", err)
	}

	if auditVulnerabilitiesReported(output) {
		return nil, fmt.Errorf("rustvulnscanner: security vulnerabilities detected:\n%s", output)
	}

	reportContainer := container.WithNewFile("/tmp/vuln-report.txt", output)
	return reportContainer.File("/tmp/vuln-report.txt").GetRealFile(), nil
}

// auditCombinedOutput builds the text auditVulnerabilitiesReported scans for
// cargo-audit's summary line. Unlike GoVulnScanner's analogous helper,
// dagger.ExecError.Error() does NOT embed the failed command's stdout/stderr
// (see wrapExecError's own doc comment) — appending err.Error() to output
// therefore never surfaced cargo-audit's actual CVE details. Read the
// ExecError's exported Stdout/Stderr fields directly instead, the same
// pattern wrapExecError uses.
func auditCombinedOutput(output string, err error) string {
	var execErr *dagger.ExecError
	if errors.As(err, &execErr) {
		return output + execErr.Stdout + execErr.Stderr
	}
	return output + err.Error()
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
