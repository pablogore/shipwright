package interp

import (
	"fmt"
	"strings"
)

// Render combines tokens into a single Value, resolving each reference
// via resolve.
//
// A field containing exactly one Reference token and no literal text
// resolves directly to that reference's Value — so a field whose entire
// content is `${{ secrets.tok }}` resolves end-to-end to a KindSecret
// Value, exactly as the workflow-execution spec requires ("Secret
// Interpolation Resolves To Typed Handles, Never Plaintext").
//
// Any field mixing more than one token — literal text with a reference,
// or multiple references — must produce a single concatenated string,
// which requires calling String() on every resolved Value. Value's
// String() reports ok=false for KindSecret (design.md D-L: there is no
// accessor that returns a secret as a string), so Render rejects any such
// combination containing a secret reference BEFORE ever attempting to
// build the concatenated string — it never emits a partial result
// containing just the literal portion (tasks.md 5.4).
func Render(tokens []Token, resolve func(Reference) (Value, error)) (Value, error) {
	if len(tokens) == 0 {
		return NewString(""), nil
	}

	if len(tokens) == 1 && tokens[0].Kind == TokenReference {
		return resolve(tokens[0].Ref)
	}

	var b strings.Builder
	for _, tok := range tokens {
		switch tok.Kind {
		case TokenLiteral:
			b.WriteString(tok.Literal)

		case TokenReference:
			v, err := resolve(tok.Ref)
			if err != nil {
				return Value{}, err
			}
			s, ok := v.String()
			if !ok {
				return Value{}, fmt.Errorf(
					"interp: cannot concatenate secret reference %s with other content — a secret must be a field's entire value, never mixed with literal text or other references",
					describeRef(tok.Ref),
				)
			}
			b.WriteString(s)
		}
	}

	return NewString(b.String()), nil
}

// describeRef renders a Reference for an error message, without ever
// touching a resolved Value — it only ever sees the grammar-level
// reference, never a secret's payload.
func describeRef(ref Reference) string {
	switch ref.Namespace {
	case NamespaceVariables:
		return fmt.Sprintf("variables.%s", ref.Name)
	case NamespaceSecrets:
		return fmt.Sprintf("secrets.%s", ref.Name)
	case NamespaceSteps:
		return fmt.Sprintf("steps.%s.output", ref.StepID)
	default:
		return "unknown reference"
	}
}
