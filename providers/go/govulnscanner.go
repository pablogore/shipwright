package golang

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// affectedCountRegexp captures the vulnerability count from govulncheck's
// "Your code is affected by N vulnerabilities" summary line.
var affectedCountRegexp = regexp.MustCompile(`Your code is affected by (\d+) vulnerabilit`)

// GoVulnScanner runs govulncheck against a source Directory and returns
// its output as the report File. Extracted from the legacy go-service
// pipeline's Vuln logic (internal/pipelines/go-service/pipeline.go) — one
// of three independent Tester implementations produced by the go-service
// decomposition (design.md D-F).
type GoVulnScanner struct {
	// Client is the Dagger client used to construct the scan container.
	Client *dagger.Client
	// GoVersion selects the Go toolchain image used to install and run
	// govulncheck. Defaults to defaultGoVersion when left empty.
	GoVersion string
}

// Compile-time conformance assertion (tasks.md 3.5): GoVulnScanner must
// satisfy Layer 1's Tester interface.
var _ shipwright.Tester = (*GoVulnScanner)(nil)

// Test installs and runs govulncheck against the source Directory. It
// fails when known vulnerabilities are reported (matching legacy
// behavior), otherwise returns the captured output as the report File.
//
// Behavioral judgment call: the legacy pipeline fetched stdout and stderr
// with two separate WithExec calls on failure to build a combined output
// for the vulnerability-substring check. Dagger's ExecError embeds both
// stdout and stderr in err.Error() itself, so this implementation checks
// output+err.Error() instead of issuing a second container execution —
// equivalent detection coverage with one exec instead of two.
func (v *GoVulnScanner) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if v.Client == nil {
		return nil, errors.New("govulnscanner: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("govulnscanner: source directory is nil")
	}

	goVersion := resolveGoVersion(v.GoVersion)

	container := v.Client.Container().
		From("golang:"+goVersion).
		WithMountedDirectory("/app", source).
		WithWorkdir("/app").
		WithEnvVariable("GO111MODULE", "on").
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"})

	output, err := container.WithExec([]string{"govulncheck", "./..."}).Stdout(ctx)
	if err != nil {
		combined := output + err.Error()
		if vulnerabilitiesReported(combined) {
			return nil, fmt.Errorf("govulnscanner: security vulnerabilities detected:\n%s", combined)
		}
		return nil, fmt.Errorf("govulnscanner: govulncheck failed: %w", err)
	}

	if vulnerabilitiesReported(output) {
		return nil, fmt.Errorf("govulnscanner: security vulnerabilities detected:\n%s", output)
	}

	reportContainer := container.WithNewFile("/tmp/vuln-report.txt", output)
	return reportContainer.File("/tmp/vuln-report.txt"), nil
}

// vulnerabilitiesReported inspects govulncheck's human-readable output for
// its vulnerability markers.
//
// Bugfix (found via this package's real-engine integration test, not
// present in a mocked unit test): the legacy pipeline's check was a bare
// strings.Contains(output, "Your code is affected") — but govulncheck's
// own *clean* output for zero findings reads "Your code is affected by 0
// vulnerabilities", which that substring matches unconditionally. Faithful
// extraction would have reproduced a false-positive bug, so this
// implementation instead parses the reported count and only flags a
// vulnerability when it is greater than zero. "Vulnerabilities found" is
// kept as a secondary marker for older/alternate govulncheck output
// formats, matching the legacy check's intent.
func vulnerabilitiesReported(output string) bool {
	if strings.Contains(output, "Vulnerabilities found") {
		return true
	}

	matches := affectedCountRegexp.FindStringSubmatch(output)
	if len(matches) < 2 {
		return false
	}

	count, err := strconv.Atoi(matches[1])
	if err != nil {
		return false
	}

	return count > 0
}
