// Package shipwright defines Shipwright's public, versioned capability
// contract (Layer 1). Every capability interface here is a plain exported
// Go interface — no generic type parameters, no module-defined Object types
// in method signatures — so it can be projected onto Dagger's module type
// system (see .dagger/, Layer 2) and consumed through generated
// cross-language SDK bindings. Signature rule (design.md D-A, binding per
// the proposal's Dagger type-system constraint): every method uses only
// Dagger core types (*dagger.Directory, *dagger.File, *dagger.Container,
// *dagger.Secret), Go scalars, context.Context, and error.
package shipwright

import (
	"context"

	"dagger.io/dagger"
)

// Builder builds a source Directory into a build-output Directory. It has no
// knowledge of Test, Artifact, Deploy, or Run — capabilities compose, they
// do not depend on siblings.
type Builder interface {
	Build(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error)
}

// Tester runs tests against a build-output Directory and returns a report
// File. Multiple independent Tester implementations MAY exist for the same
// input (unit, lint, vulnerability scan, ...); none is privileged.
type Tester interface {
	Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error)
}

// Artifactor publishes a build-output Directory as a versioned artifact and
// returns its resolved reference (for example an image reference).
type Artifactor interface {
	Publish(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error)
}

// Deployer deploys a previously published artifact reference into a named
// environment and returns a deployment result reference.
type Deployer interface {
	Deploy(ctx context.Context, artifactRef, environment string, creds *dagger.Secret) (string, error)
}

// Runner runs a build-output Directory as a live Container, for example to
// execute it locally or expose it for interactive inspection.
type Runner interface {
	Run(ctx context.Context, build *dagger.Directory) (*dagger.Container, error)
}
