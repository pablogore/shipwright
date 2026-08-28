package providers

import (
	"strconv"
	"strings"

	"dagger.io/dagger"

	golang "github.com/pablogore/shipwright/providers/go"
	godaggerkit "github.com/pablogore/shipwright/providers/go/daggerkit"
	"github.com/pablogore/shipwright/providers/rust"
	rustdaggerkit "github.com/pablogore/shipwright/providers/rust/daggerkit"

	"github.com/pablogore/shipwright/internal/daggerkit"
	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// RegisterDefaults registers Shipwright's five in-repo capability
// implementations (providers/go, extracted from the former
// internal/capabilities, WU3) into r under the provider names design.md's
// own example manifest uses ("go", "go-test", "golangci-lint",
// "govulncheck", "container"), all pinned to version "1". This is the ONLY
// place in the tree that wires a manifest-facing provider name to a
// concrete Go type — the Registry itself has no knowledge of any
// provider's identity, and no provider registered here (or anywhere else,
// see security_test.go) is loaded from a path, fetched, or dynamically
// opened via plugin.Open (design.md D-I).
//
// client is threaded through to every capability's Client field; it MAY
// be nil in a test that only exercises resolution (never actually calling
// a capability's method against a real Dagger engine) — every WU3
// capability's own method already guards against a nil Client
// (providers/go/gobuilder.go etc.) and returns an error rather than
// panicking, so registration itself never needs a live client.
func RegisterDefaults(r *Registry, client *dagger.Client) {
	goClient := newGoDaggerClient(client)
	rustClient := newRustDaggerClient(client)

	r.RegisterBuilder(Ref{Name: "go", Version: "1"}, WithSchema{
		"goVersion":  interp.KindString,
		"binaryName": interp.KindString,
	}, func(v Values) shipwright.Builder {
		return &golang.GoBuilder{
			Client: goClient,
			Config: shipwright.BuildConfig{
				GoVersion:  stringField(v, "goVersion"),
				BinaryName: stringField(v, "binaryName"),
			},
		}
	})

	r.RegisterTester(Ref{Name: "go-test", Version: "1"}, WithSchema{
		"coverage": interp.KindInt,
	}, func(v Values) shipwright.Tester {
		return &golang.GoUnitTester{
			Client: goClient,
			Config: shipwright.TestConfig{Coverage: floatField(v, "coverage")},
		}
	})

	r.RegisterTester(Ref{Name: "golangci-lint", Version: "1"}, WithSchema{}, func(v Values) shipwright.Tester {
		return &golang.GoLinter{Client: goClient}
	})

	r.RegisterTester(Ref{Name: "govulncheck", Version: "1"}, WithSchema{}, func(v Values) shipwright.Tester {
		return &golang.GoVulnScanner{Client: goClient}
	})

	r.RegisterArtifactor(Ref{Name: "container", Version: "1"}, WithSchema{
		"ref":          interp.KindString,
		"creds":        interp.KindSecret,
		"registryUser": interp.KindString,
		"binaryName":   interp.KindString,
	}, func(v Values) shipwright.Artifactor {
		return &golang.ContainerPublisher{
			Client: goClient,
			Config: shipwright.ArtifactConfig{
				RegistryUser: stringField(v, "registryUser"),
				BinaryName:   stringField(v, "binaryName"),
			},
		}
	})

	r.RegisterBuilder(Ref{Name: "rust", Version: "1"}, WithSchema{
		"binaryName":   interp.KindString,
		"buildMode":    interp.KindString,
		"rustVersion":  interp.KindString,
		"manifestPath": interp.KindString,
		"package":      interp.KindString,
		"bin":          interp.KindString,
		"locked":       interp.KindBool,
	}, func(v Values) shipwright.Builder {
		return &rust.RustBuilder{
			Client: rustClient,
			Config: shipwright.BuildConfig{
				BinaryName: stringField(v, "binaryName"),
				BuildMode:  stringField(v, "buildMode"),
			},
			RustVersion:  stringField(v, "rustVersion"),
			ManifestPath: stringField(v, "manifestPath"),
			Package:      stringField(v, "package"),
			Bin:          stringField(v, "bin"),
			Locked:       boolField(v, "locked"),
		}
	})

	r.RegisterTester(Ref{Name: "rust-test", Version: "1"}, WithSchema{
		"coverage":     interp.KindInt,
		"rustVersion":  interp.KindString,
		"manifestPath": interp.KindString,
		"package":      interp.KindString,
		"features":     interp.KindString,
		"allFeatures":  interp.KindBool,
		"locked":       interp.KindBool,
	}, func(v Values) shipwright.Tester {
		return &rust.RustUnitTester{
			Client:       rustClient,
			Config:       shipwright.TestConfig{Coverage: floatField(v, "coverage")},
			RustVersion:  stringField(v, "rustVersion"),
			ManifestPath: stringField(v, "manifestPath"),
			Package:      stringField(v, "package"),
			Features:     splitFeatures(stringField(v, "features")),
			AllFeatures:  boolField(v, "allFeatures"),
			Locked:       boolField(v, "locked"),
		}
	})

	// "rust-integration-test", registered under the same "test" capability
	// as "rust-test": runs a separate, service-dependent (Docker/
	// Testcontainers) suite via its own ManifestPath, distinguished from
	// "rust-test" purely by provider name and workflow step — see
	// providers/rust/rustintegrationtester.go's own doc comment for why
	// this is not a distinct "integration-test" capability kind.
	r.RegisterTester(Ref{Name: "rust-integration-test", Version: "1"}, WithSchema{
		"rustVersion":      interp.KindString,
		"manifestPath":     interp.KindString,
		"package":          interp.KindString,
		"features":         interp.KindString,
		"allFeatures":      interp.KindBool,
		"locked":           interp.KindBool,
		"dockerSocketPath": interp.KindString,
	}, func(v Values) shipwright.Tester {
		return &rust.RustIntegrationTester{
			Client:           rustClient,
			RustVersion:      stringField(v, "rustVersion"),
			ManifestPath:     stringField(v, "manifestPath"),
			Package:          stringField(v, "package"),
			Features:         splitFeatures(stringField(v, "features")),
			AllFeatures:      boolField(v, "allFeatures"),
			Locked:           boolField(v, "locked"),
			DockerSocketPath: stringField(v, "dockerSocketPath"),
		}
	})

	// "clippy", not "rust-lint": mirrors golangci-lint's own convention of
	// naming a Tester ref after the actual underlying tool rather than a
	// generic "<language>-lint" placeholder.
	r.RegisterTester(Ref{Name: "clippy", Version: "1"}, WithSchema{
		"rustVersion": interp.KindString,
	}, func(v Values) shipwright.Tester {
		return &rust.RustLinter{Client: rustClient, RustVersion: stringField(v, "rustVersion")}
	})

	// "cargo-audit", not "rust-vulncheck": mirrors govulncheck's own
	// convention of naming a Tester ref after the actual underlying tool.
	r.RegisterTester(Ref{Name: "cargo-audit", Version: "1"}, WithSchema{
		"rustVersion": interp.KindString,
	}, func(v Values) shipwright.Tester {
		return &rust.RustVulnScanner{Client: rustClient, RustVersion: stringField(v, "rustVersion")}
	})

	// rust.ContainerPublisher is registered under its own ref ("rust-
	// container"), NOT reusing "container" -> golang.ContainerPublisher:
	// unlike the rest of golang's capabilities, ContainerPublisher's logic
	// is otherwise generic, but its defaultPublishBaseImage is not — Rust's
	// default build output links dynamically against glibc (the official
	// rust:<version> image's toolchain), which fails to start under
	// golang.ContainerPublisher's alpine:latest (musl) base with a missing
	// ld-linux loader error. rust.ContainerPublisher's own
	// debian:bookworm-slim base exists specifically to run that binary
	// (see providers/rust/containerpublisher.go's own doc comment), so
	// reusing "container" here would silently ship broken images.
	r.RegisterArtifactor(Ref{Name: "rust-container", Version: "1"}, WithSchema{
		"ref":          interp.KindString,
		"creds":        interp.KindSecret,
		"registryUser": interp.KindString,
		"binaryName":   interp.KindString,
	}, func(v Values) shipwright.Artifactor {
		return &rust.ContainerPublisher{
			Client: rustClient,
			Config: shipwright.ArtifactConfig{
				RegistryUser: stringField(v, "registryUser"),
				BinaryName:   stringField(v, "binaryName"),
			},
		}
	})

	// ChangelogRunner (changelog.go) has no with-field configuration, same
	// as golangci-lint/govulncheck above.
	r.RegisterRunner(Ref{Name: "changelog", Version: "1"}, WithSchema{}, func(v Values) shipwright.Runner {
		return &ChangelogRunner{Client: newChangelogDaggerClient(client)}
	})
}

// newGoDaggerClient wraps client for providers/go's own daggerkit.DaggerClient
// interface, preserving a nil client as a nil interface (rather than a
// non-nil interface wrapping a nil pointer) so every provider's existing
// "Client == nil" guard clause keeps working for the nil-client,
// resolution-only path documented on RegisterDefaults above.
func newGoDaggerClient(client *dagger.Client) godaggerkit.DaggerClient {
	if client == nil {
		return nil
	}
	return godaggerkit.NewDaggerAdapter(client)
}

// newRustDaggerClient is newGoDaggerClient's counterpart for providers/rust's
// own daggerkit.DaggerClient interface.
func newRustDaggerClient(client *dagger.Client) rustdaggerkit.DaggerClient {
	if client == nil {
		return nil
	}
	return rustdaggerkit.NewDaggerAdapter(client)
}

// newChangelogDaggerClient is newGoDaggerClient's counterpart for this
// package's own ChangelogRunner, which uses the root module's
// internal/daggerkit.DaggerClient interface directly (no separate module
// boundary to cross, unlike providers/go and providers/rust).
func newChangelogDaggerClient(client *dagger.Client) daggerkit.DaggerClient {
	if client == nil {
		return nil
	}
	return daggerkit.NewDaggerAdapter(client)
}

// stringField returns the string form of v[key], or "" when key is
// absent or holds a non-string-representable (secret) value — a caller
// that needs a required field to actually be present validates that via
// WithSchema (checkWithSchema in registry.go) before this factory ever
// runs.
func stringField(v Values, key string) string {
	val, ok := v[key]
	if !ok {
		return ""
	}
	s, ok := val.String()
	if !ok {
		return ""
	}
	return s
}

// floatField parses v[key]'s string form as a float64, returning 0 when
// key is absent or unparseable. Values has no dedicated numeric accessor
// (interp.Value's single str field carries every non-secret kind's
// textual representation, interp.NewInt's own doc comment) — parsing here
// is this package's job, not interp's.
func floatField(v Values, key string) float64 {
	val, ok := v[key]
	if !ok {
		return 0
	}
	s, ok := val.String()
	if !ok {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// boolField parses v[key]'s string form as a bool, returning false when
// key is absent or unparseable, same absent/unparseable-defaults-to-zero
// convention as floatField.
func boolField(v Values, key string) bool {
	val, ok := v[key]
	if !ok {
		return false
	}
	s, ok := val.String()
	if !ok {
		return false
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return b
}

// splitFeatures splits a comma-separated "features" with-field (e.g.
// "test-kit, other-feature") into individual, trimmed feature names.
// Values has no list kind (interp.Kind is String/Int/Bool/Secret only), so
// a manifest passes multiple Cargo features as one delimited string.
func splitFeatures(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	features := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			features = append(features, p)
		}
	}
	return features
}
