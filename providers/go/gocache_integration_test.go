package golang_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/providers/go"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// requireDaggerClient connects to a real Dagger engine, skipping (not
// failing) when none is reachable — mirrors the established repo-wide
// pattern (internal/pipelines/shared/dagger_helpers_test.go).
func requireDaggerClient(ctx context.Context, t *testing.T) *dagger.Client {
	t.Helper()

	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Skipf("skipping: no Dagger/Docker engine available: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	return client
}

// fixtureSource returns the shared benchmark fixture (testdata/cachefixture):
// a tiny module with one real external dependency (github.com/google/uuid),
// a buildable main package, and one passing test — enough to exercise real
// module downloads and real compilation without pulling in project-specific
// noise.
func fixtureSource(t *testing.T, client *dagger.Client) *dagger.Directory {
	t.Helper()
	return client.Host().Directory("testdata/cachefixture")
}

// runAllProviders runs GoBuilder, GoUnitTester, GoLinter and GoVulnScanner
// sequentially against src, timing each individually. Used by the cold/warm
// benchmark, where sequential (not concurrent) timing is required to
// attribute cost per capability.
func runAllProviders(ctx context.Context, t *testing.T, client *dagger.Client, src *dagger.Directory) map[string]time.Duration {
	t.Helper()

	adapter := daggerkit.NewDaggerAdapter(client)
	timings := make(map[string]time.Duration, 4)

	start := time.Now()
	if _, err := (&golang.GoBuilder{Client: adapter}).Build(ctx, src); err != nil {
		t.Fatalf("GoBuilder.Build() error = %v", err)
	}
	timings["build"] = time.Since(start)

	start = time.Now()
	if _, err := (&golang.GoUnitTester{Client: adapter}).Test(ctx, src); err != nil {
		t.Fatalf("GoUnitTester.Test() error = %v", err)
	}
	timings["test"] = time.Since(start)

	start = time.Now()
	if _, err := (&golang.GoLinter{Client: adapter}).Test(ctx, src); err != nil {
		t.Fatalf("GoLinter.Test() error = %v", err)
	}
	timings["lint"] = time.Since(start)

	start = time.Now()
	if _, err := (&golang.GoVulnScanner{Client: adapter}).Test(ctx, src); err != nil {
		t.Fatalf("GoVulnScanner.Test() error = %v", err)
	}
	timings["vuln"] = time.Since(start)

	return timings
}

// logTimings prints one line per capability plus a total, tagged with
// label, so `go test -v` output doubles as the raw benchmark data.
func logTimings(t *testing.T, label string, timings map[string]time.Duration) time.Duration {
	t.Helper()
	var total time.Duration
	for _, name := range []string{"build", "test", "lint", "vuln"} {
		d := timings[name]
		total += d
		t.Logf("[%s] %-6s %v", label, name, d)
	}
	t.Logf("[%s] %-6s %v", label, "TOTAL", total)
	return total
}

// reportDelta logs the absolute and percentage change from before to after
// for one capability.
func reportDelta(t *testing.T, name string, before, after time.Duration) {
	t.Helper()
	delta := before - after
	pct := 0.0
	if before > 0 {
		pct = float64(delta) / float64(before) * 100
	}
	t.Logf("[delta] %-6s before=%v after=%v absolute=%v change=%.1f%%", name, before, after, delta, pct)
}

// TestCacheBenchmark_ColdThenWarm is scenario (1) cold + (2) warm-same-
// session + the "new Dagger session, same machine" persistence check from
// the caching audit's mandatory report (points 9-11): "shipwright-go-mod-
// cache"/"shipwright-go-build-cache" are used here for the very first time
// on this engine, so the first pass is a genuine cold cache — no volume
// wipe or engine restart needed to prove it.
func TestCacheBenchmark_ColdThenWarm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-Dagger cache benchmark in -short mode")
	}
	ctx := context.Background()

	client := requireDaggerClient(ctx, t)
	src := fixtureSource(t, client)

	cold := runAllProviders(ctx, t, client, src)
	coldTotal := logTimings(t, "cold", cold)

	warm := runAllProviders(ctx, t, client, src)
	warmTotal := logTimings(t, "warm-same-session", warm)

	for _, name := range []string{"build", "test", "lint", "vuln"} {
		reportDelta(t, name, cold[name], warm[name])
	}
	reportDelta(t, "TOTAL", coldTotal, warmTotal)

	// New Dagger session, same engine/machine: cache volumes are
	// engine-scoped, not client-scoped, so a fresh dagger.Connect() should
	// see the same warm state without any special handling.
	client2 := requireDaggerClient(ctx, t)
	warmNewSession := runAllProviders(ctx, t, client2, fixtureSource(t, client2))
	warmNewSessionTotal := logTimings(t, "warm-new-session", warmNewSession)
	reportDelta(t, "TOTAL(new-session-vs-cold)", coldTotal, warmNewSessionTotal)
}

// runOneWave executes all four providers concurrently against the same
// client and source, mirroring how the workflow engine dispatches an
// independent wave (internal/workflow/engine's runWave) with
// execution.concurrency.maxParallel: 4. Returns the first error observed,
// or nil if all four succeeded.
func runOneWave(ctx context.Context, adapter daggerkit.DaggerClient, src *dagger.Directory) error {
	var wg sync.WaitGroup
	errs := make(chan error, 4)

	run := func(fn func() error) {
		defer wg.Done()
		errs <- fn()
	}

	wg.Add(4)
	go run(func() error {
		_, err := (&golang.GoBuilder{Client: adapter}).Build(ctx, src)
		return err
	})
	go run(func() error {
		_, err := (&golang.GoUnitTester{Client: adapter}).Test(ctx, src)
		return err
	})
	go run(func() error {
		_, err := (&golang.GoLinter{Client: adapter}).Test(ctx, src)
		return err
	})
	go run(func() error {
		_, err := (&golang.GoVulnScanner{Client: adapter}).Test(ctx, src)
		return err
	})
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// TestCacheConcurrency_SharedVolumes is scenario (3): all four providers
// sharing "shipwright-go-mod-cache"/"shipwright-go-build-cache" in one
// concurrent wave, proving no corruption/deadlock/lock error under Dagger's
// default SHARED cache-sharing mode — no global mutex anywhere in this
// path.
func TestCacheConcurrency_SharedVolumes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-Dagger cache concurrency test in -short mode")
	}
	ctx := context.Background()

	client := requireDaggerClient(ctx, t)
	src := fixtureSource(t, client)
	adapter := daggerkit.NewDaggerAdapter(client)

	if err := runOneWave(ctx, adapter, src); err != nil {
		t.Fatalf("concurrent wave (build+test+lint+vuln sharing caches) failed: %v", err)
	}
}

// TestCacheStress_TenConcurrentWaves is scenario (4): 10 waves of 4
// providers each (40 total concurrent container operations against the
// same two shared cache volumes), all launched at once — the mandate's
// "minimum 10 concurrent runs, expect 10/10 green, no flakes."
func TestCacheStress_TenConcurrentWaves(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-Dagger cache stress test in -short mode")
	}
	ctx := context.Background()

	client := requireDaggerClient(ctx, t)
	src := fixtureSource(t, client)
	adapter := daggerkit.NewDaggerAdapter(client)

	const waves = 10
	results := make([]error, waves)
	var wg sync.WaitGroup
	wg.Add(waves)
	for i := range waves {
		go func(i int) {
			defer wg.Done()
			results[i] = runOneWave(ctx, adapter, src)
		}(i)
	}
	wg.Wait()

	green := 0
	for i, err := range results {
		if err != nil {
			t.Errorf("wave %d/%d failed: %v", i+1, waves, err)
			continue
		}
		green++
	}
	t.Logf("stress result: %d/%d waves green", green, waves)
	if green != waves {
		t.Fatalf("stress result: %d/%d waves green, want %d/%d", green, waves, waves, waves)
	}
}
