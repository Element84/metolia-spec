package parser

import "strings"

// ExprBody reports whether s is an expression under the specification's
// embedding rule: a string whose entire content one {{ }} delimiter pair
// sets off. Any other string is a literal. The returned body is the text
// between the delimiters with surrounding whitespace removed, which the
// specification declares insignificant.
//
// The test mirrors the published schema's pattern exactly: the string must
// begin with "{{" and end with "}}". The parser does not inspect the body;
// whether it is valid in the expression language is a concern for consumers
// that evaluate or analyze expressions.
func ExprBody(s string) (body string, ok bool) {
	if len(s) < 4 || !strings.HasPrefix(s, "{{") || !strings.HasSuffix(s, "}}") {
		return "", false
	}
	return strings.TrimSpace(s[2 : len(s)-2]), true
}
