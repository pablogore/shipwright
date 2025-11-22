package config

// Configuration key constants for consistent access.
const (
	// Pipeline configuration.
	KeyPipelineName        = "pipeline.name"
	KeyPipelineCoverage    = "pipeline.coverage"
	KeyPipelineSkipPush    = "pipeline.skip_push"
	KeyPipelineOnlyBuild   = "pipeline.only_build"
	KeyPipelineOnlyTest    = "pipeline.only_test"
	KeyPipelineVerbose     = "pipeline.verbose"
	KeyPipelineGoVersion   = "pipeline.go_version"
	KeyPipelineJavaVersion = "pipeline.java_version"

	// Registry configuration.
	KeyRegistryBaseURL = "registry.base_url"
	KeyRegistryUser    = "registry.user"
	KeyRegistryPass    = "registry.pass"
	KeyRegistryImage   = "registry.image"
	KeyRegistryTag     = "registry.tag"

	// Git configuration.
	KeyGitRepo      = "git.repo"
	KeyGitRef       = "git.ref"
	KeyGitProtocol  = "git.protocol"
	KeyGitUserEmail = "git.user_email"
	KeyGitUserName  = "git.user_name"
	KeyGitSSHKey    = "git.ssh_key"

	// Security configuration.
	KeySecurityEnableVulnCheck = "security.enable_vuln_check"
	KeySecurityEnableLinting   = "security.enable_linting"
	KeySecurityLintTimeout     = "security.lint_timeout"
	KeySecurityExcludePatterns = "security.exclude_patterns"

	// Logging configuration.
	KeyLogLevel            = "logging.level"
	KeyLogFormat           = "logging.format"
	KeyLogSamplingEnable   = "logging.sampling_enable"
	KeyLogSamplingRate     = "logging.sampling_rate"
	KeyLogSamplingInterval = "logging.sampling_interval"

	// Release configuration.
	KeyReleaseEnabled        = "release.enabled"
	KeyReleaseUseGoreleaser  = "release.use_goreleaser"
	KeyReleaseBuildTargets   = "release.build_targets"
	KeyReleaseArchiveFormats = "release.archive_formats"
	KeyReleaseChecksum       = "release.checksum"
	KeyReleaseSign           = "release.sign"

	// Dagger configuration.
	KeyDaggerLogOutput = "dagger.log_output"
	KeyDaggerTimeout   = "dagger.timeout"
)
