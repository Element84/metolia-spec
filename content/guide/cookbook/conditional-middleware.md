---
title: "Conditional middleware"
weight: 70
---

## Problem

The same workflow runs in different modes: retries make sense in production but
mask bugs in testing; results should publish to the catalog only on real runs; a
deadline applies only when the caller asks for one. Maintaining one copy of the
workflow per mode is the failure case.

## Pattern

Every middleware phase block accepts `when`, a predicate gating that phase's
action. Drive the predicates from Flow `parameters`, and the modes become
arguments:

```json
{
  "$schema": "https://mwl.dev/v0.1/flow/schema.json",
  "parameters": {
    "type": "object",
    "properties": {
      "retriesEnabled": { "type": "boolean", "default": true },
      "publish": { "type": "boolean", "default": false },
      "deadline": { "type": "string", "format": "duration", "default": "PT0S" }
    }
  },
  "middleware": [
    {
      "provider": "mwl:provider.middleware/mwl/timeout/v1",
      "onEntry": {
        "when": "{{ vars.deadline != 'PT0S' }}",
        "with": { "duration": "{{ vars.deadline }}" }
      }
    },
    {
      "provider": "mwl:provider.middleware/example/stac-index/v1",
      "onSuccess": {
        "when": "{{ vars.publish }}",
        "with": { "items": "{{ middleware.result.value.features }}" }
      }
    }
  ],
  "entrypoint": "process",
  "steps": {
    "process": {
      "action": "Call",
      "call": {
        "provider": "mwl:provider.call/example/http/v1",
        "with": { "method": "POST", "path": "/process" }
      },
      "middleware": [
        {
          "provider": "mwl:provider.middleware/mwl/retry/v1",
          "onEntry": {
            "when": "{{ vars.retriesEnabled }}",
            "with": {
              "policies": [
                { "match": { "codes": ["Provider.Call.*"] }, "attempts": 3 }
              ]
            }
          }
        }
      ],
      "next": "done"
    },
    "done": { "action": "Return" }
  }
}
```

## Why this shape

`when` is the enablement channel, separate from configuration. `with` says how
the action behaves; `when` says whether it runs. Keeping them apart beats
encoding "off" into a configuration value (a zero duration, an empty recipient
list): gated off, a `Retry` is transparent and a `Timeout` imposes no bound,
with no sentinel values to interpret. (The `deadline` parameter above shows the
line: `PT0S` is the _argument_ convention for "no deadline", but the decision is
expressed in `when`, not left for the middleware to infer.)

A gated phase skips its action and its `with`. When `when` is false, the
configuration isn't even evaluated, so a disabled entry can't fail parameter
validation on values meant for the enabled case.

Timing matters for `onEntry`. An entry's `onEntry` runs once, at establishment,
and what it decides persists: `vars.retriesEnabled` is consulted when the stack
descends, not per failure. Ascent phases evaluate their `when` when they run, so
the publish gate above is checked when the Result rises. Either way, the
predicate reads the live frame state at its phase's moment.

The author's shaping is not gated. `when` gates the _middleware's action_; any
`output`, `value`, envelope keys, or `assign` you write in the block are your
code and evaluate whenever the phase runs. An entry can therefore always observe
and capture, even with its action off.

## Variations

- **Environment-driven gates.** Platforms can surface deployment context under
  `execution.platform`; a predicate like
  `{{ execution.platform.environment == 'production' }}` gates publication
  without any caller involvement. (Reading `execution.platform` members is
  portable only across platforms that define them.)
- **Data-driven gates.** Predicates can read the value in flight:
  `"when": "{{ size(middleware.input.features) > 0.0 }}"` skips publishing empty
  result sets.
- **Beware gated transforms.** Gating a middleware whose action reshapes the
  value (decrypt, decompress) makes the downstream shape conditional; both
  branches become yours to handle. Prefer gating side-effect and control
  actions.

## See also

- [`when`: gating the action](/reference/middleware-mechanics/#when-gating-the-action)
  — semantics, timing, and the transform warning.
- [The phase block](/reference/middleware-mechanics/#the-phase-block) — key
  resolution order.
- [`parameters`](/reference/flow-object/#parameters) — declaring the knobs.
