package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
	"github.com/getsyntegrity/syntegrity-dagger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewPipelineExecutor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager)
	assert.NotNil(t, executor)
	assert.Implements(t, (*interfaces.PipelineExecutor)(nil), executor)
}

func TestPipelineExecutor_ExecutePipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ExecutePipeline with no steps (should get execution order)
	mockStepRegistry.EXPECT().GetExecutionOrder().Return([]string{"step1", "step2"}, nil)
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockStepRegistry.EXPECT().GetStepConfig("step2").Return(interfaces.StepConfig{
		Name:    "step2",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step2", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step2").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step2", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step2", interfaces.HookTypeAfter).Return(nil)

	err := executor.ExecutePipeline(t.Context(), "test-pipeline", []string{})
	require.NoError(t, err)

	// Verify pipeline status
	status, err := executor.GetPipelineStatus("test-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "test-pipeline", status.PipelineName)
	assert.Equal(t, "completed", status.Status)
	assert.Len(t, status.Steps, 2)
}

func TestPipelineExecutor_ExecutePipeline_WithSteps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ExecutePipeline with specific steps
	steps := []string{"step1", "step2"}
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockStepRegistry.EXPECT().GetStepConfig("step2").Return(interfaces.StepConfig{
		Name:    "step2",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step2", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step2").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step2", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step2", interfaces.HookTypeAfter).Return(nil)

	err := executor.ExecutePipeline(t.Context(), "test-pipeline", steps)
	require.NoError(t, err)
}

func TestPipelineExecutor_ExecutePipeline_GetExecutionOrderError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ExecutePipeline with execution order error
	mockStepRegistry.EXPECT().GetExecutionOrder().Return(nil, errors.New("execution order error"))

	err := executor.ExecutePipeline(t.Context(), "test-pipeline", []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get execution order")

	// Verify pipeline status is failed
	status, err := executor.GetPipelineStatus("test-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "failed", status.Status)
}

func TestPipelineExecutor_ExecutePipeline_StepError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ExecutePipeline with step error
	steps := []string{"step1", "step2"}
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(errors.New("step1 error"))
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeError).Return(nil)

	err := executor.ExecutePipeline(t.Context(), "test-pipeline", steps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline test-pipeline failed at step step1")

	// Verify pipeline status is failed
	status, err := executor.GetPipelineStatus("test-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "failed", status.Status)
}

func TestPipelineExecutor_ExecutePipeline_ContextCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ExecutePipeline with cancelled context
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	steps := []string{"step1"}
	err := executor.ExecutePipeline(ctx, "test-pipeline", steps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// Verify pipeline status is cancelled
	status, err := executor.GetPipelineStatus("test-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "cancelled", status.Status)
}

func TestPipelineExecutor_ExecuteStep(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ExecuteStep
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(nil)

	err := executor.ExecuteStep(t.Context(), "test-pipeline", "step1")
	require.NoError(t, err)

	// Verify step status
	stepResult, err := executor.GetStepStatus("test-pipeline", "step1")
	require.NoError(t, err)
	assert.Equal(t, "step1", stepResult.StepName)
	assert.True(t, stepResult.Success)
}

func TestPipelineExecutor_ExecuteStep_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ExecuteStep with error
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(errors.New("step1 error"))
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeError).Return(nil)

	err := executor.ExecuteStep(t.Context(), "test-pipeline", "step1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step1 error")

	// Verify step status
	stepResult, err := executor.GetStepStatus("test-pipeline", "step1")
	require.NoError(t, err)
	assert.Equal(t, "step1", stepResult.StepName)
	assert.False(t, stepResult.Success)
	assert.Error(t, stepResult.Error)
}

func TestPipelineExecutor_ExecuteStep_GetStepConfigError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ExecuteStep with GetStepConfig error
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{}, errors.New("config error"))

	err := executor.ExecuteStep(t.Context(), "test-pipeline", "step1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config error")
}

func TestPipelineExecutor_ExecuteStep_BeforeHooksError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ExecuteStep with before hooks error
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(errors.New("before hooks error"))

	err := executor.ExecuteStep(t.Context(), "test-pipeline", "step1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "before hooks error")
}

func TestPipelineExecutor_ExecuteStep_AfterHooksError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ExecuteStep with after hooks error
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(errors.New("after hooks error"))

	err := executor.ExecuteStep(t.Context(), "test-pipeline", "step1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "after hooks error")
}

func TestPipelineExecutor_GetPipelineStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test GetPipelineStatus with non-existent pipeline
	status, err := executor.GetPipelineStatus("non-existent")
	require.Error(t, err)
	assert.Equal(t, interfaces.PipelineStatus{}, status)
	assert.Contains(t, err.Error(), "pipeline not found")

	// Execute a pipeline to create status
	mockStepRegistry.EXPECT().GetExecutionOrder().Return([]string{"step1"}, nil)
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(nil)

	err = executor.ExecutePipeline(t.Context(), "test-pipeline", []string{})
	require.NoError(t, err)

	// Test GetPipelineStatus with existing pipeline
	status, err = executor.GetPipelineStatus("test-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "test-pipeline", status.PipelineName)
	assert.Equal(t, "completed", status.Status)
	assert.NotZero(t, status.StartTime)
	assert.NotNil(t, status.EndTime)
	assert.NotZero(t, status.Duration)
	assert.Len(t, status.Steps, 1)
}

func TestPipelineExecutor_GetStepStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test GetStepStatus with non-existent pipeline
	stepResult, err := executor.GetStepStatus("non-existent", "step1")
	require.Error(t, err)
	assert.Equal(t, interfaces.StepResult{}, stepResult)
	assert.Contains(t, err.Error(), "pipeline not found")

	// Execute a step to create status
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(nil)

	err = executor.ExecuteStep(t.Context(), "test-pipeline", "step1")
	require.NoError(t, err)

	// Test GetStepStatus with existing step
	stepResult, err = executor.GetStepStatus("test-pipeline", "step1")
	require.NoError(t, err)
	assert.Equal(t, "step1", stepResult.StepName)
	assert.True(t, stepResult.Success)
	assert.NotZero(t, stepResult.Duration)

	// Test GetStepStatus with non-existent step
	stepResult, err = executor.GetStepStatus("test-pipeline", "non-existent")
	require.Error(t, err)
	assert.Equal(t, interfaces.StepResult{}, stepResult)
	assert.Contains(t, err.Error(), "step not found")
}

func TestPipelineExecutor_CancelPipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test CancelPipeline with non-existent pipeline
	err := executor.CancelPipeline("non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not found")

	// Execute a pipeline to create status
	mockStepRegistry.EXPECT().GetExecutionOrder().Return([]string{"step1"}, nil)
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(nil)

	err = executor.ExecutePipeline(t.Context(), "test-pipeline", []string{})
	require.NoError(t, err)

	// Test CancelPipeline with completed pipeline (should not change status)
	err = executor.CancelPipeline("test-pipeline")
	require.NoError(t, err)

	status, err := executor.GetPipelineStatus("test-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "completed", status.Status) // Should remain completed

	// Test CancelPipeline with running pipeline
	// Create a new pipeline status manually
	executor.mutex.Lock()
	executor.status["running-pipeline"] = &PipelineStatus{
		PipelineName: "running-pipeline",
		Status:       "running",
		StartTime:    time.Now(),
		Steps:        make(map[string]StepResult),
		Metadata:     make(map[string]any),
	}
	executor.mutex.Unlock()

	err = executor.CancelPipeline("running-pipeline")
	require.NoError(t, err)

	status, err = executor.GetPipelineStatus("running-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "cancelled", status.Status)
	assert.NotNil(t, status.EndTime)
	assert.NotZero(t, status.Duration)
}

func TestPipelineExecutor_GetPipelineLogs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test GetPipelineLogs with non-existent pipeline
	logs, err := executor.GetPipelineLogs("non-existent")
	require.Error(t, err)
	assert.Nil(t, logs)
	assert.Contains(t, err.Error(), "pipeline not found")

	// Execute a pipeline to create status
	mockStepRegistry.EXPECT().GetExecutionOrder().Return([]string{"step1"}, nil)
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(nil)

	err = executor.ExecutePipeline(t.Context(), "test-pipeline", []string{})
	require.NoError(t, err)

	// Test GetPipelineLogs with existing pipeline
	logs, err = executor.GetPipelineLogs("test-pipeline")
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
	assert.Contains(t, logs[0], "Pipeline: test-pipeline")
	assert.Contains(t, logs[1], "Status: completed")
	assert.Contains(t, logs[2], "Start Time:")
	assert.Contains(t, logs[3], "End Time:")
	assert.Contains(t, logs[4], "Duration:")
	assert.Contains(t, logs[5], "Steps:")
	assert.Contains(t, logs[6], "step1: SUCCESS")
}

func TestPipelineExecutor_ListPipelines(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Test ListPipelines with no pipelines
	pipelines := executor.ListPipelines()
	assert.Empty(t, pipelines)

	// Execute pipelines to create statuses
	mockStepRegistry.EXPECT().GetExecutionOrder().Return([]string{"step1"}, nil).Times(2)
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil).Times(2)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil).Times(2)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil).Times(2)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil).Times(2)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(nil).Times(2)

	err := executor.ExecutePipeline(t.Context(), "pipeline1", []string{})
	require.NoError(t, err)
	err = executor.ExecutePipeline(t.Context(), "pipeline2", []string{})
	require.NoError(t, err)

	// Test ListPipelines
	pipelines = executor.ListPipelines()
	assert.Len(t, pipelines, 2)
	assert.Contains(t, pipelines, "pipeline1")
	assert.Contains(t, pipelines, "pipeline2")
}

func TestPipelineExecutor_ClearPipelineStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Execute a pipeline to create status
	mockStepRegistry.EXPECT().GetExecutionOrder().Return([]string{"step1"}, nil)
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(nil)

	err := executor.ExecutePipeline(t.Context(), "test-pipeline", []string{})
	require.NoError(t, err)

	// Verify pipeline exists
	status, err := executor.GetPipelineStatus("test-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "test-pipeline", status.PipelineName)

	// Clear pipeline status
	executor.ClearPipelineStatus("test-pipeline")

	// Verify pipeline is cleared
	_, err = executor.GetPipelineStatus("test-pipeline")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not found")
}

func TestPipelineExecutor_ClearAllPipelineStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStepRegistry := mocks.NewMockStepRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	executor := NewPipelineExecutor(mockStepRegistry, mockHookManager).(*PipelineExecutor)

	// Execute pipelines to create statuses
	mockStepRegistry.EXPECT().GetExecutionOrder().Return([]string{"step1"}, nil).Times(2)
	mockStepRegistry.EXPECT().GetStepConfig("step1").Return(interfaces.StepConfig{
		Name:    "step1",
		Timeout: 5 * time.Minute,
		Retries: 1,
	}, nil).Times(2)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeBefore).Return(nil).Times(2)
	mockStepRegistry.EXPECT().ExecuteStep(gomock.Any(), "step1").Return(nil).Times(2)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeSuccess).Return(nil).Times(2)
	mockHookManager.EXPECT().ExecuteHooks(gomock.Any(), "step1", interfaces.HookTypeAfter).Return(nil).Times(2)

	err := executor.ExecutePipeline(t.Context(), "pipeline1", []string{})
	require.NoError(t, err)
	err = executor.ExecutePipeline(t.Context(), "pipeline2", []string{})
	require.NoError(t, err)

	// Verify pipelines exist
	pipelines := executor.ListPipelines()
	assert.Len(t, pipelines, 2)

	// Clear all pipeline statuses
	executor.ClearAllPipelineStatus()

	// Verify all pipelines are cleared
	pipelines = executor.ListPipelines()
	assert.Empty(t, pipelines)
}

func TestConvertStepResults(t *testing.T) {
	// Test convertStepResults function
	internal := map[string]StepResult{
		"step1": {
			StepName: "step1",
			Success:  true,
			Duration: 5 * time.Second,
			Error:    nil,
			Output:   "step1 output",
			Metadata: map[string]any{"key": "value"},
			Artifacts: []Artifact{
				{
					Name:        "artifact1",
					Path:        "/path/to/artifact1",
					Type:        "file",
					Size:        1024,
					Checksum:    "abc123",
					Description: "Test artifact",
				},
			},
		},
	}

	result := convertStepResults(internal)
	assert.Len(t, result, 1)
	assert.Contains(t, result, "step1")

	stepResult := result["step1"]
	assert.Equal(t, "step1", stepResult.StepName)
	assert.True(t, stepResult.Success)
	assert.Equal(t, 5*time.Second, stepResult.Duration)
	require.NoError(t, stepResult.Error)
	assert.Equal(t, "step1 output", stepResult.Output)
	assert.Equal(t, map[string]any{"key": "value"}, stepResult.Metadata)
	assert.Len(t, stepResult.Artifacts, 1)
	assert.Equal(t, "artifact1", stepResult.Artifacts[0].Name)
}

func TestConvertArtifacts(t *testing.T) {
	// Test convertArtifacts function
	internal := []Artifact{
		{
			Name:        "artifact1",
			Path:        "/path/to/artifact1",
			Type:        "file",
			Size:        1024,
			Checksum:    "abc123",
			Description: "Test artifact",
		},
		{
			Name:        "artifact2",
			Path:        "/path/to/artifact2",
			Type:        "directory",
			Size:        2048,
			Checksum:    "def456",
			Description: "Another test artifact",
		},
	}

	result := convertArtifacts(internal)
	assert.Len(t, result, 2)

	assert.Equal(t, "artifact1", result[0].Name)
	assert.Equal(t, "/path/to/artifact1", result[0].Path)
	assert.Equal(t, "file", result[0].Type)
	assert.Equal(t, int64(1024), result[0].Size)
	assert.Equal(t, "abc123", result[0].Checksum)
	assert.Equal(t, "Test artifact", result[0].Description)

	assert.Equal(t, "artifact2", result[1].Name)
	assert.Equal(t, "/path/to/artifact2", result[1].Path)
	assert.Equal(t, "directory", result[1].Type)
	assert.Equal(t, int64(2048), result[1].Size)
	assert.Equal(t, "def456", result[1].Checksum)
	assert.Equal(t, "Another test artifact", result[1].Description)
}
