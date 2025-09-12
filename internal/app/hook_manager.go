package app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
)

// HookManager implements the HookManager interface for managing pipeline hooks.
type HookManager struct {
	hooks map[string]map[interfaces.HookType][]interfaces.HookFunc
	mutex sync.RWMutex
}

// NewHookManager creates a new hook manager.
func NewHookManager() interfaces.HookManager {
	return &HookManager{
		hooks: make(map[string]map[interfaces.HookType][]interfaces.HookFunc),
	}
}

// RegisterHook registers a hook for a specific step and hook type.
func (hm *HookManager) RegisterHook(stepName string, hookType interfaces.HookType, hook interfaces.HookFunc) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if stepName == "" {
		return errors.New("step name cannot be empty")
	}

	if hook == nil {
		return errors.New("hook function cannot be nil")
	}

	// Initialize step hooks if not exists
	if hm.hooks[stepName] == nil {
		hm.hooks[stepName] = make(map[interfaces.HookType][]interfaces.HookFunc)
	}

	// Initialize hook type slice if not exists
	if hm.hooks[stepName][hookType] == nil {
		hm.hooks[stepName][hookType] = make([]interfaces.HookFunc, 0)
	}

	// Add hook to the slice
	hm.hooks[stepName][hookType] = append(hm.hooks[stepName][hookType], hook)

	return nil
}

// GetHooks returns all hooks for a specific step and hook type.
func (hm *HookManager) GetHooks(stepName string, hookType interfaces.HookType) []interfaces.HookFunc {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	if hm.hooks[stepName] == nil {
		return nil
	}

	hooks := hm.hooks[stepName][hookType]
	if hooks == nil {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]interfaces.HookFunc, len(hooks))
	copy(result, hooks)
	return result
}

// ExecuteHooks executes all hooks for a specific step and hook type.
func (hm *HookManager) ExecuteHooks(ctx context.Context, stepName string, hookType interfaces.HookType) error {
	hooks := hm.GetHooks(stepName, hookType)

	if len(hooks) == 0 {
		return nil // No hooks to execute
	}

	for i, hook := range hooks {
		if err := hook(ctx); err != nil {
			return fmt.Errorf("hook %d for step %s (%s) failed: %w", i, stepName, hookType, err)
		}
	}

	return nil
}

// RemoveHook removes a specific hook from a step and hook type.
func (hm *HookManager) RemoveHook(stepName string, hookType interfaces.HookType, hook interfaces.HookFunc) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if hm.hooks[stepName] == nil {
		return fmt.Errorf("no hooks found for step: %s", stepName)
	}

	if hm.hooks[stepName][hookType] == nil {
		return fmt.Errorf("no hooks found for step %s and type %s", stepName, hookType)
	}

	hooks := hm.hooks[stepName][hookType]
	for i, h := range hooks {
		// Compare function pointers (this is a simple approach)
		// In a more sophisticated implementation, you might want to use function names or IDs
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", hook) {
			// Remove the hook by slicing
			hm.hooks[stepName][hookType] = append(hooks[:i], hooks[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("hook not found for step %s and type %s", stepName, hookType)
}

// ListHooks returns a map of all registered hooks.
func (hm *HookManager) ListHooks() map[string]map[interfaces.HookType]int {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	result := make(map[string]map[interfaces.HookType]int)

	for stepName, stepHooks := range hm.hooks {
		result[stepName] = make(map[interfaces.HookType]int)
		for hookType, hooks := range stepHooks {
			result[stepName][hookType] = len(hooks)
		}
	}

	return result
}

// ClearHooks removes all hooks for a specific step.
func (hm *HookManager) ClearHooks(stepName string) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	delete(hm.hooks, stepName)
}

// ClearAllHooks removes all registered hooks.
func (hm *HookManager) ClearAllHooks() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.hooks = make(map[string]map[interfaces.HookType][]interfaces.HookFunc)
}

// HookExecutor provides a convenient way to execute hooks with error handling.
type HookExecutor struct {
	hookManager interfaces.HookManager
	logger      interfaces.Logger
}

// NewHookExecutor creates a new hook executor.
func NewHookExecutor(hookManager interfaces.HookManager, logger interfaces.Logger) *HookExecutor {
	return &HookExecutor{
		hookManager: hookManager,
		logger:      logger,
	}
}

// ExecuteBeforeHooks executes all before hooks for a step.
func (he *HookExecutor) ExecuteBeforeHooks(ctx context.Context, stepName string) error {
	he.logger.Debug("Executing before hooks", "step", stepName)
	return he.hookManager.ExecuteHooks(ctx, stepName, interfaces.HookTypeBefore)
}

// ExecuteAfterHooks executes all after hooks for a step.
func (he *HookExecutor) ExecuteAfterHooks(ctx context.Context, stepName string) error {
	he.logger.Debug("Executing after hooks", "step", stepName)
	return he.hookManager.ExecuteHooks(ctx, stepName, interfaces.HookTypeAfter)
}

// ExecuteErrorHooks executes all error hooks for a step.
func (he *HookExecutor) ExecuteErrorHooks(ctx context.Context, stepName string) error {
	he.logger.Debug("Executing error hooks", "step", stepName)
	return he.hookManager.ExecuteHooks(ctx, stepName, interfaces.HookTypeError)
}

// ExecuteSuccessHooks executes all success hooks for a step.
func (he *HookExecutor) ExecuteSuccessHooks(ctx context.Context, stepName string) error {
	he.logger.Debug("Executing success hooks", "step", stepName)
	return he.hookManager.ExecuteHooks(ctx, stepName, interfaces.HookTypeSuccess)
}

// ExecuteStepWithHooks executes a step with all appropriate hooks.
func (he *HookExecutor) ExecuteStepWithHooks(ctx context.Context, stepName string, stepFunc func() error) error {
	// Execute before hooks
	if err := he.ExecuteBeforeHooks(ctx, stepName); err != nil {
		he.logger.Error("Before hooks failed", "step", stepName, "error", err)
		return fmt.Errorf("before hooks failed for step %s: %w", stepName, err)
	}

	// Execute the step
	stepErr := stepFunc()

	// Execute appropriate hooks based on step result
	if stepErr != nil {
		// Execute error hooks
		if err := he.ExecuteErrorHooks(ctx, stepName); err != nil {
			he.logger.Error("Error hooks failed", "step", stepName, "error", err)
			// Don't return this error, just log it
		}
		return stepErr
	}

	// Execute success hooks
	if err := he.ExecuteSuccessHooks(ctx, stepName); err != nil {
		he.logger.Error("Success hooks failed", "step", stepName, "error", err)
		// Don't return this error, just log it
	}

	// Execute after hooks
	if err := he.ExecuteAfterHooks(ctx, stepName); err != nil {
		he.logger.Error("After hooks failed", "step", stepName, "error", err)
		return fmt.Errorf("after hooks failed for step %s: %w", stepName, err)
	}

	return nil
}
