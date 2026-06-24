---
title: "Execution model"
weight: 90
---

This section defines the runtime semantics the rest of the specification
assumes: how a frame executes from entry to completion, how Steps execute within
it, when the expressions in a definition are evaluated, and how an execution is
interrupted. An implementation is free in its internal architecture, but it MUST
preserve the rules in this section as observable behavior.

## The completion contract

Every frame completes exactly once, producing exactly one Result. There is no
partial completion: a frame that has started eventually reaches exactly one
Result, and a frame that has completed never produces another.

The Result itself is defined in
[The Call interface and Result](../call-interface/): a discriminated union on
`type`, with one success type and four non-success types sharing the failure
envelope. This section adds the frame-level guarantee. Whatever path execution
takes through a frame (a terminal Step, an unhandled failure, an interruption
imposed from outside), the outcome is expressed as a single Result of one of
those types.

The root frame and every called frame share the contract, which is what makes a
Flow callable like a provider: a calling Step consumes a subflow's Result
exactly as it consumes a provider's
([Flow-Call Result parity](../call-interface/#flow-call-result-parity)), a
`Gather` Step observes each dispatch's Result, and the root frame's Result is
delivered to the platform that started the execution.

The frame's contract is one instance of a general rule over **target
executions** — the units a call runs, frames and provider executions alike,
defined under
[Frames and sequential execution](#frames-and-sequential-execution): every
target execution completes exactly once, with exactly one Result. A frame
completes by this section's rules; a provider execution completes when the
platform accepts its Result. Flow-Call Result parity is the rule seen from the
consumer: either kind of target execution yields the same kind of Result to the
call that ran it.

## Frames and sequential execution

A Flow executes as a frame: an execution-time instantiation of the Flow, with
its own variables and its own pass through the lifecycle below. Where Flows
appear, and what supplies a frame's input in each context, are defined in
[The Flow object](../flow-object/#where-flows-appear). Every frame except the
root has a parent context: the Call that targeted its Flow — a `Call` Step's
call, or one of a `Gather`'s dispatches.

Within a frame, exactly one Step executes at a time, and a Step completes before
its successor begins; there is no concurrent or asynchronous Step execution
within a single frame. Each transition hands one value to the Step it enters: on
success, the completed Step's output; on a failure edge, the matched `catch`
clause's output ([Failures and `catch`](../step-mechanics/#failures-and-catch)).
That handoff is the only [data-plane](../concepts/#the-data-plane) path between
Steps; what crosses Steps otherwise travels on
[the control plane](../concepts/#the-control-plane), captured into `vars` by
`assign` and read back by later expressions
([The `vars` model](../flow-object/#the-vars-model)).

A call runs its target as a **target execution**: a frame, for a flow target, or
a **provider execution**, for a provider target — the provider's single opaque
unit of work, with a Result and no expression-visible interior
([The call object](../call-interface/#the-call-object)). Within a frame, no
workflow work runs concurrently; all concurrency is between the target
executions a Step has outstanding. A `Call` Step has at most one outstanding; a
`Gather` has up to its whole fan-out in flight at once, each dispatch running
its target concurrently with its siblings ([Step actions](../step-actions/)). (A
control action racing the scope it wraps, such as a timeout, runs no workflow
work; see
[Middleware mechanics](../middleware-mechanics/#author-shaping-and-the-middleware-action).)
An execution is therefore a tree: the internal nodes are frames, serial
evaluators that dispatch, and the leaves are provider executions.

Concurrency between target executions does not make the frame's own evaluation
concurrent: a frame evaluates on a single serial thread, and a frame's variables
change only on that thread, one block at a time (the block discipline of
[Variables: `assign`](../step-mechanics/#variables-assign)). While a `Gather`'s
fan-out is in flight, no write-capable evaluation runs in the frame at all. A
dispatch's call fields evaluate as the dispatch starts, but they only read, and
what they read is the variable state at the action's start: every dispatch sees
the same state, however the concurrency limit staggers the starts. Everything
write-capable — the calls' arms, with their `assign` — is deferred to the
fan-out's completion, where the arms evaluate one at a time, in dispatch order
([Step actions](../step-actions/)). Variables mutate at the frame's serial
points or not at all: there is no mid-action interleaving, and no interleaving
order to be sensitive to.

## The frame lifecycle

Running a Flow means creating a frame from it and executing that frame through
six phases.

1. **Creation.** The frame is created with its input value, and its entry
   instant is recorded in its metadata
   ([Execution context](../execution-context/)).

2. **Variable initialization.** Caller-supplied arguments are validated against
   the Flow's `parameters` schema, and the validated values together with
   schema-declared defaults seed the frame's `vars`
   ([The `vars` model](../flow-object/#the-vars-model)). Variables are available
   to every expression evaluated in the frame from this point on, including
   those in Flow-level middleware phase blocks. On validation failure, the
   frame's Result is `System.ParameterValidationFailed`
   ([definition](../flow-object/#systemparametervalidationfailed)) and the frame
   proceeds directly to completion: no middleware entry is established and the
   Step graph does not run.

3. **Descent.** The Flow-level `middleware` stack is descended: each entry's
   `onEntry` phase runs in array order, outermost first, establishing the entry.
   The frame's input threads down the stack, and the value emerging from the
   innermost entry is delivered to the Step graph. Phase semantics, threading,
   and establishment are defined in
   [Middleware mechanics](../middleware-mechanics/).

4. **Step-graph execution.** The graph runs from `entrypoint` toward terminal
   completion, one Step at a time
   ([Frames and sequential execution](#frames-and-sequential-execution)). It
   completes by one of the three in-definition endings of
   [How a Flow completes](../flow-object/#how-a-flow-completes): a `Return`, a
   `Raise`, or an unhandled failure propagating out. A concurrent control
   action, such as a Flow-level timeout, races this phase and may interrupt it
   ([Cancellation](#cancellation)).

5. **Ascent.** The Step graph's Result ascends the stack in reverse array order.
   At each established entry, `onSuccess` or `onFailure` runs, selected by the
   rising Result, followed by `onAlways`; a control action may substitute a
   Result or re-run its inner scope. The frame's Result is not final until the
   outermost entry emits ([Middleware mechanics](../middleware-mechanics/)).

6. **Completion.** The frame's exit instant is recorded, and its Result is
   delivered to the parent context: the calling Step consumes it as its Call's
   Result (for a flow-targeted Call, the completed frame is also exposed to the
   call's arms as `flow`; see
   [The target windows](../call-interface/#the-target-windows-flow-and-provider)),
   a `Gather` Step observes it among its dispatches' Results, and for the root
   frame the execution ends. How a platform surfaces a root Result, as terminal
   states or alerting, is a platform mapping over Result types
   ([Result types](../call-interface/#result-types)).

## The scoping rule

A frame nests its machinery: the frame contains its middleware stack, the stack
wraps the Step graph, and Steps run inside the graph.

```
┌─ frame ─────────────────────────────────────────────┐
│  vars                                               │
│                                                     │
│  ┌─ middleware stack ──────────────────────────┐    │
│  │                                             │    │
│  │   input  ──→  outer → … → inner             │    │
│  │                                             │    │
│  │   ┌─ Step graph ──────────────────────┐     │    │
│  │   │  catch routing lives here         │     │    │
│  │   │  ┌──────┐   ┌──────┐              │     │    │
│  │   │  │ Step │ → │ Step │ → …          │     │    │
│  │   │  └──────┘   └──────┘              │     │    │
│  │   └───────────────────────────────────┘     │    │
│  │                                             │    │
│  │   Result ←──  outer ← … ← inner             │    │
│  │                                             │    │
│  └─────────────────────────────────────────────┘    │
│                                                     │
│  ▼ the Result propagates to the parent frame        │
└─────────────────────────────────────────────────────┘
```

Failure handling reaches exactly as far as its position in this nesting allows.
That is the **scoping rule**: an inner construct cannot intercept an outcome
determined by an outer one.

A Step's `catch` clauses live inside the Step graph, strictly inside the
middleware round trip. A failure produced on the ascent—by a Flow-level
middleware phase or by an interruption—comes into being past the point where any
`catch` could have run, so no `catch` in that frame can match it
([`catch` and frame-level failures](../step-mechanics/#catch-and-frame-level-failures)).
Such a failure becomes the frame's Result and propagates to the parent context,
where it is an ordinary failure Result, matched by the calling Step's `catch`
like any other ([Failures and `catch`](../step-mechanics/#failures-and-catch)).
What cannot be caught within a frame is plain data one level up.

The same rule, read one level down, is why a `Call` Step's middleware failures
are catchable: its Step-level stack wraps the single dispatch inside the Step
graph, so the Result the outermost entry emits is the Step's Result, matched by
that Step's `catch` from outside the stack
([Where `catch` sits](../step-mechanics/#where-catch-sits)). A `Gather`'s
dispatches carry no Step-level stack: where a dispatch targets a Flow, that
Flow's own flow-level stack wraps its graph inside the dispatch's frame, and
what its outermost entry emits is the frame's Result — that dispatch's Result,
data the `Gather` observes, never something its `catch` matches
([Step actions](../step-actions/)). There is no special case for frames; there
is one nesting, and each construct handles only what arises within its reach.

## The Step lifecycle

A Step executes through a fixed sequence of phases. Not every phase applies to
every action: the table marks applicability, and each action's reference gives
its complete account ([Step actions](../step-actions/)).

| Phase             | `Call` | `Gather` | `Match` | `Pass` | `Sleep` | `Return` | `Raise` |
| ----------------- | ------ | -------- | ------- | ------ | ------- | -------- | ------- |
| 1. Entry          | ✓      | ✓        | ✓       | ✓      | ✓       | ✓        | ✓       |
| 2. Input shaping  | ✓      | —        | ✓       | —      | —       | —        | —       |
| 3. Action         | ✓      | ✓        | ✓       | —      | ✓       | ✓        | ✓       |
| 4. Result         | ✓      | ✓        | —       | —      | —       | —        | —       |
| 5. Output shaping | ✓      | ✓        | (c)     | ✓      | —       | —        | —       |
| 6. Assignment     | ✓      | ✓        | (c)     | ✓      | —       | —        | —       |
| 7. Transition     | ✓      | ✓        | (c)     | ✓      | ✓       | ✓        | ✓       |

(c): delegated to the matched clause.

1. **Entry.** The Step begins executing. `step.input` is set to the value the
   Step received: the previous Step's output (on a failure edge, the matched
   `catch` clause's output) or, for the entry Step, the value the frame's
   descent delivered. The Step's entry instant is recorded; that instant is the
   pin read by the clock functions ([The clock pin](#the-clock-pin)).

2. **Input shaping.** On actions that accept it, `input` is evaluated to produce
   the value the action consumes
   ([Data flow: `input` and `output`](../step-mechanics/#data-flow-input-and-output)).
   A `Call`'s shaped input is what enters its middleware stack and its Call; a
   `Match`'s is the value its predicates test.

3. **Action execution.** The action runs: a `Call` descends its Step-level
   middleware stack, dispatches its Call, and ascends; a `Gather` makes its
   dispatches and collects their Results; a `Match` selects a clause; a `Sleep`
   waits; a `Return` or `Raise` constructs the Result that completes the frame.
   `Pass` performs no action work: it exists for the shaping phases around it.
   Each action is defined in [Step actions](../step-actions/).

4. **Result.** The action's product becomes readable: `step.result` holds a
   `Call` Step's Result (the Result the outermost entry of its middleware stack
   emitted), and `step.results` holds a `Gather` Step's collected Results.
   Binding shapes are defined in [Execution context](../execution-context/).

5. **Output shaping.** On success, `output` is evaluated to produce the value
   the Step emits
   ([Data flow: `input` and `output`](../step-mechanics/#data-flow-input-and-output)).
   Each action defines its own default, given with the actions in
   [Step actions](../step-actions/): a `Call`'s `output` reads the value its
   Result carries, a `Gather`'s projects its collected Results' successes, a
   `Pass`'s reads its input. A `Match` delegates output to the matched clause.

6. **Assignment.** On success, `assign` captures values into the frame's `vars`,
   after `output`, under the ordering defined in
   [Variables: `assign`](../step-mechanics/#variables-assign). A `Match`
   delegates assignment to the matched clause.

7. **Transition.** The Step completes and control moves on. A transitioning Step
   passes control to its `next` (for a `Match`, the matched clause's `next`); a
   terminal Step completes the frame
   ([How a Flow completes](../flow-object/#how-a-flow-completes)).

The Step's **exit instant** — the `exitedAt` of its metadata record
([Execution context](../execution-context/)) — is recorded when the Step's work
settles: at the Result phase on the actions that have one, otherwise when the
action completes (immediately, for a `Pass`). What follows is the Step's tail:
output shaping, assignment, and transition run after the exit instant, reading a
Step whose work is complete, which is why `step.metadata.exitedAt` is readable
in them while `step` remains in scope.

### The failure exit

A Step whose Result is a failure skips output shaping and assignment: only a
successful exit shapes and captures. The failure is instead matched against the
Step's `catch` clauses, on the actions that carry them: a matching clause routes
control onward, running only its own `output` and `assign`, and an unmatched
failure propagates out of the frame
([Failures and `catch`](../step-mechanics/#failures-and-catch)). A failure
settles the Step's work the way a success does: the exit instant is recorded
before `catch` is consulted, so a clause's expressions read the completed Step's
record.

A Step interrupted from outside takes neither exit. Interruption is not a Step
outcome but a frame-level one: the Step's remaining phases are abandoned and its
`catch` is never consulted, per the scoping rule
([Cancellation](#cancellation)).

## Expression evaluation timing

The lifecycles above say when constructs run; the rules here say how often the
expressions inside them are evaluated. Every expression-valued field belongs to
one **construct**, the unit whose lifecycle evaluates it. The constructs are the
Step, the middleware phase, and the Call, and one run of a construct is a
**construct execution**: a single pass of a Step through its lifecycle, a single
run of one middleware phase, or a single execution of a Call — one dispatch of
its `call` object, a `Call` Step's or a `Gather`'s alike. A Step's shared fields
belong to the Step, a phase block's fields to its phase, and a `call` object's
fields — its core fields and its arms — to the Call. A clause's fields (a
`Match` clause's, a `catch` clause's) belong to the Step that carries the
clause.

An implementation MUST evaluate each expression-valued field exactly once per
execution of its containing construct. A Step's `input` is evaluated once when
its input-shaping phase runs; a phase block's `with` once when its phase runs; a
`call` object's `input` once per Call execution; a `catch` clause's `output`
once when that clause matches. A field whose phase does not run is not evaluated
at all, and no evaluation is repeated within one construct execution.

### Re-execution evaluates afresh

When a construct re-runs its inner scope—a retrying middleware re-running on a
matching failure, an iterating one re-running while its continuation holds—
every expression inside the re-run scope is freshly evaluated: each new
execution evaluates its fields against the current bindings, and no value
carries over from a prior run. Re-entry is exactly that: the re-run scope's
constructs execute anew. A step-level retry re-runs the Call, so each attempt is
a new Call execution evaluating the `call` object's fields afresh, while the
enclosing Step's own pass — its shared fields already evaluated — continues; a
flow-level re-run creates fresh Step executions outright. The re-running entry's
own `onEntry` is not part of its inner scope: it evaluates exactly once, at
first entry, and its captured configuration persists across every re-run
([Re-execution and re-entry](../middleware-mechanics/#re-execution-and-re-entry)).

### Nondeterministic sources

Fresh evaluation means a nondeterministic source is read anew each time: an
expression that draws on one need not reproduce its earlier value when its
construct re-executes. A workflow that needs a value held stable across
re-executions MUST capture it into `vars` with `assign` on first evaluation and
reference the variable thereafter. This rule is what keeps nondeterministic work
on the provider side of the expression / provider boundary
([Expressions](../expressions/#the-expression-provider-boundary)); the clock
functions below are the sanctioned exception, and a clock-derived value that
must stay stable obeys the same capture rule.

### The clock pin

Each construct execution has a single entry instant, recorded when it begins.
That instant is the execution's **clock pin**: the expression profile's `now()`
function reads it, and within one construct execution, every evaluation of
`now()` MUST return that execution's entry instant. Two `now()` calls in one
expression agree, and `now()` in a Step's `assign` matches `now()` in the same
Step's `output`.

Every construct is a tracked context, so the pin is also exposed as data: the
pin and the `enteredAt` of the construct's metadata record are the same instant
([Execution context](../execution-context/)) — a Step's in `step.metadata`, a
phase's in `middleware.metadata`, a Call's in `call.metadata`.

The pin is per construct execution, so re-execution pins afresh: a retry attempt
or a loop iteration re-executes the constructs in its scope, and each new
execution carries a new entry instant. `now()` is therefore stable within an
attempt and advances across attempts — it exists for determinism within a
construct execution, not across executions. An expression that needs an instant
stable across attempts reads it from an enclosing context's record instead:
inside a retried Call, `step.metadata.enteredAt` names the Step's entry, attempt
after attempt, where `now()` names each attempt's own. A value derived some
other way that must stay fixed is captured into `vars`, per the rule above.
`wallTime()` is never pinned: it reads the clock afresh at every evaluation.
Both functions are defined in [MWL functions](../expressions/#mwl-functions).

## Cancellation

Cancellation is the only execution signal the language defines: a directive,
originating outside the running scope, to stop its execution. It may come from
outside the execution entirely—a user, an operator, the platform—or from a
construct within the execution that governs other work: a `Gather` cancelling
dispatches it no longer needs, or a control action such as a timeout preempting
the scope it wraps
([Middleware mechanics](../middleware-mechanics/#author-shaping-and-the-middleware-action)).
However it originates, an interruption stops work that has not completed; the
rules below govern what still runs and what Result emerges.

### The unwind

An interrupted scope is torn down by ascending out of it. The engine stops the
current work, abandoning the executing Step's remaining phases, and exits every
construct between the point of execution and the interrupting boundary. On the
way out, every established middleware entry's `onAlways` phase runs, exactly
once, innermost outward through every established stack in the scope: the
executing `Call` Step's step-level entries, then the Flow-level entries, when a
whole frame is torn down. Nothing else participates: no `onSuccess` or
`onFailure` phase runs, no call arm runs, no `catch` clause is consulted, and no
Step routing occurs. The unwind is teardown, not data flow; `onAlways`'s
guarantee and its supersession rule are defined in
[Middleware mechanics](../middleware-mechanics/#onalways).

Interrupting a scope also interrupts the target executions it awaits. When the
interrupted scope contains a Step with target executions outstanding — a `Call`
Step running a subflow, a `Gather` with dispatches in flight — the interruption
extends into each of them: a frame is torn down under these same rules, its
established entries running their `onAlways` phases, and an outstanding provider
execution is cancelled through the platform. Each resolves a Result, per
[the completion contract](#the-completion-contract), before the unwinding
continues in its parent, and the Result is the cancellation in flight: a frame's
teardown may supersede the chain's head with a cleanup failure before it
completes, where a provider execution — an opaque leaf, with no seams of its own
— records the chain as imposed. Nothing routes on these Results; the scope that
awaited them is itself unwinding. The interruption takes only work not yet
committed: a target execution whose Result the platform has already accepted —
the `acceptedAt` instant of its call's record
([`Call` metadata](../step-actions/#call-metadata)) — resolves as that Result,
untouched. The unwind thus proceeds innermost outward across execution
boundaries exactly as within a frame, and no target execution is left running
after the scope that awaited it has been torn down.

The Result in flight during the unwind is a cancellation: a non-success Result
of type `cancellation` ([Result types](../call-interface/#result-types)). An
interrupting construct inside the execution MUST first construct its explanatory
failure—the Result that names why it interrupted—and impose a cancellation that
chains that explanation as its `previous`. A timeout's unwind therefore carries
`System.Cancelled` with the timeout's own failure beneath it. An `onAlways`
phase running during the unwind sees this chain as its in-flight Result, the
cancellation at its head and the cause beneath it; a cleanup that itself fails
supersedes the chain, and the engine links the superseded chain via `previous`,
per the supersession rule.

An interruption arriving at a scope that is already unwinding does not restart
the teardown; established entries' `onAlways` phases still run exactly once.

### The conversion seam

The construct that owns an interruption is the single seam where the unwind's
outcome re-enters normal execution. When the unwind reaches the owner's
position, what happens next is decided by one inspection of the arriving Result.

If the Result is the owner's own cancellation, unchanged, the owner MUST convert
it: the cancellation is removed, and its `previous`, the explanatory failure
attached at imposition, is emitted in its place. That failure ascends the
forward path from the owner's position like any rising failure: outer
`onFailure` phases see it, and when it becomes a Step's or a frame's Result,
`catch` can match it. A timeout surfaces as a failure of type `timeout`, not as
a cancellation. On this path the cancellation leaves no trace; it was mechanism,
not history.

If the Result is anything else, a cleanup failure superseded the cancellation
during the unwind, and the owner MUST NOT convert it. The chain ascends as-is:
the cleanup failure at its head, the cancellation and the explanation beneath
it—`error`, then `cancellation`, then `timeout`, reading down the chain. A
failed cleanup is never hidden behind the friendlier converted code, where a
`catch` could match the conversion and route past it; it stays at the head of
the chain, with the full account of what happened beneath it.

Together the two cases keep cancellation off the data plane inside the scope it
tears down: within that scope, only `onAlways` can observe the cancellation, and
what emerges from the seam is always an ordinary failure. A Result of type
`cancellation` becomes visible to routing only as a completed target execution's
Result, in the scope that ran it: the external case below, and the `Gather`
case, where a cancelled dispatch's Result is observed by the `Gather`
([Step actions](../step-actions/)).

One owner performs no conversion: a `Gather` cancelling its own dispatches —
under `wait: false`, or resolving the fan-out before its own failure
([Step actions](../step-actions/)). Its imposition never re-enters its forward
path as a rising failure; it resolves each interrupted dispatch as data. The
dispatch unwinds under the rules above — a flow-targeted dispatch's frame is
torn down, its established entries running `onAlways` only; a provider-targeted
dispatch is cancelled through the platform; no arm of its call runs — and
resolves `System.GatherDispatchCancelled`, one more Result among the fan-out's
outcomes. There is no seam because nothing ascends: the `Gather` consumes the
Result it imposed.

The supersession rule still applies inside that unwind, in miniature: a cleanup
failure arising in a flow-targeted dispatch's teardown supersedes the
cancellation in flight, and the superseded chain is that dispatch's Result — an
`error` whose `previous` reads down to the imposed cancellation — counted by the
`Gather`'s policy like any other non-success.

The boundary between an interrupted dispatch and a settled one is acceptance,
the unwind's general boundary ([The unwind](#the-unwind)): a dispatch whose
Result the platform has already accepted — its call record's `acceptedAt`
([Step actions](../step-actions/#call-metadata)) — resolves as that Result,
untouched; the imposition takes only the dispatches not yet accepted.

### External cancellation

A cancellation directed at a frame from outside the execution—a user, an
operator, a platform deadline, quota, or other operational trigger—has no owner
within the execution and so no conversion seam. The frame unwinds as above, and
the bare cancellation becomes the frame's Result: type `cancellation`, code
`System.Cancelled`, no `previous`. The language treats every external
cancellation uniformly regardless of its operational reason; the reason MAY be
surfaced through the Result's `message` or `details`.

In the cancelled frame's parent, that Result is an ordinary failure Result: the
calling Step's `catch` can match `System.Cancelled` and route on it, per the
scoping rule. When a `Gather` cancels dispatches it no longer needs, each
cancelled dispatch resolves the same way under the code
`System.GatherDispatchCancelled`, and the `Gather` observes those Results among
its dispatches' outcomes ([Step actions](../step-actions/)).

A timeout is not a cancellation. A timeout middleware uses cancellation
internally, to tear down the scope it preempts, but what its seam emits is a
failure of type `timeout` (under its catalog code,
`Provider.Middleware.Timeout.Exceeded`; see [Providers](../providers/)). The two
outcomes are distinguishable wherever failures are matched, by `type` and by
`code`: a `catch` clause can match one without the other.

All other external interaction—events, human decisions, external data—reaches a
workflow through `Call` Steps and providers ([Providers](../providers/)).
Cancellation is the language's only signal mechanism, and the only outcome a
frame can be given from outside its own definition.
