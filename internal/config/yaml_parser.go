package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the structure of the .syntegrity-dagger.yml file
type YAMLConfig struct {
	Pipeline struct {
		Name        string   `yaml:"name"`
		Environment string   `yaml:"environment"`
		Coverage    float64  `yaml:"coverage"`
		GoVersion   string   `yaml:"goVersion"`
		Steps       []string `yaml:"steps"`
	} `yaml:"pipeline"`

	Registry struct {
		BaseURL string `yaml:"baseUrl"`
		Image   string `yaml:"image"`
		User    string `yaml:"user"`
	} `yaml:"registry"`

	Security struct {
		EnableVulnCheck bool `yaml:"enableVulnCheck"`
		EnableLinting   bool `yaml:"enableLinting"`
	} `yaml:"security"`

	Release struct {
		Enabled             bool     `yaml:"enabled"`
		UseGoreleaser       bool     `yaml:"useGoreleaser"`
		CreateGithubRelease bool     `yaml:"createGithubRelease"`
		Platforms           []string `yaml:"platforms"`
	} `yaml:"release"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`

	Git struct {
		Protocol string `yaml:"protocol"`
	} `yaml:"git"`
}

// YAMLParser handles parsing of YAML configuration files
type YAMLParser struct{}

// NewYAMLParser creates a new YAML parser
func NewYAMLParser() *YAMLParser {
	return &YAMLParser{}
}

// ParseFile parses a YAML configuration file
func (p *YAMLParser) ParseFile(filePath string) (*YAMLConfig, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", filePath)
	}

	// Read file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse YAML
	var config YAMLConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	return &config, nil
}

// ApplyToConfiguration applies YAML config to the main configuration
func (p *YAMLParser) ApplyToConfiguration(yamlConfig *YAMLConfig, config interfaces.Configuration) error {
	// Apply pipeline settings
	if yamlConfig.Pipeline.Name != "" {
		config.Set("pipeline.name", yamlConfig.Pipeline.Name)
	}
	if yamlConfig.Pipeline.Environment != "" {
		config.Set("pipeline.environment", yamlConfig.Pipeline.Environment)
	}
	if yamlConfig.Pipeline.Coverage > 0 {
		config.Set("pipeline.coverage", yamlConfig.Pipeline.Coverage)
	}
	if yamlConfig.Pipeline.GoVersion != "" {
		config.Set("pipeline.go_version", yamlConfig.Pipeline.GoVersion)
	}

	// Apply registry settings
	if yamlConfig.Registry.BaseURL != "" {
		config.Set("registry.base_url", yamlConfig.Registry.BaseURL)
	}
	if yamlConfig.Registry.Image != "" {
		config.Set("registry.image", yamlConfig.Registry.Image)
	}
	if yamlConfig.Registry.User != "" {
		config.Set("registry.user", yamlConfig.Registry.User)
	}

	// Apply security settings
	config.Set("security.enable_vuln_check", yamlConfig.Security.EnableVulnCheck)
	config.Set("security.enable_linting", yamlConfig.Security.EnableLinting)

	// Apply release settings
	config.Set("release.enabled", yamlConfig.Release.Enabled)
	config.Set("release.use_goreleaser", yamlConfig.Release.UseGoreleaser)
	config.Set("release.create_github_release", yamlConfig.Release.CreateGithubRelease)

	// Apply logging settings
	if yamlConfig.Logging.Level != "" {
		config.Set("logging.level", yamlConfig.Logging.Level)
	}

	// Apply git settings
	if yamlConfig.Git.Protocol != "" {
		config.Set("git.protocol", yamlConfig.Git.Protocol)
	}

	return nil
}

// GetSteps returns the list of steps from YAML config
func (p *YAMLParser) GetSteps(yamlConfig *YAMLConfig) []string {
	return yamlConfig.Pipeline.Steps
}

// ValidateConfig validates the YAML configuration
func (p *YAMLParser) ValidateConfig(yamlConfig *YAMLConfig) error {
	if yamlConfig.Pipeline.Name == "" {
		return errors.New("pipeline name is required")
	}

	if len(yamlConfig.Pipeline.Steps) == 0 {
		return errors.New("at least one step must be defined")
	}

	// Validate steps
	validSteps := map[string]bool{
		"setup":    true,
		"build":    true,
		"test":     true,
		"lint":     true,
		"security": true,
		"tag":      true,
		"package":  true,
		"push":     true,
		"release":  true,
	}

	for _, step := range yamlConfig.Pipeline.Steps {
		if !validSteps[step] {
			return fmt.Errorf("invalid step: %s", step)
		}
	}

	return nil
}

// FindConfigFile looks for configuration files in common locations
func (p *YAMLParser) FindConfigFile() (string, error) {
	// List of possible config file names and locations
	configFiles := []string{
		".syntegrity-dagger.yml",
		".syntegrity-dagger.yaml",
		"syntegrity-dagger.yml",
		"syntegrity-dagger.yaml",
		".github/syntegrity-dagger.yml",
		".github/syntegrity-dagger.yaml",
	}

	// Check current directory first
	for _, filename := range configFiles {
		if _, err := os.Stat(filename); err == nil {
			return filename, nil
		}
	}

	// Check parent directories (up to 3 levels)
	currentDir, _ := os.Getwd()
	for i := 0; i < 3; i++ {
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // Reached root
		}

		for _, filename := range configFiles {
			fullPath := filepath.Join(parentDir, filename)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath, nil
			}
		}
		currentDir = parentDir
	}

	return "", errors.New("no configuration file found")
}
