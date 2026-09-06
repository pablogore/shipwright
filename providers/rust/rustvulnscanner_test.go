package rust_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/providers/rust"
)

func TestRustVulnScanner_Test_NilClient(t *testing.T) {
	scanner := &rust.RustVulnScanner{}

	out, err := scanner.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("RustVulnScanner.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("RustVulnScanner.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("RustVulnScanner.Test() file = %v, want nil on error", out)
	}
}
