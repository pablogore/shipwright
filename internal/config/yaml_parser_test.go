package config

import (
	"os"
	"testing"

	"github.com/getsyntegrity/syntegrity-dagger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Helper function to create a test YAMLConfig
func createTestYAMLConfig(steps []string) *YAMLConfig {
	return &YAMLConfig{
		Pipeline: struct {
			Name        string   `yaml:"name"`
			Environment string   `yaml:"environment"`
			Coverage    float64  `yaml:"coverage"`
			GoVersion   string   `yaml:"goVersion"`
			Steps       []string `yaml:"steps"`
		}{
			Name:        "test-pipeline",
			Environment: "dev",
			Coverage:    95.0,
			GoVersion:   "1.21",
			Steps:       steps,
		},
	}
}

func TestNewYAMLParser(t *testing.T) {
	parser := NewYAMLParser()
	assert.NotNil(t, parser)
}

func TestYAMLParser_ParseFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		filePath    string
		wantErr     bool
		errContains string
		setup       func() func()
	}{
		{
			name:     "valid YAML file",
			content:  "pipeline:\n  name: test-pipeline\n  environment: dev\n  coverage: 95.0\n  go_version: 1.21\nsteps:\n  - setup\n  - build\n  - test\nregistry:\n  base_url: registry.test.com\n  image: test-image\n  user: test-user\nsecurity:\n  enable_vuln_check: true\n  enable_linting: true\nrelease:\n  enabled: false\n  use_goreleaser: true\n  create_github_release: false\n  platforms:\n    - linux/amd64\nlogging:\n  level: info",
			filePath: "test-config.yml",
			wantErr:  false,
			setup: func() func() {
				_ = os.WriteFile("test-config.yml", []byte("pipeline:\n  name: test-pipeline\n  environment: dev\n  coverage: 95.0\n  go_version: 1.21\nsteps:\n  - setup\n  - build\n  - test\nregistry:\n  base_url: registry.test.com\n  image: test-image\n  user: test-user\nsecurity:\n  enable_vuln_check: true\n  enable_linting: true\nrelease:\n  enabled: false\n  use_goreleaser: true\n  create_github_release: false\n  platforms:\n    - linux/amd64\nlogging:\n  level: info"), 0644)
				return func() { os.Remove("test-config.yml") }
			},
		},
		{
			name:        "file not found",
			filePath:    "nonexistent.yml",
			wantErr:     true,
			errContains: "configuration file not found",
			setup: func() func() {
				return func() {}
			},
		},
		{
			name:        "invalid YAML",
			content:     "invalid: yaml: content: [",
			filePath:    "invalid.yml",
			wantErr:     true,
			errContains: "failed to parse YAML configuration",
			setup: func() func() {
				_ = os.WriteFile("invalid.yml", []byte("invalid: yaml: content: ["), 0644)
				return func() { os.Remove("invalid.yml") }
			},
		},
		{
			name:     "empty file",
			content:  "",
			filePath: "empty.yml",
			wantErr:  false,
			setup: func() func() {
				_ = os.WriteFile("empty.yml", []byte(""), 0644)
				return func() { os.Remove("empty.yml") }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			parser := NewYAMLParser()
			config, err := parser.ParseFile(tt.filePath)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, config)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, config)
			}
		})
	}
}

func TestYAMLParser_ApplyToConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)

	parser := NewYAMLParser()
	yamlConfig := &YAMLConfig{
		Pipeline: struct {
			Name        string   `yaml:"name"`
			Environment string   `yaml:"environment"`
			Coverage    float64  `yaml:"coverage"`
			GoVersion   string   `yaml:"goVersion"`
			Steps       []string `yaml:"steps"`
		}{
			Name:        "test-pipeline",
			Environment: "dev",
			Coverage:    95.0,
			GoVersion:   "1.21",
			Steps:       []string{"setup", "build", "test"},
		},
		Registry: struct {
			BaseURL string `yaml:"baseUrl"`
			Image   string `yaml:"image"`
			User    string `yaml:"user"`
		}{
			BaseURL: "registry.test.com",
			Image:   "test-image",
			User:    "test-user",
		},
		Security: struct {
			EnableVulnCheck bool `yaml:"enableVulnCheck"`
			EnableLinting   bool `yaml:"enableLinting"`
		}{
			EnableVulnCheck: true,
			EnableLinting:   true,
		},
		Release: struct {
			Enabled             bool     `yaml:"enabled"`
			UseGoreleaser       bool     `yaml:"useGoreleaser"`
			CreateGithubRelease bool     `yaml:"createGithubRelease"`
			Platforms           []string `yaml:"platforms"`
		}{
			Enabled:             false,
			UseGoreleaser:       true,
			CreateGithubRelease: false,
			Platforms:           []string{"linux/amd64"},
		},
		Logging: struct {
			Level string `yaml:"level"`
		}{
			Level: "info",
		},
	}

	// Set up expectations
	mockConfig.EXPECT().Set("pipeline.name", "test-pipeline").Times(1)
	mockConfig.EXPECT().Set("pipeline.environment", "dev").Times(1)
	mockConfig.EXPECT().Set("pipeline.coverage", 95.0).Times(1)
	mockConfig.EXPECT().Set("pipeline.go_version", "1.21").Times(1)
	mockConfig.EXPECT().Set("registry.base_url", "registry.test.com").Times(1)
	mockConfig.EXPECT().Set("registry.image", "test-image").Times(1)
	mockConfig.EXPECT().Set("registry.user", "test-user").Times(1)
	mockConfig.EXPECT().Set("security.enable_vuln_check", true).Times(1)
	mockConfig.EXPECT().Set("security.enable_linting", true).Times(1)
	mockConfig.EXPECT().Set("release.enabled", false).Times(1)
	mockConfig.EXPECT().Set("release.use_goreleaser", true).Times(1)
	mockConfig.EXPECT().Set("release.create_github_release", false).Times(1)
	mockConfig.EXPECT().Set("logging.level", "info").Times(1)

	err := parser.ApplyToConfiguration(yamlConfig, mockConfig)
	require.NoError(t, err)
}

func TestYAMLParser_ApplyToConfiguration_EmptyValues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfiguration(ctrl)

	parser := NewYAMLParser()
	yamlConfig := &YAMLConfig{
		Pipeline: struct {
			Name        string   `yaml:"name"`
			Environment string   `yaml:"environment"`
			Coverage    float64  `yaml:"coverage"`
			GoVersion   string   `yaml:"goVersion"`
			Steps       []string `yaml:"steps"`
		}{
			Name:        "",
			Environment: "",
			Coverage:    0,
			GoVersion:   "",
			Steps:       []string{},
		},
		Registry: struct {
			BaseURL string `yaml:"baseUrl"`
			Image   string `yaml:"image"`
			User    string `yaml:"user"`
		}{
			BaseURL: "",
			Image:   "",
			User:    "",
		},
		Logging: struct {
			Level string `yaml:"level"`
		}{
			Level: "",
		},
	}

	// Only security and release settings should be set (they have default values)
	mockConfig.EXPECT().Set("security.enable_vuln_check", false).Times(1)
	mockConfig.EXPECT().Set("security.enable_linting", false).Times(1)
	mockConfig.EXPECT().Set("release.enabled", false).Times(1)
	mockConfig.EXPECT().Set("release.use_goreleaser", false).Times(1)
	mockConfig.EXPECT().Set("release.create_github_release", false).Times(1)

	err := parser.ApplyToConfiguration(yamlConfig, mockConfig)
	require.NoError(t, err)
}

func TestYAMLParser_GetSteps(t *testing.T) {
	parser := NewYAMLParser()
	yamlConfig := createTestYAMLConfig([]string{"setup", "build", "test", "lint"})

	steps := parser.GetSteps(yamlConfig)
	assert.Equal(t, []string{"setup", "build", "test", "lint"}, steps)
}

func TestYAMLParser_GetSteps_Empty(t *testing.T) {
	parser := NewYAMLParser()
	yamlConfig := createTestYAMLConfig([]string{})

	steps := parser.GetSteps(yamlConfig)
	assert.Equal(t, []string{}, steps)
}

func TestYAMLParser_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *YAMLConfig
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid configuration",
			config:  createTestYAMLConfig([]string{"setup", "build", "test"}),
			wantErr: false,
		},
		{
			name: "missing pipeline name",
			config: &YAMLConfig{
				Pipeline: struct {
					Name        string   `yaml:"name"`
					Environment string   `yaml:"environment"`
					Coverage    float64  `yaml:"coverage"`
					GoVersion   string   `yaml:"goVersion"`
					Steps       []string `yaml:"steps"`
				}{
					Name:  "",
					Steps: []string{"setup", "build", "test"},
				},
			},
			wantErr:     true,
			errContains: "pipeline name is required",
		},
		{
			name:        "no steps defined",
			config:      createTestYAMLConfig([]string{}),
			wantErr:     true,
			errContains: "at least one step must be defined",
		},
		{
			name:        "invalid step",
			config:      createTestYAMLConfig([]string{"setup", "invalid-step", "test"}),
			wantErr:     true,
			errContains: "invalid step: invalid-step",
		},
		{
			name:    "all valid steps",
			config:  createTestYAMLConfig([]string{"setup", "build", "test", "lint", "security", "tag", "package", "push", "release"}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewYAMLParser()
			err := parser.ValidateConfig(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestYAMLParser_FindConfigFile(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() func()
		wantErr      bool
		errContains  string
		expectedFile string
	}{
		{
			name: "find .syntegrity-dagger.yml in current directory",
			setup: func() func() {
				_ = os.WriteFile(".syntegrity-dagger.yml", []byte("test"), 0644)
				return func() { os.Remove(".syntegrity-dagger.yml") }
			},
			wantErr:      false,
			expectedFile: ".syntegrity-dagger.yml",
		},
		{
			name: "find .syntegrity-dagger.yaml in current directory",
			setup: func() func() {
				_ = os.WriteFile(".syntegrity-dagger.yaml", []byte("test"), 0644)
				return func() { os.Remove(".syntegrity-dagger.yaml") }
			},
			wantErr:      false,
			expectedFile: ".syntegrity-dagger.yaml",
		},
		{
			name: "find syntegrity-dagger.yml in current directory",
			setup: func() func() {
				_ = os.WriteFile("syntegrity-dagger.yml", []byte("test"), 0644)
				return func() { os.Remove("syntegrity-dagger.yml") }
			},
			wantErr:      false,
			expectedFile: "syntegrity-dagger.yml",
		},
		{
			name: "find syntegrity-dagger.yaml in current directory",
			setup: func() func() {
				_ = os.WriteFile("syntegrity-dagger.yaml", []byte("test"), 0644)
				return func() { os.Remove("syntegrity-dagger.yaml") }
			},
			wantErr:      false,
			expectedFile: "syntegrity-dagger.yaml",
		},
		{
			name: "find config in .github directory",
			setup: func() func() {
				_ = os.MkdirAll(".github", 0755)
				_ = os.WriteFile(".github/syntegrity-dagger.yml", []byte("test"), 0644)
				return func() {
					os.Remove(".github/syntegrity-dagger.yml")
					os.Remove(".github")
				}
			},
			wantErr:      false,
			expectedFile: ".github/syntegrity-dagger.yml",
		},
		{
			name: "no config file found",
			setup: func() func() {
				// Remove any existing config files
				configFiles := []string{
					".syntegrity-dagger.yml",
					".syntegrity-dagger.yaml",
					"syntegrity-dagger.yml",
					"syntegrity-dagger.yaml",
				}
				for _, file := range configFiles {
					os.Remove(file)
				}
				// Return a cleanup function that restores the config file
				return func() {
					// Restore the config file for other tests
					configContent := `pipeline:
  name: go-kit
  steps:
    - setup
    - build
    - test
  coverage: 90
  skip_push: false
  only_build: false
  only_test: false
  verbose: false
  go_version: "1.21"

service:
  name: "test-service"
  version: "1.0.0"
  environment: "dev"

registry:
  base_url: "test-registry"
  user: "test-user"
  pass: "test-pass"
  image: "test-image"
  tag: "test-tag"

git:
  repo: "test-repo"
  ref: "main"
  protocol: "https"
  user_email: "test@example.com"
  user_name: "test-user"
  ssh_key: ""

security:
  enable_vuln_check: true
  enable_linting: true
  lint_timeout: "1m"
  exclude_patterns: []

logging:
  level: "info"
  format: "json"
  sampling_enable: false
  sampling_rate: 1.0
  sampling_interval: "1s"

release:
  enabled: false
  use_goreleaser: false
  build_targets: []
  archive_formats: []
  checksum: false
  sign: false

dagger:
  log_output: false
  timeout: "1m"`
					_ = os.WriteFile(".syntegrity-dagger.yml", []byte(configContent), 0644)
				}
			},
			wantErr:     true,
			errContains: "no configuration file found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "no config file found" {
				t.Skip("Skipping test that requires no config file to be present")
			}
			cleanup := tt.setup()
			defer cleanup()

			parser := NewYAMLParser()
			filePath, err := parser.FindConfigFile()

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, filePath)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedFile, filePath)
			}
		})
	}
}

func TestYAMLParser_FindConfigFile_Priority(t *testing.T) {
	// Test that files are found in the correct priority order
	parser := NewYAMLParser()

	// Create multiple config files to test priority
	_ = os.WriteFile(".syntegrity-dagger.yaml", []byte("test"), 0644)
	defer os.Remove(".syntegrity-dagger.yaml")

	_ = os.WriteFile("syntegrity-dagger.yml", []byte("test"), 0644)
	defer os.Remove("syntegrity-dagger.yml")

	filePath, err := parser.FindConfigFile()
	require.NoError(t, err)
	// Should find .syntegrity-dagger.yaml first (higher priority)
	assert.Equal(t, ".syntegrity-dagger.yaml", filePath)
}

func TestYAMLParser_FindConfigFile_ParentDirectories(t *testing.T) {
	// This test would require creating a more complex directory structure
	// For now, we'll test the basic functionality
	parser := NewYAMLParser()

	// Create a config file in current directory
	err := os.WriteFile(".syntegrity-dagger.yml", []byte("test"), 0644)
	require.NoError(t, err)
	defer os.Remove(".syntegrity-dagger.yml")

	filePath, err := parser.FindConfigFile()
	require.NoError(t, err)
	assert.Equal(t, ".syntegrity-dagger.yml", filePath)
}

func TestYAMLConfig_Structure(t *testing.T) {
	// Test that YAMLConfig has the expected structure
	config := &YAMLConfig{
		Pipeline: struct {
			Name        string   `yaml:"name"`
			Environment string   `yaml:"environment"`
			Coverage    float64  `yaml:"coverage"`
			GoVersion   string   `yaml:"goVersion"`
			Steps       []string `yaml:"steps"`
		}{
			Name:        "test",
			Environment: "dev",
			Coverage:    90.0,
			GoVersion:   "1.21",
			Steps:       []string{"setup", "build"},
		},
		Registry: struct {
			BaseURL string `yaml:"baseUrl"`
			Image   string `yaml:"image"`
			User    string `yaml:"user"`
		}{
			BaseURL: "registry.test.com",
			Image:   "test-image",
			User:    "test-user",
		},
		Security: struct {
			EnableVulnCheck bool `yaml:"enableVulnCheck"`
			EnableLinting   bool `yaml:"enableLinting"`
		}{
			EnableVulnCheck: true,
			EnableLinting:   true,
		},
		Release: struct {
			Enabled             bool     `yaml:"enabled"`
			UseGoreleaser       bool     `yaml:"useGoreleaser"`
			CreateGithubRelease bool     `yaml:"createGithubRelease"`
			Platforms           []string `yaml:"platforms"`
		}{
			Enabled:             false,
			UseGoreleaser:       true,
			CreateGithubRelease: false,
			Platforms:           []string{"linux/amd64"},
		},
		Logging: struct {
			Level string `yaml:"level"`
		}{
			Level: "info",
		},
	}

	assert.Equal(t, "test", config.Pipeline.Name)
	assert.Equal(t, "dev", config.Pipeline.Environment)
	assert.InEpsilon(t, 90.0, config.Pipeline.Coverage, 0.001)
	assert.Equal(t, "1.21", config.Pipeline.GoVersion)
	assert.Equal(t, []string{"setup", "build"}, config.Pipeline.Steps)
	assert.Equal(t, "registry.test.com", config.Registry.BaseURL)
	assert.Equal(t, "test-image", config.Registry.Image)
	assert.Equal(t, "test-user", config.Registry.User)
	assert.True(t, config.Security.EnableVulnCheck)
	assert.True(t, config.Security.EnableLinting)
	assert.False(t, config.Release.Enabled)
	assert.True(t, config.Release.UseGoreleaser)
	assert.False(t, config.Release.CreateGithubRelease)
	assert.Equal(t, []string{"linux/amd64"}, config.Release.Platforms)
	assert.Equal(t, "info", config.Logging.Level)
}
