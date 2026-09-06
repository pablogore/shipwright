// Package interp implements stage 4 (references) of the manifest's fixed
// seven-stage validation pipeline (design.md D-H): a hand-written scanner
// over a CLOSED interpolation grammar (design.md D-L), plus the typed
// Value carrier that makes secrecy a structural property rather than a
// runtime discipline.
//
// This is deliberately NOT a general expression evaluator. design.md D-L
// rejected text/template (supports function calls and reflective field
// traversal), a CEL/expr library (arbitrary-expression evaluation), and
// os.Expand ($VAR leaks the process environment) in favor of a small,
// closed, fully-static grammar:
//
//	placeholder := "${{" ws ref ws "}}"
//	ref         := "variables." name
//	             | "secrets."   name
//	             | "steps."     name ".output"
//	name        := [A-Za-z_][A-Za-z0-9_-]*
//
// Anything else is a parse error at Scan, never a fallback to literal
// text — there is no evaluation step to attack.
//
// Stage 7 (value binding, "Secret reference in a non-secret-typed field
// is a stage-7 validation error") is NOT implemented here: it requires a
// provider's declared WithSchema field kinds, which do not exist until
// internal/workflow/providers (Phase 7). This package ships the primitive
// that stage 7 depends on (Reference.StaticKind) — see reference_test.go
// for the exact boundary.
package interp

import (
	"strconv"

	"dagger.io/dagger"
)

// Kind identifies what a Value carries. It is deliberately a closed,
// small enum matching design.md D-L's type sketch exactly — no
// provider-specific kinds, no open extension point.
type Kind int

const (
	KindString Kind = iota
	KindInt
	KindBool
	KindSecret
)

// String renders a Kind's name for error messages and diagnostics. This
// is safe for a KindSecret value itself (it only ever names the KIND, it
// never touches a Value's payload) — do not confuse this with
// Value.String(), which is the one that must never leak a secret.
func (k Kind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindInt:
		return "int"
	case KindBool:
		return "bool"
	case KindSecret:
		return "secret"
	default:
		return "unknown"
	}
}

// Value is the typed result of resolving an interpolation reference
// (design.md D-L, the mechanically-enforced secret rule). Its shape
// matches design.md's type sketch exactly:
//
//	type Value struct {
//		kind   Kind
//		str    string          // never set when kind == KindSecret
//		secret *dagger.Secret  // only set when kind == KindSecret
//	}
//
// Secrecy is a STRUCTURAL property, not a runtime discipline: str is
// never populated by any constructor when kind == KindSecret (see
// NewSecret), and no exported accessor returns a secret's payload as a
// Go string (see String, and the reflection-based proof in
// security_test.go). This is a stronger guarantee than "any policy that
// merely forbids substitution" — there is no code path that COULD produce
// a plaintext secret string, because the string-producing path cannot
// hold a secret value at all.
type Value struct {
	kind   Kind
	str    string
	secret *dagger.Secret
}

// NewString constructs a KindString Value.
func NewString(s string) Value {
	return Value{kind: KindString, str: s}
}

// NewInt constructs a KindInt Value. Value's shape (design.md D-L) has a
// single str field carrying every non-secret kind's textual
// representation — there is no separate numeric field to invent, and no
// consumer in this change needs int arithmetic on a Value (that belongs
// to a later phase's execution engine, if it ever needs it).
func NewInt(n int64) Value {
	return Value{kind: KindInt, str: strconv.FormatInt(n, 10)}
}

// NewBool constructs a KindBool Value, following the same
// single-str-field representation as NewInt.
func NewBool(b bool) Value {
	return Value{kind: KindBool, str: strconv.FormatBool(b)}
}

// NewSecret constructs a KindSecret Value wrapping handle. This is the
// ONLY constructor that ever sets the secret field, and it deliberately
// never touches str — a KindSecret Value's str is always the zero value,
// by construction, never by convention.
func NewSecret(handle *dagger.Secret) Value {
	return Value{kind: KindSecret, secret: handle}
}

// Kind reports the Value's kind. This is always safe to call and never
// exposes a secret's payload.
func (v Value) Kind() Kind {
	return v.kind
}

// String returns v's textual representation and reports whether one
// exists. It reports ok=false for a KindSecret Value — there is no
// accessor on Value that returns a secret as a string (design.md D-L).
// This check is explicit, not merely incidental to str being unset: even
// if a future refactor accidentally populated str alongside a secret,
// String() would still refuse to return it.
func (v Value) String() (s string, ok bool) {
	if v.kind == KindSecret {
		return "", false
	}
	return v.str, true
}

// Secret returns v's *dagger.Secret handle and reports whether v actually
// carries one. It is the ONLY accessor that can observe a KindSecret
// Value's payload, and it never exposes a plaintext string — callers
// receive the same opaque handle type every other secret-consuming
// surface in this codebase already uses (pkg/shipwright's capability
// interfaces, internal/pipelines/shared/docker.go's client.SetSecret
// pattern).
func (v Value) Secret() (handle *dagger.Secret, ok bool) {
	if v.kind != KindSecret {
		return nil, false
	}
	return v.secret, true
}
