package app

import (
	"context"
	"errors"
	"testing"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHookManager(t *testing.T) {
	manager := NewHookManager()
	assert.NotNil(t, manager)
	assert.Implements(t, (*interfaces.HookManager)(nil), manager)
}

func TestHookManager_RegisterHook(t *testing.T) {
	manager := NewHookManager().(*HookManager)

	tests := []struct {
		name        string
		stepName    string
		hookType    interfaces.HookType
		hook        interfaces.HookFunc
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful registration",
			stepName: "test-step",
			hookType: interfaces.HookTypeBefore,
			hook:     func(_ context.Context) error { return nil },
			wantErr:  false,
		},
		{
			name:        "empty step name",
			stepName:    "",
			hookType:    interfaces.HookTypeBefore,
			hook:        func(_ context.Context) error { return nil },
			wantErr:     true,
			errContains: "step name cannot be empty",
		},
		{
			name:        "nil hook",
			stepName:    "test-step",
			hookType:    interfaces.HookTypeBefore,
			hook:        nil,
			wantErr:     true,
			errContains: "hook function cannot be nil",
		},
		{
			name:     "register multiple hooks for same step and type",
			stepName: "test-step",
			hookType: interfaces.HookTypeBefore,
			hook:     func(_ context.Context) error { return nil },
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RegisterHook(tt.stepName, tt.hookType, tt.hook)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHookManager_RegisterHook_MultipleHooks(t *testing.T) {
	manager := NewHookManager().(*HookManager)

	// Register multiple hooks for the same step and type
	hook1 := func(_ context.Context) error { return nil }
	hook2 := func(_ context.Context) error { return nil }

	err := manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook1)
	require.NoError(t, err)

	err = manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook2)
	require.NoError(t, err)

	// Verify both hooks are registered
	hooks := manager.GetHooks("test-step", interfaces.HookTypeBefore)
	assert.Len(t, hooks, 2)
}

func TestHookManager_GetHooks(t *testing.T) {
	manager := NewHookManager().(*HookManager)

	// Test getting hooks for non-existent step
	hooks := manager.GetHooks("non-existent", interfaces.HookTypeBefore)
	assert.Nil(t, hooks)

	// Register a hook
	hook := func(_ context.Context) error { return nil }
	err := manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook)
	require.NoError(t, err)

	// Test getting hooks for existing step
	hooks = manager.GetHooks("test-step", interfaces.HookTypeBefore)
	assert.Len(t, hooks, 1)

	// Test getting hooks for different hook type
	hooks = manager.GetHooks("test-step", interfaces.HookTypeAfter)
	assert.Nil(t, hooks)

	// Test that returned hooks are a copy (not the original slice)
	hooks2 := manager.GetHooks("test-step", interfaces.HookTypeBefore)
	assert.Len(t, hooks2, 1)
	// Modifying the returned slice should not affect the original
	_ = append(hooks2, func(_ context.Context) error { return nil })
	hooks3 := manager.GetHooks("test-step", interfaces.HookTypeBefore)
	assert.Len(t, hooks3, 1) // Should still be 1
}

func TestHookManager_ExecuteHooks(t *testing.T) {
	manager := NewHookManager().(*HookManager)

	// Test executing hooks when none are registered
	err := manager.ExecuteHooks(t.Context(), "test-step", interfaces.HookTypeBefore)
	require.NoError(t, err)

	// Register a successful hook
	hook1 := func(_ context.Context) error { return nil }
	err = manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook1)
	require.NoError(t, err)

	// Test executing successful hook
	err = manager.ExecuteHooks(t.Context(), "test-step", interfaces.HookTypeBefore)
	require.NoError(t, err)

	// Register a failing hook
	hook2 := func(_ context.Context) error { return errors.New("hook error") }
	err = manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook2)
	require.NoError(t, err)

	// Test executing hooks with failure
	err = manager.ExecuteHooks(t.Context(), "test-step", interfaces.HookTypeBefore)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook 1 for step test-step (before) failed")
}

func TestHookManager_ExecuteHooks_MultipleHooks(t *testing.T) {
	manager := NewHookManager().(*HookManager)

	// Register multiple hooks
	hook1 := func(_ context.Context) error { return nil }
	hook2 := func(_ context.Context) error { return nil }
	hook3 := func(_ context.Context) error { return errors.New("third hook error") }

	err := manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook1)
	require.NoError(t, err)
	err = manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook2)
	require.NoError(t, err)
	err = manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook3)
	require.NoError(t, err)

	// Test executing multiple hooks (should fail on third)
	err = manager.ExecuteHooks(t.Context(), "test-step", interfaces.HookTypeBefore)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook 2 for step test-step (before) failed")
}

func TestHookManager_RemoveHook(t *testing.T) {
	manager := NewHookManager().(*HookManager)

	// Test removing hook from non-existent step
	hook := func(_ context.Context) error { return nil }
	err := manager.RemoveHook("non-existent", interfaces.HookTypeBefore, hook)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no hooks found for step: non-existent")

	// Register a hook
	err = manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook)
	require.NoError(t, err)

	// Test removing hook from non-existent hook type
	err = manager.RemoveHook("test-step", interfaces.HookTypeAfter, hook)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no hooks found for step test-step and type after")

	// Test removing non-existent hook
	differentHook := func(_ context.Context) error { return nil }
	err = manager.RemoveHook("test-step", interfaces.HookTypeBefore, differentHook)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook not found for step test-step and type before")

	// Test removing existing hook
	err = manager.RemoveHook("test-step", interfaces.HookTypeBefore, hook)
	require.NoError(t, err)

	// Verify hook is removed
	hooks := manager.GetHooks("test-step", interfaces.HookTypeBefore)
	assert.Empty(t, hooks)
}

func TestHookManager_RemoveHook_MultipleHooks(t *testing.T) {
	manager := NewHookManager().(*HookManager)

	// Register multiple hooks
	hook1 := func(_ context.Context) error { return nil }
	hook2 := func(_ context.Context) error { return nil }
	hook3 := func(_ context.Context) error { return nil }

	err := manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook1)
	require.NoError(t, err)
	err = manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook2)
	require.NoError(t, err)
	err = manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook3)
	require.NoError(t, err)

	// Verify all hooks are registered
	hooks := manager.GetHooks("test-step", interfaces.HookTypeBefore)
	assert.Len(t, hooks, 3)

	// Remove middle hook
	err = manager.RemoveHook("test-step", interfaces.HookTypeBefore, hook2)
	require.NoError(t, err)

	// Verify only 2 hooks remain
	hooks = manager.GetHooks("test-step", interfaces.HookTypeBefore)
	assert.Len(t, hooks, 2)
}

func TestHookManager_ListHooks(t *testing.T) {
	manager := NewHookManager().(*HookManager)

	// Test with no hooks
	hooks := manager.ListHooks()
	assert.Empty(t, hooks)

	// Register hooks for different steps and types
	hook1 := func(_ context.Context) error { return nil }
	hook2 := func(_ context.Context) error { return nil }
	hook3 := func(_ context.Context) error { return nil }

	err := manager.RegisterHook("step1", interfaces.HookTypeBefore, hook1)
	require.NoError(t, err)
	err = manager.RegisterHook("step1", interfaces.HookTypeAfter, hook2)
	require.NoError(t, err)
	err = manager.RegisterHook("step2", interfaces.HookTypeBefore, hook3)
	require.NoError(t, err)

	// Test ListHooks
	hooks = manager.ListHooks()
	assert.Len(t, hooks, 2)
	assert.Contains(t, hooks, "step1")
	assert.Contains(t, hooks, "step2")
	assert.Equal(t, 1, hooks["step1"][interfaces.HookTypeBefore])
	assert.Equal(t, 1, hooks["step1"][interfaces.HookTypeAfter])
	assert.Equal(t, 1, hooks["step2"][interfaces.HookTypeBefore])
}

func TestHookManager_ClearHooks(t *testing.T) {
	manager := NewHookManager().(*HookManager)

	// Register hooks for a step
	hook1 := func(_ context.Context) error { return nil }
	hook2 := func(_ context.Context) error { return nil }

	err := manager.RegisterHook("test-step", interfaces.HookTypeBefore, hook1)
	require.NoError(t, err)
	err = manager.RegisterHook("test-step", interfaces.HookTypeAfter, hook2)
	require.NoError(t, err)

	// Verify hooks are registered
	hooks := manager.ListHooks()
	assert.Contains(t, hooks, "test-step")

	// Clear hooks for the step
	manager.ClearHooks("test-step")

	// Verify hooks are cleared
	hooks = manager.ListHooks()
	assert.NotContains(t, hooks, "test-step")
}

func TestHookManager_ClearAllHooks(t *testing.T) {
	manager := NewHookManager().(*HookManager)

	// Register hooks for multiple steps
	hook1 := func(_ context.Context) error { return nil }
	hook2 := func(_ context.Context) error { return nil }

	err := manager.RegisterHook("step1", interfaces.HookTypeBefore, hook1)
	require.NoError(t, err)
	err = manager.RegisterHook("step2", interfaces.HookTypeAfter, hook2)
	require.NoError(t, err)

	// Verify hooks are registered
	hooks := manager.ListHooks()
	assert.Len(t, hooks, 2)

	// Clear all hooks
	manager.ClearAllHooks()

	// Verify all hooks are cleared
	hooks = manager.ListHooks()
	assert.Empty(t, hooks)
}

func TestNewHookExecutor(t *testing.T) {
	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)
	assert.NotNil(t, executor)
	assert.Equal(t, mockHookManager, executor.hookManager)
	assert.Equal(t, mockLogger, executor.logger)
}

func TestHookExecutor_ExecuteBeforeHooks(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}

	// Test ExecuteBeforeHooks
	err := executor.ExecuteBeforeHooks(t.Context(), "test-step")
	require.NoError(t, err)
}

func TestHookExecutor_ExecuteBeforeHooks_Error(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, hookType interfaces.HookType) error {
		if hookType == interfaces.HookTypeBefore {
			return errors.New("hook error")
		}
		return nil
	}

	// Test ExecuteBeforeHooks with error
	err := executor.ExecuteBeforeHooks(t.Context(), "test-step")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook error")
}

func TestHookExecutor_ExecuteAfterHooks(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}

	// Test ExecuteAfterHooks
	err := executor.ExecuteAfterHooks(t.Context(), "test-step")
	require.NoError(t, err)
}

func TestHookExecutor_ExecuteAfterHooks_Error(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, hookType interfaces.HookType) error {
		if hookType == interfaces.HookTypeAfter {
			return errors.New("hook error")
		}
		return nil
	}

	// Test ExecuteAfterHooks with error
	err := executor.ExecuteAfterHooks(t.Context(), "test-step")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook error")
}

func TestHookExecutor_ExecuteErrorHooks(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}

	// Test ExecuteErrorHooks
	err := executor.ExecuteErrorHooks(t.Context(), "test-step")
	require.NoError(t, err)
}

func TestHookExecutor_ExecuteErrorHooks_Error(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, hookType interfaces.HookType) error {
		if hookType == interfaces.HookTypeError {
			return errors.New("hook error")
		}
		return nil
	}

	// Test ExecuteErrorHooks with error
	err := executor.ExecuteErrorHooks(t.Context(), "test-step")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook error")
}

func TestHookExecutor_ExecuteSuccessHooks(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}

	// Test ExecuteSuccessHooks
	err := executor.ExecuteSuccessHooks(t.Context(), "test-step")
	require.NoError(t, err)
}

func TestHookExecutor_ExecuteSuccessHooks_Error(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, hookType interfaces.HookType) error {
		if hookType == interfaces.HookTypeSuccess {
			return errors.New("hook error")
		}
		return nil
	}

	// Test ExecuteSuccessHooks with error
	err := executor.ExecuteSuccessHooks(t.Context(), "test-step")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook error")
}

func TestHookExecutor_ExecuteStepWithHooks_Success(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations for successful execution
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}

	// Test ExecuteStepWithHooks with successful step
	stepFunc := func() error { return nil }
	err := executor.ExecuteStepWithHooks(t.Context(), "test-step", stepFunc)
	require.NoError(t, err)
}

func TestHookExecutor_ExecuteStepWithHooks_StepError(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations for failed step execution
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}

	// Test ExecuteStepWithHooks with failed step
	stepFunc := func() error { return errors.New("step error") }
	err := executor.ExecuteStepWithHooks(t.Context(), "test-step", stepFunc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step error")
}

func TestHookExecutor_ExecuteStepWithHooks_BeforeHooksError(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations for before hooks error
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockLogger.ErrorFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, hookType interfaces.HookType) error {
		if hookType == interfaces.HookTypeBefore {
			return errors.New("before hooks error")
		}
		return nil
	}

	// Test ExecuteStepWithHooks with before hooks error
	stepFunc := func() error { return nil }
	err := executor.ExecuteStepWithHooks(t.Context(), "test-step", stepFunc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "before hooks failed for step test-step")
}

func TestHookExecutor_ExecuteStepWithHooks_AfterHooksError(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations for after hooks error
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockLogger.ErrorFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, hookType interfaces.HookType) error {
		if hookType == interfaces.HookTypeAfter {
			return errors.New("after hooks error")
		}
		return nil
	}

	// Test ExecuteStepWithHooks with after hooks error
	stepFunc := func() error { return nil }
	err := executor.ExecuteStepWithHooks(t.Context(), "test-step", stepFunc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "after hooks failed for step test-step")
}

func TestHookExecutor_ExecuteStepWithHooks_ErrorHooksError(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations for error hooks error (should not fail the execution)
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockLogger.ErrorFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, hookType interfaces.HookType) error {
		if hookType == interfaces.HookTypeError {
			return errors.New("error hooks error")
		}
		return nil
	}

	// Test ExecuteStepWithHooks with error hooks error
	stepFunc := func() error { return errors.New("step error") }
	err := executor.ExecuteStepWithHooks(t.Context(), "test-step", stepFunc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step error") // Should return step error, not hook error
}

func TestHookExecutor_ExecuteStepWithHooks_SuccessHooksError(t *testing.T) {

	mockHookManager := NewMockHookManager()
	mockLogger := NewMockLogger()

	executor := NewHookExecutor(mockHookManager, mockLogger)

	// Set up expectations for success hooks error (should not fail the execution)
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockLogger.ErrorFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, hookType interfaces.HookType) error {
		if hookType == interfaces.HookTypeSuccess {
			return errors.New("success hooks error")
		}
		return nil
	}
	mockLogger.DebugFunc = func(_ string, _ ...any) {}
	mockHookManager.ExecuteHooksFunc = func(_ context.Context, _ string, _ interfaces.HookType) error {
		return nil
	}

	// Test ExecuteStepWithHooks with success hooks error
	stepFunc := func() error { return nil }
	err := executor.ExecuteStepWithHooks(t.Context(), "test-step", stepFunc)
	require.NoError(t, err) // Should not fail due to success hooks error
}
