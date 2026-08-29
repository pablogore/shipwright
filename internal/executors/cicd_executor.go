package executors

import (
	"context"
	"os"
)

// CICDType represents the type of CI/CD system detected.
type CICDType string

const (
	// CICDTypeGitHubActions represents GitHub Actions.
	CICDTypeGitHubActions CICDType = "github-actions"
	// CICDTypeGitLabCI represents GitLab CI.
	CICDTypeGitLabCI CICDType = "gitlab-ci"
	// CICDTypeJenkins represents Jenkins.
	CICDTypeJenkins CICDType = "jenkins"
	// CICDTypeCircleCI represents CircleCI.
	CICDTypeCircleCI CICDType = "circleci"
	// CICDTypeLocal represents local execution (not in CI/CD).
	CICDTypeLocal CICDType = "local"
)

// CICDExecutor detects and provides information about the CI/CD environment.
type CICDExecutor struct {
	detectedType CICDType
}

// NewCICDExecutor creates a new CICDExecutor.
func NewCICDExecutor() *CICDExecutor {
	return &CICDExecutor{
		detectedType: DetectCICD(context.Background()),
	}
}

// DetectCICD detects the CI/CD system from environment variables.
//
// Parameters:
//   - ctx: The context for managing execution.
//
// Returns:
//   - The detected CICDType, or CICDTypeLocal if no CI/CD is detected.
func DetectCICD(_ context.Context) CICDType {
	// Check for GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") != "" {
		return CICDTypeGitHubActions
	}

	// Check for GitLab CI
	if os.Getenv("GITLAB_CI") != "" {
		return CICDTypeGitLabCI
	}

	// Check for Jenkins
	if os.Getenv("JENKINS_URL") != "" {
		return CICDTypeJenkins
	}

	// Check for CircleCI
	if os.Getenv("CIRCLECI") != "" {
		return CICDTypeCircleCI
	}

	// Default to local
	return CICDTypeLocal
}

// GetDetectedType returns the detected CI/CD type.
func (e *CICDExecutor) GetDetectedType() CICDType {
	return e.detectedType
}

// IsCIEnvironment checks if we're running in any CI/CD environment.
func (e *CICDExecutor) IsCIEnvironment() bool {
	return e.detectedType != CICDTypeLocal
}
