package engine_test

import (
	"errors"
	"testing"
	"time"

	"github.com/pablogore/shipwright/internal/workflow/engine"
)

// TestErrorMessages exercises every exported error type's Error() string
// and StepStatus's String() — small, mechanical, but real: each message
// must actually name the step id (and any other identifying field) it
// claims to, not just satisfy the error interface.
func TestErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "StepFailedError",
			err:  &engine.StepFailedError{StepID: "build", Err: errors.New("boom")},
			want: `engine: step "build" failed: boom`,
		},
		{
			name: "StepTimeoutError",
			err:  &engine.StepTimeoutError{StepID: "build", Timeout: 5 * time.Second},
			want: `engine: step "build" exceeded its 5s timeout`,
		},
		{
			name: "UnknownCapabilityError",
			err:  &engine.UnknownCapabilityError{StepID: "build", Capability: "teleport"},
			want: `engine: step "build" has unknown capability "teleport"`,
		},
		{
			name: "MissingStepOutputError",
			err:  &engine.MissingStepOutputError{StepID: "publish", ReferencedStepID: "build"},
			want: `engine: step "publish" references steps.build.output, but "build" has no available output`,
		},
		{
			name: "OutputKindMismatchError",
			err:  &engine.OutputKindMismatchError{StepID: "run", ReferencedStepID: "unit", Field: "input", Want: "directory"},
			want: `engine: step "run" field "input" references steps.unit.output, which does not produce a directory value`,
		},
		{
			name: "InvalidInputReferenceError",
			err:  &engine.InvalidInputReferenceError{StepID: "run"},
			want: `engine: step "run" field "input" must be exactly one steps.<id>.output reference`,
		},
		{
			name: "UndeclaredVariableError",
			err:  &engine.UndeclaredVariableError{StepID: "publish", Name: "imageRef"},
			want: `engine: step "publish" references undeclared variable "imageRef"`,
		},
		{
			name: "UndeclaredSecretError",
			err:  &engine.UndeclaredSecretError{StepID: "publish", Name: "registry"},
			want: `engine: step "publish" references undeclared secret "registry"`,
		},
		{
			name: "MissingWithFieldError",
			err:  &engine.MissingWithFieldError{StepID: "publish", Field: "ref"},
			want: `engine: step "publish" is missing required with field "ref"`,
		},
		{
			name: "UnknownStepError",
			err:  &engine.UnknownStepError{StepID: "ghost"},
			want: `engine: unknown step "ghost"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStepFailedError_Unwrap(t *testing.T) {
	t.Parallel()

	inner := errors.New("boom")
	err := &engine.StepFailedError{StepID: "build", Err: inner}
	if !errors.Is(err, inner) {
		t.Fatalf("errors.Is(err, inner) = false, want true (Unwrap must expose the inner error)")
	}
}

func TestStepStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status engine.StepStatus
		want   string
	}{
		{engine.StatusSucceeded, "succeeded"},
		{engine.StatusFailed, "failed"},
		{engine.StatusSkipped, "skipped"},
		{engine.StepStatus(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Fatalf("StepStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}
