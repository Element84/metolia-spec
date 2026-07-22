package parser

import "fmt"

// This file holds the machinery both check passes share: the checker,
// which accumulates diagnostics and renders document paths for the rules
// that span a bound document.

type checker struct {
	diags Diagnostics
	path  []pathSeg
}

func (c *checker) pushKey(name string) { c.path = append(c.path, pathSeg{key: name}) }
func (c *checker) pushIdx(i int)       { c.path = append(c.path, pathSeg{idx: i, isIdx: true}) }
func (c *checker) pop()                { c.path = c.path[:len(c.path)-1] }

func (c *checker) report(code string, pos Position, path, format string, args ...any) {
	c.diags = append(c.diags, Diagnostic{
		Severity: severityOf(code),
		Code:     code,
		Pos:      pos,
		Path:     path,
		Message:  fmt.Sprintf(format, args...),
	})
}
