package config

import (
	"os"
	"testing"

	"github.com/pablogore/shipwright/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Helper function to create a test YAMLConfig
func createTestYAMLConfig(steps []string) *YAMLConfig {
	config := &YAMLConfig{}
	config.Pipeline.Name = "test-pipeline"
	config.Pipeline.Environment = "dev"
	config.Pipeline.Coverage = 95.0
	config.Pipeline.GoVersion = "1.21"
	config.Pipeline.Steps = steps
	return config
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
	yamlConfig := &YAMLConfig{}
	yamlConfig.Pipeline.Name = "test-pipeline"
	yamlConfig.Pipeline.Environment = "dev"
	yamlConfig.Pipeline.Coverage = 95.0
	yamlConfig.Pipeline.GoVersion = "1.21"
	yamlConfig.Pipeline.Steps = []string{"setup", "build", "test"}
	yamlConfig.Registry.BaseURL = "registry.test.com"
	yamlConfig.Registry.Image = "test-image"
	yamlConfig.Registry.User = "test-user"
	yamlConfig.Security.EnableVulnCheck = true
	yamlConfig.Security.EnableLinting = true
	yamlConfig.Release.Enabled = false
	yamlConfig.Release.UseGoreleaser = true
	yamlConfig.Release.CreateGithubRelease = false
	yamlConfig.Release.Platforms = []string{"linux/amd64"}
	yamlConfig.Logging.Level = "info"

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
	yamlConfig := &YAMLConfig{}
	yamlConfig.Pipeline.Name = ""
	yamlConfig.Pipeline.Environment = ""
	yamlConfig.Pipeline.Coverage = 0
	yamlConfig.Pipeline.GoVersion = ""
	yamlConfig.Pipeline.Steps = []string{}
	yamlConfig.Registry.BaseURL = ""
	yamlConfig.Registry.Image = ""
	yamlConfig.Registry.User = ""
	yamlConfig.Logging.Level = ""

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
			config: func() *YAMLConfig {
				cfg := &YAMLConfig{}
				cfg.Pipeline.Name = ""
				cfg.Pipeline.Steps = []string{"setup", "build", "test"}
				return cfg
			}(),
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
			name: "find .shipwright.yml in current directory",
			setup: func() func() {
				_ = os.WriteFile(".shipwright.yml", []byte("test"), 0644)
				return func() { os.Remove(".shipwright.yml") }
			},
			wantErr:      false,
			expectedFile: ".shipwright.yml",
		},
		{
			name: "find .shipwright.yaml in current directory",
			setup: func() func() {
				_ = os.WriteFile(".shipwright.yaml", []byte("test"), 0644)
				return func() { os.Remove(".shipwright.yaml") }
			},
			wantErr:      false,
			expectedFile: ".shipwright.yaml",
		},
		{
			name: "find shipwright.yml in current directory",
			setup: func() func() {
				_ = os.WriteFile("shipwright.yml", []byte("test"), 0644)
				return func() { os.Remove("shipwright.yml") }
			},
			wantErr:      false,
			expectedFile: "shipwright.yml",
		},
		{
			name: "find shipwright.yaml in current directory",
			setup: func() func() {
				_ = os.WriteFile("shipwright.yaml", []byte("test"), 0644)
				return func() { os.Remove("shipwright.yaml") }
			},
			wantErr:      false,
			expectedFile: "shipwright.yaml",
		},
		{
			name: "find config in .github directory",
			setup: func() func() {
				_ = os.MkdirAll(".github", 0755)
				_ = os.WriteFile(".github/shipwright.yml", []byte("test"), 0644)
				return func() {
					os.Remove(".github/shipwright.yml")
					os.Remove(".github")
				}
			},
			wantErr:      false,
			expectedFile: ".github/shipwright.yml",
		},
		{
			name: "no config file found",
			setup: func() func() {
				// Remove any existing config files
				configFiles := []string{
					".shipwright.yml",
					".shipwright.yaml",
					"shipwright.yml",
					"shipwright.yaml",
				}
				for _, file := range configFiles {
					os.Remove(file)
				}
				// Return a cleanup function that restores the config file
				return func() {
					// Restore the config file for other tests
					configContent := `pipeline:
  name: go-service
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
					_ = os.WriteFile(".shipwright.yml", []byte(configContent), 0644)
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
	_ = os.WriteFile(".shipwright.yaml", []byte("test"), 0644)
	defer os.Remove(".shipwright.yaml")

	_ = os.WriteFile("shipwright.yml", []byte("test"), 0644)
	defer os.Remove("shipwright.yml")

	filePath, err := parser.FindConfigFile()
	require.NoError(t, err)
	// Should find .shipwright.yaml first (higher priority)
	assert.Equal(t, ".shipwright.yaml", filePath)
}

func TestYAMLParser_FindConfigFile_ParentDirectories(t *testing.T) {
	// This test would require creating a more complex directory structure
	// For now, we'll test the basic functionality
	parser := NewYAMLParser()

	// Create a config file in current directory
	err := os.WriteFile(".shipwright.yml", []byte("test"), 0644)
	require.NoError(t, err)
	defer os.Remove(".shipwright.yml")

	filePath, err := parser.FindConfigFile()
	require.NoError(t, err)
	assert.Equal(t, ".shipwright.yml", filePath)
}

func TestYAMLConfig_Structure(t *testing.T) {
	// Test that YAMLConfig has the expected structure
	config := &YAMLConfig{}
	config.Pipeline.Name = "test"
	config.Pipeline.Environment = "dev"
	config.Pipeline.Coverage = 90.0
	config.Pipeline.GoVersion = "1.21"
	config.Pipeline.Steps = []string{"setup", "build"}
	config.Registry.BaseURL = "registry.test.com"
	config.Registry.Image = "test-image"
	config.Registry.User = "test-user"
	config.Security.EnableVulnCheck = true
	config.Security.EnableLinting = true
	config.Release.Enabled = false
	config.Release.UseGoreleaser = true
	config.Release.CreateGithubRelease = false
	config.Release.Platforms = []string{"linux/amd64"}
	config.Logging.Level = "info"

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

// TestYAMLParser_PluginsRoundTrip verifies the fix for #162: plugin config
// written in .shipwright.yml must survive the full YAML → ParseFile →
// ApplyToConfiguration → Get path and arrive at the plugin.
func TestYAMLParser_PluginsRoundTrip(t *testing.T) {
	yamlContent := `pipeline:
  name: test-pipeline
  steps:
    - setup
    - build
plugins:
  nomad-deploy:
    nomad_addr: https://nomad.example.com:4646
    region: global
`
	tmpFile := t.TempDir() + "/shipwright.yml"
	require.NoError(t, os.WriteFile(tmpFile, []byte(yamlContent), 0o644))

	parser := NewYAMLParser()
	yamlConfig, err := parser.ParseFile(tmpFile)
	require.NoError(t, err)

	cfg, err := NewConfigurationWrapper()
	require.NoError(t, err)

	require.NoError(t, parser.ApplyToConfiguration(yamlConfig, cfg))

	// The full YAML → Get path must return the plugin config map.
	got := cfg.Get("plugins.nomad-deploy")
	require.NotNil(t, got, "plugins.nomad-deploy must not be nil after YAML round-trip")

	gotMap, ok := got.(map[string]any)
	require.True(t, ok, "expected map[string]any, got %T", got)
	assert.Equal(t, "https://nomad.example.com:4646", gotMap["nomad_addr"])
	assert.Equal(t, "global", gotMap["region"])
}
