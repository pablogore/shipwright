// Package containerutil holds small, source-language-agnostic helpers
// shared by the container-publishing Artifactor providers (providers/go,
// providers/rust). It deliberately does not live under pkg/shipwright,
// whose public contract surface is golden-locked by schema_golden_test.go.
package containerutil

import "strings"

// defaultBinaryName is the fallback in-image binary name used whenever a
// caller's binaryName is left empty — the same default every Builder
// provider in this repo assumes when its own BuildConfig.BinaryName is
// unset.
const defaultBinaryName = "app"

// ComputeEntrypoint returns the in-image path of a published binary,
// falling back to defaultBinaryName ("app") when binaryName is empty. Pure
// helper, unit-testable without a Dagger client.
func ComputeEntrypoint(binaryName string) string {
	if binaryName == "" {
		binaryName = defaultBinaryName
	}
	return "/app/" + binaryName
}

// RegistryHost extracts the registry address portion of an image
// reference, e.g. "ghcr.io/acme/api:v1" -> "ghcr.io". A reference with no
// registry-looking first segment (a bare Docker Hub name such as
// "acme/api:v1") returns ref unchanged, which Dagger's WithRegistryAuth
// then treats as docker.io-relative. Pure helper, unit-testable without a
// Dagger client.
func RegistryHost(ref string) string {
	firstSlash := strings.Index(ref, "/")
	if firstSlash == -1 {
		return ref
	}

	host := ref[:firstSlash]
	if host != "localhost" && !strings.ContainsAny(host, ".:") {
		return ref
	}

	return host
}
