package rust

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// defaultPublishBaseImage is the minimal runtime image a compiled Rust
// binary is packaged into.
//
// debian:bookworm-slim, not alpine (providers/go's GoBuilder default): a
// plain `cargo build --release` inside the official rust:<version> image
// dynamically links against glibc, not musl. alpine ships musl libc, so a
// glibc-linked binary published into it fails at container startup with a
// dynamic-linker error ("No such file or directory" from the missing
// ld-linux loader) rather than running. Cross-compiling to the
// x86_64-unknown-linux-musl target to make alpine work is a real option,
// but it is a build-time decision out of scope for this default publisher;
// debian-slim is the glibc-compatible minimal-base analog that actually
// runs the binary GoBuilder-equivalent RustBuilder produces by default.
const defaultPublishBaseImage = "debian:bookworm-slim"

// ContainerPublisher packages a build-output Directory into a minimal
// container image and publishes it to a registry. Structural mirror of
// providers/go's ContainerPublisher, whose logic is otherwise generic
// (source-agnostic Dagger container packaging) and needed no Rust-specific
// change beyond the base image above and the binary path convention shared
// with RustBuilder.
type ContainerPublisher struct {
	// Client is the Dagger client used to construct the runtime image.
	Client *dagger.Client
	// Config carries the registry username used alongside the creds
	// secret passed to Publish.
	Config shipwright.ArtifactConfig
}

// Compile-time conformance assertion: ContainerPublisher must satisfy
// Layer 1's Artifactor interface.
var _ shipwright.Artifactor = (*ContainerPublisher)(nil)

// Publish packages build into a minimal container image, authenticates
// against the target registry when creds is provided, and publishes it to
// ref, returning the resolved published reference.
func (p *ContainerPublisher) Publish(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error) {
	if p.Client == nil {
		return "", errors.New("containerpublisher: dagger client is not configured")
	}
	if build == nil {
		return "", errors.New("containerpublisher: build directory is nil")
	}
	if ref == "" {
		return "", errors.New("containerpublisher: ref is empty")
	}

	entrypoint := "/app/" + defaultBinaryName

	image := p.Client.Container().
		From(defaultPublishBaseImage).
		WithDirectory("/app", build).
		WithExec([]string{"chmod", "+x", entrypoint}).
		WithEntrypoint([]string{entrypoint})

	if creds != nil {
		if p.Config.RegistryUser == "" {
			return "", errors.New("containerpublisher: registry credentials provided but RegistryUser is not configured")
		}
		image = image.WithRegistryAuth(registryHost(ref), p.Config.RegistryUser, creds)
	}

	publishedRef, err := image.Publish(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("containerpublisher: failed to publish image: %w", err)
	}

	return publishedRef, nil
}

// registryHost extracts the registry address portion of an image
// reference, e.g. "ghcr.io/acme/api:v1" -> "ghcr.io". A reference with no
// registry-looking first segment (a bare Docker Hub name such as
// "acme/api:v1") returns ref unchanged, which Dagger's WithRegistryAuth
// then treats as docker.io-relative. Pure helper, unit-testable without a
// Dagger client.
func registryHost(ref string) string {
	firstSlash := strings.Index(ref, "/")
	if firstSlash == -1 {
		return ref
	}

	host := ref[:firstSlash]
	if host != "localhost" && !strings.ContainsAny(host, ".:") {
		return ref
	}

	return host
}
