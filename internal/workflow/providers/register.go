package providers

import (
	"strconv"

	"dagger.io/dagger"

	golang "github.com/pablogore/shipwright/providers/go"

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
	r.RegisterBuilder(Ref{Name: "go", Version: "1"}, WithSchema{
		"goVersion":  interp.KindString,
		"binaryName": interp.KindString,
	}, func(v Values) shipwright.Builder {
		return &golang.GoBuilder{
			Client: client,
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
			Client: client,
			Config: shipwright.TestConfig{Coverage: floatField(v, "coverage")},
		}
	})

	r.RegisterTester(Ref{Name: "golangci-lint", Version: "1"}, WithSchema{}, func(v Values) shipwright.Tester {
		return &golang.GoLinter{Client: client}
	})

	r.RegisterTester(Ref{Name: "govulncheck", Version: "1"}, WithSchema{}, func(v Values) shipwright.Tester {
		return &golang.GoVulnScanner{Client: client}
	})

	r.RegisterArtifactor(Ref{Name: "container", Version: "1"}, WithSchema{
		"ref":          interp.KindString,
		"creds":        interp.KindSecret,
		"registryUser": interp.KindString,
	}, func(v Values) shipwright.Artifactor {
		return &golang.ContainerPublisher{
			Client: client,
			Config: shipwright.ArtifactConfig{RegistryUser: stringField(v, "registryUser")},
		}
	})
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
