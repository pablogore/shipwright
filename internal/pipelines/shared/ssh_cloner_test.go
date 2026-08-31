package shared

import (
	"context"
	"os"
	"strings"
	"testing"
)

// NOTE: host-key-mismatch causing a real SSH failure is not covered by an
// automated test here (no SSH server test harness exists in this repo); that
// behavior is guaranteed by OpenSSH itself once StrictHostKeyChecking=yes and
// a real UserKnownHostsFile are in effect, which the tests below prove is
// always what gets emitted.

func TestValidateKnownHosts_RejectsEmpty(t *testing.T) {
	if err := validateKnownHosts(""); err == nil {
		t.Fatal("expected error for empty known_hosts content, got nil")
	}
	if err := validateKnownHosts("   \n  "); err == nil {
		t.Fatal("expected error for blank known_hosts content, got nil")
	}
}

func TestValidateKnownHosts_RejectsMalformed(t *testing.T) {
	content := "not-known-hosts\nalso-garbage\n"
	if err := validateKnownHosts(content); err == nil {
		t.Fatal("expected error for malformed known_hosts content, got nil")
	}
}

func TestValidateKnownHosts_AcceptsValid(t *testing.T) {
	content := "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n" +
		"# a comment line\n" +
		"\n" +
		"gitlab.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC\n"
	if err := validateKnownHosts(content); err != nil {
		t.Fatalf("expected no error for valid known_hosts content, got %v", err)
	}
}

func TestBuildSSHCommand_NeverDisablesHostKeyChecking(t *testing.T) {
	cmd := buildSSHCommand("/root/.ssh/known_hosts")
	if strings.Contains(cmd, "StrictHostKeyChecking=no") {
		t.Fatalf("buildSSHCommand output must never disable host key checking, got: %q", cmd)
	}
	if strings.Contains(cmd, "UserKnownHostsFile=/dev/null") {
		t.Fatalf("buildSSHCommand output must never point known_hosts at /dev/null, got: %q", cmd)
	}
}

func TestBuildSSHCommand_UsesStrictCheckingAndControlledKnownHosts(t *testing.T) {
	knownHostsPath := "/root/.ssh/known_hosts"
	cmd := buildSSHCommand(knownHostsPath)
	if !strings.Contains(cmd, "StrictHostKeyChecking=yes") {
		t.Fatalf("expected buildSSHCommand output to enforce strict host key checking, got: %q", cmd)
	}
	if !strings.Contains(cmd, knownHostsPath) {
		t.Fatalf("expected buildSSHCommand output to reference known_hosts path %q, got: %q", knownHostsPath, cmd)
	}
}

func TestSSHCloner_Clone_FailsClosedWhenKnownHostsMissing(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS", "")

	opts := GitCloneOpts{
		Repo:   "git@github.com:example/repo.git",
		Branch: "main",
		Name:   "repo",
	}

	// client is nil to prove the known_hosts check runs before any Dagger
	// client/network work; a panic here would mean the check ran too late.
	_, err := (&SSHCloner{}).Clone(context.Background(), nil, opts)
	if err == nil {
		t.Fatal("expected error when SSH_KNOWN_HOSTS is unset, got nil")
	}
}

func TestSSHCloner_Clone_FailsClosedWhenKnownHostsMalformed(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS", "garbage-not-known-hosts-format")

	opts := GitCloneOpts{
		Repo:   "git@github.com:example/repo.git",
		Branch: "main",
		Name:   "repo",
	}

	_, err := (&SSHCloner{}).Clone(context.Background(), nil, opts)
	if err == nil {
		t.Fatal("expected error when SSH_KNOWN_HOSTS is malformed, got nil")
	}
}

// This preserves the pre-existing "no private key available" fail-closed
// behavior. Since known_hosts validation now runs first (cheapest, no I/O),
// SSH_KNOWN_HOSTS is set to valid content here to get past that check and
// reach the private-key resolution path, which fails without touching a
// real dagger.Client because no key is available.
func TestSSHCloner_Clone_FailsClosedWhenPrivateKeyMissing(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS", "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n")
	t.Setenv("SSH_PRIVATE_KEY", "")
	t.Setenv("HOME", t.TempDir())

	opts := GitCloneOpts{
		Repo:   "git@github.com:example/repo.git",
		Branch: "main",
		Name:   "repo",
	}

	_, err := (&SSHCloner{}).Clone(context.Background(), nil, opts)
	if err == nil {
		t.Fatal("expected error when neither SSH_PRIVATE_KEY nor a local key is available, got nil")
	}
}

// Durable structural regression guard, independent of any future refactor of
// this file: scans this package's own production source (not this test
// file) to ensure the insecure SSH options never reappear.
func TestSSHClonerSource_NeverDisablesHostKeyChecking(t *testing.T) {
	src, err := os.ReadFile("ssh_cloner.go")
	if err != nil {
		t.Fatalf("failed to read ssh_cloner.go: %v", err)
	}
	content := string(src)
	if strings.Contains(content, "StrictHostKeyChecking=no") {
		t.Fatal("ssh_cloner.go must never contain StrictHostKeyChecking=no")
	}
	if strings.Contains(content, "UserKnownHostsFile=/dev/null") {
		t.Fatal("ssh_cloner.go must never contain UserKnownHostsFile=/dev/null")
	}
}
