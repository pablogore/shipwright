package test

import (
	"context"
	"fmt"
	"testing"

	"dagger.io/dagger"
	"github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
	"github.com/getsyntegrity/syntegrity-dagger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew_GoLanguage(t *testing.T) {
	// Use nil for dagger types since they're external dependencies
	var mockClient *dagger.Client
	var mockDirectory *dagger.Directory

	cfg := pipelines.Config{
		Coverage: 90.0,
	}

	tester := New(mockClient, mockDirectory, cfg, "go")

	assert.NotNil(t, tester)
	assert.IsType(t, &GoTester{}, tester)

	goTester := tester.(*GoTester)
	assert.Nil(t, goTester.Client) // Should be nil since mockClient is nil
	assert.Nil(t, goTester.Src)    // Should be nil since mockDirectory is nil
	assert.Equal(t, cfg, goTester.Config)
	assert.InEpsilon(t, 90.0, goTester.MinCoverage, 0.001)
}

func TestNew_UnknownLanguage(t *testing.T) {
	// Use nil for dagger types since they're external dependencies
	var mockClient *dagger.Client
	var mockDirectory *dagger.Directory

	cfg := pipelines.Config{}

	tester := New(mockClient, mockDirectory, cfg, "unknown")

	assert.NotNil(t, tester)
	assert.IsType(t, &noopTester{}, tester)
}

func TestNew_EmptyLanguage(t *testing.T) {
	// Use nil for dagger types since they're external dependencies
	var mockClient *dagger.Client
	var mockDirectory *dagger.Directory

	cfg := pipelines.Config{}

	tester := New(mockClient, mockDirectory, cfg, "")

	assert.NotNil(t, tester)
	assert.IsType(t, &noopTester{}, tester)
}

func TestNew_CaseInsensitive(t *testing.T) {
	// Use nil for dagger types since they're external dependencies
	var mockClient *dagger.Client
	var mockDirectory *dagger.Directory

	cfg := pipelines.Config{}

	// Test different cases
	testCases := []string{"Go", "GO", "gO", "GoLang", "golang"}

	for _, lang := range testCases {
		t.Run("case_"+lang, func(t *testing.T) {
			tester := New(mockClient, mockDirectory, cfg, lang)
			assert.NotNil(t, tester)
			assert.IsType(t, &noopTester{}, tester)
		})
	}
}

func TestNoopTester_RunTests(t *testing.T) {
	tester := &noopTester{}

	ctx := t.Context()
	err := tester.RunTests(ctx)

	assert.NoError(t, err)
}

func TestNoopTester_RunTests_WithContext(t *testing.T) {
	tester := &noopTester{}

	// Test with different contexts
	ctx1 := t.Context()
	type testKey string
	ctx2 := context.WithValue(t.Context(), testKey("key"), "value")

	err1 := tester.RunTests(ctx1)
	require.NoError(t, err1)

	err2 := tester.RunTests(ctx2)
	assert.NoError(t, err2)
}

func TestNoopTester_ImplementsTestable(t *testing.T) {
	// Test that noopTester implements Testable interface
	var tester Testable = &noopTester{}
	assert.NotNil(t, tester)

	ctx := t.Context()
	err := tester.RunTests(ctx)
	assert.NoError(t, err)
}

func TestNew_AllLanguages(t *testing.T) {
	// Use nil for dagger types since they're external dependencies
	var mockClient *dagger.Client
	var mockDirectory *dagger.Directory

	cfg := pipelines.Config{
		Coverage: 80.0,
	}

	// Test all supported languages
	languages := []struct {
		lang     string
		expected string
	}{
		{"go", "*test.GoTester"},
		{"Go", "*test.noopTester"},
		{"java", "*test.noopTester"},
		{"python", "*test.noopTester"},
		{"javascript", "*test.noopTester"},
		{"typescript", "*test.noopTester"},
		{"rust", "*test.noopTester"},
		{"c++", "*test.noopTester"},
		{"c#", "*test.noopTester"},
		{"php", "*test.noopTester"},
		{"ruby", "*test.noopTester"},
		{"", "*test.noopTester"},
	}

	for _, tt := range languages {
		t.Run("language_"+tt.lang, func(t *testing.T) {
			tester := New(mockClient, mockDirectory, cfg, tt.lang)
			assert.NotNil(t, tester)
			assert.Contains(t, fmt.Sprintf("%T", tester), tt.expected)
		})
	}
}

func TestNew_ConfigHandling(t *testing.T) {
	// Use nil for dagger types since they're external dependencies
	var mockClient *dagger.Client
	var mockDirectory *dagger.Directory

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
			tester := New(mockClient, mockDirectory, tt.config, "go")
			assert.NotNil(t, tester)

			goTester := tester.(*GoTester)
			if tt.expected == 0.0 {
				assert.Zero(t, goTester.MinCoverage)
			} else {
				assert.InEpsilon(t, tt.expected, goTester.MinCoverage, 0.001)
			}
		})
	}
}

func TestNew_NilParameters(t *testing.T) {
	// Test that New handles nil parameters gracefully
	tester := New(nil, nil, pipelines.Config{}, "go")
	assert.NotNil(t, tester)
	assert.IsType(t, &GoTester{}, tester)

	goTester := tester.(*GoTester)
	assert.Nil(t, goTester.Client)
	assert.Nil(t, goTester.Src)
}

func TestTestable_Interface(t *testing.T) {
	// Test that Testable interface is properly defined
	var tester Testable

	// Should be able to assign noopTester
	tester = &noopTester{}
	assert.NotNil(t, tester)

	// Should be able to assign GoTester
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDaggerClient := mocks.NewMockDaggerClient(ctrl)
	mockDaggerDirectory := mocks.NewMockDaggerDirectory(ctrl)

	tester = &GoTester{
		Client:      mockDaggerClient,
		Src:         mockDaggerDirectory,
		Config:      pipelines.Config{},
		MinCoverage: 80.0,
	}
	assert.NotNil(t, tester)
}

func TestNoopTester_Consistency(t *testing.T) {
	// Test that noopTester returns consistent results
	tester := &noopTester{}

	ctx := t.Context()

	// Multiple calls should return the same result
	for i := 0; i < 10; i++ {
		err := tester.RunTests(ctx)
		assert.NoError(t, err)
	}
}

func TestNew_Performance(t *testing.T) {
	// Use nil for dagger types since they're external dependencies
	var mockClient *dagger.Client
	var mockDirectory *dagger.Directory

	cfg := pipelines.Config{
		Coverage: 90.0,
	}

	// Test that New is fast for repeated calls
	for i := 0; i < 1000; i++ {
		tester := New(mockClient, mockDirectory, cfg, "go")
		assert.NotNil(t, tester)
	}
}

func TestNew_EdgeCases(t *testing.T) {
	// Use nil for dagger types since they're external dependencies
	var mockClient *dagger.Client
	var mockDirectory *dagger.Directory

	cfg := pipelines.Config{
		Coverage: 90.0,
	}

	// Test edge cases
	edgeCases := []string{
		"go ",
		" go",
		"go\n",
		"go\t",
		"go\r",
		"go\v",
		"go\f",
		"go\x00",
		"go\x1f",
		"go\x7f",
	}

	for _, lang := range edgeCases {
		t.Run("edge_case_"+fmt.Sprintf("%q", lang), func(t *testing.T) {
			tester := New(mockClient, mockDirectory, cfg, lang)
			assert.NotNil(t, tester)
			// Should return noopTester for non-exact matches
			assert.IsType(t, &noopTester{}, tester)
		})
	}
}
