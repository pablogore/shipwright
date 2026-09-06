// Package graph_test exercises internal/workflow/graph's stage-5
// validation pipeline (design.md D-H/D-J, tasks.md Phase 6): dependency
// graph construction from needs[], the undeclared-data-reference and
// kind checks that are statically knowable without a resolved provider.
// Cycle-detection tests live in kahn_test.go.
package graph_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
)

// tasks.md 6.7: duplicate step ids rejected — graph.Build re-validates
// this defensively; it is directly callable without
// manifest.ValidateStructure having run first, mirroring every other
// stage package in this pipeline (internal/workflow/interp is likewise
// independently testable without manifest.Parse).
func TestBuild_DuplicateStepIDRejected(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "build"},
		{ID: "build"},
	}

	_, err := graph.Build(steps)
	if err == nil {
		t.Fatal("Build() error = nil, want a duplicate step id error")
	}

	var dupErr *graph.DuplicateStepIDError
	if !errors.As(err, &dupErr) {
		t.Fatalf("Build() error = %v (%T), want *graph.DuplicateStepIDError", err, err)
	}
	if dupErr.StepID != "build" {
		t.Fatalf("DuplicateStepIDError.StepID = %q, want %q", dupErr.StepID, "build")
	}
	if want := `graph: duplicate step id "build"`; dupErr.Error() != want {
		t.Fatalf("DuplicateStepIDError.Error() = %q, want %q", dupErr.Error(), want)
	}
}

// tasks.md 6.6: needs[] naming an unknown step id is rejected, and its
// error type must be DISTINCT from CycleError so a caller can tell the
// two failure modes apart with errors.As, not string matching.
func TestBuild_UnknownNeedsRejected(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "deploy", Needs: []string{"ghost"}},
	}

	_, err := graph.Build(steps)
	if err == nil {
		t.Fatal("Build() error = nil, want an unknown-needs error")
	}

	var unknownErr *graph.UnknownNeedsError
	if !errors.As(err, &unknownErr) {
		t.Fatalf("Build() error = %v (%T), want *graph.UnknownNeedsError", err, err)
	}
	if unknownErr.StepID != "deploy" || unknownErr.UnknownID != "ghost" {
		t.Fatalf("UnknownNeedsError = %+v, want StepID=deploy UnknownID=ghost", unknownErr)
	}
	if want := `graph: step "deploy" needs unknown step "ghost"`; unknownErr.Error() != want {
		t.Fatalf("UnknownNeedsError.Error() = %q, want %q", unknownErr.Error(), want)
	}

	var cycleErr *graph.CycleError
	if errors.As(err, &cycleErr) {
		t.Fatal("unknown-needs error also matches *graph.CycleError — the two failure modes must be distinct types")
	}
}

// tasks.md 6.4: diamond fan-in is ACCEPTED, never flagged as a cycle —
// the false-positive risk design.md names as equally dangerous as a
// missed cycle. Also proves Kahn's wave production (tasks.md 6.10) for
// the shape design.md's own manifest example uses (build -> unit,vuln ->
// publish).
func TestBuild_DiamondFanInAccepted(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "build"},
		{ID: "unit", Needs: []string{"build"}},
		{ID: "vuln", Needs: []string{"build"}},
		{ID: "publish", Needs: []string{"unit", "vuln"}},
	}

	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("Build() error = %v, want nil — diamond fan-in must be accepted", err)
	}

	wantWaves := [][]string{
		{"build"},
		{"unit", "vuln"},
		{"publish"},
	}
	assertWaves(t, g.Waves, wantWaves)
}

// tasks.md 6.5: disconnected components (multiple independent roots, no
// edges between them) are accepted, and both roots land in the same
// first wave since neither depends on the other.
func TestBuild_DisconnectedComponentsAccepted(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "build-a"},
		{ID: "build-b"},
		{ID: "deploy-a", Needs: []string{"build-a"}},
	}

	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("Build() error = %v, want nil — disconnected components must be accepted", err)
	}

	wantWaves := [][]string{
		{"build-a", "build-b"},
		{"deploy-a"},
	}
	assertWaves(t, g.Waves, wantWaves)
}

// tasks.md 6.8: a steps.<id>.output reference in a step's input/with
// fields, to a step id NOT present in that step's own needs[], is
// rejected — reading another step's output without a declared ordering
// dependency would be an unordered/racy read once execution ever widens
// within a wave (design.md D-K).
func TestBuild_DataReferenceWithoutNeedsRejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		steps     []manifest.Step
		wantField string
	}{
		{
			name: "input field references an undeclared step",
			steps: []manifest.Step{
				{ID: "build"},
				{ID: "publish", Input: "${{ steps.build.output }}"},
			},
			wantField: "input",
		},
		{
			name: "with field references an undeclared step",
			steps: []manifest.Step{
				{ID: "build"},
				{ID: "publish", With: map[string]any{"ref": "${{ steps.build.output }}"}},
			},
			wantField: "with.ref",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := graph.Build(tt.steps)
			if err == nil {
				t.Fatal("Build() error = nil, want an undeclared data reference error")
			}

			var refErr *graph.UndeclaredDataReferenceError
			if !errors.As(err, &refErr) {
				t.Fatalf("Build() error = %v (%T), want *graph.UndeclaredDataReferenceError", err, err)
			}
			if refErr.StepID != "publish" || refErr.ReferencedStepID != "build" || refErr.Field != tt.wantField {
				t.Fatalf("UndeclaredDataReferenceError = %+v, want StepID=publish ReferencedStepID=build Field=%s", refErr, tt.wantField)
			}
			wantMsg := fmt.Sprintf(
				`graph: step "publish" field %q references steps.build.output, but "build" is not in step "publish"'s needs`,
				tt.wantField,
			)
			if refErr.Error() != wantMsg {
				t.Fatalf("UndeclaredDataReferenceError.Error() = %q, want %q", refErr.Error(), wantMsg)
			}
		})
	}
}

// Triangulation: the same data references ARE accepted once the
// referenced step is declared in needs[] — proves 6.8's check is about
// needs[] membership, not about rejecting steps.output references
// outright.
func TestBuild_DataReferenceWithNeedsAccepted(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "build"},
		{
			ID:    "publish",
			Needs: []string{"build"},
			Input: "${{ steps.build.output }}",
			With:  map[string]any{"ref": "${{ variables.imageRef }}"},
		},
	}

	if _, err := graph.Build(steps); err != nil {
		t.Fatalf("Build() error = %v, want nil — steps.build.output is declared in needs", err)
	}
}

// tasks.md 6.9: a step's declared input is a Directory-typed field
// (design.md D-H); a secrets.* reference — unconditionally KindSecret
// per interp.Reference.StaticKind — can never satisfy that, and this IS
// statically knowable without any provider schema. See build.go's doc
// comment for exactly what is and is not checked here (steps.<id>.output
// kind checking is deferred to Phase 7 — StaticKind reports ok=false for
// it, on purpose).
func TestBuild_InputSecretReferenceRejected(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "deploy", Input: "${{ secrets.registry }}"},
	}

	_, err := graph.Build(steps)
	if err == nil {
		t.Fatal("Build() error = nil, want a kind-mismatch error")
	}

	var kindErr *graph.KindMismatchError
	if !errors.As(err, &kindErr) {
		t.Fatalf("Build() error = %v (%T), want *graph.KindMismatchError", err, err)
	}
	if kindErr.StepID != "deploy" || kindErr.Field != "input" {
		t.Fatalf("KindMismatchError = %+v, want StepID=deploy Field=input", kindErr)
	}
	if kindErr.Got != interp.KindSecret {
		t.Fatalf("KindMismatchError.Got = %v, want KindSecret", kindErr.Got)
	}
	if want := `graph: step "deploy" field "input" has kind secret, which cannot satisfy its directory-typed input`; kindErr.Error() != want {
		t.Fatalf("KindMismatchError.Error() = %q, want %q", kindErr.Error(), want)
	}
}

// Triangulation: an input referencing a variable (KindString, also
// statically known) is accepted — proves the check flags KindSecret
// specifically, not every statically-known kind.
func TestBuild_InputVariableReferenceAccepted(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "deploy", Input: "${{ variables.workdir }}"},
	}

	if _, err := graph.Build(steps); err != nil {
		t.Fatalf("Build() error = %v, want nil — a variables.* input is a valid string, not a kind mismatch", err)
	}
}

// A needs[] entry duplicated verbatim must not inflate Kahn's in-degree
// bookkeeping — a manifest author writing needs: [build, build] declares
// exactly one dependency, and Build must accept it exactly like
// needs: [build].
func TestBuild_DuplicateNeedsEntryDeduplicated(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "build"},
		{ID: "test", Needs: []string{"build", "build"}},
	}

	g, err := graph.Build(steps)
	if err != nil {
		t.Fatalf("Build() error = %v, want nil — a duplicated needs[] entry is still one valid dependency", err)
	}
	assertWaves(t, g.Waves, [][]string{{"build"}, {"test"}})
}

// A malformed interpolation placeholder in a step's input field must
// propagate as an error rather than being silently ignored — Build
// calls interp.Scan while looking for data references (tasks.md 6.8),
// and a scan failure is a real validation failure, not a skippable one.
func TestBuild_MalformedInputInterpolationRejected(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "deploy", Input: "${{ not.a.valid.ref }}"},
	}

	if _, err := graph.Build(steps); err == nil {
		t.Fatal("Build() error = nil, want an interpolation scan error for a malformed input placeholder")
	}
}

// Same malformed-grammar propagation, but through a with field instead
// of input — proves checkFieldDataRefs' scan-error path is reached
// regardless of which field the reference appears in.
func TestBuild_MalformedWithInterpolationRejected(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "deploy", With: map[string]any{"ref": "${{ not.a.valid.ref }}"}},
	}

	if _, err := graph.Build(steps); err == nil {
		t.Fatal("Build() error = nil, want an interpolation scan error for a malformed with placeholder")
	}
}

// A step's input mixing literal text with a reference (not a "pure"
// single-reference form) is not something the current kind check can
// attribute a single Kind to — it must be accepted here, not
// mishandled as a mismatch or a panic on an empty tokens slice.
func TestBuild_InputWithMixedLiteralAndReferenceAccepted(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "deploy", Input: "prefix-${{ variables.workdir }}"},
	}

	if _, err := graph.Build(steps); err != nil {
		t.Fatalf("Build() error = %v, want nil — a mixed literal+reference input is not a checkable kind mismatch at this stage", err)
	}
}

// Triangulation: an input referencing another step's output has an
// UNDETERMINED static kind (StaticKind ok=false) — this stage must not
// guess, so it is accepted here and deferred to Phase 7 provider
// resolution (see build.go's doc comment).
func TestBuild_InputStepsOutputReferenceAcceptedPendingPhase7(t *testing.T) {
	t.Parallel()

	steps := []manifest.Step{
		{ID: "build"},
		{ID: "deploy", Needs: []string{"build"}, Input: "${{ steps.build.output }}"},
	}

	if _, err := graph.Build(steps); err != nil {
		t.Fatalf("Build() error = %v, want nil — steps.<id>.output kind is unknown until Phase 7, not a mismatch by default", err)
	}
}

func assertWaves(t *testing.T, got, want [][]string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("Waves = %v (%d waves), want %v (%d waves)", got, len(got), want, len(want))
	}
	for i := range want {
		if len(got[i]) != len(want[i]) {
			t.Fatalf("Waves[%d] = %v, want %v", i, got[i], want[i])
		}
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Fatalf("Waves[%d] = %v, want %v", i, got[i], want[i])
			}
		}
	}
}
