package executors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearCICDEnv resets every CI/CD signal DetectCICD checks, via t.Setenv so
// each is restored automatically after the test. Real CI runners (e.g.
// GITHUB_ACTIONS on the GitHub Actions runner itself) already have some of
// these set, so every test below must start from a known-clean slate before
// setting the one signal it means to simulate.
func clearCICDEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITLAB_CI", "")
	t.Setenv("JENKINS_URL", "")
	t.Setenv("CIRCLECI", "")
}

func TestDetectCICD_GitHubActions(t *testing.T) {
	// Arrange
	clearCICDEnv(t)
	t.Setenv("GITHUB_ACTIONS", "true")

	ctx := context.Background()

	// Act
	detected := DetectCICD(ctx)

	// Assert
	assert.Equal(t, CICDTypeGitHubActions, detected)
}

func TestDetectCICD_GitLabCI(t *testing.T) {
	// Arrange
	clearCICDEnv(t)
	t.Setenv("GITLAB_CI", "true")

	ctx := context.Background()

	// Act
	detected := DetectCICD(ctx)

	// Assert
	assert.Equal(t, CICDTypeGitLabCI, detected)
}

func TestDetectCICD_Jenkins(t *testing.T) {
	// Arrange
	clearCICDEnv(t)
	t.Setenv("JENKINS_URL", "http://jenkins.example.com")

	ctx := context.Background()

	// Act
	detected := DetectCICD(ctx)

	// Assert
	assert.Equal(t, CICDTypeJenkins, detected)
}

func TestDetectCICD_CircleCI(t *testing.T) {
	// Arrange
	clearCICDEnv(t)
	t.Setenv("CIRCLECI", "true")

	ctx := context.Background()

	// Act
	detected := DetectCICD(ctx)

	// Assert
	assert.Equal(t, CICDTypeCircleCI, detected)
}

func TestDetectCICD_Local(t *testing.T) {
	// Arrange - ensure no CI environment variables are set
	clearCICDEnv(t)

	ctx := context.Background()

	// Act
	detected := DetectCICD(ctx)

	// Assert
	assert.Equal(t, CICDTypeLocal, detected)
}

func TestNewCICDExecutor(t *testing.T) {
	// Arrange
	clearCICDEnv(t)

	// Act
	executor := NewCICDExecutor()

	// Assert
	require.NotNil(t, executor)
	assert.Equal(t, CICDTypeLocal, executor.GetDetectedType())
	assert.False(t, executor.IsCIEnvironment())
}

func TestCICDExecutor_IsCIEnvironment_True(t *testing.T) {
	// Arrange
	t.Setenv("GITHUB_ACTIONS", "true")

	// Act
	executor := NewCICDExecutor()

	// Assert
	assert.True(t, executor.IsCIEnvironment())
}

func TestCICDExecutor_IsCIEnvironment_False(t *testing.T) {
	// Arrange
	clearCICDEnv(t)

	// Act
	executor := NewCICDExecutor()

	// Assert
	assert.False(t, executor.IsCIEnvironment())
}
