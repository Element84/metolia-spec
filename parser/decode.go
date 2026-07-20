package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// decode is the well-formedness layer: it reads src token by token into a
// positioned value tree, rejecting what is not legal input at the document
// level — JSON syntax errors, a repeated member name within one object, and
// content after the root value. A syntax error yields a nil tree; every
// other finding is recorded as a diagnostic beside the tree it was found in,
// so later passes can keep collecting.
func decode(src []byte, filename string) (*Value, Diagnostics) {
	d := &decoder{
		src:   src,
		dec:   json.NewDecoder(bytes.NewReader(src)),
		file:  filename,
		index: newLineIndex(src),
	}
	// Numbers arrive as json.Number so the literal survives as written.
	d.dec.UseNumber()

	root, err := d.readValue()
	if err != nil {
		d.syntaxError(err)
		return nil, d.diags
	}

	before := d.dec.InputOffset()
	if _, err := d.dec.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			d.syntaxError(err)
		} else {
			pos := d.position(d.tokenStart(before))
			d.errorAt("trailing-content", pos, "", "unexpected content after the root value")
		}
	}
	return root, d.diags
}

type decoder struct {
	src   []byte
	dec   *json.Decoder
	file  string
	index *lineIndex
	path  []pathSeg
	diags Diagnostics
}

// readValue reads one complete JSON value, including nested structure.
func (d *decoder) readValue() (*Value, error) {
	before := d.dec.InputOffset()
	tok, err := d.dec.Token()
	if err != nil {
		return nil, err
	}
	pos := d.position(d.tokenStart(before))

	switch t := tok.(type) {
	case json.Delim:
		if t == '{' {
			return d.object(pos)
		}
		return d.array(pos)
	case nil:
		return &Value{Kind: KindNull, Pos: pos}, nil
	case bool:
		return &Value{Kind: KindBool, Pos: pos, Bool: t}, nil
	case json.Number:
		return &Value{Kind: KindNumber, Pos: pos, Number: t.String()}, nil
	case string:
		return &Value{Kind: KindString, Pos: pos, Str: t}, nil
	default:
		return nil, fmt.Errorf("unexpected token %v", tok)
	}
}

func (d *decoder) object(pos Position) (*Value, error) {
	v := &Value{Kind: KindObject, Pos: pos}
	var seen map[string]Position
	for d.dec.More() {
		before := d.dec.InputOffset()
		tok, err := d.dec.Token()
		if err != nil {
			return nil, err
		}
		name := tok.(string) // object keys are always strings
		namePos := d.position(d.tokenStart(before))

		d.path = append(d.path, pathSeg{key: name})
		val, err := d.readValue()
		d.path = d.path[:len(d.path)-1]
		if err != nil {
			return nil, err
		}

		if seen == nil {
			seen = make(map[string]Position)
		}
		if first, dup := seen[name]; dup {
			d.errorAt("duplicate-member", namePos, renderPath(d.path, name),
				"duplicate object member %q; first defined at %s", name, first)
			continue
		}
		seen[name] = namePos
		v.Object = append(v.Object, Member{Name: name, NamePos: namePos, Value: val})
	}
	// Consume the closing brace.
	if _, err := d.dec.Token(); err != nil {
		return nil, err
	}
	return v, nil
}

func (d *decoder) array(pos Position) (*Value, error) {
	v := &Value{Kind: KindArray, Pos: pos}
	for d.dec.More() {
		d.path = append(d.path, pathSeg{idx: len(v.Array), isIdx: true})
		elem, err := d.readValue()
		d.path = d.path[:len(d.path)-1]
		if err != nil {
			return nil, err
		}
		v.Array = append(v.Array, elem)
	}
	// Consume the closing bracket.
	if _, err := d.dec.Token(); err != nil {
		return nil, err
	}
	return v, nil
}

// tokenStart finds the exact start offset of the next token at or after
// from, which is a Decoder.InputOffset checkpoint: between two tokens the
// grammar permits only whitespace and the , and : separators, which the
// Decoder consumes silently.
func (d *decoder) tokenStart(from int64) int {
	i := int(from)
	for i < len(d.src) {
		switch d.src[i] {
		case ' ', '\t', '\r', '\n', ',', ':':
			i++
		default:
			return i
		}
	}
	return i
}

func (d *decoder) position(offset int) Position {
	line, col := d.index.lineCol(offset)
	return Position{Filename: d.file, Offset: offset, Line: line, Column: col}
}

func (d *decoder) errorAt(code string, pos Position, path string, format string, args ...any) {
	d.diags = append(d.diags, Diagnostic{
		Severity: SeverityError,
		Code:     code,
		Pos:      pos,
		Path:     path,
		Message:  fmt.Sprintf(format, args...),
	})
}

// syntaxError converts a Decoder error into a diagnostic.
func (d *decoder) syntaxError(err error) {
	var syn *json.SyntaxError
	switch {
	// The Decoder reports plain io.EOF both for an empty document and for
	// input truncated between tokens; the source tells the two apart.
	case errors.Is(err, io.EOF) && len(bytes.TrimSpace(d.src)) == 0:
		d.errorAt("empty-document", d.position(0), "",
			"the document is empty")
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		d.errorAt("json-syntax", d.position(len(d.src)), "",
			"unexpected end of input")
	case errors.As(err, &syn):
		offset := min(int(syn.Offset), len(d.src))
		d.errorAt("json-syntax", d.position(offset), "", "%s", syn.Error())
	default:
		d.errorAt("json-syntax", d.position(0), "", "%s", err.Error())
	}
}

// pathSeg is one segment of a document path: an object member name or an
// array index.
type pathSeg struct {
	key   string
	idx   int
	isIdx bool
}

// renderPath renders segments as a dotted path with bracketed indexes,
// e.g. "steps.greet.catch[0]". A non-empty finalKey is appended as a last
// member segment.
func renderPath(segs []pathSeg, finalKey string) string {
	var b strings.Builder
	writeKey := func(key string) {
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(key)
	}
	for _, s := range segs {
		if s.isIdx {
			fmt.Fprintf(&b, "[%d]", s.idx)
		} else {
			writeKey(s.key)
		}
	}
	if finalKey != "" {
		writeKey(finalKey)
	}
	return b.String()
}

// lineIndex translates byte offsets into 1-based line and column numbers.
type lineIndex struct {
	starts []int // offset of each line's first byte
}

func newLineIndex(src []byte) *lineIndex {
	starts := []int{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return &lineIndex{starts: starts}
}

func (li *lineIndex) lineCol(offset int) (line, col int) {
	i := sort.SearchInts(li.starts, offset+1) - 1
	return i + 1, offset - li.starts[i] + 1
}
