package executors

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCICD_GitHubActions(t *testing.T) {
	// Arrange
	os.Setenv("GITHUB_ACTIONS", "true")
	defer os.Unsetenv("GITHUB_ACTIONS")

	ctx := context.Background()

	// Act
	detected := DetectCICD(ctx)

	// Assert
	assert.Equal(t, CICDTypeGitHubActions, detected)
}

func TestDetectCICD_GitLabCI(t *testing.T) {
	// Arrange
	os.Setenv("GITLAB_CI", "true")
	defer os.Unsetenv("GITLAB_CI")

	ctx := context.Background()

	// Act
	detected := DetectCICD(ctx)

	// Assert
	assert.Equal(t, CICDTypeGitLabCI, detected)
}

func TestDetectCICD_Jenkins(t *testing.T) {
	// Arrange
	os.Setenv("JENKINS_URL", "http://jenkins.example.com")
	defer os.Unsetenv("JENKINS_URL")

	ctx := context.Background()

	// Act
	detected := DetectCICD(ctx)

	// Assert
	assert.Equal(t, CICDTypeJenkins, detected)
}

func TestDetectCICD_CircleCI(t *testing.T) {
	// Arrange
	os.Setenv("CIRCLECI", "true")
	defer os.Unsetenv("CIRCLECI")

	ctx := context.Background()

	// Act
	detected := DetectCICD(ctx)

	// Assert
	assert.Equal(t, CICDTypeCircleCI, detected)
}

func TestDetectCICD_Local(t *testing.T) {
	// Arrange - ensure no CI environment variables are set
	os.Unsetenv("GITHUB_ACTIONS")
	os.Unsetenv("GITLAB_CI")
	os.Unsetenv("JENKINS_URL")
	os.Unsetenv("CIRCLECI")

	ctx := context.Background()

	// Act
	detected := DetectCICD(ctx)

	// Assert
	assert.Equal(t, CICDTypeLocal, detected)
}

func TestNewCICDExecutor(t *testing.T) {
	// Arrange
	os.Unsetenv("GITHUB_ACTIONS")
	os.Unsetenv("GITLAB_CI")
	os.Unsetenv("JENKINS_URL")
	os.Unsetenv("CIRCLECI")

	// Act
	executor := NewCICDExecutor()

	// Assert
	require.NotNil(t, executor)
	assert.Equal(t, CICDTypeLocal, executor.GetDetectedType())
	assert.False(t, executor.IsCIEnvironment())
}

func TestCICDExecutor_IsCIEnvironment_True(t *testing.T) {
	// Arrange
	os.Setenv("GITHUB_ACTIONS", "true")
	defer os.Unsetenv("GITHUB_ACTIONS")

	// Act
	executor := NewCICDExecutor()

	// Assert
	assert.True(t, executor.IsCIEnvironment())
}

func TestCICDExecutor_IsCIEnvironment_False(t *testing.T) {
	// Arrange
	os.Unsetenv("GITHUB_ACTIONS")
	os.Unsetenv("GITLAB_CI")
	os.Unsetenv("JENKINS_URL")
	os.Unsetenv("CIRCLECI")

	// Act
	executor := NewCICDExecutor()

	// Assert
	assert.False(t, executor.IsCIEnvironment())
}
