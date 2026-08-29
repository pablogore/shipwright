package shared

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveGitCredentials_CI(t *testing.T) {
	// Setup
	t.Setenv("CI", "true")
	t.Setenv("CI_JOB_TOKEN", "ci-token")

	// Test
	creds, err := ResolveGitCredentials()
	require.NoError(t, err)
	assert.NotNil(t, creds)
	assert.Equal(t, "gitlab-ci-token", creds.User)
	assert.Equal(t, "ci-token", creds.Token)
	assert.Equal(t, string(SourceCI), creds.Source)
	assert.True(t, creds.ExpiresAt.After(time.Now()))
}

func TestResolveGitCredentials_Anonymous(t *testing.T) {
	// Setup - ensure no credentials are set
	t.Setenv("CI", "")
	t.Setenv("CI_JOB_TOKEN", "")
	t.Setenv("GITLAB_PAT", "")
	t.Setenv("SSH_PRIVATE_KEY", "")

	// Test
	creds, err := ResolveGitCredentials()
	require.NoError(t, err)
	assert.NotNil(t, creds)
	assert.Empty(t, creds.User)
	assert.Empty(t, creds.Token)
	assert.Equal(t, string(SourceAnonymous), creds.Source)
	assert.True(t, creds.ExpiresAt.After(time.Now()))
}

func TestGitCredentials_Validate(t *testing.T) {
	tests := []struct {
		name    string
		creds   *GitCredentials
		wantErr bool
	}{
		{
			name:    "nil credentials",
			creds:   nil,
			wantErr: true,
		},
		{
			name: "expired credentials",
			creds: &GitCredentials{
				User:      "test",
				Token:     "test",
				Source:    string(SourceCI),
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "valid CI credentials",
			creds: &GitCredentials{
				User:      "gitlab-ci-token",
				Token:     "test",
				Source:    string(SourceCI),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "invalid CI credentials - empty user",
			creds: &GitCredentials{
				User:      "",
				Token:     "test",
				Source:    string(SourceCI),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "invalid CI credentials - empty token",
			creds: &GitCredentials{
				User:      "test",
				Token:     "",
				Source:    string(SourceCI),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "valid SSH credentials",
			creds: &GitCredentials{
				User:      "git",
				Token:     "ssh-key",
				Source:    string(SourceSSH),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "invalid SSH credentials - empty key",
			creds: &GitCredentials{
				User:      "git",
				Token:     "",
				Source:    string(SourceSSH),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "valid anonymous credentials",
			creds: &GitCredentials{
				User:      "",
				Token:     "",
				Source:    string(SourceAnonymous),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "unknown source",
			creds: &GitCredentials{
				User:      "test",
				Token:     "test",
				Source:    "UNKNOWN",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGitCredentials_String(t *testing.T) {
	tests := []struct {
		name     string
		creds    *GitCredentials
		expected string
	}{
		{
			name:     "nil credentials",
			creds:    nil,
			expected: "<nil>",
		},
		{
			name: "valid credentials",
			creds: &GitCredentials{
				User:      "test",
				Token:     "test",
				Source:    string(SourceCI),
				ExpiresAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: "GitCredentials{Source: CI, User: test, ExpiresAt: 2024-01-01T00:00:00Z}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.creds.String())
		})
	}
}

func TestValidateRequiredSecrets(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		setupEnv  func(t *testing.T)
		wantErr   bool
	}{
		{
			name:      "push operation requires credentials",
			operation: "push",
			setupEnv: func(t *testing.T) {
				t.Helper()
				// No credentials set - use empty strings
				t.Setenv("GITHUB_TOKEN", "")
				t.Setenv("CI_JOB_TOKEN", "")
				t.Setenv("GITLAB_PAT", "")
				t.Setenv("SSH_PRIVATE_KEY", "")
				t.Setenv("CI", "")
				t.Setenv("GITHUB_ACTIONS", "")
				t.Setenv("GITLAB_CI", "")
			},
			wantErr: true,
		},
		{
			name:      "push operation with GitHub token",
			operation: "push",
			setupEnv: func(t *testing.T) {
				t.Helper()
				t.Setenv("GITHUB_TOKEN", "test-token")
			},
			wantErr: false,
		},
		{
			name:      "push operation with GitLab CI token",
			operation: "push",
			setupEnv: func(t *testing.T) {
				t.Helper()
				t.Setenv("CI", "true")
				t.Setenv("CI_JOB_TOKEN", "test-token")
			},
			wantErr: false,
		},
		{
			name:      "clone operation does not require credentials",
			operation: "clone",
			setupEnv: func(t *testing.T) {
				t.Helper()
				// No credentials set - use empty strings
				t.Setenv("GITHUB_TOKEN", "")
				t.Setenv("CI_JOB_TOKEN", "")
				t.Setenv("GITLAB_PAT", "")
				t.Setenv("SSH_PRIVATE_KEY", "")
				t.Setenv("CI", "")
			},
			wantErr: false,
		},
		{
			name:      "unknown operation does not require credentials",
			operation: "unknown",
			setupEnv: func(t *testing.T) {
				t.Helper()
				// No credentials set - use empty strings
				t.Setenv("GITHUB_TOKEN", "")
				t.Setenv("CI_JOB_TOKEN", "")
				t.Setenv("GITLAB_PAT", "")
				t.Setenv("SSH_PRIVATE_KEY", "")
				t.Setenv("CI", "")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv(t)

			err := ValidateRequiredSecrets(tt.operation)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "credentials required")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
