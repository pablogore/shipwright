package rust

import (
	"context"
	"errors"
	"fmt"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/containerutil"
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

	entrypoint := containerutil.ComputeEntrypoint(p.Config.BinaryName)

	staged := p.Client.Container().
		From(defaultPublishBaseImage).
		WithDirectory("/app", build)

	// PR #176 review finding #7: Config.BinaryName here and the paired
	// RustBuilder's Config.BinaryName are independently configured
	// manifest fields with no cross-validation, so a mismatch previously
	// surfaced only as chmod's opaque "no such file or directory". Fail
	// with an actionable message before that happens.
	if _, err := staged.WithExec([]string{"test", "-f", entrypoint}).Sync(ctx); err != nil {
		return "", fmt.Errorf(
			"containerpublisher: expected binary at %q not found in container — check that binaryName matches the value used by the paired builder step: %w",
			entrypoint, err,
		)
	}

	image := staged.
		WithExec([]string{"chmod", "+x", entrypoint}).
		WithEntrypoint([]string{entrypoint})

	if creds != nil {
		if p.Config.RegistryUser == "" {
			return "", errors.New("containerpublisher: registry credentials provided but RegistryUser is not configured")
		}
		image = image.WithRegistryAuth(containerutil.RegistryHost(ref), p.Config.RegistryUser, creds)
	}

	publishedRef, err := image.Publish(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("containerpublisher: failed to publish image: %w", err)
	}

	return publishedRef, nil
}
