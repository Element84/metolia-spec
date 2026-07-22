package parser

// This file defines the typed AST for a workflow definition: the Flow
// object, the Step variants, and the shapes they carry (calls, middleware
// entries, and catch clauses). Fields typed *Value hold their positioned
// subtree from the decode pass directly; their string leaves are
// classified with ExprBody.

// inline call.flow value. All three carry the same fields; only the root
// carries Schema.
type Flow struct {
	Pos     Position
	Comment string

	// Schema is the root document's $schema URI; empty on a named or
	// inline Flow.
	Schema     string
	Entrypoint Name
	Flows      []NamedFlow
	Middleware []MiddlewareEntry
	// Parameters is the Flow's JSON Schema document, held as its raw
	// positioned subtree. The parser checks only that it is an object
	// declaring "type": "object"; evaluating it is a consumer's concern.
	Parameters *Value
	Steps      []NamedStep

	stepIndex map[string]int
	flowIndex map[string]int
}

// NamedFlow is one flows entry, in document order.
type NamedFlow struct {
	Name    string
	NamePos Position
	Flow    *Flow
}

// NamedStep is one steps entry, in document order.
type NamedStep struct {
	Name    string
	NamePos Position
	Step    Step
}

// StepByName returns the named Step, or false when the name is not a key of
// this Flow's steps.
func (f *Flow) StepByName(name string) (NamedStep, bool) {
	i, ok := f.stepIndex[name]
	if !ok {
		return NamedStep{}, false
	}
	return f.Steps[i], true
}

// FlowByName returns the named subflow declared in this Flow's own flows
// map, or false when absent. It does not walk the lexical chain; scope
// resolution is the check pass's concern.
func (f *Flow) FlowByName(name string) (NamedFlow, bool) {
	i, ok := f.flowIndex[name]
	if !ok {
		return NamedFlow{}, false
	}
	return f.Flows[i], true
}

// Step is a Step definition: one of the seven action variants. The set is
// closed; consumers dispatch by type switch over *CallStep, *GatherStep,
// *MatchStep, *PassStep, *SleepStep, *ReturnStep, and *RaiseStep.
type Step interface {
	isStep()
}

func (*CallStep) isStep()   {}
func (*GatherStep) isStep() {}
func (*MatchStep) isStep()  {}
func (*PassStep) isStep()   {}
func (*SleepStep) isStep()  {}
func (*ReturnStep) isStep() {}
func (*RaiseStep) isStep()  {}

// CallStep dispatches a single call and routes on the Result it yields.
type CallStep struct {
	Pos     Position
	Comment string

	Call       *Call
	Input      *Value
	Output     *Value
	Assign     []Member
	Middleware []MiddlewareEntry
	Catch      []CatchClause
	Next       Name
}

// GatherStep fans dispatches out concurrently and gathers the Results back
// in. It carries exactly one of its two forms: Over with Call (iterate), or
// Calls (scatter).
type GatherStep struct {
	Pos     Position
	Comment string

	// Over and Call are the iterate form: one dispatch of the Call
	// template per element of the Over collection.
	Over *OrExpr[[]*Value]
	Call *Call
	// Calls is the scatter form: one dispatch per entry.
	Calls []*Call

	Completion *Completion
	// Concurrency caps the dispatches active at once; nil means no cap:
	// the field is absent, or an explicit null, which binds identically.
	Concurrency *ConcurrencyCap
	Output      *Value
	Assign      []Member
	Catch       []CatchClause
	Next        Name
}

// ConcurrencyCap is a Gather's concurrency cap: the cap value and its
// position. A Gather without a cap carries no ConcurrencyCap at all.
type ConcurrencyCap struct {
	Pos   Position
	Value int64
}

// Completion is a Gather's completion policy.
type Completion struct {
	Pos Position

	Successes OrExpr[int64]
	// Wait is a literal; nil means the default, true.
	Wait *bool
}

// MatchStep routes to one of several successors by testing predicates
// against a single shaped value.
type MatchStep struct {
	Pos     Position
	Comment string

	Input   *Value
	Cases   []MatchCase
	Default MatchDefault
}

// MatchCase is one cases clause: a predicate with the routing, shaping, and
// capture that apply when it is selected.
type MatchCase struct {
	Pos     Position
	Comment string

	When   Expr
	Output *Value
	Assign []Member
	Next   Name
}

// MatchDefault is the required fallback clause: the case shape without a
// predicate.
type MatchDefault struct {
	Pos     Position
	Comment string

	Output *Value
	Assign []Member
	Next   Name
}

// PassStep performs no action work: it emits a value and captures
// variables, then transitions.
type PassStep struct {
	Pos     Position
	Comment string

	Output *Value
	Assign []Member
	Next   Name
}

// SleepStep pauses the frame for a duration or until an instant. Exactly
// one of For and Until is set; the bind pass guarantees the invariant.
type SleepStep struct {
	Pos     Position
	Comment string

	For   *OrExpr[string]
	Until *OrExpr[string]
	Next  Name
}

// ReturnStep is the terminal success: the frame completes with a success
// Result carrying the shaped value.
type ReturnStep struct {
	Pos     Position
	Comment string

	Value *Value
}

// RaiseStep is the terminal failure. With a Result, it constructs the
// failure; bare, it re-raises the failure being handled.
type RaiseStep struct {
	Pos     Position
	Comment string

	Result *RaiseResult
}

// RaiseResult holds the authorable fields of a constructed failure
// envelope. Unset fields are absent from the envelope.
type RaiseResult struct {
	Pos Position

	// Type names the failure Result's type; nil defaults to "error".
	Type      *OrExpr[string]
	Code      OrExpr[string]
	Message   *OrExpr[string]
	Details   *Value
	Retryable *RetryableField
	Previous  *PreviousField
}

// Call is a call object: a request to run a target and obtain its Result.
// Exactly one of Provider and Flow is set; the bind pass guarantees the
// invariant.
type Call struct {
	Pos     Position
	Comment string

	Provider *ProviderRef
	Flow     *FlowTarget

	Input     *Value
	With      *OrExpr[[]Member]
	OnSuccess *SuccessArm
	OnFailure *FailureArm
}

// ProviderRef is a provider URI as written, checked as a well-formed name
// of the kind its position requires and left unresolved.
type ProviderRef struct {
	URI string
	Pos Position
}

// FlowTarget is a call's flow target: a named Flow referenced by bare name,
// or an inline Flow object. Exactly one of Name and Inline is set.
type FlowTarget struct {
	Name   *Name
	Inline *Flow
}

// SuccessArm is a call's onSuccess arm: shapes the value placed in the
// Call's success Result and captures variables.
type SuccessArm struct {
	Pos Position

	Value  *Value
	Assign []Member
}

// FailureArm is a call's onFailure arm: captures variables from the failed
// dispatch's live context.
type FailureArm struct {
	Pos Position

	Assign []Member
}

// CatchClause is a conditional failure edge: when the Step's failure
// matches, control transitions to the clause's Next.
type CatchClause struct {
	Pos     Position
	Comment string

	Match  FailureMatcher
	Output *Value
	Assign []Member
	Next   Name
}

// FailureMatcher selects failures. Each present member constrains one
// contract field of the failure envelope, and every member present must
// match; the bind pass guarantees at least one member is present.
type FailureMatcher struct {
	Pos Position

	Codes     []StringLit
	Types     []StringLit
	Retryable *bool
}

// MiddlewareEntry names a middleware by provider URI and configures its
// participation phase by phase. An absent phase is equivalent to an empty
// one.
type MiddlewareEntry struct {
	Pos     Position
	Comment string

	Provider  ProviderRef
	OnEntry   *PhaseOnEntry
	OnSuccess *PhaseOnSuccess
	OnFailure *PhaseOnFailure
	OnAlways  *PhaseOnAlways
}

// PhaseOnEntry configures the entry's onEntry phase.
type PhaseOnEntry struct {
	Pos Position

	When   *OrExpr[bool]
	With   *OrExpr[[]Member]
	Assign []Member
	Output *Value
}

// PhaseOnSuccess configures the entry's onSuccess phase.
type PhaseOnSuccess struct {
	Pos Position

	When   *OrExpr[bool]
	With   *OrExpr[[]Member]
	Assign []Member
	Value  *Value
}

// PhaseOnFailure configures the entry's onFailure phase. Writing any
// envelope field constructs a new failure superseding the rising one.
type PhaseOnFailure struct {
	Pos Position

	When   *OrExpr[bool]
	With   *OrExpr[[]Member]
	Assign []Member

	Type      *OrExpr[string]
	Code      *OrExpr[string]
	Message   *OrExpr[string]
	Details   *Value
	Retryable *RetryableField
	Previous  *PreviousField
}

// PhaseOnAlways configures the entry's onAlways phase, which passes the
// rising Result through unchanged.
type PhaseOnAlways struct {
	Pos Position

	When   *OrExpr[bool]
	With   *OrExpr[[]Member]
	Assign []Member
}
