package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name:    "default configuration",
			envVars: map[string]string{},
			wantErr: false,
		},
		{
			name: "with environment variables",
			envVars: map[string]string{
				"SYNTEGRITY_DAGGER_PIPELINE_NAME":         "test-pipeline",
				"SYNTEGRITY_DAGGER_PIPELINE_COVERAGE":     "95.5",
				"SYNTEGRITY_DAGGER_REGISTRY_BASE_URL":     "test-registry.com",
				"SYNTEGRITY_DAGGER_PIPELINE_GO_VERSION":   "1.21",
				"SYNTEGRITY_DAGGER_SECURITY_LINT_TIMEOUT": "10m",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			cfg, err := New()
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.Equal(t, DefaultEnvironment, cfg.Environment)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: &Config{
				Pipeline: PipelineConfig{
					Name:      "test-pipeline",
					GoVersion: "1.21",
					Coverage:  90.0,
				},
				Registry: RegistryConfig{
					BaseURL: "registry.test.com",
				},
			},
			wantErr: false,
		},
		{
			name: "missing pipeline name",
			config: &Config{
				Pipeline: PipelineConfig{
					GoVersion: "1.21",
					Coverage:  90.0,
				},
				Registry: RegistryConfig{
					BaseURL: "registry.test.com",
				},
			},
			wantErr: true,
			errMsg:  "pipeline name is required",
		},
		{
			name: "missing registry base URL",
			config: &Config{
				Pipeline: PipelineConfig{
					Name:      "test-pipeline",
					GoVersion: "1.21",
					Coverage:  90.0,
				},
				Registry: RegistryConfig{},
			},
			wantErr: true,
			errMsg:  "registry base URL is required",
		},
		{
			name: "missing Go version",
			config: &Config{
				Pipeline: PipelineConfig{
					Name:     "test-pipeline",
					Coverage: 90.0,
				},
				Registry: RegistryConfig{
					BaseURL: "registry.test.com",
				},
			},
			wantErr: true,
			errMsg:  "go version is required",
		},
		{
			name: "invalid coverage - negative",
			config: &Config{
				Pipeline: PipelineConfig{
					Name:      "test-pipeline",
					GoVersion: "1.21",
					Coverage:  -1.0,
				},
				Registry: RegistryConfig{
					BaseURL: "registry.test.com",
				},
			},
			wantErr: true,
			errMsg:  "coverage must be between 0 and 100",
		},
		{
			name: "invalid coverage - over 100",
			config: &Config{
				Pipeline: PipelineConfig{
					Name:      "test-pipeline",
					GoVersion: "1.21",
					Coverage:  101.0,
				},
				Registry: RegistryConfig{
					BaseURL: "registry.test.com",
				},
			},
			wantErr: true,
			errMsg:  "coverage must be between 0 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewConfigurationWrapper(t *testing.T) {
	// Test with clean environment
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, DefaultEnvironment, cfg.Environment())
}

func TestConfigurationWrapper_Load(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	err = cfg.Load()
	assert.NoError(t, err) // Should always succeed as config is already loaded
}

func TestConfigurationWrapper_LoadWithDefaults(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	defaults := map[string]any{
		"test_key": "test_value",
	}

	err = cfg.LoadWithDefaults(defaults)
	assert.NoError(t, err) // Not implemented, should not error
}

func TestConfigurationWrapper_GetConfigSummary(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	summary := cfg.GetConfigSummary()
	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, "Pipeline:")
	assert.Contains(t, summary, "Environment:")
	assert.Contains(t, summary, "Go Version:")
	assert.Contains(t, summary, "Registry:")
}

func TestConfigurationWrapper_GetString(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	tests := []struct {
		key    string
		expect string
	}{
		{KeyServiceName, "syntegrity-dagger"},
		{KeyServiceVersion, "1.0.0"},
		{KeyEnvironment, DefaultEnvironment},
		{KeyPipelineName, DefaultPipelineName},
		{KeyPipelineGoVersion, DefaultGoVersion},
		{KeyRegistryBaseURL, DefaultRegistryBaseURL},
		{KeyGitRef, DefaultGitRef},
		{KeyGitProtocol, DefaultGitProtocol},
		{KeyLogLevel, DefaultLogLevel},
		{KeyLogFormat, DefaultLogFormat},
		{"nonexistent_key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := cfg.GetString(tt.key)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestConfigurationWrapper_GetInt(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	result := cfg.GetInt("nonexistent_key")
	assert.Equal(t, 0, result)
}

func TestConfigurationWrapper_GetBool(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	tests := []struct {
		key    string
		expect bool
	}{
		{KeyPipelineSkipPush, false},
		{KeyPipelineOnlyBuild, false},
		{KeyPipelineOnlyTest, false},
		{KeyPipelineVerbose, false},
		{KeySecurityEnableVulnCheck, true},
		{KeySecurityEnableLinting, true},
		{KeyLogSamplingEnable, false},
		{KeyReleaseEnabled, false},
		{KeyReleaseUseGoreleaser, true},
		{KeyReleaseChecksum, true},
		{KeyReleaseSign, false},
		{KeyDaggerLogOutput, true},
		{"nonexistent_key", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := cfg.GetBool(tt.key)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestConfigurationWrapper_GetDuration(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	tests := []struct {
		key    string
		expect time.Duration
	}{
		{KeyLogSamplingInterval, 1 * time.Second},
		{KeyDaggerTimeout, DefaultDaggerTimeout},
		{"nonexistent_key", 0},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := cfg.GetDuration(tt.key)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestConfigurationWrapper_GetFloat(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	tests := []struct {
		key    string
		expect float64
	}{
		{KeyPipelineCoverage, DefaultCoverage},
		{KeyLogSamplingRate, 0.1},
		{"nonexistent_key", 0},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := cfg.GetFloat(tt.key)
			if tt.expect == 0 {
				assert.Zero(t, result)
			} else {
				assert.InEpsilon(t, tt.expect, result, 0.0001)
			}
		})
	}
}

func TestConfigurationWrapper_Get(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	tests := []struct {
		key    string
		expect any
	}{
		{KeySecurityExcludePatterns, []string{}},
		{KeyReleaseBuildTargets, []string{"linux/amd64", "darwin/amd64", "windows/amd64"}},
		{KeyReleaseArchiveFormats, []string{"tar.gz", "zip"}},
		{"nonexistent_key", nil},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := cfg.Get(tt.key)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestConfigurationWrapper_Set(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	// Set should not error (even though it's not implemented)
	cfg.Set("test_key", "test_value")
	assert.NoError(t, err)
}

func TestConfigurationWrapper_Has(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	tests := []struct {
		key    string
		expect bool
	}{
		{KeyServiceName, true},
		{KeyServiceVersion, true},
		{KeyEnvironment, true},
		{KeyPipelineName, true},
		{KeyPipelineGoVersion, true},
		{KeyPipelineJavaVersion, true},
		{KeyRegistryBaseURL, true},
		{KeyRegistryUser, true},
		{KeyGitRef, true},
		{KeyGitProtocol, true},
		{KeySecurityLintTimeout, true},
		{KeyLogLevel, true},
		{KeyLogFormat, true},
		{"nonexistent_key", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := cfg.Has(tt.key)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestConfigurationWrapper_All(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	all := cfg.All()
	assert.NotNil(t, all)
	assert.Contains(t, all, KeyServiceName)
	assert.Contains(t, all, KeyServiceVersion)
	assert.Contains(t, all, KeyEnvironment)
	assert.Contains(t, all, KeyPipelineName)
	assert.Contains(t, all, KeyPipelineGoVersion)
	assert.Contains(t, all, KeyRegistryBaseURL)
	assert.Contains(t, all, KeyGitRef)
	assert.Contains(t, all, KeyGitProtocol)
	assert.Contains(t, all, KeyLogLevel)
	assert.Contains(t, all, KeyLogFormat)
}

func TestConfigurationWrapper_Pipeline(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	pipeline := cfg.Pipeline()
	assert.Equal(t, DefaultPipelineName, pipeline.Name)
	assert.Equal(t, DefaultEnvironment, pipeline.Environment)
	assert.InEpsilon(t, DefaultCoverage, pipeline.Coverage, 0.0001)
	assert.Equal(t, DefaultGoVersion, pipeline.GoVersion)
	assert.Equal(t, DefaultJavaVersion, pipeline.JavaVersion)
}

func TestConfigurationWrapper_Registry(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	registry := cfg.Registry()
	assert.Equal(t, DefaultRegistryBaseURL, registry.BaseURL)
	assert.Equal(t, DefaultRegistryUser, registry.User)
}

func TestConfigurationWrapper_Security(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	security := cfg.Security()
	assert.True(t, security.EnableVulnCheck)
	assert.True(t, security.EnableLinting)
	assert.Equal(t, DefaultLintTimeout, security.LintTimeout)
	assert.NotNil(t, security.ExcludePatterns)
}

func TestConfigurationWrapper_Logging(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	logging := cfg.Logging()
	assert.Equal(t, DefaultLogLevel, logging.Level)
	assert.Equal(t, DefaultLogFormat, logging.Format)
	assert.False(t, logging.SamplingEnable)
	assert.InEpsilon(t, 0.1, logging.SamplingRate, 0.0001)
	assert.Equal(t, 1*time.Second, logging.SamplingInterval)
}

func TestConfigurationWrapper_Environment(t *testing.T) {
	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	env := cfg.Environment()
	assert.Equal(t, DefaultEnvironment, env)
}

func TestLoadDotEnv(t *testing.T) {
	tests := []struct {
		name    string
		envFile string
		content string
		wantErr bool
		setup   func() func()
	}{
		{
			name:    "no .env file",
			envFile: ".env",
			wantErr: false,
			setup: func() func() {
				// Remove .env file if it exists
				os.Remove(".env")
				return func() {}
			},
		},
		{
			name:    "valid .env file",
			envFile: ".env",
			content: "TEST_VAR=test_value\nANOTHER_VAR=another_value",
			wantErr: false,
			setup: func() func() {
				// Create .env file
				_ = os.WriteFile(".env", []byte("TEST_VAR=test_value\nANOTHER_VAR=another_value"), 0644)
				return func() {
					os.Remove(".env")
				}
			},
		},
		{
			name:    "invalid .env file",
			envFile: ".env",
			content: "INVALID_CONTENT_WITHOUT_EQUALS",
			wantErr: false, // godotenv.Load doesn't error on invalid content
			setup: func() func() {
				// Create invalid .env file
				_ = os.WriteFile(".env", []byte("INVALID_CONTENT_WITHOUT_EQUALS"), 0644)
				return func() {
					os.Remove(".env")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			err := loadDotEnv()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, DefaultEnvironment, cfg.Environment)
	assert.Equal(t, DefaultPipelineName, cfg.Pipeline.Name)
	assert.InEpsilon(t, DefaultCoverage, cfg.Pipeline.Coverage, 0.001)
	assert.Equal(t, DefaultGoVersion, cfg.Pipeline.GoVersion)
	assert.Equal(t, DefaultJavaVersion, cfg.Pipeline.JavaVersion)
	assert.Equal(t, DefaultRegistryBaseURL, cfg.Registry.BaseURL)
	assert.Equal(t, DefaultRegistryUser, cfg.Registry.User)
	assert.Equal(t, DefaultGitRef, cfg.Git.Ref)
	assert.Equal(t, DefaultGitProtocol, cfg.Git.Protocol)
	assert.True(t, cfg.Security.EnableVulnCheck)
	assert.True(t, cfg.Security.EnableLinting)
	assert.Equal(t, DefaultLintTimeout, cfg.Security.LintTimeout)
	assert.Equal(t, DefaultLogLevel, cfg.Logging.Level)
	assert.Equal(t, DefaultLogFormat, cfg.Logging.Format)
	assert.False(t, cfg.Release.Enabled)
	assert.True(t, cfg.Release.UseGoreleaser)
	assert.True(t, cfg.Dagger.LogOutput)
	assert.Equal(t, DefaultDaggerTimeout, cfg.Dagger.Timeout)
}
