package shared

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTag_FromEnv(t *testing.T) {
	// Setup
	tagName := "test-tag-v1"
	t.Setenv("TAG_NAME", tagName)

	ctx := t.Context()
	client := requireDaggerClient(ctx, t)

	src := client.Directory()
	src = src.WithNewFile(".git/HEAD", "ref: refs/heads/main\n")

	// Test
	tag, err := GenerateTag(ctx, client, src)
	require.NoError(t, err)
	assert.Equal(t, tagName, tag)

	// Verify file was created
	content, err := os.ReadFile(".tag_name")
	require.NoError(t, err)
	assert.Equal(t, tagName, string(content))

	// Cleanup
	os.Remove(".tag_name")
}

func TestGenerateTag_FromGit(t *testing.T) {
	// Setup - ensure TAG_NAME is not set
	t.Setenv("TAG_NAME", "")

	ctx := t.Context()
	client := requireDaggerClient(ctx, t)

	// Create a fake git repo
	src := client.Directory()
	src = src.WithNewFile(".git/HEAD", "ref: refs/heads/main\n")
	src = src.WithNewFile(".git/refs/heads/main", "1234567890abcdef\n")
	src = src.WithNewFile(".git/config", "[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n\tbare = false\n\tlogallrefupdates = true\n")

	// Test
	tag, err := GenerateTag(ctx, client, src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error obtaining short commit hash")
	assert.Empty(t, tag)
}

func TestGenerateTag_Error_NoGit(t *testing.T) {
	// Setup - ensure TAG_NAME is not set
	t.Setenv("TAG_NAME", "")

	ctx := t.Context()
	client := requireDaggerClient(ctx, t)

	// Create a directory without git
	src := client.Directory()

	// Test
	tag, err := GenerateTag(ctx, client, src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error obtaining short commit hash")
	assert.Empty(t, tag)

	// Verify file was not created
	_, err = os.ReadFile(".tag_name")
	require.Error(t, err)
}

func TestSaveTagLocally(t *testing.T) {
	// Test
	tag := "test-tag"
	savedTag, err := saveTagLocally(tag)
	require.NoError(t, err)
	assert.Equal(t, tag, savedTag)

	// Verify file was created
	content, err := os.ReadFile(".tag_name")
	require.NoError(t, err)
	assert.Equal(t, tag, string(content))

	// Cleanup
	os.Remove(".tag_name")
}

func TestSaveTagLocally_Error(t *testing.T) {
	// Create a directory that can't be written to
	err := os.Mkdir(".tag_name", 0o000)
	require.NoError(t, err)
	defer os.Remove(".tag_name")

	// Test
	tag := "test-tag"
	savedTag, err := saveTagLocally(tag)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not save the tag to .tag_name")
	assert.Empty(t, savedTag)
}
