package goservice

import (
	"testing"
	"time"

	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestDefaultGoServiceOptions(t *testing.T) {
	opts := DefaultGoServiceOptions()

	assert.Equal(t, "app", opts.BinaryName)
	assert.Equal(t, DefaultBuildMode, opts.BuildMode)
	assert.Equal(t, 90.0, opts.CoverageThreshold)
	assert.Equal(t, "./Dockerfile", opts.DockerfilePath)
	assert.Equal(t, DefaultFramework, opts.Framework)
	assert.NotNil(t, opts.FrameworkConfig)
	assert.Equal(t, ".golangci.yml", opts.LinterConfig)
	assert.Equal(t, 5*time.Minute, opts.LinterTimeout)
	assert.Equal(t, DefaultTagStrategy, opts.TagStrategy)
	assert.NotNil(t, opts.TestTags)
	assert.Equal(t, 15*time.Minute, opts.TestTimeout)
	assert.True(t, opts.VulnCheckEnabled)
	assert.Equal(t, DefaultVulnCheckLevel, opts.VulnCheckLevel)
}

func TestParseGoServiceOptions_WithDefaults(t *testing.T) {
	cfg := pipelines.Config{}

	opts := parseGoServiceOptions(cfg)

	assert.Equal(t, DefaultGoServiceOptions().BinaryName, opts.BinaryName)
	assert.Equal(t, DefaultBuildMode, opts.BuildMode)
	assert.Equal(t, 90.0, opts.CoverageThreshold)
}

func TestParseGoServiceOptions_WithCoverage(t *testing.T) {
	cfg := pipelines.Config{
		Coverage: 95.0,
	}

	opts := parseGoServiceOptions(cfg)

	assert.Equal(t, 95.0, opts.CoverageThreshold)
}

func TestParseGoServiceOptions_WithRegistryURL(t *testing.T) {
	cfg := pipelines.Config{
		RegistryURL: "registry.example.com",
	}

	opts := parseGoServiceOptions(cfg)

	assert.Equal(t, "registry.example.com", opts.RegistryURL)
}

func TestParseGoServiceOptions_WithRegistry(t *testing.T) {
	cfg := pipelines.Config{
		Registry: "registry.example.com",
	}

	opts := parseGoServiceOptions(cfg)

	assert.Equal(t, "registry.example.com", opts.RegistryURL)
}

func TestParseGoServiceOptions_WithImageName(t *testing.T) {
	cfg := pipelines.Config{
		ImageName: "my-service",
	}

	opts := parseGoServiceOptions(cfg)

	assert.Equal(t, "my-service", opts.ImageName)
}

func TestParseGoServiceOptions_RegistryURLTakesPrecedence(t *testing.T) {
	cfg := pipelines.Config{
		RegistryURL: "registry.example.com",
		Registry:    "old-registry.example.com",
	}

	opts := parseGoServiceOptions(cfg)

	assert.Equal(t, "registry.example.com", opts.RegistryURL)
}
