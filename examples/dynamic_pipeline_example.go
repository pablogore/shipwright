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

// Example of how to use the dynamic pipeline system
func main() {
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

	// 3. Get the container
	container := app.GetContainer()

	// 4. Get the step registry
	stepRegistry, err := container.Get("stepRegistry")
	if err != nil {
		log.Fatalf("Failed to get step registry: %v", err)
	}

	registry := stepRegistry.(interfaces.StepRegistry)

	// 5. List available steps
	steps := registry.ListSteps()
	fmt.Printf("Available steps: %v\n", steps)

	// 6. Get step configuration
	for _, stepName := range steps {
		stepConfig, err := registry.GetStepConfig(stepName)
		if err != nil {
			fmt.Printf("Error getting config for step %s: %v\n", stepName, err)
			continue
		}
		fmt.Printf("Step %s: %s (Required: %t, Timeout: %v)\n",
			stepName, stepConfig.Description, stepConfig.Required, stepConfig.Timeout)
	}

	// 7. Execute a specific step
	fmt.Println("\n=== Executing setup step ===")
	handler, err := registry.GetStepHandler("setup")
	if err != nil {
		log.Fatalf("Failed to get setup handler: %v", err)
	}

	stepConfig := handler.GetStepInfo("setup")
	err = handler.Execute(ctx, "setup", stepConfig)
	if err != nil {
		log.Fatalf("Setup step failed: %v", err)
	}

	// 8. Execute multiple steps in sequence
	fmt.Println("\n=== Executing multiple steps ===")
	stepNames := []string{"setup", "build", "test", "lint"}

	for _, stepName := range stepNames {
		fmt.Printf("\n--- Executing %s step ---\n", stepName)

		handler, err := registry.GetStepHandler(stepName)
		if err != nil {
			fmt.Printf("Error getting handler for %s: %v\n", stepName, err)
			continue
		}

		stepConfig := handler.GetStepInfo(stepName)

		// Check if step should be executed based on conditions
		if !shouldExecuteStep(stepConfig, cfg) {
			fmt.Printf("Skipping %s step (conditions not met)\n", stepName)
			continue
		}

		start := time.Now()
		err = handler.Execute(ctx, stepName, stepConfig)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("❌ %s step failed after %v: %v\n", stepName, duration, err)
			break
		}

		fmt.Printf("✅ %s step completed in %v\n", stepName, duration)
	}

	// 9. Example of adding a custom step dynamically
	fmt.Println("\n=== Adding custom step ===")
	customHandler := NewDynamicCustomStepHandler(cfg)
	err = registry.RegisterStep("custom", customHandler)
	if err != nil {
		fmt.Printf("Failed to register custom step: %v\n", err)
	} else {
		fmt.Println("✅ Custom step registered successfully")

		// Execute the custom step
		stepConfig := customHandler.GetStepInfo("custom")
		err = customHandler.Execute(ctx, "custom", stepConfig)
		if err != nil {
			fmt.Printf("❌ Custom step failed: %v\n", err)
		} else {
			fmt.Println("✅ Custom step executed successfully")
		}
	}
}

// shouldExecuteStep determines if a step should be executed based on its conditions
func shouldExecuteStep(config interfaces.StepConfig, cfg interfaces.Configuration) bool {
	for condition, expectedValue := range config.Conditions {
		switch condition {
		case "source_exists":
			// Check if source code exists
			return expectedValue == "true"
		case "linting_enabled":
			return cfg.GetBool("security.enable_linting")
		case "security_enabled":
			return cfg.GetBool("security.enable_vuln_check")
		case "tests_available":
			// Check if test files exist
			return expectedValue == "true"
		default:
			// Default to true for unknown conditions
			return true
		}
	}
	return true
}

// DynamicCustomStepHandler is an example of a custom step handler for dynamic example
type DynamicCustomStepHandler struct {
	config interfaces.Configuration
}

// NewDynamicCustomStepHandler creates a new custom step handler
func NewDynamicCustomStepHandler(config interfaces.Configuration) interfaces.StepHandler {
	return &DynamicCustomStepHandler{config: config}
}

func (h *DynamicCustomStepHandler) CanHandle(stepName string) bool {
	return stepName == "custom"
}

func (h *DynamicCustomStepHandler) Execute(ctx context.Context, stepName string, config interfaces.StepConfig) error {
	fmt.Printf("🎯 Executing custom step: %s\n", config.Description)

	// Custom logic here
	fmt.Println("  - Performing custom validation...")
	time.Sleep(100 * time.Millisecond) // Simulate work

	fmt.Println("  - Generating custom report...")
	time.Sleep(100 * time.Millisecond) // Simulate work

	fmt.Println("  - Custom step completed!")
	return nil
}

func (h *DynamicCustomStepHandler) GetStepInfo(stepName string) interfaces.StepConfig {
	return interfaces.StepConfig{
		Name:        "custom",
		Description: "Custom step for demonstration purposes",
		Required:    false,
		Parallel:    true,
		Timeout:     2 * time.Minute,
		Retries:     1,
		DependsOn:   []string{"test"},
		Conditions: map[string]string{
			"custom_enabled": "true",
		},
		Metadata: map[string]any{
			"category": "custom",
			"priority": "low",
			"author":   "example",
		},
	}
}

func (h *DynamicCustomStepHandler) Validate(stepName string, config interfaces.StepConfig) error {
	if config.Name != "custom" {
		return fmt.Errorf("invalid step name for custom handler: %s", stepName)
	}
	return nil
}
