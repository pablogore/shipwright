package goservice

import (
	"time"

	"github.com/pablogore/shipwright/internal/pipelines"
)

// BuildMode defines the build mode for the pipeline.
type BuildMode string

const (
	// BuildModeBinary builds only the binary without Docker.
	BuildModeBinary BuildMode = "binary"
	// BuildModeDocker builds only the Docker image.
	BuildModeDocker BuildMode = "docker"
	// BuildModeBoth builds both binary and Docker image.
	BuildModeBoth BuildMode = "both"
)

// DefaultBuildMode is the default build mode.
const DefaultBuildMode = BuildModeBinary

// Framework defines the framework used by the service.
type Framework string

const (
	// FrameworkStandard is the standard Go framework.
	FrameworkStandard Framework = "standard"
	// FrameworkGoKit is the go-kit framework.
	FrameworkGoKit Framework = "go-kit"
	// FrameworkGin is the Gin framework.
	FrameworkGin Framework = "gin"
	// FrameworkEcho is the Echo framework.
	FrameworkEcho Framework = "echo"
)

// DefaultFramework is the default framework.
const DefaultFramework = FrameworkStandard

// TagStrategy defines how version tags are generated.
type TagStrategy string

const (
	// TagStrategySemver uses semantic versioning.
	TagStrategySemver TagStrategy = "semver"
	// TagStrategyCommit uses commit SHA.
	TagStrategyCommit TagStrategy = "commit"
	// TagStrategyBranch uses branch name.
	TagStrategyBranch TagStrategy = "branch"
)

// DefaultTagStrategy is the default tag strategy.
const DefaultTagStrategy = TagStrategyCommit

// VulnCheckLevel defines the vulnerability check level.
type VulnCheckLevel string

const (
	// VulnCheckLevelModerate checks for moderate and higher severity vulnerabilities.
	VulnCheckLevelModerate VulnCheckLevel = "moderate"
	// VulnCheckLevelHigh checks for high and critical severity vulnerabilities.
	VulnCheckLevelHigh VulnCheckLevel = "high"
	// VulnCheckLevelCritical checks only for critical severity vulnerabilities.
	VulnCheckLevelCritical VulnCheckLevel = "critical"
)

// DefaultVulnCheckLevel is the default vulnerability check level.
const DefaultVulnCheckLevel = VulnCheckLevelModerate

// GoServiceOptions defines configurable options for the go-service pipeline.
// All fields are ordered alphabetically as per project standards.
type GoServiceOptions struct {
	// BinaryName is the name of the output binary file.
	BinaryName string
	// BuildMode defines how to build (binary, docker, both).
	BuildMode BuildMode
	// CoverageThreshold is the minimum test coverage percentage required.
	CoverageThreshold float64
	// DockerfilePath is the path to the Dockerfile (optional, defaults to ./Dockerfile).
	DockerfilePath string
	// Framework specifies the framework used (standard, go-kit, gin, echo).
	Framework Framework
	// FrameworkConfig contains framework-specific configuration.
	FrameworkConfig map[string]interface{}
	// ImageName is the name of the Docker image.
	ImageName string
	// LinterConfig is the path to the linter configuration file (e.g., .golangci.yml).
	LinterConfig string
	// LinterTimeout is the timeout for the linter execution.
	LinterTimeout time.Duration
	// RegistryURL is the container registry URL.
	RegistryURL string
	// TagStrategy defines how version tags are generated (semver, commit, branch).
	TagStrategy TagStrategy
	// TestTags are build tags to use when running tests.
	TestTags []string
	// TestTimeout is the timeout for test execution.
	TestTimeout time.Duration
	// VulnCheckEnabled indicates whether vulnerability checking is enabled.
	VulnCheckEnabled bool
	// VulnCheckLevel specifies the minimum severity level for vulnerability checks.
	VulnCheckLevel VulnCheckLevel
}

// DefaultGoServiceOptions returns GoServiceOptions with default values.
func DefaultGoServiceOptions() GoServiceOptions {
	return GoServiceOptions{
		BinaryName:        "app",
		BuildMode:         DefaultBuildMode,
		CoverageThreshold: 90.0,
		DockerfilePath:    "./Dockerfile",
		Framework:         DefaultFramework,
		FrameworkConfig:   make(map[string]interface{}),
		ImageName:         "",
		LinterConfig:      ".golangci.yml",
		LinterTimeout:     5 * time.Minute,
		RegistryURL:       "",
		TagStrategy:       DefaultTagStrategy,
		TestTags:          []string{},
		TestTimeout:       15 * time.Minute,
		VulnCheckEnabled:  true,
		VulnCheckLevel:    DefaultVulnCheckLevel,
	}
}

// parseGoServiceOptions parses GoServiceOptions from the pipeline configuration.
// It reads configuration values using the Options pattern with sensible defaults.
func parseGoServiceOptions(cfg pipelines.Config) GoServiceOptions {
	opts := DefaultGoServiceOptions()

	// Parse build mode from config (if available in future config structure)
	// For now, use defaults and allow override via environment variables or config file

	// Override with values from config if available
	if cfg.Coverage > 0 {
		opts.CoverageThreshold = cfg.Coverage
	}

	if cfg.GoVersion != "" {
		// GoVersion is already in Config, no need to duplicate
	}

	// RegistryURL takes precedence over Registry
	if cfg.Registry != "" {
		opts.RegistryURL = cfg.Registry
	}
	if cfg.RegistryURL != "" {
		opts.RegistryURL = cfg.RegistryURL
	}

	if cfg.ImageName != "" {
		opts.ImageName = cfg.ImageName
	}

	return opts
}
