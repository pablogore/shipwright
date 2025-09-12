// Package test provides testing utilities and implementations for pipelines.
package test

import (
	"context"
	"fmt"

	"dagger.io/dagger"
	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
)

// Testable is an interface that defines a contract for running tests.
type Testable interface {
	RunTests(ctx context.Context) error
}

// New returns a Testable implementation based on the configured language.
// Extend this switch to support other languages.
func New(client *dagger.Client, _ *dagger.Directory, cfg pipelines.Config, language string) Testable {
	switch language {
	case "go":
		// Only create adapter if client is not nil
		if client != nil {
			// Convert real Dagger client to our interface using adapter
			daggerClient := pipelines.NewDaggerAdapter(client)
			// Convert real Dagger directory to our interface using adapter
			daggerSrc := pipelines.NewDaggerAdapter(client).Host().Directory(".", pipelines.DaggerHostDirectoryOpts{})
			return &GoTester{
				Client:      daggerClient,
				Src:         daggerSrc,
				Config:      cfg,
				MinCoverage: cfg.Coverage,
			}
		}
		// For nil client, return a GoTester with nil client (for testing)
		return &GoTester{
			Client:      nil,
			Src:         nil,
			Config:      cfg,
			MinCoverage: cfg.Coverage,
		}
	default:
		return &noopTester{}
	}
}

// noopTester implements Testable but does nothing — fallback or placeholder.
type noopTester struct{}

func (n *noopTester) RunTests(_ context.Context) error {
	fmt.Println("⚠️ No valid tester available for selected language.")
	return nil
}
