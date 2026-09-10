//go:build integration

package golang_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/workflow/engine"
	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
	"github.com/pablogore/shipwright/internal/workflow/providers"
	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// TestGoIntegrationTester_Test_RealEngine_DinDServiceReachable is
// design.md's D-12 real-engine proof for the change closing issue #278: a
// real, privileged docker:27-dind Dagger Service (withDockerDaemon,
// dockerdaemon.go) is attached to GoIntegrationTester's container, and a
// vanilla testcontainers-go-backed fixture module — with no Shipwright
// awareness of any kind, proving design.md D-9's "zero changes to the
// consumer repo's test code" success criterion — starts a real container
// against it over DOCKER_HOST. Per shipwright-testing-strategy, any test
// reaching a real Dagger container belongs at the integration level, never
// as a plain unit test, and MUST be guarded by the `integration` build tag
// so `go test ./...` stays fast and skips it cleanly.
func TestGoIntegrationTester_Test_RealEngine_DinDServiceReachable(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to dagger: %v", err)
	}
	defer client.Close()

	tmpDir := writeDinDFixtureModule(t)
	src := client.Host().Directory(tmpDir)

	tester := &golang.GoIntegrationTester{
		Client: daggerkit.NewDaggerAdapter(client),
		// The fixture's go.mod deliberately does not pin testcontainers-go
		// (kept minimal — see writeDinDFixtureModule's own doc comment), so
		// `go mod tidy` resolves it before the tagged suite runs. `go mod
		// tidy` has no `-tags` flag (unlike `go build`/`go test`) — it
		// already considers build-tag-gated files, including this fixture's
		// `//go:build integration` test file, without one. The default
		// Command alone assumes an already-tidy module, which this fixture
		// is not; overriding Command this way is exactly the "combine a
		// fixed default with an arbitrary override" reach design.md D-2/D-3
		// exist for.
		Command: "go mod tidy && go test -tags=integration -v ./...",
	}

	out, err := tester.Test(ctx, src)
	if err != nil {
		t.Fatalf("GoIntegrationTester.Test() error = %v, want nil (a real testcontainers-go fixture container must start against the DinD service)", err)
	}
	if out == nil {
		t.Fatal("GoIntegrationTester.Test() returned a nil File on success")
	}
}

// writeDinDFixtureModule writes a minimal Go module whose only test — build
// -tagged "integration", mirroring this file's own guard so a plain `go
// build ./...`/`go vet ./...` inside the container never touches it before
// `go test -tags=integration` explicitly opts in — starts a real container
// via testcontainers-go against whatever DOCKER_HOST the environment
// provides. Deliberately ordinary testcontainers-go usage: no Shipwright
// import, no awareness of GoIntegrationTester or where DOCKER_HOST comes
// from, proving the DinD mechanism needs zero changes to a consumer repo's
// own integration suite (design.md D-9).
func writeDinDFixtureModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	goMod := "module dindfixture\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	fixtureTest := `//go:build integration

package dindfixture

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

// TestRealContainerViaDinD proves a real testcontainers-go client can start
// a real container against whatever Docker daemon DOCKER_HOST points at —
// no reference to how that daemon got there.
func TestRealContainerViaDinD(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image: "alpine:3",
		Cmd:   []string{"sleep", "30"},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("testcontainers.GenericContainer() error = %v, want nil", err)
	}
	defer func() {
		_ = container.Terminate(ctx)
	}()

	state, err := container.State(ctx)
	if err != nil {
		t.Fatalf("container.State() error = %v, want nil", err)
	}
	if !state.Running {
		t.Fatal("container.State().Running = false, want true")
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "dind_fixture_test.go"), []byte(fixtureTest), 0o644); err != nil {
		t.Fatalf("failed to write dind_fixture_test.go: %v", err)
	}

	return dir
}

// timeline records each step's real [start, end) wall-clock interval under
// a mutex, plus each step's captured stdout, so
// TestGoIntegrationTester_Test_RealEngine_ConcurrentDinDDaemonsAreIsolated
// can present concrete temporal/output evidence — not just "the test
// passed." Duplicated from
// testing/integration/rust/concurrency_test.go's own timeline (which is
// itself duplicated from internal/workflow/engine/concurrency_test.go's)
// rather than shared: all three are independently buildable/runnable
// trees (plain `go test` vs. two separately `-tags integration`-gated
// packages), matching that file's own stated rationale.
type timeline struct {
	mu        sync.Mutex
	intervals map[string][2]time.Time
	outputs   map[string]string
}

func newTimeline() *timeline {
	return &timeline{intervals: make(map[string][2]time.Time), outputs: make(map[string]string)}
}

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

// instrumentedTester wraps a real shipwright.Tester (here, a real
// *golang.GoIntegrationTester), recording its [start,end) interval and
// captured stdout in tl under id — test-only instrumentation around a
// real production capability, never inside it. Mirrors
// testing/integration/rust/concurrency_test.go's own instrumentedTester.
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

// goConcurrencyWaveConfig builds an engine.Config for two independent
// same-wave GoIntegrationTester steps (no needs[] edge) — this package's
// own copy of testing/integration/rust/concurrency_test.go's
// runWaveConfig, duplicated for the same "independently buildable trees"
// reason that file gives, not shared.
func goConcurrencyWaveConfig(steps []manifest.Step, reg *providers.Registry, source *dagger.Directory, maxParallel int) (engine.Config, error) {
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

// writeDinDIsolationFixtureModule writes a minimal real Go module whose
// only program speaks raw HTTP/1.1 to the Docker Engine API over
// DOCKER_HOST — no testcontainers-go, no other dependency — to create a
// uniquely-named container on its own daemon and then list every
// container that daemon knows about.
//
// Ported from testing/integration/rust/concurrency_test.go's
// container_ops.rs fixture binary (its "create-and-list" mode, the only
// mode that test file actually drives), adapted to Go idiom (net.Dial
// plus a raw HTTP/1.1 request/response instead of std::net::TcpStream):
// same wire protocol, same two Docker Engine API calls
// (POST /images/create?fromImage=alpine, POST /containers/create,
// GET /containers/json?all=true), same single-process "create-and-list"
// shape. A single step's single command is the only way to inspect "this
// step's own daemon" after seeding it — daemon identity is scoped to the
// step (invocation.StepID, dockerdaemon.go), so splitting create and list
// into two separate steps would, by design, hand them two different
// isolated daemons and prove nothing about isolation.
func writeDinDIsolationFixtureModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	goMod := "module dindisolationfixture\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	mainGo := `package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func request(addr, req string) string {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect to DOCKER_HOST:", err)
		os.Exit(1)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(req)); err != nil {
		fmt.Fprintln(os.Stderr, "write request:", err)
		os.Exit(1)
	}
	body, err := io.ReadAll(conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read response:", err)
		os.Exit(1)
	}
	return string(body)
}

func create(addr, marker string) {
	pull := request(addr, "POST /images/create?fromImage=alpine&tag=latest HTTP/1.1\r\nHost: docker\r\nConnection: close\r\n\r\n")
	if strings.Contains(pull, "\"error\"") {
		fmt.Fprintln(os.Stderr, "image pull failed:", pull)
		os.Exit(1)
	}

	body := "{\"Image\":\"alpine:latest\",\"Cmd\":[\"sleep\",\"3600\"]}"
	createReq := fmt.Sprintf("POST /containers/create?name=%s HTTP/1.1\r\nHost: docker\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", marker, len(body), body)
	resp := request(addr, createReq)
	if !strings.HasPrefix(resp, "HTTP/1.1 201") {
		fmt.Fprintln(os.Stderr, "container create failed:", resp)
		os.Exit(1)
	}
	fmt.Println("CREATED=" + marker)
}

func list(addr string) {
	resp := request(addr, "GET /containers/json?all=true HTTP/1.1\r\nHost: docker\r\nConnection: close\r\n\r\n")
	if !strings.HasPrefix(resp, "HTTP/1.1 200") {
		fmt.Fprintln(os.Stderr, "container list failed:", resp)
		os.Exit(1)
	}
	fmt.Println("CONTAINERS=" + resp)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: dindisolationfixture <marker>")
		os.Exit(1)
	}
	marker := os.Args[1]
	host := os.Getenv("DOCKER_HOST")
	addr := strings.TrimPrefix(host, "tcp://")
	create(addr, marker)
	list(addr)
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	return dir
}

// TestGoIntegrationTester_Test_RealEngine_ConcurrentDinDDaemonsAreIsolated
// is the Go equivalent of
// testing/integration/rust/concurrency_test.go's
// TestRealDaggerConcurrency_CaseD_ContainerIsolationBetweenConcurrentDaemons
// — the direct precedent for spec.md's "Per-Step Docker Service Isolation"
// requirement (scenario: "Concurrent steps do not share a daemon"), which
// design.md's own Testing Strategy table names as the pattern to mirror.
//
// Two GoIntegrationTester steps run concurrently in the same engine wave
// (no needs[] edge, MaxParallel: 2), Docker-in-Docker always attached
// (design.md D-9 — no opt-in field, unlike Rust's Docker: true). Each
// creates a uniquely-named marker container on its own daemon, then lists
// every container its own daemon knows about — proving not just that the
// two daemons are distinct but that they share no state: step A's own
// listing must show marker A and must NOT show marker B, and vice versa.
// This is the mechanism the user asked to have ported faithfully, not
// redesigned: withDockerDaemon's SHIPWRIGHT_DIND_INSTANCE cache-busting
// (dockerdaemon.go, keyed off invocation.StepID) is shared, structurally
// identical logic between providers/go and providers/rust (design.md
// D-4b) — this test proves the same outcome the Rust precedent already
// proves, using the same container-marker mechanism, for
// GoIntegrationTester specifically.
func TestGoIntegrationTester_Test_RealEngine_ConcurrentDinDDaemonsAreIsolated(t *testing.T) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatalf("dagger.Connect() error = %v", err)
	}
	defer client.Close()

	fixtureDir := writeDinDIsolationFixtureModule(t)
	source := client.Host().Directory(fixtureDir)
	goClient := daggerkit.NewDaggerAdapter(client)
	tl := newTimeline()

	const markerA = "shipwright-go-isolation-marker-a"
	const markerB = "shipwright-go-isolation-marker-b"

	reg := providers.NewRegistry()
	reg.RegisterTester(providers.Ref{Name: "go-docker-isolation-a", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "isolationA", inner: &golang.GoIntegrationTester{Client: goClient, Command: "go run . " + markerA}}
	})
	reg.RegisterTester(providers.Ref{Name: "go-docker-isolation-b", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return instrumentedTester{tl: tl, id: "isolationB", inner: &golang.GoIntegrationTester{Client: goClient, Command: "go run . " + markerB}}
	})

	steps := []manifest.Step{
		{ID: "isolationA", Capability: "test", Uses: manifest.UsesSpec{Provider: "go-docker-isolation-a", Version: "1"}},
		{ID: "isolationB", Capability: "test", Uses: manifest.UsesSpec{Provider: "go-docker-isolation-b", Version: "1"}},
	}
	cfg, err := goConcurrencyWaveConfig(steps, reg, source, 2)
	if err != nil {
		t.Fatalf("goConcurrencyWaveConfig() error = %v", err)
	}

	res, err := engine.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("engine.Execute() error = %v, want nil (two independent GoIntegrationTester DinD daemons must both start and stay isolated)", err)
	}
	if res.Failed() {
		t.Fatalf("engine.Execute() Failures = %v, want none", res.Failures)
	}
	tl.logAll(t)
	if !tl.overlaps("isolationA", "isolationB") {
		t.Fatal("the two GoIntegrationTester steps never overlapped — want genuine concurrent execution against real Dagger")
	}

	outA := tl.output("isolationA")
	outB := tl.output("isolationB")
	if !strings.Contains(outA, "CREATED="+markerA) {
		t.Fatalf("isolationA output = %q, want it to confirm creating %q", outA, markerA)
	}
	if !strings.Contains(outB, "CREATED="+markerB) {
		t.Fatalf("isolationB output = %q, want it to confirm creating %q", outB, markerB)
	}

	// Each step's own listing, taken from its own daemon right after
	// creating its own marker: A's daemon must never see B's marker, and
	// B's daemon must never see A's — this is the actual spec scenario
	// text ("neither step observes containers started by the other
	// step's daemon"), not merely a proxy like distinct daemon IDs.
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
