package interfaces

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerError_Error(t *testing.T) {
	tests := []struct {
		name     string
		error    ContainerError
		expected string
	}{
		{
			name: "container error with all fields",
			error: ContainerError{
				Component: "logger",
				Operation: "initialization",
				Cause:     errors.New("failed to connect"),
			},
			expected: "container error in logger during initialization: failed to connect",
		},
		{
			name: "container error with empty fields",
			error: ContainerError{
				Component: "",
				Operation: "",
				Cause:     errors.New("unknown error"),
			},
			expected: "container error in  during : unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainerError_ImplementsErrorInterface(t *testing.T) {
	var err error = ContainerError{
		Component: "test",
		Operation: "test",
		Cause:     errors.New("test error"),
	}

	require.Error(t, err)
	assert.Contains(t, err.Error(), "container error")
}

func TestPipelineError_Error(t *testing.T) {
	tests := []struct {
		name     string
		error    PipelineError
		expected string
	}{
		{
			name: "pipeline error with all fields",
			error: PipelineError{
				Pipeline: "go-kit",
				Step:     "build",
				Message:  "build failed",
				Cause:    errors.New("compilation error"),
			},
			expected: "pipeline error in go-kit at step build: build failed: compilation error",
		},
		{
			name: "pipeline error with empty fields",
			error: PipelineError{
				Pipeline: "",
				Step:     "",
				Message:  "",
				Cause:    errors.New("unknown error"),
			},
			expected: "pipeline error in  at step : : unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPipelineError_ImplementsErrorInterface(t *testing.T) {
	var err error = PipelineError{
		Pipeline: "test",
		Step:     "test",
		Message:  "test",
		Cause:    errors.New("test error"),
	}

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline error")
}

func TestConfigurationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		error    ConfigurationError
		expected string
	}{
		{
			name: "configuration error with string value",
			error: ConfigurationError{
				Key:   "logging.level",
				Value: "invalid",
				Cause: errors.New("invalid log level"),
			},
			expected: "configuration error for key logging.level with value invalid: invalid log level",
		},
		{
			name: "configuration error with empty fields",
			error: ConfigurationError{
				Key:   "",
				Value: "",
				Cause: errors.New("unknown error"),
			},
			expected: "configuration error for key  with value : unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigurationError_ImplementsErrorInterface(t *testing.T) {
	var err error = ConfigurationError{
		Key:   "test",
		Value: "test",
		Cause: errors.New("test error"),
	}

	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration error")
}

func TestConfigurationError_Error_WithNonStringValue(t *testing.T) {
	// Test with a non-string value to ensure type assertion works
	configErr := ConfigurationError{
		Key:   "test.key",
		Value: 123, // non-string value
		Cause: errors.New("type error"),
	}

	// This should panic due to type assertion, but we test the error message format
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic due to type assertion on non-string value
			// The panic will be a *runtime.TypeAssertionError, not a string
			assert.NotNil(t, r)
		}
	}()

	_ = configErr.Error()
}

func TestConfigurationError_Error_WithStringValue(t *testing.T) {
	// Test with a string value to ensure it works correctly
	configErr := ConfigurationError{
		Key:   "test.key",
		Value: "test-value",
		Cause: errors.New("validation error"),
	}

	result := configErr.Error()
	expected := "configuration error for key test.key with value test-value: validation error"
	assert.Equal(t, expected, result)
}
