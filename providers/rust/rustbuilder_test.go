package rust_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/providers/rust"
)

func TestRustBuilder_Build_NilClient(t *testing.T) {
	builder := &rust.RustBuilder{}

	out, err := builder.Build(context.Background(), nil)

	if err == nil {
		t.Fatal("RustBuilder.Build() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("RustBuilder.Build() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("RustBuilder.Build() directory = %v, want nil on error", out)
	}
}
