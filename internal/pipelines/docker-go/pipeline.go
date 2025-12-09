// Package dockergo provides Docker-based Go pipeline implementations.
package dockergo

import (
	"context"
	"errors"
	"fmt"
	"os"

	"dagger.io/dagger"
	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines/shared"
)

type Pipeline struct {
	Client *dagger.Client
	Config pipelines.Config
	Src    *dagger.Directory
	Image  *dagger.Container
	Cloner shared.Cloner
}

func New(client *dagger.Client, cfg pipelines.Config) pipelines.Pipeline {
	return &Pipeline{
		Client: client,
		Config: cfg,
	}
}

func (p *Pipeline) Test(ctx context.Context) error {
	if p.Src == nil {
		return errors.New("pipeline not set up: source directory is nil")
	}

	fmt.Println("🧪 running tests for docker-go...")

	// Use Go version from config, default to 1.25.5 if not set
	goVersion := p.Config.GoVersion
	if goVersion == "" {
		goVersion = "1.25.5"
	}

	// Create a Go container
	goContainer := p.Client.Container().
		From("golang:" + goVersion).
		WithWorkdir("/app")

	// Mount the source code
	goContainer = goContainer.WithMountedDirectory("/app", p.Src)

	// Run tests
	_, err := goContainer.
		WithExec([]string{"go", "test", "-v", "./..."}).
		Sync(ctx)
	if err != nil {
		return fmt.Errorf("failed to run tests: %w", err)
	}

	fmt.Println("✅ tests passed")
	return nil
}

func (p *Pipeline) Build(ctx context.Context) error {
	if p.Src == nil {
		return errors.New("pipeline not set up: source directory is nil")
	}
	fmt.Printf("🔧 build docker image %s...\n", p.Name())

	entries, _ := p.Src.Entries(ctx)
	for _, e := range entries {
		fmt.Printf("  - %s\n", e)
	}

	img := p.Src.DockerBuild()
	p.Image = img

	fmt.Println("✅ image built in memory correctly")
	return nil
}

func (p *Pipeline) Package(_ context.Context) error {
	return nil
}

func (p *Pipeline) Tag(_ context.Context) error {
	fmt.Println("🏷️ Tagging image in memory...")

	if p.Image == nil {
		return errors.New("❌ image not built - run the Build step first")
	}

	if p.Config.ImageTag == "" {
		if short := os.Getenv("CI_COMMIT_SHORT_SHA"); short != "" {
			p.Config.ImageTag = short
		} else {
			fmt.Println("⚠️  CI_COMMIT_SHORT_SHA not available. Using 'dev' as the default tag.")
			p.Config.ImageTag = "dev"
		}
	}

	envRegistry := fmt.Sprintf("%s/%s", p.Config.RegistryURL, p.Name())
	p.Config.Registry = envRegistry

	if err := validateConfig(p.Config); err != nil {
		return fmt.Errorf("❌ invalid configuration: %w", err)
	}

	fmt.Printf("✅ image prepared for tag: %s:%s\n", p.Config.Registry, p.Config.ImageTag)
	return nil
}

func (p *Pipeline) Name() string {
	return "docker-go"
}

func (p *Pipeline) Setup(ctx context.Context) error {
	if p.Cloner != nil {
		dir, err := p.Cloner.Clone(ctx, p.Client, shared.GitCloneOpts{})
		if err != nil {
			return err
		}
		p.Src = dir
	}
	return nil
}

func (p *Pipeline) Push(ctx context.Context) error {
	fmt.Println("📦 pushing an image to the GitLab Container registry...")

	if p.Image == nil {
		return errors.New("❌ no image built to push")
	}
	if err := validateConfig(p.Config); err != nil {
		return err
	}

	fullTag := fmt.Sprintf("%s:%s", p.Config.Registry, p.Config.ImageTag)
	fmt.Printf("📌 Push to: %s\n", fullTag)

	var (
		username string
		secret   *dagger.Secret
	)

	// Try to get credentials from environment (CI or local)
	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" || os.Getenv("GITLAB_CI") == "true" {
		// Try GitHub token first
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			username = "x-access-token"
			secret = p.Client.SetSecret("github-token", token)
			fmt.Println("🔐 using GitHub CI authentication")
		} else if token := os.Getenv("CI_JOB_TOKEN"); token != "" {
			// GitLab CI token
			username = "gitlab-ci-token"
			secret = p.Client.SetSecret("ci-job-token", token)
			fmt.Println("🔐 using GitLab CI authentication")
		} else {
			return errors.New("❌ No CI token available (GITHUB_TOKEN or CI_JOB_TOKEN)")
		}
	} else {
		username = p.Config.RegistryUser
		if username == "" {
			return errors.New("❌ CI_REGISTRY_USER empty in local environment")
		}
		password := p.Config.RegistryToken
		if password == "" {
			return errors.New("❌ CI_REGISTRY_USER empty in local environment")
		}
		secret = p.Client.SetSecret("local-registry-password", password)
		fmt.Println("🔐 using local authentication")
	}

	container := p.Image.WithRegistryAuth(p.Config.Registry, username, secret)

	url, err := container.Publish(ctx, fullTag)
	if err != nil {
		return fmt.Errorf("❌ error when pushing the image: %w", err)
	}

	fmt.Printf("✅ published image: %s\n", url)
	return nil
}

func (p *Pipeline) BeforeStep(_ context.Context, _ string) pipelines.HookFunc {
	return nil
}

func (p *Pipeline) AfterStep(_ context.Context, _ string) pipelines.HookFunc {
	return nil
}

func validateConfig(cfg pipelines.Config) error {
	if cfg.BranchName == "" {
		return errors.New("❌ BranchName not defined")
	}
	if cfg.Registry == "" {
		return errors.New("❌ Registry URL not defined")
	}
	if cfg.ImageTag == "" {
		return errors.New("❌ ImageTag not defined")
	}
	return nil
}
