package main

import (
	"context"
	"testing"

	"dagger/shipwright/internal/dagger"
)

// fakeDeployer is a minimal dagger.DaggerObject + Deployer double used only
// to prove Plan.Execute's control flow. It is a pure Go stub — it records
// whether Deploy was invoked and needs no live Dagger engine, since the
// guard under test (PR #148 review finding 2) must reject the call BEFORE
// any interface method runs: Deploy requires an Artifact to have been
// chained first, so calling it with an empty artifactRef must fail closed,
// per design.md's Threat Matrix routing/composition invariant ("MUST fail
// closed, never silently proceed with missing/invalid state").
type fakeDeployer struct {
	called bool
}

func (f *fakeDeployer) XXX_GraphQLType() string   { return "FakeDeployer" }
func (f *fakeDeployer) XXX_GraphQLIDType() string { return "FakeDeployerID" }

func (f *fakeDeployer) XXX_GraphQLID(ctx context.Context) (string, error) {
	return "fake-deployer-id", nil
}

func (f *fakeDeployer) MarshalJSON() ([]byte, error) {
	return []byte(`"fake-deployer-id"`), nil
}

func (f *fakeDeployer) ID(ctx context.Context) (dagger.ID, error) {
	return dagger.ID("fake-deployer-id"), nil
}

func (f *fakeDeployer) Deploy(ctx context.Context, artifactRef, environment string, creds *dagger.Secret) (string, error) {
	f.called = true
	return "deployed", nil
}

// TestPlanExecute_DeployWithoutArtifactFailsClosed asserts that chaining
// WithDeploy without a preceding WithArtifact returns an error and never
// invokes Deploy.Deploy — a plain control-flow/validation test, no
// dagger.Directory or live engine involved, since the check must happen
// before any interface method call.
func TestPlanExecute_DeployWithoutArtifactFailsClosed(t *testing.T) {
	deployer := &fakeDeployer{}
	plan := (&Plan{}).WithDeploy(deployer, "staging", nil)

	_, err := plan.Execute(context.Background())
	if err == nil {
		t.Fatal("Execute() with Deploy chained but no Artifact must return an error, got nil")
	}
	if deployer.called {
		t.Fatal("Execute() must not invoke Deploy.Deploy when no Artifact was chained — deploy requires an artifact")
	}
}

// TestPlanExecute_DeployWithArtifactInvokesDeploy is the paired positive
// case: when an Artifactor IS chained, the new guard must not block a
// legitimate Deploy call. Publish itself needs a real dagger.Directory to
// resolve, so this asserts only that the guard does not fire — it does not
// exercise a full Artifact+Deploy round trip (that remains an engine-level
// concern, see make dagger-test).
func TestPlanExecute_DeployWithArtifactInvokesDeploy(t *testing.T) {
	deployer := &fakeDeployer{}
	plan := &Plan{Artifact: &fakeArtifactor{}}
	plan.WithDeploy(deployer, "staging", nil)

	_, err := plan.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil when Artifact was chained before Deploy", err)
	}
	if !deployer.called {
		t.Fatal("Execute() must invoke Deploy.Deploy when an Artifact was chained")
	}
}

// fakeArtifactor is the minimal double needed to exercise the positive
// (Artifact-then-Deploy) path above.
type fakeArtifactor struct{}

func (f *fakeArtifactor) XXX_GraphQLType() string   { return "FakeArtifactor" }
func (f *fakeArtifactor) XXX_GraphQLIDType() string { return "FakeArtifactorID" }

func (f *fakeArtifactor) XXX_GraphQLID(ctx context.Context) (string, error) {
	return "fake-artifactor-id", nil
}

func (f *fakeArtifactor) MarshalJSON() ([]byte, error) {
	return []byte(`"fake-artifactor-id"`), nil
}

func (f *fakeArtifactor) ID(ctx context.Context) (dagger.ID, error) {
	return dagger.ID("fake-artifactor-id"), nil
}

func (f *fakeArtifactor) Publish(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error) {
	return "published-ref", nil
}
