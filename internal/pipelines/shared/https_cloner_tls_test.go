package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dagger.io/dagger"
	"github.com/stretchr/testify/require"
)

// insecureTLSPatterns are the exact strings a P0 fix must guarantee are
// absent from the HTTPS cloner's source: any of them means TLS verification
// could be silently disabled or degraded for an authenticated clone.
var insecureTLSPatterns = []string{
	"sslVerify", // covers `http.sslVerify false` and `-c http.sslVerify=false`
	"GIT_SSL_NO_VERIFY",
	"InsecureSkipVerify",
	"curl -k",
	"--insecure",
}

// TestHTTPSClonerSource_NeverDisablesTLSVerification is a static regression
// guard: it fails if any future change reintroduces an insecure TLS escape
// hatch in the HTTPS cloner source, independent of whether a Dagger engine
// is available to run the container-level test below.
func TestHTTPSClonerSource_NeverDisablesTLSVerification(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "https_cloner.go"))
	require.NoError(t, err)

	content := string(src)
	for _, pattern := range insecureTLSPatterns {
		require.NotContains(t, content, pattern,
			"https_cloner.go must never contain %q: TLS verification must stay enabled for HTTPS clones", pattern)
	}
}

// TestHTTPSClonerSource_DoesNotLogCredentials guards against credentials
// leaking into logs: no logger.L() call in the HTTPS cloner may reference
// the resolved token, user, or the .netrc content it is written into.
func TestHTTPSClonerSource_DoesNotLogCredentials(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "https_cloner.go"))
	require.NoError(t, err)

	for _, line := range strings.Split(string(src), "\n") {
		if !strings.Contains(line, "logger.L()") {
			continue
		}
		require.NotContains(t, line, "creds.Token", "logging must never include the resolved Git token")
		require.NotContains(t, line, "creds.User", "logging must never include the resolved Git user")
		require.NotContains(t, line, "netrc", "logging must never include .netrc content")
	}
}

// TestBuildCloneContainer_DoesNotPersistInsecureGitConfig exercises the real
// container built for an authenticated HTTPS clone (the exact path that
// used to run `git config --global http.sslVerify false`) and asserts that
// no insecure TLS setting is persisted in the container's global Git config.
func TestBuildCloneContainer_DoesNotPersistInsecureGitConfig(t *testing.T) {
	ctx := t.Context()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	opts := GitCloneOpts{
		Repo:      "https://github.com/example/private-repo.git",
		Branch:    "main",
		Name:      "test-repo",
		UserEmail: "test@example.com",
		UserName:  "Test User",
	}
	creds := &GitCredentials{
		User:      "x-access-token",
		Token:     "fake-token-for-test-only",
		Source:    string(SourcePAT),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	container, err := buildCloneContainer(client, opts, creds)
	require.NoError(t, err)

	configOutput, err := container.
		WithExec([]string{"sh", "-c", "git config --global --list"}).
		Stdout(ctx)
	require.NoError(t, err)

	lowered := strings.ToLower(configOutput)
	require.NotContains(t, lowered, "sslverify",
		"global Git config must not set http.sslVerify; TLS verification must stay on by default")

	// `git config --get` exits non-zero when the key is unset; ANY lets us
	// inspect that exit code instead of failing the WithExec chain outright.
	getResult := container.WithExec(
		[]string{"sh", "-c", "git config --global --get http.sslVerify"},
		dagger.ContainerWithExecOpts{Expect: dagger.ReturnTypeAny},
	)
	exitCode, err := getResult.ExitCode(ctx)
	require.NoError(t, err)
	require.NotEqual(t, 0, exitCode, "http.sslVerify must be unset, not merely absent from --list formatting")

	// Credentials must still reach the container (auth is not broken by the
	// fix) via .netrc rather than an insecure TLS bypass.
	netrcContent, err := container.File("/root/.netrc").Contents(ctx)
	require.NoError(t, err)
	require.Contains(t, netrcContent, "github.com")
	require.Contains(t, netrcContent, "x-access-token")
	require.Contains(t, netrcContent, "fake-token-for-test-only")
}

// TestBuildCloneContainer_AnonymousDoesNotTouchGitConfig ensures the
// anonymous path (no credentials) never writes credential or TLS
// configuration at all.
func TestBuildCloneContainer_AnonymousDoesNotTouchGitConfig(t *testing.T) {
	ctx := t.Context()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	opts := GitCloneOpts{
		Repo:      "https://github.com/example/public-repo.git",
		Branch:    "main",
		Name:      "test-repo",
		UserEmail: "test@example.com",
		UserName:  "Test User",
	}
	creds := &GitCredentials{
		Source:    string(SourceAnonymous),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	container, err := buildCloneContainer(client, opts, creds)
	require.NoError(t, err)

	configOutput, err := container.
		WithExec([]string{"sh", "-c", "git config --global --list"}).
		Stdout(ctx)
	require.NoError(t, err)

	lowered := strings.ToLower(configOutput)
	require.NotContains(t, lowered, "sslverify")
	require.NotContains(t, lowered, "credential.helper")
}
