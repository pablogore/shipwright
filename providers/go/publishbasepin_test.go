package golang

import "testing"

// TestDefaultPublishBaseImageIsDigestPinned guards the production-
// readiness P0 fix: ContainerPublisher's runtime base image must never
// depend on a mutable tag such as "alpine:latest" — the same Shipwright
// revision must always publish against the same base image
// (containerpublisher.go:defaultPublishBaseImage is the single source of
// truth for the pin).
func TestDefaultPublishBaseImageIsDigestPinned(t *testing.T) {
	if defaultPublishBaseImage == "alpine:latest" {
		t.Fatal("defaultPublishBaseImage = \"alpine:latest\" — the publisher base image must not use the mutable \"latest\" tag")
	}

	if !validatePinnedImageReference(defaultPublishBaseImage) {
		t.Fatalf("defaultPublishBaseImage = %q is not a digest-pinned image reference (want repo:tag@sha256:<64-hex-digest>, no \"latest\" tag)", defaultPublishBaseImage)
	}
}

func TestValidatePinnedImageReference(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{name: "pinned alpine reference", ref: "alpine:3.22@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce", want: true},
		{name: "pinned reference without a human tag", ref: "alpine@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce", want: true},
		{name: "bare latest tag, no digest", ref: "alpine:latest", want: false},
		{name: "latest tag alongside a digest is still rejected", ref: "alpine:latest@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce", want: false},
		{name: "exact tag with no digest", ref: "alpine:3.22", want: false},
		{name: "digest too short", ref: "alpine:3.22@sha256:1234", want: false},
		{name: "digest with non-hex characters", ref: "alpine:3.22@sha256:zz358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dc", want: false},
		{name: "unsupported digest algorithm", ref: "alpine:3.22@sha1:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce", want: false},
		{name: "empty reference", ref: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validatePinnedImageReference(tt.ref); got != tt.want {
				t.Fatalf("validatePinnedImageReference(%q) = %v, want %v", tt.ref, got, tt.want)
			}
		})
	}
}
