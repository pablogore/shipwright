//go:build tools

package main

//go:generate mockgen -source=../internal/interfaces/interfaces.go -destination=interfaces_mock.go -package=mocks
//go:generate mockgen -source=../internal/pipelines/shared/cloner.go -destination=cloner_mock.go -package=mocks
//go:generate mockgen -source=../internal/pipelines/test/test.go -destination=test_mock.go -package=mocks
//go:generate mockgen -source=../internal/pipelines/dagger_interfaces.go -destination=dagger_interfaces_mock.go -package=mocks

// This file is used to generate mocks for all interfaces in the project.
// Run 'go generate ./mocks' to regenerate all mocks.
