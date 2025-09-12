package interfaces

import (
	"context"
	"time"

	"dagger.io/dagger"
)

// Configuration defines the application configuration interface.
type Configuration interface {
	Load() error
	LoadWithDefaults(defaults map[string]any) error
	Validate() error
	GetConfigSummary() string

	// Core configuration methods
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetDuration(key string) time.Duration
	GetFloat(key string) float64
	Get(key string) any
	Set(key string, value any)
	Has(key string) bool
	All() map[string]any

	// Pipeline-specific configuration methods
	Pipeline() PipelineConfig
	Registry() RegistryConfig
	Security() SecurityConfig
	Logging() LoggingConfig
	Environment() string
}

// PipelineConfig defines pipeline configuration.
type PipelineConfig struct {
	Name        string  `json:"name"         yaml:"name"`
	Environment string  `json:"environment"  yaml:"environment"`
	Coverage    float64 `json:"coverage"     yaml:"coverage"`
	SkipPush    bool    `json:"skipPush"    yaml:"skipPush"`
	OnlyBuild   bool    `json:"onlyBuild"   yaml:"onlyBuild"`
	OnlyTest    bool    `json:"onlyTest"    yaml:"onlyTest"`
	Verbose     bool    `json:"verbose"      yaml:"verbose"`
	GoVersion   string  `json:"goVersion"   yaml:"goVersion"`
	JavaVersion string  `json:"javaVersion" yaml:"javaVersion"`
}

// RegistryConfig defines container registry configuration.
type RegistryConfig struct {
	BaseURL string `json:"baseUrl" yaml:"baseUrl"`
	User    string `json:"user"     yaml:"user"`
	Pass    string `json:"pass"     yaml:"pass"`
	Image   string `json:"image"    yaml:"image"`
	Tag     string `json:"tag"      yaml:"tag"`
}

// SecurityConfig defines security configuration.
type SecurityConfig struct {
	EnableVulnCheck bool     `json:"enableVulnCheck" yaml:"enableVulnCheck"`
	EnableLinting   bool     `json:"enableLinting"    yaml:"enableLinting"`
	LintTimeout     string   `json:"lintTimeout"      yaml:"lintTimeout"`
	ExcludePatterns []string `json:"excludePatterns"  yaml:"excludePatterns"`
}

// LoggingConfig defines logging configuration.
type LoggingConfig struct {
	Level            string        `json:"level"             yaml:"level"`
	Format           string        `json:"format"            yaml:"format"`
	SamplingEnable   bool          `json:"samplingEnable"   yaml:"samplingEnable"`
	SamplingRate     float64       `json:"samplingRate"     yaml:"samplingRate"`
	SamplingInterval time.Duration `json:"samplingInterval" yaml:"samplingInterval"`
}

// GitConfig defines Git configuration.
type GitConfig struct {
	Repo      string `json:"repo"       yaml:"repo"`
	Ref       string `json:"ref"        yaml:"ref"`
	Protocol  string `json:"protocol"   yaml:"protocol"`
	UserEmail string `json:"userEmail" yaml:"userEmail"`
	UserName  string `json:"userName"  yaml:"userName"`
	SSHKey    string `json:"sshKey"    yaml:"sshKey"`
}

// ReleaseConfig defines release configuration.
type ReleaseConfig struct {
	Enabled        bool     `json:"enabled"         yaml:"enabled"`
	UseGoreleaser  bool     `json:"useGoreleaser"  yaml:"useGoreleaser"`
	BuildTargets   []string `json:"buildTargets"   yaml:"buildTargets"`
	ArchiveFormats []string `json:"archiveFormats" yaml:"archiveFormats"`
	Checksum       bool     `json:"checksum"        yaml:"checksum"`
	Sign           bool     `json:"sign"            yaml:"sign"`
}

// Container defines the interface for the dependency injection container.
type Container interface {
	// Core container methods
	Register(name string, factory func() (any, error))
	Get(name string) (any, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Validate() error

	// Pipeline-specific providers
	PipelineProvider
	RegistryProvider
	SecurityProvider
	LoggingProvider
}

// PipelineProvider defines the interface for pipeline access.
type PipelineProvider interface {
	GetPipelineRegistry() (PipelineRegistry, error)
	GetPipeline(name string) (Pipeline, error)
	GetDaggerClient() (*dagger.Client, error)
}

// RegistryProvider defines the interface for registry access.
type RegistryProvider interface {
	GetRegistryConfig() (RegistryConfig, error)
	GetRegistryAuth() (string, string, error)
}

// SecurityProvider defines the interface for security tools access.
type SecurityProvider interface {
	GetVulnChecker() (VulnChecker, error)
	GetLinter() (Linter, error)
}

// LoggingProvider defines the interface for logging access.
type LoggingProvider interface {
	GetLogger() (Logger, error)
}

// Pipeline defines the core pipeline interface with dynamic step execution.
type Pipeline interface {
	Name() string
	GetAvailableSteps() []string
	ExecuteStep(ctx context.Context, stepName string) error
	BeforeStep(ctx context.Context, stepName string) HookFunc
	AfterStep(ctx context.Context, stepName string) HookFunc
	GetStepConfig(stepName string) StepConfig
	ValidateStep(stepName string) error
}

// StepConfig defines configuration for a pipeline step.
type StepConfig struct {
	Name        string            `json:"name"        yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Required    bool              `json:"required"    yaml:"required"`
	Parallel    bool              `json:"parallel"    yaml:"parallel"`
	Timeout     time.Duration     `json:"timeout"     yaml:"timeout"`
	Retries     int               `json:"retries"     yaml:"retries"`
	DependsOn   []string          `json:"dependsOn"  yaml:"dependsOn"`
	Conditions  map[string]string `json:"conditions"  yaml:"conditions"`
	Metadata    map[string]any    `json:"metadata"    yaml:"metadata"`
}

// StepExecutor defines the interface for executing pipeline steps.
type StepExecutor interface {
	Execute(ctx context.Context, stepName string, config StepConfig) error
	GetStepResult(stepName string) (StepResult, error)
	GetStepLogs(stepName string) ([]string, error)
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

// PipelineRegistry defines the interface for pipeline registration and retrieval.
type PipelineRegistry interface {
	Register(name string, factory func(*dagger.Client, Configuration) Pipeline)
	Get(name string, client *dagger.Client, cfg Configuration) (Pipeline, error)
	List() []string
}

// VulnChecker defines the interface for vulnerability checking.
type VulnChecker interface {
	Check(ctx context.Context, src *dagger.Directory) error
	GetReport(ctx context.Context) (string, error)
}

// Linter defines the interface for code linting.
type Linter interface {
	Lint(ctx context.Context, src *dagger.Directory) error
	GetReport(ctx context.Context) (string, error)
}

// Logger defines the interface for logging.
type Logger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
	Fatal(msg string, fields ...any)
	WithField(key string, value any) Logger
	WithFields(fields map[string]any) Logger
}

// HookFunc defines a function type for pipeline hooks.
type HookFunc func(ctx context.Context) error

// StepHandler defines the interface for handling pipeline steps.
type StepHandler interface {
	CanHandle(stepName string) bool
	Execute(ctx context.Context, stepName string, config StepConfig) error
	GetStepInfo(stepName string) StepConfig
	Validate(stepName string, config StepConfig) error
}

// HookManager defines the interface for managing pipeline hooks.
type HookManager interface {
	RegisterHook(stepName string, hookType HookType, hook HookFunc) error
	GetHooks(stepName string, hookType HookType) []HookFunc
	ExecuteHooks(ctx context.Context, stepName string, hookType HookType) error
	RemoveHook(stepName string, hookType HookType, hook HookFunc) error
}

// HookType defines the type of hook.
type HookType string

const (
	HookTypeBefore  HookType = "before"
	HookTypeAfter   HookType = "after"
	HookTypeError   HookType = "error"
	HookTypeSuccess HookType = "success"
)

// StepRegistry defines the interface for registering and managing steps.
type StepRegistry interface {
	RegisterStep(stepName string, handler StepHandler) error
	GetStepHandler(stepName string) (StepHandler, error)
	ListSteps() []string
	GetStepConfig(stepName string) (StepConfig, error)
	ValidateStep(stepName string) error
	ExecuteStep(ctx context.Context, stepName string) error
	GetExecutionOrder() ([]string, error)
}

// PipelineExecutor defines the interface for executing pipelines with dynamic steps.
type PipelineExecutor interface {
	ExecutePipeline(ctx context.Context, pipelineName string, steps []string) error
	ExecuteStep(ctx context.Context, pipelineName string, stepName string) error
	GetPipelineStatus(pipelineName string) (PipelineStatus, error)
	GetStepStatus(pipelineName string, stepName string) (StepResult, error)
	CancelPipeline(pipelineName string) error
	GetPipelineLogs(pipelineName string) ([]string, error)
}

// PipelineStatus represents the current status of a pipeline.
type PipelineStatus struct {
	PipelineName string                `json:"pipelineName"`
	Status       string                `json:"status"` // running, completed, failed, cancelled
	StartTime    time.Time             `json:"startTime"`
	EndTime      *time.Time            `json:"endTime,omitempty"`
	Duration     time.Duration         `json:"duration"`
	Steps        map[string]StepResult `json:"steps"`
	Metadata     map[string]any        `json:"metadata,omitempty"`
}

// ContainerError represents an error that occurred in the container.
type ContainerError struct {
	Component string
	Operation string
	Cause     error
}

func (e ContainerError) Error() string {
	return "container error in " + e.Component + " during " + e.Operation + ": " + e.Cause.Error()
}

// PipelineError represents a pipeline-specific error.
type PipelineError struct {
	Pipeline string
	Step     string
	Message  string
	Cause    error
}

func (e PipelineError) Error() string {
	return "pipeline error in " + e.Pipeline + " at step " + e.Step + ": " + e.Message + ": " + e.Cause.Error()
}

// ConfigurationError represents a configuration error.
type ConfigurationError struct {
	Key   string
	Value any
	Cause error
}

func (e ConfigurationError) Error() string {
	return "configuration error for key " + e.Key + " with value " + e.Value.(string) + ": " + e.Cause.Error()
}
