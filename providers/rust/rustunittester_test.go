package rust_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/providers/rust"
)

func TestRustUnitTester_Test_NilClient(t *testing.T) {
	tester := &rust.RustUnitTester{}

	out, err := tester.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("RustUnitTester.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("RustUnitTester.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("RustUnitTester.Test() file = %v, want nil on error", out)
	}
}
