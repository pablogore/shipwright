// Package engine implements the wave-scheduling execution engine for
// Shipwright's declarative workflow layer (design.md D-K, workflow-execution
// spec). It is the FIRST place in the fixed seven-stage validation pipeline
// (design.md D-H) where a manifest's compiled graph.Graph (stage 5), its
// resolved providers.Registry (stage 6), and interpolated with/input values
// (stage 7) are all in scope simultaneously — see WU7's apply-progress
// ("Closing the WU5/WU6 loop") for the exact boundary this package closes:
// a steps.<id>.output reference's Kind is not knowable until the producing
// step has actually run, and a with-field's Kind is not checkable against a
// provider's declared WithSchema until that provider is resolved. Both
// happen here, per step, as each wave executes.
//
// Execute invokes the Layer 1 capability interfaces (pkg/shipwright.
// Builder/Tester/Artifactor/Deployer/Runner) DIRECTLY — never through Plan
// (the .dagger/ Layer 2 composition type). This is a deliberate
// architectural decision (design.md D-G): Plan's five-slot chain cannot
// express needs[]/DAG dependencies, so the workflow engine bypasses it
// entirely.
//
// Scheduling (design.md D-K): Kahn's waves (internal/workflow/graph) are
// executed in order; within a single wave, steps run SEQUENTIALLY in
// manifest-declaration order — never map iteration order, never
// concurrently. Options.MaxParallel is validated/recorded (see
// OptionsFromSpec) but is NOT used to widen execution in this work unit;
// sequential execution is a correct schedule for any MaxParallel >= 1.
// Concurrent widening within a wave is explicitly deferred (design.md D-K).
//
// Approval gates (design.md D-M): spec.environments.<name>.approvals is
// parsed metadata only. This package contains NO blocking, queueing, or
// "wait for approval" logic anywhere — a manifest declaring approvals
// executes exactly as if it had none.
package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
	"github.com/pablogore/shipwright/internal/workflow/providers"
)

// Options configures the engine-level execution controls design.md D-K
// implements now: FailFast, per-step Timeout, and bounded per-step Retries.
//
// Retries is the TOTAL number of attempts made per step (not the count of
// additional retries after the first) — a value less than 1 is treated as
// 1, i.e. no retry. The manifest schema (internal/workflow/manifest) has no
// per-step or workflow-level "retries" field at all; Retries is therefore
// engine/caller-supplied only in this work unit. This is a genuine schema
// gap, not an oversight of this package — flagged for sdd-verify rather
// than reaching back into WU4's manifest package from here.
//
// Timeout is applied per step via context.WithTimeout, using the single
// workflow-level spec.execution.timeout value as each step's budget (the
// manifest schema has no PER-STEP timeout field either — see the same gap
// note above). A zero Timeout means no deadline is applied.
//
// MaxParallel is recorded from spec.execution.concurrency.maxParallel (see
// OptionsFromSpec) but is NEVER used to widen execution — see this
// package's doc comment.
type Options struct {
	FailFast    bool
	Timeout     time.Duration
	Retries     int
	MaxParallel int
}

// OptionsFromSpec derives Options from a manifest's ExecutionSpec. This is
// the concrete place spec.execution.concurrency.maxParallel is "validated
// and recorded" (tasks.md 8.5, design.md D-K): it is copied into
// Options.MaxParallel verbatim and never used to widen execution anywhere
// in this package. It performs NO additional validation of MaxParallel
// (for example rejecting <= 0) — design.md D-K states that check belongs to
// manifest stage 3 (internal/workflow/manifest.ValidateStructure). At the
// time this package was written, ValidateStructure does NOT enforce
// maxParallel <= 0 — this is a confirmed gap, flagged for sdd-verify rather
// than fixed here (fixing it would mean reaching back into WU4's package,
// which is out of this work unit's scope per its own launch instructions).
func OptionsFromSpec(spec manifest.ExecutionSpec) (Options, error) {
	opts := Options{
		FailFast:    spec.FailFast,
		MaxParallel: spec.Concurrency.MaxParallel,
	}

	if spec.Timeout != "" {
		d, err := time.ParseDuration(spec.Timeout)
		if err != nil {
			return Options{}, fmt.Errorf("engine: invalid spec.execution.timeout %q: %w", spec.Timeout, err)
		}
		opts.Timeout = d
	}

	return opts, nil
}

// Config bundles everything Execute needs beyond the compiled Graph and
// resolved Registry to run a manifest end-to-end. Steps carries the
// manifest's own step declarations (capability, uses, with, input, when) —
// deliberately separate from Graph, whose Node type carries only id+needs
// (internal/workflow/graph's own package doc comment: "everything else
// about the step belongs to a later phase's concern" — this IS that later
// phase).
type Config struct {
	// Steps is the manifest's ordered step list (declaration order is
	// significant — see this package's doc comment on wave ordering).
	Steps []manifest.Step
	// Graph is the compiled DAG (internal/workflow/graph.Build's output).
	Graph *graph.Graph
	// Registry resolves each step's uses.provider/uses.module to a
	// concrete Layer 1 capability instance (internal/workflow/providers).
	Registry *providers.Registry
	// Source is the workflow's default Directory input (spec.source),
	// bound to a step's Directory-typed input whenever that step
	// declares no explicit "input" field (design.md D-H schema
	// simplification note).
	Source *dagger.Directory
	// Variables resolves "${{ variables.<name> }}" references
	// (spec.variables).
	Variables map[string]string
	// Secrets resolves "${{ secrets.<name> }}" references
	// (spec.secrets), already bound to their *dagger.Secret handles —
	// never a plaintext string (design.md D-L).
	Secrets map[string]*dagger.Secret
	// Predicates is the actual runtime value for each "when" key this
	// run evaluates against (for example {"branch": "main"}). A step
	// declaring "when: {branch: [main]}" executes only when
	// Predicates["branch"] is present in that list (design.md D-L,
	// exact-match evaluation, never an expression).
	Predicates map[string]string
	// Options configures failure strategy, per-step timeout, and
	// per-step retry.
	Options Options
}

// StepStatus is the terminal state of one step in a Result.
type StepStatus int

const (
	// StatusSucceeded means the step's capability method returned no
	// error within its configured attempt budget.
	StatusSucceeded StepStatus = iota
	// StatusFailed means every configured attempt returned an error.
	StatusFailed
	// StatusSkipped means the step's "when" predicate did not match, or
	// fail-fast stopped scheduling before this step's wave ran.
	StatusSkipped
)

// String renders a StepStatus's name for diagnostics.
func (s StepStatus) String() string {
	switch s {
	case StatusSucceeded:
		return "succeeded"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// StepOutcome records one step's terminal result, in the order it was
// decided (wave order, manifest-declaration order within a wave).
type StepOutcome struct {
	StepID   string
	Status   StepStatus
	Attempts int
	Err      error
}

// Result is Execute's complete report. Outcomes is ordered by execution
// (wave order, then manifest-declaration order within a wave) — the same
// determinism Kahn's algorithm already guarantees for Graph.Waves
// (internal/workflow/graph, design.md D-K).
type Result struct {
	Outcomes []StepOutcome
	// Failures is the ordered list of step ids whose every attempt
	// failed. Empty means the run succeeded end-to-end.
	Failures []string
}

// Failed reports whether any step failed.
func (r *Result) Failed() bool {
	return len(r.Failures) > 0
}

// StepFailedError is Execute's returned error whenever at least one step
// fails — it names the first failing step id (tasks.md 8.2), wrapping that
// step's last-attempt error.
type StepFailedError struct {
	StepID string
	Err    error
}

func (e *StepFailedError) Error() string {
	return fmt.Sprintf("engine: step %q failed: %v", e.StepID, e.Err)
}

func (e *StepFailedError) Unwrap() error { return e.Err }

// StepTimeoutError reports a step whose execution exceeded Options.Timeout
// (tasks.md 8.3) — context.WithTimeout fired and the step was canceled,
// never left hanging.
type StepTimeoutError struct {
	StepID  string
	Timeout time.Duration
}

func (e *StepTimeoutError) Error() string {
	return fmt.Sprintf("engine: step %q exceeded its %s timeout", e.StepID, e.Timeout)
}

// UnknownCapabilityError reports a step whose Capability is not one of the
// five contract types. Defensive only: manifest.ValidateStructure (stage 3)
// already rejects this before Execute would ever see it when the normal
// parse→graph→engine pipeline is followed — this package does not assume
// that happened, mirroring internal/workflow/graph's own "does not assume
// stage 3 already ran" discipline.
type UnknownCapabilityError struct{ StepID, Capability string }

func (e *UnknownCapabilityError) Error() string {
	return fmt.Sprintf("engine: step %q has unknown capability %q", e.StepID, e.Capability)
}

// MissingStepOutputError reports a "${{ steps.<id>.output }}" reference to
// a step whose output is not available — because it has not executed yet
// (should not happen given Graph.Waves ordering), it failed, or it was
// skipped (fail-fast or a non-matching "when").
type MissingStepOutputError struct{ StepID, ReferencedStepID string }

func (e *MissingStepOutputError) Error() string {
	return fmt.Sprintf("engine: step %q references steps.%s.output, but %q has no available output", e.StepID, e.ReferencedStepID, e.ReferencedStepID)
}

// OutputKindMismatchError reports a "${{ steps.<id>.output }}" reference
// whose resolved output kind cannot satisfy the field it is used in — the
// steps.<id>.output kind check design.md's Migration Sequence table and
// WU7's apply-progress ("Closing the WU5/WU6 loop") both name as this
// package's job: a step's output kind is fixed by whichever capability
// produced it, which is only known once that step has actually run.
type OutputKindMismatchError struct {
	StepID, ReferencedStepID, Field, Want string
}

func (e *OutputKindMismatchError) Error() string {
	return fmt.Sprintf(
		"engine: step %q field %q references steps.%s.output, which does not produce a %s value",
		e.StepID, e.Field, e.ReferencedStepID, e.Want,
	)
}

// InvalidInputReferenceError reports a step's "input" field containing
// anything other than exactly one "${{ steps.<id>.output }}" reference —
// input is a Directory-typed field (design.md D-H); a literal string or a
// variables./secrets. reference can never satisfy that.
type InvalidInputReferenceError struct{ StepID string }

func (e *InvalidInputReferenceError) Error() string {
	return fmt.Sprintf("engine: step %q field \"input\" must be exactly one steps.<id>.output reference", e.StepID)
}

// UndeclaredVariableError reports a "${{ variables.<name> }}" reference to
// a name absent from spec.variables.
type UndeclaredVariableError struct{ StepID, Name string }

func (e *UndeclaredVariableError) Error() string {
	return fmt.Sprintf("engine: step %q references undeclared variable %q", e.StepID, e.Name)
}

// UndeclaredSecretError reports a "${{ secrets.<name> }}" reference to a
// name absent from spec.secrets.
type UndeclaredSecretError struct{ StepID, Name string }

func (e *UndeclaredSecretError) Error() string {
	return fmt.Sprintf("engine: step %q references undeclared secret %q", e.StepID, e.Name)
}

// MissingWithFieldError reports a required capability-specific "with"
// field (for example Artifactor's "ref") absent from a step's resolved
// with-values at dispatch time.
type MissingWithFieldError struct{ StepID, Field string }

func (e *MissingWithFieldError) Error() string {
	return fmt.Sprintf("engine: step %q is missing required with field %q", e.StepID, e.Field)
}

// outputKind discriminates which field of result actually carries a
// step's produced value — never inferred from nil-ness, since a nil
// *dagger.Directory/*dagger.File/*dagger.Container is itself a legitimate
// (if unusual) value a fake or real capability may return.
type outputKind int

const (
	outputNone outputKind = iota
	outputDirectory
	outputFile
	outputText
	outputContainer
)

// result is one step's typed output, keyed by step id in Execute's outputs
// map. Which kind is populated mirrors the capability that produced it:
// Builder -> directory, Tester -> file, Artifactor/Deployer -> text,
// Runner -> container.
type result struct {
	kind      outputKind
	directory *dagger.Directory
	file      *dagger.File
	text      string
	container *dagger.Container
}

// Execute runs cfg's manifest steps over cfg.Graph's Kahn waves
// (internal/workflow/graph, design.md D-J), invoking Layer 1 capability
// interfaces directly through cfg.Registry (never through Plan). See this
// package's doc comment for the full scheduling, approval, and
// capability-dispatch contract.
func Execute(ctx context.Context, cfg Config) (*Result, error) {
	stepByID := make(map[string]manifest.Step, len(cfg.Steps))
	for _, s := range cfg.Steps {
		stepByID[s.ID] = s
	}

	outputs := make(map[string]result, len(cfg.Steps))
	res := &Result{}

waveLoop:
	for _, wave := range cfg.Graph.Waves {
		for _, id := range wave {
			s, ok := stepByID[id]
			if !ok {
				return res, fmt.Errorf("engine: graph references step %q not present in cfg.Steps", id)
			}

			if !matchesWhen(s.When, cfg.Predicates) {
				res.Outcomes = append(res.Outcomes, StepOutcome{StepID: id, Status: StatusSkipped})
				continue
			}

			out, attempts, err := executeStepWithRetry(ctx, s, outputs, cfg)
			if err != nil {
				res.Outcomes = append(res.Outcomes, StepOutcome{StepID: id, Status: StatusFailed, Attempts: attempts, Err: err})
				res.Failures = append(res.Failures, id)
				if cfg.Options.FailFast {
					break waveLoop
				}
				continue
			}

			outputs[id] = out
			res.Outcomes = append(res.Outcomes, StepOutcome{StepID: id, Status: StatusSucceeded, Attempts: attempts})
		}
	}

	if len(res.Failures) > 0 {
		first := res.Failures[0]
		return res, &StepFailedError{StepID: first, Err: firstFailureErr(res, first)}
	}
	return res, nil
}

// firstFailureErr finds the recorded error for the first failing step id,
// so StepFailedError can wrap it without Execute's main loop carrying an
// extra variable through the fail-fast break.
func firstFailureErr(res *Result, stepID string) error {
	for _, o := range res.Outcomes {
		if o.StepID == stepID && o.Status == StatusFailed {
			return o.Err
		}
	}
	return nil
}

// matchesWhen evaluates a step's structured "when" predicate map against
// the run's actual predicate values (design.md D-L: exact match, never an
// expression). An empty/nil "when" always matches — a step with no
// condition always executes. A key present in "when" but absent from
// predicates never matches (there is no value to compare).
func matchesWhen(when map[string][]string, predicates map[string]string) bool {
	for key, allowed := range when {
		actual, ok := predicates[key]
		if !ok || !containsString(allowed, actual) {
			return false
		}
	}
	return true
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// maxAttempts normalizes Options.Retries: a value less than 1 means "try
// once, no retry" (see Options' doc comment).
func maxAttempts(retries int) int {
	if retries < 1 {
		return 1
	}
	return retries
}

// executeStepWithRetry attempts s up to its configured attempt budget
// (tasks.md 8.4), returning the first successful result or the last
// attempt's error once the budget is exhausted. A step-level Attempts
// field overrides the workflow-level Options.Retries when set. A nil
// value falls back to Options.Retries.
func executeStepWithRetry(ctx context.Context, s manifest.Step, outputs map[string]result, cfg Config) (result, int, error) {
	retries := cfg.Options.Retries
	if s.Attempts != nil {
		retries = *s.Attempts
	}
	attempts := maxAttempts(retries)

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		out, err := executeStepOnce(ctx, s, outputs, cfg)
		if err == nil {
			return out, attempt, nil
		}
		lastErr = err
	}
	return result{}, attempts, lastErr
}

// executeStepOnce runs one attempt of s: it resolves its Directory-typed
// input, binds its "with" values (interpolating and validating them
// against the resolved provider's WithSchema), applies Options.Timeout via
// context.WithTimeout (tasks.md 8.3), and dispatches to the capability
// method the step's declared capability names.
func executeStepOnce(ctx context.Context, s manifest.Step, outputs map[string]result, cfg Config) (result, error) {
	stepCtx := ctx
	if cfg.Options.Timeout > 0 {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, cfg.Options.Timeout)
		defer cancel()
	}

	input, err := resolveInput(stepCtx, s, outputs, cfg.Source)
	if err != nil {
		return result{}, err
	}

	values, err := resolveWith(s, outputs, cfg)
	if err != nil {
		return result{}, err
	}

	out, err := dispatch(stepCtx, s, input, values, cfg.Registry)
	if err != nil {
		if stepCtx.Err() == context.DeadlineExceeded {
			return result{}, &StepTimeoutError{StepID: s.ID, Timeout: cfg.Options.Timeout}
		}
		return result{}, err
	}
	return out, nil
}

// resolveInput resolves a step's Directory-typed input: the workflow's
// default Source when Input is unset (design.md D-H schema simplification
// note — "A step's Directory-typed input is bound by steps[].input,
// default spec.source"), or the Directory produced by exactly one
// "${{ steps.<id>.output }}" reference — either directly (a Builder's
// outputDirectory) or exported from a Runner's outputContainer via
// containerOutputDirectory.
func resolveInput(ctx context.Context, s manifest.Step, outputs map[string]result, source *dagger.Directory) (*dagger.Directory, error) {
	if s.Input == "" {
		return source, nil
	}

	tokens, err := interp.Scan(s.Input)
	if err != nil {
		return nil, fmt.Errorf("engine: step %q field \"input\": %w", s.ID, err)
	}
	if len(tokens) != 1 || tokens[0].Kind != interp.TokenReference || tokens[0].Ref.Namespace != interp.NamespaceSteps {
		return nil, &InvalidInputReferenceError{StepID: s.ID}
	}

	refStepID := tokens[0].Ref.StepID
	out, ok := outputs[refStepID]
	if !ok {
		return nil, &MissingStepOutputError{StepID: s.ID, ReferencedStepID: refStepID}
	}

	switch out.kind {
	case outputDirectory:
		return out.directory, nil
	case outputContainer:
		dir, err := containerOutputDirectory(ctx, out.container)
		if err != nil {
			return nil, fmt.Errorf("engine: step %q field \"input\": %w", s.ID, err)
		}
		return dir, nil
	default:
		return nil, &OutputKindMismatchError{StepID: s.ID, ReferencedStepID: refStepID, Field: "input", Want: "directory"}
	}
}

// containerOutputDirectory exports a Runner's produced Container as a
// Directory, so a downstream step's Input can consume it the same way it
// consumes a Builder's Directory output — otherwise a Runner's output (for
// example ChangelogRunner's updated CHANGELOG.md) has no path back into the
// workflow and is silently discarded.
//
// The exported subtree is rooted at the container's own working directory
// (dagger.Container.Workdir), not "/": a Runner's container commonly still
// carries its full base-image filesystem (see providers/changelog.go's
// alpine base), so exporting "/" would hand a downstream Directory-typed
// input the entire OS layer instead of just the workspace the Runner
// actually mutated. A Runner that wants its container consumed downstream
// this way is expected to leave WithWorkdir pointing at that workspace
// before returning — the same convention a container image's own WORKDIR
// already carries.
func containerOutputDirectory(ctx context.Context, container *dagger.Container) (*dagger.Directory, error) {
	if container == nil {
		return nil, errors.New("engine: runner produced a nil container, cannot export its directory")
	}
	workdir, err := container.Workdir(ctx)
	if err != nil {
		return nil, fmt.Errorf("engine: failed to resolve container workdir: %w", err)
	}
	if workdir == "" {
		workdir = "/"
	}
	return container.Directory(workdir), nil
}

// resolveWith interpolates and binds every entry in s.With into a
// providers.Values map (stage 7, "value binding"). This is the exact
// primitive that makes providers.Registry's WithSchema-checked Resolve*
// calls (WU7, registry.go's checkWithSchema) actually see a step's real
// interpolated values, rather than raw manifest strings.
func resolveWith(s manifest.Step, outputs map[string]result, cfg Config) (providers.Values, error) {
	if len(s.With) == 0 {
		return providers.Values{}, nil
	}

	resolve := makeResolver(s.ID, outputs, cfg)

	keys := make([]string, 0, len(s.With))
	for k := range s.With {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	values := make(providers.Values, len(s.With))
	for _, key := range keys {
		v, err := bindWithValue(s.ID, key, s.With[key], resolve)
		if err != nil {
			return nil, err
		}
		values[key] = v
	}
	return values, nil
}

// bindWithValue converts one raw "with" entry (as decoded by yaml.v3 into
// an "any") into a typed interp.Value. A string entry is scanned and
// rendered through the closed interpolation grammar (design.md D-L); a
// non-string scalar (bool/int/float64 — yaml.v3's own decode shapes for an
// untyped YAML scalar) is converted directly, since there is no
// interpolation grammar to apply to a value that was never a string.
func bindWithValue(stepID, field string, raw any, resolve func(interp.Reference) (interp.Value, error)) (interp.Value, error) {
	switch t := raw.(type) {
	case string:
		tokens, err := interp.Scan(t)
		if err != nil {
			return interp.Value{}, fmt.Errorf("engine: step %q field %q: %w", stepID, field, err)
		}
		return interp.Render(tokens, resolve)
	case bool:
		return interp.NewBool(t), nil
	case int:
		return interp.NewInt(int64(t)), nil
	case int64:
		return interp.NewInt(t), nil
	case float64:
		return interp.NewInt(int64(t)), nil
	default:
		return interp.Value{}, fmt.Errorf("engine: step %q field %q: unsupported with-value type %T", stepID, field, t)
	}
}

// makeResolver returns the interp.Render resolve function for stepID:
// variables.* and secrets.* resolve against cfg's declared maps (failing
// closed on an undeclared name — engine.go's doc comment on the schema gap
// this closes), and steps.<id>.output resolves against outputs, failing
// closed when the referenced step has no available output or its output
// kind cannot be represented as a string (only Artifactor/Deployer's
// string result can — a Directory/File/Container output has no string
// form, so referencing one from a "with" field is a kind mismatch, not a
// missing case).
func makeResolver(stepID string, outputs map[string]result, cfg Config) func(interp.Reference) (interp.Value, error) {
	return func(ref interp.Reference) (interp.Value, error) {
		switch ref.Namespace {
		case interp.NamespaceVariables:
			v, ok := cfg.Variables[ref.Name]
			if !ok {
				return interp.Value{}, &UndeclaredVariableError{StepID: stepID, Name: ref.Name}
			}
			return interp.NewString(v), nil

		case interp.NamespaceSecrets:
			s, ok := cfg.Secrets[ref.Name]
			if !ok {
				return interp.Value{}, &UndeclaredSecretError{StepID: stepID, Name: ref.Name}
			}
			return interp.NewSecret(s), nil

		case interp.NamespaceSteps:
			out, ok := outputs[ref.StepID]
			if !ok {
				return interp.Value{}, &MissingStepOutputError{StepID: stepID, ReferencedStepID: ref.StepID}
			}
			if out.kind != outputText {
				return interp.Value{}, &OutputKindMismatchError{StepID: stepID, ReferencedStepID: ref.StepID, Field: "with", Want: "string"}
			}
			return interp.NewString(out.text), nil

		default:
			return interp.Value{}, fmt.Errorf("engine: step %q: unknown reference namespace", stepID)
		}
	}
}

// dispatch invokes the Layer 1 capability method s's declared capability
// names, directly through cfg's resolved Registry — never through Plan
// (design.md D-G). Each capability has a distinct method signature
// (pkg/shipwright.{Builder,Tester,Artifactor,Deployer,Runner}), so this is
// the one place in the engine that must know all five shapes; every other
// helper in this package is capability-agnostic.
func dispatch(ctx context.Context, s manifest.Step, input *dagger.Directory, values providers.Values, reg *providers.Registry) (result, error) {
	ref := providers.Ref{Name: s.Uses.Provider, Module: s.Uses.Module, Version: s.Uses.Version}

	switch s.Capability {
	case "build":
		return dispatchBuild(ctx, ref, input, values, reg)
	case "test":
		return dispatchTest(ctx, ref, input, values, reg)
	case "artifact":
		return dispatchArtifact(ctx, s.ID, ref, input, values, reg)
	case "deploy":
		return dispatchDeploy(ctx, s.ID, ref, values, reg)
	case "run":
		return dispatchRun(ctx, ref, input, values, reg)
	default:
		return result{}, &UnknownCapabilityError{StepID: s.ID, Capability: s.Capability}
	}
}

func dispatchBuild(ctx context.Context, ref providers.Ref, input *dagger.Directory, values providers.Values, reg *providers.Registry) (result, error) {
	b, err := reg.ResolveBuilder(ref, values)
	if err != nil {
		return result{}, err
	}
	dir, err := b.Build(ctx, input)
	if err != nil {
		return result{}, err
	}
	return result{kind: outputDirectory, directory: dir}, nil
}

func dispatchTest(ctx context.Context, ref providers.Ref, input *dagger.Directory, values providers.Values, reg *providers.Registry) (result, error) {
	t, err := reg.ResolveTester(ref, values)
	if err != nil {
		return result{}, err
	}
	f, err := t.Test(ctx, input)
	if err != nil {
		return result{}, err
	}
	return result{kind: outputFile, file: f}, nil
}

// artifactRefField/artifactCredsField are the with-field names an
// Artifactor step's ref/creds arguments are bound from — matching
// register.go's RegisterDefaults ContainerPublisher WithSchema exactly
// ("ref": KindString, "creds": KindSecret).
const (
	artifactRefField   = "ref"
	artifactCredsField = "creds"
)

func dispatchArtifact(ctx context.Context, stepID string, ref providers.Ref, input *dagger.Directory, values providers.Values, reg *providers.Registry) (result, error) {
	a, err := reg.ResolveArtifactor(ref, values)
	if err != nil {
		return result{}, err
	}
	refVal, err := stringWith(stepID, values, artifactRefField)
	if err != nil {
		return result{}, err
	}
	creds := secretWith(values, artifactCredsField)

	out, err := a.Publish(ctx, input, refVal, creds)
	if err != nil {
		return result{}, err
	}
	return result{kind: outputText, text: out}, nil
}

// deployArtifactRefField/deployEnvironmentField/deployCredsField are the
// with-field naming convention this package establishes for Deployer steps
// (no in-repo Deployer provider is registered yet — pkg/shipwright.
// DeployConfig is still an empty stub, WU3/WU7 — so there is no existing
// manifest to confirm this naming against; it mirrors Deployer.Deploy's own
// three arguments as closely as Artifactor's convention mirrors Publish's).
const (
	deployArtifactRefField = "artifactRef"
	deployEnvironmentField = "environment"
	deployCredsField       = "creds"
)

func dispatchDeploy(ctx context.Context, stepID string, ref providers.Ref, values providers.Values, reg *providers.Registry) (result, error) {
	d, err := reg.ResolveDeployer(ref, values)
	if err != nil {
		return result{}, err
	}
	artifactRef, err := stringWith(stepID, values, deployArtifactRefField)
	if err != nil {
		return result{}, err
	}
	environment, err := stringWith(stepID, values, deployEnvironmentField)
	if err != nil {
		return result{}, err
	}
	creds := secretWith(values, deployCredsField)

	out, err := d.Deploy(ctx, artifactRef, environment, creds)
	if err != nil {
		return result{}, err
	}
	return result{kind: outputText, text: out}, nil
}

func dispatchRun(ctx context.Context, ref providers.Ref, input *dagger.Directory, values providers.Values, reg *providers.Registry) (result, error) {
	r, err := reg.ResolveRunner(ref, values)
	if err != nil {
		return result{}, err
	}
	c, err := r.Run(ctx, input)
	if err != nil {
		return result{}, err
	}
	return result{kind: outputContainer, container: c}, nil
}

func stringWith(stepID string, values providers.Values, field string) (string, error) {
	v, ok := values[field]
	if !ok {
		return "", &MissingWithFieldError{StepID: stepID, Field: field}
	}
	s, ok := v.String()
	if !ok {
		return "", &MissingWithFieldError{StepID: stepID, Field: field}
	}
	return s, nil
}

func secretWith(values providers.Values, field string) *dagger.Secret {
	v, ok := values[field]
	if !ok {
		return nil
	}
	s, _ := v.Secret()
	return s
}
