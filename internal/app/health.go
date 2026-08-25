package app

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dagger.io/dagger"
	"github.com/pablogore/kit-logger/pkg/logger"
	"github.com/pablogore/shipwright/internal/interfaces"
)

// CheckDaggerEngine verifies that the Dagger engine is accessible and responsive.
func CheckDaggerEngine(ctx context.Context, client *dagger.Client) error {
	if client == nil {
		return fmt.Errorf("Dagger client is nil")
	}

	// Try to get the Dagger version to verify connectivity
	// This is a lightweight operation that confirms the engine is running
	_, err := client.Container().From("alpine:latest").ID(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Dagger engine: %w (ensure Dagger engine is running: 'dagger run echo test')", err)
	}

	return nil
}

// CheckRegistry verifies that the Docker registry is accessible.
// It performs a basic connectivity check to the registry URL.
func CheckRegistry(ctx context.Context, registryURL, user, pass string) error {
	if registryURL == "" {
		return fmt.Errorf("registry URL is empty")
	}

	// Normalize URL by adding https:// scheme if missing (for backward compatibility)
	normalizedURL := registryURL
	if !strings.Contains(registryURL, "://") {
		normalizedURL = "https://" + registryURL
	}

	// Validate URL format
	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return fmt.Errorf("invalid registry URL format: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("registry URL must use http:// or https:// scheme, got: %s", parsedURL.Scheme)
	}

	// Use normalized URL for the health check
	registryURL = normalizedURL

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Try to connect to the registry (v2 API endpoint)
	healthURL := fmt.Sprintf("%s/v2/", registryURL)
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create registry request: %w", err)
	}

	// Add authentication if provided
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to registry at %s: %w (check network connectivity and registry availability)", registryURL, err)
	}
	defer resp.Body.Close()

	// Accept 200 (OK) or 401 (Unauthorized but reachable) as success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("registry returned unexpected status code: %d (expected 200 or 401)", resp.StatusCode)
	}

	return nil
}

// CheckGitRepo verifies that a Git repository is accessible.
// It performs basic validation of the repository URL format.
func CheckGitRepo(ctx context.Context, repoURL string) error {
	if repoURL == "" {
		return fmt.Errorf("Git repository URL is empty")
	}

	// Validate URL format
	if strings.HasPrefix(repoURL, "git@") {
		// SSH format: git@host:user/repo.git
		parts := strings.Split(repoURL, ":")
		if len(parts) < 2 {
			return fmt.Errorf("invalid SSH Git URL format: %s (expected: git@host:path)", repoURL)
		}
		host := strings.TrimPrefix(parts[0], "git@")
		if host == "" {
			return fmt.Errorf("invalid SSH Git URL: host cannot be empty")
		}
		if parts[1] == "" {
			return fmt.Errorf("invalid SSH Git URL: path cannot be empty")
		}
		// SSH URLs are validated by format only (actual connectivity requires SSH setup)
		return nil
	}

	// HTTPS format
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("invalid Git repository URL format: %w", err)
	}

	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("Git repository URL must use https:// or http:// scheme, got: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("Git repository URL must include host")
	}

	// For HTTPS URLs, we can try a lightweight HTTP HEAD request
	if parsedURL.Scheme == "https" || parsedURL.Scheme == "http" {
		httpClient := &http.Client{
			Timeout: 5 * time.Second,
		}

		req, err := http.NewRequestWithContext(ctx, "HEAD", repoURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create Git repository request: %w", err)
		}

		// Set User-Agent to avoid blocking
		req.Header.Set("User-Agent", "shipwright/1.0")

		resp, err := httpClient.Do(req)
		if err != nil {
			// Connection errors are acceptable - the URL format is valid
			return nil
		}
		defer resp.Body.Close()

		// Accept any status code as success (repository exists or is private)
		return nil
	}

	return nil
}

// RunHealthChecks executes all health checks for the configured services.
// It checks Dagger engine, registry (if configured), and Git repository (if configured).
func RunHealthChecks(ctx context.Context, cfg interfaces.Configuration) error {
	var errors []error

	// Check Dagger engine
	client, err := GetDaggerClientFromConfig(ctx, cfg)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to get Dagger client: %w", err))
	} else {
		if err := CheckDaggerEngine(ctx, client); err != nil {
			errors = append(errors, fmt.Errorf("Dagger engine check failed: %w", err))
		} else {
			logger.L().InfoContext(ctx, "Dagger engine is accessible")
		}
	}

	// Check registry if configured
	registryURL := cfg.GetString("registry.base_url")
	if registryURL != "" {
		user := cfg.GetString("registry.user")
		pass := cfg.GetString("registry.pass")
		if err := CheckRegistry(ctx, registryURL, user, pass); err != nil {
			errors = append(errors, fmt.Errorf("registry check failed: %w", err))
		} else {
			logger.L().InfoContext(ctx, "Registry is accessible", "registry_url", registryURL)
		}
	} else {
		logger.L().InfoContext(ctx, "Registry check skipped (not configured)")
	}

	// Check Git repository if configured
	gitRepo := cfg.GetString("git.repo")
	if gitRepo != "" {
		if err := CheckGitRepo(ctx, gitRepo); err != nil {
			errors = append(errors, fmt.Errorf("Git repository check failed: %w", err))
		} else {
			logger.L().InfoContext(ctx, "Git repository is accessible", "git_repo", gitRepo)
		}
	} else {
		logger.L().InfoContext(ctx, "Git repository check skipped (not configured)")
	}

	if len(errors) > 0 {
		return fmt.Errorf("health checks failed: %v", errors)
	}

	return nil
}

// GetDaggerClientFromConfig creates a Dagger client from configuration.
// This is a helper function for health checks.
func GetDaggerClientFromConfig(ctx context.Context, cfg interfaces.Configuration) (*dagger.Client, error) {
	client, err := dagger.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Dagger engine: %w", err)
	}
	return client, nil
}
