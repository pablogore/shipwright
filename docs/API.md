# API Reference

This document provides comprehensive API documentation for Syntegrity Dagger, including programmatic interfaces, data structures, and usage examples.

## Core Interfaces

### Pipeline Interface

The `Pipeline` interface defines the contract for all pipeline implementations:

```go
type Pipeline interface {
    // Name returns the name of the pipeline
    Name() string
    
    // Setup performs initial setup required for the pipeline
    Setup(ctx context.Context) error
    
    // Build compiles or builds the necessary components
    Build(ctx context.Context) error
    
    // Test executes the test step of the pipeline
    Test(ctx context.Context) error
    
    // Package creates a distributable package
    Package(ctx context.Context) error
    
    // Tag applies a tag to the pipeline's output
    Tag(ctx context.Context) error
    
    // Push uploads or deploys the pipeline's output
    Push(ctx context.Context) error
    
    // BeforeStep returns a hook function to execute before a step
    BeforeStep(ctx context.Context, step string) HookFunc
    
    // AfterStep returns a hook function to execute after a step
    AfterStep(ctx context.Context, step string) HookFunc
}
```

### Step Handler Interface

The `StepHandler` interface defines the contract for individual pipeline steps:

```go
type StepHandler interface {
    // Execute runs the step logic
    Execute(ctx context.Context) error
    
    // GetStepInfo returns information about the step
    GetStepInfo() StepInfo
    
    // Validate checks if the step can be executed
    Validate(ctx context.Context) error
}
```

### Configuration Interface

The `Configuration` interface provides access to configuration values:

```go
type Configuration interface {
    // GetString returns a string configuration value
    GetString(key string) string
    
    // GetInt returns an integer configuration value
    GetInt(key string) int
    
    // GetFloat64 returns a float64 configuration value
    GetFloat64(key string) float64
    
    // GetBool returns a boolean configuration value
    GetBool(key string) bool
    
    // GetDuration returns a duration configuration value
    GetDuration(key string) time.Duration
    
    // GetStringSlice returns a string slice configuration value
    GetStringSlice(key string) []string
    
    // Set sets a configuration value
    Set(key string, value interface{}) error
    
    // Validate validates the configuration
    Validate() error
}
```

## Data Structures

### Pipeline Configuration

```go
type Config struct {
    // Pipeline settings
    PipelineName    string        `yaml:"pipeline_name" json:"pipeline_name"`
    Environment     string        `yaml:"environment" json:"environment"`
    Coverage        float64       `yaml:"coverage" json:"coverage"`
    SkipPush        bool          `yaml:"skip_push" json:"skip_push"`
    OnlyBuild       bool          `yaml:"only_build" json:"only_build"`
    OnlyTest        bool          `yaml:"only_test" json:"only_test"`
    Verbose         bool          `yaml:"verbose" json:"verbose"`
    Timeout         time.Duration `yaml:"timeout" json:"timeout"`
    
    // Git settings
    GitURL          string        `yaml:"git_url" json:"git_url"`
    GitRef          string        `yaml:"git_ref" json:"git_ref"`
    GitProtocol     string        `yaml:"git_protocol" json:"git_protocol"`
    GitUsername     string        `yaml:"git_username" json:"git_username"`
    GitPassword     string        `yaml:"git_password" json:"git_password"`
    SSHPrivateKey   string        `yaml:"ssh_private_key" json:"ssh_private_key"`
    
    // Registry settings
    RegistryURL     string        `yaml:"registry_url" json:"registry_url"`
    RegistryUsername string       `yaml:"registry_username" json:"registry_username"`
    RegistryPassword string       `yaml:"registry_password" json:"registry_password"`
    RegistryNamespace string      `yaml:"registry_namespace" json:"registry_namespace"`
    RegistryInsecure bool         `yaml:"registry_insecure" json:"registry_insecure"`
    
    // Build settings
    GoVersion       string        `yaml:"go_version" json:"go_version"`
    CGOEnabled      bool          `yaml:"cgo_enabled" json:"cgo_enabled"`
    BuildFlags      []string      `yaml:"build_flags" json:"build_flags"`
    TestFlags       []string      `yaml:"test_flags" json:"test_flags"`
    
    // Security settings
    SecurityEnabled bool          `yaml:"security_enabled" json:"security_enabled"`
    SecurityLevel   string        `yaml:"security_level" json:"security_level"`
    FailOnVulns     bool          `yaml:"fail_on_vulnerabilities" json:"fail_on_vulnerabilities"`
    
    // Step settings
    Steps           []StepConfig  `yaml:"steps" json:"steps"`
    
    // Hook settings
    Hooks           HookConfig    `yaml:"hooks" json:"hooks"`
}
```

### Step Information

```go
type StepInfo struct {
    Name         string        `yaml:"name" json:"name"`
    Description  string        `yaml:"description" json:"description"`
    Dependencies []string      `yaml:"dependencies" json:"dependencies"`
    Timeout      time.Duration `yaml:"timeout" json:"timeout"`
    Retries      int           `yaml:"retries" json:"retries"`
    Parallel     bool          `yaml:"parallel" json:"parallel"`
    Required     bool          `yaml:"required" json:"required"`
}
```

### Step Configuration

```go
type StepConfig struct {
    Name      string        `yaml:"name" json:"name"`
    Enabled   bool          `yaml:"enabled" json:"enabled"`
    Required  bool          `yaml:"required" json:"required"`
    Timeout   time.Duration `yaml:"timeout" json:"timeout"`
    Retries   int           `yaml:"retries" json:"retries"`
    Parallel  bool          `yaml:"parallel" json:"parallel"`
}
```

### Hook Configuration

```go
type HookConfig struct {
    PrePipeline  []HookDefinition `yaml:"pre_pipeline" json:"pre_pipeline"`
    PostPipeline []HookDefinition `yaml:"post_pipeline" json:"post_pipeline"`
    PreStep      []HookDefinition `yaml:"pre_step" json:"pre_step"`
    PostStep     []HookDefinition `yaml:"post_step" json:"post_step"`
}

type HookDefinition struct {
    Name    string `yaml:"name" json:"name"`
    Command string `yaml:"command" json:"command"`
    Script  string `yaml:"script" json:"script"`
    URL     string `yaml:"url" json:"url"`
    Method  string `yaml:"method" json:"method"`
    Headers map[string]string `yaml:"headers" json:"headers"`
    Body    string `yaml:"body" json:"body"`
}
```

## Application API

### Application Initialization

```go
package main

import (
    "context"
    "log"
    
    "github.com/getsyntegrity/syntegrity-dagger/internal/app"
    "github.com/getsyntegrity/syntegrity-dagger/internal/config"
)

func main() {
    ctx := context.Background()
    
    // Load configuration
    cfg, err := config.NewConfigurationWrapper()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }
    
    // Initialize application
    err = app.Initialize(ctx, cfg)
    if err != nil {
        log.Fatalf("Failed to initialize application: %v", err)
    }
    defer app.Reset()
    
    // Get application instance
    application := app.NewApp(app.GetContainer())
    
    // Use the application...
}
```

### Pipeline Execution

```go
// Get pipeline registry
container := app.GetContainer()
registry, err := container.GetPipelineRegistry()
if err != nil {
    log.Fatalf("Failed to get pipeline registry: %v", err)
}

// Get specific pipeline
pipeline, err := registry.Get("go-kit", client, config)
if err != nil {
    log.Fatalf("Failed to get pipeline: %v", err)
}

// Execute pipeline steps
ctx := context.Background()

err = pipeline.Setup(ctx)
if err != nil {
    log.Fatalf("Setup failed: %v", err)
}

err = pipeline.Build(ctx)
if err != nil {
    log.Fatalf("Build failed: %v", err)
}

err = pipeline.Test(ctx)
if err != nil {
    log.Fatalf("Test failed: %v", err)
}

err = pipeline.Package(ctx)
if err != nil {
    log.Fatalf("Package failed: %v", err)
}

err = pipeline.Push(ctx)
if err != nil {
    log.Fatalf("Push failed: %v", err)
}
```

### Step Execution

```go
// Get step registry
stepRegistry, err := container.GetStepRegistry()
if err != nil {
    log.Fatalf("Failed to get step registry: %v", err)
}

// Get specific step
step, err := stepRegistry.GetStep("build")
if err != nil {
    log.Fatalf("Failed to get step: %v", err)
}

// Execute step
ctx := context.Background()
err = step.Execute(ctx)
if err != nil {
    log.Fatalf("Step execution failed: %v", err)
}

// Get step information
info := step.GetStepInfo()
fmt.Printf("Step: %s\n", info.Name)
fmt.Printf("Description: %s\n", info.Description)
fmt.Printf("Dependencies: %v\n", info.Dependencies)
```

### Hook Management

```go
// Get hook manager
hookManager, err := container.GetHookManager()
if err != nil {
    log.Fatalf("Failed to get hook manager: %v", err)
}

// Register a custom hook
err = hookManager.RegisterHook("pre-build", func(ctx context.Context) error {
    fmt.Println("Executing pre-build hook")
    return nil
})
if err != nil {
    log.Fatalf("Failed to register hook: %v", err)
}

// Execute hooks
ctx := context.Background()
err = hookManager.ExecuteHooks(ctx, "pre-build")
if err != nil {
    log.Fatalf("Hook execution failed: %v", err)
}
```

## Pipeline Registry API

### Pipeline Registration

```go
// Create new registry
registry := pipelines.NewRegistry()

// Register custom pipeline
registry.Register("my-pipeline", func(client *dagger.Client, cfg pipelines.Config) pipelines.Pipeline {
    return &MyPipeline{
        Client: client,
        Config: cfg,
    }
})

// Get pipeline
pipeline, err := registry.Get("my-pipeline", client, config)
if err != nil {
    log.Fatalf("Failed to get pipeline: %v", err)
}

// List available pipelines
pipelines := registry.List()
for _, name := range pipelines {
    fmt.Printf("Available pipeline: %s\n", name)
}
```

### Step Registry API

```go
// Create new step registry
stepRegistry := app.NewStepRegistry()

// Register custom step
err := stepRegistry.RegisterStep("my-step", func(config interfaces.Configuration, client *dagger.Client, logger interfaces.Logger) interfaces.StepHandler {
    return &MyStepHandler{
        config: config,
        client: client,
        logger: logger,
    }
})
if err != nil {
    log.Fatalf("Failed to register step: %v", err)
}

// Get step
step, err := stepRegistry.GetStep("my-step")
if err != nil {
    log.Fatalf("Failed to get step: %v", err)
}

// List available steps
steps := stepRegistry.ListSteps()
for _, name := range steps {
    fmt.Printf("Available step: %s\n", name)
}
```

## Configuration API

### Configuration Loading

```go
// Load configuration from multiple sources
config := config.NewConfigurationWrapper()

// Load from file
err := config.LoadFromFile(".syntegrity-dagger.yml")
if err != nil {
    log.Fatalf("Failed to load config file: %v", err)
}

// Load from environment variables
config.LoadFromEnvironment()

// Set configuration values programmatically
err = config.Set("pipeline.name", "go-kit")
if err != nil {
    log.Fatalf("Failed to set config: %v", err)
}

// Get configuration values
pipelineName := config.GetString("pipeline.name")
coverage := config.GetFloat64("pipeline.coverage")
timeout := config.GetDuration("pipeline.timeout")

// Validate configuration
err = config.Validate()
if err != nil {
    log.Fatalf("Configuration validation failed: %v", err)
}
```

### YAML Configuration Loading

```go
// Load YAML configuration
yamlConfig, err := config.LoadYAMLConfig(".syntegrity-dagger.yml")
if err != nil {
    log.Fatalf("Failed to load YAML config: %v", err)
}

// Access YAML configuration
pipelineConfig := yamlConfig.Pipeline
gitConfig := yamlConfig.Git
registryConfig := yamlConfig.Registry

// Merge with defaults
finalConfig := config.MergeWithDefaults(yamlConfig)
```

## Dagger Integration API

### Dagger Client Management

```go
// Get Dagger client from container
client, err := container.GetDaggerClient()
if err != nil {
    log.Fatalf("Failed to get Dagger client: %v", err)
}

// Use Dagger client for container operations
ctx := context.Background()

// Create container
container := client.Container().
    From("golang:1.25.1-alpine").
    WithMountedDirectory("/src", client.Host().Directory(".")).
    WithWorkdir("/src")

// Execute commands
result, err := container.WithExec([]string{"go", "build", "-o", "app", "./cmd/app"}).Stdout(ctx)
if err != nil {
    log.Fatalf("Build failed: %v", err)
}

fmt.Printf("Build output: %s\n", result)
```

### Container Operations

```go
// Build Docker image
image := client.Container().
    From("golang:1.25.1-alpine").
    WithMountedDirectory("/src", client.Host().Directory(".")).
    WithWorkdir("/src").
    WithExec([]string{"go", "build", "-o", "app", "./cmd/app"}).
    WithEntrypoint([]string{"./app"})

// Tag image
taggedImage := image.WithRegistryAuth("registry.example.com", "username", "password")

// Push image
digest, err := taggedImage.Publish(ctx, "registry.example.com/myapp:latest")
if err != nil {
    log.Fatalf("Failed to push image: %v", err)
}

fmt.Printf("Image pushed with digest: %s\n", digest)
```

## Error Handling

### Custom Error Types

```go
// Define custom error types
var (
    ErrPipelineNotFound = errors.New("pipeline not found")
    ErrStepNotFound     = errors.New("step not found")
    ErrConfigurationInvalid = errors.New("configuration is invalid")
    ErrDaggerClientFailed = errors.New("failed to create Dagger client")
)

// Wrap errors with context
func (p *MyPipeline) Build(ctx context.Context) error {
    if err := p.performBuild(ctx); err != nil {
        return fmt.Errorf("build failed for pipeline %s: %w", p.Name(), err)
    }
    return nil
}
```

### Error Handling Patterns

```go
// Handle errors with context
func executePipeline(ctx context.Context, pipeline interfaces.Pipeline) error {
    steps := []struct {
        name string
        fn   func(context.Context) error
    }{
        {"setup", pipeline.Setup},
        {"build", pipeline.Build},
        {"test", pipeline.Test},
        {"package", pipeline.Package},
        {"push", pipeline.Push},
    }
    
    for _, step := range steps {
        select {
        case <-ctx.Done():
            return fmt.Errorf("pipeline cancelled: %w", ctx.Err())
        default:
        }
        
        if err := step.fn(ctx); err != nil {
            return fmt.Errorf("step %s failed: %w", step.name, err)
        }
    }
    
    return nil
}
```

## Logging API

### Structured Logging

```go
// Get logger from container
logger, err := container.Get("logger")
if err != nil {
    log.Fatalf("Failed to get logger: %v", err)
}
log := logger.(*LoggerAdapter)

// Use structured logging
log.Info("Pipeline execution started",
    "pipeline", "go-kit",
    "environment", "production",
    "coverage", 90.0)

log.Error("Step execution failed",
    "step", "build",
    "error", err,
    "duration", time.Since(start))

// Create child logger with additional context
childLogger := log.WithFields(map[string]interface{}{
    "pipeline": "go-kit",
    "step": "build",
})

childLogger.Debug("Executing build step")
```

## Testing API

### Mock Interfaces

```go
// Create mock configuration
mockConfig := &mocks.MockConfiguration{}
mockConfig.On("GetString", "pipeline.name").Return("go-kit")
mockConfig.On("GetFloat64", "pipeline.coverage").Return(90.0)
mockConfig.On("Validate").Return(nil)

// Create mock Dagger client
mockClient := &mocks.MockDaggerClient{}

// Create mock logger
mockLogger := &mocks.MockLogger{}

// Test pipeline creation
pipeline := gokit.New(mockClient, mockConfig)
assert.Equal(t, "go-kit", pipeline.Name())
```

### Integration Testing

```go
func TestPipelineIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Set up test environment
    ctx := context.Background()
    
    // Create real Dagger client
    client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
    require.NoError(t, err)
    defer client.Close()
    
    // Create test configuration
    config := pipelines.Config{
        PipelineName: "go-kit",
        Environment: "test",
        Coverage: 80.0,
        GitURL: "https://github.com/test/repo.git",
        GitRef: "main",
        GitProtocol: "https",
    }
    
    // Create pipeline
    pipeline := gokit.New(client, config)
    
    // Test pipeline execution
    err = pipeline.Setup(ctx)
    require.NoError(t, err)
    
    err = pipeline.Build(ctx)
    require.NoError(t, err)
    
    err = pipeline.Test(ctx)
    require.NoError(t, err)
}
```

## Performance Considerations

### Resource Management

```go
// Use context for cancellation
func (p *MyPipeline) Build(ctx context.Context) error {
    // Set timeout for build operation
    buildCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
    defer cancel()
    
    // Check for cancellation
    select {
    case <-buildCtx.Done():
        return buildCtx.Err()
    default:
    }
    
    // Perform build...
    return nil
}
```

### Parallel Execution

```go
// Execute steps in parallel when possible
func (p *MyPipeline) executeStepsParallel(ctx context.Context, steps []string) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(steps))
    
    for _, step := range steps {
        wg.Add(1)
        go func(stepName string) {
            defer wg.Done()
            
            if err := p.executeStep(ctx, stepName); err != nil {
                errChan <- fmt.Errorf("step %s failed: %w", stepName, err)
            }
        }(step)
    }
    
    wg.Wait()
    close(errChan)
    
    // Check for errors
    for err := range errChan {
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

## Best Practices

### Interface Design

```go
// Keep interfaces small and focused
type StepHandler interface {
    Execute(ctx context.Context) error
    GetStepInfo() StepInfo
    Validate(ctx context.Context) error
}

// Use composition over inheritance
type MyStepHandler struct {
    config interfaces.Configuration
    client *dagger.Client
    logger interfaces.Logger
}
```

### Error Handling

```go
// Always wrap errors with context
func (p *MyPipeline) Build(ctx context.Context) error {
    if err := p.performBuild(ctx); err != nil {
        return fmt.Errorf("build failed for pipeline %s: %w", p.Name(), err)
    }
    return nil
}

// Use sentinel errors for common cases
var ErrPipelineNotFound = errors.New("pipeline not found")

func (r *Registry) Get(name string, client *dagger.Client, cfg Config) (Pipeline, error) {
    factory, ok := r.pipelines[name]
    if !ok {
        return nil, fmt.Errorf("%w: %s", ErrPipelineNotFound, name)
    }
    return factory(client, cfg), nil
}
```

### Configuration Management

```go
// Validate configuration early
func (p *MyPipeline) ValidateConfig() error {
    if p.Config.PipelineName == "" {
        return fmt.Errorf("pipeline name is required")
    }
    
    if p.Config.Coverage < 0 || p.Config.Coverage > 100 {
        return fmt.Errorf("coverage must be between 0 and 100, got %f", p.Config.Coverage)
    }
    
    return nil
}
```
