---
title: "Data flow"
weight: 125
---

This section follows a value through a workflow, end to end: into a frame,
across its Steps, down and back up the middleware stack, over subflow and
`Gather` boundaries, and along the failure path. The values that move this way
are [the data plane](../concepts/#the-data-plane). Every rule composed here is
defined in an owning section, and each passage points to its owner; what this
section adds is the connected picture.

Three threads run through that picture. Every shaping default is a passthrough,
so a value moves unchanged until a field says otherwise. Nothing crosses a
boundary implicitly except a Result. And wherever a value would outlive the seam
that can see it, it is carried forward explicitly, captured into the frame's
variables with `assign`.

## Through a frame: input to Result

One round trip carries a value through a frame: in at the frame's creation, Step
to Step through the graph, and out as the frame's Result.

```
  the caller's input
          │
          ▼
     frame input                  set at creation, immutable
          │
          ▼
flow-level middleware             each entry's onEntry output
     (the descent)                reshapes the value downward
          │
          ▼
     entry Step                   within a Step: step.input →
          │                       input shaping → action →
          │                       step.result → output shaping
          │
          │ next                  the emitted value becomes
          ▼                       the successor's step.input
      … Steps …
          │
          ▼
    Return's value                the value the Result carries
          │
          ▼
flow-level middleware             each entry's onSuccess value
     (the ascent)                 reshapes the value upward
          │
          ▼
   the frame's Result
          │
          ▼
 the caller's call.result,
 or the platform's at the root
```

A frame is created with its input: the execution input, for the root frame; the
call's evaluated `input`, for a called Flow
([Where Flows appear](../flow-object/#where-flows-appear)). The input is set
once and never changes, so an expression anywhere in the frame can recover the
value the frame was given as `frame.input`
([`frame`](../execution-context/#frame)).

Between the frame and its first Step sits the Flow-level middleware stack. The
frame's input descends it, each entry's `onEntry` `output` reshaping the value
on the way down ([the next section](#through-the-middleware-stack)); what
emerges from the innermost entry is what the entry Step receives
([The frame lifecycle](../execution-model/#the-frame-lifecycle)).

Inside the graph, each transition hands exactly one value: the completed Step's
output becomes its successor's `step.input`, and that handoff is the only
data-plane path between Steps
([Frames and sequential execution](../execution-model/#frames-and-sequential-execution)).
Within a Step, the value crosses a fixed sequence of seams: `input` shapes what
the action consumes, the action produces the Step's Result, and `output` shapes
what the Step emits from it
([Data flow: `input` and `output`](../step-mechanics/#data-flow-input-and-output)).
Every field on this path has a passthrough default, the received value in and
the action's product out
([Field defaults and passthrough](../execution-context/#field-defaults-and-passthrough)),
so a Step reshapes the value only where its definition says so.

Two channels travel beside this path rather than on it, on
[the control plane](../concepts/#the-control-plane). Configuration rides `with`,
validated against the target's declared `parameters` and kept out of the payload
([The three axes](../call-interface/#the-three-axes-parameters-with-and-input)).
Capture rides `assign`, which writes values into the frame's variables for later
expressions to read back ([The `vars` model](../flow-object/#the-vars-model),
[Variables: `assign`](../step-mechanics/#variables-assign)). The handoff between
Steps carries one value and nothing else; a value needed two Steps from now
crosses by capture, not by any implicit channel.

The trip ends at a terminal Step. A `Return`'s `value` is the value the frame's
success Result carries
([How a Flow completes](../flow-object/#how-a-flow-completes),
[Success Result](../call-interface/#success-result)); the Result ascends the
Flow-level stack, and what the outermost entry emits is delivered to the parent
context—the caller's `call.result`, one of a `Gather`'s collected Results, or
the platform's, at the root
([The frame lifecycle](../execution-model/#the-frame-lifecycle)).

Because every default on the path passes through, the composition is transparent
end to end: a Flow whose fields shape nothing delivers its input to a bare
`Return` unchanged and yields it back as its Result's `value`. Data transforms
exactly where a definition says it does, and nowhere else.

## Through the middleware stack

A middleware stack is an onion around an operation—the Call a `Call` Step
dispatches, or a Flow's Step graph—with a defined value at every layer
([Where middleware attaches](../middleware-mechanics/#where-middleware-attaches),
[How values thread the stack](../middleware-mechanics/#how-values-thread-the-stack)).

On the way down, the operation input—the `Call` Step's shaped `input`, or the
frame's input—enters the outermost entry, and each entry's `onEntry` `output`
becomes the next inner entry's input. What the innermost entry emits is what the
operation receives: at a `Call` Step it arrives at the dispatch as `call.input`
([`call`](../execution-context/#call)); at the Flow level it is the value the
entry Step receives.

On the way up, the operation's Result ascends one position at a time. Where it
is a success, each entry's `onSuccess` `value` becomes the value the next outer
entry sees as its `middleware.result.value`
([`onSuccess`](../middleware-mechanics/#onsuccess)); what the outermost entry
emits is the value the Step's Result carries—or the frame's, at the Flow level.

Where the rising Result is a failure, the envelope ascends unchanged unless an
entry's `onFailure` block writes envelope fields, constructing a new failure:
the new one supersedes the rising one, and the engine chains what it displaced
as `previous` ([`onFailure`](../middleware-mechanics/#onfailure),
[Chaining](../execution-context/#chaining)). A failure is reshaped by
supersession, never by mutation; the original is intact one link down.

`onAlways` stands outside this traffic entirely: whatever its action and
expressions produce is discarded, and the in-flight Result continues to rise
unchanged—unless the phase itself fails, the one case where it supersedes
([`onAlways`](../middleware-mechanics/#onalways)).

What a middleware learns or measures stays at its position unless the author
carries it out: contributed metadata is readable only as `middleware.metadata`
in the entry's own phase blocks, and a phase's `assign` into `vars` is how any
of it outlives the entry
([Middleware-contributed metadata](../execution-context/#middleware-contributed-metadata)).
The stack obeys the same transparency rule as the Step path: every shaping
default passes through, so a stack whose blocks shape nothing leaves the data
plane untouched.

## Into and out of a subflow

A call that targets a Flow starts a new frame, and data crosses into it on
exactly two channels, both written at the call site: the call's `input` product
becomes the new frame's input, and its `with` arguments are validated against
the Flow's `parameters` and seed the new frame's variables
([The three axes](../call-interface/#the-three-axes-parameters-with-and-input),
[Where Flows appear](../flow-object/#where-flows-appear),
[The `vars` model](../flow-object/#the-vars-model)). Nothing else crosses:
frames are isolated, and the inner frame can read nothing of the frame that
dispatched it—not its variables, not its input
([`frame`](../execution-context/#frame)).

Coming back, one value crosses on its own: the Result. A `Return` arrives as a
success Result carrying its `value`; a `Raise`, or an unhandled failure, arrives
as a failure envelope; and the caller consumes either as `call.result`, exactly
as it would a provider's
([Flow-Call Result parity](../call-interface/#flow-call-result-parity)).

For everything else there is the window, and it is open briefly. In the call's
two arms, and nowhere else, the completed target stands exposed whole: a flow
target as `flow`, the completed frame with its `result`, `vars`, `input`, and
`metadata`; a provider target as `provider`, with its `input`, `result`, and
declared `metadata`
([The target windows](../call-interface/#the-target-windows-flow-and-provider),
[`flow`](../execution-context/#flow),
[`provider`](../execution-context/#provider)). The window is in scope in both
arms, and promotion out of it is explicit and immediate: the arm's `assign`
captures `flow.vars.<name>`, a metadata record, or whatever else later fields
need into the caller's variables, and nothing of the window outlives the call
object.

A failed subflow communicates through its failure Result plus whatever its
caller's failure arm captured at the boundary. Past the call object, the
envelope is the only carrier: an inner Flow that must surface more than its
caller captures places it in the envelope it raises
([The failure envelope](../call-interface/#the-failure-envelope)).

## Fan-out and fan-in: `Gather`

A `Gather` multiplies the call boundary. Every dispatch is one execution of a
`call` object, and the crossing rules above apply to each: the dispatch's
inbound payload arrives as its `call.input`—the element, in the iterate form;
the value the Step received, in the scatter form—with `call.index` carrying its
position ([The dispatch model](../step-actions/#the-dispatch-model)). A
flow-targeted dispatch is a subflow like any other: `input` and `with` cross,
nothing else does, and a dispatch whose Flow needs the element or the index is
given it explicitly.

While the fan-out is in flight, the frame's data plane holds still. A dispatch's
fields evaluate as it starts and only read, against the variable state at the
action's start; nothing writes until the fan-out completes, and no dispatch can
observe a sibling's writes, progress, or outcome
([Frames and sequential execution](../execution-model/#frames-and-sequential-execution)).

Fan-in happens in two motions. The arms run first, deferred to fan-out
completion, one dispatch at a time in dispatch order: each `onSuccess` `value`
shapes the value its dispatch's Result carries, and each arm's `assign` captures
into the frame's variables, in a fixed order, so cross-dispatch accumulation is
deterministic
([The arms at fan-out completion](../step-actions/#the-arms-at-fan-out-completion)).
Then the record is read: `step.results` holds one Result per dispatch, flat and
in dispatch order, every dispatch in its true position whatever its outcome,
complete by the time any expression can read it
([The collected Results](../step-actions/#the-collected-results-stepresults)).

### The projection idiom

`step.results` comes with no machinery: no success list beside it, no failure
list, no per-outcome counts. It needs none: because the record is flat, uniform,
and position-faithful, every consumer is a one-line expression over it, a
projection. The canonical one is the success projection,

```
{{ step.results.filter(r, r.type == 'success').map(r, r.value) }}
```

the succeeded dispatches' values, in dispatch order. It is the `Gather`'s
`output` default ([Step actions](../step-actions/#gather)): a fan-out that
shapes nothing emits exactly its dispatches' values. Its dual collects the
failures,

```
{{ step.results.filter(r, r.type != 'success') }}
```

and the two are the model for whatever else the fan-in must mean:

- The whole record, one slot per dispatch, success or failure, positionally
  aligned with the input: `{{ step.results }}`.
- Each element's value where it succeeded and a placeholder where it did not:
  `{{ step.results.map(r, r.type == 'success' ? r.value : null) }}`.
- The codes that failed:
  `{{ step.results.filter(r, r.type != 'success').map(r, r.code) }}`.
- A count by type: `{{ size(step.results.filter(r, r.type == 'error')) }}`.

There is no engine-provided projection: the record is kept flat, uniform, and
position-faithful precisely so that the projection an author writes is the whole
interface. Shaping the fan-in is writing the expression that says what the
fan-out meant.

A dispatch's failure is data here, not an event: it fills its slot in the record
like any Result, it never sets the frame's failure context, and the `Gather`'s
`catch` never matches it
([Failures and `catch`](../step-actions/#failures-and-catch),
[Lifecycle](../execution-context/#lifecycle)). Dispatch failures reach beyond
the frame only when the completion policy fails: the `Gather`'s own
`System.GatherCompletionUnmet` failure carries the fan-out's non-success Results
in its `details`, each entry pairing the dispatch's `index` with its `result`
([`System.GatherCompletionUnmet`](../step-actions/#systemgathercompletionunmet)).
The duplication with `step.results` is deliberate, and it is a data-flow fact:
an envelope propagates to parent frames, and a failed Step's `step.results` does
not—the evidence travels in the value that crosses.

## On the failure path

When an outcome is not success, what moves is the failure envelope: an ordinary
structured value—`type`, `code`, `message`, `details`, `retryable`,
`previous`—carried by the same machinery that carries any value
([The failure envelope](../call-interface/#the-failure-envelope)). Routing keys
on its `code`, expressions read its members, and it propagates whole.

The first seam a failure crosses is the call's failure arm, and it is the last
place the failed dispatch's context is alive. There the envelope is read as
`call.result`, and the arm's `assign` captures what would otherwise vanish with
the call execution—a provider's `metadata`, the call's timing record, a failed
frame's window
([`onFailure`: capture only](../call-interface/#onfailure-capture-only)).

From there the envelope ascends the middleware stack, transformed only by
supersession ([above](#through-the-middleware-stack)), and the Step resolves to
it. A failed Step shapes nothing: its `output` and `assign` belong to the
success exit, and what runs instead is `catch`, matching the outermost emitted
Result ([The failure exit](../execution-model/#the-failure-exit),
[Where `catch` sits](../step-mechanics/#where-catch-sits)). The matched clause
is the failure edge's only shaping: its `output` produces the value the handler
Step receives, defaulting to the value the failed Step received, since no
success value exists, and its `assign` captures
([`catch` clauses](../step-mechanics/#catch-clauses)).

Along the handler path the envelope stays in reach as the `failure` binding,
frame-wide, until the first successful Step completion clears it; a handler that
needs the failure beyond that point captures what it needs into `vars` first
([`failure`](../execution-context/#failure),
[Lifecycle](../execution-context/#lifecycle)).

The envelope also carries its own history. Whenever one failure supersedes
another—a handler Step fails during recovery, an `onFailure` block or a `Raise`
constructs a successor, a cleanup fails with a failure in flight, a cancellation
is imposed over the work it interrupts—the engine links the superseded failure
as the new one's `previous`, and the chain rides the envelope wherever it goes
([Chaining](../execution-context/#chaining)).

A failure no clause matches leaves the frame as its Result, and there it is
ordinary data: the calling Step's `catch` matches it like any failure Result,
per the scoping rule—what cannot be caught within a frame is plain data one
level up ([The scoping rule](../execution-model/#the-scoping-rule),
[`catch` and frame-level failures](../step-mechanics/#catch-and-frame-level-failures)).
What must cross frames travels in the envelope its producer writes, the same
rule `System.GatherCompletionUnmet` instantiates with its collected failures.

One exit moves no data at all. An interrupted scope is torn down, not completed:
on the unwind no arm runs, no `catch` is consulted, and no shaping evaluates;
established entries' `onAlways` phases run, and the Result that emerges is the
unwind's to determine ([The unwind](../execution-model/#the-unwind)). Where data
flows, it flows by the rules above; where execution is interrupted, the data
plane simply stops.
