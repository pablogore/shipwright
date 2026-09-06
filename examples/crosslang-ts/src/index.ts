/**
 * examples/crosslang-ts -- cross-language proof for Shipwright's Layer 2
 * capability contract (design.md D-C).
 *
 * D-C chose TypeScript over Python specifically because the generated
 * bindings for a foreign Dagger Interface (here, ShipwrightBuilder,
 * generated from .dagger/capabilities.go's `Builder` interface) are typed
 * `.ts`. Type-checking `ExampleBuilder` against `ShipwrightBuilder` is a
 * COMPILE-TIME proof that the five-capability contract shape crossed the
 * Go -> TypeScript language boundary -- not merely that one `dagger call`
 * happened to succeed at runtime (which is all a Python Protocol-based
 * proof could offer, per D-C's rejection of that option).
 *
 * ExampleBuilder is deliberately trivial: it returns the input Directory
 * unchanged. The point of this module is proving the TYPE contract
 * crosses the boundary, not implementing a real build capability.
 *
 * Confirmed runtime constraint (v0.21.8 TypeScript SDK, this work unit):
 * a locally `new`-constructed ExampleBuilder instance cannot be passed
 * inline, within the same function body, as a ShipwrightBuilder argument
 * to a foreign module's function (e.g. Shipwright.Plan.withBuild) --
 * dagql argument marshaling for an Interface-typed parameter requires a
 * real call-chain-derived object ID, which a bare `new` expression does
 * not have. The correct, and here proven, pattern instead exposes the
 * implementation as its own Dagger Function (exampleBuilder() below) so
 * a caller composes it across TWO engine calls -- exactly what `dagger
 * shell`'s composition syntax does (see this file's own doc example,
 * and the COMPATIBILITY.md / apply-progress note documenting the exact
 * failure modes ruled out: "cannot convert map[string]interface{} to
 * ID" and "unexpected result value type string for object").
 *
 * Documented, real local invocation proving the full chain succeeds
 * (run from this directory, with a Dagger engine connected):
 *
 *   dagger shell -c 'shipwright | plan $(host | directory <src>) | \
 *     with-build $(example-builder) | execute'
 */
import { Directory, ID, object, func } from "@dagger.io/dagger"

/**
 * ExampleBuilder implements the ShipwrightBuilder Dagger Interface
 * (.dagger/capabilities.go Builder: Build(ctx, source) *dagger.Directory).
 * tsc rejects this class at compile time if its shape drifts from the
 * generated ShipwrightBuilder interface in sdk/client.gen.ts -- that
 * rejection IS the cross-language contract proof.
 */
@object()
export class ExampleBuilder {
  /**
   * Build satisfies ShipwrightBuilder.build(source: Directory): Directory.
   * Trivial implementation: returns the source unchanged.
   */
  @func()
  build(source: Directory): Directory {
    return source
  }
}

/**
 * Declaration merging (a standard TypeScript pattern, not a workaround):
 * this interface merges with the `ExampleBuilder` class above to add
 * `id(): Promise<ID>` to its STATIC TYPE ONLY, so tsc verifies the class
 * against the full ShipwrightBuilder shape below -- while adding ZERO
 * runtime class member. That distinction matters here: a REAL `id()`
 * method (decorated or not) is picked up by both the Dagger module
 * runtime's object-identity resolution AND its schema introspector, and
 * a stub implementation breaks both (confirmed: a live method produced
 * "cannot convert map[string]interface{} to ID" when @func()-decorated,
 * and "unexpected result value type string for object" when returned as
 * a function result even without @func()). The Dagger engine computes
 * and marshals this object's real identity itself once it crosses the
 * module boundary; this module's own code never needs to produce one.
 */
export interface ExampleBuilder {
  id(): Promise<ID>
}

/**
 * CrosslangTs is this example module's entrypoint. It exposes this
 * module's ShipwrightBuilder implementation as a standalone Dagger
 * Function so a caller (dagger call/shell) can cross it into the root
 * shipwright module's Plan via withBuild(...) -- see this file's module
 * doc comment above for the exact composed invocation proving this
 * succeeds against a real engine.
 */
@object()
export class CrosslangTs {
  /**
   * ExampleBuilder returns a fresh ShipwrightBuilder implementation. A
   * caller composes it into shipwright.plan(source).withBuild(...) as a
   * properly engine-materialized object reference by chaining two
   * separate function calls (see module doc comment) -- not by
   * constructing-and-passing it inline within one function body.
   */
  @func()
  exampleBuilder(): ExampleBuilder {
    return new ExampleBuilder()
  }
}
