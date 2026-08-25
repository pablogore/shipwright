package capabilities_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/internal/capabilities"
)

func TestGoLinter_Test_NilClient(t *testing.T) {
	linter := &capabilities.GoLinter{}

	out, err := linter.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("GoLinter.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("GoLinter.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("GoLinter.Test() file = %v, want nil on error", out)
	}
}
