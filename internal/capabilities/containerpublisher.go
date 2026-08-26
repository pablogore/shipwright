package capabilities

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
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
//   - shipwright.ArtifactConfig has no binary-filename field (only
//     ImageName, the published image's name), so this type assumes the
//     build Directory contains a single binary named defaultBinaryName
//     ("app") — the same default GoBuilder uses when its own
//     BuildConfig.BinaryName is left empty. Threading a custom binary name
//     across the Build -> Artifact boundary would require a manifest-level
//     binding (Phase 7's provider `with` values) and is out of scope for
//     this purely-additive work unit.
type ContainerPublisher struct {
	// Client is the Dagger client used to construct the runtime image.
	Client *dagger.Client
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
