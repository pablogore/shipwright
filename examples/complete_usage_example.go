package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pablogore/shipwright/internal/app"
	"github.com/pablogore/shipwright/internal/config"
	"github.com/pablogore/shipwright/internal/interfaces"
)

// Complete example demonstrating the new dynamic pipeline architecture
func mainComplete() {
	ctx := context.Background()

	// 1. Load configuration
	cfg, err := config.NewConfigurationWrapper()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Initialize the application
	err = app.Initialize(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer app.Reset()

	// 3. Get the application instance
	application := app.NewApp(app.GetContainer())

	// 4. Demonstrate step registry functionality
	fmt.Println("=== Step Registry Demo ===")
	demonstrateStepRegistry(application)

	// 5. Demonstrate hook system
	fmt.Println("\n=== Hook System Demo ===")
	demonstrateHookSystem(application)

	// 6. Demonstrate pipeline execution
	fmt.Println("\n=== Pipeline Execution Demo ===")
	demonstratePipelineExecution(application)

	// 7. Demonstrate custom step registration
	fmt.Println("\n=== Custom Step Registration Demo ===")
	demonstrateCustomStepRegistration(application)
}

// demonstrateStepRegistry shows how to work with the step registry
func demonstrateStepRegistry(app *app.App) {
	container := app.GetContainer()
	stepRegistry, err := container.Get("stepRegistry")
	if err != nil {
		fmt.Printf("Error getting step registry: %v\n", err)
		return
	}

	registry := stepRegistry.(interfaces.StepRegistry)

	// List available steps
	steps := registry.ListSteps()
	fmt.Printf("Available steps: %v\n", steps)

	// Get step information
	for _, stepName := range steps {
		config, err := registry.GetStepConfig(stepName)
		if err != nil {
			fmt.Printf("Error getting config for %s: %v\n", stepName, err)
			continue
		}
		fmt.Printf("Step %s: %s (Required: %t, Timeout: %v)\n",
			stepName, config.Description, config.Required, config.Timeout)
	}

	// Get execution order
	order, err := registry.GetExecutionOrder()
	if err != nil {
		fmt.Printf("Error getting execution order: %v\n", err)
	} else {
		fmt.Printf("Execution order: %v\n", order)
	}
}

// demonstrateHookSystem shows how to register and use hooks
func demonstrateHookSystem(app *app.App) {
	container := app.GetContainer()
	hookManager, err := container.Get("hookManager")
	if err != nil {
		fmt.Printf("Error getting hook manager: %v\n", err)
		return
	}

	manager := hookManager.(interfaces.HookManager)

	// Register some example hooks
	err = manager.RegisterHook("build", interfaces.HookTypeBefore, func(ctx context.Context) error {
		fmt.Println("  🔧 Pre-build hook: Preparing build environment...")
		return nil
	})
	if err != nil {
		fmt.Printf("Error registering before hook: %v\n", err)
	}

	err = manager.RegisterHook("build", interfaces.HookTypeAfter, func(ctx context.Context) error {
		fmt.Println("  🧹 Post-build hook: Cleaning up build artifacts...")
		return nil
	})
	if err != nil {
		fmt.Printf("Error registering after hook: %v\n", err)
	}

	err = manager.RegisterHook("test", interfaces.HookTypeSuccess, func(ctx context.Context) error {
		fmt.Println("  🎉 Test success hook: Generating test report...")
		return nil
	})
	if err != nil {
		fmt.Printf("Error registering success hook: %v\n", err)
	}

	// List registered hooks
	// Note: ListHooks is not part of the HookManager interface, so we'll skip this for now
	fmt.Printf("Hooks registered successfully\n")

	// Execute hooks manually (for demonstration)
	fmt.Println("Executing build before hooks:")
	err = manager.ExecuteHooks(context.Background(), "build", interfaces.HookTypeBefore)
	if err != nil {
		fmt.Printf("Error executing hooks: %v\n", err)
	}
}

// demonstratePipelineExecution shows how to execute pipelines
func demonstratePipelineExecution(app *app.App) {
	container := app.GetContainer()
	executor, err := container.Get("pipelineExecutor")
	if err != nil {
		fmt.Printf("Error getting pipeline executor: %v\n", err)
		return
	}

	pipelineExecutor := executor.(interfaces.PipelineExecutor)

	// Execute a single step
	fmt.Println("Executing setup step:")
	err = pipelineExecutor.ExecuteStep(context.Background(), "demo-pipeline", "setup")
	if err != nil {
		fmt.Printf("Error executing setup step: %v\n", err)
	}

	// Get pipeline status
	status, err := pipelineExecutor.GetPipelineStatus("demo-pipeline")
	if err != nil {
		fmt.Printf("Error getting pipeline status: %v\n", err)
	} else {
		fmt.Printf("Pipeline status: %s (Duration: %v)\n", status.Status, status.Duration)
		fmt.Printf("Completed steps: %d\n", len(status.Steps))
	}

	// Get pipeline logs
	logs, err := pipelineExecutor.GetPipelineLogs("demo-pipeline")
	if err != nil {
		fmt.Printf("Error getting pipeline logs: %v\n", err)
	} else {
		fmt.Println("Pipeline logs:")
		for _, log := range logs {
			fmt.Printf("  %s\n", log)
		}
	}
}

// demonstrateCustomStepRegistration shows how to register custom steps
func demonstrateCustomStepRegistration(app *app.App) {
	container := app.GetContainer()
	stepRegistry, err := container.Get("stepRegistry")
	if err != nil {
		fmt.Printf("Error getting step registry: %v\n", err)
		return
	}

	registry := stepRegistry.(interfaces.StepRegistry)

	// Create a custom step handler
	cfg, _ := config.NewConfigurationWrapper()
	customHandler := &CustomStepHandler{
		config: cfg,
	}

	// Register the custom step
	err = registry.RegisterStep("custom-deploy", customHandler)
	if err != nil {
		fmt.Printf("Error registering custom step: %v\n", err)
		return
	}

	fmt.Println("✅ Custom step 'custom-deploy' registered successfully")

	// Get information about the custom step
	config, err := registry.GetStepConfig("custom-deploy")
	if err != nil {
		fmt.Printf("Error getting custom step config: %v\n", err)
	} else {
		fmt.Printf("Custom step config: %+v\n", config)
	}

	// Execute the custom step
	fmt.Println("Executing custom step:")
	err = registry.ExecuteStep(context.Background(), "custom-deploy")
	if err != nil {
		fmt.Printf("Error executing custom step: %v\n", err)
	} else {
		fmt.Println("✅ Custom step executed successfully")
	}
}

// CustomStepHandler is an example of a custom step handler
type CustomStepHandler struct {
	config interfaces.Configuration
}

func (h *CustomStepHandler) CanHandle(stepName string) bool {
	return stepName == "custom-deploy"
}

func (h *CustomStepHandler) Execute(ctx context.Context, stepName string, config interfaces.StepConfig) error {
	fmt.Printf("  🚀 Executing custom deployment step: %s\n", config.Description)

	// Simulate deployment work
	time.Sleep(100 * time.Millisecond)
	fmt.Println("  📦 Deploying application...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("  🔍 Verifying deployment...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("  ✅ Deployment completed successfully!")

	return nil
}

func (h *CustomStepHandler) GetStepInfo(stepName string) interfaces.StepConfig {
	return interfaces.StepConfig{
		Name:        "custom-deploy",
		Description: "Custom deployment step for demonstration",
		Required:    false,
		Parallel:    false,
		Timeout:     5 * time.Minute,
		Retries:     2,
		DependsOn:   []string{"build", "test"},
		Conditions: map[string]string{
			"deployment_enabled": "true",
		},
		Metadata: map[string]any{
			"category": "deployment",
			"priority": "high",
			"author":   "example",
		},
	}
}

func (h *CustomStepHandler) Validate(stepName string, config interfaces.StepConfig) error {
	if config.Name != "custom-deploy" {
		return fmt.Errorf("invalid step name for custom handler: %s", stepName)
	}
	return nil
}
