package rust_test

import (
	"context"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/providers/rust"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
)

func TestRustLinter_Test_NilClient(t *testing.T) {
	linter := &rust.RustLinter{}

	out, err := linter.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("RustLinter.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("RustLinter.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("RustLinter.Test() file = %v, want nil on error", out)
	}
}

// TestRustLinter_Test_MockClient_ReportIncludesClippyDiagnostics mirrors
// TestRustLinter_Test_RealEngine_ReportIncludesClippyDiagnostics
// (integration_test.go): clippy writes its diagnostics to stderr, never
// stdout, so RustLinter.Test must fold Stderr into the returned report file
// alongside Stdout. Proven here via daggerkit's mocks by asserting
// WithNewFile is invoked with a report string that contains the mocked
// stderr text.
func TestRustLinter_Test_MockClient_ReportIncludesClippyDiagnostics(t *testing.T) {
	client := &daggerkit.MockDaggerClient{}
	container := &daggerkit.MockDaggerContainer{}
	container.On("From", mock.Anything).Return(container)
	container.On("WithMountedCache", mock.Anything, mock.Anything).Return(container)
	container.On("WithMountedDirectory", mock.Anything, mock.Anything).Return(container)
	container.On("WithWorkdir", mock.Anything).Return(container)
	container.On("WithExec", []string{"rustup", "component", "add", "clippy"}).Return(container)

	lintContainer := &daggerkit.MockDaggerContainer{}
	lintContainer.On("Stdout", mock.Anything).Return("", nil)
	lintContainer.On("Stderr", mock.Anything).Return("Compiling clippytest v0.1.0\nFinished dev target(s)\n", nil)
	container.On("WithExec", []string{"cargo", "clippy", "--all-targets", "--", "-D", "warnings"}).Return(lintContainer)

	realFile := &dagger.File{}
	reportFile := &daggerkit.MockDaggerFile{}
	reportFile.On("GetRealFile").Return(realFile)
	container.On("WithNewFile", "/tmp/lint-report.txt", mock.MatchedBy(func(report string) bool {
		return strings.Contains(report, "Compiling") && strings.Contains(report, "Finished")
	})).Return(container)
	container.On("File", "/tmp/lint-report.txt").Return(reportFile)

	client.On("Container").Return(container)
	client.On("CacheVolume", mock.Anything).Return(&daggerkit.MockDaggerCacheVolume{})

	linter := &rust.RustLinter{Client: client}

	out, err := linter.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	require.Same(t, realFile, out)
	container.AssertExpectations(t)
	lintContainer.AssertExpectations(t)
}
