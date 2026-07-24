// The JSON encoding of the parser's AST, per DESIGN.md ("The AST encoding
// makes the parser's decisions visible"). The bulk of the encoding is
// derived from the AST structs by reflection -- field names camelCased,
// nil pointers and empty slices omitted (which collapses the
// literal-or-expression pair types to their one-key form), unexported
// fields invisible -- so a field added to the AST appears in the output
// without touching this file. The action discriminator comes from the
// variant itself (Step.Action). What remains hand-written is exactly the
// set of semantic decisions the struct shapes cannot express: position
// rendering, value trees with classified string leaves, the explicit-null
// pair types, and the "pos"-last key rule.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/Element84/metolia-spec/parser"
)

// writeEnvelope emits the {"filename", "ast", "diagnostics"} envelope,
// indented, with HTML escaping off so expression bodies read as written.
// ast is the document's encoded AST (encFlow, encProvider); nil marks a
// failed parse and renders as JSON null.
func writeEnvelope(w io.Writer, filename string, ast *obj, ds parser.Diagnostics) error {
	env := newObj()
	if filename != "" {
		env.set("filename", filename)
	}
	if ast != nil {
		env.set("ast", ast)
	} else {
		env.set("ast", nil)
	}
	dlist := make([]any, 0, len(ds))
	for _, d := range ds {
		dlist = append(dlist, encDiagnostic(d))
	}
	env.set("diagnostics", dlist)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

// obj is a JSON object that marshals its members in insertion order.
// encoding/json's map type randomizes key order; deterministic output is a
// requirement here (golden files diff byte-for-byte).
type obj struct {
	keys []string
	vals []any
}

func newObj() *obj { return &obj{} }

func (o *obj) set(key string, val any) *obj {
	o.keys = append(o.keys, key)
	o.vals = append(o.vals, val)
	return o
}

func (o *obj) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, err := marshalNoEscape(k)
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteByte(':')
		vb, err := marshalNoEscape(o.vals[i])
		if err != nil {
			return nil, err
		}
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// marshalNoEscape is json.Marshal without HTML escaping, so "&&" and "<" in
// expression bodies survive as written.
func marshalNoEscape(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

func encPos(p parser.Position) *obj {
	// Filename is deliberately dropped: identical for every position in a
	// parse, it is hoisted to the envelope.
	return newObj().set("line", p.Line).set("col", p.Column).set("offset", p.Offset)
}

func encDiagnostic(d parser.Diagnostic) *obj {
	return newObj().
		set("severity", d.Severity.String()).
		set("code", d.Code).
		set("path", d.Path).
		set("message", d.Message).
		set("pos", encPos(d.Pos))
}

func encFlow(f *parser.Flow) *obj {
	return walkStruct(reflect.ValueOf(f).Elem(), "", true)
}

// encProvider walks with classification off throughout: providers have no
// expression fields, so string leaves in the embedded schemas stay plain
// strings.
func encProvider(p *parser.Provider) *obj {
	return walkStruct(reflect.ValueOf(p).Elem(), "", false)
}

// fieldKey names one field of one AST struct; the policy tables below are
// keyed by it so a policy never leaks onto a same-named field of another
// type.
type fieldKey struct {
	typ  reflect.Type
	name string
}

// keyOverrides holds the fields whose camelCased name is not the source's
// key.
var keyOverrides = map[fieldKey]string{
	{flowType, "Schema"}:     "$schema",
	{providerType, "Schema"}: "$schema",
	{providerType, "URI"}:    "uri",
	{providerRefType, "URI"}: "uri",
}

// fieldOrder overrides the struct declaration order where the output order
// deliberately differs from it: both roots lead with "$schema" the way a
// document does, and Provider renders its kind arm between the shared
// identity fields and the shared trailing fields, mirroring document order.
var fieldOrder = map[reflect.Type][]string{
	reflect.TypeOf(parser.Flow{}): {
		"Schema", "Comment", "Entrypoint", "Parameters",
		"Middleware", "Flows", "Steps", "Pos",
	},
	reflect.TypeOf(parser.Provider{}): {
		"Schema", "URI", "CodePrefix", "Description",
		"Call", "Middleware", "Metadata", "FailureCatalog", "Pos",
	},
}

// omitWhenEmptyString names the string fields where the empty string means
// "absent" rather than an authored empty value. Comment is not listed: like
// Pos, it is an AST-wide convention — optional on every struct that carries
// it — and walkStruct omits it by name.
var omitWhenEmptyString = map[fieldKey]bool{
	{flowType, "Schema"}:            true,
	{providerType, "Schema"}:        true,
	{phaseActionType, "Acceptance"}: true,
}

var (
	flowType        = reflect.TypeOf(parser.Flow{})
	providerType    = reflect.TypeOf(parser.Provider{})
	providerRefType = reflect.TypeOf(parser.ProviderRef{})
	phaseActionType = reflect.TypeOf(parser.PhaseAction{})
	positionType    = reflect.TypeOf(parser.Position{})
	valuePtrType    = reflect.TypeOf((*parser.Value)(nil))
)

// walkStruct derives a struct's JSON object from its fields: skip
// unexported fields, nil pointers, empty slices, and absent-meaning empty
// strings; camelCase the key; recurse on the value. An action discriminator
// leads when present, and "pos" always trails (DESIGN.md's key-order rule).
// classify is carried down to string leaves in value trees; it turns off
// under Flow.Parameters, the one flow-side Value field that is not a
// template.
func walkStruct(v reflect.Value, action string, classify bool) *obj {
	t := v.Type()
	o := newObj()
	if action != "" {
		o.set("action", action)
	}

	names := fieldOrder[t]
	if names == nil {
		for i := 0; i < t.NumField(); i++ {
			names = append(names, t.Field(i).Name)
		}
	}

	var pos any
	for _, name := range names {
		f, ok := t.FieldByName(name)
		if !ok || !f.IsExported() {
			continue
		}
		fv := v.FieldByIndex(f.Index)
		omitEmpty := name == "Comment" || omitWhenEmptyString[fieldKey{t, name}]
		switch {
		case fv.Kind() == reflect.Pointer && fv.IsNil(),
			fv.Kind() == reflect.Interface && fv.IsNil(),
			fv.Kind() == reflect.Slice && fv.Len() == 0,
			fv.Kind() == reflect.String && fv.String() == "" && omitEmpty:
			continue
		}
		if name == "Pos" && f.Type == positionType {
			pos = encPos(fv.Interface().(parser.Position))
			continue
		}
		key := keyOverrides[fieldKey{t, name}]
		if key == "" {
			key = lowerFirst(name)
		}
		schemaField := t == flowType && name == "Parameters"
		o.set(key, walkValue(fv, classify && !schemaField))
	}
	if pos != nil {
		o.set("pos", pos)
	}
	return o
}

// walkValue derives the JSON form of one value. The hand-written cases are
// the semantic decisions: positions, value trees, the Step discriminator,
// and the explicit-null pair types; everything else is generic.
func walkValue(v reflect.Value, classify bool) any {
	switch v.Type() {
	case positionType:
		return encPos(v.Interface().(parser.Position))
	case valuePtrType:
		return encValue(v.Interface().(*parser.Value), classify)
	}

	switch v.Kind() {
	case reflect.Pointer:
		return walkValue(v.Elem(), classify)
	case reflect.Interface:
		if step, ok := v.Interface().(parser.Step); ok {
			// Step -> *XStep -> XStep; the discriminator comes from the
			// variant itself.
			return walkStruct(v.Elem().Elem(), step.Action(), classify)
		}
		return walkValue(v.Elem(), classify)
	case reflect.Struct:
		switch x := v.Interface().(type) {
		case parser.RetryableField:
			return encRetryable(&x)
		case parser.PreviousField:
			return encPrevious(&x)
		}
		return walkStruct(v, "", classify)
	case reflect.Slice:
		out := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			out = append(out, walkValue(v.Index(i), classify))
		}
		return out
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int64:
		return v.Int()
	default:
		panic(fmt.Sprintf("unhandled kind %s for type %s", v.Kind(), v.Type()))
	}
}

func lowerFirst(s string) string {
	return string(s[0]|0x20) + s[1:]
}

func encRetryable(r *parser.RetryableField) *obj {
	switch {
	case r.Expr != nil:
		return newObj().set("expr", walkStruct(reflect.ValueOf(r.Expr).Elem(), "", true))
	case r.Null:
		return newObj().set("null", true)
	default:
		return newObj().set("literal", *r.Literal)
	}
}

func encPrevious(p *parser.PreviousField) *obj {
	switch {
	case p.Expr != nil:
		return newObj().set("expr", walkStruct(reflect.ValueOf(p.Expr).Elem(), "", true))
	case p.Null:
		return newObj().set("null", true)
	default:
		return newObj().set("literal", encValue(p.Literal, true))
	}
}

// encValue encodes a positioned value subtree. With classify set, string
// leaves are split into "string" and "expr" nodes by the same rule the
// parser applies everywhere (parser.ExprBody); without it (the parameters
// schema, which is not a template field) strings stay strings.
func encValue(v *parser.Value, classify bool) *obj {
	o := newObj()
	switch v.Kind {
	case parser.KindNull:
		o.set("kind", "null")
	case parser.KindBool:
		o.set("kind", "bool").set("value", v.Bool)
	case parser.KindNumber:
		// The literal text as written, emitted as a JSON string: no
		// float round-trip, no altered digits.
		o.set("kind", "number").set("value", v.Number)
	case parser.KindString:
		if body, ok := parser.ExprBody(v.Str); classify && ok {
			o.set("kind", "expr").set("body", body)
		} else {
			o.set("kind", "string").set("value", v.Str)
		}
	case parser.KindObject:
		members := make([]any, 0, len(v.Object))
		for _, m := range v.Object {
			members = append(members, newObj().
				set("name", m.Name).
				set("namePos", encPos(m.NamePos)).
				set("value", encValue(m.Value, classify)))
		}
		o.set("kind", "object").set("members", members)
	case parser.KindArray:
		elems := make([]any, 0, len(v.Array))
		for _, e := range v.Array {
			elems = append(elems, encValue(e, classify))
		}
		o.set("kind", "array").set("elements", elems)
	default:
		panic(fmt.Sprintf("unknown value kind %v", v.Kind))
	}
	return o.set("pos", encPos(v.Pos))
}
