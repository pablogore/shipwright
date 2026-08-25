package shipwright_test

import (
	"regexp"
	"testing"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// semVerPattern matches a bare "MAJOR.MINOR.PATCH" version, per design.md
// D-E: "pkg/shipwright/version.go -> const ContractVersion = "1.0.0"".
var semVerPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// contractVersionIsCompileTimeConstant fails to compile if
// shipwright.ContractVersion is ever redeclared as a var. A var could be
// overwritten by a CLI build-time ldflags injection (as main.go's own
// `Version = "dev"` is) or read from a file such as dagger.json at runtime;
// a const cannot be either, which is the structural proof that
// ContractVersion "resolves independently of the CLI SemVer / dagger.json
// engine pin" (task 1.6).
const contractVersionIsCompileTimeConstant = shipwright.ContractVersion

func TestContractVersion_IsValidSemVer(t *testing.T) {
	t.Parallel()

	if !semVerPattern.MatchString(shipwright.ContractVersion) {
		t.Fatalf("shipwright.ContractVersion = %q, want a bare MAJOR.MINOR.PATCH SemVer string", shipwright.ContractVersion)
	}
}

func TestContractVersion_MatchesDocumentedInitialValue(t *testing.T) {
	t.Parallel()

	const want = "1.0.0"
	if shipwright.ContractVersion != want {
		t.Fatalf("shipwright.ContractVersion = %q, want %q (design.md D-E source of truth)", shipwright.ContractVersion, want)
	}
}

func TestContractVersion_IsIndependentOfCLIBuildVersion(t *testing.T) {
	t.Parallel()

	// contractVersionIsCompileTimeConstant only exists (and only compiles)
	// because ContractVersion is a const, not a var — so it cannot be the
	// same storage location the CLI's ldflags-injected `Version` var (in
	// main.go, default "dev") or a `dagger.json` engineVersion read would
	// populate. The two are structurally distinct version spaces.
	if contractVersionIsCompileTimeConstant != shipwright.ContractVersion {
		t.Fatalf("contract version constant drifted from shipwright.ContractVersion")
	}
}
