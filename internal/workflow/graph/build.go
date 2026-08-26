// Package graph implements stage 5 of the manifest's fixed seven-stage
// validation pipeline (design.md D-H): dependency-graph construction
// from each step's needs[], cycle detection via Kahn's algorithm
// (design.md D-J), rejection of an undeclared data reference (a
// steps.<id>.output interpolation reference to a step not present in
// the consuming step's own needs[]), and the output-kind/input-kind
// checks that are statically knowable without a resolved provider.
//
// Build is directly callable with a raw []manifest.Step slice — it does
// not assume manifest.ValidateStructure (stage 3) already ran, and
// re-validates duplicate ids and unknown needs[] references defensively
// for its own independent testability (tasks.md 6.6, 6.7), mirroring
// every other stage package in this pipeline (internal/workflow/interp
// is likewise independently testable without manifest.Parse).
//
// Kind checking (tasks.md 6.9) is deliberately PARTIAL, following the
// same honesty standard internal/workflow/interp's Reference.StaticKind
// set (tasks.md 5.3): a step's declared Input is a Directory-typed field
// (design.md D-H, workflow-manifest spec), and a secrets.* reference —
// unconditionally KindSecret by construction (interp.StaticKind) — can
// never satisfy that, so it is rejected here. That is the ONLY kind
// mismatch this package can prove without a provider schema. Two classes
// of check are explicitly NOT implemented here and deferred to
// internal/workflow/providers (Phase 7, design.md D-I):
//
//  1. A steps.<id>.output reference's Kind is fixed by whichever
//     capability produced that step, which is not knowable until stage
//     6 provider resolution — StaticKind reports ok=false for exactly
//     this case, and this package accepts it rather than guessing.
//  2. Kind compatibility for `with` field values against a provider's
//     declared WithSchema (stage 7, forbidPlaintext and general type
//     mismatch) requires the provider's field-kind declaration, which
//     does not exist until Phase 7 — this package has no schema to
//     compare a with-field's resolved kind against, so it does not
//     attempt that comparison.
package graph

import (
	"fmt"
	"sort"

	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
)

// Node is one step in the graph as this package's stage-5 validation
// sees it: just the id and the resolved, deduplicated set of upstream
// dependency ids (needs[]). Everything else about the step (capability,
// uses, with) belongs to a later phase's concern.
type Node struct {
	ID    string
	Needs []string
}

// Graph is the engine's compiled DAG artifact (design.md D-G: "the
// engine's compiled artifact is a Graph, never a 'plan'"). Waves is the
// topological schedule Kahn's algorithm produces: Waves[i] is the set of
// step ids that can run in round i, ordered within each wave by manifest
// declaration order (design.md D-K: sequential execution within a wave,
// in manifest-declaration order — the exact seam a worker pool drops
// into later).
type Graph struct {
	Nodes map[string]Node
	Waves [][]string
}

// DuplicateStepIDError reports a step id used more than once (tasks.md
// 6.7).
type DuplicateStepIDError struct{ StepID string }

func (e *DuplicateStepIDError) Error() string {
	return fmt.Sprintf("graph: duplicate step id %q", e.StepID)
}

// UnknownNeedsError reports a needs[] entry naming a step id that does
// not exist among the manifest's steps (tasks.md 6.6). Deliberately a
// distinct Go type from CycleError so a caller can tell "you referenced
// a step that doesn't exist" apart from "the steps you referenced form a
// cycle" with errors.As, not string matching.
type UnknownNeedsError struct{ StepID, UnknownID string }

func (e *UnknownNeedsError) Error() string {
	return fmt.Sprintf("graph: step %q needs unknown step %q", e.StepID, e.UnknownID)
}

// UndeclaredDataReferenceError reports a steps.<id>.output interpolation
// reference used in a step's input/with fields where <id> is not also
// present in that step's own declared needs[] (tasks.md 6.8) — reading
// another step's output without a declared ordering dependency would be
// an unordered/racy read once execution ever widens within a wave
// (design.md D-K).
type UndeclaredDataReferenceError struct{ StepID, ReferencedStepID, Field string }

func (e *UndeclaredDataReferenceError) Error() string {
	return fmt.Sprintf(
		"graph: step %q field %q references steps.%s.output, but %q is not in step %q's needs",
		e.StepID, e.Field, e.ReferencedStepID, e.ReferencedStepID, e.StepID,
	)
}

// KindMismatchError reports an interpolation reference whose statically
// known Kind cannot satisfy the structural expectation of the field it
// is used in (tasks.md 6.9). See this file's package doc comment for
// exactly what IS and is NOT checked.
type KindMismatchError struct {
	StepID, Field string
	Got           interp.Kind
}

func (e *KindMismatchError) Error() string {
	return fmt.Sprintf(
		"graph: step %q field %q has kind %s, which cannot satisfy its directory-typed input",
		e.StepID, e.Field, e.Got,
	)
}

// Build constructs a Graph from steps, running every stage-5 check
// (design.md D-H) in order: duplicate ids, unknown needs[] references,
// undeclared data references, statically-knowable kind mismatches, and
// finally Kahn's algorithm (cycle detection + topological waves).
func Build(steps []manifest.Step) (*Graph, error) {
	nodes, err := buildNodes(steps)
	if err != nil {
		return nil, err
	}

	if err := validateDataReferences(steps, nodes); err != nil {
		return nil, err
	}

	if err := validateKinds(steps); err != nil {
		return nil, err
	}

	waves, err := kahn(steps, nodes)
	if err != nil {
		return nil, err
	}

	return &Graph{Nodes: nodes, Waves: waves}, nil
}

// buildNodes constructs the node set and validates duplicate ids
// (tasks.md 6.7) and unknown needs[] references (tasks.md 6.6). It does
// NOT detect cycles — that is kahn's job, run after every other stage-5
// check has already admitted the graph's shape.
func buildNodes(steps []manifest.Step) (map[string]Node, error) {
	nodes := make(map[string]Node, len(steps))

	for _, s := range steps {
		if _, exists := nodes[s.ID]; exists {
			return nil, &DuplicateStepIDError{StepID: s.ID}
		}
		nodes[s.ID] = Node{ID: s.ID, Needs: dedupe(s.Needs)}
	}

	for _, s := range steps {
		for _, dep := range nodes[s.ID].Needs {
			if _, ok := nodes[dep]; !ok {
				return nil, &UnknownNeedsError{StepID: s.ID, UnknownID: dep}
			}
		}
	}

	return nodes, nil
}

// dedupe returns ids with duplicate entries removed, preserving first
// occurrence order. A manifest declaring the same needs[] entry twice
// must not inflate Kahn's in-degree count — that would misreport a
// perfectly valid graph as short one decrement, or worse, as a cycle.
func dedupe(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// validateDataReferences implements tasks.md 6.8: every steps.<id>.output
// interpolation reference found in a step's Input or With fields must
// name a step already present in that step's own needs[].
func validateDataReferences(steps []manifest.Step, nodes map[string]Node) error {
	for _, s := range steps {
		needs := make(map[string]bool, len(nodes[s.ID].Needs))
		for _, dep := range nodes[s.ID].Needs {
			needs[dep] = true
		}

		if err := checkFieldDataRefs(s.ID, "input", s.Input, needs); err != nil {
			return err
		}

		for _, key := range sortedKeys(s.With) {
			str, ok := s.With[key].(string)
			if !ok {
				continue
			}
			if err := checkFieldDataRefs(s.ID, "with."+key, str, needs); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkFieldDataRefs scans value for steps.<id>.output references and
// verifies each referenced id is present in needs.
func checkFieldDataRefs(stepID, field, value string, needs map[string]bool) error {
	if value == "" {
		return nil
	}

	tokens, err := interp.Scan(value)
	if err != nil {
		return fmt.Errorf("graph: step %q field %q: %w", stepID, field, err)
	}

	for _, tok := range tokens {
		if tok.Kind != interp.TokenReference || tok.Ref.Namespace != interp.NamespaceSteps {
			continue
		}
		if !needs[tok.Ref.StepID] {
			return &UndeclaredDataReferenceError{StepID: stepID, ReferencedStepID: tok.Ref.StepID, Field: field}
		}
	}
	return nil
}

// validateKinds implements the portion of tasks.md 6.9 that is
// statically checkable now — see this file's package doc comment for
// the full boundary. Only Input is checked: it is the one field whose
// structural expectation (a Directory-typed value) is fixed by the
// schema itself, independent of any provider.
func validateKinds(steps []manifest.Step) error {
	for _, s := range steps {
		if s.Input == "" {
			continue
		}

		tokens, err := interp.Scan(s.Input)
		if err != nil {
			return fmt.Errorf("graph: step %q field \"input\": %w", s.ID, err)
		}

		// Only a "pure" single-reference form has an attributable
		// kind — literal text mixed with a reference isn't something
		// this stage can reason about, and interp.Render already
		// rejects concatenating a secret with literal text (tasks.md
		// 5.4) wherever Render is eventually called.
		if len(tokens) != 1 || tokens[0].Kind != interp.TokenReference {
			continue
		}

		kind, ok := tokens[0].Ref.StaticKind()
		if !ok {
			// steps.<id>.output: the kind is fixed by the resolved
			// provider, not knowable until Phase 7 (design.md D-I).
			// Deliberately not checked here.
			continue
		}
		if kind == interp.KindSecret {
			return &KindMismatchError{StepID: s.ID, Field: "input", Got: kind}
		}
	}
	return nil
}

// sortedKeys returns m's keys in sorted order, for deterministic
// iteration over Step.With — a map[string]any has no stable Go
// iteration order otherwise.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
