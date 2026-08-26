package capabilities_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/internal/capabilities"
)

func TestGoBuilder_Build_NilClient(t *testing.T) {
	builder := &capabilities.GoBuilder{}

	out, err := builder.Build(context.Background(), nil)

	if err == nil {
		t.Fatal("GoBuilder.Build() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("GoBuilder.Build() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("GoBuilder.Build() directory = %v, want nil on error", out)
	}
}
