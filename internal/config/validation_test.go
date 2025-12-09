package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateRegistryURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid HTTPS registry URL",
			url:     "https://registry.example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTP registry URL",
			url:     "http://localhost:5000",
			wantErr: false,
		},
		{
			name:    "valid registry URL with path",
			url:     "https://ghcr.io/v2",
			wantErr: false,
		},
		{
			name:    "valid registry URL with port",
			url:     "https://registry.example.com:5000",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "URL without scheme (normalized to https://)",
			url:     "registry.example.com",
			wantErr: false,
		},
		{
			name:    "URL without scheme - backward compatibility (test-registry)",
			url:     "test-registry",
			wantErr: false,
		},
		{
			name:    "URL format (normalized, may be invalid hostname but passes url.Parse)",
			url:     "not-a-url",
			wantErr: false, // url.Parse accepts this as valid after normalization
		},
		{
			name:    "URL without host",
			url:     "https://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistryURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGitRepoURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid HTTPS Git URL",
			url:     "https://github.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid HTTPS Git URL without .git",
			url:     "https://github.com/user/repo",
			wantErr: false,
		},
		{
			name:    "valid SSH Git URL",
			url:     "git@github.com:user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid SSH Git URL with custom host",
			url:     "git@gitlab.com:group/project.git",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "invalid URL format",
			url:     "not-a-url",
			wantErr: true,
		},
		{
			name:    "HTTPS URL without host",
			url:     "https://",
			wantErr: true,
		},
		{
			name:    "SSH URL without colon",
			url:     "git@github.com",
			wantErr: true,
		},
		{
			name:    "SSH URL without path",
			url:     "git@github.com:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitRepoURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGoVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{
			name:    "valid version X.Y.Z",
			version: "1.25.5",
			wantErr: false,
		},
		{
			name:    "valid version X.Y",
			version: "1.25",
			wantErr: false,
		},
		{
			name:    "valid version with patch zero",
			version: "1.25.0",
			wantErr: false,
		},
		{
			name:    "valid version single digit",
			version: "1.2",
			wantErr: false,
		},
		{
			name:    "empty version",
			version: "",
			wantErr: true,
		},
		{
			name:    "invalid format with letters",
			version: "1.25a",
			wantErr: true,
		},
		{
			name:    "invalid format with prefix",
			version: "v1.25.1",
			wantErr: true,
		},
		{
			name:    "invalid format with suffix",
			version: "1.25.1-beta",
			wantErr: true,
		},
		{
			name:    "invalid format single number",
			version: "1",
			wantErr: true,
		},
		{
			name:    "invalid format too many parts",
			version: "1.25.1.2",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGoVersion(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		wantErr bool
	}{
		{
			name:    "valid dev",
			env:     "dev",
			wantErr: false,
		},
		{
			name:    "valid development",
			env:     "development",
			wantErr: false,
		},
		{
			name:    "valid staging",
			env:     "staging",
			wantErr: false,
		},
		{
			name:    "valid production",
			env:     "production",
			wantErr: false,
		},
		{
			name:    "valid prod",
			env:     "prod",
			wantErr: false,
		},
		{
			name:    "empty environment",
			env:     "",
			wantErr: true,
		},
		{
			name:    "invalid environment",
			env:     "invalid",
			wantErr: true,
		},
		{
			name:    "case sensitive invalid",
			env:     "DEV",
			wantErr: true,
		},
		{
			name:    "invalid with numbers",
			env:     "dev1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvironment(tt.env)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}


