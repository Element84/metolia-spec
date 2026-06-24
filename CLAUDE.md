# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## What this repository is

A Hugo static-site build of the **Metolia Workflow Language (MWL)**
specification — a JSON-based language for describing orchestrated workflows
(directed graphs of named steps that call services, branch, run concurrently,
wait, and handle failures). There is **no application code**; the deliverable is
the rendered specification site under `public/`. The Hextra theme is loaded as a
Go module — Go is required only for Hugo's module system, no Go source is built.

## ⚠️ Spec changes REQUIRE a schema and test review

The normative spec under `content/reference/` is not the only source of truth
about MWL. Two other artifacts encode the same language and **drift silently**
if the spec changes without them:

- **JSON Schemas** in `static/v0.1/` (`flow/schema.json`,
  `provider/schema.json`) — the machine-checkable grammar of a Flow and a
  provider definition.
- **Tests** in `tests/` — schema fixtures (`tests/schemas/valid/`,
  `tests/schemas/invalid/`) and conformance cases (`tests/conformance/`, each a
  `definition.json` + `case.json` + `scenarios/`).

**Whenever you change anything normative in `content/reference/`** (a field, a
discriminator value, a failure code, a default, a constraint, the shape of any
object, RFC 2119 wording that implies a rule), you MUST, in the same change:

1. **Review the JSON Schemas** and update them so they still accept exactly the
   valid documents and reject exactly the invalid ones the spec now describes.
2. **Review the tests** and add, update, or remove fixtures and conformance
   cases so they cover the new or changed behavior. A new construct or
   constraint needs both a positive and a negative case.
3. **Run `bash scripts/check-schemas.sh`** and confirm it passes.

If a spec change genuinely needs no schema or test change, say so explicitly and
why — do not skip the review silently. Treat "the prose changed but the schema
and tests did not" as a smell to justify, not a default.

## Commands

Prefer the wrapper scripts in `scripts/` over bare `hugo`/`dprint` invocations:
they run the same checks the pre-commit hook runs, in the right order.

```sh
hugo serve                          # local preview at http://localhost:1313/ with live reload
bash scripts/check-build.sh         # hugo build + spec-link check + HTML anchor check (all post-build checks)
bash scripts/check-schemas.sh       # validate the JSON Schemas, instance fixtures, and conformance cases
dprint fmt                          # format all Markdown/JSON
dprint check                        # verify formatting without writing
lefthook install                    # install the pre-commit hook
```

`check-schemas.sh` requires `check-jsonschema`
(`brew install check-jsonschema`).

The pre-commit hook (see `lefthook.yml`) runs three checks against the staged
index: `dprint fmt --fail-on-change`, `check-schemas.sh`, and `check-build.sh`.
The dprint command has `stage_fixed: false` — formatting edits land in the
working tree but are not auto-staged. Review and stage them yourself.

## Single-file spec output

In addition to the HTML site, `hugo` generates a single-file Markdown rendering
of the reference at `public/reference/spec.md`. This file is meant for
distribution to reviewers or upload as LLM context — it concatenates every
reference page in weight order, rewrites cross-page links to in-document
anchors, and strips Hugo shortcodes.

The output format is `SpecMarkdown`, defined in `hugo.yaml` and enabled on the
reference section via `outputs: ["html", "SpecMarkdown"]` in
`content/reference/_index.md`. The generator template is
`layouts/_default/list.specmarkdown.md`. `check-build.sh` runs
`check-spec-links.sh` against this file after every build to confirm every
in-document anchor link resolves to a real heading anchor — Hugo's
`--panicOnWarning` does not catch broken links inside the generated file because
the template emits raw Markdown without going through the rendering pipeline.

The reference's two `_index.md` files (`content/reference/_index.md` and
`content/reference/providers/_index.md`) set `singleFileSkipTOC: true` in
frontmatter, which causes the template to strip "In this document", "In this
section", and "Key terms" H2 blocks (TOC-shaped lists that were navigation aids
for the multi-page site and are redundant in the flat file).

## Architecture of the content

The specification is split into three top-level sections under `content/`, each
a Hugo section with its own `_index.md`:

- `content/guide/` — learning material (introduction, why MWL, by-example tour,
  cookbook patterns, end-to-end example). Non-normative.
- `content/reference/` — the **normative** specification: concepts, data model,
  definition format, the Flow object, expressions, execution model, execution
  context, step mechanics, step actions, the call interface, data flow,
  middleware mechanics, providers, conformance, failure-code reference, and a
  glossary. Most entries are single pages; `providers/` is a folder of per-topic
  pages with its own `_index.md`.
- `content/rationale/` — design principles, non-goals, architecture &
  motivation, changelog. Explains _why_, not _what_.

The navigation order across the site is Guide → Reference → Rationale, set in
`hugo.yaml` under `menu.main`.

## Authoring conventions

- **dprint formats Markdown with `textWrap: "always"` at 80 columns** (see
  `dprint.json`).
- Markdown italics use underscores (dprint default).
- **Use em-dashes sparingly, only where the sentence structure genuinely
  requires one** (a true parenthetical break or an abrupt shift the sentence
  cannot carry otherwise). Do not reach for an em-dash where a comma, a colon,
  or a restructured clause reads as well or better. An em-dash appositive that
  could be a comma-appositive should be a comma; a list item gloss that could be
  a clause should be a clause. Default to the lighter punctuation and earn the
  dash. When an em-dash is used, set it tight against the surrounding words with
  no spaces (`word—word`, not `word — word`).
- Hextra shortcodes (`{{< cards >}}`, `{{< card >}}`, etc.) are used in section
  landing pages — keep them when restructuring.
- The reference uses **RFC 2119** normative language ("MUST", "SHOULD", "MAY").
  Preserve it precisely; conformance statements are consolidated in
  `content/reference/conformance.md`.
- Do not use bold inline text as a pseudo-heading (e.g.,
  `**Section Title.** prose...`). Use real Markdown headings (`###`, `####`,
  etc.) for anything that names a topic. Bold-as-heading does not generate
  anchors, breaks the document outline, and prevents linking.
- **Use GFM alert syntax for notes and callouts**, not a heading or
  bold-as-heading. Begin a blockquote with `> [!NOTE]`, `> [!TIP]`,
  `> [!IMPORTANT]`, `> [!WARNING]`, or `> [!CAUTION]`, then the body on
  following `>` lines:

  ```markdown
  > [!IMPORTANT]
  > A note on terminology
  >
  > This specification also calls a non-success Result a failure Result…
  ```

  An alert is the right device for an aside that interrupts the main flow (a
  terminology note, a caveat, a pitfall) but does not name a topic in the
  document outline. It generates no heading anchor, so it does not clutter the
  outline or become a link target. It is blockquote-based GFM (not a Hugo
  shortcode), so it survives the single-file SpecMarkdown rendering as a plain
  blockquote.
- **Treat the spec as the initial version.** Describe what the language _is_,
  not how it differs from any prior iteration. Do not contrast with earlier
  designs or name constructs that no longer exist ("there is no longer a
  `finally` clause", "renamed from `arguments`", "this used to be…"). A reader
  has never seen another version; change-framing is noise to them. State the
  current behavior positively. (Migration notes, if ever needed, belong in the
  rationale changelog, not the normative reference.)

### Casing

Casing carries the role of an identifier — "PascalCase means type-or-variant;
camelCase means slot-or-binding." The JSON identifier rules are decided in
[ADR 0002](working-docs/decisions/0002-rename-field-keys-to-camelcase.md):

- **Field keys are camelCase**: `steps`, `entrypoint`, `failureCodes`, `with`,
  `onEntry`. (`$schema` is the lone exception, fixed by JSON Schema convention.)
- **Runtime binding identifiers are camelCase**: `vars`, `step`, `call`, `flow`,
  `middleware`, `failure`.
- **Discriminator values are PascalCase**: `"action": "Call"`,
  `"action": "Gather"`. So are **terminal states** (`Succeeded`, `Cancelled`)
  and **failure codes** (`System.ParameterValidationFailed`,
  `Provider.Call.Payments.CardDeclined`) — these are type/variant names in a
  taxonomy.
- **Result `type` values are the exception**: the five spec-defined types are
  lowercase (`success`, `error`, `cancellation`, `timeout`, `skipped`), and the
  lowercase space is reserved to the spec; extension Result types are PascalCase
  (`ProcessingError`).

In **prose**, the same principle extends to concept nouns: when a word names a
**defined model type**, capitalize it. Generic or instance nouns stay lowercase.

- **PascalCase in prose** (defined types): **Flow**, **Result**, **Step**,
  **Call** — and any enumeration member named as such (`Return`, `Raise`,
  `Gather`, `Match`). Example: "a Flow describes a workflow; each Step has a
  Call; the Flow produces a Result."
- **lowercase in prose** (generic / instance / not a type): `workflow`,
  `subflow`, `frame`, `middleware`, `provider`, and the verb "to call" ("a Step
  that calls a flow").
- **Backticks** whenever the word is a literal field key or binding regardless
  of the above — `steps`, `middleware` (the field), `provider` (the field),
  `flow` (the call field). The backtick form always uses the JSON casing.

### Headings and titles

- **Headings and frontmatter titles are sentence case**, not title case: only
  the first word and words that are otherwise capitalized by the casing rule
  above (PascalCase types, code identifiers) are capitalized. So "The Flow
  object", "Where Flows appear", "Result types", "`value`: shaping the produced
  Result" — but not "The Flow Object" or "Result Types".
