// Package infra provides infrastructure-related pipeline implementations.
package infra

import (
	"context"
	"errors"

	"dagger.io/dagger"
	"github.com/pablogore/kit-logger/pkg/logger"
	"github.com/pablogore/shipwright/internal/pipelines"
	"github.com/pablogore/shipwright/internal/pipelines/shared"
)

// Package infra implementa el pipeline para la infraestructura de Syntegrity.

// SyntegrityInfraPipeline represents a pipeline for the syntegrity-infra pipeline
//
// Fields:
//   - Client: The Dagger client used for container operations.
//   - Config: The configuration for the pipeline.
//   - Src: The source directory of the cloned repository.
//   - Cloner: The cloner used for cloning repositories.
type SyntegrityInfraPipeline struct {
	Client pipelines.DaggerClient
	Config pipelines.Config
	Src    pipelines.DaggerDirectory
	Cloner shared.Cloner
}

// New creates a new instance of SyntegrityInfraPipeline.
//
// Parameters:
//   - client: The Dagger client used for container operations.
//   - cfg: The configuration for the pipeline.
//
// Returns:
//   - A new instance of SyntegrityInfraPipeline.
func New(client *dagger.Client, cfg pipelines.Config) pipelines.Pipeline {
	var daggerClient pipelines.DaggerClient
	var src pipelines.DaggerDirectory

	// Handle nil client gracefully
	if client != nil {
		// Convert real Dagger client to our interface using adapter
		daggerClient = pipelines.NewDaggerAdapter(client)
		src = daggerClient.Host().Directory(".", pipelines.DaggerHostDirectoryOpts{
			Exclude: []string{"**/node_modules", "**/.git", "**/.dagger-cache"},
		})
	}
	// If client is nil, all fields will remain nil

	return &SyntegrityInfraPipeline{
		Client: daggerClient,
		Config: cfg,
		Src:    src,
	}
}

func (s *SyntegrityInfraPipeline) Test(_ context.Context) error {
	return nil
}

func (s *SyntegrityInfraPipeline) Build(_ context.Context) error {
	return nil
}

func (s *SyntegrityInfraPipeline) Package(_ context.Context) error {
	return nil
}

func (s *SyntegrityInfraPipeline) Tag(_ context.Context) error {
	return errors.New("not implemented: Tag method is not yet implemented for infra pipeline")
}

func (s *SyntegrityInfraPipeline) Name() string {
	return "syntegrity-infra"
}

func (s *SyntegrityInfraPipeline) Setup(ctx context.Context) error {
	logger.L().InfoContext(ctx, "Running tests for syntegrity-infra")

	// Check if client is nil
	if s.Client == nil {
		return errors.New("Setup method requires real Dagger client, not nil")
	}

	if s.Cloner != nil {
		// Extract real client from adapter
		realClient := s.Client.(*pipelines.DaggerAdapter).GetRealClient()
		_, err := s.Cloner.Clone(ctx, realClient, shared.GitCloneOpts{})
		if err != nil {
			return err
		}
		// Convert real directory to our interface
		s.Src = pipelines.NewDaggerAdapter(realClient).Host().Directory(".", pipelines.DaggerHostDirectoryOpts{})
	}
	// Extract real types for shared functions (only if using adapter, not mocks)
	if adapter, ok := s.Client.(*pipelines.DaggerAdapter); ok {
		realClient := adapter.GetRealClient()
		if srcAdapter, ok := s.Src.(*pipelines.DaggerDirectoryAdapter); ok {
			realSrc := srcAdapter.GetRealDirectory()
			return shared.RunTestsWithCoverage(ctx, realClient, realSrc, s.Config.Coverage, s.Config.GoVersion)
		}
	}
	// For mocks, return an error indicating this requires real client
	return errors.New("Setup method requires real Dagger client, not mock")
}

func (s *SyntegrityInfraPipeline) Push(_ context.Context) error {
	return errors.New("not implemented: Push method is not yet implemented for infra pipeline")
}

func (s *SyntegrityInfraPipeline) BeforeStep(_ context.Context, _ string) pipelines.HookFunc {
	// Not implemented yet - return nil to indicate no hook
	return nil
}

func (s *SyntegrityInfraPipeline) AfterStep(_ context.Context, _ string) pipelines.HookFunc {
	// Not implemented yet - return nil to indicate no hook
	return nil
}
