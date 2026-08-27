package containerutil

import "testing"

// TestComputeEntrypoint locks the shared behavior providers/go and
// providers/rust both relied on before their local computeEntrypoint copies
// were deduplicated into this package (PR #176 review finding #10): an
// empty binaryName falls back to "app", any other value changes the
// entrypoint path.
func TestComputeEntrypoint(t *testing.T) {
	tests := []struct {
		name       string
		binaryName string
		want       string
	}{
		{name: "empty falls back to default", binaryName: "", want: "/app/app"},
		{name: "explicit binary name changes the entrypoint", binaryName: "my-service", want: "/app/my-service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeEntrypoint(tt.binaryName)
			if got != tt.want {
				t.Fatalf("ComputeEntrypoint(%q) = %q, want %q", tt.binaryName, got, tt.want)
			}
		})
	}
}

// TestRegistryHost locks the shared behavior providers/go and providers/rust
// both relied on before their local registryHost copies were deduplicated
// into this package (PR #176 review finding #10).
func TestRegistryHost(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{name: "dotted registry host", ref: "ghcr.io/acme/api:v1", want: "ghcr.io"},
		{name: "host with port", ref: "localhost:5000/acme/api:v1", want: "localhost:5000"},
		{name: "bare docker hub name has no registry segment", ref: "acme/api:v1", want: "acme/api:v1"},
		{name: "no slash at all", ref: "api:v1", want: "api:v1"},
		{name: "localhost without port", ref: "localhost/acme/api:v1", want: "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RegistryHost(tt.ref)
			if got != tt.want {
				t.Fatalf("RegistryHost(%q) = %q, want %q", tt.ref, got, tt.want)
			}
		})
	}
}
