package config

import (
	"fmt"
	"time"
)

const (
	// MaxStepTimeout is the maximum allowed timeout for a step (2 hours)
	MaxStepTimeout = 2 * time.Hour
	// DefaultStepTimeout is the default timeout for steps (5 minutes)
	DefaultStepTimeout = 5 * time.Minute
	// DefaultStepRetries is the default number of retries for steps
	DefaultStepRetries = 0
)

// parseStepTimeout parses a timeout string (e.g., "5m", "1h") into a time.Duration.
// Returns an error if the format is invalid.
func parseStepTimeout(timeoutStr string) (time.Duration, error) {
	if timeoutStr == "" {
		return 0, nil
	}

	duration, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout format '%s': %w (expected format: 5m, 1h, etc.)", timeoutStr, err)
	}

	return duration, nil
}

// validateStepTimeout validates that a timeout is within acceptable limits.
func validateStepTimeout(timeout, maxTimeout time.Duration) error {
	if timeout < 0 {
		return fmt.Errorf("timeout cannot be negative: %v", timeout)
	}

	if timeout > maxTimeout {
		return fmt.Errorf("timeout %v exceeds maximum allowed timeout of %v", timeout, maxTimeout)
	}

	return nil
}

// GetStepTimeout returns the timeout for a step from configuration.
// It checks stepConfigs first, then falls back to defaults.
func (p *YAMLParser) GetStepTimeout(yamlConfig *YAMLConfig, stepName string) (time.Duration, error) {
	// Check detailed step configs
	for _, stepConfig := range yamlConfig.Pipeline.StepConfigs {
		if stepConfig.Name == stepName && stepConfig.Timeout != "" {
			timeout, err := parseStepTimeout(stepConfig.Timeout)
			if err != nil {
				return 0, err
			}
			if err := validateStepTimeout(timeout, MaxStepTimeout); err != nil {
				return 0, err
			}
			return timeout, nil
		}
	}

	// Check if steps are objects with timeout
	if steps, ok := yamlConfig.Pipeline.Steps.([]interface{}); ok {
		for _, step := range steps {
			if stepMap, ok := step.(map[string]interface{}); ok {
				if name, ok := stepMap["name"].(string); ok && name == stepName {
					if timeoutStr, ok := stepMap["timeout"].(string); ok {
						timeout, err := parseStepTimeout(timeoutStr)
						if err != nil {
							return 0, err
						}
						if err := validateStepTimeout(timeout, MaxStepTimeout); err != nil {
							return 0, err
						}
						return timeout, nil
					}
				}
			}
		}
	}

	// Return default timeout
	return DefaultStepTimeout, nil
}

// GetStepRetries returns the number of retries for a step from configuration.
func (p *YAMLParser) GetStepRetries(yamlConfig *YAMLConfig, stepName string) int {
	// Check detailed step configs
	for _, stepConfig := range yamlConfig.Pipeline.StepConfigs {
		if stepConfig.Name == stepName {
			return stepConfig.Retries
		}
	}

	// Check if steps are objects with retries
	if steps, ok := yamlConfig.Pipeline.Steps.([]interface{}); ok {
		for _, step := range steps {
			if stepMap, ok := step.(map[string]interface{}); ok {
				if name, ok := stepMap["name"].(string); ok && name == stepName {
					if retries, ok := stepMap["retries"].(int); ok {
						return retries
					}
				}
			}
		}
	}

	// Return default retries
	return DefaultStepRetries
}


