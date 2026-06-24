---
title: "Pagination accumulation"
weight: 40
---

## Problem

An API returns results a page at a time, each response carrying a cursor for the
next request. You want all the items as one collection before moving on.

## Pattern

`Loop` on the `Call` Step. The carried value drives the cursor; `vars`
accumulates the pages:

```json
"fetch-all-items": {
  "action": "Call",
  "call": {
    "provider": "mwl:provider.call/example/http/v1",
    "with": { "method": "GET", "path": "/items" }
  },
  "middleware": [
    {
      "provider": "mwl:provider.middleware/mwl/loop/v1",
      "onSuccess": {
        "when": "{{ middleware.result.value.nextCursor != null }}",
        "assign": { "items": "{{ vars.items + middleware.result.value.items }}" }
      }
    }
  ],
  "output": "{{ vars.items }}",
  "next": "process-items"
}
```

Declare the accumulator in the Flow's `parameters` so it starts defined:

```json
"parameters": {
  "type": "object",
  "properties": { "items": { "type": "array", "default": [] } }
}
```

## Why this shape

The carried value is the pagination protocol. `Loop`'s `onSuccess` `value`
defaults to passing each iteration's output through as the next iteration's
input, so each response — cursor included — becomes the next dispatch's
`call.input`. An API that echoes its own next-request shape paginates with no
plumbing at all; one that doesn't gets a `value` expression building the next
request from the response.

The accumulator is loop state, so it lives in `vars`. Variables persist across
`Loop` iterations (an iteration is progress, not a retry), and the `assign` runs
every pass, concatenating each page's items in order. When the continuation goes
false, the loop emits, and the Step's `output` reads the final list — note
`output` reads the accumulated variable, not the last response.

Termination reads the response. The continuation
`{{ middleware.result.value.nextCursor != null }}` is a do-while: dispatch once,
keep going while there's a cursor. There is no max-iterations parameter to
forget; a runaway-API guard is one more conjunct,
`{{ ... && middleware.metadata.iteration < 1000.0 }}`.

## Variations

- **Multi-Step pages.** When each page needs fetch-then-transform-then-store,
  move the loop up a level: wrap the Steps in a subflow and put `Loop` on the
  Flow's `middleware`, with the cursor in `vars`
  (`"when": "{{ vars.cursor != null }}"`) since the carried value is then the
  whole graph's output. The same shape as [Polling with timeout](../polling/),
  with a cursor where the done-flag was.
- **Bound the whole crawl.** `Timeout` outside the `Loop` bounds all pages
  together; see [Retry composition](../retry-composition/) for the positioning
  logic.
- **Flaky pages.** `Retry` inside the `Loop` (after it in the array) retries an
  individual page fetch without restarting the crawl; the loop only ever sees
  successes rise.
- **Very large result sets.** Accumulating in `vars` holds everything in the
  execution context. When the collection is large, accumulate _references_
  (write each page to storage behind a provider and collect the keys), or have
  the consumer take pages one at a time instead of gathering them.

## See also

- [The `Loop` middleware](/reference/providers/middleware-providers/#the-loop-middleware)
  — carried value, continuation, and the `Retry` contrast.
- [The `vars` model](/reference/flow-object/#the-vars-model) — why variables
  persist here and restore under `Retry`.
- [Polling with timeout](../polling/) — the same loop driven by a flag.
