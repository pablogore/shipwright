package rust_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/providers/rust"
)

func TestRustLinter_Test_NilClient(t *testing.T) {
	linter := &rust.RustLinter{}

	out, err := linter.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("RustLinter.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("RustLinter.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("RustLinter.Test() file = %v, want nil on error", out)
	}
}
