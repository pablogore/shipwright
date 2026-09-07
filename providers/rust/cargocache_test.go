package rust

import "testing"

// TestTargetCacheKeys_AreMutuallyDistinct guards the per-responsibility cache
// isolation this package relies on: RustBuilder, RustUnitTester, RustLinter
// and RustIntegrationTester each mount their own target-directory
// CacheVolume, precisely so that workspace-check/workspace-tests/
// workspace-lint never share one target cache and reintroduce the
// Cargo/fingerprint race a shared target directory previously caused (see
// cargocache.go's doc comment). A key collision here — e.g. a future edit
// accidentally copy-pasting rustUnitTesterTargetCacheKey's value into a new
// constant — would silently merge two capabilities' incremental-compilation
// state without any other test catching it, since each capability's own
// mock-based test only checks the CacheVolume method was called, never
// which key two different capabilities resolved to.
func TestTargetCacheKeys_AreMutuallyDistinct(t *testing.T) {
	keys := map[string]string{
		"rustBuilderTargetCacheKey":           rustBuilderTargetCacheKey,
		"rustUnitTesterTargetCacheKey":        rustUnitTesterTargetCacheKey,
		"rustLinterTargetCacheKey":            rustLinterTargetCacheKey,
		"rustIntegrationTesterTargetCacheKey": rustIntegrationTesterTargetCacheKey,
		"rustCommandDefaultTargetCacheKey":    rustCommandDefaultTargetCacheKey,
	}

	seen := make(map[string]string, len(keys))
	for name, key := range keys {
		if key == "" {
			t.Fatalf("%s is empty", name)
		}
		if other, collision := seen[key]; collision {
			t.Fatalf("%s and %s resolve to the same cache key %q — this would merge their target caches", name, other, key)
		}
		seen[key] = name
	}
}

// TestResolveCommandCacheKey_NamespacesAwayFromFixedKeys guards the boundary
// between RustCommand's dynamic, caller-chosen CacheKey and the four fixed
// capability keys above: an explicit CacheKey must never resolve to one of
// the fixed keys (which would let a RustCommand step silently share a target
// cache with RustBuilder/RustUnitTester/RustLinter/RustIntegrationTester),
// and the default (empty CacheKey) must resolve to its own dedicated key,
// also disjoint from the fixed four.
func TestResolveCommandCacheKey_NamespacesAwayFromFixedKeys(t *testing.T) {
	fixed := map[string]bool{
		rustBuilderTargetCacheKey:           true,
		rustUnitTesterTargetCacheKey:        true,
		rustLinterTargetCacheKey:            true,
		rustIntegrationTesterTargetCacheKey: true,
	}

	cases := []string{"", "xtask", "integration-tests", rustBuilderTargetCacheKey}
	for _, cacheKey := range cases {
		resolved := resolveCommandCacheKey(cacheKey)
		if fixed[resolved] {
			t.Fatalf("resolveCommandCacheKey(%q) = %q, collides with a fixed capability cache key", cacheKey, resolved)
		}
	}
}
