package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      Error
		expected string
	}{
		{
			name:     "port error",
			err:      Error{Err: errors.New("PORT is not defined")},
			expected: "Environment variable: PORT is not defined",
		},
		{
			name:     "custom error",
			err:      Error{Err: errors.New("custom error message")},
			expected: "Environment variable: custom error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	tests := []struct {
		name     string
		err      Error
		expected error
	}{
		{
			name:     "port error",
			err:      Error{Err: errors.New("PORT is not defined")},
			expected: errors.New("PORT is not defined"),
		},
		{
			name:     "custom error",
			err:      Error{Err: errors.New("custom error message")},
			expected: errors.New("custom error message"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Unwrap()
			assert.Equal(t, tt.expected.Error(), result.Error())
		})
	}
}

func TestErrNoPort(t *testing.T) {
	assert.NotNil(t, ErrNoPort)
	assert.Equal(t, "Environment variable: PORT is not defined", ErrNoPort.Error())

	// Test that it can be unwrapped
	unwrapped := ErrNoPort.Unwrap()
	require.Error(t, unwrapped)
	assert.Equal(t, "PORT is not defined", unwrapped.Error())
}

func TestError_ImplementsErrorInterface(t *testing.T) {
	var err error = Error{Err: errors.New("test error")}
	require.Error(t, err)
	assert.Equal(t, "Environment variable: test error", err.Error())
}

func TestError_CanBeWrapped(t *testing.T) {
	originalErr := errors.New("original error")
	configErr := Error{Err: originalErr}

	// Test that the error can be wrapped and unwrapped
	unwrapped := configErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped)

	// Test that errors.Is works
	assert.ErrorIs(t, configErr, originalErr)
}
