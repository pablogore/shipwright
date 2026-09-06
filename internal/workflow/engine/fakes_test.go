package engine_test

import (
	"context"
	"sync"

	"dagger.io/dagger"
)

// fakeBuilder/fakeTester/fakeArtifactor/fakeDeployer/fakeRunner are
// hand-rolled fakes satisfying each Layer 1 capability interface
// (pkg/shipwright) — this package tests the ENGINE's wave scheduling,
// dispatch, and value binding, not any concrete provider's own behavior,
// so a minimal configurable-func fake is the correct test double
// (testing-tdd skill's double-selection order: no existing fake/mock for
// these interfaces exists yet, and each is a one-method interface). This
// mirrors internal/workflow/providers/registry_test.go's own fake style,
// extended with a configurable Func field per the pattern this repo's
// MockExecutor/MockPipelinesPipeline already use elsewhere.
type fakeBuilder struct {
	BuildFunc func(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error)
}

func (f fakeBuilder) Build(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
	if f.BuildFunc != nil {
		return f.BuildFunc(ctx, source)
	}
	return source, nil
}

type fakeTester struct {
	TestFunc func(ctx context.Context, source *dagger.Directory) (*dagger.File, error)
}

func (f fakeTester) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if f.TestFunc != nil {
		return f.TestFunc(ctx, source)
	}
	return nil, nil
}

type fakeArtifactor struct {
	PublishFunc func(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error)
}

func (f fakeArtifactor) Publish(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error) {
	if f.PublishFunc != nil {
		return f.PublishFunc(ctx, build, ref, creds)
	}
	return ref, nil
}

type fakeDeployer struct {
	DeployFunc func(ctx context.Context, artifactRef, environment string, creds *dagger.Secret) (string, error)
}

func (f fakeDeployer) Deploy(ctx context.Context, artifactRef, environment string, creds *dagger.Secret) (string, error) {
	if f.DeployFunc != nil {
		return f.DeployFunc(ctx, artifactRef, environment, creds)
	}
	return artifactRef, nil
}

type fakeRunner struct {
	RunFunc func(ctx context.Context, build *dagger.Directory) (*dagger.Container, error)
}

func (f fakeRunner) Run(ctx context.Context, build *dagger.Directory) (*dagger.Container, error) {
	if f.RunFunc != nil {
		return f.RunFunc(ctx, build)
	}
	return nil, nil
}

// fakeRuntimeInspector satisfies shipwright.RuntimeInspector
// (runtime-toolchain-upgrade, design.md D-4b) — same configurable-func fake
// style as the original five above.
type fakeRuntimeInspector struct {
	InspectFunc func(ctx context.Context, source *dagger.Directory) (string, error)
}

func (f fakeRuntimeInspector) Inspect(ctx context.Context, source *dagger.Directory) (string, error) {
	if f.InspectFunc != nil {
		return f.InspectFunc(ctx, source)
	}
	return "{}", nil
}

// fakeRuntimeUpgrader satisfies shipwright.RuntimeUpgrader
// (runtime-toolchain-upgrade, design.md D-9) — same configurable-func fake
// style as the others above.
type fakeRuntimeUpgrader struct {
	UpgradeFunc func(ctx context.Context, source *dagger.Directory, targetVersion string) (*dagger.Directory, error)
}

func (f fakeRuntimeUpgrader) Upgrade(ctx context.Context, source *dagger.Directory, targetVersion string) (*dagger.Directory, error) {
	if f.UpgradeFunc != nil {
		return f.UpgradeFunc(ctx, source, targetVersion)
	}
	return source, nil
}

// recorder records step invocation order, guarded by a mutex so `go test
// -race` genuinely proves serial access rather than merely assuming it —
// if wave execution ever became concurrent (it must not, design.md D-K),
// a data race on an unguarded slice would be a weaker signal than this
// mutex-guarded recorder's own ability to still produce a deterministic,
// race-free order under -race.
type recorder struct {
	mu    sync.Mutex
	order []string
}

func (r *recorder) record(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, id)
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}
