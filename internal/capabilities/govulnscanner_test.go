package capabilities_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/internal/capabilities"
)

func TestGoVulnScanner_Test_NilClient(t *testing.T) {
	scanner := &capabilities.GoVulnScanner{}

	out, err := scanner.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("GoVulnScanner.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("GoVulnScanner.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("GoVulnScanner.Test() file = %v, want nil on error", out)
	}
}
