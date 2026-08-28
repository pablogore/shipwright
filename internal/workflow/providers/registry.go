// Package providers implements stage 6 (provider resolution) of the
// manifest's fixed seven-stage validation pipeline (design.md D-H) and,
// via WithSchema-checked resolution, the general-with-value-kind-mismatch
// portion of stage 7 (design.md D-H's "type mismatch in with") — see
// registry.go's checkWithSchema and this package's doc comment on
// WithSchemaMismatchError for the exact boundary this closes from WU5
// (internal/workflow/interp, Reference.StaticKind) and WU6
// (internal/workflow/graph, build.go's package doc comment).
//
// D-I, binding: uses.provider/uses.module resolves ONLY to already
// compiled, self-registered providers. This is a deliberate security
// decision — reusing internal/plugins/loader.go's plugin.Open for
// manifest-declared providers was rejected specifically because it would
// let a manifest a pipeline author or PR contributor controls run
// arbitrary native code in-process (supply-chain / arbitrary-code-
// execution risk). There is no fetch, no download, no cache, no `.so`
// load, and no registry service anywhere in this package — see
// security_test.go for the static proof that "plugin" is unreachable from
// this package's import graph, and register_test.go's
// TestRegisterDefaults_UnregisteredModuleFailsClosed for the fail-closed
// behavior that boundary produces at resolution time.
package providers

import (
	"fmt"
	"sort"

	"github.com/pablogore/shipwright/internal/workflow/interp"
	"github.com/pablogore/shipwright/pkg/shipwright"
)

// Ref identifies one registered provider implementation for a capability
// (design.md D-I's type sketch, exactly). Module == "" means an in-repo
// provider — one registered directly by this binary's own
// provider-registration file (register.go). A non-empty Module names the
// external module path a manifest's uses.module field references; it is
// resolved through the identical Register*/Resolve* mechanism as an
// in-repo provider — there is no separate, more permissive path for
// "external" providers (workflow-execution spec, "External module:
// Providers Satisfy The Same Dagger Contract As In-Repo Providers").
//
// Ref is keyed by BOTH Name and Module (never Name alone): a manifest
// requesting uses.provider: go and a manifest requesting
// uses.module: go must never resolve to the same registered entry,
// because they are declaring different trust boundaries (in-repo vs.
// external) even if they happen to share a literal name.
type Ref struct {
	Name    string
	Module  string
	Version string
}

// WithSchema is a provider's declaration of its accepted `with` field
// names and their expected interp.Kind (design.md D-I). It is what makes
// stage 7 checkable and what keeps providers receiving typed Values,
// never an interpolated shell string (the security skill's `sh -c` rule,
// and design.md's Threat Matrix row 4, "Provider argument construction").
type WithSchema map[string]interp.Kind

// Values carries a step's resolved `with` values, keyed by field name,
// already interpolated into typed interp.Value carriers — never a raw Go
// string that might hide a secret (interp.Value's own structural
// guarantee that a secret cannot be represented as a plain string).
type Values map[string]interp.Value

// UnregisteredProviderError reports a uses.provider/uses.module reference
// naming no provider registered at build time for the requested
// capability (design.md D-I, tasks.md 7.3). This is the exact fail-closed
// boundary that makes plugin.Open unreachable from provider resolution:
// there is no fetch/download/cache/dynamic-load fallback here, only this
// error.
type UnregisteredProviderError struct {
	Capability string
	Ref        Ref
}

func (e *UnregisteredProviderError) Error() string {
	if e.Ref.Module != "" {
		return fmt.Sprintf(
			"providers: unregistered module %q for capability %q (version %q) — module: references resolve only to providers compiled into this binary and self-registered at build time",
			e.Ref.Module, e.Capability, e.Ref.Version,
		)
	}
	return fmt.Sprintf("providers: unregistered provider %q for capability %q (version %q)", e.Ref.Name, e.Capability, e.Ref.Version)
}

// UnsupportedVersionError reports a registered provider Name/Module that
// exists, but not at the requested Version (tasks.md 7.2). Deliberately a
// distinct type from UnregisteredProviderError: "you asked for a version
// we don't ship" is a different failure mode from "you asked for a
// provider that doesn't exist at all", and a caller/operator needs to be
// able to tell them apart with errors.As, not string matching.
type UnsupportedVersionError struct {
	Capability        string
	Name              string
	Module            string
	RequestedVersion  string
	SupportedVersions []string
}

func (e *UnsupportedVersionError) Error() string {
	name := e.Name
	if e.Module != "" {
		name = e.Module
	}
	return fmt.Sprintf(
		"providers: %q does not support capability %q version %q (supported: %v)",
		name, e.Capability, e.RequestedVersion, e.SupportedVersions,
	)
}

// WithSchemaMismatchError reports a `with` field value whose resolved
// Kind does not match the resolved provider's declared WithSchema entry
// for that field name (tasks.md 7.5, stage 7's forbidPlaintext and
// general with-value type mismatch).
//
// This is the check WU5's task 5.3 and WU6's task 6.9 both explicitly
// deferred here: WU5's interp.Reference.StaticKind() reports ok=false for
// steps.<id>.output because that Kind is fixed by whichever provider
// resolves the step, and WU6's graph package could not compare a
// with-field's kind against anything because no provider schema existed
// in that package. Both boundaries close HERE, because this is the first
// point in the pipeline where a provider's declared WithSchema actually
// exists to compare against.
type WithSchemaMismatchError struct {
	Capability string
	Field      string
	Want       interp.Kind
	Got        interp.Kind
}

func (e *WithSchemaMismatchError) Error() string {
	return fmt.Sprintf(
		"providers: capability %q with field %q has kind %s, provider requires %s",
		e.Capability, e.Field, e.Got, e.Want,
	)
}

// providerKey identifies one registered provider's Name+Module pair,
// independent of Version — see the per-key version map in table.
type providerKey struct{ Name, Module string }

// entry is one registered (schema, factory) pair for a single provider
// Name+Module+Version.
type entry[T any] struct {
	schema  WithSchema
	factory func(Values) T
}

// table is the generic per-capability resolution table shared by all
// five Register*/Resolve* pairs. Using a type parameter here is an
// internal engine implementation detail only — it does NOT touch
// pkg/shipwright's public capability interfaces, which remain the
// no-generics, Dagger-codegen-projectable surface design.md D-A requires.
// design.md D-I's sketch names five separate typed methods on Registry;
// this shared table is what implements each of those five without
// duplicating the same register/resolve/version/schema logic five times.
type table[T any] struct {
	byKey map[providerKey]map[string]entry[T]
}

func newTable[T any]() *table[T] {
	return &table[T]{byKey: make(map[providerKey]map[string]entry[T])}
}

func (t *table[T]) register(ref Ref, schema WithSchema, f func(Values) T) {
	key := providerKey{Name: ref.Name, Module: ref.Module}
	if t.byKey[key] == nil {
		t.byKey[key] = make(map[string]entry[T])
	}
	t.byKey[key][ref.Version] = entry[T]{schema: schema, factory: f}
}

func (t *table[T]) resolve(capability string, ref Ref, v Values) (T, error) {
	var zero T

	key := providerKey{Name: ref.Name, Module: ref.Module}
	versions, ok := t.byKey[key]
	if !ok {
		return zero, &UnregisteredProviderError{Capability: capability, Ref: ref}
	}

	e, ok := versions[ref.Version]
	if !ok {
		return zero, &UnsupportedVersionError{
			Capability:        capability,
			Name:              ref.Name,
			Module:            ref.Module,
			RequestedVersion:  ref.Version,
			SupportedVersions: sortedVersions(versions),
		}
	}

	if err := checkWithSchema(capability, e.schema, v); err != nil {
		return zero, err
	}

	return e.factory(v), nil
}

func sortedVersions[T any](versions map[string]entry[T]) []string {
	out := make([]string, 0, len(versions))
	for version := range versions {
		out = append(out, version)
	}
	sort.Strings(out)
	return out
}

// checkWithSchema rejects any with-field the schema declares whose
// provided value's Kind does not match. A schema field the manifest's
// with map never supplied is not an error here — an absent optional
// field is the concrete provider factory's concern, not this structural
// kind check, which only compares kinds it can actually observe (mirrors
// interp.Reference.StaticKind's and graph.validateKinds's own "report
// what is provable, do not guess" discipline).
func checkWithSchema(capability string, schema WithSchema, v Values) error {
	keys := make([]string, 0, len(schema))
	for key := range schema {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		want := schema[key]
		val, ok := v[key]
		if !ok {
			continue
		}
		if got := val.Kind(); got != want {
			return &WithSchemaMismatchError{Capability: capability, Field: key, Want: want, Got: got}
		}
	}

	return nil
}

// Registry is the typed, per-capability provider table (design.md D-I).
// Its zero value is not usable — construct with NewRegistry.
type Registry struct {
	builders    *table[shipwright.Builder]
	testers     *table[shipwright.Tester]
	artifactors *table[shipwright.Artifactor]
	deployers   *table[shipwright.Deployer]
	runners     *table[shipwright.Runner]
	inspectors  *table[shipwright.RuntimeInspector]
}

// NewRegistry constructs an empty Registry. Register the in-repo WU3
// capabilities into it via RegisterDefaults (register.go).
func NewRegistry() *Registry {
	return &Registry{
		builders:    newTable[shipwright.Builder](),
		testers:     newTable[shipwright.Tester](),
		artifactors: newTable[shipwright.Artifactor](),
		deployers:   newTable[shipwright.Deployer](),
		runners:     newTable[shipwright.Runner](),
		inspectors:  newTable[shipwright.RuntimeInspector](),
	}
}

// RegisterBuilder registers a Builder provider factory under ref,
// declaring its accepted `with` fields via schema.
func (r *Registry) RegisterBuilder(ref Ref, schema WithSchema, f func(Values) shipwright.Builder) {
	r.builders.register(ref, schema, f)
}

// ResolveBuilder resolves ref (and validates v against its declared
// WithSchema) to a concrete shipwright.Builder.
func (r *Registry) ResolveBuilder(ref Ref, v Values) (shipwright.Builder, error) {
	return r.builders.resolve("build", ref, v)
}

// RegisterTester registers a Tester provider factory under ref. Multiple
// independent Tester providers MAY register (design.md D-F's
// orthogonality win: GoUnitTester, GoLinter, GoVulnScanner each register
// separately for capability: test) — none is privileged over another.
func (r *Registry) RegisterTester(ref Ref, schema WithSchema, f func(Values) shipwright.Tester) {
	r.testers.register(ref, schema, f)
}

// ResolveTester resolves ref (and validates v against its declared
// WithSchema) to a concrete shipwright.Tester.
func (r *Registry) ResolveTester(ref Ref, v Values) (shipwright.Tester, error) {
	return r.testers.resolve("test", ref, v)
}

// RegisterArtifactor registers an Artifactor provider factory under ref.
func (r *Registry) RegisterArtifactor(ref Ref, schema WithSchema, f func(Values) shipwright.Artifactor) {
	r.artifactors.register(ref, schema, f)
}

// ResolveArtifactor resolves ref (and validates v against its declared
// WithSchema) to a concrete shipwright.Artifactor.
func (r *Registry) ResolveArtifactor(ref Ref, v Values) (shipwright.Artifactor, error) {
	return r.artifactors.resolve("artifact", ref, v)
}

// RegisterDeployer registers a Deployer provider factory under ref. No
// in-repo Deployer implementation exists yet (pkg/shipwright.DeployConfig
// is deliberately empty — concrete deploy adapters are deferred to a
// follow-up change, design.md D-D); this method exists so the Registry's
// public surface has the same five typed pairs for every capability,
// ready for a future provider to register into.
func (r *Registry) RegisterDeployer(ref Ref, schema WithSchema, f func(Values) shipwright.Deployer) {
	r.deployers.register(ref, schema, f)
}

// ResolveDeployer resolves ref (and validates v against its declared
// WithSchema) to a concrete shipwright.Deployer.
func (r *Registry) ResolveDeployer(ref Ref, v Values) (shipwright.Deployer, error) {
	return r.deployers.resolve("deploy", ref, v)
}

// RegisterRunner registers a Runner provider factory under ref. See
// RegisterDeployer's doc comment — no in-repo Runner implementation
// exists yet either (pkg/shipwright.RunConfig is likewise empty).
func (r *Registry) RegisterRunner(ref Ref, schema WithSchema, f func(Values) shipwright.Runner) {
	r.runners.register(ref, schema, f)
}

// ResolveRunner resolves ref (and validates v against its declared
// WithSchema) to a concrete shipwright.Runner.
func (r *Registry) ResolveRunner(ref Ref, v Values) (shipwright.Runner, error) {
	return r.runners.resolve("run", ref, v)
}

// RegisterRuntimeInspector registers a RuntimeInspector provider factory
// under ref (runtime-toolchain-upgrade, design.md D-4b). Sixth typed pair,
// same shape as the original five above.
func (r *Registry) RegisterRuntimeInspector(ref Ref, schema WithSchema, f func(Values) shipwright.RuntimeInspector) {
	r.inspectors.register(ref, schema, f)
}

// ResolveRuntimeInspector resolves ref (and validates v against its
// declared WithSchema) to a concrete shipwright.RuntimeInspector.
func (r *Registry) ResolveRuntimeInspector(ref Ref, v Values) (shipwright.RuntimeInspector, error) {
	return r.inspectors.resolve("runtime-inspect", ref, v)
}
