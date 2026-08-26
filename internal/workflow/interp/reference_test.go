// Test for tasks.md 5.3's resolved primitive. Full stage-7 forbidPlaintext
// enforcement (design.md D-H: "Secret reference in a non-secret-typed
// field") compares a reference's eventual Kind against a PROVIDER's
// declared field kind (WithSchema), which does not exist until
// internal/workflow/providers (Phase 7, tasks.md 7.5/7.6). This package
// cannot and does not implement that comparison — it has no provider
// schema to compare against.
//
// What IS determinable from the interpolation grammar alone, and is
// therefore implemented and tested here, is StaticKind: whether a
// reference's namespace structurally FIXES its Kind (variables.* is
// always KindString per the manifest schema's `Variables
// map[string]string`; secrets.* is always KindSecret by construction) or
// whether it is provider-dependent (steps.<id>.output — the kind is fixed
// by whichever capability produced it, unknown until stage 6 provider
// resolution). Phase 7's forbidPlaintext check calls StaticKind (falling
// back to the resolved provider's WithSchema kind when StaticKind reports
// ok=false) against each field's declared kind; this test proves
// StaticKind itself is correct, not the full policy it will feed.
package interp_test

import (
	"testing"

	"github.com/pablogore/shipwright/internal/workflow/interp"
)

func TestReference_StaticKind_SecretsAlwaysKindSecret(t *testing.T) {
	t.Parallel()

	ref := interp.Reference{Namespace: interp.NamespaceSecrets, Name: "registry-token"}

	kind, ok := ref.StaticKind()
	if !ok {
		t.Fatal("StaticKind() ok = false for a secrets.* reference, want true — secrets.* is unconditionally KindSecret by construction")
	}
	if kind != interp.KindSecret {
		t.Fatalf("StaticKind() = %v, want KindSecret", kind)
	}
}

func TestReference_StaticKind_VariablesAlwaysKindString(t *testing.T) {
	t.Parallel()

	ref := interp.Reference{Namespace: interp.NamespaceVariables, Name: "env"}

	kind, ok := ref.StaticKind()
	if !ok {
		t.Fatal("StaticKind() ok = false for a variables.* reference, want true — the manifest schema types spec.variables as map[string]string")
	}
	if kind != interp.KindString {
		t.Fatalf("StaticKind() = %v, want KindString", kind)
	}
}

// TestReference_StaticKind_StepsOutputIsUndetermined proves the honest
// boundary: a steps.<id>.output reference's Kind is fixed by the
// capability that resolves to at stage 6 (internal/workflow/providers,
// Phase 7), not by the interpolation grammar. StaticKind must report
// ok=false rather than guessing, so a later stage cannot silently skip
// the provider-resolution step it actually depends on.
func TestReference_StaticKind_StepsOutputIsUndetermined(t *testing.T) {
	t.Parallel()

	ref := interp.Reference{Namespace: interp.NamespaceSteps, StepID: "build"}

	if kind, ok := ref.StaticKind(); ok {
		t.Fatalf("StaticKind() ok = true (kind %v) for a steps.<id>.output reference, want ok=false — its Kind depends on provider resolution (Phase 7), not the grammar alone", kind)
	}
}
