// Package shared provides shared functionality for pipeline implementations.
package shared

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"dagger.io/dagger"
	"github.com/pablogore/kit-logger/pkg/logger"
)

const knownHostsPath = "/root/.ssh/known_hosts"

type SSHCloner struct{}

// validateKnownHosts checks that content is well-formed OpenSSH known_hosts
// data. It is operator-supplied trusted material and is never derived by
// Shipwright itself (no ssh-keyscan, no TOFU, no runtime trust decisions).
func validateKnownHosts(content string) error {
	validEntries := 0
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "@cert-authority ")
		line = strings.TrimPrefix(line, "@revoked ")
		if len(strings.Fields(line)) < 3 {
			return errors.New("malformed known_hosts content: each entry needs at least 3 fields")
		}
		validEntries++
	}
	if validEntries == 0 {
		return errors.New("malformed known_hosts content: no valid entries found")
	}
	return nil
}

// buildSSHCommand returns the GIT_SSH_COMMAND value that enforces strict SSH
// host-key verification against the given trusted known_hosts file.
func buildSSHCommand(hostsPath string) string {
	return "ssh -o StrictHostKeyChecking=yes -o UserKnownHostsFile=" + hostsPath
}

func (c *SSHCloner) Clone(ctx context.Context, client *dagger.Client, opts GitCloneOpts) (*dagger.Directory, error) {
	logger.L().InfoContext(ctx, "Cloning repo (SSH)", "name", opts.Name, "branch", opts.Branch)

	knownHostsContent := os.Getenv("SSH_KNOWN_HOSTS")
	if strings.TrimSpace(knownHostsContent) == "" {
		return nil, errors.New("SSH_KNOWN_HOSTS not set: refusing to clone over SSH without trusted host keys")
	}
	if err := validateKnownHosts(knownHostsContent); err != nil {
		return nil, fmt.Errorf("❌ invalid SSH_KNOWN_HOSTS: %w", err)
	}

	keyContent := os.Getenv("SSH_PRIVATE_KEY")

	if keyContent == "" {
		sshKeyPath := os.ExpandEnv("$HOME/.ssh/syntegrity")
		data, err := os.ReadFile(sshKeyPath)
		if err != nil {
			return nil, fmt.Errorf("❌ SSH_PRIVATE_KEY not set and no local key found: %w", err)
		}
		keyContent = string(data)
	}

	hostDir := client.Host().Directory(".").
		WithNewFile("id_rsa", keyContent).
		WithNewFile("known_hosts", knownHostsContent)

	email := opts.UserEmail
	if email == "" {
		email = "ci@getsyntegrity.com"
	}
	name := opts.UserName
	if name == "" {
		name = "Syntegrity CI"
	}

	container := client.Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "--no-cache", "git", "openssh"}).
		WithMountedFile("/root/.ssh/id_rsa", hostDir.File("id_rsa")).
		WithExec([]string{"chmod", "600", "/root/.ssh/id_rsa"}).
		WithMountedFile(knownHostsPath, hostDir.File("known_hosts")).
		WithExec([]string{"chmod", "600", knownHostsPath}).
		WithEnvVariable("GIT_SSH_COMMAND", buildSSHCommand(knownHostsPath)).
		WithExec([]string{"git", "config", "--global", "user.email", email}).
		WithExec([]string{"git", "config", "--global", "user.name", name}).
		WithExec([]string{"git", "clone", "--depth=1", "--branch", opts.Branch, opts.Repo, opts.Name}).
		WithWorkdir("/")

	dir := container.Directory(opts.Name)
	entries, err := dir.Entries(ctx)
	if err != nil {
		return nil, fmt.Errorf("❌ error accessing repository files: %w", err)
	}
	if len(entries) == 0 {
		return nil, errors.New("❌ repository cloned but is empty")
	}

	logger.L().InfoContext(ctx, "Repository cloned successfully")
	return dir, nil
}
