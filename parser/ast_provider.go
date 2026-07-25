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

	Schema         string         // the document's $schema URI
	URI            StringLit      // the provider's own URI
	CodePrefix     StringLit      // its identity segment in failure codes
	Description    string         // the human-readable description
	Metadata       *Value         // the metadata schema; or nil
	FailureCatalog FailureCatalog // the codes it can emit under its prefix

	Call       *CallProvider       // the call-provider arm
	Middleware *MiddlewareProvider // the middleware-provider arm
}

// CallProvider carries the fields only a call provider declares.
type CallProvider struct {
	Parameters *Value // the schema a call's with is validated against
}

// MiddlewareProvider carries the fields only a middleware provider
// declares.
type MiddlewareProvider struct {
	Attachment []StringLit    // "step", "flow", or both, each at most once
	Phases     PhaseContracts // the per-phase contract
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

	Action     PhaseAction // the action the phase declares
	Parameters *Value      // the schema the phase's with is validated against; or nil
	Structural []StringLit // names of the parameters taken structurally
}

// PhaseAction is the action a phase declares.
type PhaseAction struct {
	Pos Position

	Kind        StringLit // one of the ActionKind values
	Description string    // the human-readable description
	Acceptance  string    // a control action's commit point; empty when absent
}

// FailureCatalog declares the codes a provider can emit under its prefix.
type FailureCatalog struct {
	Pos Position

	Closed []StringLit // codes declared exactly
	Open   []StringLit // sub-prefixes where undeclared codes may appear
}
