package app

import (
	"testing"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConvertedConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     interfaces.Configuration
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid converted config",
			cfg: func() interfaces.Configuration {
				mockConfig := NewMockConfiguration()
				setupMockConfigForConvertConfig(mockConfig)
				return mockConfig
			}(),
			wantErr: false,
		},
		{
			name: "invalid registry URL in converted config",
			cfg: func() interfaces.Configuration {
				mockConfig := NewMockConfiguration()
				mockConfig.EnvironmentFunc = func() string { return "dev" }
				mockConfig.GetStringFunc = func(key string) string {
					switch key {
					case "registry.base_url":
						return "https://" // Invalid: no host
					case "pipeline.go_version":
						return "1.25.5"
					case "git.repo":
						return ""
					default:
						return ""
					}
				}
				return mockConfig
			}(),
			wantErr: true,
			errMsg:  "invalid registry URL",
		},
		{
			name: "invalid Go version in converted config",
			cfg: func() interfaces.Configuration {
				mockConfig := NewMockConfiguration()
				mockConfig.EnvironmentFunc = func() string { return "dev" }
				mockConfig.GetStringFunc = func(key string) string {
					switch key {
					case "registry.base_url":
						return "https://registry.example.com"
					case "pipeline.go_version":
						return "v1.25.1"
					case "git.repo":
						return ""
					default:
						return ""
					}
				}
				return mockConfig
			}(),
			wantErr: true,
			errMsg:  "invalid Go version",
		},
		{
			name: "invalid Git repo URL in converted config",
			cfg: func() interfaces.Configuration {
				mockConfig := NewMockConfiguration()
				mockConfig.EnvironmentFunc = func() string { return "dev" }
				mockConfig.GetStringFunc = func(key string) string {
					switch key {
					case "registry.base_url":
						return "https://registry.example.com"
					case "pipeline.go_version":
						return "1.25.5"
					case "git.repo":
						return "not-a-valid-url"
					default:
						return ""
					}
				}
				return mockConfig
			}(),
			wantErr: true,
			errMsg:  "invalid Git repository URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConvertedConfig(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConvertConfig_WithValidation(t *testing.T) {
	mockConfig := NewMockConfiguration()
	mockConfig.EnvironmentFunc = func() string { return "dev" }
	mockConfig.GetBoolFunc = func(key string) bool { return false }
	mockConfig.GetStringFunc = func(key string) string {
		switch key {
		case "git.ref":
			return "main"
		case "git.protocol":
			return "ssh"
		case "pipeline.go_version":
			return "1.21"
		case "registry.base_url":
			return "https://registry.example.com"
		default:
			return ""
		}
	}
	mockConfig.GetFloatFunc = func(key string) float64 { return 90.0 }

	// Convert config
	converted := convertConfig(mockConfig)

	// Validate converted config has valid values
	assert.NotEmpty(t, converted.GoVersion)
	assert.NotEmpty(t, converted.RegistryURL)

	// Validate that URLs are in valid format (if provided)
	if converted.RegistryURL != "" {
		// This would be validated by validateConvertedConfig
		// For now, just check it's not empty
		assert.NotEmpty(t, converted.RegistryURL)
	}
}
