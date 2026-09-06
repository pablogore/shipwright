// Package releaseguard guards design.md D6's release-automation decision.
// See tags_test.go's package doc comment for the full rationale.
package releaseguard

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// describeTagsCallRe matches a single-line `git describe --tags ...`
// invocation the way root release.yml writes it (one shell assignment per
// line), so each match can be checked in isolation for a `--match` flag.
var describeTagsCallRe = regexp.MustCompile(`git describe --tags[^\n]*`)

// UnfilteredDescribeCalls reads the workflow YAML at path and returns every
// `git describe --tags` invocation that lacks a `--match` flag -- design.md
// D6's blast-radius finding: without one, the first reachable
// providers/go/vX.Y.Z tag becomes LATEST_TAG and corrupts the root release
// line's derived version.
func UnfilteredDescribeCalls(filePath string) ([]string, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("releaseguard: read %s: %w", filePath, err)
	}

	var unfiltered []string
	for _, call := range describeTagsCallRe.FindAllString(string(raw), -1) {
		if !strings.Contains(call, "--match") {
			unfiltered = append(unfiltered, call)
		}
	}

	return unfiltered, nil
}

// workflowTriggers is the minimal shape releaseguard needs from a GitHub
// Actions workflow file: the literal `on.push.tags` glob list. Every other
// key (workflow_dispatch, permissions, jobs, ...) is intentionally ignored.
type workflowTriggers struct {
	On struct {
		Push struct {
			Tags []string `yaml:"tags"`
		} `yaml:"push"`
	} `yaml:"on"`
}

// PushTagGlobs reads the workflow YAML at filePath and returns its
// `on.push.tags` glob list, in file order.
func PushTagGlobs(filePath string) ([]string, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("releaseguard: read %s: %w", filePath, err)
	}

	var wf workflowTriggers
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		return nil, fmt.Errorf("releaseguard: parse %s: %w", filePath, err)
	}

	return wf.On.Push.Tags, nil
}

// GlobMatches reports whether tag matches the GitHub Actions tag glob
// pattern glob. GitHub Actions tag/branch filters use the same
// non-`/`-crossing `*` semantics as a shell filename glob (design.md D6:
// "Actions globs do not cross `/`"), which path.Match already implements.
// A malformed glob never matches anything, rather than erroring the guard.
func GlobMatches(glob, tag string) bool {
	ok, err := path.Match(glob, tag)
	if err != nil {
		return false
	}
	return ok
}

// shapeRegexAssignmentRe extracts the exact SHAPE_REGEX shell assignment
// release-provider-go.yml embeds in its shape-validation step, so this
// package tests the literal regex the workflow runs instead of a copy that
// could drift out of sync.
var shapeRegexAssignmentRe = regexp.MustCompile(`SHAPE_REGEX='([^']*)'`)

// ExtractShapeRegex reads the workflow YAML at filePath and returns the
// literal regex assigned to SHAPE_REGEX in its shape-validation step.
func ExtractShapeRegex(filePath string) (string, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("releaseguard: read %s: %w", filePath, err)
	}

	m := shapeRegexAssignmentRe.FindSubmatch(raw)
	if m == nil {
		return "", fmt.Errorf("releaseguard: no SHAPE_REGEX= assignment found in %s", filePath)
	}

	return string(m[1]), nil
}
