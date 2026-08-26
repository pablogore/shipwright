package interp

import (
	"fmt"
	"strings"
)

// openDelim and closeDelim are the placeholder delimiters (design.md
// D-L). They are fixed literal strings, never configurable — a
// configurable delimiter would reopen the "arbitrary syntax" surface this
// grammar exists to close off.
const (
	openDelim  = "${{"
	closeDelim = "}}"
)

// TokenKind identifies what a Token carries.
type TokenKind int

const (
	// TokenLiteral is a run of ordinary text with no interpolation.
	TokenLiteral TokenKind = iota
	// TokenReference is a resolved "${{ ... }}" placeholder.
	TokenReference
)

// Namespace identifies which of the three closed grammar prefixes a
// Reference belongs to (design.md D-L's ref production).
type Namespace int

const (
	NamespaceVariables Namespace = iota
	NamespaceSecrets
	NamespaceSteps
)

// Reference is one resolved "${{ ... }}" placeholder body. Name is set
// for NamespaceVariables and NamespaceSecrets; StepID is set for
// NamespaceSteps. Every Reference produced by Scan is guaranteed to match
// the closed grammar exactly — there is no reference shape Scan can
// return that is not one of the three ref alternatives.
type Reference struct {
	Namespace Namespace
	Name      string
	StepID    string
}

// StaticKind reports the Value Kind a Reference is GUARANTEED to resolve
// to, when that is determinable from the grammar and manifest schema
// alone, or ok=false when the eventual Kind is not yet knowable at this
// stage.
//
// secrets.* is unconditionally KindSecret by construction (only
// NewSecret ever produces one). variables.* is unconditionally KindString
// because the manifest schema types spec.variables as
// map[string]string (internal/workflow/manifest.Spec.Variables) — there
// is no way for a variables.* reference to resolve to anything else.
//
// steps.<id>.output's Kind is fixed by whichever capability produced
// step <id>, which is not known until stage 6 provider resolution
// (internal/workflow/providers, Phase 7, design.md D-I's WithSchema).
// StaticKind reports ok=false rather than guessing, so a later stage
// cannot silently skip the provider-resolution step its OWN correctness
// actually depends on. This is the primitive tasks.md 5.3 ships: full
// stage-7 forbidPlaintext enforcement (comparing a field's declared kind
// against the reference's eventual kind) is Phase 7's job, and it MUST
// call StaticKind for variables./secrets. references and fall back to the
// resolved provider's WithSchema kind for steps.<id>.output references —
// this package has no provider schema to compare against, so it cannot
// and does not implement that comparison itself.
func (r Reference) StaticKind() (Kind, bool) {
	switch r.Namespace {
	case NamespaceSecrets:
		return KindSecret, true
	case NamespaceVariables:
		return KindString, true
	default:
		return 0, false
	}
}

// Token is one scanned unit: either a literal text run or a resolved
// reference. Scan emits []Token with no evaluation step — there is
// nothing further for a caller to interpret except look each Reference up
// (design.md D-L: "there is no evaluation step to attack").
type Token struct {
	Kind    TokenKind
	Literal string
	Ref     Reference
}

// Scan parses s against the closed interpolation grammar (design.md
// D-L). Every placeholder in s MUST match exactly one of the three ref
// alternatives; anything else — an operator, a function call, a nested
// placeholder, an unknown namespace, a trailing path segment, or a
// malformed delimiter — is a parse error. Scan never returns a non-nil
// token slice alongside a non-nil error: a rejected form is a hard
// failure, never a partial or best-effort literal-text fallback
// (tasks.md 5.1).
func Scan(s string) ([]Token, error) {
	var tokens []Token
	i := 0

	for i < len(s) {
		openIdx := strings.Index(s[i:], openDelim)
		if openIdx == -1 {
			rest := s[i:]
			if strings.Contains(rest, closeDelim) {
				return nil, fmt.Errorf("interp: stray %q delimiter with no matching %q", closeDelim, openDelim)
			}
			if rest != "" {
				tokens = append(tokens, Token{Kind: TokenLiteral, Literal: rest})
			}
			break
		}
		openIdx += i

		literal := s[i:openIdx]
		if strings.Contains(literal, closeDelim) {
			return nil, fmt.Errorf("interp: stray %q delimiter with no matching %q", closeDelim, openDelim)
		}
		if literal != "" {
			tokens = append(tokens, Token{Kind: TokenLiteral, Literal: literal})
		}

		bodyStart := openIdx + len(openDelim)

		nextOpenIdx := strings.Index(s[bodyStart:], openDelim)
		closeIdx := strings.Index(s[bodyStart:], closeDelim)
		if closeIdx == -1 {
			return nil, fmt.Errorf("interp: unclosed %q delimiter", openDelim)
		}
		if nextOpenIdx != -1 && nextOpenIdx < closeIdx {
			return nil, fmt.Errorf("interp: nested placeholder is not allowed")
		}

		body := s[bodyStart : bodyStart+closeIdx]
		ref, err := parseRef(body)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, Token{Kind: TokenReference, Ref: ref})

		i = bodyStart + closeIdx + len(closeDelim)
	}

	return tokens, nil
}

// parseRef parses a placeholder's trimmed body against the closed
// ref grammar. It is a plain string-prefix match, not a general parser —
// design.md D-L's ~80-line estimate for the whole scanner assumes exactly
// this shape.
func parseRef(body string) (Reference, error) {
	trimmed := strings.TrimSpace(body)

	switch {
	case strings.HasPrefix(trimmed, "variables."):
		name := trimmed[len("variables."):]
		if !validName(name) {
			return Reference{}, fmt.Errorf("interp: invalid variables reference %q", body)
		}
		return Reference{Namespace: NamespaceVariables, Name: name}, nil

	case strings.HasPrefix(trimmed, "secrets."):
		name := trimmed[len("secrets."):]
		if !validName(name) {
			return Reference{}, fmt.Errorf("interp: invalid secrets reference %q", body)
		}
		return Reference{Namespace: NamespaceSecrets, Name: name}, nil

	case strings.HasPrefix(trimmed, "steps."):
		rest := trimmed[len("steps."):]
		const outputSuffix = ".output"
		if !strings.HasSuffix(rest, outputSuffix) {
			return Reference{}, fmt.Errorf("interp: invalid steps reference %q, want steps.<id>.output", body)
		}
		id := strings.TrimSuffix(rest, outputSuffix)
		if !validName(id) {
			return Reference{}, fmt.Errorf("interp: invalid steps reference %q, want steps.<id>.output", body)
		}
		return Reference{Namespace: NamespaceSteps, StepID: id}, nil

	default:
		return Reference{}, fmt.Errorf("interp: %q is not a variables./secrets./steps.<id>.output reference", body)
	}
}

// validName reports whether s matches the grammar's name production:
// [A-Za-z_][A-Za-z0-9_-]*. This rejects everything an operator, function
// call, or trailing path segment would leave behind — a dotted, spaced,
// or punctuated remainder never matches this pattern.
func validName(s string) bool {
	if s == "" {
		return false
	}
	if !isNameStart(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isNameChar(s[i]) {
			return false
		}
	}
	return true
}

func isNameStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isNameChar(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9') || c == '-'
}
