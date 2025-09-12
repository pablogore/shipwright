package test

import (
	"testing"

	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
	"github.com/getsyntegrity/syntegrity-dagger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGoTester_ImplementsTestable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDaggerClient(ctrl)
	mockDirectory := mocks.NewMockDaggerDirectory(ctrl)

	cfg := pipelines.Config{
		Coverage: 85.0,
	}

	// Test that GoTester implements Testable interface
	var tester Testable = &GoTester{
		Client:      mockClient,
		Src:         mockDirectory,
		Config:      cfg,
		MinCoverage: cfg.Coverage,
	}

	assert.NotNil(t, tester)
}

func TestGoTester_Fields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDaggerClient(ctrl)
	mockDirectory := mocks.NewMockDaggerDirectory(ctrl)

	cfg := pipelines.Config{
		Coverage: 90.0,
	}

	tester := &GoTester{
		Client:      mockClient,
		Src:         mockDirectory,
		Config:      cfg,
		MinCoverage: cfg.Coverage,
	}

	// Test that fields are set correctly
	assert.Equal(t, mockClient, tester.Client)
	assert.Equal(t, mockDirectory, tester.Src)
	assert.Equal(t, cfg, tester.Config)
	assert.InEpsilon(t, 90.0, tester.MinCoverage, 0.001)
}

func TestGoTester_WithNilValues(t *testing.T) {
	cfg := pipelines.Config{
		Coverage: 80.0,
	}

	tester := &GoTester{
		Client:      nil,
		Src:         nil,
		Config:      cfg,
		MinCoverage: cfg.Coverage,
	}

	// Test that tester can be created with nil values
	assert.NotNil(t, tester)
	assert.Nil(t, tester.Client)
	assert.Nil(t, tester.Src)
	assert.Equal(t, cfg, tester.Config)
	assert.InEpsilon(t, 80.0, tester.MinCoverage, 0.001)
}

func TestGoTester_ConfigHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDaggerClient(ctrl)
	mockDirectory := mocks.NewMockDaggerDirectory(ctrl)

	tests := []struct {
		name     string
		config   pipelines.Config
		expected float64
	}{
		{
			name: "zero coverage",
			config: pipelines.Config{
				Coverage: 0.0,
			},
			expected: 0.0,
		},
		{
			name: "high coverage",
			config: pipelines.Config{
				Coverage: 100.0,
			},
			expected: 100.0,
		},
		{
			name: "negative coverage",
			config: pipelines.Config{
				Coverage: -10.0,
			},
			expected: -10.0,
		},
		{
			name:     "default config",
			config:   pipelines.Config{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := &GoTester{
				Client:      mockClient,
				Src:         mockDirectory,
				Config:      tt.config,
				MinCoverage: tt.config.Coverage,
			}

			assert.NotNil(t, tester)
			if tt.expected == 0.0 {
				assert.Zero(t, tester.MinCoverage)
			} else {
				assert.InEpsilon(t, tt.expected, tester.MinCoverage, 0.001)
			}
		})
	}
}

func TestGoTester_RunTests_WithMocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDaggerClient(ctrl)
	mockDirectory := mocks.NewMockDaggerDirectory(ctrl)
	mockCacheVolume := mocks.NewMockDaggerCacheVolume(ctrl)
	mockContainer := mocks.NewMockDaggerContainer(ctrl)
	mockFile := mocks.NewMockDaggerFile(ctrl)

	cfg := pipelines.Config{
		Coverage: 85.0,
	}

	// Set up mock expectations
	mockClient.EXPECT().CacheVolume("go-mod-cache").Return(mockCacheVolume).Times(1)
	mockClient.EXPECT().CacheVolume("go-build-cache").Return(mockCacheVolume).Times(1)
	mockClient.EXPECT().Container().Return(mockContainer).Times(1)

	// Mock the container chain
	mockContainer.EXPECT().From("golang:1.21-alpine").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedDirectory("/app", mockDirectory).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedCache("/go/pkg/mod", mockCacheVolume).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedCache("/root/.cache/go-build", mockCacheVolume).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithWorkdir("/app").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithEnvVariable("GOPATH", "/go").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithEnvVariable("GOCACHE", "/root/.cache/go-build").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithExec(gomock.Any(), gomock.Any()).Return(mockContainer).Times(2)
	mockContainer.EXPECT().File("/tmp/coverage.txt").Return(mockFile).Times(1)

	// Mock file contents
	mockFile.EXPECT().Contents(gomock.Any()).Return("total: (statements) 90.0%", nil).Times(1)

	tester := &GoTester{
		Client:      mockClient,
		Src:         mockDirectory,
		Config:      cfg,
		MinCoverage: cfg.Coverage,
	}

	// Test RunTests method
	ctx := t.Context()
	err := tester.RunTests(ctx)

	// Should succeed since coverage (90%) is above threshold (85%)
	assert.NoError(t, err)
}

func TestGoTester_RunTests_FileContentsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDaggerClient(ctrl)
	mockDirectory := mocks.NewMockDaggerDirectory(ctrl)
	mockCacheVolume := mocks.NewMockDaggerCacheVolume(ctrl)
	mockContainer := mocks.NewMockDaggerContainer(ctrl)
	mockFile := mocks.NewMockDaggerFile(ctrl)

	cfg := pipelines.Config{
		Coverage: 85.0,
	}

	// Set up mock expectations
	mockClient.EXPECT().CacheVolume("go-mod-cache").Return(mockCacheVolume).Times(1)
	mockClient.EXPECT().CacheVolume("go-build-cache").Return(mockCacheVolume).Times(1)
	mockClient.EXPECT().Container().Return(mockContainer).Times(1)

	// Mock the container chain
	mockContainer.EXPECT().From("golang:1.21-alpine").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedDirectory("/app", mockDirectory).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedCache("/go/pkg/mod", mockCacheVolume).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedCache("/root/.cache/go-build", mockCacheVolume).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithWorkdir("/app").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithEnvVariable("GOPATH", "/go").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithEnvVariable("GOCACHE", "/root/.cache/go-build").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithExec(gomock.Any(), gomock.Any()).Return(mockContainer).Times(2)
	mockContainer.EXPECT().File("/tmp/coverage.txt").Return(mockFile).Times(1)

	// Mock file contents error
	mockFile.EXPECT().Contents(gomock.Any()).Return("", assert.AnError).Times(1)

	tester := &GoTester{
		Client:      mockClient,
		Src:         mockDirectory,
		Config:      cfg,
		MinCoverage: cfg.Coverage,
	}

	// Test RunTests method
	ctx := t.Context()
	err := tester.RunTests(ctx)

	// Should fail with file contents error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Error generating coverage report")
}

func TestGoTester_RunTests_InsufficientCoverage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDaggerClient(ctrl)
	mockDirectory := mocks.NewMockDaggerDirectory(ctrl)
	mockCacheVolume := mocks.NewMockDaggerCacheVolume(ctrl)
	mockContainer := mocks.NewMockDaggerContainer(ctrl)
	mockFile := mocks.NewMockDaggerFile(ctrl)

	cfg := pipelines.Config{
		Coverage: 85.0,
	}

	// Set up mock expectations
	mockClient.EXPECT().CacheVolume("go-mod-cache").Return(mockCacheVolume).Times(1)
	mockClient.EXPECT().CacheVolume("go-build-cache").Return(mockCacheVolume).Times(1)
	mockClient.EXPECT().Container().Return(mockContainer).Times(1)

	// Mock the container chain
	mockContainer.EXPECT().From("golang:1.21-alpine").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedDirectory("/app", mockDirectory).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedCache("/go/pkg/mod", mockCacheVolume).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedCache("/root/.cache/go-build", mockCacheVolume).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithWorkdir("/app").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithEnvVariable("GOPATH", "/go").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithEnvVariable("GOCACHE", "/root/.cache/go-build").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithExec(gomock.Any(), gomock.Any()).Return(mockContainer).Times(2)
	mockContainer.EXPECT().File("/tmp/coverage.txt").Return(mockFile).Times(1)

	// Mock file contents with insufficient coverage
	mockFile.EXPECT().Contents(gomock.Any()).Return("total: (statements) 80.0%", nil).Times(1)

	tester := &GoTester{
		Client:      mockClient,
		Src:         mockDirectory,
		Config:      cfg,
		MinCoverage: cfg.Coverage,
	}

	// Test RunTests method
	ctx := t.Context()
	err := tester.RunTests(ctx)

	// Should fail with insufficient coverage error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Insufficient coverage")
	assert.Contains(t, err.Error(), "80.0%")
	assert.Contains(t, err.Error(), "85.0%")
}

func TestGoTester_RunTests_NoTotalLine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDaggerClient(ctrl)
	mockDirectory := mocks.NewMockDaggerDirectory(ctrl)
	mockCacheVolume := mocks.NewMockDaggerCacheVolume(ctrl)
	mockContainer := mocks.NewMockDaggerContainer(ctrl)
	mockFile := mocks.NewMockDaggerFile(ctrl)

	cfg := pipelines.Config{
		Coverage: 85.0,
	}

	// Set up mock expectations
	mockClient.EXPECT().CacheVolume("go-mod-cache").Return(mockCacheVolume).Times(1)
	mockClient.EXPECT().CacheVolume("go-build-cache").Return(mockCacheVolume).Times(1)
	mockClient.EXPECT().Container().Return(mockContainer).Times(1)

	// Mock the container chain
	mockContainer.EXPECT().From("golang:1.21-alpine").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedDirectory("/app", mockDirectory).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedCache("/go/pkg/mod", mockCacheVolume).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedCache("/root/.cache/go-build", mockCacheVolume).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithWorkdir("/app").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithEnvVariable("GOPATH", "/go").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithEnvVariable("GOCACHE", "/root/.cache/go-build").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithExec(gomock.Any(), gomock.Any()).Return(mockContainer).Times(2)
	mockContainer.EXPECT().File("/tmp/coverage.txt").Return(mockFile).Times(1)

	// Mock file contents without total line
	mockFile.EXPECT().Contents(gomock.Any()).Return("some other content\nwithout total line", nil).Times(1)

	tester := &GoTester{
		Client:      mockClient,
		Src:         mockDirectory,
		Config:      cfg,
		MinCoverage: cfg.Coverage,
	}

	// Test RunTests method
	ctx := t.Context()
	err := tester.RunTests(ctx)

	// Should fail with no total line error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No line with total coverage found")
}

func TestGoTester_RunTests_CoverageParsingError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDaggerClient(ctrl)
	mockDirectory := mocks.NewMockDaggerDirectory(ctrl)
	mockCacheVolume := mocks.NewMockDaggerCacheVolume(ctrl)
	mockContainer := mocks.NewMockDaggerContainer(ctrl)
	mockFile := mocks.NewMockDaggerFile(ctrl)

	cfg := pipelines.Config{
		Coverage: 85.0,
	}

	// Set up mock expectations
	mockClient.EXPECT().CacheVolume("go-mod-cache").Return(mockCacheVolume).Times(1)
	mockClient.EXPECT().CacheVolume("go-build-cache").Return(mockCacheVolume).Times(1)
	mockClient.EXPECT().Container().Return(mockContainer).Times(1)

	// Mock the container chain
	mockContainer.EXPECT().From("golang:1.21-alpine").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedDirectory("/app", mockDirectory).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedCache("/go/pkg/mod", mockCacheVolume).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithMountedCache("/root/.cache/go-build", mockCacheVolume).Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithWorkdir("/app").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithEnvVariable("GOPATH", "/go").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithEnvVariable("GOCACHE", "/root/.cache/go-build").Return(mockContainer).Times(1)
	mockContainer.EXPECT().WithExec(gomock.Any(), gomock.Any()).Return(mockContainer).Times(2)
	mockContainer.EXPECT().File("/tmp/coverage.txt").Return(mockFile).Times(1)

	// Mock file contents with invalid coverage format
	mockFile.EXPECT().Contents(gomock.Any()).Return("total: (statements) invalid%", nil).Times(1)

	tester := &GoTester{
		Client:      mockClient,
		Src:         mockDirectory,
		Config:      cfg,
		MinCoverage: cfg.Coverage,
	}

	// Test RunTests method
	ctx := t.Context()
	err := tester.RunTests(ctx)

	// Should fail with coverage parsing error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Error parsing coverage")
}
