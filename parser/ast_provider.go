package parser

// This file defines the typed AST for a provider definition document: the
// Provider object, its two kind arms, and the shapes they carry. Embedded
// schemas (parameters, metadata, per-phase parameters) are retained as
// positioned value trees whose string leaves stay plain strings: providers
// have no expression fields.

// The action kinds a middleware phase may declare. Published for consumer
// comparisons against PhaseAction.Kind; the bind pass validates against the
// same set.
const (
	ActionKindSideEffect = "side-effect"
	ActionKindControl    = "control"
	ActionKindTransform  = "transform"
)

// The levels a middleware provider's attachment may name. Published for
// consumer comparisons against MiddlewareProvider.Attachment entries; the
// bind pass validates against the same set.
const (
	AttachmentStep = "step"
	AttachmentFlow = "flow"
)

// Provider is a provider definition document: the interchange form of a
// catalog entry, for either provider kind. Fields the kinds share are flat;
// exactly one of Call and Middleware is set, discriminated by the URI's
// type segment. The bind pass guarantees the invariant.
type Provider struct {
	Pos Position

	// Schema is the document's $schema URI.
	Schema      string
	URI         StringLit
	CodePrefix  StringLit
	Description string
	// Metadata is the provider's metadata schema, held as its raw
	// positioned subtree; nil when absent.
	Metadata       *Value
	FailureCatalog FailureCatalog

	Call       *CallProvider
	Middleware *MiddlewareProvider
}

// CallProvider carries the fields only a call provider declares.
type CallProvider struct {
	// Parameters is the provider's parameter schema, held as its raw
	// positioned subtree.
	Parameters *Value
}

// MiddlewareProvider carries the fields only a middleware provider
// declares.
type MiddlewareProvider struct {
	// Attachment lists the levels the middleware may attach at: "step",
	// "flow", or both, each at most once.
	Attachment []StringLit
	Phases     PhaseContracts
}

// PhaseContracts is a middleware provider's per-phase contract. A nil phase
// declares no action there; at least one is set.
type PhaseContracts struct {
	Pos Position

	OnEntry   *PhaseContract
	OnSuccess *PhaseContract
	OnFailure *PhaseContract
	OnAlways  *PhaseContract
}

// PhaseContract is what the middleware declares for one phase.
type PhaseContract struct {
	Pos Position

	Action PhaseAction
	// Parameters is the schema the phase's with is validated against,
	// held as its raw positioned subtree; nil when absent.
	Parameters *Value
	// Structural names the parameters taken structurally.
	Structural []StringLit
}

// PhaseAction is the action a phase declares.
type PhaseAction struct {
	Pos Position

	// Kind is one of the ActionKind values.
	Kind        StringLit
	Description string
	// Acceptance is the commit point of a control action that races the
	// operation it wraps; empty when absent.
	Acceptance string
}

// FailureCatalog declares the codes a provider can emit under its prefix:
// closed codes, declared exactly, and open sub-prefixes, regions where
// codes not knowable in advance may appear.
type FailureCatalog struct {
	Pos Position

	Closed []StringLit
	Open   []StringLit
}
