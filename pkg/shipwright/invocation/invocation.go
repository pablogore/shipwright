// Package invocation carries the identity of the step currently being
// dispatched through context.Context, so that Layer 1 providers (which only
// ever see ctx and Values — never manifest.Step) can differentiate
// per-invocation resources, such as Dagger service graphs that would
// otherwise be content-addressed to the same node when two steps build an
// identical container definition concurrently (see
// providers/rust/dockerdaemon.go).
//
// This lives under pkg/shipwright (the public capability contract module
// boundary), not internal/: providers/rust and providers/go are separately
// versioned Go modules that only ever depend on shipwright's public API —
// they cannot import internal/... packages of the root module — so any type
// shared between the engine and a provider must live here.
package invocation

import "context"

type stepIDKey struct{}

// WithStepID returns a context carrying id as the current step's identity.
func WithStepID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, stepIDKey{}, id)
}

// StepID returns the step identity carried by ctx, if any.
func StepID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(stepIDKey{}).(string)
	return id, ok
}
