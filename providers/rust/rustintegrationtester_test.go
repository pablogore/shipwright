package rust_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/providers/rust"
)

func TestRustIntegrationTester_Test_NilClient(t *testing.T) {
	tester := &rust.RustIntegrationTester{}

	out, err := tester.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("RustIntegrationTester.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("RustIntegrationTester.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("RustIntegrationTester.Test() file = %v, want nil on error", out)
	}
}
