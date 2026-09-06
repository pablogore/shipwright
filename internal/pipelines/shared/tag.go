package shared

import (
	"context"
	"fmt"
	"os"
	"strings"

	"dagger.io/dagger"
	"github.com/pablogore/kit-logger/pkg/logger"
)

// GenerateTag generates a tag and saves it on the host as .tag_name.
// It prioritizes the TAG_NAME environment variable; otherwise, it uses
// `git rev-parse --short HEAD` to generate the tag.
//
// Parameters:
//   - ctx: The context for managing execution.
//   - client: The Dagger client used to execute pipeline operations.
//   - src: A pointer to a Dagger directory representing the source repository.
//
// Returns:
//   - A string representing the generated tag.
//   - An error if the tag generation or saving process fails.
func GenerateTag(ctx context.Context, client *dagger.Client, src *dagger.Directory) (string, error) {
	// Check if TAG_NAME is set in the environment
	tag := os.Getenv("TAG_NAME")
	if tag != "" {
		return saveTagLocally(tag)
	}

	// Fallback: Use `git rev-parse` to get the short commit hash
	output, err := client.Container().
		From("alpine/git").
		WithMountedDirectory("/repo", src).
		WithWorkdir("/repo").
		WithExec([]string{"git", "rev-parse", "--short", "HEAD"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("❌ error obtaining short commit hash: %w", err)
	}

	// Generate the tag using the short commit hash
	tag = "v-" + strings.TrimSpace(output)
	return saveTagLocally(tag)
}

// saveTagLocally writes the tag to the .tag_name file on the host.
//
// Parameters:
//   - tag: The tag string to be saved.
//
// Returns:
//   - A string representing the saved tag.
//   - An error if the file writing process fails.
func saveTagLocally(tag string) (string, error) {
	err := os.WriteFile(".tag_name", []byte(tag), 0o644)
	if err != nil {
		return "", fmt.Errorf("❌ could not save the tag to .tag_name: %w", err)
	}
	logger.L().InfoContext(context.Background(), "Tag generated", "tag", tag)
	return tag, nil
}
