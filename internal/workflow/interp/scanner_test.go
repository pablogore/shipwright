// Tests for tasks.md 5.1 (RED — every rejected grammar form) and the
// accepted-form baseline that 5.1's rejections are contrasted against
// (design.md D-L's closed grammar).
package interp_test

import (
	"strings"
	"testing"

	"github.com/pablogore/shipwright/internal/workflow/interp"
)

// TestScan_AcceptsClosedGrammar is the accepted-form baseline: each of the
// three ref forms design.md D-L defines (variables.<name>, secrets.<name>,
// steps.<id>.output) scans to exactly one TokenReference, and literal text
// around a placeholder scans to TokenLiteral runs.
func TestScan_AcceptsClosedGrammar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []interp.Token
	}{
		{
			name:  "variables reference alone",
			input: "${{ variables.env }}",
			want: []interp.Token{
				{Kind: interp.TokenReference, Ref: interp.Reference{Namespace: interp.NamespaceVariables, Name: "env"}},
			},
		},
		{
			name:  "secrets reference alone",
			input: "${{ secrets.registry-token }}",
			want: []interp.Token{
				{Kind: interp.TokenReference, Ref: interp.Reference{Namespace: interp.NamespaceSecrets, Name: "registry-token"}},
			},
		},
		{
			name:  "steps output reference alone",
			input: "${{ steps.build.output }}",
			want: []interp.Token{
				{Kind: interp.TokenReference, Ref: interp.Reference{Namespace: interp.NamespaceSteps, StepID: "build"}},
			},
		},
		{
			name:  "literal text with no placeholder",
			input: "just plain text",
			want: []interp.Token{
				{Kind: interp.TokenLiteral, Literal: "just plain text"},
			},
		},
		{
			name:  "literal surrounding a reference",
			input: "prefix ${{ variables.env }} suffix",
			want: []interp.Token{
				{Kind: interp.TokenLiteral, Literal: "prefix "},
				{Kind: interp.TokenReference, Ref: interp.Reference{Namespace: interp.NamespaceVariables, Name: "env"}},
				{Kind: interp.TokenLiteral, Literal: " suffix"},
			},
		},
		{
			name:  "no interior whitespace still parses",
			input: "${{variables.env}}",
			want: []interp.Token{
				{Kind: interp.TokenReference, Ref: interp.Reference{Namespace: interp.NamespaceVariables, Name: "env"}},
			},
		},
		{
			name:  "empty input produces no tokens",
			input: "",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := interp.Scan(tt.input)
			if err != nil {
				t.Fatalf("Scan(%q) error = %v, want nil", tt.input, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Scan(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("Scan(%q)[%d] = %#v, want %#v", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestScan_RejectsEveryNonGrammarForm is tasks.md 5.1's exhaustive RED
// list. Every one of these MUST fail at Scan (design.md D-H stage 4,
// "interpolation reference to an undeclared variable/secret/step" and
// D-L's "anything else... is a parse error at stage 4, not a fallback to
// literal text") — Scan must never return tokens alongside a non-nil
// error, proving there is no partial/best-effort interpretation.
func TestScan_RejectsEveryNonGrammarForm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "operator inside placeholder",
			input: `${{ variables.env == "prod" }}`,
		},
		{
			name:  "function call",
			input: "${{ upper(variables.env) }}",
		},
		{
			name:  "nested placeholder",
			input: "${{ variables.${{ x }} }}",
		},
		{
			name:  "unknown namespace",
			input: "${{ env.FOO }}",
		},
		{
			name:  "trailing path segment on secrets",
			input: "${{ secrets.tok.extra }}",
		},
		{
			name:  "trailing path segment on steps output",
			input: "${{ steps.build.output.extra }}",
		},
		{
			name:  "steps reference missing .output suffix",
			input: "${{ steps.build }}",
		},
		{
			name:  "unclosed placeholder",
			input: "prefix ${{ variables.env",
		},
		{
			name:  "stray closing delimiter with no opener",
			input: "some text }} more text",
		},
		{
			name:  "empty placeholder body",
			input: "${{ }}",
		},
		{
			name:  "namespace with no name",
			input: "${{ variables. }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := interp.Scan(tt.input)
			if err == nil {
				t.Fatalf("Scan(%q) error = nil, want a stage-4 parse error", tt.input)
			}
			if got != nil {
				t.Fatalf("Scan(%q) returned %d tokens alongside an error — a rejected form must never fall back to a partial/literal-text interpretation", tt.input, len(got))
			}
		})
	}
}

// TestScan_LiteralContainingCloseDelimiterIsRejected proves a literal run
// that contains a stray "}}" (outside any "${{...}}" placeholder) is
// rejected rather than passed through — the closed-grammar scanner treats
// an unmatched close delimiter as malformed, never as ordinary text
// (tasks.md 5.1, "malformed delimiters").
func TestScan_LiteralContainingCloseDelimiterIsRejected(t *testing.T) {
	t.Parallel()

	_, err := interp.Scan("weird }} text")
	if err == nil {
		t.Fatal("Scan() with a stray }} delimiter must return an error, got nil")
	}
	if !strings.Contains(err.Error(), "}}") {
		t.Fatalf("Scan() error = %v, want it to name the stray delimiter", err)
	}
}
