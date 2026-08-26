// SECURITY tests — tasks.md 5.2 (compile/runtime-verifiable proof that no
// exported Value accessor can return a secret as a string) and 5.4
// (secret+literal concatenation rejected), both design.md D-L: "secrets
// never become strings."
package interp_test

import (
	"reflect"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/pablogore/shipwright/internal/workflow/interp"
)

// TestValue_NoExportedAccessorReturnsSecretAsString is tasks.md 5.2's
// SECURITY RED test, made GREEN by construction: it reflects over EVERY
// exported, zero-argument method on a KindSecret-kind interp.Value and
// calls each one, asserting that no returned string is non-empty. This is
// stronger than reviewing the current three accessors by hand — it also
// fails closed against a FUTURE accessor added to Value without updating
// this test: any new zero-arg exported method that returns a bare string
// is caught here the moment it returns something non-empty for a secret.
//
// Design rationale (design.md D-L): Value.str is documented as "never set
// when kind == KindSecret" — NewSecret never populates it — so this test
// is a genuine structural proof, not a policy check bolted on afterward.
// String() additionally guards explicitly (returns ok=false for
// KindSecret) as defense in depth, in case a future refactor ever DID
// populate str alongside a secret by mistake.
func TestValue_NoExportedAccessorReturnsSecretAsString(t *testing.T) {
	t.Parallel()

	secretHandle := &dagger.Secret{}
	v := interp.NewSecret(secretHandle)

	rv := reflect.ValueOf(v)
	numMethods := rv.NumMethod()
	if numMethods == 0 {
		t.Fatal("interp.Value has no exported methods — this sweep is not exercising anything, investigate before trusting it")
	}

	for i := 0; i < numMethods; i++ {
		method := rv.Method(i)
		name := rv.Type().Method(i).Name

		if method.Type().NumIn() != 0 {
			t.Fatalf("Value.%s takes arguments — this security sweep only knows how to safely invoke zero-argument accessors; extend it deliberately before adding a parameterized method to Value", name)
		}

		results := method.Call(nil)
		for j, res := range results {
			if res.Kind() != reflect.String {
				continue
			}
			if res.String() != "" {
				t.Fatalf("Value.%s() return #%d on a KindSecret value = %q, want empty — no exported accessor on Value may return a secret's payload as a string (design.md D-L)", name, j, res.String())
			}
		}
	}
}

// TestValue_KindSecretNeverHasStringSet is a narrower, non-reflection
// companion: it proves the specific claim design.md D-L makes about the
// struct shape itself — a KindSecret Value's String() always reports
// ok=false, which is only possible because str is never populated
// alongside a secret in the first place.
func TestValue_KindSecretNeverHasStringSet(t *testing.T) {
	t.Parallel()

	v := interp.NewSecret(&dagger.Secret{})

	if s, ok := v.String(); ok || s != "" {
		t.Fatalf("String() = (%q, %v) for a KindSecret value, want (\"\", false)", s, ok)
	}
}

// TestRender_SecretPlusLiteralConcatenationRejected is tasks.md 5.4's
// SECURITY RED test: a field mixing literal text with a secret reference
// (`"Bearer ${{ secrets.tok }}"`) MUST be rejected, because producing the
// single concatenated string requires calling a string accessor on the
// resolved secrets.tok Value — and Value's String() accessor structurally
// cannot produce one for KindSecret (proven above). Render must surface
// this as an error, never a partial/best-effort string containing the
// literal portion only.
func TestRender_SecretPlusLiteralConcatenationRejected(t *testing.T) {
	t.Parallel()

	tokens, err := interp.Scan(`Bearer ${{ secrets.tok }}`)
	if err != nil {
		t.Fatalf("Scan() error = %v, want nil — the grammar itself accepts this input; concatenation is rejected at Render, not Scan", err)
	}

	resolve := func(ref interp.Reference) (interp.Value, error) {
		if ref.Namespace == interp.NamespaceSecrets && ref.Name == "tok" {
			return interp.NewSecret(&dagger.Secret{}), nil
		}
		t.Fatalf("unexpected reference resolved: %+v", ref)
		return interp.Value{}, nil
	}

	_, err = interp.Render(tokens, resolve)
	if err == nil {
		t.Fatal("Render() with a secret reference mixed into literal text must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "secret") {
		t.Fatalf("Render() error = %v, want it to name the secret concatenation rejection", err)
	}
}

// TestRender_ConcatenationErrorNamesTheOffendingReference proves the
// rejection error identifies WHICH reference forced the rejection,
// covering describeRef's rendering for a steps.<id>.output reference (the
// secrets.* case is already covered by
// TestRender_SecretPlusLiteralConcatenationRejected above) — useful for a
// manifest author debugging why a field was rejected.
func TestRender_ConcatenationErrorNamesTheOffendingReference(t *testing.T) {
	t.Parallel()

	tokens, err := interp.Scan(`token: ${{ steps.build.output }}`)
	if err != nil {
		t.Fatalf("Scan() error = %v, want nil", err)
	}

	// A steps.<id>.output reference resolving to a secret is unrealistic
	// in production (no capability outputs KindSecret today), but Render
	// only inspects the RESOLVED Value's kind, not the reference's
	// namespace — this fake resolver exercises that path deliberately, to
	// prove Render's rejection is driven by the Value, not a namespace
	// special-case.
	resolve := func(ref interp.Reference) (interp.Value, error) {
		return interp.NewSecret(&dagger.Secret{}), nil
	}

	_, err = interp.Render(tokens, resolve)
	if err == nil {
		t.Fatal("Render() with a secret-resolving reference mixed into literal text must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "steps.build.output") {
		t.Fatalf("Render() error = %v, want it to name steps.build.output as the offending reference", err)
	}
}

// TestRender_SecretAloneInAFieldResolvesToSecretValue proves the accepted
// counterpart: a field whose ENTIRE value is a single secret reference
// (no literal text mixed in) resolves end-to-end to a KindSecret Value,
// exactly as the workflow-execution spec's "Secret value never appears as
// a plaintext string" scenario requires.
func TestRender_SecretAloneInAFieldResolvesToSecretValue(t *testing.T) {
	t.Parallel()

	tokens, err := interp.Scan(`${{ secrets.tok }}`)
	if err != nil {
		t.Fatalf("Scan() error = %v, want nil", err)
	}

	handle := &dagger.Secret{}
	resolve := func(ref interp.Reference) (interp.Value, error) {
		return interp.NewSecret(handle), nil
	}

	got, err := interp.Render(tokens, resolve)
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	if got.Kind() != interp.KindSecret {
		t.Fatalf("Render() Kind() = %v, want KindSecret", got.Kind())
	}
	secret, ok := got.Secret()
	if !ok || secret != handle {
		t.Fatalf("Render() Secret() = (%p, %v), want (%p, true)", secret, ok, handle)
	}
}

// TestRender_LiteralTextConcatenatesWithNonSecretReferences proves Render
// is not simply rejecting ALL multi-token fields — only ones that would
// need to stringify a secret. Ordinary variables interpolation still
// concatenates normally.
func TestRender_LiteralTextConcatenatesWithNonSecretReferences(t *testing.T) {
	t.Parallel()

	tokens, err := interp.Scan(`hello ${{ variables.name }}!`)
	if err != nil {
		t.Fatalf("Scan() error = %v, want nil", err)
	}

	resolve := func(ref interp.Reference) (interp.Value, error) {
		return interp.NewString("world"), nil
	}

	got, err := interp.Render(tokens, resolve)
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	want := "hello world!"
	s, ok := got.String()
	if !ok || s != want {
		t.Fatalf("Render() String() = (%q, %v), want (%q, true)", s, ok, want)
	}
}
