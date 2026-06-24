---
title: "Concepts"
weight: 20
---

The Metolia Workflow Language (MWL) describes workflows as JSON documents:
directed graphs of named Steps that call services, branch, run work
concurrently, wait, and handle failures.

This section introduces the model: the smallest complete workflow, the execution
loop, and the concepts the rest of the reference specifies.

## A minimal Flow

A workflow definition is a JSON document whose root object is a Flow. The
following definition is complete and runnable: a Flow of two Steps, one that
calls an HTTP service and one that ends the workflow.

```json
{
  "$schema": "https://mwl.dev/v0.1/flow/schema.json",
  "entrypoint": "greet",
  "steps": {
    "greet": {
      "action": "Call",
      "call": {
        "provider": "mwl:provider.call/example/http/v1",
        "with": { "method": "GET", "path": "/hello" }
      },
      "next": "done"
    },
    "done": { "action": "Return" }
  }
}
```

Execution enters at the Step named by `entrypoint`. Here `greet` runs first: it
dispatches its Call to an HTTP provider and, on success, follows its `next` to
`done`. `done` is terminal: `Return` completes the Flow, which produces a Result
recording how it ended. That is the whole loop: run the current Step; if it
transitions, follow `next` and repeat; if it is terminal, the Flow completes
with its Result.

## Flows and frames

A Flow is the unit of definition: an `entrypoint`, a map of `steps`, and
optionally named subflows (`flows`), parameters, and middleware. The same Flow
object describes a whole workflow or a piece of one that a larger Flow composes.
When a Flow runs, it runs inside a frame, an execution-time instantiation of the
Flow with its own variables and lifecycle; one Flow definition may be running as
many frames at once. A frame evaluates one thing at a time: all concurrency is
between the target executions a Step has outstanding — the subflow frames and
provider executions its Calls run. The Flow object and its fields are specified
in [The Flow object](../flow-object/); the frame lifecycle in
[Execution model](../execution-model/).

## Steps and actions

A Step is a named node in a Flow's graph. Each Step does one thing, named by its
`action`: `Call` dispatches a Call, `Gather` runs many Calls concurrently,
`Match` branches, `Pass` reshapes data without calling anything, `Sleep` waits,
and `Return` and `Raise` end the Flow. Around the action, Steps share a small
uniform field set for shaping data, capturing variables, and routing. The shared
fields and the Step lifecycle are specified in
[Steps and step mechanics](../step-mechanics/); each action, with its complete
field set, in [Step actions](../step-actions/).

## Routing and terminal Steps

A Step either transitions or terminates. A transitioning Step names its
successor in `next`, and control passes there on success; failures on Steps that
dispatch Calls route separately, through `catch` clauses — the success fields
and `catch` forming the two arms of a match on the Step's Result. A terminal
Step ends the Flow: `Return` completes it successfully, and `Raise` completes it
with a failure. Routing, `catch`, and the terminal mechanics are specified in
[Steps and step mechanics](../step-mechanics/).

## Calls and Results

A Call is the dispatch unit: it names a target, a provider or a Flow, gives it
arguments (`with`) and a data payload (`input`), and yields a Result. A Result
is a discriminated value recording one outcome: a success carrying a `value`, or
a non-success carrying a structured failure envelope. Every Call produces
exactly one Result, and so does every frame, which is what lets a Flow be called
like any provider. The `call` object and the Result are specified in
[The Call interface and Result](../call-interface/).

## The data plane

A Flow moves data between its Steps. When a Flow runs, its caller supplies the
frame's input; the entry Step receives a value, does its work, and emits an
output. Each transition hands one Step's output to its successor as input, and
the value a terminal `Return` emits is what the Flow's success Result carries.

The values that move this way—Step inputs and outputs, the `value` a Result
carries, what an expression evaluates to—are collectively **the data plane**.
Configuration travels beside the data plane, not through it, on
[the control plane](#the-control-plane): arguments supplied as `with` are
validated against a target's declared `parameters`, while the data payload flows
on `input`.

The data plane carries JSON values, typed according to the rules specified in
[The data model](../data-model/). A Step's `input` and `output` shaping is
specified in [Steps and step mechanics](../step-mechanics/), and the Result and
the `with`/`input` channels in
[The Call interface and Result](../call-interface/). How data moves end to end,
across Steps, middleware, and subflows, is synthesized in
[Data flow](../data-flow/).

## The control plane

Beside the data plane sits the state that steers a workflow rather than flowing
through it: **the control plane**. Configuration enters through a target's
declared `parameters`, supplied as `with` arguments at the call site; it seeds
the frame's variable namespace, `vars`, which a Step's `assign` writes as
execution proceeds. A subflow never shares its caller's variables; values cross
the boundary only explicitly. The engine contributes execution state: frame and
Step metadata and the failure context, exposed to expressions as the execution
context. The variable model is specified in [The Flow object](../flow-object/);
`assign` timing in [Steps and step mechanics](../step-mechanics/); the execution
context in [Execution context](../execution-context/).

## Expressions

Definitions stay declarative by embedding expressions where values need
computing: a string field consisting of exactly one `{{ ... }}` expression
evaluates to the expression's typed result. Expressions read a small set of
bindings, such as `vars` and `step`, that expose the running workflow's state,
and every expression produces a data-model value. The embedding, the binding
roots, and the expression-language profile are specified in
[Expressions](../expressions/).

## Providers

A provider is a platform-supplied capability addressed by URI; providers are
MWL's extension surface. A Call's target may be a provider
(`mwl:provider.call/example/http/v1`), and middleware are providers too
(`mwl:provider.middleware/mwl/retry/v1`): the language defines the shapes, and
providers supply the behavior. A provider declares the `parameters` schema its
arguments are validated against and a catalog of the failure codes it can
produce. Providers, their URI namespacing, and their catalogs are specified in
[Providers](../providers/).

## Middleware

Middleware wraps work without changing it: an ordered stack of middleware
providers around a `Call` Step's dispatch, or around a whole Flow's Step graph,
acting on entry, on success, on failure, or always. Cross-cutting behavior such
as retry, caching, and notification lives here rather than in the graph itself.
The phase model and composition are specified in
[Middleware mechanics](../middleware-mechanics/); the middleware catalog in
[Providers](../providers/).
