package shipwright_test

import (
	"context"
	"reflect"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// Concrete, non-generic stub implementations of each capability interface.
//
// These types deliberately use no type parameters. If a capability
// interface required a Go generic type parameter as part of its public
// contract, one of the following declarations would fail to compile — that
// is the RED/GREEN proof for design.md D-A's binding constraint: "capability
// interface methods use ONLY Dagger core types (Directory, File, Container,
// Secret) and scalars... capability interfaces ... MUST have zero Go generic
// type parameters."
type stubBuilder struct{}

func (stubBuilder) Build(_ context.Context, source *dagger.Directory) (*dagger.Directory, error) {
	return source, nil
}

type stubTester struct{}

func (stubTester) Test(_ context.Context, _ *dagger.Directory) (*dagger.File, error) {
	return nil, nil
}

type stubArtifactor struct{}

func (stubArtifactor) Publish(_ context.Context, _ *dagger.Directory, ref string, _ *dagger.Secret) (string, error) {
	return ref, nil
}

type stubDeployer struct{}

func (stubDeployer) Deploy(_ context.Context, artifactRef, _ string, _ *dagger.Secret) (string, error) {
	return artifactRef, nil
}

type stubRunner struct{}

func (stubRunner) Run(_ context.Context, _ *dagger.Directory) (*dagger.Container, error) {
	return nil, nil
}

// Compile-time proof: each capability interface is satisfied by a concrete,
// non-generic type — referencing "shipwright.Builder" (etc.) with no type
// argument would itself fail to compile if the interface declared one.
var (
	_ shipwright.Builder    = stubBuilder{}
	_ shipwright.Tester     = stubTester{}
	_ shipwright.Artifactor = stubArtifactor{}
	_ shipwright.Deployer   = stubDeployer{}
	_ shipwright.Runner     = stubRunner{}
)

// allowedSignatureType reports whether t is one of the types design.md D-A
// permits in a capability interface method signature: context.Context,
// error, a Go scalar (string), or one of the four Dagger core types
// (*dagger.Directory, *dagger.File, *dagger.Container, *dagger.Secret).
// Module-defined Object types are never permitted.
func allowedSignatureType(t reflect.Type) bool {
	allowed := []reflect.Type{
		reflect.TypeFor[context.Context](),
		reflect.TypeFor[error](),
		reflect.TypeFor[string](),
		reflect.TypeFor[*dagger.Directory](),
		reflect.TypeFor[*dagger.File](),
		reflect.TypeFor[*dagger.Container](),
		reflect.TypeFor[*dagger.Secret](),
	}
	for _, a := range allowed {
		if t == a {
			return true
		}
	}
	return false
}

func TestCapabilityInterfaces_NoGenericTypeParametersAndDaggerOnlySignatures(t *testing.T) {
	t.Parallel()

	interfaces := map[string]reflect.Type{
		"Builder":    reflect.TypeFor[shipwright.Builder](),
		"Tester":     reflect.TypeFor[shipwright.Tester](),
		"Artifactor": reflect.TypeFor[shipwright.Artifactor](),
		"Deployer":   reflect.TypeFor[shipwright.Deployer](),
		"Runner":     reflect.TypeFor[shipwright.Runner](),
	}

	for name, ifaceType := range interfaces {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if ifaceType.Kind() != reflect.Interface {
				t.Fatalf("shipwright.%s is not an interface type", name)
			}

			if ifaceType.NumMethod() == 0 {
				t.Fatalf("shipwright.%s declares no methods", name)
			}

			for i := 0; i < ifaceType.NumMethod(); i++ {
				method := ifaceType.Method(i)
				sig := method.Type

				for p := 0; p < sig.NumIn(); p++ {
					in := sig.In(p)
					if !allowedSignatureType(in) {
						t.Errorf("%s.%s parameter %d has disallowed type %s (D-A permits only Dagger core types + scalars)",
							name, method.Name, p, in)
					}
				}

				for r := 0; r < sig.NumOut(); r++ {
					out := sig.Out(r)
					if !allowedSignatureType(out) {
						t.Errorf("%s.%s return %d has disallowed type %s (D-A permits only Dagger core types + scalars)",
							name, method.Name, r, out)
					}
				}
			}
		})
	}
}
