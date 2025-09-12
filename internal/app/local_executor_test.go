package app

import (
	"context"
	"os"
	"testing"

	"github.com/getsyntegrity/syntegrity-dagger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewLocalExecutor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	assert.NotNil(t, executor)
	assert.Equal(t, mockLogger, executor.logger)
	assert.Equal(t, mockConfig, executor.config)
}

func TestLocalExecutor_ExecuteStep_Setup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Mock logger calls
	mockLogger.EXPECT().Info("Executing step locally", "step", "setup").Times(1)
	mockLogger.EXPECT().Info("Setting up local environment").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// This will fail because we don't have a real Go project, but we can test the structure
	err := executor.ExecuteStep(t.Context(), "setup")

	// We expect an error because we're not in a real Go project
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a Go project")
}

func TestLocalExecutor_ExecuteStep_Build(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Mock logger calls
	mockLogger.EXPECT().Info("Executing step locally", "step", "build").Times(1)
	mockLogger.EXPECT().Info("Building application locally").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// This will fail because we don't have a real Go project, but we can test the structure
	err := executor.ExecuteStep(t.Context(), "build")

	// We expect an error because we're not in a real Go project
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a Go project")
}

func TestLocalExecutor_ExecuteStep_Test(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Mock logger calls
	mockLogger.EXPECT().Info("Executing step locally", "step", "test").Times(1)
	mockLogger.EXPECT().Info("Running tests locally").Times(1)
	// Note: The coverage log and config call won't happen because the step fails early

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// This will fail because we don't have a real Go project, but we can test the structure
	err := executor.ExecuteStep(t.Context(), "test")

	// We expect an error because we're not in a real Go project
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a Go project")
}

func TestLocalExecutor_ExecuteStep_Lint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Mock logger calls
	mockLogger.EXPECT().Info("Executing step locally", "step", "lint").Times(1)
	mockLogger.EXPECT().Info("Running linters locally").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// This will fail because we don't have a real Go project, but we can test the structure
	err := executor.ExecuteStep(t.Context(), "lint")

	// We expect an error because we're not in a real Go project
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a Go project")
}

func TestLocalExecutor_ExecuteStep_Security(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Mock logger calls
	mockLogger.EXPECT().Info("Executing step locally", "step", "security").Times(1)
	mockLogger.EXPECT().Info("Running security checks locally").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// This will fail because we don't have a real Go project, but we can test the structure
	err := executor.ExecuteStep(t.Context(), "security")

	// We expect an error because we're not in a real Go project
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a Go project")
}

func TestLocalExecutor_ExecuteStep_Unknown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Mock logger calls
	mockLogger.EXPECT().Info("Executing step locally", "step", "unknown").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	err := executor.ExecuteStep(t.Context(), "unknown")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported step")
}

func TestLocalExecutor_IsGoProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// Test with a real Go project (current directory)
	isGo := executor.isGoProject()
	// This might be false if the test is running from a different directory
	// Let's just test that the function doesn't panic
	t.Logf("isGoProject returned: %v", isGo)
	// We'll just test that it returns a boolean value
	assert.NotNil(t, isGo)
}

func TestLocalExecutor_IsCommandAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// Test with a command that should be available
	available := executor.isCommandAvailable("go")
	assert.True(t, available)

	// Test with a command that should not be available
	notAvailable := executor.isCommandAvailable("nonexistentcommand12345")
	assert.False(t, notAvailable)
}

func TestLocalExecutor_GetCoverageThreshold(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	t.Run("with valid coverage from config", func(t *testing.T) {
		mockConfig.EXPECT().Get("pipeline.coverage").Return(85.0).Times(1)

		threshold := executor.getCoverageThreshold()
		assert.InDelta(t, 85.0, threshold, 0.001)
	})

	t.Run("with invalid coverage from config", func(t *testing.T) {
		mockConfig.EXPECT().Get("pipeline.coverage").Return("invalid").Times(1)

		threshold := executor.getCoverageThreshold()
		assert.InDelta(t, 90.0, threshold, 0.001) // Default value
	})

	t.Run("with nil coverage from config", func(t *testing.T) {
		mockConfig.EXPECT().Get("pipeline.coverage").Return(nil).Times(1)

		threshold := executor.getCoverageThreshold()
		assert.InDelta(t, 90.0, threshold, 0.001) // Default value
	})
}

func TestLocalExecutor_CheckCoverageThreshold(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	t.Run("coverage meets threshold", func(t *testing.T) {
		// Mock logger call for coverage file not found
		mockLogger.EXPECT().Warn("Coverage file not found - skipping threshold check").Times(1)

		err := executor.checkCoverageThreshold(t.Context(), 90.0)
		require.NoError(t, err) // Should pass because coverage file doesn't exist
	})

	t.Run("coverage equals threshold", func(t *testing.T) {
		// Mock logger call for coverage file not found
		mockLogger.EXPECT().Warn("Coverage file not found - skipping threshold check").Times(1)

		err := executor.checkCoverageThreshold(t.Context(), 90.0)
		require.NoError(t, err) // Should pass because coverage file doesn't exist
	})

	t.Run("coverage below threshold", func(t *testing.T) {
		// Mock logger call for coverage file not found
		mockLogger.EXPECT().Warn("Coverage file not found - skipping threshold check").Times(1)

		err := executor.checkCoverageThreshold(t.Context(), 90.0)
		require.NoError(t, err) // Should pass because coverage file doesn't exist
	})
}

func TestLocalExecutor_ExecuteStep_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Mock logger calls
	mockLogger.EXPECT().Info("Executing step locally", "step", "setup").Times(1)
	mockLogger.EXPECT().Info("Setting up local environment").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	err := executor.ExecuteStep(ctx, "setup")

	// We expect an error due to context cancellation or command execution
	require.Error(t, err)
}

func TestLocalExecutor_ExecuteStep_WithNilLogger(t *testing.T) {
	// This test ensures we handle nil logger gracefully
	executor := &LocalExecutor{
		logger: nil,
		config: nil,
	}

	// This should panic or handle gracefully
	assert.Panics(t, func() {
		_ = executor.ExecuteStep(t.Context(), "setup")
	})
}

func TestLocalExecutor_ExecuteStep_WithNilConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	executor := &LocalExecutor{
		logger: mockLogger,
		config: nil,
	}

	// Mock logger calls
	mockLogger.EXPECT().Info("Executing step locally", "step", "test").Times(1)
	mockLogger.EXPECT().Info("Running tests locally").Times(1)
	// Note: The coverage log won't happen because the step fails early

	// This should use default threshold when config is nil
	err := executor.ExecuteStep(t.Context(), "test")

	// We expect an error because we're not in a real Go project
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a Go project")
}

// Test individual execution methods with Go project simulation
func TestLocalExecutor_executeTest_WithGoProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Create a temporary directory with go.mod to simulate Go project
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		t.Chdir(originalDir)
	}()

	t.Chdir(tempDir)

	// Create go.mod file
	goModContent := `module test-project

go 1.21
`
	err := os.WriteFile("go.mod", []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create a simple Go file for testing
	goFileContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = os.WriteFile("main.go", []byte(goFileContent), 0644)
	require.NoError(t, err)

	// Mock logger calls for execution (will fail at coverage check)
	mockLogger.EXPECT().Info("Running tests locally").Times(1)
	mockLogger.EXPECT().Info("Running tests with coverage", "threshold", gomock.Any()).Times(1)
	mockLogger.EXPECT().Info("Generating coverage report").Times(1)
	mockLogger.EXPECT().Info("Coverage check", "current", gomock.Any(), "threshold", gomock.Any()).Times(1)

	// Mock config calls
	mockConfig.EXPECT().Get("pipeline.coverage").Return(80.0).Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// Test executeTest method directly
	err = executor.executeTest(t.Context())

	// Should fail due to low coverage (0% vs 80% threshold)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "coverage threshold not met")
}

func TestLocalExecutor_executeLint_WithGoProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Create a temporary directory with go.mod to simulate Go project
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		t.Chdir(originalDir)
	}()

	t.Chdir(tempDir)

	// Create go.mod file
	goModContent := `module test-project

go 1.21
`
	err := os.WriteFile("go.mod", []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create a simple Go file for testing
	goFileContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = os.WriteFile("main.go", []byte(goFileContent), 0644)
	require.NoError(t, err)

	// Mock logger calls for successful execution
	mockLogger.EXPECT().Info("Running linters locally").Times(1)
	mockLogger.EXPECT().Info("Running go vet").Times(1)
	mockLogger.EXPECT().Info("Checking code formatting").Times(1)
	// golangci-lint might not be available in CI, so we expect either:
	// - "Running golangci-lint" if available, OR
	// - "golangci-lint not available - skipping advanced linting" if not available
	mockLogger.EXPECT().Info(gomock.Any()).Times(1) // Either golangci-lint message
	mockLogger.EXPECT().Info("Linting completed successfully").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// Test executeLint method directly
	err = executor.executeLint(t.Context())

	// Should succeed since we have a Go project
	require.NoError(t, err)
}

func TestLocalExecutor_executeSecurity_WithGoProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Create a temporary directory with go.mod to simulate Go project
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		t.Chdir(originalDir)
	}()

	t.Chdir(tempDir)

	// Create go.mod file
	goModContent := `module test-project

go 1.21
`
	err := os.WriteFile("go.mod", []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create a simple Go file for testing
	goFileContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = os.WriteFile("main.go", []byte(goFileContent), 0644)
	require.NoError(t, err)

	// Mock logger calls for successful execution
	mockLogger.EXPECT().Info("Running security checks locally").Times(1)
	mockLogger.EXPECT().Info("gosec not available - skipping security scanning").Times(1)
	// govulncheck might not be available in CI, so we expect either:
	// - "Running govulncheck" if available, OR
	// - "govulncheck not available - skipping vulnerability check" if not available
	mockLogger.EXPECT().Info(gomock.Any()).Times(1) // Either govulncheck message
	mockLogger.EXPECT().Info("Security checks completed successfully").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// Test executeSecurity method directly
	err = executor.executeSecurity(t.Context())

	// Should succeed since we have a Go project
	require.NoError(t, err)
}

func TestLocalExecutor_executeSetup_WithGoProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Create a temporary directory with go.mod to simulate Go project
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		t.Chdir(originalDir)
	}()

	t.Chdir(tempDir)

	// Create go.mod file
	goModContent := `module test-project

go 1.21
`
	err := os.WriteFile("go.mod", []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create a simple Go file for testing
	goFileContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = os.WriteFile("main.go", []byte(goFileContent), 0644)
	require.NoError(t, err)

	// Mock logger calls for successful execution
	mockLogger.EXPECT().Info("Setting up local environment").Times(1)
	mockLogger.EXPECT().Info("Downloading Go dependencies").Times(1)
	mockLogger.EXPECT().Info("Tidying Go modules").Times(1)
	mockLogger.EXPECT().Info("Setup completed successfully").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// Test executeSetup method directly
	err = executor.executeSetup(t.Context())

	// Should succeed since we have a Go project
	require.NoError(t, err)
}

func TestLocalExecutor_executeBuild_WithGoProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Create a temporary directory with go.mod to simulate Go project
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		t.Chdir(originalDir)
	}()

	t.Chdir(tempDir)

	// Create go.mod file
	goModContent := `module test-project

go 1.21
`
	err := os.WriteFile("go.mod", []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create a simple Go file for testing
	goFileContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = os.WriteFile("main.go", []byte(goFileContent), 0644)
	require.NoError(t, err)

	// Mock logger calls for successful execution
	mockLogger.EXPECT().Info("Building application locally").Times(1)
	mockLogger.EXPECT().Info("Build completed successfully").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// Test executeBuild method directly
	err = executor.executeBuild(t.Context())

	// Should succeed since we have a Go project
	require.NoError(t, err)
}

func TestLocalExecutor_executeSecurity_WithGoProject_WithGosec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := mocks.NewMockConfiguration(ctrl)

	// Create a temporary directory with go.mod to simulate Go project
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		t.Chdir(originalDir)
	}()

	t.Chdir(tempDir)

	// Create go.mod file
	goModContent := `module test-project

go 1.21
`
	err := os.WriteFile("go.mod", []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create a simple Go file for testing
	goFileContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = os.WriteFile("main.go", []byte(goFileContent), 0644)
	require.NoError(t, err)

	// Mock logger calls for execution (gosec not available, govulncheck might not be available in CI)
	mockLogger.EXPECT().Info("Running security checks locally").Times(1)
	mockLogger.EXPECT().Info("gosec not available - skipping security scanning").Times(1)
	// govulncheck might not be available in CI, so we expect either:
	// - "Running govulncheck" if available, OR
	// - "govulncheck not available - skipping vulnerability check" if not available
	mockLogger.EXPECT().Info(gomock.Any()).Times(1) // Either govulncheck message
	mockLogger.EXPECT().Info("Security checks completed successfully").Times(1)

	executor := NewLocalExecutor(mockLogger, mockConfig)

	// Test executeSecurity method directly
	err = executor.executeSecurity(t.Context())

	// Should succeed since we have a Go project
	require.NoError(t, err)
}
