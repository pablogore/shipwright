package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/getsyntegrity/syntegrity-dagger/internal/interfaces"
)

// PipelineExecutor implements the PipelineExecutor interface for executing pipelines.
type PipelineExecutor struct {
	stepRegistry interfaces.StepRegistry
	hookManager  interfaces.HookManager
	status       map[string]*PipelineStatus
	mutex        sync.RWMutex
}

// PipelineStatus represents the current status of a pipeline execution.
type PipelineStatus struct {
	PipelineName string                `json:"pipelineName"`
	Status       string                `json:"status"` // running, completed, failed, cancelled
	StartTime    time.Time             `json:"startTime"`
	EndTime      *time.Time            `json:"endTime,omitempty"`
	Duration     time.Duration         `json:"duration"`
	Steps        map[string]StepResult `json:"steps"`
	Metadata     map[string]any        `json:"metadata,omitempty"`
	mutex        sync.RWMutex
}

// StepResult contains the result of a step execution.
type StepResult struct {
	StepName  string         `json:"stepName"`
	Success   bool           `json:"success"`
	Duration  time.Duration  `json:"duration"`
	Error     error          `json:"error,omitempty"`
	Output    string         `json:"output,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Artifacts []Artifact     `json:"artifacts,omitempty"`
}

// Artifact represents a file or artifact produced by a step.
type Artifact struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum,omitempty"`
	Description string `json:"description,omitempty"`
}

// NewPipelineExecutor creates a new pipeline executor.
func NewPipelineExecutor(stepRegistry interfaces.StepRegistry, hookManager interfaces.HookManager) interfaces.PipelineExecutor {
	return &PipelineExecutor{
		stepRegistry: stepRegistry,
		hookManager:  hookManager,
		status:       make(map[string]*PipelineStatus),
	}
}

// ExecutePipeline executes a complete pipeline with the specified steps.
func (pe *PipelineExecutor) ExecutePipeline(ctx context.Context, pipelineName string, steps []string) error {
	// Initialize pipeline status
	status := &PipelineStatus{
		PipelineName: pipelineName,
		Status:       "running",
		StartTime:    time.Now(),
		Steps:        make(map[string]StepResult),
		Metadata:     make(map[string]any),
	}

	pe.mutex.Lock()
	pe.status[pipelineName] = status
	pe.mutex.Unlock()

	defer func() {
		status.mutex.Lock()
		status.EndTime = &[]time.Time{time.Now()}[0]
		status.Duration = status.EndTime.Sub(status.StartTime)
		status.mutex.Unlock()
	}()

	// Get execution order if no steps specified
	if len(steps) == 0 {
		orderedSteps, err := pe.stepRegistry.GetExecutionOrder()
		if err != nil {
			status.Status = "failed"
			return fmt.Errorf("failed to get execution order: %w", err)
		}
		steps = orderedSteps
	}

	// Execute steps in order
	for _, stepName := range steps {
		// Check if pipeline was cancelled
		select {
		case <-ctx.Done():
			status.Status = "cancelled"
			return ctx.Err()
		default:
		}

		// Execute step
		stepResult, err := pe.executeStep(ctx, pipelineName, stepName)

		status.mutex.Lock()
		status.Steps[stepName] = stepResult
		status.mutex.Unlock()

		if err != nil {
			status.Status = "failed"
			return fmt.Errorf("pipeline %s failed at step %s: %w", pipelineName, stepName, err)
		}
	}

	status.Status = "completed"
	return nil
}

// ExecuteStep executes a single step in a pipeline.
func (pe *PipelineExecutor) ExecuteStep(ctx context.Context, pipelineName string, stepName string) error {
	// Initialize pipeline status if not exists
	pe.mutex.Lock()
	if pe.status[pipelineName] == nil {
		pe.status[pipelineName] = &PipelineStatus{
			PipelineName: pipelineName,
			Status:       "running",
			StartTime:    time.Now(),
			Steps:        make(map[string]StepResult),
			Metadata:     make(map[string]any),
		}
	}
	pe.mutex.Unlock()

	stepResult, err := pe.executeStep(ctx, pipelineName, stepName)

	pe.mutex.Lock()
	pe.status[pipelineName].Steps[stepName] = stepResult
	pe.mutex.Unlock()

	return err
}

// executeStep is the internal method for executing a step.
func (pe *PipelineExecutor) executeStep(ctx context.Context, _ string, stepName string) (StepResult, error) {
	startTime := time.Now()

	// Get step configuration
	config, err := pe.stepRegistry.GetStepConfig(stepName)
	if err != nil {
		return StepResult{
			StepName: stepName,
			Success:  false,
			Duration: time.Since(startTime),
			Error:    err,
		}, err
	}

	// Create step context with timeout
	stepCtx := ctx
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	// Execute before hooks
	if err := pe.hookManager.ExecuteHooks(stepCtx, stepName, interfaces.HookTypeBefore); err != nil {
		return StepResult{
			StepName: stepName,
			Success:  false,
			Duration: time.Since(startTime),
			Error:    fmt.Errorf("before hooks failed: %w", err),
		}, err
	}

	// Execute the step
	stepErr := pe.stepRegistry.ExecuteStep(stepCtx, stepName)
	duration := time.Since(startTime)

	// Execute appropriate hooks based on result
	if stepErr != nil {
		// Execute error hooks
		_ = pe.hookManager.ExecuteHooks(stepCtx, stepName, interfaces.HookTypeError)

		return StepResult{
			StepName: stepName,
			Success:  false,
			Duration: duration,
			Error:    stepErr,
		}, stepErr
	}

	// Execute success hooks
	_ = pe.hookManager.ExecuteHooks(stepCtx, stepName, interfaces.HookTypeSuccess)

	// Execute after hooks
	if err := pe.hookManager.ExecuteHooks(stepCtx, stepName, interfaces.HookTypeAfter); err != nil {
		return StepResult{
			StepName: stepName,
			Success:  false,
			Duration: duration,
			Error:    fmt.Errorf("after hooks failed: %w", err),
		}, err
	}

	return StepResult{
		StepName: stepName,
		Success:  true,
		Duration: duration,
		Metadata: config.Metadata,
	}, nil
}

// GetPipelineStatus returns the current status of a pipeline.
func (pe *PipelineExecutor) GetPipelineStatus(pipelineName string) (interfaces.PipelineStatus, error) {
	pe.mutex.RLock()
	defer pe.mutex.RUnlock()

	status, exists := pe.status[pipelineName]
	if !exists {
		return interfaces.PipelineStatus{}, fmt.Errorf("pipeline not found: %s", pipelineName)
	}

	status.mutex.RLock()
	defer status.mutex.RUnlock()

	return interfaces.PipelineStatus{
		PipelineName: status.PipelineName,
		Status:       status.Status,
		StartTime:    status.StartTime,
		EndTime:      status.EndTime,
		Duration:     status.Duration,
		Steps:        convertStepResults(status.Steps),
		Metadata:     status.Metadata,
	}, nil
}

// GetStepStatus returns the status of a specific step in a pipeline.
func (pe *PipelineExecutor) GetStepStatus(pipelineName string, stepName string) (interfaces.StepResult, error) {
	pe.mutex.RLock()
	defer pe.mutex.RUnlock()

	status, exists := pe.status[pipelineName]
	if !exists {
		return interfaces.StepResult{}, fmt.Errorf("pipeline not found: %s", pipelineName)
	}

	status.mutex.RLock()
	defer status.mutex.RUnlock()

	stepResult, exists := status.Steps[stepName]
	if !exists {
		return interfaces.StepResult{}, fmt.Errorf("step not found: %s in pipeline %s", stepName, pipelineName)
	}

	return interfaces.StepResult{
		StepName:  stepResult.StepName,
		Success:   stepResult.Success,
		Duration:  stepResult.Duration,
		Error:     stepResult.Error,
		Output:    stepResult.Output,
		Metadata:  stepResult.Metadata,
		Artifacts: convertArtifacts(stepResult.Artifacts),
	}, nil
}

// CancelPipeline cancels a running pipeline.
func (pe *PipelineExecutor) CancelPipeline(pipelineName string) error {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()

	status, exists := pe.status[pipelineName]
	if !exists {
		return fmt.Errorf("pipeline not found: %s", pipelineName)
	}

	status.mutex.Lock()
	defer status.mutex.Unlock()

	if status.Status == "running" {
		status.Status = "cancelled"
		status.EndTime = &[]time.Time{time.Now()}[0]
		status.Duration = status.EndTime.Sub(status.StartTime)
	}

	return nil
}

// GetPipelineLogs returns the logs for a pipeline.
func (pe *PipelineExecutor) GetPipelineLogs(pipelineName string) ([]string, error) {
	pe.mutex.RLock()
	defer pe.mutex.RUnlock()

	status, exists := pe.status[pipelineName]
	if !exists {
		return nil, fmt.Errorf("pipeline not found: %s", pipelineName)
	}

	status.mutex.RLock()
	defer status.mutex.RUnlock()

	logs := make([]string, 0, 6) // Pre-allocate with estimated capacity
	logs = append(logs, "Pipeline: "+status.PipelineName)
	logs = append(logs, "Status: "+status.Status)
	logs = append(logs, "Start Time: "+status.StartTime.Format(time.RFC3339))

	if status.EndTime != nil {
		logs = append(logs, "End Time: "+status.EndTime.Format(time.RFC3339))
		logs = append(logs, "Duration: "+status.Duration.String())
	}

	logs = append(logs, "Steps:")
	for stepName, stepResult := range status.Steps {
		logs = append(logs, fmt.Sprintf("  %s: %s (Duration: %s)",
			stepName,
			map[bool]string{true: "SUCCESS", false: "FAILED"}[stepResult.Success],
			stepResult.Duration))

		if stepResult.Error != nil {
			logs = append(logs, fmt.Sprintf("    Error: %s", stepResult.Error))
		}
	}

	return logs, nil
}

// ListPipelines returns a list of all pipeline names.
func (pe *PipelineExecutor) ListPipelines() []string {
	pe.mutex.RLock()
	defer pe.mutex.RUnlock()

	pipelines := make([]string, 0, len(pe.status))
	for pipelineName := range pe.status {
		pipelines = append(pipelines, pipelineName)
	}

	return pipelines
}

// ClearPipelineStatus removes the status for a specific pipeline.
func (pe *PipelineExecutor) ClearPipelineStatus(pipelineName string) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()

	delete(pe.status, pipelineName)
}

// ClearAllPipelineStatus removes all pipeline statuses.
func (pe *PipelineExecutor) ClearAllPipelineStatus() {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()

	pe.status = make(map[string]*PipelineStatus)
}

// convertStepResults converts internal StepResult to interface StepResult.
func convertStepResults(internal map[string]StepResult) map[string]interfaces.StepResult {
	result := make(map[string]interfaces.StepResult)
	for name, stepResult := range internal {
		result[name] = interfaces.StepResult{
			StepName:  stepResult.StepName,
			Success:   stepResult.Success,
			Duration:  stepResult.Duration,
			Error:     stepResult.Error,
			Output:    stepResult.Output,
			Metadata:  stepResult.Metadata,
			Artifacts: convertArtifacts(stepResult.Artifacts),
		}
	}
	return result
}

// convertArtifacts converts internal Artifact to interface Artifact.
func convertArtifacts(internal []Artifact) []interfaces.Artifact {
	result := make([]interfaces.Artifact, len(internal))
	for i, artifact := range internal {
		result[i] = interfaces.Artifact{
			Name:        artifact.Name,
			Path:        artifact.Path,
			Type:        artifact.Type,
			Size:        artifact.Size,
			Checksum:    artifact.Checksum,
			Description: artifact.Description,
		}
	}
	return result
}
