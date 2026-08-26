// Package main is Shipwright's Layer 2 Dagger projection (design.md D-A/D-B).
// It re-declares the same five capabilities from Layer 1 (pkg/shipwright) as
// Dagger Interfaces, and composes them via Plan, a Dagger Object carrying
// interface-typed chaining state. This is a deliberate duplication, not a
// shared import: .dagger/ is its own Go module (Dagger-generated go.mod)
// and cannot import github.com/pablogore/shipwright/internal/**, and
// DaggerObject only exists inside this generated package. Thin adapters
// only — no business logic beyond straightforward sequential composition.
//
// Signature rule (D-A, binding): every Dagger Interface method uses ONLY
// Dagger core types (*dagger.Directory, *dagger.File, *dagger.Container,
// *dagger.Secret) and scalars/context.Context/error — never a module-defined
// Object type in a method signature. Zero Go generics.
//
// Spike finding (task 2.1, this work unit): Dagger v0.21.8's Go SDK codegen
// cannot generate a compiling client-proxy for a Dagger Interface method
// that returns a lazy-chainable Dagger core type (*dagger.Directory,
// *dagger.File, *dagger.Container) together with an explicit `error` — the
// generated dagger.gen.go emits `return (&dagger.Directory{}).WithGraphQLQuery(q)`
// for a method whose signature demands two return values, which fails to
// compile ("not enough return values"). Verified reproducible in an isolated
// spike module and isolated to lazy-chainable Dagger core return types only:
// a second interface method returning (string, error) compiled cleanly.
// Builder.Build, Tester.Test, and Runner.Run below therefore drop the
// `error` return (matching Dagger's own lazy-chainable idiom, where errors
// surface at the terminal/scalar call, e.g. Execute below or a caller's
// Sync()) — this is a signature-level deviation from design.md D-A's literal
// example code, not from the D-A decision itself (Dagger Interfaces +
// Objects), which the spike otherwise confirmed: interface-typed Object
// field state DOES survive v0.21.8 serialization once past this codegen
// limitation (see spike verdict in apply-progress for the full transcript).
package main

import (
	"context"
	"fmt"

	"dagger/shipwright/internal/dagger"
)

// Builder builds a source Directory into a build-output Directory.
type Builder interface {
	dagger.DaggerObject
	Build(ctx context.Context, source *dagger.Directory) *dagger.Directory
}

// Tester runs tests against a build-output Directory and returns a report
// File. Multiple independent Tester implementations MAY exist for the same
// input; none is privileged.
type Tester interface {
	dagger.DaggerObject
	Test(ctx context.Context, source *dagger.Directory) *dagger.File
}

// Artifactor publishes a build-output Directory as a versioned artifact and
// returns its resolved reference.
type Artifactor interface {
	dagger.DaggerObject
	Publish(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error)
}

// Deployer deploys a previously published artifact reference into a named
// environment and returns a deployment result reference.
type Deployer interface {
	dagger.DaggerObject
	Deploy(ctx context.Context, artifactRef, environment string, creds *dagger.Secret) (string, error)
}

// Runner runs a build-output Directory as a live Container.
type Runner interface {
	dagger.DaggerObject
	Run(ctx context.Context, build *dagger.Directory) *dagger.Container
}

// Shipwright is the module's entrypoint Object.
type Shipwright struct{}

// ContractVersion mirrors pkg/shipwright.ContractVersion (D-E). Layer 2
// cannot import Layer 1 (see package doc), so this is a duplicated literal,
// not a shared constant. Keep it equal to pkg/shipwright.ContractVersion by
// hand — a bump here without the matching bump there fails
// pkg/shipwright's TestContractVersion_MatchesDaggerLayer2Literal, a
// root-module test that parses this file's source text (no import, no
// live engine) and asserts the two literals match.
func (m *Shipwright) ContractVersion() string {
	return "1.0.0"
}

// Plan starts a composition with only a Source, no capability chained yet.
func (m *Shipwright) Plan(source *dagger.Directory) *Plan {
	return &Plan{Source: source}
}

// Plan is a Dagger Object carrying interface-typed chaining state — the
// exact shape proven by this work unit's spike (task 2.1). Fallback (D-A,
// task 2.2) if a future engine version regresses this: collapse to one flat
// Function Plan(ctx, source, build, test, artifact, deploy, run).
type Plan struct {
	Source *dagger.Directory

	Build Builder
	Test  Tester

	Artifact      Artifactor
	ArtifactRef   string
	ArtifactCreds *dagger.Secret

	Deploy      Deployer
	Environment string
	DeployCreds *dagger.Secret

	Run Runner
}

// WithBuild chains a Builder into the Plan's state.
func (p *Plan) WithBuild(b Builder) *Plan {
	p.Build = b
	return p
}

// WithTest chains a Tester into the Plan's state.
func (p *Plan) WithTest(t Tester) *Plan {
	p.Test = t
	return p
}

// WithArtifact chains an Artifactor and the publish parameters its Publish
// method requires into the Plan's state.
func (p *Plan) WithArtifact(a Artifactor, ref string, creds *dagger.Secret) *Plan {
	p.Artifact = a
	p.ArtifactRef = ref
	p.ArtifactCreds = creds
	return p
}

// WithDeploy chains a Deployer and the deploy parameters its Deploy method
// requires into the Plan's state.
func (p *Plan) WithDeploy(d Deployer, environment string, creds *dagger.Secret) *Plan {
	p.Deploy = d
	p.Environment = environment
	p.DeployCreds = creds
	return p
}

// WithRun chains a Runner into the Plan's state.
func (p *Plan) WithRun(r Runner) *Plan {
	p.Run = r
	return p
}

// Execute runs every chained capability in sequence over the build output,
// thin adapter composition only: Build (if set) transforms Source, Test (if
// set) runs against the build output, Artifact (if set) publishes it and
// becomes the running result, Deploy (if set) deploys that result, Run (if
// set) runs the build output as a container. Returns the last artifact or
// deployment reference produced, or an empty string if neither Artifact nor
// Deploy was chained.
func (p *Plan) Execute(ctx context.Context) (string, error) {
	output := p.Source
	if p.Build != nil {
		output = p.Build.Build(ctx, output)
		// Build returns a lazy *dagger.Directory: Dagger does not execute
		// the underlying container operations until something forces
		// evaluation (Sync, Entries, or being passed into another synced
		// call). Without this unconditional Sync, a Plan chaining only
		// WithBuild + WithDeploy (no WithTest/WithArtifact/WithRun) never
		// touches `output` again — Deploy only receives the scalar
		// `result` string, not the Directory — so a failing Build would
		// silently never surface its error and Execute could return
		// success (PR #148 review finding 1). Sync unconditionally,
		// regardless of what's chained after Build, so a Build failure
		// always surfaces at the point Build actually happened.
		if _, err := output.Sync(ctx); err != nil {
			return "", fmt.Errorf("build failed: %w", err)
		}
	}

	if p.Test != nil {
		if _, err := p.Test.Test(ctx, output).Sync(ctx); err != nil {
			return "", err
		}
	}

	result := ""
	if p.Artifact != nil {
		ref, err := p.Artifact.Publish(ctx, output, p.ArtifactRef, p.ArtifactCreds)
		if err != nil {
			return "", err
		}
		result = ref
	}

	if p.Deploy != nil {
		// Fail closed (design.md Threat Matrix): Deploy must never run
		// with an empty artifactRef just because WithArtifact wasn't
		// chained. This is a pure Go control-flow guard — checked before
		// any interface method call (PR #148 review finding 2).
		if p.Artifact == nil {
			return "", fmt.Errorf("deploy requires an artifact to be chained first (WithArtifact before WithDeploy)")
		}

		deployed, err := p.Deploy.Deploy(ctx, result, p.Environment, p.DeployCreds)
		if err != nil {
			return "", err
		}
		result = deployed
	}

	if p.Run != nil {
		if _, err := p.Run.Run(ctx, output).Sync(ctx); err != nil {
			return "", err
		}
	}

	return result, nil
}
