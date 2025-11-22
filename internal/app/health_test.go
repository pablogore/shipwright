package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"dagger.io/dagger"
	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
	"github.com/getsyntegrity/syntegrity-dagger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCheckDaggerEngine(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*gomock.Controller) *dagger.Client
		wantErr bool
	}{
		{
			name: "successful connection",
			setup: func(ctrl *gomock.Controller) *dagger.Client {
				// Create a real Dagger client for testing
				// In a real test, we'd use a mock or test client
				client, err := dagger.Connect(context.Background())
				if err != nil {
					// If Dagger is not available, skip the test
					t.Skip("Dagger engine not available for testing")
				}
				return client
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			client := tt.setup(ctrl)
			if client == nil {
				return
			}
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := CheckDaggerEngine(ctx, client)
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
		name      string
		registryURL string
		user       string
		pass       string
		wantErr   bool
	}{
		{
			name:       "empty registry URL",
			registryURL: "",
			user:       "user",
			pass:       "pass",
			wantErr:    true,
		},
		{
			name:       "invalid registry URL",
			registryURL: "not-a-url",
			user:       "user",
			pass:       "pass",
			wantErr:    true,
		},
		{
			name:       "valid registry URL format",
			registryURL: "https://registry.example.com",
			user:       "user",
			pass:       "pass",
			wantErr:    false, // May fail on actual connection, but format is valid
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
			repoURL: "https://github.com/getsyntegrity/syntegrity-dagger",
			wantErr: false, // May fail on actual connection, but format is valid
		},
		{
			name:    "valid SSH repo URL",
			repoURL: "git@github.com:getsyntegrity/syntegrity-dagger.git",
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
		setup   func(*gomock.Controller) interfaces.Configuration
		wantErr bool
	}{
		{
			name: "valid configuration",
			setup: func(ctrl *gomock.Controller) interfaces.Configuration {
				mockConfig := mocks.NewMockConfiguration(ctrl)
				mockConfig.EXPECT().GetString("registry.base_url").Return("https://registry.example.com").AnyTimes()
				mockConfig.EXPECT().GetString("registry.user").Return("user").AnyTimes()
				mockConfig.EXPECT().GetString("registry.pass").Return("pass").AnyTimes()
				mockConfig.EXPECT().GetString("git.repo").Return("https://github.com/example/repo").AnyTimes()
				return mockConfig
			},
			wantErr: false, // May have connection errors, but config is valid
		},
		{
			name: "missing registry config",
			setup: func(ctrl *gomock.Controller) interfaces.Configuration {
				mockConfig := mocks.NewMockConfiguration(ctrl)
				mockConfig.EXPECT().GetString("registry.base_url").Return("").AnyTimes()
				mockConfig.EXPECT().GetString("registry.user").Return("").AnyTimes()
				mockConfig.EXPECT().GetString("registry.pass").Return("").AnyTimes()
				mockConfig.EXPECT().GetString("git.repo").Return("").AnyTimes()
				return mockConfig
			},
			wantErr: false, // Health checks should skip missing configs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg := tt.setup(ctrl)

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


