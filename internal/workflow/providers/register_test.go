// register_test.go covers tasks.md 7.3 (SECURITY RED: an unregistered
// `module:` reference fails closed, naming the module path) and the GREEN
// evidence for tasks.md 7.6 (RegisterDefaults wires the five WU3
// capabilities — internal/capabilities.{GoBuilder, GoUnitTester,
// GoLinter, GoVulnScanner, ContainerPublisher} — into a Registry and each
// resolves successfully by the same design.md example manifest's uses/with
// shape).
package providers_test

import (
	"errors"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/internal/workflow/providers"
)

// tasks.md 7.3, SECURITY RED: this is the test that proves the
// "compile-time-only, no plugin.Open" boundary design.md D-I describes.
// uses.module resolves ONLY against providers registered at build time —
// there is no fetch, download, cache, or dynamic-load fallback for an
// unregistered module path, only a fail-closed error naming that path.
func TestRegisterDefaults_UnregisteredModuleFailsClosed(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	providers.RegisterDefaults(r, nil)

	moduleRef := providers.Ref{
		Name:    "github.com/acme/custom-builder",
		Module:  "github.com/acme/custom-builder",
		Version: "1",
	}

	_, err := r.ResolveBuilder(moduleRef, providers.Values{})
	if err == nil {
		t.Fatal("ResolveBuilder(unregistered module) error = nil, want a fail-closed error naming the module path")
	}

	var unregistered *providers.UnregisteredProviderError
	if !errors.As(err, &unregistered) {
		t.Fatalf("ResolveBuilder(unregistered module) error = %v (%T), want *providers.UnregisteredProviderError", err, err)
	}
	if unregistered.Ref.Module != "github.com/acme/custom-builder" {
		t.Fatalf("UnregisteredProviderError.Ref.Module = %q, want %q", unregistered.Ref.Module, "github.com/acme/custom-builder")
	}
	if !containsModulePath(err.Error(), "github.com/acme/custom-builder") {
		t.Fatalf("UnregisteredProviderError.Error() = %q, must name the unregistered module path", err.Error())
	}
}

func containsModulePath(msg, path string) bool {
	return len(msg) > 0 && (indexOf(msg, path) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// tasks.md 7.6 GREEN evidence: RegisterDefaults registers all five WU3
// capabilities (internal/capabilities) under the provider names
// design.md's own example manifest uses (D-I interfaces table + the
// go-service-release example), and every one resolves without error.
func TestRegisterDefaults_RegistersAllFiveWU3Capabilities(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	providers.RegisterDefaults(r, nil)

	builder, err := r.ResolveBuilder(providers.Ref{Name: "go", Version: "1"}, providers.Values{
		"goVersion": interp.NewString("1.26.1"),
	})
	if err != nil || builder == nil {
		t.Fatalf("ResolveBuilder(go) = (%v, %v), want (non-nil GoBuilder, nil)", builder, err)
	}

	unitTester, err := r.ResolveTester(providers.Ref{Name: "go-test", Version: "1"}, providers.Values{})
	if err != nil || unitTester == nil {
		t.Fatalf("ResolveTester(go-test) = (%v, %v), want (non-nil GoUnitTester, nil)", unitTester, err)
	}

	linter, err := r.ResolveTester(providers.Ref{Name: "golangci-lint", Version: "1"}, providers.Values{})
	if err != nil || linter == nil {
		t.Fatalf("ResolveTester(golangci-lint) = (%v, %v), want (non-nil GoLinter, nil)", linter, err)
	}

	vulnScanner, err := r.ResolveTester(providers.Ref{Name: "govulncheck", Version: "1"}, providers.Values{})
	if err != nil || vulnScanner == nil {
		t.Fatalf("ResolveTester(govulncheck) = (%v, %v), want (non-nil GoVulnScanner, nil)", vulnScanner, err)
	}

	publisher, err := r.ResolveArtifactor(providers.Ref{Name: "container", Version: "1"}, providers.Values{
		"ref":   interp.NewString("ghcr.io/acme/api"),
		"creds": interp.NewSecret(&dagger.Secret{}),
	})
	if err != nil || publisher == nil {
		t.Fatalf("ResolveArtifactor(container) = (%v, %v), want (non-nil ContainerPublisher, nil)", publisher, err)
	}
}

// providers/rust mirrors providers/go file-for-file (RustBuilder,
// RustUnitTester, RustLinter, RustVulnScanner, ContainerPublisher). This is
// the GREEN evidence that RegisterDefaults wires all five of them into a
// Registry under their own manifest-facing names ("rust", "rust-test",
// "clippy", "cargo-audit", "rust-container") and each resolves without
// error, mirroring TestRegisterDefaults_RegistersAllFiveWU3Capabilities.
func TestRegisterDefaults_RegistersRustProviders(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	providers.RegisterDefaults(r, nil)

	builder, err := r.ResolveBuilder(providers.Ref{Name: "rust", Version: "1"}, providers.Values{
		"binaryName": interp.NewString("myapp"),
	})
	if err != nil || builder == nil {
		t.Fatalf("ResolveBuilder(rust) = (%v, %v), want (non-nil RustBuilder, nil)", builder, err)
	}

	unitTester, err := r.ResolveTester(providers.Ref{Name: "rust-test", Version: "1"}, providers.Values{})
	if err != nil || unitTester == nil {
		t.Fatalf("ResolveTester(rust-test) = (%v, %v), want (non-nil RustUnitTester, nil)", unitTester, err)
	}

	linter, err := r.ResolveTester(providers.Ref{Name: "clippy", Version: "1"}, providers.Values{})
	if err != nil || linter == nil {
		t.Fatalf("ResolveTester(clippy) = (%v, %v), want (non-nil RustLinter, nil)", linter, err)
	}

	vulnScanner, err := r.ResolveTester(providers.Ref{Name: "cargo-audit", Version: "1"}, providers.Values{})
	if err != nil || vulnScanner == nil {
		t.Fatalf("ResolveTester(cargo-audit) = (%v, %v), want (non-nil RustVulnScanner, nil)", vulnScanner, err)
	}

	publisher, err := r.ResolveArtifactor(providers.Ref{Name: "rust-container", Version: "1"}, providers.Values{
		"ref":   interp.NewString("ghcr.io/acme/api-rust"),
		"creds": interp.NewSecret(&dagger.Secret{}),
	})
	if err != nil || publisher == nil {
		t.Fatalf("ResolveArtifactor(rust-container) = (%v, %v), want (non-nil rust.ContainerPublisher, nil)", publisher, err)
	}

	// "container" (golang.ContainerPublisher's ref) must remain unaffected
	// by the rust registrations above, and rust's own container ref must
	// not collide with it.
	if _, err := r.ResolveArtifactor(providers.Ref{Name: "container", Version: "1"}, providers.Values{
		"ref":   interp.NewString("ghcr.io/acme/api"),
		"creds": interp.NewSecret(&dagger.Secret{}),
	}); err != nil {
		t.Fatalf("ResolveArtifactor(container) after rust registration = %v, want nil", err)
	}
}

// A same-named provider requested via uses.provider (Module=="") must
// never resolve as though it had been requested via uses.module, and vice
// versa (design.md D-I: "Module == "" means in-repo") — Ref is keyed by
// both Name AND Module, not Name alone.
func TestRegisterDefaults_InRepoProviderNotConfusedWithSameNamedModule(t *testing.T) {
	t.Parallel()

	r := providers.NewRegistry()
	providers.RegisterDefaults(r, nil)

	_, err := r.ResolveBuilder(providers.Ref{Name: "go", Module: "go", Version: "1"}, providers.Values{})
	if err == nil {
		t.Fatal("ResolveBuilder(same Name via Module) error = nil, want unregistered — Module-qualified lookups must not fall back to the in-repo entry")
	}
}
