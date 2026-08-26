package main

import (
	"context"
	"errors"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/internal/app"
	"github.com/pablogore/shipwright/internal/workflow/providers"
)

// fakePluginLifecycle records the load/cleanup calls runWithPluginLifecycle
// makes. A hand-rolled stub (testing-tdd skill's double order 4) is the right
// double here: pluginLifecycle is a two-method, main-package-local interface
// used by exactly one function.
type fakePluginLifecycle struct {
	loadErr    error
	cleanupErr error

	loadCalls    int
	cleanupCalls int
	gotRegistry  *providers.Registry
	gotClient    *dagger.Client
}

func (f *fakePluginLifecycle) LoadAndInitializePlugins(_ context.Context, reg *providers.Registry, client *dagger.Client) error {
	f.loadCalls++
	f.gotRegistry = reg
	f.gotClient = client
	return f.loadErr
}

func (f *fakePluginLifecycle) CleanupPlugins(context.Context) error {
	f.cleanupCalls++
	return f.cleanupErr
}

// TestRunWithPluginLifecycle_CleansUpAfterSuccess is the baseline: the happy
// path still releases plugin resources.
func TestRunWithPluginLifecycle_CleansUpAfterSuccess(t *testing.T) {
	lc := &fakePluginLifecycle{}
	reg := providers.NewRegistry()
	ran := false

	err := runWithPluginLifecycle(context.Background(), lc, reg, nil, func() error {
		ran = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, ran)
	assert.Equal(t, 1, lc.loadCalls)
	assert.Equal(t, 1, lc.cleanupCalls)
	assert.Same(t, reg, lc.gotRegistry,
		"plugins must be wired into the SAME registry the engine resolves against")
}

// TestRunWithPluginLifecycle_CleansUpAfterRunFailure is the RED test the PR
// review asked for explicitly: cleanup must run through `defer`, so a failing
// workflow still releases plugin resources instead of leaking them on the
// error path.
func TestRunWithPluginLifecycle_CleansUpAfterRunFailure(t *testing.T) {
	lc := &fakePluginLifecycle{}
	runErr := errors.New("step failed")

	err := runWithPluginLifecycle(context.Background(), lc, providers.NewRegistry(), nil, func() error {
		return runErr
	})

	require.ErrorIs(t, err, runErr)
	assert.Equal(t, 1, lc.cleanupCalls, "cleanup must fire on the failure path too, not only on success")
}

// TestRunWithPluginLifecycle_CleansUpAfterLoadFailure covers partial
// initialization: LoadBuiltinPlugins can initialize plugin A and then fail on
// plugin B, so A still needs cleanup. Deferring cleanup BEFORE the load error
// check is what makes that true.
func TestRunWithPluginLifecycle_CleansUpAfterLoadFailure(t *testing.T) {
	loadErr := errors.New("plugin init failed")
	lc := &fakePluginLifecycle{loadErr: loadErr}
	ran := false

	err := runWithPluginLifecycle(context.Background(), lc, providers.NewRegistry(), nil, func() error {
		ran = true
		return nil
	})

	require.ErrorIs(t, err, loadErr)
	assert.False(t, ran, "the workflow must not run when plugin initialization failed")
	assert.Equal(t, 1, lc.cleanupCalls, "a partially initialized plugin set still needs cleanup")
}

// TestRunWithPluginLifecycle_CleanupFailureDoesNotMaskRunResult keeps the
// caller's real outcome authoritative — a cleanup warning must never turn a
// successful run into a failure, nor overwrite the run's own error.
func TestRunWithPluginLifecycle_CleanupFailureDoesNotMaskRunResult(t *testing.T) {
	lc := &fakePluginLifecycle{cleanupErr: errors.New("cleanup boom")}

	require.NoError(t, runWithPluginLifecycle(context.Background(), lc, providers.NewRegistry(), nil, func() error {
		return nil
	}))

	runErr := errors.New("step failed")
	err := runWithPluginLifecycle(context.Background(), lc, providers.NewRegistry(), nil, func() error {
		return runErr
	})
	require.ErrorIs(t, err, runErr)
}

// pluginLifecycleConformance is the compile-level assertion that the real
// *app.App is what executeWorkflow actually passes — without it the fake
// above could drift away from the production implementation silently.
var _ pluginLifecycle = (*app.App)(nil)
