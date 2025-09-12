package pipelines

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig_DefaultValues(t *testing.T) {
	cfg := NewConfig()

	assert.Equal(t, "dev", cfg.Env)
	assert.Equal(t, "main", cfg.GitRef)
	assert.False(t, cfg.SkipPush)
	assert.False(t, cfg.OnlyTest)
	assert.False(t, cfg.OnlyBuild)
	assert.False(t, cfg.Verbose)
	assert.Empty(t, cfg.GitRepo)
	assert.Empty(t, cfg.GitProtocol)
	assert.Empty(t, cfg.GitUserEmail)
	assert.Empty(t, cfg.GitUserName)
	assert.Empty(t, cfg.RegistryURL)
	assert.Empty(t, cfg.RegistryToken)
	assert.Empty(t, cfg.Version)
	assert.Empty(t, cfg.BuildTag)
	assert.Empty(t, cfg.CommitSHA)
	assert.Empty(t, cfg.BranchName)
	assert.Empty(t, cfg.Token)
	assert.Zero(t, cfg.Coverage)
	assert.Nil(t, cfg.Image)
	assert.Empty(t, cfg.ImageRef)
	assert.Nil(t, cfg.ImageContainer)
	assert.Empty(t, cfg.ImageName)
	assert.Empty(t, cfg.RegistryUser)
	assert.Empty(t, cfg.RegistryPass)
	assert.Empty(t, cfg.Registry)
	assert.Empty(t, cfg.ImageTag)
	assert.Empty(t, cfg.GoVersion)
	assert.Empty(t, cfg.JavaVersion)
	assert.Empty(t, cfg.SSHPrivateKey)
}

func TestNewConfig_WithEnv(t *testing.T) {
	cfg := NewConfig(WithEnv("production"))

	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "main", cfg.GitRef) // Default value should remain
}

func TestNewConfig_WithSkipPush(t *testing.T) {
	cfg := NewConfig(WithSkipPush(true))

	assert.True(t, cfg.SkipPush)
	assert.Equal(t, "dev", cfg.Env) // Default value should remain
}

func TestNewConfig_WithOnlyTest(t *testing.T) {
	cfg := NewConfig(WithOnlyTest(true))

	assert.True(t, cfg.OnlyTest)
	assert.False(t, cfg.SkipPush) // Default value should remain
}

func TestNewConfig_WithOnlyBuild(t *testing.T) {
	cfg := NewConfig(WithOnlyBuild(true))

	assert.True(t, cfg.OnlyBuild)
	assert.False(t, cfg.OnlyTest) // Default value should remain
}

func TestNewConfig_WithVerbose(t *testing.T) {
	cfg := NewConfig(WithVerbose(true))

	assert.True(t, cfg.Verbose)
	assert.False(t, cfg.OnlyBuild) // Default value should remain
}

func TestNewConfig_WithRegistry(t *testing.T) {
	url := "registry.example.com"
	token := "test-token"
	cfg := NewConfig(WithRegistry(url, token))

	assert.Equal(t, url, cfg.RegistryURL)
	assert.Equal(t, token, cfg.RegistryToken)
}

func TestNewConfig_WithRegistryUser(t *testing.T) {
	user := "test-user"
	cfg := NewConfig(WithRegistryUser(user))

	assert.Equal(t, user, cfg.RegistryUser)
}

func TestNewConfig_WithBuildTag(t *testing.T) {
	tag := "v1.0.0"
	cfg := NewConfig(WithBuildTag(tag))

	assert.Equal(t, tag, cfg.BuildTag)
}

func TestNewConfig_WithCommitSHA(t *testing.T) {
	sha := "abc123def456"
	cfg := NewConfig(WithCommitSHA(sha))

	assert.Equal(t, sha, cfg.CommitSHA)
}

func TestNewConfig_WithBranch(t *testing.T) {
	branch := "feature-branch"
	cfg := NewConfig(WithBranch(branch))

	assert.Equal(t, branch, cfg.BranchName)
}

func TestNewConfig_WithGitGitProtocol(t *testing.T) {
	protocol := "ssh"
	cfg := NewConfig(WithGitGitProtocol(protocol))

	assert.Equal(t, protocol, cfg.GitProtocol)
}

func TestNewConfig_WithGitRepo(t *testing.T) {
	repo := "https://github.com/test/repo.git"
	ref := "develop"
	cfg := NewConfig(WithGitRepo(repo, ref))

	assert.Equal(t, repo, cfg.GitRepo)
	assert.Equal(t, ref, cfg.GitRef)
}

func TestNewConfig_WithToken(t *testing.T) {
	token := "test-token"
	cfg := NewConfig(WithToken(token))

	assert.Equal(t, token, cfg.Token)
}

func TestNewConfig_WithCoverage(t *testing.T) {
	coverage := 95.5
	cfg := NewConfig(WithCoverage(coverage))

	assert.InEpsilon(t, coverage, cfg.Coverage, 0.0001)
}

func TestNewConfig_WithGoVersion(t *testing.T) {
	version := "1.21"
	cfg := NewConfig(WithGoVersion(version))

	assert.Equal(t, version, cfg.GoVersion)
}

func TestNewConfig_WithSSHPrivateKey(t *testing.T) {
	key := "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key\n-----END OPENSSH PRIVATE KEY-----"
	cfg := NewConfig(WithSSHPrivateKey(key))

	assert.Equal(t, key, cfg.SSHPrivateKey)
}

func TestNewConfig_WithGitUserEmail(t *testing.T) {
	email := "test@example.com"
	cfg := NewConfig(WithGitUserEmail(email))

	assert.Equal(t, email, cfg.GitUserEmail)
}

func TestNewConfig_WithGitUserName(t *testing.T) {
	name := "Test User"
	cfg := NewConfig(WithGitUserName(name))

	assert.Equal(t, name, cfg.GitUserName)
}

func TestNewConfig_MultipleOptions(t *testing.T) {
	cfg := NewConfig(
		WithEnv("staging"),
		WithSkipPush(true),
		WithVerbose(true),
		WithRegistry("registry.example.com", "token123"),
		WithBranch("feature-branch"),
		WithCoverage(90.0),
		WithGoVersion("1.21"),
	)

	assert.Equal(t, "staging", cfg.Env)
	assert.True(t, cfg.SkipPush)
	assert.True(t, cfg.Verbose)
	assert.Equal(t, "registry.example.com", cfg.RegistryURL)
	assert.Equal(t, "token123", cfg.RegistryToken)
	assert.Equal(t, "feature-branch", cfg.BranchName)
	assert.InEpsilon(t, 90.0, cfg.Coverage, 0.0001)
	assert.Equal(t, "1.21", cfg.GoVersion)

	// Default values should remain
	assert.Equal(t, "main", cfg.GitRef)
	assert.False(t, cfg.OnlyTest)
	assert.False(t, cfg.OnlyBuild)
}

func TestNewConfig_OptionOrder(t *testing.T) {
	// Test that later options override earlier ones
	cfg := NewConfig(
		WithEnv("dev"),
		WithEnv("production"),
		WithSkipPush(false),
		WithSkipPush(true),
	)

	assert.Equal(t, "production", cfg.Env)
	assert.True(t, cfg.SkipPush)
}

func TestNewConfig_EmptyOptions(t *testing.T) {
	cfg := NewConfig()

	// Should have default values
	assert.Equal(t, "dev", cfg.Env)
	assert.Equal(t, "main", cfg.GitRef)
}
