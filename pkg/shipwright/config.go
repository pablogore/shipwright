package shipwright

import "dagger.io/dagger"

// SourceConfig configures how a Builder obtains its source input (git
// checkout, credentials). It carries no Build/Test/Artifact/Deploy/Run
// field — decomposed per design.md D-D so orthogonality is
// compiler-enforced rather than documented.
//
// Security-relevant: SSHPrivateKey is a *dagger.Secret, never a plaintext
// string — credentials MUST cross the public contract as *dagger.Secret
// only (design.md D-D, Threat Matrix), the same invariant already enforced
// for ArtifactConfig's credential fields.
type SourceConfig struct {
	// GitRepo is the Git repository URL.
	GitRepo string
	// GitRef is the Git reference (branch or tag) to check out.
	GitRef string
	// GitProtocol is the Git transport protocol ("ssh" or "https").
	GitProtocol string
	// GitUserEmail is the Git user email used for any commits made during
	// the pipeline.
	GitUserEmail string
	// GitUserName is the Git user name used for any commits made during
	// the pipeline.
	GitUserName string
	// SSHPrivateKey is the SSH private key used for Git authentication,
	// carried as a Dagger secret so it never surfaces as plaintext.
	SSHPrivateKey *dagger.Secret
}

// BuildConfig configures a Builder implementation.
type BuildConfig struct {
	// GoVersion is the Go toolchain version to use (e.g. "1.26.1").
	GoVersion string
	// JavaVersion is the Java toolchain version to use (e.g. "17").
	JavaVersion string
	// BuildMode selects how the Builder produces its output (e.g.
	// "binary", "docker", "both"). Left as a plain string here; a
	// concrete Builder implementation owns its own enum.
	BuildMode string
	// BinaryName is the name of the output binary file, when applicable.
	BinaryName string
}

// TestConfig configures a Tester implementation.
type TestConfig struct {
	// Coverage is the minimum required test coverage percentage.
	Coverage float64
}

// ArtifactConfig configures an Artifactor implementation.
//
// Security-relevant: RegistryPass, RegistryToken, and Token are
// *dagger.Secret, never a plaintext string — credentials MUST cross the
// public contract as *dagger.Secret only (design.md D-D, Threat Matrix).
type ArtifactConfig struct {
	// Registry is the target Docker registry (e.g.
	// registry.gitlab.com/my-org/my-project/service).
	Registry string
	// RegistryURL is the Docker registry URL used for authentication.
	RegistryURL string
	// RegistryUser is the registry username (e.g. gitlab-ci-token in CI).
	RegistryUser string
	// RegistryPass is the registry password/personal access token,
	// carried as a Dagger secret so it never surfaces as plaintext.
	RegistryPass *dagger.Secret
	// RegistryToken is the registry authentication token, carried as a
	// Dagger secret so it never surfaces as plaintext.
	RegistryToken *dagger.Secret
	// ImageName is the published image name.
	ImageName string
	// BinaryName is the name of the compiled binary file inside the
	// published image, used to compute the container entrypoint
	// ("/app/"+BinaryName). Left empty, an Artifactor falls back to its own
	// default binary name (mirroring BuildConfig.BinaryName's own
	// empty-falls-back-to-default convention) — set it explicitly whenever
	// the paired Builder was configured with a non-default BuildConfig.
	// BinaryName, so the two agree on where the binary actually lives.
	BinaryName string
	// ImageTag is the published image tag (e.g. latest, sha, v1.2.3).
	ImageTag string
	// BuildTag is the build tag associated with the artifact.
	BuildTag string
	// CommitSHA is the commit SHA the artifact was built from.
	CommitSHA string
	// BranchName is the Git branch the artifact was built from.
	BranchName string
	// Version is the artifact version.
	Version string
	// Token is a generic authentication token, carried as a Dagger secret
	// so it never surfaces as plaintext.
	Token *dagger.Secret
}

// DeployConfig configures a Deployer implementation. Empty at this change —
// concrete deploy adapters (Kubernetes, Nomad, SSH, ...) are deferred to a
// follow-up change (design.md D-D).
type DeployConfig struct{}

// RunConfig configures a Runner implementation. Empty at this change —
// concrete run adapters are deferred to a follow-up change (design.md D-D).
type RunConfig struct{}
