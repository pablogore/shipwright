package golang_test

import (
	"context"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/providers/go"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

func TestGoVulnScanner_Test_NilClient(t *testing.T) {
	scanner := &golang.GoVulnScanner{}

	out, err := scanner.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("GoVulnScanner.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("GoVulnScanner.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("GoVulnScanner.Test() file = %v, want nil on error", out)
	}
}

// TestGoVulnScanner_Test_CleanScan proves the false-positive regression
// this package's own naming_test.go/internal_test.go documents: a clean
// govulncheck run whose own summary literally reads "Your code is
// affected by 0 vulnerabilities" must still be treated as clean, and its
// output returned as the report File.
func TestGoVulnScanner_Test_CleanScan(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockReportFile := &daggerkit.MockDaggerFile{}
	src := &dagger.Directory{}
	realReportFile := &dagger.File{}
	const cleanOutput = "No vulnerabilities found.\n\nYour code is affected by 0 vulnerabilities."

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.25.5").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/app", mock.Anything).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/app").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GO111MODULE", "on").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "0").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("WithExec", []string{"govulncheck", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Stdout", mock.Anything).Return(cleanOutput, nil)
	mockContainer.On("WithNewFile", "/tmp/vuln-report.txt", cleanOutput).Return(mockContainer)
	mockContainer.On("File", "/tmp/vuln-report.txt").Return(mockReportFile)
	mockReportFile.On("GetRealFile").Return(realReportFile)

	scanner := &golang.GoVulnScanner{Client: mockClient}

	out, err := scanner.Test(context.Background(), src)

	require.NoError(t, err)
	assert.Same(t, realReportFile, out)
	mockContainer.AssertExpectations(t)
}

// TestGoVulnScanner_Test_VulnerabilitiesDetected proves the
// finding-reported path: a non-zero affected count fails the run with a
// wrapped error naming the detected output, without ever writing a report
// File for the caller to read.
func TestGoVulnScanner_Test_VulnerabilitiesDetected(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	src := &dagger.Directory{}
	const vulnOutput = "Your code is affected by 2 vulnerabilities"

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.25.5").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/app", mock.Anything).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/app").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GO111MODULE", "on").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "0").Return(mockContainer)
	mockContainer.On("WithExec", mock.Anything, mock.Anything).Return(mockContainer)
	mockContainer.On("Stdout", mock.Anything).Return(vulnOutput, nil)

	scanner := &golang.GoVulnScanner{Client: mockClient}

	out, err := scanner.Test(context.Background(), src)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "govulnscanner: security vulnerabilities detected")
	assert.Contains(t, err.Error(), vulnOutput)
	assert.Nil(t, out)
	mockContainer.AssertNotCalled(t, "WithNewFile", mock.Anything, mock.Anything)
}
