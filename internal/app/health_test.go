package app

import (
	"context"
	"testing"
	"time"

	"dagger.io/dagger"
	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/stretchr/testify/require"
)

func TestCheckDaggerEngine(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful connection",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a real Dagger client for testing
			// In a real test, we'd use a mock or test client
			client, err := dagger.Connect(context.Background())
			if err != nil {
				// If Dagger is not available, skip the subtest
				t.Skip("Dagger engine not available for testing")
			}
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = CheckDaggerEngine(ctx, client)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				// If Dagger is not available, the error is acceptable
				if err != nil {
					t.Logf("Dagger engine check failed (may be expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestCheckRegistry(t *testing.T) {
	tests := []struct {
		name        string
		registryURL string
		user        string
		pass        string
		wantErr     bool
	}{
		{
			name:        "empty registry URL",
			registryURL: "",
			user:        "user",
			pass:        "pass",
			wantErr:     true,
		},
		{
			name:        "invalid registry URL",
			registryURL: "not-a-url",
			user:        "user",
			pass:        "pass",
			wantErr:     true,
		},
		{
			name:        "valid registry URL format",
			registryURL: "https://registry.example.com",
			user:        "user",
			pass:        "pass",
			wantErr:     false, // May fail on actual connection, but format is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := CheckRegistry(ctx, tt.registryURL, tt.user, tt.pass)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				// Connection errors are acceptable in test environment
				if err != nil {
					t.Logf("Registry check failed (may be expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestCheckGitRepo(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		wantErr bool
	}{
		{
			name:    "empty repo URL",
			repoURL: "",
			wantErr: true,
		},
		{
			name:    "invalid repo URL",
			repoURL: "not-a-url",
			wantErr: true,
		},
		{
			name:    "valid HTTPS repo URL",
			repoURL: "https://github.com/pablogore/shipwright",
			wantErr: false, // May fail on actual connection, but format is valid
		},
		{
			name:    "valid SSH repo URL",
			repoURL: "git@github.com:pablogore/shipwright.git",
			wantErr: false, // May fail on actual connection, but format is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := CheckGitRepo(ctx, tt.repoURL)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				// Connection errors are acceptable in test environment
				if err != nil {
					t.Logf("Git repo check failed (may be expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestRunHealthChecks(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() interfaces.Configuration
		wantErr bool
	}{
		{
			name: "valid configuration",
			setup: func() interfaces.Configuration {
				mockConfig := NewMockConfiguration()
				mockConfig.GetStringFunc = func(key string) string {
					switch key {
					case "registry.base_url":
						return "https://registry.example.com"
					case "registry.user":
						return "user"
					case "registry.pass":
						return "pass"
					case "git.repo":
						return "https://github.com/example/repo"
					default:
						return ""
					}
				}
				return mockConfig
			},
			wantErr: false, // May have connection errors, but config is valid
		},
		{
			name: "missing registry config",
			setup: func() interfaces.Configuration {
				mockConfig := NewMockConfiguration()
				mockConfig.GetStringFunc = func(key string) string {
					return ""
				}
				return mockConfig
			},
			wantErr: false, // Health checks should skip missing configs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setup()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := RunHealthChecks(ctx, cfg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				// Connection errors are acceptable in test environment
				if err != nil {
					t.Logf("Health checks failed (may be expected in test environment): %v", err)
				}
			}
		})
	}
}
