package daggerkit

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockDaggerDirectory_File_Entries proves the new read-side mock methods
// (design.md D-9) are correctly wired to testify's mock.Mock plumbing —
// GoRuntimeInspector's tests (double-selection order rule 1) depend on this.
func TestMockDaggerDirectory_File_Entries(t *testing.T) {
	mockDir := &MockDaggerDirectory{}
	mockFile := &MockDaggerFile{}

	mockDir.On("File", "go.mod").Return(mockFile)
	mockDir.On("Entries", context.Background()).Return([]string{"go.mod", "go.sum"}, nil)

	gotFile := mockDir.File("go.mod")
	assert.Same(t, mockFile, gotFile)

	entries, err := mockDir.Entries(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"go.mod", "go.sum"}, entries)
}

// TestMockDaggerDirectory_Entries_PropagatesError proves a mocked engine
// failure surfaces as a plain Go error, exactly as GoRuntimeInspector's
// nil-safe error handling expects.
func TestMockDaggerDirectory_Entries_PropagatesError(t *testing.T) {
	mockDir := &MockDaggerDirectory{}
	wantErr := errors.New("engine unavailable")

	mockDir.On("Entries", context.Background()).Return(nil, wantErr)

	entries, err := mockDir.Entries(context.Background())
	assert.Nil(t, entries)
	assert.Equal(t, wantErr, err)
}

// TestMockDaggerFile_Contents proves the new DaggerFile.Contents mock method
// is correctly wired to testify's mock.Mock plumbing.
func TestMockDaggerFile_Contents(t *testing.T) {
	mockFile := &MockDaggerFile{}
	mockFile.On("Contents", context.Background()).Return("module example.com/x\n\ngo 1.26.7\n", nil)

	contents, err := mockFile.Contents(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "module example.com/x\n\ngo 1.26.7\n", contents)
}

// TestMockDaggerDirectory_WithNewFile proves the write-side mock method
// (design.md D-9) is correctly wired to testify's mock.Mock plumbing —
// GoRuntimeUpgrader depends on this to write mutated go.mod/go.work/
// .go-version content and the report file into a returned Directory.
func TestMockDaggerDirectory_WithNewFile(t *testing.T) {
	mockDir := &MockDaggerDirectory{}
	mockUpdatedDir := &MockDaggerDirectory{}

	mockDir.On("WithNewFile", "go.mod", "module example.com/x\n\ngo 1.27.0\n").Return(mockUpdatedDir)

	got := mockDir.WithNewFile("go.mod", "module example.com/x\n\ngo 1.27.0\n")
	assert.Same(t, mockUpdatedDir, got)
}
