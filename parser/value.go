package parser

// Kind identifies the JSON kind of a Value.
type Kind int

const (
	KindNull Kind = iota
	KindBool
	KindNumber
	KindString
	KindObject
	KindArray
)

func (k Kind) String() string {
	switch k {
	case KindNull:
		return "null"
	case KindBool:
		return "boolean"
	case KindNumber:
		return "number"
	case KindString:
		return "string"
	case KindObject:
		return "object"
	case KindArray:
		return "array"
	default:
		return "invalid"
	}
}

// Value is a node in the positioned JSON tree the decode pass produces.
// Exactly the field matching Kind is meaningful; the others hold their zero
// values. Template fields of the typed AST hold subtrees of this type
// directly.
type Value struct {
	Kind Kind
	Pos  Position

	Bool   bool     // KindBool
	Number string   // KindNumber: the literal as written in the source
	Str    string   // KindString: the decoded string contents
	Object []Member // KindObject: members in document order
	Array  []*Value // KindArray: elements in document order
}

// Member is one member of an object Value. NamePos locates the member's name;
// Value.Pos locates its value.
type Member struct {
	Name    string
	NamePos Position
	Value   *Value
}

// Field returns the value of the named member, or nil when v is not an
// object or has no member of that name.
func (v *Value) Field(name string) *Value {
	if v == nil || v.Kind != KindObject {
		return nil
	}
	for _, m := range v.Object {
		if m.Name == name {
			return m.Value
		}
	}
	return nil
}
