//go:build integration

// Package rust_test — real-Dagger wave-concurrency validation. Every test
// here exists to answer one question the unit-level concurrency suite
// (internal/workflow/engine/concurrency_test.go, fakes only + go test
// -race) cannot: do Shipwright's own real providers, sharing one real
// Dagger session/client, actually behave correctly when the engine
// dispatches them concurrently — especially rust-command + a real
// Docker-in-Docker service. All instrumentation (the timeline type) lives
// in this test file only; no production logging was added for it.
package rust_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/workflow/engine"
	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
	"github.com/pablogore/shipwright/internal/workflow/providers"
	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/rust"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
)

// timeline records each step's real [start, end) wall-clock interval under
// a mutex, so these tests can present concrete temporal evidence of
// overlap — not just "the test passed." Duplicated from
// internal/workflow/engine/concurrency_test.go's own timeline rather than
// shared, since these are two independently buildable trees (engine's is
// plain `go test`, this one is `-tags integration`).
type timeline struct {
	mu        sync.Mutex
	intervals map[string][2]time.Time
	outputs   map[string]string
}

func newTimeline() *timeline {
	return &timeline{intervals: make(map[string][2]time.Time), outputs: make(map[string]string)}
}

// setOutput/output capture a step's real stdout text, read directly off the
// *dagger.File a Tester returns. The engine itself never surfaces this into
// StepOutcome.Output — Result treats a Tester's *dagger.File as a
// non-serializable handle (execute.go's outcomeOutput, outputFile case) —
// so these tests read it here instead, at the one place they already wrap
// every real Tester for timing.
func (tl *timeline) setOutput(id, text string) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.outputs[id] = text
}

func (tl *timeline) output(id string) string {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.outputs[id]
}

func (tl *timeline) track(id string) func() {
	start := time.Now()
	return func() {
		end := time.Now()
		tl.mu.Lock()
		defer tl.mu.Unlock()
		tl.intervals[id] = [2]time.Time{start, end}
	}
}

func (tl *timeline) get(id string) (start, end time.Time) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	iv := tl.intervals[id]
	return iv[0], iv[1]
}

func (tl *timeline) overlaps(a, b string) bool {
	aStart, aEnd := tl.get(a)
	bStart, bEnd := tl.get(b)
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

func (tl *timeline) logAll(t *testing.T) {
	t.Helper()
	tl.mu.Lock()
	defer tl.mu.Unlock()
	for id, iv := range tl.intervals {
		t.Logf("timeline: %-16s start=%s end=%s duration=%s", id, iv[0].Format(time.RFC3339Nano), iv[1].Format(time.RFC3339Nano), iv[1].Sub(iv[0]))
	}
}

// instrumentedTester wraps a real shipwright.Tester, recording its
// [start,end) interval in tl under id — test-only instrumentation around a
// real production capability, never inside it.
type instrumentedTester struct {
	tl    *timeline
	id    string
	inner shipwright.Tester
}

func (i instrumentedTester) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	done := i.tl.track(i.id)
	defer done()
	f, err := i.inner.Test(ctx, source)
	if err == nil && f != nil {
		if text, cerr := f.Contents(ctx); cerr == nil {
			i.tl.setOutput(i.id, text)
		}
	}
	return f, err
}

// writeConcurrencyFixtureCrate writes a minimal real Cargo crate to a fresh
// t.TempDir(): a trivial lib.rs (rust-command "build"/"test" and clippy all
// have something real to compile and lint) plus two dependency-free
// binaries used only by the DinD validation (Case C) — docker_check.rs
// speaks raw HTTP/1.1 to the Docker Engine API's GET /info over
// DOCKER_HOST and prints the daemon's own "ID" field, and env_check.rs
// reports whether DOCKER_HOST leaked into a step that never set Docker:
// true. Both use only std (no crates.io dependency) so the fixture builds
// fast and never depends on network access to a registry beyond crates.io
// itself for cargo's own index.
func writeConcurrencyFixtureCrate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	mustWrite := func(rel, contents string) {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
			t.Fatalf("write %q: %v", full, err)
		}
	}

	mustWrite("Cargo.toml", `[package]
name = "concurrencyfixture"
version = "0.1.0"
edition = "2021"
`)
	mustWrite("src/lib.rs", `pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_add() {
        assert_eq!(add(2, 3), 5);
    }
}
`)
	mustWrite("src/bin/docker_check.rs", `use std::io::{Read, Write};
use std::net::TcpStream;
use std::time::Duration;

// Optional argv[1]: seconds to sleep between two daemon checks, so a step
// can prove its own daemon is still alive and reachable well after a
// sibling step (running concurrently, without the delay) has already
// finished — see the lifecycle-independence test.
fn main() {
    check();
    if let Some(secs) = std::env::args().nth(1).and_then(|s| s.parse::<u64>().ok()) {
        std::thread::sleep(Duration::from_secs(secs));
        check();
    }
}

fn check() {
    let host = std::env::var("DOCKER_HOST").unwrap_or_default();
    let addr = host.trim_start_matches("tcp://");
    let mut stream = TcpStream::connect(addr).expect("connect to DOCKER_HOST");
    stream
        .write_all(b"GET /info HTTP/1.1\r\nHost: docker\r\nConnection: close\r\n\r\n")
        .expect("write request");
    let mut resp = String::new();
    stream.read_to_string(&mut resp).expect("read response");
    if !resp.starts_with("HTTP/1.1 200") {
        eprintln!("unexpected response: {}", resp);
        std::process::exit(1);
    }
    let id = extract(&resp, "\"ID\":\"").unwrap_or_else(|| "unknown".to_string());
    println!("DAEMON_ID={}", id);
}

fn extract(body: &str, marker: &str) -> Option<String> {
    let start = body.find(marker)? + marker.len();
    let end = body[start..].find('"')? + start;
    Some(body[start..end].to_string())
}
`)
	mustWrite("src/bin/env_check.rs", `fn main() {
    match std::env::var("DOCKER_HOST") {
        Ok(v) => println!("DOCKER_HOST={}", v),
        Err(_) => println!("DOCKER_HOST=<unset>"),
    }
}
`)
	// container_ops speaks raw HTTP/1.1 to the Docker Engine API to prove
	// FUNCTIONAL container isolation between two daemons — not just a
	// different daemon ID. "create <marker>" pulls a tiny image and starts a
	// uniquely-named container; "list" dumps every container the daemon
	// knows about (docker ps -a's own equivalent, /containers/json?all=true)
	// so the test can assert a sibling daemon's marker name is absent.
	mustWrite("src/bin/container_ops.rs", `use std::io::{Read, Write};
use std::net::TcpStream;

fn connect() -> TcpStream {
    let host = std::env::var("DOCKER_HOST").unwrap_or_default();
    let addr = host.trim_start_matches("tcp://");
    TcpStream::connect(addr).expect("connect to DOCKER_HOST")
}

fn request(req: &str) -> String {
    let mut stream = connect();
    stream.write_all(req.as_bytes()).expect("write request");
    let mut resp = String::new();
    stream.read_to_string(&mut resp).expect("read response");
    resp
}

fn create(marker: &str) {
    let pull = request(
        "POST /images/create?fromImage=alpine&tag=latest HTTP/1.1\r\nHost: docker\r\nConnection: close\r\n\r\n",
    );
    if pull.contains("\"error\"") {
        eprintln!("image pull failed: {}", pull);
        std::process::exit(1);
    }

    let body = "{\"Image\":\"alpine:latest\",\"Cmd\":[\"sleep\",\"3600\"]}";
    let create_req = format!(
        "POST /containers/create?name={} HTTP/1.1\r\nHost: docker\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
        marker,
        body.len(),
        body
    );
    let resp = request(&create_req);
    if !resp.starts_with("HTTP/1.1 201") {
        eprintln!("container create failed: {}", resp);
        std::process::exit(1);
    }
    println!("CREATED={}", marker);
}

fn list() {
    let resp = request("GET /containers/json?all=true HTTP/1.1\r\nHost: docker\r\nConnection: close\r\n\r\n");
    if !resp.starts_with("HTTP/1.1 200") {
        eprintln!("container list failed: {}", resp);
        std::process::exit(1);
    }
    println!("CONTAINERS={}", resp);
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    match args.get(1).map(String::as_str) {
        Some("create") => create(args.get(2).expect("marker name required")),
        Some("list") => list(),
        Some("create-and-list") => {
            // Both against the SAME daemon within this one process — unlike
            // splitting create/list across two steps, which would (by
            // design) land on two different isolated daemons.
            create(args.get(2).expect("marker name required"));
            list();
        }
        _ => {
            eprintln!("usage: container_ops <create <marker>|list|create-and-list <marker>>");
            std::process::exit(1);
        }
    }
}
`)
	return dir
}

// runWaveConfig builds an engine.Config for two independent same-wave
// steps (no needs[] edge), shared across every test in this file.
func runWaveConfig(steps []manifest.Step, reg *providers.Registry, source *dagger.Directory, maxParallel int) (engine.Config, error) {
	g, err := graph.Build(steps)
	if err != nil {
		return engine.Config{}, err
	}
	return engine.Config{
		Steps:    steps,
		Graph:    g,
		Registry: reg,
		Source:   source,
		Options:  engine.Options{MaxParallel: maxParallel},
	}, nil
}

// TestRealDaggerConcurrency_CaseA_RustCommandAndClippyOverlap is Case A:
// two independent, real Rust/Dagger provider steps in one wave — the
// general rust-command primitive running `cargo build`, and clippy — both
// declared in examples/workflow/rust-concurrency-validation.yaml with
// execution.concurrency.maxParallel: 2. Proves genuine wall-clock overlap
// via timeline, not just a passing assertion.
func TestRealDaggerConcurrency_CaseA_RustCommandAndClippyOverlap(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("dagger.Connect() error = %v", err)
	}
	defer client.Close()

	m, err := manifest.ParseFile("../../../examples/workflow/rust-concurrency-validation.yaml")
	if err != nil {
		t.Fatalf("manifest.ParseFile() error = %v", err)
	}
	if got := m.Spec.Execution.Concurrency.MaxParallel; got != 2 {
		t.Fatalf("fixture manifest maxParallel = %d, want 2", got)
	}

	fixtureDir := writeConcurrencyFixtureCrate(t)
	source := client.Host().Directory(fixtureDir)
	rustClient := daggerkit.NewDaggerAdapter(client)
	tl := newTimeline()

	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "rust-command", Version: "1"}, providers.WithSchema{
		"command": interp.KindString,
	}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "cargoBuild", inner: &rust.RustCommand{Client: rustClient, Command: "build", CacheKey: "case-a"}}
	})
	reg.RegisterTester(providers.Ref{Name: "clippy", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "lint", inner: &rust.RustLinter{Client: rustClient}}
	})

	cfg, err := runWaveConfig(m.Spec.Steps, reg, source, m.Spec.Execution.Concurrency.MaxParallel)
	if err != nil {
		t.Fatalf("runWaveConfig() error = %v", err)
	}

	start := time.Now()
	res, err := engine.Execute(ctx, cfg)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("engine.Execute() error = %v, want nil", err)
	}
	if res.Failed() {
		t.Fatalf("engine.Execute() Failures = %v, want none", res.Failures)
	}
	t.Logf("Case A total wall-clock: %s", elapsed)
	tl.logAll(t)

	if !tl.overlaps("cargoBuild", "lint") {
		t.Fatal("real \"cargoBuild\" (rust-command) and \"lint\" (clippy) never overlapped — want genuine concurrent execution against real Dagger")
	}
}

// TestRealDaggerConcurrency_CaseB_TwoRustCommandStepsSameClient is Case B:
// two independent rust-command steps against the SAME real source AND the
// SAME default cache volume (the harder case rustcommand.go's own doc
// comment warns about: distinct CacheKeys are needed for DIFFERENT
// workspaces, but this deliberately targets the SAME workspace to stress
// the shared `target` cache volume under real concurrent writers). Proves:
// one *dagger.Client/session throughout, no panic, no GraphQL/session
// error, correct exit codes, both results usable.
func TestRealDaggerConcurrency_CaseB_TwoRustCommandStepsSameClient(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("dagger.Connect() error = %v", err)
	}
	defer client.Close()

	fixtureDir := writeConcurrencyFixtureCrate(t)
	source := client.Host().Directory(fixtureDir)
	rustClient := daggerkit.NewDaggerAdapter(client)
	tl := newTimeline()

	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "cmd-build", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "cmdBuild", inner: &rust.RustCommand{Client: rustClient, Command: "build"}}
	})
	reg.RegisterTester(providers.Ref{Name: "cmd-test", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "cmdTest", inner: &rust.RustCommand{Client: rustClient, Command: "test"}}
	})

	steps := []manifest.Step{
		{ID: "cmdBuild", Capability: "test", Uses: manifest.UsesSpec{Provider: "cmd-build", Version: "1"}},
		{ID: "cmdTest", Capability: "test", Uses: manifest.UsesSpec{Provider: "cmd-test", Version: "1"}},
	}
	cfg, err := runWaveConfig(steps, reg, source, 2)
	if err != nil {
		t.Fatalf("runWaveConfig() error = %v", err)
	}

	res, err := engine.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("engine.Execute() error = %v, want nil (both concurrent rust-command steps must succeed against the shared cache volume)", err)
	}
	if res.Failed() {
		t.Fatalf("engine.Execute() Failures = %v, want none", res.Failures)
	}
	tl.logAll(t)

	for _, o := range res.Outcomes {
		if o.Status != engine.StatusSucceeded {
			t.Fatalf("Outcomes[%q].Status = %v, want StatusSucceeded", o.StepID, o.Status)
		}
	}
}

// TestRealDaggerConcurrency_CaseC_DinDDoesNotLeakOrCollide is Case C: one
// rust-command step with Docker: true (a real docker:27-dind Dagger
// Service) running concurrently with another rust-command step that has NO
// Docker daemon attached, in the same wave under maxParallel: 2. Proves:
// the DinD service starts and is reachable (docker_check's real HTTP call
// to GET /info succeeds), and DOCKER_HOST does not leak into the sibling
// step that never opted in (env_check reports it unset).
func TestRealDaggerConcurrency_CaseC_DinDDoesNotLeakOrCollide(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("dagger.Connect() error = %v", err)
	}
	defer client.Close()

	fixtureDir := writeConcurrencyFixtureCrate(t)
	source := client.Host().Directory(fixtureDir)
	rustClient := daggerkit.NewDaggerAdapter(client)
	tl := newTimeline()

	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "docker-check", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "dockerCheck", inner: &rust.RustCommand{Client: rustClient, Command: "run --bin docker_check", Docker: true, CacheKey: "case-c-docker"}}
	})
	reg.RegisterTester(providers.Ref{Name: "env-check", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "envCheck", inner: &rust.RustCommand{Client: rustClient, Command: "run --bin env_check", CacheKey: "case-c-noDocker"}}
	})

	steps := []manifest.Step{
		{ID: "dockerCheck", Capability: "test", Uses: manifest.UsesSpec{Provider: "docker-check", Version: "1"}},
		{ID: "envCheck", Capability: "test", Uses: manifest.UsesSpec{Provider: "env-check", Version: "1"}},
	}
	cfg, err := runWaveConfig(steps, reg, source, 2)
	if err != nil {
		t.Fatalf("runWaveConfig() error = %v", err)
	}

	res, err := engine.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("engine.Execute() error = %v, want nil (DinD service must start and the sibling step must be unaffected)", err)
	}
	if res.Failed() {
		t.Fatalf("engine.Execute() Failures = %v, want none", res.Failures)
	}
	tl.logAll(t)
	if !tl.overlaps("dockerCheck", "envCheck") {
		t.Fatal("\"dockerCheck\" (Docker: true) and \"envCheck\" (no Docker) never overlapped — want genuine concurrent execution")
	}

	dockerOut := tl.output("dockerCheck")
	if !strings.Contains(dockerOut, "DAEMON_ID=") {
		t.Fatalf("dockerCheck output = %q, want it to contain a DAEMON_ID (real DinD daemon reachable via DOCKER_HOST)", dockerOut)
	}
	envOut := tl.output("envCheck")
	if !strings.Contains(envOut, "DOCKER_HOST=<unset>") {
		t.Fatalf("envCheck output = %q, want DOCKER_HOST=<unset> — a sibling with Docker: true must never leak DOCKER_HOST into a step that did not request it", envOut)
	}
}

// TestRealDaggerConcurrency_CaseD_TwoConcurrentDinDDaemonsAreIsolated is
// Case D (formerly Case C's "bonus" case, now the fixed and required
// scenario): TWO rust-command steps, both Docker: true, running
// concurrently in the same wave under maxParallel: 2. Each daemon's own
// Engine API GET /info must report a distinct "ID" — proof that
// withDockerDaemon (providers/rust/dockerdaemon.go) no longer hands two
// concurrent steps the same content-addressed DinD service, now that each
// invocation's DinD container graph is differentiated by the dispatching
// step's own id (via pkg/shipwright/invocation, set by the engine's
// dispatch()). Before the fix this failed deterministically: both steps
// reported the identical daemon ID (observed: both
// 1f120812-7a95-43d1-9424-f5fcb610bf46).
func TestRealDaggerConcurrency_CaseD_TwoConcurrentDinDDaemonsAreIsolated(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("dagger.Connect() error = %v", err)
	}
	defer client.Close()

	fixtureDir := writeConcurrencyFixtureCrate(t)
	source := client.Host().Directory(fixtureDir)
	rustClient := daggerkit.NewDaggerAdapter(client)
	tl := newTimeline()

	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "docker-check-1", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "dockerCheck1", inner: &rust.RustCommand{Client: rustClient, Command: "run --bin docker_check", Docker: true, CacheKey: "case-c-dual-1"}}
	})
	reg.RegisterTester(providers.Ref{Name: "docker-check-2", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "dockerCheck2", inner: &rust.RustCommand{Client: rustClient, Command: "run --bin docker_check", Docker: true, CacheKey: "case-c-dual-2"}}
	})

	steps := []manifest.Step{
		{ID: "dockerCheck1", Capability: "test", Uses: manifest.UsesSpec{Provider: "docker-check-1", Version: "1"}},
		{ID: "dockerCheck2", Capability: "test", Uses: manifest.UsesSpec{Provider: "docker-check-2", Version: "1"}},
	}
	cfg, err := runWaveConfig(steps, reg, source, 2)
	if err != nil {
		t.Fatalf("runWaveConfig() error = %v", err)
	}

	res, err := engine.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("engine.Execute() error = %v, want nil (two independent DinD daemons must both start and stay isolated)", err)
	}
	if res.Failed() {
		t.Fatalf("engine.Execute() Failures = %v, want none", res.Failures)
	}
	tl.logAll(t)
	if !tl.overlaps("dockerCheck1", "dockerCheck2") {
		t.Fatal("the two Docker: true steps never overlapped — want genuine concurrent DinD execution")
	}

	id1 := daemonID(t, tl.output("dockerCheck1"))
	id2 := daemonID(t, tl.output("dockerCheck2"))
	t.Logf("daemon 1 ID = %s, daemon 2 ID = %s", id1, id2)
	if id1 == id2 {
		t.Fatalf("both concurrent DinD steps reported the same daemon ID %q, want two distinct isolated daemons", id1)
	}
}

func daemonID(t *testing.T, output string) string {
	t.Helper()
	const marker = "DAEMON_ID="
	i := strings.Index(output, marker)
	if i < 0 {
		t.Fatalf("Output = %q, want it to contain %q", output, marker)
	}
	id := output[i+len(marker):]
	if nl := strings.IndexAny(id, "\r\n"); nl >= 0 {
		id = id[:nl]
	}
	return strings.TrimSpace(id)
}

// TestRealDaggerConcurrency_CaseD_ContainerIsolationBetweenConcurrentDaemons
// goes one step further than distinct daemon IDs: it proves FUNCTIONAL
// isolation. Step A creates a uniquely-named container on its own DinD
// daemon; step B, running concurrently with its own Docker: true daemon,
// must not see that container via its own "docker ps -a" equivalent
// (/containers/json?all=true) — and vice versa. Two daemons that merely
// reported different IDs but secretly shared storage/state would fail
// this.
func TestRealDaggerConcurrency_CaseD_ContainerIsolationBetweenConcurrentDaemons(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("dagger.Connect() error = %v", err)
	}
	defer client.Close()

	fixtureDir := writeConcurrencyFixtureCrate(t)
	source := client.Host().Directory(fixtureDir)
	rustClient := daggerkit.NewDaggerAdapter(client)
	tl := newTimeline()

	const markerA = "shipwright-isolation-marker-a"
	const markerB = "shipwright-isolation-marker-b"

	// create-and-list runs both operations in one process against one
	// daemon: daemon identity is scoped to the step (invocation.StepID), so
	// splitting create and list into two separate steps would — correctly,
	// by design — hand them two different isolated daemons and prove
	// nothing about isolation. A single step's single command is the only
	// way to inspect "this step's own daemon" after seeding it.
	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "container-ops-a", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "containerOpsA", inner: &rust.RustCommand{Client: rustClient, Command: "run --bin container_ops -- create-and-list " + markerA, Docker: true, CacheKey: "case-d-isolation-a"}}
	})
	reg.RegisterTester(providers.Ref{Name: "container-ops-b", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "containerOpsB", inner: &rust.RustCommand{Client: rustClient, Command: "run --bin container_ops -- create-and-list " + markerB, Docker: true, CacheKey: "case-d-isolation-b"}}
	})

	steps := []manifest.Step{
		{ID: "containerOpsA", Capability: "test", Uses: manifest.UsesSpec{Provider: "container-ops-a", Version: "1"}},
		{ID: "containerOpsB", Capability: "test", Uses: manifest.UsesSpec{Provider: "container-ops-b", Version: "1"}},
	}
	cfg, err := runWaveConfig(steps, reg, source, 2)
	if err != nil {
		t.Fatalf("runWaveConfig() error = %v", err)
	}

	res, err := engine.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("engine.Execute() error = %v, want nil (both concurrent container-creating steps must succeed on their own isolated daemons)", err)
	}
	if res.Failed() {
		t.Fatalf("engine.Execute() Failures = %v, want none", res.Failures)
	}
	tl.logAll(t)
	if !tl.overlaps("containerOpsA", "containerOpsB") {
		t.Fatal("the two container-creating Docker: true steps never overlapped — want genuine concurrent execution")
	}

	outA := tl.output("containerOpsA")
	outB := tl.output("containerOpsB")
	if !strings.Contains(outA, "CREATED="+markerA) {
		t.Fatalf("containerOpsA output = %q, want it to confirm creating %q", outA, markerA)
	}
	if !strings.Contains(outB, "CREATED="+markerB) {
		t.Fatalf("containerOpsB output = %q, want it to confirm creating %q", outB, markerB)
	}

	// Each step's own listing, taken from its own daemon right after
	// creating its own marker: A's daemon must never see B's marker, and
	// B's daemon must never see A's.
	if !strings.Contains(outA, markerA) {
		t.Fatalf("daemon A's own container listing = %q, want it to see its own marker %q", outA, markerA)
	}
	if strings.Contains(outA, markerB) {
		t.Fatalf("daemon A's container listing = %q, must NOT see daemon B's marker %q — daemons are sharing container state", outA, markerB)
	}
	if !strings.Contains(outB, markerB) {
		t.Fatalf("daemon B's own container listing = %q, want it to see its own marker %q", outB, markerB)
	}
	if strings.Contains(outB, markerA) {
		t.Fatalf("daemon B's container listing = %q, must NOT see daemon A's marker %q — daemons are sharing container state", outB, markerA)
	}
}

// TestRealDaggerConcurrency_CaseD_LifecycleIndependence proves that one
// Docker: true step finishing does not tear down or otherwise disturb a
// concurrently running sibling's own daemon. Step A ("fast") runs a single
// daemon check and returns almost immediately; step B ("slow") checks its
// daemon, sleeps past the point where A has certainly already finished, and
// checks its daemon again — that second check must still succeed.
func TestRealDaggerConcurrency_CaseD_LifecycleIndependence(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("dagger.Connect() error = %v", err)
	}
	defer client.Close()

	fixtureDir := writeConcurrencyFixtureCrate(t)
	source := client.Host().Directory(fixtureDir)
	rustClient := daggerkit.NewDaggerAdapter(client)
	tl := newTimeline()

	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "docker-fast", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "dockerFast", inner: &rust.RustCommand{Client: rustClient, Command: "run --bin docker_check", Docker: true, CacheKey: "case-d-lifecycle-fast"}}
	})
	reg.RegisterTester(providers.Ref{Name: "docker-slow", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "dockerSlow", inner: &rust.RustCommand{Client: rustClient, Command: "run --bin docker_check -- 8", Docker: true, CacheKey: "case-d-lifecycle-slow"}}
	})

	steps := []manifest.Step{
		{ID: "dockerFast", Capability: "test", Uses: manifest.UsesSpec{Provider: "docker-fast", Version: "1"}},
		{ID: "dockerSlow", Capability: "test", Uses: manifest.UsesSpec{Provider: "docker-slow", Version: "1"}},
	}
	cfg, err := runWaveConfig(steps, reg, source, 2)
	if err != nil {
		t.Fatalf("runWaveConfig() error = %v", err)
	}

	res, err := engine.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("engine.Execute() error = %v, want nil (the slow step's daemon must survive the fast step finishing first)", err)
	}
	if res.Failed() {
		t.Fatalf("engine.Execute() Failures = %v, want none", res.Failures)
	}
	tl.logAll(t)

	_, fastEnd := tl.get("dockerFast")
	_, slowEnd := tl.get("dockerSlow")
	if !fastEnd.Before(slowEnd) {
		t.Fatalf("dockerFast end (%s) is not before dockerSlow end (%s) — test setup didn't actually stagger completion, so lifecycle independence wasn't exercised", fastEnd.Format(time.RFC3339Nano), slowEnd.Format(time.RFC3339Nano))
	}
	t.Logf("dockerFast finished at %s, dockerSlow finished at %s (%s later)", fastEnd.Format(time.RFC3339Nano), slowEnd.Format(time.RFC3339Nano), slowEnd.Sub(fastEnd))

	slowOut := tl.output("dockerSlow")
	if got := strings.Count(slowOut, "DAEMON_ID="); got != 2 {
		t.Fatalf("dockerSlow output = %q, want exactly 2 DAEMON_ID checks (one before, one after the fast sibling finished), got %d", slowOut, got)
	}
}

// TestRealDaggerConcurrency_StressTenConsecutiveRuns runs the scenario that
// used to fail deterministically before the DinD isolation fix — two
// concurrent Docker:true rust-command steps in the same wave — 10
// consecutive times, each against a fresh Dagger session. Looks
// specifically for what a single passing run and go test -race cannot
// catch: flaky content-hash collisions, service-lifecycle problems, or
// intermittent Dagger client/session errors. Every iteration must report
// two distinct daemon IDs.
func TestRealDaggerConcurrency_StressTenConsecutiveRuns(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 10-iteration real-Dagger stress run in -short mode")
	}
	const iterations = 10
	for i := 0; i < iterations; i++ {
		func() {
			ctx := context.Background()
			client, err := dagger.Connect(ctx)
			if err != nil {
				t.Fatalf("iteration %d: dagger.Connect() error = %v", i, err)
			}
			defer client.Close()

			fixtureDir := writeConcurrencyFixtureCrate(t)
			source := client.Host().Directory(fixtureDir)
			rustClient := daggerkit.NewDaggerAdapter(client)
			tl := newTimeline()

			reg := providers.NewRegistry()
			reg.RegisterTester(providers.Ref{Name: "docker-check-1", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
				return instrumentedTester{tl: tl, id: "dockerCheck1", inner: &rust.RustCommand{Client: rustClient, Command: "run --bin docker_check", Docker: true, CacheKey: "stress-docker-1"}}
			})
			reg.RegisterTester(providers.Ref{Name: "docker-check-2", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
				return instrumentedTester{tl: tl, id: "dockerCheck2", inner: &rust.RustCommand{Client: rustClient, Command: "run --bin docker_check", Docker: true, CacheKey: "stress-docker-2"}}
			})

			steps := []manifest.Step{
				{ID: "dockerCheck1", Capability: "test", Uses: manifest.UsesSpec{Provider: "docker-check-1", Version: "1"}},
				{ID: "dockerCheck2", Capability: "test", Uses: manifest.UsesSpec{Provider: "docker-check-2", Version: "1"}},
			}
			cfg, err := runWaveConfig(steps, reg, source, 2)
			if err != nil {
				t.Fatalf("iteration %d: runWaveConfig() error = %v", i, err)
			}

			res, err := engine.Execute(ctx, cfg)
			if err != nil {
				var stepFailed *engine.StepFailedError
				if errors.As(err, &stepFailed) {
					t.Fatalf("iteration %d: step %q failed: %v", i, stepFailed.StepID, stepFailed.Err)
				}
				t.Fatalf("iteration %d: engine.Execute() error = %v, want nil", i, err)
			}
			if res.Failed() {
				t.Fatalf("iteration %d: Failures = %v, want none", i, res.Failures)
			}

			id1 := daemonID(t, tl.output("dockerCheck1"))
			id2 := daemonID(t, tl.output("dockerCheck2"))
			if id1 == id2 {
				t.Fatalf("iteration %d: both concurrent DinD steps reported the same daemon ID %q, want two distinct isolated daemons", i, id1)
			}
			t.Logf("iteration %d: OK (overlap=%v, daemon1=%s, daemon2=%s)", i, tl.overlaps("dockerCheck1", "dockerCheck2"), id1, id2)
		}()
	}
}
