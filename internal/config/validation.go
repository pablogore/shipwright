package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ValidateRegistryURL validates that the registry URL has a valid format.
// It accepts URLs with or without schemes. If no scheme is provided, https:// is assumed.
// This maintains backward compatibility with existing configurations.
func ValidateRegistryURL(registryURL string) error {
	if registryURL == "" {
		return errors.New("registry URL cannot be empty")
	}

	// Normalize URL by adding https:// scheme if missing (for backward compatibility)
	normalizedURL := registryURL
	if !strings.Contains(registryURL, "://") {
		normalizedURL = "https://" + registryURL
	}

	parsed, err := url.Parse(normalizedURL)
	if err != nil {
		return fmt.Errorf("invalid registry URL format: %w", err)
	}

	// Validate scheme (after normalization, should always have one)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("registry URL must use http:// or https:// scheme, got: %s", parsed.Scheme)
	}

	// Validate host
	if parsed.Host == "" {
		return errors.New("registry URL must include host")
	}

	return nil
}

// ValidateGitRepoURL validates that the Git repository URL has a valid format.
// It supports both HTTPS (https://host/user/repo.git) and SSH (git@host:user/repo.git) formats.
func ValidateGitRepoURL(repoURL string) error {
	if repoURL == "" {
		return errors.New("Git repository URL cannot be empty")
	}

	// Handle SSH format: git@host:user/repo.git
	if strings.HasPrefix(repoURL, "git@") {
		parts := strings.Split(repoURL, ":")
		if len(parts) < 2 {
			return errors.New("invalid SSH Git URL format: must be git@host:path")
		}
		if parts[1] == "" {
			return errors.New("invalid SSH Git URL format: path cannot be empty")
		}
		host := strings.TrimPrefix(parts[0], "git@")
		if host == "" {
			return errors.New("invalid SSH Git URL format: host cannot be empty")
		}
		return nil
	}

	// Handle HTTPS format: https://host/user/repo.git
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("invalid Git repository URL: %w", err)
	}

	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("Git repository URL must use https:// or http:// scheme, got: %s", parsed.Scheme)
	}

	if parsed.Host == "" {
		return errors.New("Git repository URL must include host")
	}

	return nil
}

// ValidateGoVersion validates that the Go version follows semantic versioning format.
// It accepts X.Y or X.Y.Z formats (e.g., "1.25" or "1.25.5").
func ValidateGoVersion(version string) error {
	if version == "" {
		return errors.New("Go version cannot be empty")
	}

	// Validate format: X.Y or X.Y.Z where X, Y, Z are digits
	matched, err := regexp.MatchString(`^\d+\.\d+(\.\d+)?$`, version)
	if err != nil {
		return fmt.Errorf("failed to validate Go version format: %w", err)
	}

	if !matched {
		return fmt.Errorf("invalid Go version format: %s (expected X.Y or X.Y.Z, e.g., 1.25 or 1.25.5)", version)
	}

	return nil
}

// ValidateEnvironment validates that the environment is one of the allowed values.
// Valid values: "dev", "development", "staging", "production", "prod".
func ValidateEnvironment(env string) error {
	if env == "" {
		return errors.New("environment cannot be empty")
	}

	validEnvironments := []string{"dev", "development", "staging", "production", "prod"}

	for _, valid := range validEnvironments {
		if env == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid environment: %s (must be one of: %v)", env, validEnvironments)
}
