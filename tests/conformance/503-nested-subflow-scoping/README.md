# 503 — Nested subflows and Step-name scoping

Three frames deep: the root calls `Outer`, whose own `flows` map declares
`Inner`, and every Flow reuses the same Step names (`go`, `done`). Step
references resolve against the `steps` map that contains the referencing Step,
so the reuse is unambiguous, and each frame's value threads through the
composition.

## Checks

- A subflow declares its own `flows`; a `call` inside it resolves the name
  against that map.
- Step names are scoped per `steps` object: identical names in different Flows
  never collide.
- Values thread through nested frames: input down through each call, the
  Result's value back up through each `output` default.

Reference: The Flow object § flows, § Step-name scoping.
