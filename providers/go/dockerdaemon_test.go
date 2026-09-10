package golang

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/pablogore/shipwright/pkg/shipwright/invocation"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// Internal (package golang, not golang_test) unit tests for withDockerDaemon
// — the unexported helper is not part of any capability's public contract,
// same convention as internal_test.go's resolveGoVersion/resolveBinaryName
// coverage. Mirrors providers/rust/dockerdaemon.go's own mechanism (design.md
// D-4), ported to this module's daggerkit surface (tasks.md Phase 2).

// dindTestFixture wires a fresh set of mocks for one withDockerDaemon call:
// a DaggerClient producing the DinD service container, and the caller's own
// container that gets the service bound onto it.
type dindTestFixture struct {
	client        *daggerkit.MockDaggerClient
	dindContainer *daggerkit.MockDaggerContainer
	dindService   *daggerkit.MockDaggerService
	caller        *daggerkit.MockDaggerContainer
	wired         *daggerkit.MockDaggerContainer
}

// newDindTestFixture stubs the DinD wiring chain (From/WithExposedPort/
// AsService on the service side, WithServiceBinding/WithEnvVariable on the
// caller side). The SHIPWRIGHT_DIND_INSTANCE WithEnvVariable expectation is
// deliberately NOT stubbed here — every test below registers its own, some
// (TestWithDockerDaemon_DistinctInstancePerStepID,
// TestWithDockerDaemon_NoStepIDFallsBackToUUID) needing to capture the
// generated value via mock.Arguments, which only the first registered
// matching expectation would ever see.
func newDindTestFixture() *dindTestFixture {
	f := &dindTestFixture{
		client:        &daggerkit.MockDaggerClient{},
		dindContainer: &daggerkit.MockDaggerContainer{},
		dindService:   &daggerkit.MockDaggerService{},
		caller:        &daggerkit.MockDaggerContainer{},
		wired:         &daggerkit.MockDaggerContainer{},
	}

	f.client.On("Container").Return(f.dindContainer)
	f.dindContainer.On("From", "docker:27-dind").Return(f.dindContainer)
	f.dindContainer.On("WithExposedPort", 2375).Return(f.dindContainer)
	f.dindContainer.On("AsService", []string{"dockerd", "--host=tcp://0.0.0.0:2375", "--tls=false"}).Return(f.dindService)
	f.caller.On("WithServiceBinding", "docker", f.dindService).Return(f.wired)
	f.wired.On("WithEnvVariable", "DOCKER_HOST", "tcp://docker:2375").Return(f.wired)

	return f
}

// TestWithDockerDaemon_DinDWiring asserts the exact DinD wiring chain the
// design specifies: From("docker:27-dind"), WithExposedPort(2375),
// AsService([]string{"dockerd", "--host=tcp://0.0.0.0:2375", "--tls=false"}),
// WithServiceBinding("docker", svc), WithEnvVariable("DOCKER_HOST",
// "tcp://docker:2375") on the returned container.
func TestWithDockerDaemon_DinDWiring(t *testing.T) {
	f := newDindTestFixture()
	f.dindContainer.On("WithEnvVariable", dindInstanceEnv, mock.AnythingOfType("string")).Return(f.dindContainer)

	got := withDockerDaemon(context.Background(), f.client, f.caller)

	assert.Same(t, f.wired, got)
	f.dindContainer.AssertExpectations(t)
	f.caller.AssertExpectations(t)
	f.wired.AssertExpectations(t)
}

// TestWithDockerDaemon_NeverCallsWithExecBeforeAsService pins D-4(a), the
// single assertion nothing else in this suite would catch: dockerd's
// startup args must go straight to AsService, never a prior WithExec.
// WithExec's semantics are "run to completion and snapshot the result,"
// which a daemon that runs forever never does — chaining
// WithExec().AsService() leaves the container permanently pending and
// AsService itself never resolves. A real, isolated probe hung for the
// full 30-minute timeout under exactly this mistake; AsService(Args:...)
// resolved in ~2s. This test pins that regression permanently.
func TestWithDockerDaemon_NeverCallsWithExecBeforeAsService(t *testing.T) {
	f := newDindTestFixture()
	f.dindContainer.On("WithEnvVariable", dindInstanceEnv, mock.AnythingOfType("string")).Return(f.dindContainer)

	withDockerDaemon(context.Background(), f.client, f.caller)

	f.dindContainer.AssertNotCalled(t, "WithExec", dockerdCommand, mock.Anything)
}

// TestWithDockerDaemon_DistinctInstancePerStepID pins D-4(b): Dagger's
// graph is content-addressed, so two identical From/WithExposedPort/
// AsService chains resolve to the same node — two concurrent steps would
// share one live daemon without a per-invocation-unique
// SHIPWRIGHT_DIND_INSTANCE value. Two ctxs carrying distinct StepIDs must
// produce two distinct env values.
func TestWithDockerDaemon_DistinctInstancePerStepID(t *testing.T) {
	var gotFirst, gotSecond string

	first := newDindTestFixture()
	first.dindContainer.On("WithEnvVariable", dindInstanceEnv, mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) { gotFirst = args.String(1) }).
		Return(first.dindContainer)
	ctxFirst := invocation.WithStepID(context.Background(), "step-alpha")
	withDockerDaemon(ctxFirst, first.client, first.caller)

	second := newDindTestFixture()
	second.dindContainer.On("WithEnvVariable", dindInstanceEnv, mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) { gotSecond = args.String(1) }).
		Return(second.dindContainer)
	ctxSecond := invocation.WithStepID(context.Background(), "step-beta")
	withDockerDaemon(ctxSecond, second.client, second.caller)

	assert.Equal(t, "step-alpha", gotFirst)
	assert.Equal(t, "step-beta", gotSecond)
	assert.NotEqual(t, gotFirst, gotSecond)
}

// TestWithDockerDaemon_NoStepIDFallsBackToUUID covers the direct-unit-test
// call path: a ctx carrying no step id (as here) must still produce a
// non-empty, per-call-unique SHIPWRIGHT_DIND_INSTANCE value via
// github.com/google/uuid, so isolation still holds even without the
// reproducibility guarantee invocation.StepID provides.
func TestWithDockerDaemon_NoStepIDFallsBackToUUID(t *testing.T) {
	var gotFirst, gotSecond string

	first := newDindTestFixture()
	first.dindContainer.On("WithEnvVariable", dindInstanceEnv, mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) { gotFirst = args.String(1) }).
		Return(first.dindContainer)
	withDockerDaemon(context.Background(), first.client, first.caller)

	second := newDindTestFixture()
	second.dindContainer.On("WithEnvVariable", dindInstanceEnv, mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) { gotSecond = args.String(1) }).
		Return(second.dindContainer)
	withDockerDaemon(context.Background(), second.client, second.caller)

	assert.NotEmpty(t, gotFirst)
	assert.NotEmpty(t, gotSecond)
	assert.NotEqual(t, gotFirst, gotSecond)
}
