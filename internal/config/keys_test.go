package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigurationKeys(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		category string
	}{
		// Pipeline configuration
		{name: "pipeline name", key: KeyPipelineName, category: "pipeline"},
		{name: "pipeline coverage", key: KeyPipelineCoverage, category: "pipeline"},
		{name: "pipeline skip push", key: KeyPipelineSkipPush, category: "pipeline"},
		{name: "pipeline only build", key: KeyPipelineOnlyBuild, category: "pipeline"},
		{name: "pipeline only test", key: KeyPipelineOnlyTest, category: "pipeline"},
		{name: "pipeline verbose", key: KeyPipelineVerbose, category: "pipeline"},
		{name: "pipeline go version", key: KeyPipelineGoVersion, category: "pipeline"},
		{name: "pipeline java version", key: KeyPipelineJavaVersion, category: "pipeline"},

		// Registry configuration
		{name: "registry base URL", key: KeyRegistryBaseURL, category: "registry"},
		{name: "registry user", key: KeyRegistryUser, category: "registry"},
		{name: "registry pass", key: KeyRegistryPass, category: "registry"},
		{name: "registry image", key: KeyRegistryImage, category: "registry"},
		{name: "registry tag", key: KeyRegistryTag, category: "registry"},

		// Git configuration
		{name: "git repo", key: KeyGitRepo, category: "git"},
		{name: "git ref", key: KeyGitRef, category: "git"},
		{name: "git protocol", key: KeyGitProtocol, category: "git"},
		{name: "git user email", key: KeyGitUserEmail, category: "git"},
		{name: "git user name", key: KeyGitUserName, category: "git"},
		{name: "git ssh key", key: KeyGitSSHKey, category: "git"},

		// Security configuration
		{name: "security enable vuln check", key: KeySecurityEnableVulnCheck, category: "security"},
		{name: "security enable linting", key: KeySecurityEnableLinting, category: "security"},
		{name: "security lint timeout", key: KeySecurityLintTimeout, category: "security"},
		{name: "security exclude patterns", key: KeySecurityExcludePatterns, category: "security"},

		// Logging configuration
		{name: "log level", key: KeyLogLevel, category: "logging"},
		{name: "log format", key: KeyLogFormat, category: "logging"},
		{name: "log sampling enable", key: KeyLogSamplingEnable, category: "logging"},
		{name: "log sampling rate", key: KeyLogSamplingRate, category: "logging"},
		{name: "log sampling interval", key: KeyLogSamplingInterval, category: "logging"},

		// Release configuration
		{name: "release enabled", key: KeyReleaseEnabled, category: "release"},
		{name: "release use goreleaser", key: KeyReleaseUseGoreleaser, category: "release"},
		{name: "release build targets", key: KeyReleaseBuildTargets, category: "release"},
		{name: "release archive formats", key: KeyReleaseArchiveFormats, category: "release"},
		{name: "release checksum", key: KeyReleaseChecksum, category: "release"},
		{name: "release sign", key: KeyReleaseSign, category: "release"},

		// Dagger configuration
		{name: "dagger log output", key: KeyDaggerLogOutput, category: "dagger"},
		{name: "dagger timeout", key: KeyDaggerTimeout, category: "dagger"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.key, "Key should not be empty")
			assert.Contains(t, tt.key, tt.category, "Key should contain category")
		})
	}
}

func TestKeyFormat(t *testing.T) {
	// Test that all keys follow the expected format: category.field
	allKeys := []string{
		KeyPipelineName, KeyPipelineCoverage, KeyPipelineSkipPush, KeyPipelineOnlyBuild,
		KeyPipelineOnlyTest, KeyPipelineVerbose, KeyPipelineGoVersion, KeyPipelineJavaVersion,
		KeyRegistryBaseURL, KeyRegistryUser, KeyRegistryPass, KeyRegistryImage, KeyRegistryTag,
		KeyGitRepo, KeyGitRef, KeyGitProtocol, KeyGitUserEmail, KeyGitUserName, KeyGitSSHKey,
		KeySecurityEnableVulnCheck, KeySecurityEnableLinting, KeySecurityLintTimeout, KeySecurityExcludePatterns,
		KeyLogLevel, KeyLogFormat, KeyLogSamplingEnable, KeyLogSamplingRate, KeyLogSamplingInterval,
		KeyReleaseEnabled, KeyReleaseUseGoreleaser, KeyReleaseBuildTargets, KeyReleaseArchiveFormats,
		KeyReleaseChecksum, KeyReleaseSign,
		KeyDaggerLogOutput, KeyDaggerTimeout,
	}

	for _, key := range allKeys {
		t.Run(key, func(t *testing.T) {
			// Each key should contain a dot (category.field format)
			assert.Contains(t, key, ".", "Key should be in format 'category.field'")

			// Key should not be empty
			assert.NotEmpty(t, key, "Key should not be empty")

			// Key should not start or end with a dot
			assert.NotEqual(t, ".", key[:1], "Key should not start with dot")
			assert.NotEqual(t, ".", key[len(key)-1:], "Key should not end with dot")
		})
	}
}

func TestKeyUniqueness(t *testing.T) {
	// Test that all keys are unique
	allKeys := []string{
		KeyPipelineName, KeyPipelineCoverage, KeyPipelineSkipPush, KeyPipelineOnlyBuild,
		KeyPipelineOnlyTest, KeyPipelineVerbose, KeyPipelineGoVersion, KeyPipelineJavaVersion,
		KeyRegistryBaseURL, KeyRegistryUser, KeyRegistryPass, KeyRegistryImage, KeyRegistryTag,
		KeyGitRepo, KeyGitRef, KeyGitProtocol, KeyGitUserEmail, KeyGitUserName, KeyGitSSHKey,
		KeySecurityEnableVulnCheck, KeySecurityEnableLinting, KeySecurityLintTimeout, KeySecurityExcludePatterns,
		KeyLogLevel, KeyLogFormat, KeyLogSamplingEnable, KeyLogSamplingRate, KeyLogSamplingInterval,
		KeyReleaseEnabled, KeyReleaseUseGoreleaser, KeyReleaseBuildTargets, KeyReleaseArchiveFormats,
		KeyReleaseChecksum, KeyReleaseSign,
		KeyDaggerLogOutput, KeyDaggerTimeout,
	}

	keyMap := make(map[string]bool)
	for _, key := range allKeys {
		assert.False(t, keyMap[key], "Key %s should be unique", key)
		keyMap[key] = true
	}
}

func TestKeyCategories(t *testing.T) {
	// Test that keys are properly categorized
	categories := map[string][]string{
		"pipeline": {
			KeyPipelineName, KeyPipelineCoverage, KeyPipelineSkipPush, KeyPipelineOnlyBuild,
			KeyPipelineOnlyTest, KeyPipelineVerbose, KeyPipelineGoVersion, KeyPipelineJavaVersion,
		},
		"registry": {KeyRegistryBaseURL, KeyRegistryUser, KeyRegistryPass, KeyRegistryImage, KeyRegistryTag},
		"git":      {KeyGitRepo, KeyGitRef, KeyGitProtocol, KeyGitUserEmail, KeyGitUserName, KeyGitSSHKey},
		"security": {KeySecurityEnableVulnCheck, KeySecurityEnableLinting, KeySecurityLintTimeout, KeySecurityExcludePatterns},
		"logging":  {KeyLogLevel, KeyLogFormat, KeyLogSamplingEnable, KeyLogSamplingRate, KeyLogSamplingInterval},
		"release":  {KeyReleaseEnabled, KeyReleaseUseGoreleaser, KeyReleaseBuildTargets, KeyReleaseArchiveFormats, KeyReleaseChecksum, KeyReleaseSign},
		"dagger":   {KeyDaggerLogOutput, KeyDaggerTimeout},
	}

	for category, keys := range categories {
		for _, key := range keys {
			t.Run(key, func(t *testing.T) {
				assert.Contains(t, key, category+".", "Key %s should belong to category %s", key, category)
			})
		}
	}
}
