package providers

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// changelogUnreleasedHeading is the Keep a Changelog section header this
// Runner prepends generated entries under.
const changelogUnreleasedHeading = "## [Unreleased]"

// changelogFileName is the changelog file's path relative to the mounted
// source directory.
const changelogFileName = "CHANGELOG.md"

// changelogImage is the base image ChangelogRunner installs git into. A
// bare alpine image (mirroring providers/go's ContainerPublisher default
// runtime base, providers/go/containerpublisher.go) keeps the image small;
// git itself is not preinstalled on it, unlike golang:/rust:'s toolchain
// images, so it is added explicitly via apk.
const changelogImage = "alpine:latest"

// changelogConventionalCommitPattern extracts the type, optional scope,
// optional breaking-change bang, and description from a
// conventional-commit subject line (e.g. "feat(cli)!: add --dry-run
// flag"). Ported as-is from the former internal/app.ChangelogStepHandler
// (internal/app/changelog_step_handler.go's conventionalCommitPattern),
// which this Runner supersedes — see register.go's RegisterDefaults
// "changelog" entry.
var changelogConventionalCommitPattern = regexp.MustCompile(`(?i)^([a-zA-Z]+)(\([^)]*\))?(!)?:\s*(.*)$`)

// changelogCommitClassification buckets commit subjects into the three
// Keep a Changelog sections this repository's CHANGELOG.md actually uses.
type changelogCommitClassification struct {
	Added   []string
	Changed []string
	Fixed   []string
}

// ChangelogRunner generates a Keep a Changelog "Unreleased" summary from
// git history since the last tag and prepends it into build's CHANGELOG.md,
// returning the resulting Container. It is Shipwright's live-system
// replacement for the now-deleted internal/app.ChangelogStepHandler, which
// shelled out to git on the host and wrote directly to the host
// filesystem — behavior with no place in a capability contract whose only
// inputs/outputs are Dagger core types (mirroring
// providers/go/containerpublisher.go's own "Behavioral judgment calls"
// note about host filesystem side effects). Every text-processing helper
// below (classification, section building, prepend) is ported unchanged
// from that handler; only the git access and file I/O move inside a Dagger
// container.
//
// build is expected to contain a `.git` history — typically a checkout
// step's source Directory, not a Builder's compiled-output Directory (the
// Runner interface's build parameter name, pkg/shipwright/capabilities.go,
// describes the common case, not a hard requirement: any
// Directory-producing step can feed a Runner via `${{ steps.<id>.output
// }}` wiring, and this Runner's own contract is "a Directory with git
// history in it", whatever step produced it).
type ChangelogRunner struct {
	// Client is the Dagger client used to construct the git/changelog
	// container.
	Client *dagger.Client
}

// Compile-time conformance assertion: ChangelogRunner must satisfy Layer
// 1's Runner interface.
var _ shipwright.Runner = (*ChangelogRunner)(nil)

// Run reads build's git history, classifies commits since the last tag
// (or the full history when there is none) into Added/Changed/Fixed
// entries, and prepends the result into build's CHANGELOG.md under its
// "## [Unreleased]" heading (creating both the heading and the file when
// absent). When there are no classifiable commits, the changelog is left
// untouched.
func (r *ChangelogRunner) Run(ctx context.Context, build *dagger.Directory) (*dagger.Container, error) {
	if r.Client == nil {
		return nil, errors.New("changelogrunner: dagger client is not configured")
	}
	if build == nil {
		return nil, errors.New("changelogrunner: build directory is nil")
	}

	container := r.Client.Container().
		From(changelogImage).
		WithMountedDirectory("/work", build).
		WithWorkdir("/work").
		WithExec([]string{"apk", "add", "--no-cache", "git"}).
		// Dagger mounts /work with an owner that commonly differs from the
		// container's git user, which git's own ownership check refuses to
		// operate on ("detected dubious ownership") unless explicitly
		// trusted.
		WithExec([]string{"git", "config", "--global", "--add", "safe.directory", "/work"})

	lastTag := ""
	if tagOut, err := container.WithExec([]string{"git", "describe", "--tags", "--abbrev=0"}).Stdout(ctx); err == nil {
		lastTag = strings.TrimSpace(tagOut)
	}
	// A non-nil error above (no reachable tag, or no commits at all) is
	// the bootstrap case: fall through and diff the full history, mirroring
	// the former host-side handler's lastGitTag behavior.

	logArgs := []string{"git", "log", "--pretty=format:%s"}
	if lastTag != "" {
		logArgs = []string{"git", "log", lastTag + "..HEAD", "--pretty=format:%s"}
	}
	logOutput, err := container.WithExec(logArgs).Stdout(ctx)
	if err != nil {
		// An unborn HEAD (a genuinely fresh repository with zero commits)
		// makes `git log` itself fail, not just find nothing — the same "no
		// history yet" bootstrap case the `git describe` failure above
		// already tolerates. Treated as "no classifiable commits" rather
		// than a hard failure, matching Run's own doc comment.
		logOutput = ""
	}

	subjects := changelogCommitSubjects(logOutput)
	classification := classifyChangelogCommits(subjects)
	section := buildChangelogSection(classification)
	if section == "" {
		return container, nil
	}

	existing := ""
	if content, err := container.File(changelogFileName).Contents(ctx); err == nil {
		existing = content
	}
	// A non-nil error above means changelogFileName does not exist yet (or
	// is unreadable) — either way, treated as "no existing changelog",
	// mirroring the former handler's os.IsNotExist branch.

	updated := prependUnreleasedSection(existing, section)

	return container.WithNewFile(changelogFileName, updated), nil
}

// changelogCommitSubjects splits git log's `--pretty=format:%s` stdout
// into individual subject lines, mirroring the former host-side handler's
// commitSubjectsSince trimming/splitting behavior.
func changelogCommitSubjects(logOutput string) []string {
	trimmed := strings.TrimSpace(logOutput)
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// classifyChangelogCommits groups commit subjects the same way the former
// internal/app.ChangelogStepHandler did (breaking / feat / fix /
// everything else), collapsed onto this repository's Added/Changed/Fixed
// CHANGELOG.md sections. Ported unchanged from
// internal/app/changelog_step_handler.go's classifyCommits.
func classifyChangelogCommits(subjects []string) changelogCommitClassification {
	var c changelogCommitClassification

	for _, raw := range subjects {
		subject := strings.TrimSpace(raw)
		if subject == "" {
			continue
		}

		commitType, breakingBang, description := parseChangelogConventionalCommit(subject)
		isBreaking := breakingBang || strings.Contains(strings.ToUpper(subject), "BREAKING")

		switch {
		case isBreaking:
			c.Changed = append(c.Changed, "**BREAKING:** "+description)
		case strings.EqualFold(commitType, "feat") || strings.EqualFold(commitType, "feature"):
			c.Added = append(c.Added, description)
		case strings.EqualFold(commitType, "fix") || strings.EqualFold(commitType, "bugfix") || strings.EqualFold(commitType, "hotfix"):
			c.Fixed = append(c.Fixed, description)
		default:
			c.Changed = append(c.Changed, description)
		}
	}

	return c
}

// parseChangelogConventionalCommit splits a subject into its
// conventional-commit type, breaking-change bang, and description. When
// the subject doesn't follow the "type(scope)!: description" shape, the
// full subject is returned as the description unchanged. Ported unchanged
// from internal/app/changelog_step_handler.go's parseConventionalCommit.
func parseChangelogConventionalCommit(subject string) (commitType string, breaking bool, description string) {
	m := changelogConventionalCommitPattern.FindStringSubmatch(subject)
	if m == nil {
		return "", false, subject
	}
	return m[1], m[3] == "!", m[4]
}

// buildChangelogSection renders only the non-empty categories, in Keep a
// Changelog's canonical Added/Changed/Fixed order. Ported unchanged from
// internal/app/changelog_step_handler.go's buildChangelogSection.
func buildChangelogSection(c changelogCommitClassification) string {
	var parts []string

	if len(c.Added) > 0 {
		parts = append(parts, "### Added\n"+changelogBulletList(c.Added))
	}
	if len(c.Changed) > 0 {
		parts = append(parts, "### Changed\n"+changelogBulletList(c.Changed))
	}
	if len(c.Fixed) > 0 {
		parts = append(parts, "### Fixed\n"+changelogBulletList(c.Fixed))
	}

	return strings.Join(parts, "\n\n")
}

// changelogBulletList renders items as a "- item" list. Ported unchanged
// from internal/app/changelog_step_handler.go's bulletList.
func changelogBulletList(items []string) string {
	bullets := make([]string, len(items))
	for i, item := range items {
		bullets[i] = "- " + item
	}
	return strings.Join(bullets, "\n")
}

// prependUnreleasedSection inserts section directly under the
// "## [Unreleased]" heading, ahead of whatever is already there, so
// repeated runs accumulate entries instead of overwriting history. A
// no-op section returns existing untouched. When no Unreleased heading
// exists yet, one is created at the top of the document. Ported unchanged
// from internal/app/changelog_step_handler.go's prependUnreleasedSection.
func prependUnreleasedSection(existing, section string) string {
	if section == "" {
		return existing
	}

	idx := strings.Index(existing, changelogUnreleasedHeading)
	if idx == -1 {
		if existing == "" {
			return changelogUnreleasedHeading + "\n\n" + section + "\n"
		}
		return changelogUnreleasedHeading + "\n\n" + section + "\n\n" + existing
	}

	headingEnd := idx + len(changelogUnreleasedHeading)
	before := existing[:headingEnd]
	after := strings.TrimLeft(existing[headingEnd:], "\n")

	if after == "" {
		return before + "\n\n" + section + "\n"
	}
	return before + "\n\n" + section + "\n\n" + after
}
