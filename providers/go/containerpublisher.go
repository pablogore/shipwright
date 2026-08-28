package golang

import (
	"context"
	"errors"
	"fmt"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/containerutil"
	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// defaultPublishBaseImage matches the legacy pipeline's minimal runtime
// image for a compiled Go binary
// (internal/pipelines/go-service/pipeline.go's Package method).
const defaultPublishBaseImage = "alpine:latest"

// ContainerPublisher packages a build-output Directory into a minimal
// container image and publishes it to a registry. Extracted from the
// legacy go-service pipeline's Package / Tag / Push logic
// (internal/pipelines/go-service/pipeline.go).
//
// Behavioral judgment calls:
//   - The legacy Tag step generated a tag from TAG_NAME or `git
//     rev-parse` and wrote it to a host-local .tag_name file — a host
//     filesystem side effect that has no place in a capability contract
//     whose only inputs/outputs are Dagger core types (design.md D-A).
//     Under the new manifest model the caller supplies the fully-resolved
//     image reference, tag included, via the ref parameter (e.g.
//     ${{ variables.imageRef }}); tag generation becomes the caller's
//     concern, not this capability's.
//   - Artifactor.Publish's signature (pkg/shipwright, shipped in WU1) has
//     no username parameter, only a ref and a *dagger.Secret. Config.
//     RegistryUser (shipwright.ArtifactConfig, also shipped in WU1)
//     supplies that missing piece, preserving the legacy split between a
//     configured username and a secret credential.
//   - shipwright.ArtifactConfig.BinaryName carries the binary filename
//     across the Build -> Artifact boundary (a manifest-level `with`
//     binding, register.go), computed via containerutil.ComputeEntrypoint.
//     Left empty, this type assumes the build Directory contains a single
//     binary named "app" — the same default GoBuilder uses when its own
//     BuildConfig.BinaryName is left empty. A caller that sets a
//     non-default BuildConfig.BinaryName on GoBuilder MUST set the
//     matching ArtifactConfig.BinaryName here, or Publish now fails fast
//     with an actionable error instead of a bare chmod failure (PR #176
//     review finding #7).
type ContainerPublisher struct {
	// Client is the Dagger client used to construct the runtime image.
	Client daggerkit.DaggerClient
	// Config carries the registry username used alongside the creds
	// secret passed to Publish.
	Config shipwright.ArtifactConfig
}

// Compile-time conformance assertion (tasks.md 3.5): ContainerPublisher
// must satisfy Layer 1's Artifactor interface.
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
		WithDirectory("/app", daggerkit.NewDaggerDirectoryAdapter(build))

	// PR #176 review finding #7: Config.BinaryName here and the paired
	// Builder's BuildConfig.BinaryName are independently configured
	// manifest fields with no cross-validation, so a mismatch previously
	// surfaced only as chmod's opaque "no such file or directory". Fail
	// with an actionable message before that happens.
	if _, err := staged.WithExec([]string{"test", "-f", entrypoint}, daggerkit.DaggerContainerWithExecOpts{}).Sync(ctx); err != nil {
		return "", fmt.Errorf(
			"containerpublisher: expected binary at %q not found in container — check that binaryName matches the value used by the paired builder step: %w",
			entrypoint, err,
		)
	}

	image := staged.
		WithExec([]string{"chmod", "+x", entrypoint}, daggerkit.DaggerContainerWithExecOpts{}).
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
