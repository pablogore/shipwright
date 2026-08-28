package app

import (
	"testing"
	"time"

	"dagger.io/dagger"
	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseStepHandler(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBaseStepHandler(mockConfig, nil)
	assert.NotNil(t, handler)
	assert.Equal(t, mockConfig, handler.config)
	assert.Nil(t, handler.client)
	// Logger is used directly via logger.L() - no field to check
}

func TestBaseStepHandler_CanHandle(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBaseStepHandler(mockConfig, nil)

	// Base handler should not handle any specific steps
	assert.False(t, handler.CanHandle("setup"))
	assert.False(t, handler.CanHandle("build"))
	assert.False(t, handler.CanHandle("test"))
}

func TestBaseStepHandler_Execute(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBaseStepHandler(mockConfig, nil)

	// Base handler should return error for any step
	config := interfaces.StepConfig{Name: "test-step"}
	err := handler.Execute(t.Context(), "test-step", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base step handler cannot execute step")
}

func TestBaseStepHandler_GetStepInfo(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBaseStepHandler(mockConfig, nil)

	// Test GetStepInfo
	config := handler.GetStepInfo("test-step")
	assert.Equal(t, "test-step", config.Name)
	assert.Equal(t, "Base step handler - not implemented", config.Description)
	assert.False(t, config.Required)
	assert.False(t, config.Parallel)
	assert.Equal(t, 1*time.Minute, config.Timeout)
	assert.Equal(t, 0, config.Retries)
	assert.Empty(t, config.DependsOn)
	assert.Empty(t, config.Conditions)
	assert.Empty(t, config.Metadata)
}

func TestBaseStepHandler_Validate(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBaseStepHandler(mockConfig, nil)

	// Base handler should return error for any validation
	config := interfaces.StepConfig{Name: "test-step"}
	err := handler.Validate("test-step", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base step handler cannot validate step")
}

func TestNewSetupStepHandler(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewSetupStepHandler(mockConfig, nil)
	assert.NotNil(t, handler)
	assert.Implements(t, (*interfaces.StepHandler)(nil), handler)
}

func TestSetupStepHandler_CanHandle(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewSetupStepHandler(mockConfig, nil).(*SetupStepHandler)

	// Should handle setup step
	assert.True(t, handler.CanHandle("setup"))
	assert.False(t, handler.CanHandle("build"))
	assert.False(t, handler.CanHandle("test"))
}

func TestSetupStepHandler_Execute(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewSetupStepHandler(mockConfig, nil).(*SetupStepHandler)

	// Test Execute with nil client
	config := interfaces.StepConfig{
		Name:     "setup",
		Timeout:  5 * time.Minute,
		Required: true,
	}

	// Logger is used directly via logger.L() - no need to mock

	err := handler.Execute(t.Context(), "setup", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dagger client not available")
}

func TestSetupStepHandler_Execute_WithClient(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	// Use nil for dagger client since it's an external dependency
	var mockClient *dagger.Client
	handler := NewSetupStepHandler(mockConfig, mockClient).(*SetupStepHandler)

	// Test Execute with client
	config := interfaces.StepConfig{
		Name:     "setup",
		Timeout:  5 * time.Minute,
		Required: true,
	}

	// Logger is used directly via logger.L() - no need to mock

	err := handler.Execute(t.Context(), "setup", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dagger client not available")
}

func TestSetupStepHandler_GetStepInfo(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewSetupStepHandler(mockConfig, nil).(*SetupStepHandler)

	// Test GetStepInfo
	config := handler.GetStepInfo("setup")
	assert.Equal(t, "setup", config.Name)
	assert.Equal(t, "Initialize the pipeline environment and prepare source code", config.Description)
	assert.True(t, config.Required)
	assert.False(t, config.Parallel)
	assert.Equal(t, 5*time.Minute, config.Timeout)
	assert.Equal(t, 2, config.Retries)
	assert.Empty(t, config.DependsOn)
	assert.Contains(t, config.Conditions, "source_exists")
	assert.Equal(t, "true", config.Conditions["source_exists"])
	assert.Contains(t, config.Metadata, "category")
	assert.Equal(t, "initialization", config.Metadata["category"])
	assert.Contains(t, config.Metadata, "priority")
	assert.Equal(t, "high", config.Metadata["priority"])
}

func TestSetupStepHandler_Validate(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewSetupStepHandler(mockConfig, nil).(*SetupStepHandler)

	// Test Validate with correct step name
	config := interfaces.StepConfig{Name: "setup"}
	err := handler.Validate("setup", config)
	require.NoError(t, err)

	// Test Validate with incorrect step name
	config = interfaces.StepConfig{Name: "build"}
	err = handler.Validate("setup", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid step name for setup handler")
}

func TestNewBuildStepHandler(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBuildStepHandler(mockConfig, nil)
	assert.NotNil(t, handler)
	assert.Implements(t, (*interfaces.StepHandler)(nil), handler)
}

func TestBuildStepHandler_CanHandle(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBuildStepHandler(mockConfig, nil).(*BuildStepHandler)

	// Should handle build step
	assert.True(t, handler.CanHandle("build"))
	assert.False(t, handler.CanHandle("setup"))
	assert.False(t, handler.CanHandle("test"))
}

func TestBuildStepHandler_Execute(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBuildStepHandler(mockConfig, nil).(*BuildStepHandler)

	// Test Execute
	config := interfaces.StepConfig{Name: "build"}

	// Mock config calls
	mockConfig.GetStringFunc = func(key string) string {
		if key == "pipeline.go_version" {
			return "1.21"
		}
		return ""
	}

	err := handler.Execute(t.Context(), "build", config)
	require.NoError(t, err)
}

func TestBuildStepHandler_Execute_DefaultGoVersion(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBuildStepHandler(mockConfig, nil).(*BuildStepHandler)

	// Test Execute with empty Go version
	config := interfaces.StepConfig{Name: "build"}

	// Mock config calls
	mockConfig.GetStringFunc = func(key string) string {
		if key == "pipeline.go_version" {
			return ""
		}
		return ""
	}

	err := handler.Execute(t.Context(), "build", config)
	require.NoError(t, err)
}

func TestBuildStepHandler_GetStepInfo(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBuildStepHandler(mockConfig, nil).(*BuildStepHandler)

	// Test GetStepInfo
	config := handler.GetStepInfo("build")
	assert.Equal(t, "build", config.Name)
	assert.Equal(t, "Build the application binary or container image", config.Description)
	assert.True(t, config.Required)
	assert.False(t, config.Parallel)
	assert.Equal(t, 10*time.Minute, config.Timeout)
	assert.Equal(t, 1, config.Retries)
	assert.Contains(t, config.DependsOn, "setup")
	assert.Contains(t, config.Conditions, "source_available")
	assert.Equal(t, "true", config.Conditions["source_available"])
	assert.Contains(t, config.Metadata, "category")
	assert.Equal(t, "compilation", config.Metadata["category"])
	assert.Contains(t, config.Metadata, "priority")
	assert.Equal(t, "high", config.Metadata["priority"])
}

func TestBuildStepHandler_Validate(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewBuildStepHandler(mockConfig, nil).(*BuildStepHandler)

	// Test Validate with correct step name
	config := interfaces.StepConfig{Name: "build"}
	err := handler.Validate("build", config)
	require.NoError(t, err)

	// Test Validate with incorrect step name
	config = interfaces.StepConfig{Name: "setup"}
	err = handler.Validate("build", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid step name for build handler")
}

func TestNewTestStepHandler(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewTestStepHandler(mockConfig, nil)
	assert.NotNil(t, handler)
	assert.Implements(t, (*interfaces.StepHandler)(nil), handler)
}

func TestTestStepHandler_CanHandle(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewTestStepHandler(mockConfig, nil).(*TestStepHandler)

	// Should handle test step
	assert.True(t, handler.CanHandle("test"))
	assert.False(t, handler.CanHandle("setup"))
	assert.False(t, handler.CanHandle("build"))
}

func TestTestStepHandler_Execute(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewTestStepHandler(mockConfig, nil).(*TestStepHandler)

	// Test Execute
	config := interfaces.StepConfig{Name: "test"}

	// Mock config calls
	mockConfig.GetFloatFunc = func(key string) float64 {
		if key == "pipeline.coverage" {
			return 90.0
		}
		return 0
	}

	err := handler.Execute(t.Context(), "test", config)
	require.NoError(t, err)
}

func TestTestStepHandler_GetStepInfo(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewTestStepHandler(mockConfig, nil).(*TestStepHandler)

	// Test GetStepInfo
	config := handler.GetStepInfo("test")
	assert.Equal(t, "test", config.Name)
	assert.Equal(t, "Run unit tests and generate coverage reports", config.Description)
	assert.True(t, config.Required)
	assert.True(t, config.Parallel)
	assert.Equal(t, 15*time.Minute, config.Timeout)
	assert.Equal(t, 2, config.Retries)
	assert.Contains(t, config.DependsOn, "build")
	assert.Contains(t, config.Conditions, "tests_available")
	assert.Equal(t, "true", config.Conditions["tests_available"])
	assert.Contains(t, config.Metadata, "category")
	assert.Equal(t, "testing", config.Metadata["category"])
	assert.Contains(t, config.Metadata, "priority")
	assert.Equal(t, "high", config.Metadata["priority"])
}

func TestTestStepHandler_Validate(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewTestStepHandler(mockConfig, nil).(*TestStepHandler)

	// Test Validate with correct step name
	config := interfaces.StepConfig{Name: "test"}
	err := handler.Validate("test", config)
	require.NoError(t, err)

	// Test Validate with incorrect step name
	config = interfaces.StepConfig{Name: "build"}
	err = handler.Validate("test", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid step name for test handler")
}

func TestNewLintStepHandler(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewLintStepHandler(mockConfig, nil)
	assert.NotNil(t, handler)
	assert.Implements(t, (*interfaces.StepHandler)(nil), handler)
}

func TestLintStepHandler_CanHandle(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewLintStepHandler(mockConfig, nil).(*LintStepHandler)

	// Should handle lint step
	assert.True(t, handler.CanHandle("lint"))
	assert.False(t, handler.CanHandle("setup"))
	assert.False(t, handler.CanHandle("build"))
}

func TestLintStepHandler_Execute(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewLintStepHandler(mockConfig, nil).(*LintStepHandler)

	// Test Execute
	config := interfaces.StepConfig{Name: "lint"}

	// Mock config calls
	mockConfig.GetStringFunc = func(key string) string {
		if key == "security.lint_timeout" {
			return "5m"
		}
		return ""
	}

	err := handler.Execute(t.Context(), "lint", config)
	require.NoError(t, err)
}

func TestLintStepHandler_GetStepInfo(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewLintStepHandler(mockConfig, nil).(*LintStepHandler)

	// Test GetStepInfo
	config := handler.GetStepInfo("lint")
	assert.Equal(t, "lint", config.Name)
	assert.Equal(t, "Run code linting and formatting checks", config.Description)
	assert.False(t, config.Required)
	assert.True(t, config.Parallel)
	assert.Equal(t, 5*time.Minute, config.Timeout)
	assert.Equal(t, 1, config.Retries)
	assert.Contains(t, config.DependsOn, "setup")
	assert.Contains(t, config.Conditions, "linting_enabled")
	assert.Equal(t, "true", config.Conditions["linting_enabled"])
	assert.Contains(t, config.Metadata, "category")
	assert.Equal(t, "quality", config.Metadata["category"])
	assert.Contains(t, config.Metadata, "priority")
	assert.Equal(t, "medium", config.Metadata["priority"])
}

func TestLintStepHandler_Validate(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewLintStepHandler(mockConfig, nil).(*LintStepHandler)

	// Test Validate with correct step name
	config := interfaces.StepConfig{Name: "lint"}
	err := handler.Validate("lint", config)
	require.NoError(t, err)

	// Test Validate with incorrect step name
	config = interfaces.StepConfig{Name: "build"}
	err = handler.Validate("lint", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid step name for lint handler")
}

func TestNewSecurityStepHandler(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewSecurityStepHandler(mockConfig, nil)
	assert.NotNil(t, handler)
	assert.Implements(t, (*interfaces.StepHandler)(nil), handler)
}

func TestSecurityStepHandler_CanHandle(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewSecurityStepHandler(mockConfig, nil).(*SecurityStepHandler)

	// Should handle security step
	assert.True(t, handler.CanHandle("security"))
	assert.False(t, handler.CanHandle("setup"))
	assert.False(t, handler.CanHandle("build"))
}

func TestSecurityStepHandler_Execute(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewSecurityStepHandler(mockConfig, nil).(*SecurityStepHandler)

	// Test Execute
	config := interfaces.StepConfig{Name: "security"}

	// Mock config calls
	mockConfig.GetBoolFunc = func(key string) bool {
		return key == "security.enable_vuln_check"
	}

	err := handler.Execute(t.Context(), "security", config)
	require.NoError(t, err)
}

func TestSecurityStepHandler_GetStepInfo(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewSecurityStepHandler(mockConfig, nil).(*SecurityStepHandler)

	// Test GetStepInfo
	config := handler.GetStepInfo("security")
	assert.Equal(t, "security", config.Name)
	assert.Equal(t, "Run security scans and vulnerability checks", config.Description)
	assert.False(t, config.Required)
	assert.True(t, config.Parallel)
	assert.Equal(t, 10*time.Minute, config.Timeout)
	assert.Equal(t, 1, config.Retries)
	assert.Contains(t, config.DependsOn, "build")
	assert.Contains(t, config.Conditions, "security_enabled")
	assert.Equal(t, "true", config.Conditions["security_enabled"])
	assert.Contains(t, config.Metadata, "category")
	assert.Equal(t, "security", config.Metadata["category"])
	assert.Contains(t, config.Metadata, "priority")
	assert.Equal(t, "high", config.Metadata["priority"])
}

func TestSecurityStepHandler_Validate(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewSecurityStepHandler(mockConfig, nil).(*SecurityStepHandler)

	// Test Validate with correct step name
	config := interfaces.StepConfig{Name: "security"}
	err := handler.Validate("security", config)
	require.NoError(t, err)

	// Test Validate with incorrect step name
	config = interfaces.StepConfig{Name: "build"}
	err = handler.Validate("security", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid step name for security handler")
}

func TestNewTagStepHandler(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewTagStepHandler(mockConfig, nil)
	assert.NotNil(t, handler)
	assert.Implements(t, (*interfaces.StepHandler)(nil), handler)
}

func TestNewPackageStepHandler(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewPackageStepHandler(mockConfig, nil)
	assert.NotNil(t, handler)
	assert.Implements(t, (*interfaces.StepHandler)(nil), handler)
}

func TestNewPushStepHandler(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewPushStepHandler(mockConfig, nil)
	assert.NotNil(t, handler)
	assert.Implements(t, (*interfaces.StepHandler)(nil), handler)
}

func TestNewReleaseStepHandler(t *testing.T) {
	mockConfig := NewMockConfiguration()
	// Logger is used directly via logger.L() - no need to mock

	handler := NewReleaseStepHandler(mockConfig, nil)
	assert.NotNil(t, handler)
	assert.Implements(t, (*interfaces.StepHandler)(nil), handler)
}
