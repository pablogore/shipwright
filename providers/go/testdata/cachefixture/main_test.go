package main

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewStringIsValidUUID(t *testing.T) {
	id := uuid.NewString()
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("uuid.Parse(%q) error = %v, want nil", id, err)
	}
}
