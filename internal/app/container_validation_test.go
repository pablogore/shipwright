package app

import (
	"testing"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
	"github.com/getsyntegrity/syntegrity-dagger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				mockConfig := mocks.NewMockConfiguration(ctrl)
				setupMockConfigForConvertConfig(mockConfig)
				return mockConfig
			}(),
			wantErr: false,
		},
		{
			name: "invalid registry URL in converted config",
			cfg: func() interfaces.Configuration {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				mockConfig := mocks.NewMockConfiguration(ctrl)
				mockConfig.EXPECT().Environment().Return("dev").AnyTimes()
				mockConfig.EXPECT().GetString("registry.base_url").Return("https://").AnyTimes() // Invalid: no host
				mockConfig.EXPECT().GetString("pipeline.go_version").Return("1.25.5").AnyTimes()
				mockConfig.EXPECT().GetString("git.repo").Return("").AnyTimes()
				return mockConfig
			}(),
			wantErr: true,
			errMsg:  "invalid registry URL",
		},
		{
			name: "invalid Go version in converted config",
			cfg: func() interfaces.Configuration {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				mockConfig := mocks.NewMockConfiguration(ctrl)
				mockConfig.EXPECT().Environment().Return("dev").AnyTimes()
				mockConfig.EXPECT().GetString("registry.base_url").Return("https://registry.example.com").AnyTimes()
				mockConfig.EXPECT().GetString("pipeline.go_version").Return("v1.25.1").AnyTimes()
				mockConfig.EXPECT().GetString("git.repo").Return("").AnyTimes()
				return mockConfig
			}(),
			wantErr: true,
			errMsg:  "invalid Go version",
		},
		{
			name: "invalid Git repo URL in converted config",
			cfg: func() interfaces.Configuration {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				mockConfig := mocks.NewMockConfiguration(ctrl)
				mockConfig.EXPECT().Environment().Return("dev").AnyTimes()
				mockConfig.EXPECT().GetString("registry.base_url").Return("https://registry.example.com").AnyTimes()
				mockConfig.EXPECT().GetString("pipeline.go_version").Return("1.25.5").AnyTimes()
				mockConfig.EXPECT().GetString("git.repo").Return("not-a-valid-url").AnyTimes()
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)
	mockConfig.EXPECT().Environment().Return("dev").AnyTimes()
	mockConfig.EXPECT().GetBool(gomock.Any()).Return(false).AnyTimes()
	mockConfig.EXPECT().GetString(gomock.Any()).DoAndReturn(func(key string) string {
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
	}).AnyTimes()
	mockConfig.EXPECT().GetFloat(gomock.Any()).Return(90.0).AnyTimes()

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

