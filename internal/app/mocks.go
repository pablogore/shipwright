package app

import (
	"context"
	"time"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/pipelines"
)

// MockConfiguration implements interfaces.Configuration interface for testing.
type MockConfiguration struct {
	AllFunc              func() map[string]any
	EnvironmentFunc      func() string
	GetBoolFunc          func(key string) bool
	GetConfigSummaryFunc func() string
	GetDurationFunc      func(key string) time.Duration
	GetFloatFunc         func(key string) float64
	GetFunc              func(key string) any
	GetIntFunc           func(key string) int
	GetStringFunc        func(key string) string
	HasFunc              func(key string) bool
	LoadFunc             func() error
	LoadWithDefaultsFunc func(defaults map[string]any) error
	LoggingFunc          func() interfaces.LoggingConfig
	PipelineFunc         func() interfaces.PipelineConfig
	RegistryFunc         func() interfaces.RegistryConfig
	SecurityFunc         func() interfaces.SecurityConfig
	SetFunc              func(key string, value any)
	ValidateFunc         func() error
}

// All implements interfaces.Configuration interface.
func (m *MockConfiguration) All() map[string]any {
	if m.AllFunc != nil {
		return m.AllFunc()
	}
	return make(map[string]any)
}

// Environment implements interfaces.Configuration interface.
func (m *MockConfiguration) Environment() string {
	if m.EnvironmentFunc != nil {
		return m.EnvironmentFunc()
	}
	return ""
}

// GetBool implements interfaces.Configuration interface.
func (m *MockConfiguration) GetBool(key string) bool {
	if m.GetBoolFunc != nil {
		return m.GetBoolFunc(key)
	}
	return false
}

// GetConfigSummary implements interfaces.Configuration interface.
func (m *MockConfiguration) GetConfigSummary() string {
	if m.GetConfigSummaryFunc != nil {
		return m.GetConfigSummaryFunc()
	}
	return ""
}

// GetDuration implements interfaces.Configuration interface.
func (m *MockConfiguration) GetDuration(key string) time.Duration {
	if m.GetDurationFunc != nil {
		return m.GetDurationFunc(key)
	}
	return 0
}

// GetFloat implements interfaces.Configuration interface.
func (m *MockConfiguration) GetFloat(key string) float64 {
	if m.GetFloatFunc != nil {
		return m.GetFloatFunc(key)
	}
	return 0
}

// Get implements interfaces.Configuration interface.
func (m *MockConfiguration) Get(key string) any {
	if m.GetFunc != nil {
		return m.GetFunc(key)
	}
	return nil
}

// GetInt implements interfaces.Configuration interface.
func (m *MockConfiguration) GetInt(key string) int {
	if m.GetIntFunc != nil {
		return m.GetIntFunc(key)
	}
	return 0
}

// GetString implements interfaces.Configuration interface.
func (m *MockConfiguration) GetString(key string) string {
	if m.GetStringFunc != nil {
		return m.GetStringFunc(key)
	}
	return ""
}

// Has implements interfaces.Configuration interface.
func (m *MockConfiguration) Has(key string) bool {
	if m.HasFunc != nil {
		return m.HasFunc(key)
	}
	return false
}

// Load implements interfaces.Configuration interface.
func (m *MockConfiguration) Load() error {
	if m.LoadFunc != nil {
		return m.LoadFunc()
	}
	return nil
}

// LoadWithDefaults implements interfaces.Configuration interface.
func (m *MockConfiguration) LoadWithDefaults(defaults map[string]any) error {
	if m.LoadWithDefaultsFunc != nil {
		return m.LoadWithDefaultsFunc(defaults)
	}
	return nil
}

// Logging implements interfaces.Configuration interface.
func (m *MockConfiguration) Logging() interfaces.LoggingConfig {
	if m.LoggingFunc != nil {
		return m.LoggingFunc()
	}
	return interfaces.LoggingConfig{}
}

// Pipeline implements interfaces.Configuration interface.
func (m *MockConfiguration) Pipeline() interfaces.PipelineConfig {
	if m.PipelineFunc != nil {
		return m.PipelineFunc()
	}
	return interfaces.PipelineConfig{}
}

// Registry implements interfaces.Configuration interface.
func (m *MockConfiguration) Registry() interfaces.RegistryConfig {
	if m.RegistryFunc != nil {
		return m.RegistryFunc()
	}
	return interfaces.RegistryConfig{}
}

// Security implements interfaces.Configuration interface.
func (m *MockConfiguration) Security() interfaces.SecurityConfig {
	if m.SecurityFunc != nil {
		return m.SecurityFunc()
	}
	return interfaces.SecurityConfig{}
}

// Set implements interfaces.Configuration interface.
func (m *MockConfiguration) Set(key string, value any) {
	if m.SetFunc != nil {
		m.SetFunc(key, value)
	}
}

// Validate implements interfaces.Configuration interface.
func (m *MockConfiguration) Validate() error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc()
	}
	return nil
}

// NewMockConfiguration creates a new mock configuration.
func NewMockConfiguration() *MockConfiguration {
	return &MockConfiguration{}
}

// MockPipeline implements interfaces.Pipeline interface for testing.
type MockPipeline struct {
	AfterStepFunc         func(ctx context.Context, stepName string) interfaces.HookFunc
	BeforeStepFunc        func(ctx context.Context, stepName string) interfaces.HookFunc
	ExecuteStepFunc       func(ctx context.Context, stepName string) error
	GetAvailableStepsFunc func() []string
	GetStepConfigFunc     func(stepName string) interfaces.StepConfig
	NameFunc              func() string
	ValidateStepFunc      func(stepName string) error
}

// AfterStep implements interfaces.Pipeline interface.
func (m *MockPipeline) AfterStep(ctx context.Context, stepName string) interfaces.HookFunc {
	if m.AfterStepFunc != nil {
		return m.AfterStepFunc(ctx, stepName)
	}
	return nil
}

// BeforeStep implements interfaces.Pipeline interface.
func (m *MockPipeline) BeforeStep(ctx context.Context, stepName string) interfaces.HookFunc {
	if m.BeforeStepFunc != nil {
		return m.BeforeStepFunc(ctx, stepName)
	}
	return nil
}

// ExecuteStep implements interfaces.Pipeline interface.
func (m *MockPipeline) ExecuteStep(ctx context.Context, stepName string) error {
	if m.ExecuteStepFunc != nil {
		return m.ExecuteStepFunc(ctx, stepName)
	}
	return nil
}

// GetAvailableSteps implements interfaces.Pipeline interface.
func (m *MockPipeline) GetAvailableSteps() []string {
	if m.GetAvailableStepsFunc != nil {
		return m.GetAvailableStepsFunc()
	}
	return nil
}

// GetStepConfig implements interfaces.Pipeline interface.
func (m *MockPipeline) GetStepConfig(stepName string) interfaces.StepConfig {
	if m.GetStepConfigFunc != nil {
		return m.GetStepConfigFunc(stepName)
	}
	return interfaces.StepConfig{}
}

// Name implements interfaces.Pipeline interface.
func (m *MockPipeline) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock-pipeline"
}

// ValidateStep implements interfaces.Pipeline interface.
func (m *MockPipeline) ValidateStep(stepName string) error {
	if m.ValidateStepFunc != nil {
		return m.ValidateStepFunc(stepName)
	}
	return nil
}

// NewMockPipeline creates a new mock pipeline.
func NewMockPipeline() *MockPipeline {
	return &MockPipeline{}
}

// MockPipelineRegistry implements interfaces.PipelineRegistry interface for testing.
type MockPipelineRegistry struct {
	GetFunc      func(name string, client *dagger.Client, cfg interfaces.Configuration) (interfaces.Pipeline, error)
	ListFunc     func() []string
	RegisterFunc func(name string, factory func(*dagger.Client, interfaces.Configuration) interfaces.Pipeline)
}

// Get implements interfaces.PipelineRegistry interface.
func (m *MockPipelineRegistry) Get(name string, client *dagger.Client, cfg interfaces.Configuration) (interfaces.Pipeline, error) {
	if m.GetFunc != nil {
		return m.GetFunc(name, client, cfg)
	}
	return nil, nil
}

// List implements interfaces.PipelineRegistry interface.
func (m *MockPipelineRegistry) List() []string {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return nil
}

// Register implements interfaces.PipelineRegistry interface.
func (m *MockPipelineRegistry) Register(name string, factory func(*dagger.Client, interfaces.Configuration) interfaces.Pipeline) {
	if m.RegisterFunc != nil {
		m.RegisterFunc(name, factory)
	}
}

// NewMockPipelineRegistry creates a new mock pipeline registry.
func NewMockPipelineRegistry() *MockPipelineRegistry {
	return &MockPipelineRegistry{}
}

// MockLogger implements interfaces.Logger interface for testing.
type MockLogger struct {
	DebugFunc      func(msg string, fields ...any)
	ErrorFunc      func(msg string, fields ...any)
	FatalFunc      func(msg string, fields ...any)
	InfoFunc       func(msg string, fields ...any)
	WarnFunc       func(msg string, fields ...any)
	WithFieldFunc  func(key string, value any) interfaces.Logger
	WithFieldsFunc func(fields map[string]any) interfaces.Logger
}

// Debug implements interfaces.Logger interface.
func (m *MockLogger) Debug(msg string, fields ...any) {
	if m.DebugFunc != nil {
		m.DebugFunc(msg, fields...)
	}
}

// Error implements interfaces.Logger interface.
func (m *MockLogger) Error(msg string, fields ...any) {
	if m.ErrorFunc != nil {
		m.ErrorFunc(msg, fields...)
	}
}

// Fatal implements interfaces.Logger interface.
func (m *MockLogger) Fatal(msg string, fields ...any) {
	if m.FatalFunc != nil {
		m.FatalFunc(msg, fields...)
	}
}

// Info implements interfaces.Logger interface.
func (m *MockLogger) Info(msg string, fields ...any) {
	if m.InfoFunc != nil {
		m.InfoFunc(msg, fields...)
	}
}

// Warn implements interfaces.Logger interface.
func (m *MockLogger) Warn(msg string, fields ...any) {
	if m.WarnFunc != nil {
		m.WarnFunc(msg, fields...)
	}
}

// WithField implements interfaces.Logger interface.
func (m *MockLogger) WithField(key string, value any) interfaces.Logger {
	if m.WithFieldFunc != nil {
		return m.WithFieldFunc(key, value)
	}
	return m
}

// WithFields implements interfaces.Logger interface.
func (m *MockLogger) WithFields(fields map[string]any) interfaces.Logger {
	if m.WithFieldsFunc != nil {
		return m.WithFieldsFunc(fields)
	}
	return m
}

// NewMockLogger creates a new mock logger.
func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

// MockHookManager implements interfaces.HookManager interface for testing.
type MockHookManager struct {
	ExecuteHooksFunc func(ctx context.Context, stepName string, hookType interfaces.HookType) error
	GetHooksFunc     func(stepName string, hookType interfaces.HookType) []interfaces.HookFunc
	RegisterHookFunc func(stepName string, hookType interfaces.HookType, hook interfaces.HookFunc) error
	RemoveHookFunc   func(stepName string, hookType interfaces.HookType, hook interfaces.HookFunc) error
}

// ExecuteHooks implements interfaces.HookManager interface.
func (m *MockHookManager) ExecuteHooks(ctx context.Context, stepName string, hookType interfaces.HookType) error {
	if m.ExecuteHooksFunc != nil {
		return m.ExecuteHooksFunc(ctx, stepName, hookType)
	}
	return nil
}

// GetHooks implements interfaces.HookManager interface.
func (m *MockHookManager) GetHooks(stepName string, hookType interfaces.HookType) []interfaces.HookFunc {
	if m.GetHooksFunc != nil {
		return m.GetHooksFunc(stepName, hookType)
	}
	return nil
}

// RegisterHook implements interfaces.HookManager interface.
func (m *MockHookManager) RegisterHook(stepName string, hookType interfaces.HookType, hook interfaces.HookFunc) error {
	if m.RegisterHookFunc != nil {
		return m.RegisterHookFunc(stepName, hookType, hook)
	}
	return nil
}

// RemoveHook implements interfaces.HookManager interface.
func (m *MockHookManager) RemoveHook(stepName string, hookType interfaces.HookType, hook interfaces.HookFunc) error {
	if m.RemoveHookFunc != nil {
		return m.RemoveHookFunc(stepName, hookType, hook)
	}
	return nil
}

// NewMockHookManager creates a new mock hook manager.
func NewMockHookManager() *MockHookManager {
	return &MockHookManager{}
}

// MockStepRegistry implements interfaces.StepRegistry interface for testing.
type MockStepRegistry struct {
	ExecuteStepFunc       func(ctx context.Context, stepName string) error
	GetExecutionOrderFunc func() ([]string, error)
	GetStepConfigFunc     func(stepName string) (interfaces.StepConfig, error)
	GetStepHandlerFunc    func(stepName string) (interfaces.StepHandler, error)
	ListStepsFunc         func() []string
	RegisterStepFunc      func(stepName string, handler interfaces.StepHandler) error
	ValidateStepFunc      func(stepName string) error
}

// ExecuteStep implements interfaces.StepRegistry interface.
func (m *MockStepRegistry) ExecuteStep(ctx context.Context, stepName string) error {
	if m.ExecuteStepFunc != nil {
		return m.ExecuteStepFunc(ctx, stepName)
	}
	return nil
}

// GetExecutionOrder implements interfaces.StepRegistry interface.
func (m *MockStepRegistry) GetExecutionOrder() ([]string, error) {
	if m.GetExecutionOrderFunc != nil {
		return m.GetExecutionOrderFunc()
	}
	return nil, nil
}

// GetStepConfig implements interfaces.StepRegistry interface.
func (m *MockStepRegistry) GetStepConfig(stepName string) (interfaces.StepConfig, error) {
	if m.GetStepConfigFunc != nil {
		return m.GetStepConfigFunc(stepName)
	}
	return interfaces.StepConfig{}, nil
}

// GetStepHandler implements interfaces.StepRegistry interface.
func (m *MockStepRegistry) GetStepHandler(stepName string) (interfaces.StepHandler, error) {
	if m.GetStepHandlerFunc != nil {
		return m.GetStepHandlerFunc(stepName)
	}
	return nil, nil
}

// ListSteps implements interfaces.StepRegistry interface.
func (m *MockStepRegistry) ListSteps() []string {
	if m.ListStepsFunc != nil {
		return m.ListStepsFunc()
	}
	return nil
}

// RegisterStep implements interfaces.StepRegistry interface.
func (m *MockStepRegistry) RegisterStep(stepName string, handler interfaces.StepHandler) error {
	if m.RegisterStepFunc != nil {
		return m.RegisterStepFunc(stepName, handler)
	}
	return nil
}

// ValidateStep implements interfaces.StepRegistry interface.
func (m *MockStepRegistry) ValidateStep(stepName string) error {
	if m.ValidateStepFunc != nil {
		return m.ValidateStepFunc(stepName)
	}
	return nil
}

// NewMockStepRegistry creates a new mock step registry.
func NewMockStepRegistry() *MockStepRegistry {
	return &MockStepRegistry{}
}

// MockStepHandler implements interfaces.StepHandler interface for testing.
type MockStepHandler struct {
	CanHandleFunc   func(stepName string) bool
	ExecuteFunc     func(ctx context.Context, stepName string, config interfaces.StepConfig) error
	GetStepInfoFunc func(stepName string) interfaces.StepConfig
	ValidateFunc    func(stepName string, config interfaces.StepConfig) error
}

// CanHandle implements interfaces.StepHandler interface.
func (m *MockStepHandler) CanHandle(stepName string) bool {
	if m.CanHandleFunc != nil {
		return m.CanHandleFunc(stepName)
	}
	return false
}

// Execute implements interfaces.StepHandler interface.
func (m *MockStepHandler) Execute(ctx context.Context, stepName string, config interfaces.StepConfig) error {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, stepName, config)
	}
	return nil
}

// GetStepInfo implements interfaces.StepHandler interface.
func (m *MockStepHandler) GetStepInfo(stepName string) interfaces.StepConfig {
	if m.GetStepInfoFunc != nil {
		return m.GetStepInfoFunc(stepName)
	}
	return interfaces.StepConfig{}
}

// Validate implements interfaces.StepHandler interface.
func (m *MockStepHandler) Validate(stepName string, config interfaces.StepConfig) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(stepName, config)
	}
	return nil
}

// NewMockStepHandler creates a new mock step handler.
func NewMockStepHandler() *MockStepHandler {
	return &MockStepHandler{}
}

// MockVulnChecker implements interfaces.VulnChecker interface for testing.
type MockVulnChecker struct {
	CheckFunc     func(ctx context.Context, src *dagger.Directory) error
	GetReportFunc func(ctx context.Context) (string, error)
}

// Check implements interfaces.VulnChecker interface.
func (m *MockVulnChecker) Check(ctx context.Context, src *dagger.Directory) error {
	if m.CheckFunc != nil {
		return m.CheckFunc(ctx, src)
	}
	return nil
}

// GetReport implements interfaces.VulnChecker interface.
func (m *MockVulnChecker) GetReport(ctx context.Context) (string, error) {
	if m.GetReportFunc != nil {
		return m.GetReportFunc(ctx)
	}
	return "", nil
}

// NewMockVulnChecker creates a new mock vulnerability checker.
func NewMockVulnChecker() *MockVulnChecker {
	return &MockVulnChecker{}
}

// MockLinter implements interfaces.Linter interface for testing.
type MockLinter struct {
	LintFunc      func(ctx context.Context, src *dagger.Directory) error
	GetReportFunc func(ctx context.Context) (string, error)
}

// Lint implements interfaces.Linter interface.
func (m *MockLinter) Lint(ctx context.Context, src *dagger.Directory) error {
	if m.LintFunc != nil {
		return m.LintFunc(ctx, src)
	}
	return nil
}

// GetReport implements interfaces.Linter interface.
func (m *MockLinter) GetReport(ctx context.Context) (string, error) {
	if m.GetReportFunc != nil {
		return m.GetReportFunc(ctx)
	}
	return "", nil
}

// NewMockLinter creates a new mock linter.
func NewMockLinter() *MockLinter {
	return &MockLinter{}
}

// MockPipelinesPipeline implements pipelines.Pipeline interface for testing.
type MockPipelinesPipeline struct {
	AfterStepFunc  func(ctx context.Context, step string) pipelines.HookFunc
	BeforeStepFunc func(ctx context.Context, step string) pipelines.HookFunc
	BuildFunc      func(ctx context.Context) error
	NameFunc       func() string
	PackageFunc    func(ctx context.Context) error
	PushFunc       func(ctx context.Context) error
	SetupFunc      func(ctx context.Context) error
	TagFunc        func(ctx context.Context) error
	TestFunc       func(ctx context.Context) error
}

// AfterStep implements pipelines.Pipeline interface.
func (m *MockPipelinesPipeline) AfterStep(ctx context.Context, step string) pipelines.HookFunc {
	if m.AfterStepFunc != nil {
		return m.AfterStepFunc(ctx, step)
	}
	return nil
}

// BeforeStep implements pipelines.Pipeline interface.
func (m *MockPipelinesPipeline) BeforeStep(ctx context.Context, step string) pipelines.HookFunc {
	if m.BeforeStepFunc != nil {
		return m.BeforeStepFunc(ctx, step)
	}
	return nil
}

// Build implements pipelines.Pipeline interface.
func (m *MockPipelinesPipeline) Build(ctx context.Context) error {
	if m.BuildFunc != nil {
		return m.BuildFunc(ctx)
	}
	return nil
}

// Name implements pipelines.Pipeline interface.
func (m *MockPipelinesPipeline) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock-pipeline"
}

// Package implements pipelines.Pipeline interface.
func (m *MockPipelinesPipeline) Package(ctx context.Context) error {
	if m.PackageFunc != nil {
		return m.PackageFunc(ctx)
	}
	return nil
}

// Push implements pipelines.Pipeline interface.
func (m *MockPipelinesPipeline) Push(ctx context.Context) error {
	if m.PushFunc != nil {
		return m.PushFunc(ctx)
	}
	return nil
}

// Setup implements pipelines.Pipeline interface.
func (m *MockPipelinesPipeline) Setup(ctx context.Context) error {
	if m.SetupFunc != nil {
		return m.SetupFunc(ctx)
	}
	return nil
}

// Tag implements pipelines.Pipeline interface.
func (m *MockPipelinesPipeline) Tag(ctx context.Context) error {
	if m.TagFunc != nil {
		return m.TagFunc(ctx)
	}
	return nil
}

// Test implements pipelines.Pipeline interface.
func (m *MockPipelinesPipeline) Test(ctx context.Context) error {
	if m.TestFunc != nil {
		return m.TestFunc(ctx)
	}
	return nil
}

// NewMockPipelinesPipeline creates a new mock pipelines pipeline.
func NewMockPipelinesPipeline() *MockPipelinesPipeline {
	return &MockPipelinesPipeline{}
}
