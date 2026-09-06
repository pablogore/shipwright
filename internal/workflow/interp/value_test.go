// Tests for tasks.md 5.5 (GREEN — the typed Value carrier design.md D-L
// specifies: Value{kind, str, secret}, KindSecret with no string
// accessor).
package interp_test

import (
	"testing"

	"dagger.io/dagger"
	"github.com/pablogore/shipwright/internal/workflow/interp"
)

func TestNewString_KindAndAccessor(t *testing.T) {
	t.Parallel()

	v := interp.NewString("hello")

	if v.Kind() != interp.KindString {
		t.Fatalf("Kind() = %v, want KindString", v.Kind())
	}

	got, ok := v.String()
	if !ok {
		t.Fatal("String() ok = false for a KindString value, want true")
	}
	if got != "hello" {
		t.Fatalf("String() = %q, want %q", got, "hello")
	}

	if _, ok := v.Secret(); ok {
		t.Fatal("Secret() ok = true for a KindString value, want false")
	}
}

func TestNewInt_KindAndAccessor(t *testing.T) {
	t.Parallel()

	v := interp.NewInt(42)

	if v.Kind() != interp.KindInt {
		t.Fatalf("Kind() = %v, want KindInt", v.Kind())
	}

	got, ok := v.String()
	if !ok {
		t.Fatal("String() ok = false for a KindInt value, want true")
	}
	if got != "42" {
		t.Fatalf("String() = %q, want %q", got, "42")
	}
}

func TestNewBool_KindAndAccessor(t *testing.T) {
	t.Parallel()

	v := interp.NewBool(true)

	if v.Kind() != interp.KindBool {
		t.Fatalf("Kind() = %v, want KindBool", v.Kind())
	}

	got, ok := v.String()
	if !ok {
		t.Fatal("String() ok = false for a KindBool value, want true")
	}
	if got != "true" {
		t.Fatalf("String() = %q, want %q", got, "true")
	}
}

// TestNewSecret_StringAccessorRefuses is the non-reflection-based
// companion to the dedicated security sweep in security_test.go: it
// asserts, via the ordinary typed API a caller would actually use, that
// String() on a KindSecret Value reports ok=false rather than returning
// the secret's payload as a string.
func TestNewSecret_StringAccessorRefuses(t *testing.T) {
	t.Parallel()

	handle := &dagger.Secret{}
	v := interp.NewSecret(handle)

	if v.Kind() != interp.KindSecret {
		t.Fatalf("Kind() = %v, want KindSecret", v.Kind())
	}

	if s, ok := v.String(); ok {
		t.Fatalf("String() ok = true for a KindSecret value (returned %q), want ok=false", s)
	}

	got, ok := v.Secret()
	if !ok {
		t.Fatal("Secret() ok = false for a KindSecret value, want true")
	}
	if got != handle {
		t.Fatalf("Secret() = %p, want the exact handle passed to NewSecret (%p)", got, handle)
	}
}

func TestKind_StringRepresentations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind interp.Kind
		want string
	}{
		{interp.KindString, "string"},
		{interp.KindInt, "int"},
		{interp.KindBool, "bool"},
		{interp.KindSecret, "secret"},
	}

	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Fatalf("Kind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}
