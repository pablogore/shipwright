package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/knadh/koanf/providers/env"
	koanfv2 "github.com/knadh/koanf/v2"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
)

// Static errors for err113 compliance.
var (
	ErrPipelineNameRequired    = errors.New("pipeline name is required")
	ErrRegistryBaseURLRequired = errors.New("registry base URL is required")
	ErrGoVersionRequired       = errors.New("go version is required")
	ErrInvalidCoverage         = errors.New("coverage must be between 0 and 100")
)

// Configuration constants.
const (
	// Default values.
	DefaultEnvironment     = "development"
	DefaultPipelineName    = "go-kit"
	DefaultCoverage        = 90.0
	DefaultGoVersion       = "1.25.1"
	DefaultJavaVersion     = "17"
	DefaultRegistryBaseURL = "registry.gitlab.com/syntegrity"
	DefaultRegistryUser    = "gitlab-ci-token"
	DefaultGitProtocol     = "ssh"
	DefaultGitRef          = "main"
	DefaultLogLevel        = "info"
	DefaultLogFormat       = "json"
	DefaultLintTimeout     = "5m"
	DefaultDaggerTimeout   = 30 * time.Second

	// Environment prefix.
	EnvPrefix = "SYNTEGRITY_DAGGER_"
)

// koanfInstance is created fresh for each configuration load to avoid test interference

// Config provides a simplified configuration interface with sane defaults.
type Config struct {
	Environment string         `koanf:"environment"`
	Pipeline    PipelineConfig `koanf:"pipeline"`
	Registry    RegistryConfig `koanf:"registry"`
	Git         GitConfig      `koanf:"git"`
	Security    SecurityConfig `koanf:"security"`
	Logging     LoggingConfig  `koanf:"logging"`
	Release     ReleaseConfig  `koanf:"release"`
	Dagger      DaggerConfig   `koanf:"dagger"`
}

// PipelineConfig defines pipeline configuration.
type PipelineConfig struct {
	Name        string  `koanf:"name"`
	Environment string  `koanf:"environment"`
	Coverage    float64 `koanf:"coverage"`
	SkipPush    bool    `koanf:"skip_push"`
	OnlyBuild   bool    `koanf:"only_build"`
	OnlyTest    bool    `koanf:"only_test"`
	Verbose     bool    `koanf:"verbose"`
	GoVersion   string  `koanf:"go_version"`
	JavaVersion string  `koanf:"java_version"`
}

// RegistryConfig defines container registry configuration.
type RegistryConfig struct {
	BaseURL string `koanf:"base_url"`
	User    string `koanf:"user"`
	Pass    string `koanf:"pass"`
	Image   string `koanf:"image"`
	Tag     string `koanf:"tag"`
}

// GitConfig defines Git configuration.
type GitConfig struct {
	Repo      string `koanf:"repo"`
	Ref       string `koanf:"ref"`
	Protocol  string `koanf:"protocol"`
	UserEmail string `koanf:"user_email"`
	UserName  string `koanf:"user_name"`
	SSHKey    string `koanf:"ssh_key"`
}

// SecurityConfig defines security configuration.
type SecurityConfig struct {
	EnableVulnCheck bool     `koanf:"enable_vuln_check"`
	EnableLinting   bool     `koanf:"enable_linting"`
	LintTimeout     string   `koanf:"lint_timeout"`
	ExcludePatterns []string `koanf:"exclude_patterns"`
}

// LoggingConfig defines logging configuration.
type LoggingConfig struct {
	Level            string        `koanf:"level"`
	Format           string        `koanf:"format"`
	SamplingEnable   bool          `koanf:"sampling_enable"`
	SamplingRate     float64       `koanf:"sampling_rate"`
	SamplingInterval time.Duration `koanf:"sampling_interval"`
}

// ReleaseConfig defines release configuration.
type ReleaseConfig struct {
	Enabled        bool     `koanf:"enabled"`
	UseGoreleaser  bool     `koanf:"use_goreleaser"`
	BuildTargets   []string `koanf:"build_targets"`
	ArchiveFormats []string `koanf:"archive_formats"`
	Checksum       bool     `koanf:"checksum"`
	Sign           bool     `koanf:"sign"`
}

// DaggerConfig defines Dagger configuration.
type DaggerConfig struct {
	LogOutput bool          `koanf:"log_output"`
	Timeout   time.Duration `koanf:"timeout"`
}

// defaultConfig returns a new Config with sane defaults.
func defaultConfig() *Config {
	return &Config{
		Environment: DefaultEnvironment,
		Pipeline: PipelineConfig{
			Name:        DefaultPipelineName,
			Environment: DefaultEnvironment,
			Coverage:    DefaultCoverage,
			SkipPush:    false,
			OnlyBuild:   false,
			OnlyTest:    false,
			Verbose:     false,
			GoVersion:   DefaultGoVersion,
			JavaVersion: DefaultJavaVersion,
		},
		Registry: RegistryConfig{
			BaseURL: DefaultRegistryBaseURL,
			User:    DefaultRegistryUser,
			Pass:    "",
			Image:   "",
			Tag:     "",
		},
		Git: GitConfig{
			Repo:      "",
			Ref:       DefaultGitRef,
			Protocol:  DefaultGitProtocol,
			UserEmail: "",
			UserName:  "",
			SSHKey:    "",
		},
		Security: SecurityConfig{
			EnableVulnCheck: true,
			EnableLinting:   true,
			LintTimeout:     DefaultLintTimeout,
			ExcludePatterns: []string{},
		},
		Logging: LoggingConfig{
			Level:            DefaultLogLevel,
			Format:           DefaultLogFormat,
			SamplingEnable:   false,
			SamplingRate:     0.1,
			SamplingInterval: 1 * time.Second,
		},
		Release: ReleaseConfig{
			Enabled:        false,
			UseGoreleaser:  true,
			BuildTargets:   []string{"linux/amd64", "darwin/amd64", "windows/amd64"},
			ArchiveFormats: []string{"tar.gz", "zip"},
			Checksum:       true,
			Sign:           false,
		},
		Dagger: DaggerConfig{
			LogOutput: true,
			Timeout:   DefaultDaggerTimeout,
		},
	}
}

// loadDotEnv loads .env file if it exists.
func loadDotEnv() error {
	envFile := ".env"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return nil
	}
	if err := godotenv.Load(envFile); err != nil {
		return fmt.Errorf("failed to load env file %s: %w", envFile, err)
	}
	return nil
}

// New creates a new configuration with the following precedence:
// 1. Environment variables (SYNTEGRITY_DAGGER_*)
// 2. Default values.
func New() (*Config, error) {
	if err := loadDotEnv(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create a new koanf instance for each configuration load to avoid test interference
	koanfInstance := koanfv2.New(".")

	err := koanfInstance.Load(env.Provider(EnvPrefix, "_", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, EnvPrefix))
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	cfg := defaultConfig()
	err = koanfInstance.Unmarshal("", cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Pipeline.Name == "" {
		return ErrPipelineNameRequired
	}
	if c.Registry.BaseURL == "" {
		return ErrRegistryBaseURLRequired
	}
	if c.Pipeline.GoVersion == "" {
		return ErrGoVersionRequired
	}
	if c.Pipeline.Coverage < 0 || c.Pipeline.Coverage > 100 {
		return ErrInvalidCoverage
	}
	return nil
}

// ConfigurationWrapper wraps Config to implement interfaces.Configuration.
type ConfigurationWrapper struct {
	*Config
}

// NewConfigurationWrapper creates a new configuration wrapper.
func NewConfigurationWrapper() (*ConfigurationWrapper, error) {
	cfg, err := New()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return &ConfigurationWrapper{Config: cfg}, nil
}

// Load loads the configuration.
func (cw *ConfigurationWrapper) Load() error {
	return nil // Already loaded in constructor
}

// LoadWithDefaults loads configuration with defaults.
func (cw *ConfigurationWrapper) LoadWithDefaults(_ map[string]any) error {
	return nil // Not implemented for this wrapper
}

// GetConfigSummary returns a summary of the configuration.
func (cw *ConfigurationWrapper) GetConfigSummary() string {
	return fmt.Sprintf("Pipeline: %s, Environment: %s, Go Version: %s, Registry: %s",
		cw.Config.Pipeline.Name, cw.Config.Environment, cw.Config.Pipeline.GoVersion, cw.Config.Registry.BaseURL)
}

// stringConfigMap holds the string configuration values.
func (cw *ConfigurationWrapper) getStringConfigMap() map[string]string {
	return map[string]string{
		KeyServiceName:         "syntegrity-dagger",
		KeyServiceVersion:      "1.0.0",
		KeyEnvironment:         cw.Config.Environment,
		KeyPipelineName:        cw.Config.Pipeline.Name,
		KeyPipelineGoVersion:   cw.Config.Pipeline.GoVersion,
		KeyPipelineJavaVersion: cw.Config.Pipeline.JavaVersion,
		KeyRegistryBaseURL:     cw.Config.Registry.BaseURL,
		KeyRegistryUser:        cw.Config.Registry.User,
		KeyRegistryPass:        cw.Config.Registry.Pass,
		KeyRegistryImage:       cw.Config.Registry.Image,
		KeyRegistryTag:         cw.Config.Registry.Tag,
		KeyGitRepo:             cw.Config.Git.Repo,
		KeyGitRef:              cw.Config.Git.Ref,
		KeyGitProtocol:         cw.Config.Git.Protocol,
		KeyGitUserEmail:        cw.Config.Git.UserEmail,
		KeyGitUserName:         cw.Config.Git.UserName,
		KeyGitSSHKey:           cw.Config.Git.SSHKey,
		KeySecurityLintTimeout: cw.Config.Security.LintTimeout,
		KeyLogLevel:            cw.Config.Logging.Level,
		KeyLogFormat:           cw.Config.Logging.Format,
		KeyDaggerLogOutput:     strconv.FormatBool(cw.Config.Dagger.LogOutput),
	}
}

// GetString gets a string value using map lookup for better performance.
func (cw *ConfigurationWrapper) GetString(key string) string {
	stringMap := cw.getStringConfigMap()
	if value, exists := stringMap[key]; exists {
		return value
	}
	return ""
}

// getIntConfigMap holds the int configuration values.
func (cw *ConfigurationWrapper) getIntConfigMap() map[string]int {
	return map[string]int{}
}

// GetInt gets an int value using map lookup for better performance.
func (cw *ConfigurationWrapper) GetInt(key string) int {
	intMap := cw.getIntConfigMap()
	if value, exists := intMap[key]; exists {
		return value
	}
	return 0
}

// getBoolConfigMap holds the bool configuration values.
func (cw *ConfigurationWrapper) getBoolConfigMap() map[string]bool {
	return map[string]bool{
		KeyPipelineSkipPush:        cw.Config.Pipeline.SkipPush,
		KeyPipelineOnlyBuild:       cw.Config.Pipeline.OnlyBuild,
		KeyPipelineOnlyTest:        cw.Config.Pipeline.OnlyTest,
		KeyPipelineVerbose:         cw.Config.Pipeline.Verbose,
		KeySecurityEnableVulnCheck: cw.Config.Security.EnableVulnCheck,
		KeySecurityEnableLinting:   cw.Config.Security.EnableLinting,
		KeyLogSamplingEnable:       cw.Config.Logging.SamplingEnable,
		KeyReleaseEnabled:          cw.Config.Release.Enabled,
		KeyReleaseUseGoreleaser:    cw.Config.Release.UseGoreleaser,
		KeyReleaseChecksum:         cw.Config.Release.Checksum,
		KeyReleaseSign:             cw.Config.Release.Sign,
		KeyDaggerLogOutput:         cw.Config.Dagger.LogOutput,
	}
}

// GetBool gets a bool value using map lookup for better performance.
func (cw *ConfigurationWrapper) GetBool(key string) bool {
	boolMap := cw.getBoolConfigMap()
	if value, exists := boolMap[key]; exists {
		return value
	}
	return false
}

// getDurationConfigMap holds the duration configuration values.
func (cw *ConfigurationWrapper) getDurationConfigMap() map[string]time.Duration {
	return map[string]time.Duration{
		KeyLogSamplingInterval: cw.Config.Logging.SamplingInterval,
		KeyDaggerTimeout:       cw.Config.Dagger.Timeout,
	}
}

// GetDuration gets a duration value using map lookup for better performance.
func (cw *ConfigurationWrapper) GetDuration(key string) time.Duration {
	durationMap := cw.getDurationConfigMap()
	if value, exists := durationMap[key]; exists {
		return value
	}
	return 0
}

// getFloatConfigMap holds the float configuration values.
func (cw *ConfigurationWrapper) getFloatConfigMap() map[string]float64 {
	return map[string]float64{
		KeyPipelineCoverage: cw.Config.Pipeline.Coverage,
		KeyLogSamplingRate:  cw.Config.Logging.SamplingRate,
	}
}

// GetFloat gets a float value using map lookup for better performance.
func (cw *ConfigurationWrapper) GetFloat(key string) float64 {
	floatMap := cw.getFloatConfigMap()
	if value, exists := floatMap[key]; exists {
		return value
	}
	return 0
}

// Get gets an interface value.
func (cw *ConfigurationWrapper) Get(key string) any {
	switch key {
	case KeySecurityExcludePatterns:
		return cw.Config.Security.ExcludePatterns
	case KeyReleaseBuildTargets:
		return cw.Config.Release.BuildTargets
	case KeyReleaseArchiveFormats:
		return cw.Config.Release.ArchiveFormats
	default:
		return nil
	}
}

// Set sets a value.
func (cw *ConfigurationWrapper) Set(_ string, _ any) {
	// Not implemented for this wrapper - configuration is read-only
}

// Has checks if a key exists.
func (cw *ConfigurationWrapper) Has(key string) bool {
	// Check if the key has a non-empty value
	switch key {
	case KeyServiceName, KeyServiceVersion, KeyEnvironment,
		KeyPipelineName, KeyPipelineGoVersion, KeyPipelineJavaVersion,
		KeyRegistryBaseURL, KeyRegistryUser, KeyGitRef, KeyGitProtocol,
		KeySecurityLintTimeout, KeyLogLevel, KeyLogFormat:
		return true
	default:
		return false
	}
}

// All returns all configuration as map.
func (cw *ConfigurationWrapper) All() map[string]any {
	return map[string]any{
		KeyServiceName:       "syntegrity-dagger",
		KeyServiceVersion:    "1.0.0",
		KeyEnvironment:       cw.Config.Environment,
		KeyPipelineName:      cw.Config.Pipeline.Name,
		KeyPipelineGoVersion: cw.Config.Pipeline.GoVersion,
		KeyRegistryBaseURL:   cw.Config.Registry.BaseURL,
		KeyGitRef:            cw.Config.Git.Ref,
		KeyGitProtocol:       cw.Config.Git.Protocol,
		KeyLogLevel:          cw.Config.Logging.Level,
		KeyLogFormat:         cw.Config.Logging.Format,
	}
}

// Pipeline returns pipeline configuration.
func (cw *ConfigurationWrapper) Pipeline() interfaces.PipelineConfig {
	return interfaces.PipelineConfig{
		Name:        cw.Config.Pipeline.Name,
		Environment: cw.Config.Pipeline.Environment,
		Coverage:    cw.Config.Pipeline.Coverage,
		SkipPush:    cw.Config.Pipeline.SkipPush,
		OnlyBuild:   cw.Config.Pipeline.OnlyBuild,
		OnlyTest:    cw.Config.Pipeline.OnlyTest,
		Verbose:     cw.Config.Pipeline.Verbose,
		GoVersion:   cw.Config.Pipeline.GoVersion,
		JavaVersion: cw.Config.Pipeline.JavaVersion,
	}
}

// Registry returns registry configuration.
func (cw *ConfigurationWrapper) Registry() interfaces.RegistryConfig {
	return interfaces.RegistryConfig{
		BaseURL: cw.Config.Registry.BaseURL,
		User:    cw.Config.Registry.User,
		Pass:    cw.Config.Registry.Pass,
		Image:   cw.Config.Registry.Image,
		Tag:     cw.Config.Registry.Tag,
	}
}

// Security returns security configuration.
func (cw *ConfigurationWrapper) Security() interfaces.SecurityConfig {
	return interfaces.SecurityConfig{
		EnableVulnCheck: cw.Config.Security.EnableVulnCheck,
		EnableLinting:   cw.Config.Security.EnableLinting,
		LintTimeout:     cw.Config.Security.LintTimeout,
		ExcludePatterns: cw.Config.Security.ExcludePatterns,
	}
}

// Logging returns logging configuration.
func (cw *ConfigurationWrapper) Logging() interfaces.LoggingConfig {
	return interfaces.LoggingConfig{
		Level:            cw.Config.Logging.Level,
		Format:           cw.Config.Logging.Format,
		SamplingEnable:   cw.Config.Logging.SamplingEnable,
		SamplingRate:     cw.Config.Logging.SamplingRate,
		SamplingInterval: cw.Config.Logging.SamplingInterval,
	}
}

// Environment returns the environment.
func (cw *ConfigurationWrapper) Environment() string {
	return cw.Config.Environment
}
