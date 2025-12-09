package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseStepConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		timeout time.Duration
		retries int
	}{
		{
			name:    "valid timeout string",
			input:   "5m",
			wantErr: false,
			timeout: 5 * time.Minute,
		},
		{
			name:    "valid timeout with hours",
			input:   "2h",
			wantErr: false,
			timeout: 2 * time.Hour,
		},
		{
			name:    "invalid timeout format",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty timeout",
			input:   "",
			wantErr: false,
			timeout: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := parseStepTimeout(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.timeout, duration)
			}
		})
	}
}

func TestValidateStepTimeout(t *testing.T) {
	tests := []struct {
		name      string
		timeout   time.Duration
		maxTimeout time.Duration
		wantErr   bool
	}{
		{
			name:      "valid timeout within limit",
			timeout:   5 * time.Minute,
			maxTimeout: 2 * time.Hour,
			wantErr:   false,
		},
		{
			name:      "timeout exceeds maximum",
			timeout:   3 * time.Hour,
			maxTimeout: 2 * time.Hour,
			wantErr:   true,
		},
		{
			name:      "zero timeout is valid",
			timeout:   0,
			maxTimeout: 2 * time.Hour,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStepTimeout(tt.timeout, tt.maxTimeout)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}


