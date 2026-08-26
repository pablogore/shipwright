package golang_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/providers/go"
)

func TestGoUnitTester_Test_NilClient(t *testing.T) {
	tester := &golang.GoUnitTester{}

	out, err := tester.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("GoUnitTester.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("GoUnitTester.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("GoUnitTester.Test() file = %v, want nil on error", out)
	}
}
