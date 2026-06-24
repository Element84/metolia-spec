---
title: "Branching and variables"
weight: 30
---

Real workflows make decisions and carry values forward. MWL routes with the
`Match` action and carries state in `vars`, the Flow's variable namespace, kept
deliberately separate from the data plane: this is the **control plane**.

## Routing with `Match`

A `Match` Step tests predicates against one value and routes to the first clause
that holds:

```json
"route-by-value": {
  "action": "Match",
  "input": "{{ step.input.order }}",
  "cases": [
    {
      "when": "{{ match.input.status == 'approved' && match.input.amount > 1000.0 }}",
      "next": "manual-review"
    },
    {
      "when": "{{ match.input.status == 'approved' }}",
      "next": "auto-approve"
    }
  ],
  "default": { "next": "reject" }
}
```

- The Step's `input` is evaluated once, and every clause reads it as
  `match.input`. Clauses are tried in order; the first `when` that holds wins,
  and later predicates never evaluate.
- `default` is required: the route taken when no case matches. Routing always
  has a defined exit.
- Shaping belongs to the clause, not the Step: a clause may carry its own
  `output` (what the chosen successor receives, defaulting to `match.input`) and
  `assign`, so each route can shape differently.
- A `when` is an ordinary expression: one that fails to evaluate fails the Flow,
  like a fault anywhere else. When routing on data whose shape you aren't sure
  of, write the predicate to tolerate it —
  `{{ has(match.input.status) && match.input.status == 'approved' }}` — rather
  than counting on a broken read to skip the clause.

## Carrying values with `assign`

Data-plane values flow Step to Step, but plenty of state shouldn't have to ride
in the payload: an ID captured early and needed late, a count, a flag. `assign`
captures values into `vars` on successful Step exit:

```json
"validate-customer": {
  "action": "Call",
  "call": {
    "provider": "mwl:provider.call/example/http/v1",
    "with": { "method": "POST", "path": "/customers/validate" }
  },
  "assign": { "customerId": "{{ step.result.value.id }}" },
  "next": "check-inventory"
}
```

Every later expression in the Flow reads `vars.customerId`, however many Steps
away, without the payload between Steps carrying it. Two details worth
internalizing:

- `assign` runs after `output`, so a Step's own `output` cannot read what its
  `assign` writes.
- Within one `assign` block, every expression sees the variable state from
  before the block: entries in the same block can't read each other, so a block
  is a simultaneous write, not a sequence.

A name holds one binding at a time; assigning an existing name replaces it.

## Configuring a Flow with `parameters`

Variables don't only come from `assign`. A Flow declares **`parameters`**, a
JSON Schema describing the arguments it accepts; the caller's validated
arguments, overlaid on the schema's declared defaults, seed `vars` at frame
entry:

```json
{
  "$schema": "https://mwl.dev/v0.1/flow/schema.json",
  "parameters": {
    "type": "object",
    "properties": {
      "highValueThreshold": { "type": "number", "default": 1000 },
      "region": { "type": "string" }
    },
    "required": ["region"]
  },
  "entrypoint": "route-by-value",
  "steps": { "...": "..." }
}
```

From the first Step on, `vars.highValueThreshold` and `vars.region` are bound:
operational knobs, typed, defaulted, and validated, never threaded through the
data payload. The earlier `Match` predicate becomes
`{{ match.input.amount > vars.highValueThreshold }}`, and tuning the threshold
is a calling-time argument rather than an edit to the definition.

Validation is closed by default: an argument whose name matches no declared
property is rejected, and a Flow that declares no `parameters` accepts no
arguments at all. Where the arguments come from — the platform starting the
workflow, or a calling Flow's `with` — is the subject of the
[next page](../calls-and-results/) and [Subflows](../subflows/).

## Where the spec covers this

- [`Match`](/reference/step-actions/#match) — clauses, predicate evaluation, and
  failure behavior.
- [Variables: `assign`](/reference/step-mechanics/#variables-assign) — timing
  and the block discipline.
- [The `vars` model](/reference/flow-object/#the-vars-model) — scoping and
  seeding.
- [`parameters`](/reference/flow-object/#parameters) — the schema rules and
  validation behavior.
