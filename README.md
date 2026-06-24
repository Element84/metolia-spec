# Metolia Workflow Language (MWL) Specification

This repository contains the specification for the **Metolia Workflow Language**
(MWL), a JSON-based language for describing orchestrated workflows. A workflow
is a directed graph of named steps, each performing an action and transitioning
to the next; steps can call external services, branch on conditions, run work
concurrently, wait, and handle failures.

The specification is published as a [Hugo](https://gohugo.io/) static site using
the [Hextra](https://github.com/imfing/hextra) theme. Source content lives under
`content/`:

- `content/guide/` — introductory material, by-example tour, cookbook patterns,
  end-to-end example, glossary.
- `content/reference/` — the normative specification (concepts, data model,
  definition format, expressions, the call interface, the flow object, step
  mechanics and actions, execution model and context, middleware mechanics,
  providers, conformance, failure-code reference).
- `content/rationale/` — design principles, non-goals, and architectural
  reasoning.

## Future goals

While this repo's contents are currently limited to the spec and some test
cases, we full expect to extend the contents of this repo to include API specs
for the Metolia workflow orchestration runtime, as well as validation and
conformance tooling for users and developers.

## Dependencies

The repository's tooling assumes the following are installed and available on
`PATH`:

- **[Hugo](https://gohugo.io/installation/)** — static site generator. Tested
  with Hugo Extended v0.160+.
- **[Go](https://go.dev/dl/)** — required by Hugo's module system, which is how
  this project loads the Hextra theme. Tested with Go 1.26+. No Go code is built
  directly; Go is invoked transparently by Hugo when fetching theme modules.
- **[dprint](https://dprint.dev/install/)** — code formatter used for Markdown
  and JSON. Configured via `dprint.json`.
- **[lefthook](https://github.com/evilmartians/lefthook)** — Git hook manager.
  Configured via `lefthook.yml`.
- **[check-jsonschema](https://github.com/python-jsonschema/check-jsonschema)**
  — validates the MWL JSON Schemas and the documents that instantiate them
  (`brew install check-jsonschema`). Used by `scripts/check-schemas.sh`.

## Setup

After cloning the repository:

```sh
lefthook install
```

This installs the Git hooks defined in `lefthook.yml` into `.git/hooks/`. On
`pre-commit`, lefthook runs `dprint fmt` on staged Markdown and JSON files, the
JSON Schema validation (`scripts/check-schemas.sh`), and the build and link
checks (`scripts/check-build.sh`). Formatting changes are written to the working
tree but not auto-staged; review and verify the changes then add them yourself.

## Running the docs locally

To preview the rendered specification site:

```sh
hugo serve
```

Hugo will serve the site at <http://localhost:1313/> with live reload on content
changes.

To produce a one-shot build to the `public/` directory:

```sh
hugo
```

To verify the site builds without warnings (broken internal links, missing
references, shortcode errors):

```sh
hugo --renderToMemory --logLevel warn --panicOnWarning
```

## Formatting

To format all Markdown and JSON files in the repository:

```sh
dprint fmt
```

To check whether files would be reformatted without modifying them:

```sh
dprint check
```

Configuration is in `dprint.json`. Lefthook will run dprint formatting as a
pre-commit hook.

## Checks

Alongside the HTML site, a `hugo` build emits a single-file Markdown rendering
of the reference at `public/reference/spec.md`, concatenating every reference
page in order with cross-page links rewritten to in-document anchors. This file
is meant for distribution to reviewers or upload as LLM context.

The scripts under `scripts/` verify the build. To run them all (build, then link
and anchor checks):

```sh
bash scripts/check-build.sh
```

Individually:

```sh
bash scripts/check-spec-links.sh                 # verify in-doc anchors in public/reference/spec.md
python3 scripts/check-html-anchors.py reference  # verify fragment anchors in the rendered HTML (run hugo first)
bash scripts/check-schemas.sh                    # validate the JSON Schemas and the documents that instantiate them
```

Lefthook runs the schema and build checks on `pre-commit`.
