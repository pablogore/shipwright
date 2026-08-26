package plugins

import (
	"context"
	"time"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/pipelines"
)

// MockPlugin implements Plugin interface for testing.
type MockPlugin struct {
	CleanupFunc    func(ctx context.Context) error
	InitializeFunc func(ctx context.Context, pluginCtx PluginContext) error
	NameFunc       func() string
	VersionFunc    func() string
}

// Cleanup implements Plugin interface.
func (m *MockPlugin) Cleanup(ctx context.Context) error {
	if m.CleanupFunc != nil {
		return m.CleanupFunc(ctx)
	}
	return nil
}

// Initialize implements Plugin interface.
func (m *MockPlugin) Initialize(ctx context.Context, pluginCtx PluginContext) error {
	if m.InitializeFunc != nil {
		return m.InitializeFunc(ctx, pluginCtx)
	}
	return nil
}

// Name implements Plugin interface.
func (m *MockPlugin) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock-plugin"
}

// Version implements Plugin interface.
func (m *MockPlugin) Version() string {
	if m.VersionFunc != nil {
		return m.VersionFunc()
	}
	return "1.0.0"
}

// NewMockPlugin creates a new mock plugin.
func NewMockPlugin() *MockPlugin {
	return &MockPlugin{}
}

// MockPluginContext implements PluginContext interface for testing.
type MockPluginContext struct {
	GetConfigurationFunc func() interfaces.Configuration
	GetDaggerClientFunc  func() (*dagger.Client, error)
	GetHookManagerFunc   func() interfaces.HookManager
	GetLoggerFunc        func() interfaces.Logger
	GetConfigFunc        func() pipelines.Config
	GetCapabilitiesFunc  func() Capabilities
	GetStepRegistryFunc  func() interfaces.StepRegistry
}

// GetConfiguration implements PluginContext interface.
func (m *MockPluginContext) GetConfiguration() interfaces.Configuration {
	if m.GetConfigurationFunc != nil {
		return m.GetConfigurationFunc()
	}
	return nil
}

// GetDaggerClient implements PluginContext interface.
func (m *MockPluginContext) GetDaggerClient() (*dagger.Client, error) {
	if m.GetDaggerClientFunc != nil {
		return m.GetDaggerClientFunc()
	}
	return nil, nil
}

// GetHookManager implements PluginContext interface.
func (m *MockPluginContext) GetHookManager() interfaces.HookManager {
	if m.GetHookManagerFunc != nil {
		return m.GetHookManagerFunc()
	}
	return nil
}

// GetLogger implements PluginContext interface.
func (m *MockPluginContext) GetLogger() interfaces.Logger {
	if m.GetLoggerFunc != nil {
		return m.GetLoggerFunc()
	}
	return nil
}

// GetCapabilities implements PluginContext interface.
func (m *MockPluginContext) GetCapabilities() Capabilities {
	if m.GetCapabilitiesFunc != nil {
		return m.GetCapabilitiesFunc()
	}
	return Capabilities{}
}

// GetConfig implements PluginContext interface.
func (m *MockPluginContext) GetConfig() pipelines.Config {
	if m.GetConfigFunc != nil {
		return m.GetConfigFunc()
	}
	return pipelines.Config{}
}

// GetStepRegistry implements PluginContext interface.
func (m *MockPluginContext) GetStepRegistry() interfaces.StepRegistry {
	if m.GetStepRegistryFunc != nil {
		return m.GetStepRegistryFunc()
	}
	return nil
}

// NewMockPluginContext creates a new mock plugin context.
func NewMockPluginContext() *MockPluginContext {
	return &MockPluginContext{}
}

// MockPluginLoader implements PluginLoader interface for testing.
type MockPluginLoader struct {
	LoadBuiltinFunc     func(ctx context.Context, name string) (Plugin, error)
	LoadFromConfigFunc  func(ctx context.Context, config map[string]interface{}) (Plugin, error)
	LoadFromFileFunc    func(ctx context.Context, filePath string) (Plugin, error)
	RegisterBuiltinFunc func(name string, factory func() Plugin)
}

// LoadBuiltin implements PluginLoader interface.
func (m *MockPluginLoader) LoadBuiltin(ctx context.Context, name string) (Plugin, error) {
	if m.LoadBuiltinFunc != nil {
		return m.LoadBuiltinFunc(ctx, name)
	}
	return nil, nil
}

// LoadFromConfig implements PluginLoader interface.
func (m *MockPluginLoader) LoadFromConfig(ctx context.Context, config map[string]interface{}) (Plugin, error) {
	if m.LoadFromConfigFunc != nil {
		return m.LoadFromConfigFunc(ctx, config)
	}
	return nil, nil
}

// LoadFromFile implements PluginLoader interface.
func (m *MockPluginLoader) LoadFromFile(ctx context.Context, filePath string) (Plugin, error) {
	if m.LoadFromFileFunc != nil {
		return m.LoadFromFileFunc(ctx, filePath)
	}
	return nil, nil
}

// RegisterBuiltin implements PluginLoader interface.
func (m *MockPluginLoader) RegisterBuiltin(name string, factory func() Plugin) {
	if m.RegisterBuiltinFunc != nil {
		m.RegisterBuiltinFunc(name, factory)
	}
}

// NewMockPluginLoader creates a new mock plugin loader.
func NewMockPluginLoader() *MockPluginLoader {
	return &MockPluginLoader{}
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
