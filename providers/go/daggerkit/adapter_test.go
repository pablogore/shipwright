package daggerkit

import (
	"bytes"
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
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

// TestDaggerContainerAdapter_AsService_SetsInsecureRootCapabilities is a
// static, source-level assertion (design.md's Threat Matrix "Privileged
// subprocess" row, tasks.md 1.3): no live Dagger engine runs in this
// module's unit tests, so InsecureRootCapabilities cannot be observed by
// calling AsService against a real dagger.Container. Instead this parses
// adapter.go's own source and confirms the literal appears inside
// AsService's body — a stronger guarantee than a doc comment, since it
// fails if the option is ever removed or its value ever flipped to false.
func TestDaggerContainerAdapter_AsService_SetsInsecureRootCapabilities(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "adapter.go", nil, 0)
	require.NoError(t, err, "failed to parse adapter.go")

	var body *ast.BlockStmt
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if ok && fn.Name.Name == "AsService" && fn.Recv != nil {
			body = fn.Body
			return false
		}
		return true
	})
	require.NotNil(t, body, "DaggerContainerAdapter.AsService not found in adapter.go")

	var buf bytes.Buffer
	require.NoError(t, printer.Fprint(&buf, fset, body))
	// printer.Fprint renders struct-literal fields with tabs for alignment
	// (e.g. "InsecureRootCapabilities:\ttrue,"), so whitespace runs are
	// normalized to a single space before the substring check.
	normalized := strings.Join(strings.Fields(buf.String()), " ")
	assert.Contains(t, normalized, "InsecureRootCapabilities: true",
		"AsService must set InsecureRootCapabilities: true — required to start dockerd (design.md D-6)")
}
