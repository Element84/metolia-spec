package parser

// This file defines the typed AST for a workflow definition: the Flow
// object, the Step variants, and the shapes they carry (calls, middleware
// entries, and catch clauses). Fields typed *Value hold their positioned
// subtree from the decode pass directly; their string leaves are
// classified with ExprBody.

// Flow is one workflow definition: the root document, a named subflow, or
// an inline call.flow value. All three carry the same fields; only the root
// carries Schema.
type Flow struct {
	Pos     Position
	Comment string

	Schema     string            // the root document's $schema URI; empty on nested Flows
	Entrypoint Name              // the step execution enters at
	Flows      []NamedFlow       // the subflow ledger, document order
	Middleware []MiddlewareEntry // flow-level stack, wrapping every step
	Parameters *Value            // JSON Schema for the Flow's named parameters; or nil
	Steps      []NamedStep       // the step graph, document order

	stepIndex map[string]int // name → Steps index, built by bind for StepByName
	flowIndex map[string]int // name → Flows index, built by bind for FlowByName
}

// NamedFlow is one flows entry.
type NamedFlow struct {
	Name    string // the key the author chose
	NamePos Position
	Flow    *Flow // the subflow it names
}

// NamedStep is one steps entry.
type NamedStep struct {
	Name    string // the step's name, cited by entrypoint and next
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
// *MatchStep, *PassStep, *SleepStep, *ReturnStep, and *RaiseStep. Action
// returns the variant's action discriminator as written in the source
// ("Call", "Gather", ...), for consumers that render or re-emit a Step
// rather than dispatch on it.
type Step interface {
	isStep()
	Action() string
}

func (*CallStep) isStep()   {}
func (*GatherStep) isStep() {}
func (*MatchStep) isStep()  {}
func (*PassStep) isStep()   {}
func (*SleepStep) isStep()  {}
func (*ReturnStep) isStep() {}
func (*RaiseStep) isStep()  {}

func (*CallStep) Action() string   { return "Call" }
func (*GatherStep) Action() string { return "Gather" }
func (*MatchStep) Action() string  { return "Match" }
func (*PassStep) Action() string   { return "Pass" }
func (*SleepStep) Action() string  { return "Sleep" }
func (*ReturnStep) Action() string { return "Return" }
func (*RaiseStep) Action() string  { return "Raise" }

// CallStep dispatches a single call and routes on the Result it yields.
type CallStep struct {
	Pos     Position
	Comment string

	Call       *Call             // the target, arguments, and arms
	Input      *Value            // shapes the working value; or nil to pass through
	Output     *Value            // shapes the emitted value; or nil to pass through
	Assign     []Member          // captures into vars on successful exit
	Middleware []MiddlewareEntry // step-level stack around this dispatch
	Catch      []CatchClause     // failure edges, first match wins
	Next       Name              // the success edge
}

// GatherStep fans dispatches out concurrently and gathers the Results back
// in. It carries exactly one of its two forms: Over with Call (iterate), or
// Calls (scatter). Completion is the only path by which individual dispatch
// failures affect the step.
type GatherStep struct {
	Pos     Position
	Comment string

	Over  *OrExpr[[]*Value] // iterate form: one dispatch of Call per element
	Call  *Call             // the iterate form's call template
	Calls []*Call           // scatter form: one dispatch per entry

	Completion  *Completion     // success criterion; or nil (every dispatch must succeed)
	Concurrency *ConcurrencyCap // caps dispatches in flight; or nil (no cap)
	Output      *Value          // shapes the emitted value; or nil (succeeded values, in order)
	Assign      []Member        // captures into vars on successful exit
	Catch       []CatchClause   // failure edges, first match wins
	Next        Name            // the success edge
}

// ConcurrencyCap is a Gather's concurrency cap: the cap value and its
// position. A Gather without a cap carries no ConcurrencyCap at all: an
// absent field and an explicit null bind identically.
type ConcurrencyCap struct {
	Pos   Position
	Value int64
}

// Completion is a Gather's completion policy. Successes is evaluated once,
// after dispatches are enumerated, so the dispatch count is available to
// the expression.
type Completion struct {
	Pos Position

	Successes OrExpr[int64] // dispatches that must succeed; 1 means first success wins
	Wait      *bool         // let pending dispatches finish?; or nil (true)
}

// MatchStep routes to one of several successors by testing predicates
// against a single shaped value.
type MatchStep struct {
	Pos     Position
	Comment string

	Input   *Value       // shapes the value the predicates test; or nil to pass through
	Cases   []MatchCase  // tried in order, first match wins
	Default MatchDefault // the required fallback
}

// MatchCase is one cases clause: a predicate with the routing, shaping, and
// capture that apply when it is selected.
type MatchCase struct {
	Pos     Position
	Comment string

	When   Expr     // the predicate; always an expression
	Output *Value   // shapes the emitted value; or nil (the shaped match input)
	Assign []Member // captures into vars when selected
	Next   Name     // the successor this clause routes to
}

// MatchDefault is the required fallback clause: a MatchCase, with the same
// field semantics, minus the predicate.
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

	Output *Value   // shapes the emitted value; or nil to pass through
	Assign []Member // captures into vars
	Next   Name     // the successor step's name
}

// SleepStep pauses the frame for a duration or until an instant. Exactly
// one of For and Until is set; the bind pass guarantees the invariant.
type SleepStep struct {
	Pos     Position
	Comment string

	For   *OrExpr[string] // an ISO 8601 duration to sleep
	Until *OrExpr[string] // an RFC 3339 instant to sleep until
	Next  Name            // the successor step's name
}

// ReturnStep is the terminal success: the frame completes with a success
// Result carrying the shaped value.
type ReturnStep struct {
	Pos     Position
	Comment string

	Value *Value // shapes what the Result carries; or nil (the value received)
}

// RaiseStep is the terminal failure. With a Result, it constructs the
// failure; bare, it re-raises the failure being handled.
type RaiseStep struct {
	Pos     Position
	Comment string

	Result *RaiseResult // the failure to construct; or nil to re-raise
}

// RaiseResult holds the authorable fields of a constructed failure
// envelope. A Raise inherits nothing: unset fields are absent from the
// envelope.
type RaiseResult struct {
	Pos Position

	Type      *OrExpr[string] // a non-success Result type; or nil ("error")
	Code      OrExpr[string]  // the failure code; required
	Message   *OrExpr[string] // the human-readable message
	Details   *Value          // the structured payload
	Retryable *RetryableField // bool, explicit null, or expression
	Previous  *PreviousField  // overrides the failure-chain link
}

// Call is a call object: a request to run a target and obtain its Result.
// Exactly one of Provider and Flow is set; the bind pass guarantees the
// invariant.
type Call struct {
	Pos     Position
	Comment string

	Provider *ProviderRef // the provider target
	Flow     *FlowTarget  // the flow target

	Input     *Value            // the payload sent to the target; or nil to pass through
	With      *OrExpr[[]Member] // arguments, validated against the target's parameter schema
	OnSuccess *SuccessArm       // run when the Result settles as success
	OnFailure *FailureArm       // run on any non-success Result
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
	Name   *Name // references a subflow declared in an enclosing flows ledger
	Inline *Flow // a whole Flow written at the point of use
}

// SuccessArm is a call's onSuccess arm: shapes the value placed in the
// Call's success Result and captures variables while the dispatch's context
// is still alive.
type SuccessArm struct {
	Pos Position

	Value  *Value   // reshapes the returned value; or nil to pass through
	Assign []Member // captures into vars
}

// FailureArm is a call's onFailure arm: capture only, from the failed
// dispatch's live context — the failure envelope ascends exactly as
// produced.
type FailureArm struct {
	Pos Position

	Assign []Member // captures into vars
}

// CatchClause is a conditional failure edge: when the Step's failure
// matches, control transitions to the clause's Next.
type CatchClause struct {
	Pos     Position
	Comment string

	Match  FailureMatcher // which failures this clause takes
	Output *Value         // shapes the handler's value; or nil (the value the step received)
	Assign []Member       // captures into vars on the handler path
	Next   Name           // where handling continues
}

// FailureMatcher selects failures. Each present member constrains one
// contract field of the failure envelope, and every member present must
// match; the bind pass guarantees at least one member is present.
type FailureMatcher struct {
	Pos Position

	Codes     []StringLit // code patterns: "Prefix.Code", "Prefix.*", or "*"
	Types     []StringLit // non-success Result types to accept
	Retryable *bool       // the envelope's explicit retryable value
}

// MiddlewareEntry names a middleware by provider URI and configures its
// participation phase by phase. An absent phase is equivalent to an empty
// one.
type MiddlewareEntry struct {
	Pos     Position
	Comment string

	Provider  ProviderRef     // the middleware provider's URI
	OnEntry   *PhaseOnEntry   // on the way in
	OnSuccess *PhaseOnSuccess // on ascent, success
	OnFailure *PhaseOnFailure // on ascent, failure
	OnAlways  *PhaseOnAlways  // on every exit
}

// PhaseOnEntry configures the entry's onEntry phase, run on the way in.
type PhaseOnEntry struct {
	Pos Position

	When   *OrExpr[bool]     // gates the phase; or nil (always runs)
	With   *OrExpr[[]Member] // arguments, validated against the phase's contract
	Assign []Member          // captures against the phase's bindings
	Output *Value            // shapes the value passed inward; or nil to pass through
}

// PhaseOnSuccess configures the entry's onSuccess phase, run as a success
// Result ascends.
type PhaseOnSuccess struct {
	Pos Position

	When   *OrExpr[bool]     // gates the phase; or nil (always runs)
	With   *OrExpr[[]Member] // arguments, validated against the phase's contract
	Assign []Member          // captures against the phase's bindings
	Value  *Value            // reshapes the value emitted upward; or nil to pass through
}

// PhaseOnFailure configures the entry's onFailure phase, run as a failure
// Result ascends. The envelope fields carry the RaiseResult shapes;
// writing any of them constructs a new failure superseding the rising one:
// unwritten fields inherit from it, and it is chained as previous.
type PhaseOnFailure struct {
	Pos Position

	When   *OrExpr[bool]     // gates the phase; or nil (always runs)
	With   *OrExpr[[]Member] // arguments, validated against the phase's contract
	Assign []Member          // captures against the phase's bindings

	Type      *OrExpr[string] // a non-success Result type
	Code      *OrExpr[string] // the failure code
	Message   *OrExpr[string] // the human-readable message
	Details   *Value          // the structured payload
	Retryable *RetryableField // bool, explicit null, or expression
	Previous  *PreviousField  // overrides the failure-chain link
}

// PhaseOnAlways configures the entry's onAlways phase: cleanup run on every
// exit, even when the entry's other phases failed. It has no shaping key;
// the rising Result passes through unchanged.
type PhaseOnAlways struct {
	Pos Position

	When   *OrExpr[bool]     // gates the phase; or nil (always runs)
	With   *OrExpr[[]Member] // arguments, validated against the phase's contract
	Assign []Member          // captures against the phase's bindings
}
