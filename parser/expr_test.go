package parser

import "testing"

func TestExprBody(t *testing.T) {
	tests := []struct {
		in   string
		body string
		ok   bool
	}{
		{`{{ vars.collection }}`, "vars.collection", true},
		{`{{vars.collection}}`, "vars.collection", true},
		{"{{\n  step.input\n}}", "step.input", true},
		{`{{}}`, "", true},
		{`{{ }}`, "", true},
		// Mirrors the schema pattern: starts with {{ and ends with }},
		// whatever lies between. The body may be nonsense the
		// expression language will reject; that is not this rule's
		// concern.
		{`{{a}} {{b}}`, "a}} {{b", true},
		{`{{ 'a}}b' }}`, "'a}}b'", true},

		// Literals: no interpolation, no partial delimiters.
		{`hello`, "", false},
		{`hello {{ x }}`, "", false},
		{`{{ x }} world`, "", false},
		{`{{`, "", false},
		{`}}`, "", false},
		{`{{}`, "", false},
		{`{ { x } }`, "", false},
		{``, "", false},
	}
	for _, tt := range tests {
		body, ok := ExprBody(tt.in)
		if body != tt.body || ok != tt.ok {
			t.Errorf("ExprBody(%q) = (%q, %v), want (%q, %v)", tt.in, body, ok, tt.body, tt.ok)
		}
	}
}
