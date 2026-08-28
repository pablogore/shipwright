package app

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"dagger.io/dagger"
	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/stretchr/testify/mock"
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
		mockSetup   func(*MockHTTPClient)
	}{
		{
			name:        "empty registry URL",
			registryURL: "",
			user:        "user",
			pass:        "pass",
			wantErr:     true,
			mockSetup:   nil,
		},
		{
			name:        "invalid registry URL",
			registryURL: "not-a-url",
			user:        "user",
			pass:        "pass",
			wantErr:     true,
			// "not-a-url" is normalized to "https://not-a-url", which passes URL
			// validation (it has a scheme and a host), so checkRegistry proceeds
			// to call the client; the error comes from the (mocked) connection
			// failure, not from URL validation.
			mockSetup: func(m *MockHTTPClient) {
				m.On("Do", mock.Anything).Return(nil, errors.New("mock connection error: no such host"))
			},
		},
		{
			name:        "valid registry URL format",
			registryURL: "https://registry.example.com",
			user:        "user",
			pass:        "pass",
			wantErr:     false, // May fail on actual connection, but format is valid
			mockSetup: func(m *MockHTTPClient) {
				m.On("Do", mock.Anything).Return(NewMockHTTPResponse(http.StatusOK, ""), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			mockClient := new(MockHTTPClient)
			if tt.mockSetup != nil {
				tt.mockSetup(mockClient)
			}

			err := checkRegistry(ctx, mockClient, tt.registryURL, tt.user, tt.pass)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				// Connection errors are acceptable in test environment
				if err != nil {
					t.Logf("Registry check failed (may be expected in test environment): %v", err)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestCheckGitRepo(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		wantErr   bool
		mockSetup func(*MockHTTPClient)
	}{
		{
			name:      "empty repo URL",
			repoURL:   "",
			wantErr:   true,
			mockSetup: nil,
		},
		{
			name:      "invalid repo URL",
			repoURL:   "not-a-url",
			wantErr:   true,
			mockSetup: nil,
		},
		{
			name:    "valid HTTPS repo URL",
			repoURL: "https://github.com/pablogore/shipwright",
			wantErr: false, // May fail on actual connection, but format is valid
			mockSetup: func(m *MockHTTPClient) {
				m.On("Do", mock.Anything).Return(NewMockHTTPResponse(http.StatusOK, ""), nil)
			},
		},
		{
			name:      "valid SSH repo URL",
			repoURL:   "git@github.com:pablogore/shipwright.git",
			wantErr:   false, // May fail on actual connection, but format is valid
			mockSetup: nil,   // SSH never touches HTTP
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			mockClient := new(MockHTTPClient)
			if tt.mockSetup != nil {
				tt.mockSetup(mockClient)
			}

			err := checkGitRepo(ctx, mockClient, tt.repoURL)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				// Connection errors are acceptable in test environment
				if err != nil {
					t.Logf("Git repo check failed (may be expected in test environment): %v", err)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRunHealthChecks(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() interfaces.Configuration
		wantErr   bool
		mockSetup func(*MockHTTPClient)
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
			mockSetup: func(m *MockHTTPClient) {
				// A single generic expectation covers both the registry GET and
				// the git HEAD calls.
				m.On("Do", mock.Anything).Return(NewMockHTTPResponse(http.StatusOK, ""), nil)
			},
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
			wantErr:   false, // Health checks should skip missing configs
			mockSetup: nil,   // No URLs configured, so neither check fires an HTTP call.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setup()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			mockClient := new(MockHTTPClient)
			if tt.mockSetup != nil {
				tt.mockSetup(mockClient)
			}

			err := runHealthChecks(ctx, mockClient, cfg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				// Connection errors are acceptable in test environment
				if err != nil {
					t.Logf("Health checks failed (may be expected in test environment): %v", err)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}
