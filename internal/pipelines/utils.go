package pipelines

import (
	"fmt"
	"net/url"
	"strings"
)

// ExtractHostFromRepoURL extracts the host from a repository URL.
// Supports both HTTPS (https://github.com/user/repo.git) and SSH (git@github.com:user/repo.git) formats.
func ExtractHostFromRepoURL(repoURL string) (string, error) {
	if repoURL == "" {
		return "", fmt.Errorf("repository URL is empty")
	}

	// Handle SSH format: git@host:user/repo.git
	if strings.HasPrefix(repoURL, "git@") {
		parts := strings.Split(repoURL, ":")
		if len(parts) > 0 {
			host := strings.TrimPrefix(parts[0], "git@")
			return host, nil
		}
	}

	// Handle HTTPS format: https://host/user/repo.git
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse repository URL: %w", err)
	}

	host := parsedURL.Host
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	if host == "" {
		return "", fmt.Errorf("could not extract host from repository URL: %s", repoURL)
	}

	return host, nil
}

// GenerateNetrc generates a .netrc file content for Git authentication.
// The host is extracted from the repository URL.
func GenerateNetrc(repoURL, user, token string) (string, error) {
	host, err := ExtractHostFromRepoURL(repoURL)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("machine %s login %s password %s\n", host, user, token), nil
}

// GitlabNetrc generates a .netrc file content for GitLab authentication (deprecated, use GenerateNetrc instead)
// Deprecated: Use GenerateNetrc instead which extracts host from repo URL
func GitlabNetrc(user, token string) string {
	return fmt.Sprintf("machine gitlab.com login %s password %s\n", user, token)
}

type ReturnType = []int

func ExitOk() ReturnType {
	return []int{0}
}

func ExitOkOrTestFail() ReturnType {
	return []int{0, 1}
}
