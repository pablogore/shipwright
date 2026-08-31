package golang

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/containerutil"
	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// defaultPublishBaseImage is the minimal runtime image for a compiled Go
// binary, matching the legacy pipeline's choice of Alpine
// (internal/pipelines/go-service/pipeline.go's Package method) but pinned
// to an immutable digest instead of the mutable "latest" tag: the same
// Shipwright revision must always publish against the same base image
// (production-readiness P0, supply-chain reproducibility).
//
// This is the single source of truth for the pinned reference — do not
// duplicate the digest elsewhere. The digest is alpine:3.22's OCI image
// index (manifest list), not a single-platform manifest, so Dagger/
// BuildKit still resolves the correct per-platform manifest (amd64,
// arm64/v8, etc.) from it exactly as it would from a tag; verified via:
//
//	docker manifest inspect alpine:3.22
//	  -> mediaType: application/vnd.oci.image.index.v1+json
//	  -> includes linux/amd64 and linux/arm64/v8 platform manifests
//
// To bump the Alpine version, resolve the new index digest (e.g. `docker
// pull alpine:<version> && docker inspect --format='{{index .RepoDigests
// 0}}' alpine:<version>`) and update this constant in one place.
const defaultPublishBaseImage = "alpine:3.22@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce"

// validatePinnedImageReference reports whether ref is a digest-pinned OCI
// image reference (repo[:tag]@sha256:<64-hex-digest>) that does not carry
// a "latest" tag. It deliberately does not implement a full OCI reference
// grammar: defaultPublishBaseImage is the only caller today, a repo-owned
// constant rather than caller-supplied input, so the check only needs to
// catch the mutable-reference shapes this P0 rules out (a missing digest,
// or an explicit "latest" tag).
func validatePinnedImageReference(ref string) bool {
	repoAndTag, digest, hasDigest := strings.Cut(ref, "@")
	if !hasDigest || repoAndTag == "" {
		return false
	}

	hexDigest, hasSHA256Prefix := strings.CutPrefix(digest, "sha256:")
	if !hasSHA256Prefix || len(hexDigest) != 64 {
		return false
	}
	for _, r := range hexDigest {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}

	if lastColon := strings.LastIndex(repoAndTag, ":"); lastColon != -1 {
		tag := repoAndTag[lastColon+1:]
		if !strings.Contains(tag, "/") && strings.EqualFold(tag, "latest") {
			return false
		}
	}

	return true
}

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
