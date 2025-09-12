// Package shared contiene utilidades y helpers para operaciones comunes en los pipelines.
// shared/https_cloner.go
package shared

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dagger.io/dagger"
)

// CloneOptions configures the behavior of cloning
type CloneOptions struct {
	// Timeout is the maximum time allowed for cloning
	Timeout time.Duration
	// MaxRetries is the maximum number of retries
	MaxRetries int
	// RetryDelay is the time between retries
	RetryDelay time.Duration
	// ShallowClone indicates whether a shallow clone should be done
	ShallowClone bool
	// Depth is the depth of the clone if it's shallow
	Depth int
}

// DefaultCloneOptions returns the default options
func DefaultCloneOptions() CloneOptions {
	return CloneOptions{
		Timeout:      5 * time.Minute,
		MaxRetries:   3,
		RetryDelay:   time.Second * 2,
		ShallowClone: true,
		Depth:        1,
	}
}

type HTTPSCloner struct {
	opts CloneOptions
}

// NewHTTPSCloner creates a new HTTPS cloner with custom options
func NewHTTPSCloner(opts ...CloneOptions) *HTTPSCloner {
	cloner := &HTTPSCloner{
		opts: DefaultCloneOptions(),
	}
	if len(opts) > 0 {
		cloner.opts = opts[0]
	}
	return cloner
}

func (c *HTTPSCloner) Clone(ctx context.Context, client *dagger.Client, opts GitCloneOpts) (*dagger.Directory, error) {
	if opts.Branch == "" {
		opts.Branch = "main"
	}
	if opts.UserEmail == "" {
		opts.UserEmail = "ci@example.com"
	}
	if opts.UserName == "" {
		opts.UserName = "CI User"
	}
	if opts.Repo == "" {
		return nil, errors.New("invalid repository URL: repo is empty")
	}
	fmt.Printf("🔧 Cloning repo (HTTPS): %s (%s)\n", opts.Name, opts.Branch)

	// Get and validate credentials
	creds, err := ResolveGitCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve credentials: %w", err)
	}

	if err := creds.Validate(); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	// Configure the context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	// Prepare the cloning command
	cloneCmd := []string{"git", "clone"}
	if c.opts.ShallowClone {
		cloneCmd = append(cloneCmd, fmt.Sprintf("--depth=%d", c.opts.Depth))
	}
	cloneCmd = append(cloneCmd, "--branch", opts.Branch, opts.Repo, opts.Name)

	// Configure the base container
	container := client.Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "--no-cache", "git", "ca-certificates"})

	// Configure credentials if not anonymous
	if creds.Source != string(SourceAnonymous) {
		netrc := fmt.Sprintf("machine gitlab.com login %s password %s", creds.User, creds.Token)
		container = container.
			WithEnvVariable("HOME", "/root").
			WithNewFile("/root/.netrc", dagger.ContainerWithNewFileOpts{
				Contents:    netrc,
				Permissions: 0o600,
				Owner:       "root",
			}).
			WithExec([]string{"chmod", "600", "/root/.netrc"}).
			WithExec([]string{"git", "config", "--global", "credential.helper", "store"}).
			WithExec([]string{"git", "config", "--global", "http.sslVerify", "false"})
	}

	// Configure Git user
	email := opts.UserEmail
	if email == "" {
		email = "ci@getsyntegrity.com"
	}
	name := opts.UserName
	if name == "" {
		name = "Syntegrity CI"
	}
	container = container.
		WithExec([]string{"git", "config", "--global", "user.email", email}).
		WithExec([]string{"git", "config", "--global", "user.name", name})

	// Try cloning with retries
	var dir *dagger.Directory
	var lastErr error
	for i := 0; i < c.opts.MaxRetries; i++ {
		if i > 0 {
			fmt.Printf("Retrying clone (attempt %d/%d)...\n", i+1, c.opts.MaxRetries)
			time.Sleep(c.opts.RetryDelay)
		}

		// Execute the cloning
		container = container.WithExec(cloneCmd)
		dir = container.Directory(opts.Name)

		// Check that the directory is not empty
		entries, err := dir.Entries(ctx)
		if err != nil {
			lastErr = fmt.Errorf("error accessing repository files: %w", err)

			continue
		}
		if len(entries) == 0 {
			lastErr = errors.New("repository cloned but is empty")

			continue
		}

		fmt.Println("✅ Repository cloned successfully")
		return dir, nil
	}

	return nil, fmt.Errorf("failed to clone repository after %d attempts: %w", c.opts.MaxRetries, lastErr)
}
