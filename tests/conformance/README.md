# Conformance cases

Executable example workflows that serve two purposes at once: reference material
for MWL's key workflow patterns, and conformance test cases for implementations.
Every case uses only spec-defined providers — the `mock` call provider and the
`Retry`, `Timeout`, `Loop`, and `Finally` middlewares — so any conformant
implementation can run the full suite without platform-specific integrations.

## Layout

Each case is a directory containing:

- `README.md` — the pattern the case demonstrates and the behaviors it checks.
- `case.json` — machine-readable case metadata: a one-line description, the
  [conformance profiles](#profiles-and-marks) the case requires, any execution
  [marks](#profiles-and-marks), and its [scenarios](#scenarios). Validated
  against `case.schema.json`.
- `definition.json` — a root workflow definition, valid against the flow schema
  (`static/v0.1/flow/schema.json`).
- `scenarios/<name>/` — one directory per scenario, holding that scenario's data
  files.

The definition is a pristine example workflow: anything a runner needs to know
about a case lives in `case.json`, never encoded inside the definition.

## Scenarios

A case separates the fixture from the assertions. The definition is the fixture;
a _scenario_ is one execution of it: a set of inputs and the Result that
execution must produce. A definition's outcome can depend entirely on what comes
in at execution start, so one case may carry several scenarios that probe the
same definition with different inputs — a routing Step's clauses, a validation's
accept and reject paths.

Each scenario is a directory under `scenarios/`, named by the scenario's key in
`case.json`, containing:

- `input.json` — the execution input the platform supplies to the root frame.
- `arguments.json` — _optional:_ the arguments the platform, as the root frame's
  caller, supplies for the root Flow's `parameters`. When present it MUST
  contain an object, validated against the root `parameters` schema at frame
  entry exactly as a subflow caller's `with` would be. An absent file means no
  arguments are supplied.
- `expected-result.json` — the Result the execution must produce.

A scenario's identity is `<case>/<name>`, e.g.
`300-match-routing/default-routes`. A case whose definition only needs one probe
has a single scenario conventionally named `base`.

## Running a case

For each scenario, start an execution of the case's `definition.json` with the
scenario's `input.json` as the execution input and, when `arguments.json` is
present, its object as the root frame's caller arguments, on a platform whose
catalog provides the spec-defined providers. The scenario passes when the
execution's Result matches its `expected-result.json` under the comparison rules
below; the case passes when every scenario passes.

A runner selects the cases an implementation can run by matching each case's
[`profiles`](#profiles-and-marks) against the implementation's conformance
claim, comparing the URIs character for character: a case is in scope when every
profile it lists is one the implementation claims. A runner that does not run
cases in real time may skip those marked [`temporal`](#profiles-and-marks), or
budget wall-clock time for them.

## Profiles and marks

`case.json` classifies each case along two independent axes, so a runner can
select the subset it can run and know how to run it. Both are required; either
array may be the minimal value for its axis.

### `profiles`

The conformance claim the case requires, as the profile URIs the reference's
[Conformance appendix](/reference/conformance/) defines. This is the full claim,
not a delta: every case lists the Core profile URI,
`https://mwl.dev/v0.1/conformance/core`, because a conformance claim always
includes Core. A case whose definition embeds `{{ }}` expressions adds the CEL
profile, `https://mwl.dev/v0.1/conformance/expressions/cel`, since evaluating it
requires CEL support; a case relying on a CEL extension capability would add
that extension's profile URI. The expression seam is pluggable, so a case
written in a future expression language would carry that language's profile
instead.

The `mock` call provider is not a profile: Core requires every implementation to
provide it (see the
[`mock` provider](/reference/providers/call-providers/#the-mock-provider)), so a
dispatching case needs nothing beyond Core to call it.

### `marks`

Execution marks, distinct from conformance profiles: a mark records how a runner
must execute a case, not which profiles an implementation must claim. This
version defines one.

- `temporal` — the case's outcome depends on real elapsed time (a `Sleep` pause,
  a `mock` delay, a `Timeout` bound), so the run takes wall-clock time. Every
  construct a temporal case exercises is Core; the mark is a runner hint, not a
  conformance requirement.

## Comparison rules

- `type` MUST match exactly.
- For a success Result, the actual `value` MUST be deeply equal to the expected
  `value`.
- For a failure Result, every member present in the expected envelope MUST be
  present in the actual Result with a deeply equal value, with two refinements:
  - `previous` is compared by these same envelope rules, recursively;
  - an expected member whose value is `null` also matches an absent member
    (`"previous": null` asserts that no failure is chained).
- Members absent from the expected envelope are unconstrained: `message` and
  `details` are omitted from an expected envelope wherever the specification
  leaves their content to the implementation.

## Cases

Cases are numbered by area: 0xx basics, 1xx Calls, 2xx failure handling, 3xx
Match, 4xx Gather, 5xx subflows, 6xx middleware, 7xx expressions.

| Case                               | Pattern                                                                 |
| ---------------------------------- | ----------------------------------------------------------------------- |
| `010-return-passthrough`           | The minimal Flow; `Return`'s passthrough default. Expression-free.      |
| `011-pass-reshape`                 | Reshaping a value with a `Pass` Step's `output`.                        |
| `012-assign-block-discipline`      | `assign` pre-block reads and last-write-wins.                           |
| `020-sleep-for`                    | A pure pause; the value passes through unchanged.                       |
| `021-sleep-until-past`             | `until` an instant already past completes at once.                      |
| `022-sleep-invalid-duration`       | An invalid duration: `System.ParameterValidationFailed`.                |
| `030-root-parameters`              | Root-frame parameters: defaults, platform overlay, validation.          |
| `100-call-mock-value`              | A basic dispatch; the Step `output` default reads the Result.           |
| `101-call-input-shaping`           | The `input` data channel, shaped at the call and echoed back.           |
| `102-call-arms-capture`            | `onSuccess` shaping and `provider.metadata` capture.                    |
| `103-call-onfailure-capture`       | The failure arm captures from the failed dispatch's window.             |
| `104-call-arm-fault`               | A faulting arm fails the Call though the target succeeded.              |
| `105-provider-with-validation`     | An undeclared provider argument fails `with` validation.                |
| `110-unhandled-failure`            | An uncaught failure becomes the Flow's Result.                          |
| `200-catch-exact-code`             | `catch` routing on an exact code, with clause `output`.                 |
| `201-catch-first-match-wins`       | Clause order: the first matching clause wins.                           |
| `210-raise-constructed`            | `Raise` constructs a failure envelope.                                  |
| `211-raise-bare-reraise`           | A bare `Raise` re-emits the handled failure unchanged.                  |
| `212-raise-failure-chaining`       | A new failure chains the active one as `previous`.                      |
| `213-raise-sever-previous`         | Writing `previous: null` severs the chain.                              |
| `214-extension-result-type`        | An extension Result type rides the envelope and matches by code.        |
| `220-empty-raise`                  | A bare `Raise` with no active failure: `System.EmptyRaise`.             |
| `230-failure-context-binding`      | A handler reads the matched envelope through `failure`.                 |
| `300-match-routing`                | `Match` clause selection: ordered cases and the `default` route.        |
| `302-match-predicate-fault`        | A faulting `when` falls through instead of failing.                     |
| `400-gather-iterate-echo`          | Iterate fan-out; `step.results` order; the `output` projection.         |
| `401-gather-scatter-order`         | Scatter fan-out, mixing provider and inline-flow targets.               |
| `402-gather-over-domain`           | The `over` domain: empty dispatches zero, non-array fails.              |
| `403-gather-call-index`            | `call.index` configures each dispatch.                                  |
| `410-gather-completion-unmet`      | A failed dispatch: `System.GatherCompletionUnmet` and its evidence.     |
| `411-gather-completion-successes`  | A partial-success policy; failures observed, not caught.                |
| `412-gather-wait-false`            | `wait: false` abandons the unfinished dispatch.                         |
| `413-gather-arm-fault-tightens`    | An arm fault tightens a met policy; the verdict is computed after arms. |
| `420-gather-dispatch-wrapping`     | Per-dispatch wrapping via a flow target carrying the stack.             |
| `421-gather-arm-order`             | Deferred arms run in dispatch order; accumulation is deterministic.     |
| `422-gather-catch-results`         | A `catch` clause reads the completed `step.results`.                    |
| `500-subflow-call`                 | A named subflow: `with` to `parameters`, `flow.vars` capture.           |
| `501-parameter-defaults-overlay`   | Defaults seed `vars`; supplied arguments overlay them.                  |
| `502-subflow-vars-isolation`       | A subflow cannot read its caller's variables.                           |
| `503-nested-subflow-scoping`       | Nested frames reuse Step names; scoping is per `steps` map.             |
| `510-subflow-failure-propagates`   | A subflow failure propagates and is caught by the caller.               |
| `511-parameter-validation-closed`  | Closed-by-default validation: an undeclared argument fails.             |
| `600-retry-exhausted`              | `Retry` budget exhaustion: `Provider.Middleware.Retry.Exhausted`.       |
| `601-retry-eventual-success`       | The `assign` carry across restored attempts; eventual success.          |
| `602-retry-gated-off`              | A gated-off `Retry` entry is transparent.                               |
| `610-timeout-exceeded`             | `Timeout` preempts a slow dispatch.                                     |
| `611-timeout-outside-retry`        | A bound outside `Retry` budgets all attempts together.                  |
| `612-timeout-inside-retry`         | A bound inside `Retry` budgets each attempt separately.                 |
| `620-loop-carried-value`           | `Loop`'s carried value feeds each next run.                             |
| `621-loop-vars-persist`            | Variables persist across `Loop` iterations.                             |
| `622-loop-zero-runs`               | A gated-off `Loop` emits its `onEntry` output product.                  |
| `630-finally-cleanup-failure`      | A failed `Finally` cleanup supersedes the Result in flight.             |
| `631-finally-supersedes-failure`   | A failed cleanup chains the failure it superseded.                      |
| `640-middleware-value-threading`   | `onEntry` output and `onSuccess` value shape descent and ascent.        |
| `641-middleware-translation-stack` | Stacked `onFailure` translations: order, inheritance, chaining.         |
| `642-middleware-with-validation`   | A phase `with` failing its schema fails from the entry's position.      |
| `650-flow-middleware-shaping`      | Flow-level middleware reshapes the frame input.                         |
| `651-flow-retry-graph`             | Flow-level `Retry` re-runs the whole Step graph.                        |
| `700-expression-evaluation-error`  | A faulting expression fails the frame.                                  |
| `701-expression-literal-boundary`  | A delimiter pair not spanning the string stays literal.                 |

## Repository checks

`scripts/check-schemas.sh` validates every case directory: the required files
exist, every `definition.json` validates against the flow schema, every
`expected-result.json` validates against `result.schema.json`, every `case.json`
validates against `case.schema.json`, every `input.json` parses, every
`arguments.json` present is a JSON object, and the scenario directories match
the `scenarios` declared in `case.json` exactly. What the script cannot check is
the semantics: that an expected Result actually follows from executing the
definition is established by the specification text, cited in each case's
README.
