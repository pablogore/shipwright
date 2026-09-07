package golang

import "github.com/pablogore/shipwright/providers/go/daggerkit"

// goModCacheMountPath/goBuildCacheMountPath are the official golang:<version>
// and golangci/golangci-lint:<version> images' GOMODCACHE/GOCACHE
// directories (confirmed empirically against both images: GOPATH=/go so
// GOMODCACHE=/go/pkg/mod, and HOME=/root so GOCACHE=/root/.cache/go-build —
// neither image sets GOMODCACHE/GOCACHE directly, both are Go's own
// defaults derived from GOPATH/HOME).
const (
	goModCacheMountPath   = "/go/pkg/mod"
	goBuildCacheMountPath = "/root/.cache/go-build"
)

// goModCacheKey/goBuildCacheKey name one shared pair of cache volumes for
// every capability in this package that invokes the go tool (GoBuilder,
// GoUnitTester, GoLinter, GoVulnScanner) — deliberately NOT split per
// capability the way providers/rust's target-dir caches are (cargocache.go):
// Go's build cache is content-addressed by action ID (toolchain identity,
// GOOS/GOARCH, CGO_ENABLED, build flags all feed the hash), so differing
// settings across these four providers — GoUnitTester's CGO_ENABLED=1 vs the
// others' CGO_ENABLED=0, or GoLinter's golangci-lint image shipping a
// different Go patch version than the golang:<version> image the other
// three use — cannot corrupt or "ping-pong" a shared volume, only add
// side-by-side entries that simply don't cross-reuse. GOMODCACHE is fully
// toolchain-independent (downloaded module content addressed by go.sum
// hash), so sharing it is pure upside regardless of Go version. See
// docs findings from the caches/recompilation audit this closes.
const (
	goModCacheKey   = "shipwright-go-mod-cache"
	goBuildCacheKey = "shipwright-go-build-cache"
)

// mountGoCaches mounts the shared module and build cache volumes onto
// container, using client to resolve them. Never mounts source (/app, /src)
// — those stay per-invocation WithMountedDirectory calls in each provider.
func mountGoCaches(client daggerkit.DaggerClient, container daggerkit.DaggerContainer) daggerkit.DaggerContainer {
	return container.
		WithMountedCache(goModCacheMountPath, client.CacheVolume(goModCacheKey)).
		WithMountedCache(goBuildCacheMountPath, client.CacheVolume(goBuildCacheKey))
}
