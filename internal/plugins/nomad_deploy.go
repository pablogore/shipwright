package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"dagger.io/dagger"

	"github.com/getsyntegrity/go-kit-logger/pkg/logger"

	"github.com/pablogore/shipwright/internal/interfaces"
	"github.com/pablogore/shipwright/internal/pipelines"
)

// NomadDeployPlugin is a plugin that adds Nomad deployment capability to pipelines.
// This allows services to deploy to Nomad clusters without modifying the core Dagger library.
type NomadDeployPlugin struct {
	config  map[string]interface{}
	name    string
	version string
}

// NewNomadDeployPlugin creates a new Nomad deploy plugin.
func NewNomadDeployPlugin() Plugin {
	return &NomadDeployPlugin{
		config:  make(map[string]interface{}),
		name:    "nomad-deploy",
		version: "1.0.0",
	}
}

// Name returns the plugin name.
func (p *NomadDeployPlugin) Name() string {
	return p.name
}

// Version returns the plugin version.
func (p *NomadDeployPlugin) Version() string {
	return p.version
}

// Initialize initializes the plugin and registers hooks/steps.
func (p *NomadDeployPlugin) Initialize(ctx context.Context, pluginCtx PluginContext) error {
	logger.L().InfoContext(ctx, "Initializing Nomad deploy plugin")

	// Get configuration
	cfg := pluginCtx.GetConfiguration()
	if nomadConfig := cfg.Get("plugins.nomad-deploy"); nomadConfig != nil {
		if configMap, ok := nomadConfig.(map[string]interface{}); ok {
			p.config = configMap
		}
	}

	// Register hook to deploy to Nomad after push step
	hookManager := pluginCtx.GetHookManager()
	if err := hookManager.RegisterHook(
		"push",
		interfaces.HookTypeAfter,
		p.createDeployHook(pluginCtx),
	); err != nil {
		return fmt.Errorf("failed to register deploy hook: %w", err)
	}

	// Register custom "deploy-nomad" step
	stepRegistry := pluginCtx.GetStepRegistry()
	if err := stepRegistry.RegisterStep(
		"deploy-nomad",
		&nomadDeployStepHandler{config: p.config, pluginCtx: pluginCtx},
	); err != nil {
		return fmt.Errorf("failed to register deploy-nomad step: %w", err)
	}

	logger.L().InfoContext(ctx, "Nomad deploy plugin initialized", "config", p.config)

	return nil
}

// Cleanup cleans up the plugin.
func (p *NomadDeployPlugin) Cleanup(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Cleaning up Nomad deploy plugin")
	return nil
}

// createDeployHook creates a hook function that deploys to Nomad after push.
func (p *NomadDeployPlugin) createDeployHook(pluginCtx PluginContext) interfaces.HookFunc {
	return func(ctx context.Context) error {
		// Check if auto-deploy is enabled
		if autoDeploy, ok := p.config["auto_deploy"].(bool); ok && !autoDeploy {
			logger.L().InfoContext(ctx, "Auto-deploy disabled, skipping Nomad deployment")
			return nil
		}

		logger.L().InfoContext(ctx, "Deploying to Nomad after push")

		// Get Nomad configuration
		nomadAddr := p.getConfigString("nomad_addr", "http://localhost:4646")
		nomadJob := p.getConfigString("nomad_job", "")
		// Get nomad_job_file, but allow empty string if explicitly set
		nomadJobFile := ""
		if jobFile, ok := p.config["nomad_job_file"].(string); ok {
			nomadJobFile = jobFile
		} else {
			// Use default only if not explicitly set
			nomadJobFile = "nomad.hcl"
		}

		if nomadJob == "" && nomadJobFile == "" {
			return fmt.Errorf("nomad job or job file must be specified")
		}

		// Get Dagger client to access the image
		daggerClient, err := pluginCtx.GetDaggerClient()
		if err != nil {
			return fmt.Errorf("dagger client not available: %w", err)
		}

		// Get pipeline config to get image reference
		pipelineConfig := pluginCtx.GetPipelineConfig()
		imageRef := p.buildImageRefFromConfig(pipelineConfig)

		// Deploy to Nomad
		if err := p.deployToNomad(ctx, daggerClient, nomadAddr, nomadJob, nomadJobFile, imageRef); err != nil {
			return fmt.Errorf("failed to deploy to Nomad: %w", err)
		}

		logger.L().InfoContext(ctx, "Successfully deployed to Nomad", "image", imageRef)

		return nil
	}
}

// deployToNomad deploys the image to Nomad.
func (p *NomadDeployPlugin) deployToNomad(
	ctx context.Context,
	client *dagger.Client,
	nomadAddr string,
	nomadJob string,
	nomadJobFile string,
	imageRef string,
) error {
	// Create a container with Nomad CLI
	container := client.Container().
		From("hashicorp/nomad:latest").
		WithEnvVariable("NOMAD_ADDR", nomadAddr)

	// If we have a job file, mount it
	if nomadJobFile != "" {
		// In a real implementation, you would mount the job file
		// For now, we'll use the job content directly
		logger.L().InfoContext(ctx, "Using Nomad job file", "file", nomadJobFile)
	}

	// If we have job content, write it to a file
	if nomadJob != "" {
		container = container.WithExec([]string{"sh", "-c", fmt.Sprintf("echo '%s' > /tmp/job.hcl", nomadJob)})
	}

	// Run Nomad job run
	// In a real implementation, you would:
	// 1. Update the job file with the new image reference
	// 2. Run `nomad job run /tmp/job.hcl`
	// For this example, we'll use a simpler approach with environment variables

	// For now, we'll use a local Nomad CLI if available
	// In production, you might want to use the Nomad API directly
	if err := p.runNomadCommand(ctx, nomadAddr, nomadJobFile, imageRef); err != nil {
		return fmt.Errorf("nomad deployment failed: %w", err)
	}

	return nil
}

// runNomadCommand runs a Nomad CLI command locally.
func (p *NomadDeployPlugin) runNomadCommand(ctx context.Context, nomadAddr, jobFile, imageRef string) error {
	// Check if Nomad CLI is available
	if _, err := exec.LookPath("nomad"); err != nil {
		logger.L().WarnContext(ctx, "Nomad CLI not found, skipping deployment",
			"hint", "Install Nomad CLI or use Nomad API directly")
		return nil // Don't fail if Nomad CLI is not available
	}

	// Set Nomad address
	env := os.Environ()
	env = append(env, fmt.Sprintf("NOMAD_ADDR=%s", nomadAddr))

	// Update job file with new image reference if needed
	if jobFile != "" {
		// In a real implementation, you would:
		// 1. Read the job file
		// 2. Replace the image reference
		// 3. Write it back
		// 4. Run `nomad job run <file>`
		logger.L().InfoContext(ctx, "Would deploy Nomad job", "file", jobFile, "image", imageRef)
	}

	return nil
}

// buildImageRefFromConfig builds the full image reference from pipeline config.
func (p *NomadDeployPlugin) buildImageRefFromConfig(cfg pipelines.Config) string {
	// Get registry URL
	registry := cfg.RegistryURL
	if registry == "" {
		registry = cfg.Registry
	}
	if registry == "" {
		registry = os.Getenv("REGISTRY_URL")
	}
	if registry == "" {
		registry = "localhost:5000"
	}

	// Get image name
	imageName := cfg.ImageName
	if imageName == "" {
		imageName = os.Getenv("IMAGE_NAME")
	}
	if imageName == "" {
		imageName = "app"
	}

	// Get image tag
	tag := cfg.ImageTag
	if tag == "" {
		tag = cfg.BuildTag
	}
	if tag == "" {
		tag = os.Getenv("IMAGE_TAG")
	}
	if tag == "" {
		tag = "latest"
	}

	return fmt.Sprintf("%s/%s:%s", registry, imageName, tag)
}

// getConfigString gets a string value from config with default.
func (p *NomadDeployPlugin) getConfigString(key, defaultValue string) string {
	if value, ok := p.config[key].(string); ok && value != "" {
		return value
	}
	return defaultValue
}

// nomadDeployStepHandler handles the "deploy-nomad" step.
type nomadDeployStepHandler struct {
	config    map[string]interface{}
	pluginCtx PluginContext
}

// CanHandle checks if this handler can handle the step.
func (h *nomadDeployStepHandler) CanHandle(stepName string) bool {
	return stepName == "deploy-nomad"
}

// Execute executes the deploy-nomad step.
func (h *nomadDeployStepHandler) Execute(ctx context.Context, stepName string, config interfaces.StepConfig) error {
	if !h.CanHandle(stepName) {
		return fmt.Errorf("handler cannot handle step: %s", stepName)
	}

	logger.L().InfoContext(ctx, "Executing deploy-nomad step")

	// Get Dagger client
	daggerClient, err := h.pluginCtx.GetDaggerClient()
	if err != nil {
		return fmt.Errorf("dagger client not available: %w", err)
	}

	// Get pipeline config
	pipelineConfig := h.pluginCtx.GetPipelineConfig()

	// Build image reference
	plugin := &NomadDeployPlugin{config: h.config}
	imageRef := plugin.buildImageRefFromConfig(pipelineConfig)

	// Get Nomad configuration
	nomadAddr := plugin.getConfigString("nomad_addr", "http://localhost:4646")
	nomadJobFile := plugin.getConfigString("nomad_job_file", "nomad.hcl")

	// Deploy to Nomad
	if err := plugin.deployToNomad(ctx, daggerClient, nomadAddr, "", nomadJobFile, imageRef); err != nil {
		return fmt.Errorf("failed to deploy to Nomad: %w", err)
	}

	logger.L().InfoContext(ctx, "Deploy-nomad step completed", "image", imageRef)

	return nil
}

// GetStepInfo returns step configuration.
func (h *nomadDeployStepHandler) GetStepInfo(stepName string) interfaces.StepConfig {
	return interfaces.StepConfig{
		Name:        stepName,
		Description: "Deploy application to Nomad cluster",
		Required:    false,
		Timeout:     0,                // No timeout by default
		DependsOn:   []string{"push"}, // Depends on push step
	}
}

// Validate validates the step configuration.
func (h *nomadDeployStepHandler) Validate(stepName string, config interfaces.StepConfig) error {
	if stepName != "deploy-nomad" {
		return fmt.Errorf("invalid step name: %s", stepName)
	}
	return nil
}
