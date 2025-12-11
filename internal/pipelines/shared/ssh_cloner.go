// Package shared provides shared functionality for pipeline implementations.
package shared

import (
	"context"
	"errors"
	"fmt"
	"os"

	"dagger.io/dagger"
	"github.com/getsyntegrity/go-kit-logger/pkg/logger"
)

type SSHCloner struct{}

func (c *SSHCloner) Clone(ctx context.Context, client *dagger.Client, opts GitCloneOpts) (*dagger.Directory, error) {
	logger.L().InfoContext(ctx, "Cloning repo (SSH)", "name", opts.Name, "branch", opts.Branch)

	keyContent := os.Getenv("SSH_PRIVATE_KEY")

	if keyContent == "" {
		sshKeyPath := os.ExpandEnv("$HOME/.ssh/syntegrity")
		data, err := os.ReadFile(sshKeyPath)
		if err != nil {
			return nil, fmt.Errorf("❌ SSH_PRIVATE_KEY not set and no local key found: %w", err)
		}
		keyContent = string(data)
	}

	hostDir := client.Host().Directory(".").WithNewFile("id_rsa", keyContent)

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
		WithEnvVariable("GIT_SSH_COMMAND", "ssh -o StrictHostKeyChecking=no").
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
