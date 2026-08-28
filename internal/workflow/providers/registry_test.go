// Package providers_test exercises stage 6 (provider resolution,
// design.md D-I) of the manifest's fixed seven-stage validation pipeline:
// typed Registry resolution hit/miss per capability (tasks.md 7.1),
// unsupported-version rejection (tasks.md 7.2), and `with`-value kind
// checking against a provider's declared WithSchema (tasks.md 7.5). The
// SECURITY RED tests (unregistered module fails closed, no
// manifest-reachable plugin.Open) live in their own files — register_test.go
// and security_test.go respectively — per the launch instructions'
// separation.
package providers_test

import (
	"context"
	"errors"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/internal/workflow/providers"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// fakeBuilder/fakeTester/fakeArtifactor/fakeDeployer/fakeRunner are
// hand-rolled fakes satisfying each Layer 1 capability interface
// (pkg/shipwright) — this package tests provider RESOLUTION, not any
// concrete provider's own behavior (that belongs to internal/capabilities,
// WU3), so a minimal fake is the correct test double here (testing-tdd
// skill's double-selection order: no existing fake/mock for these
// interfaces exists yet, and each is a one-method interface, so a
// hand-rolled stub is appropriate over pulling in gomock).
type fakeBuilder struct{}

func (fakeBuilder) Build(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
	return source, nil
}

type fakeTester struct{}

func (fakeTester) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	return nil, nil
}

type fakeArtifactor struct{}

func (fakeArtifactor) Publish(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error) {
	return ref, nil
}

type fakeDeployer struct{}

func (fakeDeployer) Deploy(ctx context.Context, artifactRef, environment string, creds *dagger.Secret) (string, error) {
	return artifactRef, nil
}

type fakeRunner struct{}

func (fakeRunner) Run(ctx context.Context, build *dagger.Directory) (*dagger.Container, error) {
	return nil, nil
}

// fakeRuntimeInspector satisfies shipwright.RuntimeInspector
// (runtime-toolchain-upgrade, design.md D-4b) — this package tests provider
// RESOLUTION for the sixth capability the same way it does the original
// five, not any concrete provider's own behavior.
type fakeRuntimeInspector struct{}

func (fakeRuntimeInspector) Inspect(ctx context.Context, source *dagger.Directory) (string, error) {
	return "{}", nil
}

// tasks.md 7.1: one Register*/Resolve* hit/miss pair per capability — five
// total (Builder, Tester, Artifactor, Deployer, Runner), matching
// design.md D-I's five typed register methods exactly.
func TestRegistry_BuilderResolutionHitAndMiss(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	ref := providers.Ref{Name: "go", Version: "1"}
	r.RegisterBuilder(ref, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{}
	})

	got, err := r.ResolveBuilder(ref, providers.Values{})
	if err != nil {
		t.Fatalf("ResolveBuilder(registered ref) error = %v, want nil (hit)", err)
	}
	if got == nil {
		t.Fatal("ResolveBuilder(registered ref) = nil, want a resolved shipwright.Builder")
	}

	miss := providers.Ref{Name: "gradle", Version: "1"}
	_, err = r.ResolveBuilder(miss, providers.Values{})
	if err == nil {
		t.Fatal("ResolveBuilder(unregistered ref) error = nil, want an error (miss)")
	}
	var unregistered *providers.UnregisteredProviderError
	if !errors.As(err, &unregistered) {
		t.Fatalf("ResolveBuilder(unregistered ref) error = %v (%T), want *providers.UnregisteredProviderError", err, err)
	}
}

func TestRegistry_TesterResolutionHitAndMiss(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	ref := providers.Ref{Name: "go-test", Version: "1"}
	r.RegisterTester(ref, providers.WithSchema{}, func(providers.Values) shipwright.Tester {
		return fakeTester{}
	})

	if got, err := r.ResolveTester(ref, providers.Values{}); err != nil || got == nil {
		t.Fatalf("ResolveTester(registered ref) = (%v, %v), want (non-nil, nil)", got, err)
	}

	if _, err := r.ResolveTester(providers.Ref{Name: "ghost", Version: "1"}, providers.Values{}); err == nil {
		t.Fatal("ResolveTester(unregistered ref) error = nil, want an error (miss)")
	}
}

func TestRegistry_ArtifactorResolutionHitAndMiss(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	ref := providers.Ref{Name: "container", Version: "1"}
	r.RegisterArtifactor(ref, providers.WithSchema{}, func(providers.Values) shipwright.Artifactor {
		return fakeArtifactor{}
	})

	if got, err := r.ResolveArtifactor(ref, providers.Values{}); err != nil || got == nil {
		t.Fatalf("ResolveArtifactor(registered ref) = (%v, %v), want (non-nil, nil)", got, err)
	}

	if _, err := r.ResolveArtifactor(providers.Ref{Name: "ghost", Version: "1"}, providers.Values{}); err == nil {
		t.Fatal("ResolveArtifactor(unregistered ref) error = nil, want an error (miss)")
	}
}

func TestRegistry_DeployerResolutionHitAndMiss(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	ref := providers.Ref{Name: "nomad", Version: "1"}
	r.RegisterDeployer(ref, providers.WithSchema{}, func(providers.Values) shipwright.Deployer {
		return fakeDeployer{}
	})

	if got, err := r.ResolveDeployer(ref, providers.Values{}); err != nil || got == nil {
		t.Fatalf("ResolveDeployer(registered ref) = (%v, %v), want (non-nil, nil)", got, err)
	}

	if _, err := r.ResolveDeployer(providers.Ref{Name: "ghost", Version: "1"}, providers.Values{}); err == nil {
		t.Fatal("ResolveDeployer(unregistered ref) error = nil, want an error (miss)")
	}
}

func TestRegistry_RunnerResolutionHitAndMiss(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	ref := providers.Ref{Name: "local", Version: "1"}
	r.RegisterRunner(ref, providers.WithSchema{}, func(providers.Values) shipwright.Runner {
		return fakeRunner{}
	})

	if got, err := r.ResolveRunner(ref, providers.Values{}); err != nil || got == nil {
		t.Fatalf("ResolveRunner(registered ref) = (%v, %v), want (non-nil, nil)", got, err)
	}

	if _, err := r.ResolveRunner(providers.Ref{Name: "ghost", Version: "1"}, providers.Values{}); err == nil {
		t.Fatal("ResolveRunner(unregistered ref) error = nil, want an error (miss)")
	}
}

// RuntimeInspector's own hit/miss pair (runtime-toolchain-upgrade,
// design.md D-4b task 1.16): the sixth capability follows the exact same
// Register*/Resolve* shape as the original five above.
func TestRegistry_RuntimeInspectorResolutionHitAndMiss(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	ref := providers.Ref{Name: "go-runtime", Version: "1"}
	r.RegisterRuntimeInspector(ref, providers.WithSchema{}, func(providers.Values) shipwright.RuntimeInspector {
		return fakeRuntimeInspector{}
	})

	got, err := r.ResolveRuntimeInspector(ref, providers.Values{})
	if err != nil {
		t.Fatalf("ResolveRuntimeInspector(registered ref) error = %v, want nil (hit)", err)
	}
	if got == nil {
		t.Fatal("ResolveRuntimeInspector(registered ref) = nil, want a resolved shipwright.RuntimeInspector")
	}

	miss := providers.Ref{Name: "ghost", Version: "1"}
	_, err = r.ResolveRuntimeInspector(miss, providers.Values{})
	if err == nil {
		t.Fatal("ResolveRuntimeInspector(unregistered ref) error = nil, want an error (miss)")
	}
	var unregistered *providers.UnregisteredProviderError
	if !errors.As(err, &unregistered) {
		t.Fatalf("ResolveRuntimeInspector(unregistered ref) error = %v (%T), want *providers.UnregisteredProviderError", err, err)
	}
}

// tasks.md 7.2: a provider name/module IS registered, but not at the
// requested version — this must be a DISTINCT failure mode from "not
// registered at all" (tasks.md 7.3/UnregisteredProviderError), so a
// caller/operator can tell "you asked for a version we don't ship" apart
// from "you asked for a provider that doesn't exist".
func TestRegistry_ResolveBuilder_UnsupportedVersionRejected(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	r.RegisterBuilder(providers.Ref{Name: "go", Version: "1"}, providers.WithSchema{}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{}
	})

	_, err := r.ResolveBuilder(providers.Ref{Name: "go", Version: "2"}, providers.Values{})
	if err == nil {
		t.Fatal("ResolveBuilder(registered name, unregistered version) error = nil, want an unsupported-version error")
	}

	var versionErr *providers.UnsupportedVersionError
	if !errors.As(err, &versionErr) {
		t.Fatalf("ResolveBuilder(unsupported version) error = %v (%T), want *providers.UnsupportedVersionError", err, err)
	}
	if versionErr.RequestedVersion != "2" {
		t.Fatalf("UnsupportedVersionError.RequestedVersion = %q, want %q", versionErr.RequestedVersion, "2")
	}

	// A version mismatch must NEVER be reported as UnregisteredProviderError
	// — the two failure modes are distinct (a registered name at the wrong
	// version is not the same fail-closed boundary as an unregistered
	// module, tasks.md 7.3).
	var unregistered *providers.UnregisteredProviderError
	if errors.As(err, &unregistered) {
		t.Fatal("unsupported-version error also matches *providers.UnregisteredProviderError — the two failure modes must be distinct types")
	}
}

// tasks.md 7.5: a `with` value whose resolved Kind does not match the
// provider's declared WithSchema for that field is rejected at resolution
// (stage 7). This is the general form (an int-typed schema field given a
// string value)...
func TestRegistry_ResolveBuilder_WithKindMismatchRejected(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	ref := providers.Ref{Name: "go", Version: "1"}
	r.RegisterBuilder(ref, providers.WithSchema{
		"retries": interp.KindInt,
	}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{}
	})

	_, err := r.ResolveBuilder(ref, providers.Values{
		"retries": interp.NewString("not-an-int"),
	})
	if err == nil {
		t.Fatal("ResolveBuilder(with kind mismatch) error = nil, want a WithSchemaMismatchError")
	}

	var mismatchErr *providers.WithSchemaMismatchError
	if !errors.As(err, &mismatchErr) {
		t.Fatalf("ResolveBuilder(with kind mismatch) error = %v (%T), want *providers.WithSchemaMismatchError", err, err)
	}
	if mismatchErr.Field != "retries" || mismatchErr.Want != interp.KindInt || mismatchErr.Got != interp.KindString {
		t.Fatalf("WithSchemaMismatchError = %+v, want Field=retries Want=KindInt Got=KindString", mismatchErr)
	}
}

// ...and this is the SECURITY-CRITICAL specific case: closing WU5's
// forbidPlaintext loop (tasks.md 5.3's deferred boundary) and WU6's
// deferred with-field kind check (build.go's package doc comment) — a
// `secrets.*` reference resolves to a KindSecret Value, and using it for a
// field the provider's schema declares as a NON-secret kind (e.g. a plain
// string field) must be rejected here, at stage 7, because there is no
// earlier stage with the provider's schema available to check against.
func TestRegistry_ResolveArtifactor_SecretInNonSecretFieldRejected(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	ref := providers.Ref{Name: "container", Version: "1"}
	r.RegisterArtifactor(ref, providers.WithSchema{
		"ref": interp.KindString, // NOT secret-typed
	}, func(providers.Values) shipwright.Artifactor {
		return fakeArtifactor{}
	})

	secretValue := interp.NewSecret(&dagger.Secret{})

	_, err := r.ResolveArtifactor(ref, providers.Values{
		"ref": secretValue,
	})
	if err == nil {
		t.Fatal("ResolveArtifactor(secret in non-secret field) error = nil, want forbidPlaintext rejection")
	}

	var mismatchErr *providers.WithSchemaMismatchError
	if !errors.As(err, &mismatchErr) {
		t.Fatalf("ResolveArtifactor(secret in non-secret field) error = %v (%T), want *providers.WithSchemaMismatchError", err, err)
	}
	if mismatchErr.Got != interp.KindSecret {
		t.Fatalf("WithSchemaMismatchError.Got = %s, want %s (secret)", mismatchErr.Got, interp.KindSecret)
	}
}

// A field the schema declares but the manifest's `with` map never
// supplied is not a resolution error here — an absent optional field is a
// concern for a later/consuming layer (the concrete provider factory), not
// this package's kind check, which only compares kinds it can actually
// observe.
func TestRegistry_ResolveBuilder_MissingWithFieldAccepted(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	ref := providers.Ref{Name: "go", Version: "1"}
	r.RegisterBuilder(ref, providers.WithSchema{
		"goVersion": interp.KindString,
	}, func(providers.Values) shipwright.Builder {
		return fakeBuilder{}
	})

	if _, err := r.ResolveBuilder(ref, providers.Values{}); err != nil {
		t.Fatalf("ResolveBuilder(schema field absent from with) error = %v, want nil", err)
	}
}
