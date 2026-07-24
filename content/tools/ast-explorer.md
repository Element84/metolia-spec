---
title: "AST explorer"
weight: 20
---

<!--
This page mixes markdown with raw HTML: the trees are interactive HTML, the
headings, intros, and guarantee callouts are markdown. It is excluded from
dprint formatting (see dprint.json). Each .row carries a data-field attribute
naming the parser struct fields it documents (space-separated when one row
covers several), and each section header row carries data-type;
scripts/check-explorer.py diffs these against the parser's AST declarations
(ast_flow.go, ast_provider.go, ast.go, value.go, and Position from
diagnostic.go).

ROW TEMPLATES, copy and fill in. Three shapes exist.

Section header (flush left, always expandable; chip is "struct", or "Step" on
the seven step variants):

<div class="row" data-type="TypeName"><details class="node"><summary><span class="nm">TypeName</span> <span class="ty">struct</span> <span class="gloss">one line, no trailing period</span></summary>
<div class="panel"><p>Fuller explanation.</p><span class="spec"><b>spec:</b> Page title § section</span></div>
</details></div>

Expandable field row:

<div class="row" data-field="TypeName.Field"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Field</span> <span class="ty">GoType</span> <span class="gloss">one line</span></summary>
<div class="panel"><p>Fuller explanation.</p></div>
</details></div>

Leaf field row (nm-lead is the blank slot that aligns the name with chevroned
rows at the same depth):

<div class="row" data-field="TypeName.Field"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Field</span> <span class="ty">GoType</span> <span class="gloss">one line</span></div>

Rules: the last field row of a shape uses └─ instead of ├─. One conn span per
depth level, each exactly one level wide: a rail continuation ("│  ") or three
spaces under a └─ parent, then the branch ("├─ " or "└─ "). The wrap CSS
counts conn spans, so never fold two levels into one span. Pos and Comment
fields get no rows; they are covered by the intro prose of each tree.

Every other exported field of the checked AST files gets its own visible row,
and a row's data-field names only fields the row itself shows — never fields a
panel merely mentions. Fields share a row only when the pairing is the
semantics: an exactly-one pair (Call / Middleware, Over + Call) or a
Name + NamePos pair. No row defers its documentation to another type
("as on X"). Grammar types (ast_flow.go, ast_provider.go) are expanded where
they appear in the trees; support types (ast.go, value.go, Position) are
defined once in the "Support types" section and stay opaque chips at use
sites.

Author .ty chips as plain text: the script at the bottom finds every known
type name inside every chip and wraps it in a link to where the type is
documented (a section for data-type headers, the expanding row for nested
helpers) — unmarked at rest, accent on hover. The name set and targets derive
from the data-type/data-field attributes, so new rows and types link without
markup changes. Never hand-write <a> tags inside a chip.
-->

<div class="ast-explorer">

<style>
  .ast-explorer {
    /* the accent is the site's own primary, built from the same hue variables
       Hextra derives its link/brand blue from, so it tracks any site recolor */
    --x-accent: hsl(var(--primary-hue) var(--primary-saturation) 45%);
    --x-accent-soft: color-mix(in srgb, var(--x-accent) 14%, transparent);
    --x-muted: #6b7280;
    --x-faint: #9ca3af;
    --x-body: #374151;
    --x-rule: #e5e7eb;
    --x-surface: color-mix(in srgb, var(--x-accent) 5%, transparent);
    --x-chip-border: rgba(0, 0, 0, 0.07);
    --x-page: #fff;
    --x-mono: var(--hx-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
    --x-sans: var(--hx-font-sans, ui-sans-serif, system-ui, sans-serif);
    /* Hextra's own details-chevron artwork (see the theme's typography.css),
       refilled per color scheme since currentColor cannot reach a background */
    --x-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 20 20' fill='%23374151'%3E%3Cpath fill-rule='evenodd' d='M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z' clip-rule='evenodd'/%3E%3C/svg%3E");
  }
  html.dark .ast-explorer {
    --x-accent: hsl(var(--primary-hue) var(--primary-saturation) 66%);
    --x-muted: #9ca3af;
    --x-faint: #6b7280;
    --x-body: #d4d4d4;
    --x-rule: #262626;
    --x-surface: color-mix(in srgb, var(--x-accent) 8%, transparent);
    --x-chip-border: rgba(255, 255, 255, 0.1);
    --x-page: #111;
    --x-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 20 20' fill='%23a3a3a3'%3E%3Cpath fill-rule='evenodd' d='M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z' clip-rule='evenodd'/%3E%3C/svg%3E");
  }

  /* ---- the trees ---- */
  .ast-explorer .tree {
    font-family: var(--x-mono);
    font-size: var(--hx-text-sm);
    line-height: 1.75;
    overflow-x: auto;
    margin-top: 1.75rem;
  }
  .ast-explorer .tree section { margin-bottom: 1.75rem; }

  /* rows wrap with a hanging indent: the continuation line starts where the
     row's name does, keeping the connector rail clean. Depth is read from the
     number of leading connector spans; each connector is 3ch wide and the
     chevron slot is 1.35em. */
  .ast-explorer .row {
    white-space: normal;
    padding-left: calc(3ch + 1.35em);
    text-indent: calc(-3ch - 1.35em);
  }
  .ast-explorer .row:not(:has(> .conn)) { padding-left: 1.35em; text-indent: -1.35em; }
  .ast-explorer .row:has(> .conn + .conn) { padding-left: calc(6ch + 1.35em); text-indent: calc(-6ch - 1.35em); }
  .ast-explorer .row:has(> .conn + .conn + .conn) { padding-left: calc(9ch + 1.35em); text-indent: calc(-9ch - 1.35em); }
  .ast-explorer .conn { color: var(--x-faint); user-select: none; white-space: pre; }
  .ast-explorer .nm { font-weight: 600; }
  /* the row naming a struct (the section header) reads heavier than its fields;
     the tree carries no color of its own, so blue stays exclusive to links */
  .ast-explorer .row[data-type] .nm { font-weight: 700; }
  /* a type is code: give it the site's inline-code chip so name, type, and
     description read as three distinct objects at a glance */
  .ast-explorer .ty {
    font-size: 0.86em;
    color: var(--x-body);
    background: var(--x-surface);
    border: 1px solid var(--x-chip-border);
    border-radius: 4px;
    padding: 0.08em 0.35em;
  }
  .ast-explorer .gloss { color: var(--x-body); }

  /* inline code in tree rows reads as part of the tree, not as a code chip;
     code inside expanded panels keeps the site's native inline-code style */
  .ast-explorer .tree code:not(.panel *) {
    font: inherit;
    background: none;
    border: none;
    padding: 0;
  }

  /* nodes with an expansion */
  .ast-explorer details.node { display: inline; }
  .ast-explorer details.node > summary {
    display: inline;
    list-style: none;
    cursor: pointer;
  }
  .ast-explorer details.node > summary::-webkit-details-marker { display: none; }
  /* leading disclosure chevron: the theme's own artwork (--x-chevron), sized
     for the tree. Every property Hextra sets on article summary:before
     (background-image, padding, vertical-align — see typography.css) is
     overridden here, or the two stylings combine into a double arrow. Points
     right; rotates to point down when open. Leaf rows carry a blank slot
     (.nm-lead) of the same advance so every name at a depth shares a column. */
  .ast-explorer details.node > summary::before {
    content: "";
    display: inline-block;
    width: 1em;
    height: 1em;
    background: var(--x-chevron) center / contain no-repeat;
    padding: 0;
    margin-right: 0.35em;
    vertical-align: -0.12em;
    transition: transform 0.12s ease;
  }
  .ast-explorer .nm-lead { display: inline-block; width: 1.35em; }
  @media (prefers-reduced-motion: reduce) {
    .ast-explorer details.node > summary::before { transition: none; }
  }
  .ast-explorer details.node[open] > summary::before { transform: rotate(90deg); }
  .ast-explorer details.node > summary:focus-visible { outline: 2px solid var(--x-accent); outline-offset: 2px; }
  /* the same chevron, for referring to it in prose */
  .ast-explorer .tri {
    display: inline-block;
    width: 1em;
    height: 1em;
    background: var(--x-chevron) center / contain no-repeat;
    vertical-align: -0.12em;
  }

  .ast-explorer .panel {
    font-family: var(--x-sans);
    font-size: var(--hx-text-sm);
    line-height: 1.6;
    white-space: normal;
    text-indent: 0; /* do not inherit the row's hanging indent */
    background: var(--x-surface);
    border-left: 2px solid var(--x-accent);
    border-radius: 0 4px 4px 0;
    padding: 0.6rem 0.9rem;
    margin: 0.3rem 0 0.5rem;
    max-width: 44rem;
  }
  .ast-explorer .panel p { margin: 0 0 0.5rem; }
  .ast-explorer .panel p:last-child { margin-bottom: 0; }
  .ast-explorer .panel .spec {
    display: block;
    margin-top: 0.4rem;
    font-size: var(--hx-text-xs);
    color: var(--x-faint);
  }
  .ast-explorer .panel .spec b { color: var(--x-muted); font-weight: 600; }

  /* pre blocks inherit the site's tinted code-block background; just size them
     for panel/dialog scale and strip the inline-code chip from their contents */
  .ast-explorer .panel pre,
  .ast-explorer dialog.concept pre {
    padding: 0.6rem 0.9rem;
    margin: 0.5rem 0;
    border-radius: 0.5rem;
    overflow-x: auto;
  }
  .ast-explorer .panel pre code,
  .ast-explorer dialog.concept pre code {
    background: none;
    border: none;
    padding: 0;
    font-size: var(--hx-text-xs);
    line-height: 1.5;
  }

  /* concept terms: dotted, expand a shared definition */
  .ast-explorer .term {
    color: var(--x-accent);
    border: none;
    border-bottom: 1px dotted var(--x-accent);
    cursor: pointer;
    background: none;
    font: inherit;
    padding: 0;
  }
  .ast-explorer .term:focus-visible { outline: 2px solid var(--x-accent); outline-offset: 1px; }

  /* support-type links inside chips, jumping to the type's entry in Support
     types: unmarked at rest, accent on hover — the IDE convention for
     jump-to-definition. A chip has no room for an underline (it collides
     with the chip border), and dotted stays exclusive to .term's popup
     definitions; the intro prose teaches that chip type names link. */
  .ast-explorer .ty a.ty-link {
    color: inherit;
    text-decoration: none;
  }
  .ast-explorer .ty a.ty-link:hover,
  .ast-explorer .ty a.ty-link:focus-visible {
    color: var(--x-accent);
  }
  /* jump targets land below Hextra's sticky navbar */
  .ast-explorer .tree section[id],
  .ast-explorer .tree .row[id] { scroll-margin-top: 5rem; }

  /* concept definition dialog. An ancestor with a transform/filter becomes the
     containing block for the top-layer dialog, so its default margin:auto no
     longer centers against the viewport; pin it to viewport center explicitly. */
  .ast-explorer dialog.concept {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    margin: 0;
    font-size: var(--hx-text-sm);
    line-height: 1.6;
    color: inherit;
    background: var(--x-page);
    border: 1px solid var(--x-rule);
    border-top: 3px solid var(--x-accent);
    border-radius: 6px;
    width: calc(100vw - 2rem);
    max-width: 34rem;
    max-height: calc(100vh - 4rem);
    overflow-y: auto;
    padding: 1rem 1.25rem;
    box-shadow: 0 8px 30px rgba(0, 0, 0, 0.25);
  }
  .ast-explorer dialog.concept::backdrop { background: rgba(0, 0, 0, 0.35); }
  .ast-explorer dialog.concept h3 {
    font-family: var(--x-mono);
    font-size: var(--hx-text-base);
    margin: 0 0 0.5rem;
    color: var(--x-accent);
  }
  .ast-explorer dialog.concept p { margin: 0 0 0.6rem; }
  .ast-explorer dialog.concept p:last-of-type { margin-bottom: 0.75rem; }
  .ast-explorer dialog.concept form { text-align: right; margin: 0; }
  .ast-explorer dialog.concept form button {
    font-size: var(--hx-text-sm);
    color: var(--x-accent);
    background: none;
    border: 1px solid var(--x-accent-soft);
    border-radius: 3px;
    padding: 0.25rem 0.7rem;
    cursor: pointer;
  }
  .ast-explorer dialog.concept form button:hover { background: var(--x-accent-soft); }
</style>

The trees the parser returns, explained field by field: [`parser.ParseFlow`](#the-flow-ast)
for a Flow document, [`parser.ParseProvider`](#the-provider-ast) for a provider
definition, and the [support types](#support-types) both share.
<span class="tri"></span> expands a full
explanation in place; <span class="term" style="cursor:default">dotted&nbsp;terms</span>
define the language concept they name; a type name inside a type chip links to
where that type is documented.

## The Flow AST

Returned by `parser.ParseFlow`: the root Flow, the seven step variants, and the
shared shapes they reference (Call, CatchClause, MiddlewareEntry).

Two recurring fields are not repeated in the rows. Every shape carries
`Pos Position`, its location in the source document (the ledger entries,
`NamedFlow` and `NamedStep`, record `NamePos` instead; `FlowTarget`'s arms
carry their own). The Flow, each step, each clause, the call object, and each
middleware entry also carry `Comment string`, an author annotation the
language never interprets.

<div class="tree">

<!-- ============ Flow ============ -->
<section class="root">
<div class="row" data-type="Flow"><details class="node"><summary><span class="nm">Flow</span> <span class="ty">struct</span> <span class="gloss">one workflow definition (root, named, or inline)</span></summary>
<div class="panel"><p>The shape every workflow takes, wherever it appears: the root document is a Flow, each <code>flows</code> ledger entry holds one, and a call may write one inline. All three levels share this same surface; only the root document carries <code>$schema</code>.</p><span class="spec"><b>spec:</b> The Flow object</span></div>
</details></div>

<div class="row" data-field="Flow.Schema"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Schema</span> <span class="ty">string</span> <span class="gloss">the root $schema URI; empty on nested Flows</span></summary>
<div class="panel"><p>The root document's <code>$schema</code> URI, declaring the format version. Empty on named and inline Flows: only the root document declares one.</p></div>
</details></div>

<div class="row" data-field="Flow.Entrypoint"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Entrypoint</span> <span class="ty">Name</span> <span class="gloss">the step execution enters at</span></summary>
<div class="panel"><p>Always a key of this Flow's own <code>Steps</code> — the check pass proved it resolves. From here each step's routing carries execution until a <button class="term" data-c="terminal">terminal</button> step ends the Flow.</p><span class="spec"><b>spec:</b> The Flow object § entrypoint</span></div>
</details></div>


<div class="row" data-field="Flow.Flows"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Flows</span> <span class="ty">[]NamedFlow</span> <span class="gloss">the subflow ledger: name → Flow, in order</span></summary>
<div class="panel"><p>Each entry pairs an author-chosen name with a full Flow; a Call targets one by writing the bare name in its <code>flow</code> field. Names resolve up the chain of enclosing Flows, nearest declaration first.</p>
<p>In the source JSON, <code>flows</code> is a name-keyed <em>object</em> — the name sits outside the thing it names — so the parser lifts it into an ordered slice of pairs. A <code>NamedFlow</code> is the ledger entry (<code>Name</code>, <code>NamePos</code>); the Flow it <em>holds</em> is the subflow.</p>
<pre><code>"flows": { "Greet": { ...a whole Flow... } }</code></pre>
<span class="spec"><b>spec:</b> The Flow object § flows, § Flow-name scoping</span></div>
</details></div>

<div class="row" data-field="NamedFlow.Name NamedFlow.NamePos"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Name, NamePos</span> <span class="ty">string, Position</span> <span class="gloss">the key the author chose</span></div>
<div class="row" data-field="NamedFlow.Flow"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Flow</span> <span class="ty">*Flow</span> <span class="gloss">recursive: the subflow itself</span></div>

<div class="row" data-field="Flow.Middleware"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Middleware</span> <span class="ty">[]MiddlewareEntry</span> <span class="gloss">flow-level <button class="term" data-c="middleware">stack</button>, wrapping every step</span></summary>
<div class="panel"><p>The flow-level middleware stack. Unlike <code>flows</code>, this is a JSON <em>array</em> whose entries carry their own identity inside themselves (the <code>provider</code> field) — so it is a plain slice with no <code>Named*</code> wrapper. The entry's own shape is at the bottom of this tree.</p><span class="spec"><b>spec:</b> Middleware mechanics</span></div>
</details></div>

<div class="row" data-field="Flow.Parameters"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Parameters</span> <span class="ty">*Value</span> <span class="gloss">JSON Schema for the Flow's named parameters</span></summary>
<div class="panel"><p>Describes the parameters callers pass via <code>with</code>; must declare <code>"type": "object"</code>. Held as a raw positioned tree: evaluating a schema is a validator's or runtime's concern, not the parser's.</p><span class="spec"><b>spec:</b> The Flow object § parameters</span></div>
</details></div>

<div class="row" data-field="Flow.Steps"><span class="conn">└─ </span><details class="node"><summary><span class="nm">Steps</span> <span class="ty">[]NamedStep</span> <span class="gloss">the step graph, document order</span></summary>
<div class="panel"><p>Look a step up by name with <code>StepByName</code>; look a subflow up with <code>FlowByName</code> (this Flow's own ledger only — it does not walk the enclosing chain). Each <code>NamedStep</code> pairs the step's name (what <code>entrypoint</code> and <code>next</code> cite) with the step itself.</p><span class="spec"><b>spec:</b> The Flow object § steps</span></div>
</details></div>

<div class="row" data-field="NamedStep.Name NamedStep.NamePos"><span class="conn">   </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Name, NamePos</span> <span class="ty">string, Position</span> <span class="gloss">the step's name, cited by <code>entrypoint</code> and <code>next</code></span></div>

<div class="row" data-field="NamedStep.Step"><span class="conn">   </span><span class="conn">└─ </span><details class="node"><summary><span class="nm">Step</span> <span class="ty">Step</span> <span class="gloss">closed sum of seven variants — dispatch by type switch</span></summary>
<div class="panel"><p>The set is closed: <code>*CallStep</code>, <code>*GatherStep</code>, <code>*MatchStep</code>, <code>*PassStep</code>, <code>*SleepStep</code>, <code>*ReturnStep</code>, <code>*RaiseStep</code>, and nothing else. Handle one with a type switch; a panicking <code>default</code> arm is a real safety net against future variants.</p></div>
</details></div>
</section>

</div><!-- /.tree -->

### The step variants

Each `NamedStep` holds exactly one of these seven shapes in its `Step` slot;
the sum is closed.

<div class="tree">

<!-- ============ CallStep ============ -->
<section>
<div class="row" data-type="CallStep"><details class="node"><summary><span class="nm">CallStep</span> <span class="ty">Step</span> <span class="gloss">make one <button class="term" data-c="dispatch">dispatch</button>, route on its Result</span></summary>
<div class="panel"><p>The workhorse step: dispatch a single call at a provider or a subflow, wait for its <button class="term" data-c="result">Result</button>, and route — <code>Next</code> on success, <code>Catch</code> on failure.</p><span class="spec"><b>spec:</b> Step actions § Call</span></div>
</details></div>

<div class="row" data-field="CallStep.Call"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Call</span> <span class="ty">*Call</span> <span class="gloss">the call object (target, args, arms) — see below</span></div>

<div class="row" data-field="CallStep.Input"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Input</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: incoming value → working value</span></summary>
<div class="panel"><p>Transforms the value received from the predecessor into the value this step works on. Nil means the passthrough default: work on the value as received.</p><span class="spec"><b>spec:</b> Step mechanics § Data flow</span></div>
</details></div>

<div class="row" data-field="CallStep.Output"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Output</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: working value → emitted value</span></summary>
<div class="panel"><p>Builds the value handed to the successor on success. Nil means the action's own passthrough default.</p><span class="spec"><b>spec:</b> Step mechanics § Data flow</span></div>
</details></div>

<div class="row" data-field="CallStep.Assign"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> into vars on success</span></summary>
<div class="panel"><p>The step's capture block: variable name → expression, all evaluated against the variable state from <em>before</em> the step, taking effect on successful exit.</p><span class="spec"><b>spec:</b> Step mechanics § Variables: assign</span></div>
</details></div>

<div class="row" data-field="CallStep.Middleware"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Middleware</span> <span class="ty">[]MiddlewareEntry</span> <span class="gloss">step-level <button class="term" data-c="middleware">stack</button> around the dispatch</span></summary>
<div class="panel"><p>A stack wrapped around this one dispatch, in addition to whatever the Flow-level stack already wraps.</p><span class="spec"><b>spec:</b> Middleware mechanics § Where middleware attaches</span></div>
</details></div>

<div class="row" data-field="CallStep.Catch"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Catch</span> <span class="ty">[]CatchClause</span> <span class="gloss">the <button class="term" data-c="routing">failure edges</button></span></summary>
<div class="panel"><p>Conditional failure edges: clauses tried in order, the first whose matcher takes the failure routes it. Only the dispatching actions — Call and Gather — carry <code>Catch</code>. The clause's own shape is at the bottom of this tree.</p><span class="spec"><b>spec:</b> Step mechanics § Failures and catch</span></div>
</details></div>

<div class="row" data-field="CallStep.Next"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Next</span> <span class="ty">Name</span> <span class="gloss">the <button class="term" data-c="routing">success edge</button></span></div>
</section>

<!-- ============ GatherStep ============ -->
<section>
<div class="row" data-type="GatherStep"><details class="node"><summary><span class="nm">GatherStep</span> <span class="ty">Step</span> <span class="gloss">many <button class="term" data-c="dispatch">dispatches</button> concurrently, Results gathered</span></summary>
<div class="panel"><p>Fan dispatches out, gather the Results back in. Exactly one of its two forms is present: <em>iterate</em> (a map over data) or <em>scatter</em> (a fixed set of jobs launched together).</p><span class="spec"><b>spec:</b> Step actions § Gather</span></div>
</details></div>

<div class="row" data-field="GatherStep.Over GatherStep.Call"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Over + Call</span> <span class="ty">*OrExpr[[]*Value] + *Call</span> <span class="gloss">iterate form: one dispatch per element of Over</span></summary>
<div class="panel"><p><code>Over</code> (<code>*OrExpr[[]*Value]</code>) produces a collection, and the <code>Call</code> template is dispatched once per element — a map over data.</p><span class="spec"><b>spec:</b> Step actions § The iterate form</span></div>
</details></div>

<div class="row" data-field="GatherStep.Calls"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Calls</span> <span class="ty">[]*Call</span> <span class="gloss">scatter form: one dispatch per listed call</span></summary>
<div class="panel"><p>An explicit, non-empty list of call objects, each dispatched once — a fixed set of jobs launched together. Exactly one of the two forms is present; bind guarantees it.</p><span class="spec"><b>spec:</b> Step actions § The scatter form</span></div>
</details></div>

<div class="row" data-field="GatherStep.Completion"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Completion</span> <span class="ty">*Completion</span> <span class="gloss">success criterion for the whole fan-out</span></summary>
<div class="panel"><p>What the fan-out must achieve for the Gather to succeed — and the <em>only</em> path by which individual dispatch failures affect the step. A failing dispatch does not fail the Gather; the Gather fails when the criterion becomes unachievable. Nil means every dispatch must succeed.</p><span class="spec"><b>spec:</b> Step actions § completion</span></div>
</details></div>

<div class="row" data-field="Completion.Successes"><span class="conn">│  </span><span class="conn">├─ </span><details class="node"><summary><span class="nm">Successes</span> <span class="ty">OrExpr[int64]</span> <span class="gloss">how many dispatches must succeed</span></summary>
<div class="panel"><p>Evaluated once, after dispatches are enumerated, so the dispatch count is available to the expression. <code>1</code> means "first success wins."</p></div>
</details></div>

<div class="row" data-field="Completion.Wait"><span class="conn">│  </span><span class="conn">└─ </span><details class="node"><summary><span class="nm">Wait</span> <span class="ty">*bool</span> <span class="gloss">let pending dispatches finish? nil = yes</span></summary>
<div class="panel"><p>Once the outcome is determined — criterion met, or unreachable — do pending dispatches still run to completion? Nil means the default, <code>true</code>. <code>false</code> cancels what is in flight and skips what never started.</p></div>
</details></div>

<div class="row" data-field="GatherStep.Concurrency"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Concurrency</span> <span class="ty">*ConcurrencyCap</span> <span class="gloss">max dispatches in flight; nil = unlimited</span></summary>
<div class="panel"><p>Nil means no cap — the field was absent, or an explicit null, which bind treats identically.</p></div>
</details></div>

<div class="row" data-field="ConcurrencyCap.Value"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Value</span> <span class="ty">int64</span> <span class="gloss">the cap itself</span></div>

<div class="row" data-field="GatherStep.Output"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Output</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: gathered values → emitted value</span></summary>
<div class="panel"><p>Builds the value handed to the successor on success. Nil means the default: the succeeded dispatches' values, in order.</p><span class="spec"><b>spec:</b> Step mechanics § Data flow</span></div>
</details></div>

<div class="row" data-field="GatherStep.Assign"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> into vars on success</span></summary>
<div class="panel"><p>The step's capture block: variable name → expression, all evaluated against the variable state from <em>before</em> the step, taking effect on successful exit.</p><span class="spec"><b>spec:</b> Step mechanics § Variables: assign</span></div>
</details></div>

<div class="row" data-field="GatherStep.Catch"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Catch</span> <span class="ty">[]CatchClause</span> <span class="gloss">the <button class="term" data-c="routing">failure edges</button></span></summary>
<div class="panel"><p>Conditional failure edges: clauses tried in order, the first whose matcher takes the failure routes it. The clause's own shape is at the bottom of this tree.</p><span class="spec"><b>spec:</b> Step mechanics § Failures and catch</span></div>
</details></div>

<div class="row" data-field="GatherStep.Next"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Next</span> <span class="ty">Name</span> <span class="gloss">the <button class="term" data-c="routing">success edge</button></span></div>
</section>

<!-- ============ MatchStep ============ -->
<section>
<div class="row" data-type="MatchStep"><details class="node"><summary><span class="nm">MatchStep</span> <span class="ty">Step</span> <span class="gloss">route by testing predicates in order</span></summary>
<div class="panel"><p>Routes to one of several successors by testing predicates, in order, against a single shaped value. The step does no work beyond selecting.</p><span class="spec"><b>spec:</b> Step actions § Match</span></div>
</details></div>

<div class="row" data-field="MatchStep.Input"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Input</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: the value the predicates test</span></summary>
<div class="panel"><p>Evaluated once; every clause reads it as <code>match.input</code>. Nil means test the value as received.</p></div>
</details></div>

<div class="row" data-field="MatchStep.Cases"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Cases</span> <span class="ty">[]MatchCase</span> <span class="gloss">tried in order, first match wins</span></summary>
<div class="panel"><p>The first clause whose <code>When</code> holds is selected, and no later predicate even evaluates.</p></div>
</details></div>

<div class="row" data-field="MatchCase.When"><span class="conn">│  </span><span class="conn">├─ </span><details class="node"><summary><span class="nm">When</span> <span class="ty">Expr</span> <span class="gloss">the predicate (always an expression)</span></summary>
<div class="panel"><p>Match is the one action with no expression-free form: every case requires a <code>When</code>.</p></div>
</details></div>

<div class="row" data-field="MatchCase.Output"><span class="conn">│  </span><span class="conn">├─ </span><details class="node"><summary><span class="nm">Output</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: what the selected clause emits</span></summary>
<div class="panel"><p>Shaping belongs to the clause rather than the step, because each destination may deserve a differently shaped value. Nil means <code>match.input</code> passes through.</p></div>
</details></div>

<div class="row" data-field="MatchCase.Assign"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> into vars when selected</span></div>

<div class="row" data-field="MatchCase.Next"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Next</span> <span class="ty">Name</span> <span class="gloss">the successor this clause routes to</span></div>

<div class="row" data-field="MatchStep.Default"><span class="conn">└─ </span><details class="node"><summary><span class="nm">Default</span> <span class="ty">MatchDefault</span> <span class="gloss">required fallback: a case without When</span></summary>
<div class="panel"><p>Selected when no case matched. Required so the step always routes somewhere.</p></div>
</details></div>

<div class="row" data-field="MatchDefault.Output"><span class="conn">   </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Output</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: what the fallback emits; nil = match.input</span></div>
<div class="row" data-field="MatchDefault.Assign"><span class="conn">   </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> into vars when selected</span></div>
<div class="row" data-field="MatchDefault.Next"><span class="conn">   </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Next</span> <span class="ty">Name</span> <span class="gloss">the successor the fallback routes to</span></div>
</section>

<!-- ============ PassStep ============ -->
<section>
<div class="row" data-type="PassStep"><details class="node"><summary><span class="nm">PassStep</span> <span class="ty">Step</span> <span class="gloss">no action: shape, capture, transition</span></summary>
<div class="panel"><p>No action work at all: shape a value (<code>Output</code>), capture variables (<code>Assign</code>), transition (<code>Next</code>). The no-op step that exists for its side channels.</p><span class="spec"><b>spec:</b> Step actions § Pass</span></div>
</details></div>
<div class="row" data-field="PassStep.Output"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Output</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: what the step emits; nil = pass through</span></div>
<div class="row" data-field="PassStep.Assign"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> into vars</span></div>
<div class="row" data-field="PassStep.Next"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Next</span> <span class="ty">Name</span> <span class="gloss">the <button class="term" data-c="routing">success edge</button></span></div>
</section>

<!-- ============ SleepStep ============ -->
<section>
<div class="row" data-type="SleepStep"><details class="node"><summary><span class="nm">SleepStep</span> <span class="ty">Step</span> <span class="gloss">pause the Flow</span></summary>
<div class="panel"><p>A pure pause: no shaping, no capture, no middleware, no catch — the flowing value passes through untouched. A moment already reached completes the step immediately rather than failing it.</p><span class="spec"><b>spec:</b> Step actions § Sleep</span></div>
</details></div>
<div class="row" data-field="SleepStep.For"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">For</span> <span class="ty">*OrExpr[string]</span> <span class="gloss">a duration (ISO 8601) — exactly one of For/Until</span></div>
<div class="row" data-field="SleepStep.Until"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Until</span> <span class="ty">*OrExpr[string]</span> <span class="gloss">an instant (RFC 3339) — exactly one of For/Until</span></div>
<div class="row" data-field="SleepStep.Next"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Next</span> <span class="ty">Name</span> <span class="gloss">the <button class="term" data-c="routing">success edge</button></span></div>
</section>

<!-- ============ ReturnStep ============ -->
<section>
<div class="row" data-type="ReturnStep"><details class="node"><summary><span class="nm">ReturnStep</span> <span class="ty">Step</span> <span class="gloss"><button class="term" data-c="terminal">terminal</button>: end the Flow with success</span></summary>
<div class="panel"><p>The Flow ends here, producing a success <button class="term" data-c="result">Result</button>; <code>Value</code> shapes what that Result carries. Nil means return the value that reached the step.</p><span class="spec"><b>spec:</b> Step actions § Return</span></div>
</details></div>
<div class="row" data-field="ReturnStep.Value"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Value</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: what the success Result carries</span></div>
</section>

<!-- ============ RaiseStep ============ -->
<section>
<div class="row" data-type="RaiseStep"><details class="node"><summary><span class="nm">RaiseStep</span> <span class="ty">Step</span> <span class="gloss"><button class="term" data-c="terminal">terminal</button>: end the Flow with failure</span></summary>
<div class="panel"><p>The Flow ends here, producing a failure <button class="term" data-c="result">Result</button>. <code>Result</code> describes the failure to construct; <strong>nil means a bare re-raise</strong> — re-emit the failure currently being handled, unchanged. The natural terminal for a handler path that dealt with what it could and propagates the rest.</p><span class="spec"><b>spec:</b> Step actions § Raise</span></div>
</details></div>

<div class="row" data-field="RaiseStep.Result"><span class="conn">└─ </span><details class="node"><summary><span class="nm">Result</span> <span class="ty">*RaiseResult</span> <span class="gloss">the failure to construct; nil = bare re-raise</span></summary>
<div class="panel"><p>The authorable fields of the failure envelope. A Raise constructs its envelope from nothing, so nothing can be inherited — which is why <code>Code</code> is the one required member here but inheritable in a middleware <code>onFailure</code>.</p></div>
</details></div>

<div class="row" data-field="RaiseResult.Type"><span class="conn">   </span><span class="conn">├─ </span><details class="node"><summary><span class="nm">Type</span> <span class="ty">*OrExpr[string]</span> <span class="gloss">failure type; nil defaults to "error"</span></summary>
<div class="panel"><p><code>*OrExpr[string]</code>. Must be a non-success type — a Raise cannot end a Flow in success.</p></div>
</details></div>

<div class="row" data-field="RaiseResult.Code"><span class="conn">   </span><span class="conn">├─ </span><details class="node"><summary><span class="nm">Code</span> <span class="ty">OrExpr[string]</span> <span class="gloss">the failure code (required)</span></summary>
<div class="panel"><p>The one required member. A Raise constructs from nothing, so the code must be written.</p></div>
</details></div>


<div class="row" data-field="RaiseResult.Message"><span class="conn">   </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Message</span> <span class="ty">*OrExpr[string]</span> <span class="gloss">the human-readable message; absent = absent from envelope</span></div>

<div class="row" data-field="RaiseResult.Details"><span class="conn">   </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Details</span> <span class="ty">*Value</span> <span class="gloss">the structured payload; absent = absent from envelope</span></div>

<div class="row" data-field="RaiseResult.Retryable"><span class="conn">   </span><span class="conn">├─ </span><details class="node"><summary><span class="nm">Retryable</span> <span class="ty">*RetryableField</span> <span class="gloss">bool | explicit null | expression</span></summary>
<div class="panel"><p>Whether the failure is worth retrying. The explicit null is an assertion of "no assertion" — meaningful data, distinct from omitting the field, which is why the type has three arms rather than two.</p></div>
</details></div>

<div class="row" data-field="RaiseResult.Previous"><span class="conn">   </span><span class="conn">└─ </span><details class="node"><summary><span class="nm">Previous</span> <span class="ty">*PreviousField</span> <span class="gloss">chain link | explicit null (sever) | expr</span></summary>
<div class="panel"><p>Overrides the engine's failure-chain link. When a failure is constructed while another is active, the engine normally links the old one as the new one's <code>previous</code>; writing this field overrides that — most often with an explicit null, severing the chain deliberately.</p></div>
</details></div>
</section>

</div><!-- /.tree -->

### Shared shapes

Referenced from the steps above: the call object both dispatching actions
carry, the failure edge, and one entry of a middleware stack.

<div class="tree">

<!-- ============ Call ============ -->
<section class="root">
<div class="row" data-type="Call"><details class="node"><summary><span class="nm">Call</span> <span class="ty">struct</span> <span class="gloss">a request to run a target and get a <button class="term" data-c="result">Result</button></span></summary>
<div class="panel"><p>Appears under a CallStep and inside both Gather forms. Its three data axes — the target's <code>parameters</code> schema, the call's <code>with</code> arguments, the <code>input</code> payload — are never interchangeable.</p><span class="spec"><b>spec:</b> The call interface § The three axes</span></div>
</details></div>

<div class="row" data-field="Call.Provider"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Provider</span> <span class="ty">*ProviderRef</span> <span class="gloss">the provider target — exactly one of Provider/Flow</span></summary>
<div class="panel"><p>The provider URI as written — checked as a well-formed name, deliberately unresolved: whether it names a real provider is the validation layer's question.</p></div>
</details></div>

<div class="row" data-field="ProviderRef.URI"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">URI</span> <span class="ty">string</span> <span class="gloss">the URI text, unresolved</span></div>

<div class="row" data-field="Call.Flow"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Flow</span> <span class="ty">*FlowTarget</span> <span class="gloss">the flow target — exactly one of Provider/Flow</span></summary>
<div class="panel"><p><code>FlowTarget</code> is itself an exactly-one pair: a bare name referencing a declared subflow, or a whole Flow written inline.</p></div>
</details></div>

<div class="row" data-field="FlowTarget.Name"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Name</span> <span class="ty">*Name</span> <span class="gloss">a subflow name, resolved up the enclosing chain — exactly one of Name/Inline</span></div>
<div class="row" data-field="FlowTarget.Inline"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Inline</span> <span class="ty">*Flow</span> <span class="gloss">a whole Flow written at the point of use</span></div>

<div class="row" data-field="Call.Input"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Input</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: the payload sent to the target</span></summary>
<div class="panel"><p>A second data channel, separate from the arguments — think stdin next to argv. Configuration goes in <code>with</code>; the working payload goes in <code>input</code>. Nil means pass the call's inbound payload through.</p></div>
</details></div>

<div class="row" data-field="Call.With"><span class="conn">├─ </span><details class="node"><summary><span class="nm">With</span> <span class="ty">*OrExpr[[]Member]</span> <span class="gloss">arguments, checked against the target's parameter schema</span></summary>
<div class="panel"><p><code>with</code> is to <code>parameters</code> as arguments are to a function signature; validated at dispatch. Either a literal object whose member values may each be an expression, or one whole-value expression producing the entire arguments object.</p></div>
</details></div>


<div class="row" data-field="Call.OnSuccess"><span class="conn">├─ </span><details class="node"><summary><span class="nm">OnSuccess</span> <span class="ty">*SuccessArm</span> <span class="gloss"><button class="term" data-c="result">arm</button> on success: reshape + <button class="term" data-c="capture">capture</button></span></summary>
<div class="panel"><p>Runs when the Result settles as success, at the call boundary, while the dispatch's context is still alive.</p></div>
</details></div>

<div class="row" data-field="SuccessArm.Value"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Value</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: reshapes the returned value; nil = pass through</span></div>
<div class="row" data-field="SuccessArm.Assign"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> while the dispatch's context is alive</span></div>

<div class="row" data-field="Call.OnFailure"><span class="conn">└─ </span><details class="node"><summary><span class="nm">OnFailure</span> <span class="ty">*FailureArm</span> <span class="gloss"><button class="term" data-c="result">arm</button> on failure: <button class="term" data-c="capture">capture</button> only</span></summary>
<div class="panel"><p>Runs on any non-success Result. Capture only: the failure envelope ascends exactly as produced — transforming a failure belongs to middleware <code>onFailure</code> or a Raise. The arm exists because it is the <em>only</em> seam where the failed dispatch's context (the provider's metadata, the call's timing, a failed subflow's variables) is still reachable; what it doesn't capture is gone.</p></div>
</details></div>

<div class="row" data-field="FailureArm.Assign"><span class="conn">   </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> from the failed dispatch's live context</span></div>
</section>

<!-- ============ CatchClause ============ -->
<section class="root">
<div class="row" data-type="CatchClause"><details class="node"><summary><span class="nm">CatchClause</span> <span class="ty">struct</span> <span class="gloss">one <button class="term" data-c="routing">failure edge</button></span></summary>
<div class="panel"><p>When the step's failure matches, control transitions to <code>Next</code> the same way <code>next</code> transitions on success.</p><span class="spec"><b>spec:</b> Step mechanics § Failures and catch</span></div>
</details></div>

<div class="row" data-field="CatchClause.Match"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Match</span> <span class="ty">FailureMatcher</span> <span class="gloss">which failures this clause takes</span></summary>
<div class="panel"><p>At least one member is present (bind guarantees it), and every present member must match — the members conjoin.</p></div>
</details></div>

<div class="row" data-field="FailureMatcher.Codes"><span class="conn">│  </span><span class="conn">├─ </span><details class="node"><summary><span class="nm">Codes</span> <span class="ty">[]StringLit</span> <span class="gloss">code patterns ("X.Y", "X.*", "*")</span></summary>
<div class="panel"><p>Patterns over the failure's <code>code</code>: exact (<code>"Prefix.Code"</code>), prefix (<code>"Prefix.*"</code>), or match-anything (<code>"*"</code>).</p></div>
</details></div>

<div class="row" data-field="FailureMatcher.Types"><span class="conn">│  </span><span class="conn">├─ </span><details class="node"><summary><span class="nm">Types</span> <span class="ty">[]StringLit</span> <span class="gloss">non-success <button class="term" data-c="result">Result</button> types</span></summary>
<div class="panel"><p><code>error</code>, <code>timeout</code>, ...; the spec's five are published as <code>parser.ResultType*</code> constants.</p></div>
</details></div>

<div class="row" data-field="FailureMatcher.Retryable"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Retryable</span> <span class="ty">*bool</span> <span class="gloss">the envelope's explicit retryable flag</span></div>

<div class="row" data-field="CatchClause.Output"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Output</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: what the handler step receives</span></summary>
<div class="panel"><p>Nil means the step's <em>received</em> value passes through — the step failed, so no success value exists to pass.</p></div>
</details></div>

<div class="row" data-field="CatchClause.Assign"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> on the handler path</span></div>
<div class="row" data-field="CatchClause.Next"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Next</span> <span class="ty">Name</span> <span class="gloss">where handling continues</span></div>
</section>

<!-- ============ MiddlewareEntry ============ -->
<section class="root">
<div class="row" data-type="MiddlewareEntry"><details class="node"><summary><span class="nm">MiddlewareEntry</span> <span class="ty">struct</span> <span class="gloss">one middleware in a <button class="term" data-c="middleware">stack</button></span></summary>
<div class="panel"><p>Names a middleware provider and configures, <button class="term" data-c="phase">phase by phase</button>, how it participates. An absent phase is equivalent to an empty one.</p><span class="spec"><b>spec:</b> Middleware mechanics</span></div>
</details></div>

<div class="row" data-field="MiddlewareEntry.Provider"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Provider</span> <span class="ty">ProviderRef</span></div>

<div class="row" data-field="MiddlewareEntry.OnEntry"><span class="conn">├─ </span><details class="node"><summary><span class="nm">OnEntry</span> <span class="ty">*PhaseOnEntry</span> <span class="gloss">on the way in: gate, configure, shape down</span></summary>
<div class="panel"><p>Runs once, before anything inside the entry. Its <code>Output</code> shapes the value passed down to the next inner entry — or into the wrapped operation at the innermost.</p></div>
</details></div>

<div class="row" data-field="PhaseOnEntry.When"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">When</span> <span class="ty">*OrExpr[bool]</span> <span class="gloss">gates the phase; nil = always runs</span></div>
<div class="row" data-field="PhaseOnEntry.With"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">With</span> <span class="ty">*OrExpr[[]Member]</span> <span class="gloss">arguments, validated against the phase's contract</span></div>
<div class="row" data-field="PhaseOnEntry.Assign"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> against the phase's bindings</span></div>
<div class="row" data-field="PhaseOnEntry.Output"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Output</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: the value passed inward; nil = pass through</span></div>

<div class="row" data-field="MiddlewareEntry.OnSuccess"><span class="conn">├─ </span><details class="node"><summary><span class="nm">OnSuccess</span> <span class="ty">*PhaseOnSuccess</span> <span class="gloss">on ascent, success: reshape the value</span></summary>
<div class="panel"><p>Its <code>Value</code> produces the success value this entry emits upward; the next outer entry sees the reshaped value.</p></div>
</details></div>

<div class="row" data-field="PhaseOnSuccess.When"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">When</span> <span class="ty">*OrExpr[bool]</span> <span class="gloss">gates the phase; nil = always runs</span></div>
<div class="row" data-field="PhaseOnSuccess.With"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">With</span> <span class="ty">*OrExpr[[]Member]</span> <span class="gloss">arguments, validated against the phase's contract</span></div>
<div class="row" data-field="PhaseOnSuccess.Assign"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> against the phase's bindings</span></div>
<div class="row" data-field="PhaseOnSuccess.Value"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Value</span> <span class="ty">*Value</span> <span class="gloss"><button class="term" data-c="shaping">shaping</button>: the value emitted upward; nil = pass through</span></div>

<div class="row" data-field="MiddlewareEntry.OnFailure"><span class="conn">├─ </span><details class="node"><summary><span class="nm">OnFailure</span> <span class="ty">*PhaseOnFailure</span> <span class="gloss">on ascent, failure: rewrite the envelope</span></summary>
<div class="panel"><p>Carries the authorable envelope fields. Writing <em>any</em> of them constructs a <em>new</em> failure superseding the rising one, with unwritten fields inherited from it and the superseded failure chained as <code>previous</code>. Same field shapes as <code>RaiseResult</code>; opposite inheritance rule — a Raise inherits nothing.</p></div>
</details></div>

<div class="row" data-field="PhaseOnFailure.When"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">When</span> <span class="ty">*OrExpr[bool]</span> <span class="gloss">gates the phase; nil = always runs</span></div>
<div class="row" data-field="PhaseOnFailure.With"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">With</span> <span class="ty">*OrExpr[[]Member]</span> <span class="gloss">arguments, validated against the phase's contract</span></div>
<div class="row" data-field="PhaseOnFailure.Assign"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> against the phase's bindings</span></div>
<div class="row" data-field="PhaseOnFailure.Type"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Type</span> <span class="ty">*OrExpr[string]</span> <span class="gloss">a non-success Result type; unwritten = inherited</span></div>
<div class="row" data-field="PhaseOnFailure.Code"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Code</span> <span class="ty">*OrExpr[string]</span> <span class="gloss">the failure code; unwritten = inherited</span></div>
<div class="row" data-field="PhaseOnFailure.Message"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Message</span> <span class="ty">*OrExpr[string]</span> <span class="gloss">the human-readable message; unwritten = inherited</span></div>
<div class="row" data-field="PhaseOnFailure.Details"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Details</span> <span class="ty">*Value</span> <span class="gloss">the structured payload; unwritten = inherited</span></div>
<div class="row" data-field="PhaseOnFailure.Retryable"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Retryable</span> <span class="ty">*RetryableField</span> <span class="gloss">bool | explicit null | expression</span></div>
<div class="row" data-field="PhaseOnFailure.Previous"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Previous</span> <span class="ty">*PreviousField</span> <span class="gloss">chain link | explicit null (sever) | expr</span></div>

<div class="row" data-field="MiddlewareEntry.OnAlways"><span class="conn">└─ </span><details class="node"><summary><span class="nm">OnAlways</span> <span class="ty">*PhaseOnAlways</span> <span class="gloss">on every exit: cleanup, cannot reshape</span></summary>
<div class="panel"><p>Runs on every exit — success, failure, cancellation — even when the entry's other phases failed. In exchange it has no shaping key: the rising Result passes through it unchanged.</p></div>
</details></div>

<div class="row" data-field="PhaseOnAlways.When"><span class="conn">   </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">When</span> <span class="ty">*OrExpr[bool]</span> <span class="gloss">gates the phase; nil = always runs</span></div>
<div class="row" data-field="PhaseOnAlways.With"><span class="conn">   </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">With</span> <span class="ty">*OrExpr[[]Member]</span> <span class="gloss">arguments, validated against the phase's contract</span></div>
<div class="row" data-field="PhaseOnAlways.Assign"><span class="conn">   </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Assign</span> <span class="ty">[]Member</span> <span class="gloss"><button class="term" data-c="capture">capture</button> against the phase's bindings</span></div>
</section>

</div><!-- /.tree -->

> [!IMPORTANT]
> Guaranteed on any non-nil Flow
>
> - **Exactly-one pairs:** `Call.Provider`/`Call.Flow`,
>   `FlowTarget.Name`/`.Inline`, Gather's iterate/scatter forms,
>   `SleepStep.For`/`.Until`, every `OrExpr`'s `Literal`/`Expr`, and the three
>   arms of `RetryableField`/`PreviousField`.
> - **Every `Name` resolves** — entrypoints and `next` to a step, flow targets
>   up the enclosing chain — and the flow-reference graph is acyclic.
> - **`Step` is a closed sum** of seven variants; dispatch by type switch.
> - **Nothing is defaulted:** nil in the AST means the document supplied no
>   value — the member was absent, or spelled an explicit `null` or empty
>   collection, which bind identically. Passthrough defaults quoted here are
>   what a _runtime_ assumes.
> - **Deliberately not modeled:** expression bodies are opaque text, provider
>   URIs are unresolved, shaping fields are raw `*Value` trees.

## The Provider AST

Returned by `parser.ParseProvider`: the Provider document and its two kind
arms. Provider documents carry no expression fields — embedded schemas are
retained as raw positioned value trees whose string leaves stay plain strings.
Every shape except the two kind arms also carries `Pos Position`, not repeated
in the rows.

<div class="tree">

<!-- ============ Provider ============ -->
<section class="root">
<div class="row" data-type="Provider"><details class="node"><summary><span class="nm">Provider</span> <span class="ty">struct</span> <span class="gloss">one provider definition document</span></summary>
<div class="panel"><p>The interchange form of a catalog entry, for either provider kind. Fields the kinds share are flat on this struct; the kind-specific fields live in the two arms at the bottom.</p><span class="spec"><b>spec:</b> Providers § The provider catalog</span></div>
</details></div>

<div class="row" data-field="Provider.Schema"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Schema</span> <span class="ty">string</span> <span class="gloss">the document's $schema URI</span></div>

<div class="row" data-field="Provider.URI"><span class="conn">├─ </span><details class="node"><summary><span class="nm">URI</span> <span class="ty">StringLit</span> <span class="gloss">the provider's own URI — its identity, and the kind discriminator</span></summary>
<div class="panel"><p>The URI a Flow's <code>provider</code> fields cite. Its type segment names the provider kind, which is what selects between the <code>Call</code> and <code>Middleware</code> arms below.</p><span class="spec"><b>spec:</b> Providers § Provider URIs</span></div>
</details></div>

<div class="row" data-field="Provider.CodePrefix"><span class="conn">├─ </span><details class="node"><summary><span class="nm">CodePrefix</span> <span class="ty">StringLit</span> <span class="gloss">its identity segment in failure codes</span></summary>
<div class="panel"><p>The segment a failure code carries in the taxonomy <code>Provider.&lt;Kind&gt;.&lt;codePrefix&gt;.&lt;Code&gt;</code> — how a code names which provider emitted it.</p><span class="spec"><b>spec:</b> Providers § The failure catalog</span></div>
</details></div>

<div class="row" data-field="Provider.Description"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Description</span> <span class="ty">string</span> <span class="gloss">the human-readable description</span></div>

<div class="row" data-field="Provider.Metadata"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Metadata</span> <span class="ty">*Value</span> <span class="gloss">the metadata schema; or nil</span></summary>
<div class="panel"><p>Describes the metadata the provider attaches to its Results (what <code>provider.metadata</code> exposes in a call's arms). Held as a raw positioned tree, like every embedded schema.</p></div>
</details></div>

<div class="row" data-field="Provider.FailureCatalog"><span class="conn">├─ </span><details class="node"><summary><span class="nm">FailureCatalog</span> <span class="ty">FailureCatalog</span> <span class="gloss">the codes it can emit under its prefix</span></summary>
<div class="panel"><p>What can go wrong, as data: the code space this provider claims under its kind and <code>CodePrefix</code> segments. Tooling can check a Flow's <code>catch</code> matchers against it.</p><span class="spec"><b>spec:</b> Providers § The failure catalog</span></div>
</details></div>

<div class="row" data-field="FailureCatalog.Closed"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Closed</span> <span class="ty">[]StringLit</span> <span class="gloss">codes declared exactly</span></div>
<div class="row" data-field="FailureCatalog.Open"><span class="conn">│  </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Open</span> <span class="ty">[]StringLit</span> <span class="gloss">sub-prefixes where undeclared codes may appear</span></div>

<div class="row" data-field="Provider.Call Provider.Middleware"><span class="conn">└─ </span><details class="node"><summary><span class="nm">Call / Middleware</span> <span class="ty">*CallProvider / *MiddlewareProvider</span> <span class="gloss">the kind arm — exactly one is set</span></summary>
<div class="panel"><p>Discriminated by the URI's type segment; the bind pass guarantees exactly one arm is present. Each arm carries only what its kind declares.</p></div>
</details></div>
</section>

</div><!-- /.tree -->

### The kind arms

Exactly one arm is present, selected by the URI's type segment; the
middleware arm's per-phase contract, `PhaseContract`, follows it.

<div class="tree">

<!-- ============ CallProvider ============ -->
<section>
<div class="row" data-type="CallProvider"><details class="node"><summary><span class="nm">CallProvider</span> <span class="ty">struct</span> <span class="gloss">the call-provider arm: a dispatch target</span></summary>
<div class="panel"><p>A call provider is something a <button class="term" data-c="dispatch">dispatch</button> can target: it accepts a payload and arguments and settles a <button class="term" data-c="result">Result</button>.</p><span class="spec"><b>spec:</b> Call providers § The dispatch contract</span></div>
</details></div>

<div class="row" data-field="CallProvider.Parameters"><span class="conn">└─ </span><details class="node"><summary><span class="nm">Parameters</span> <span class="ty">*Value</span> <span class="gloss">the schema a call's <code>with</code> is validated against</span></summary>
<div class="panel"><p>The provider's signature: the <code>with</code> arguments of every Call targeting this provider are validated against it at dispatch. Never nil: a call-provider definition requires it.</p><span class="spec"><b>spec:</b> Call providers § What a call provider declares</span></div>
</details></div>
</section>

<!-- ============ MiddlewareProvider ============ -->
<section>
<div class="row" data-type="MiddlewareProvider"><details class="node"><summary><span class="nm">MiddlewareProvider</span> <span class="ty">struct</span> <span class="gloss">the middleware-provider arm: a <button class="term" data-c="middleware">stack</button> participant</span></summary>
<div class="panel"><p>What a middleware declares about itself: where its stack may attach, and what it does at each of the four phases.</p><span class="spec"><b>spec:</b> Middleware providers § What a middleware provider declares</span></div>
</details></div>

<div class="row" data-field="MiddlewareProvider.Attachment"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Attachment</span> <span class="ty">[]StringLit</span> <span class="gloss">"step", "flow", or both — each at most once</span></summary>
<div class="panel"><p>The levels a <code>middleware</code> entry citing this provider may appear at. The two values are published as <code>parser.AttachmentStep</code> and <code>parser.AttachmentFlow</code>; bind validates against the same set.</p></div>
</details></div>

<div class="row" data-field="MiddlewareProvider.Phases"><span class="conn">└─ </span><details class="node"><summary><span class="nm">Phases</span> <span class="ty">PhaseContracts</span> <span class="gloss">the per-phase contract</span></summary>
<div class="panel"><p>One optional <code>*PhaseContract</code> per <button class="term" data-c="phase">phase</button>. A nil phase declares no action there; at least one is set. These are the same four phases a Flow's <code>MiddlewareEntry</code> configures — the entry writes arguments against the contract declared here.</p></div>
</details></div>

<div class="row" data-field="PhaseContracts.OnEntry"><span class="conn">   </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">OnEntry</span> <span class="ty">*PhaseContract</span> <span class="gloss">the way-in contract; nil = no action there</span></div>
<div class="row" data-field="PhaseContracts.OnSuccess"><span class="conn">   </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">OnSuccess</span> <span class="ty">*PhaseContract</span> <span class="gloss">the success-ascent contract; nil = no action there</span></div>
<div class="row" data-field="PhaseContracts.OnFailure"><span class="conn">   </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">OnFailure</span> <span class="ty">*PhaseContract</span> <span class="gloss">the failure-ascent contract; nil = no action there</span></div>
<div class="row" data-field="PhaseContracts.OnAlways"><span class="conn">   </span><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">OnAlways</span> <span class="ty">*PhaseContract</span> <span class="gloss">the every-exit contract; nil = no action there</span></div>
</section>

<!-- ============ PhaseContract ============ -->
<section class="root">
<div class="row" data-type="PhaseContract"><details class="node"><summary><span class="nm">PhaseContract</span> <span class="ty">struct</span> <span class="gloss">what the middleware declares for one <button class="term" data-c="phase">phase</button></span></summary>
<div class="panel"><p>The contract one phase publishes: the action it performs there, the parameters it takes, and which of those parameters are structural.</p><span class="spec"><b>spec:</b> Middleware providers § What a middleware provider declares</span></div>
</details></div>

<div class="row" data-field="PhaseContract.Action"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Action</span> <span class="ty">PhaseAction</span> <span class="gloss">the action the phase declares</span></summary>
<div class="panel"><p>What the phase does, classified by kind so tooling and runtimes know what the action may affect.</p></div>
</details></div>

<div class="row" data-field="PhaseAction.Kind"><span class="conn">│  </span><span class="conn">├─ </span><details class="node"><summary><span class="nm">Kind</span> <span class="ty">StringLit</span> <span class="gloss">side-effect | control | transform</span></summary>
<div class="panel"><p>The three kinds are published as <code>parser.ActionKindSideEffect</code>, <code>ActionKindControl</code>, and <code>ActionKindTransform</code>; bind validates against the same set.</p></div>
</details></div>

<div class="row" data-field="PhaseAction.Description"><span class="conn">│  </span><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Description</span> <span class="ty">string</span> <span class="gloss">the human-readable description</span></div>

<div class="row" data-field="PhaseAction.Acceptance"><span class="conn">│  </span><span class="conn">└─ </span><details class="node"><summary><span class="nm">Acceptance</span> <span class="ty">string</span> <span class="gloss">a control action's commit point; empty when absent</span></summary>
<div class="panel"><p>A concurrent control action states its acceptance semantics: the instant its bound (a timeout, a deadline) starts counting from. Only control actions declare one.</p><span class="spec"><b>spec:</b> Middleware providers § What a middleware provider declares</span></div>
</details></div>

<div class="row" data-field="PhaseContract.Parameters"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Parameters</span> <span class="ty">*Value</span> <span class="gloss">the schema the phase's <code>with</code> is validated against; or nil</span></summary>
<div class="panel"><p>The phase's signature, exactly as <code>CallProvider.Parameters</code> is the call provider's: a middleware entry's <code>With</code> for this phase is validated against it.</p></div>
</details></div>

<div class="row" data-field="PhaseContract.Structural"><span class="conn">└─ </span><details class="node"><summary><span class="nm">Structural</span> <span class="ty">[]StringLit</span> <span class="gloss">names of the parameters taken structurally</span></summary>
<div class="panel"><p>A structural parameter's value is a definition rather than a value: it is taken as written, not evaluated at the phase boundary — the seam that lets a middleware like retry accept a whole call object as an argument.</p><span class="spec"><b>spec:</b> Middleware providers § Structural parameters</span></div>
</details></div>
</section>

</div><!-- /.tree -->

> [!IMPORTANT]
> Guaranteed on any non-nil Provider
>
> - **Exactly one kind arm** — `Call` or `Middleware` — is set, discriminated
>   by the URI's type segment; the bind pass guarantees it.
> - **At least one phase** is set in a middleware provider's `Phases`, and each
>   `Attachment` value appears at most once.
> - **No expression fields:** embedded schemas (`Metadata`, `Parameters`) are
>   raw positioned `*Value` trees whose string leaves stay plain strings.
> - **Deliberately not modeled:** whether the definition agrees with any
>   platform's catalog — URI uniqueness, `codePrefix` uniqueness — is the
>   catalog's concern, not the parser's.

## Support types

The vocabulary both trees share: small carrier types that recur throughout the
type chips above, each meaning exactly the same thing in every slot. They are
defined once here and nowhere expanded in place — a chip like `*Value` or
`OrExpr[string]` always means one of these.

`Name`, `StringLit`, `Expr`, and `Value` each record their own `Pos Position`,
not repeated in the rows; `Member` records `NamePos` instead. The three
either/or shapes (`OrExpr`, `RetryableField`, `PreviousField`) carry no
position of their own — position lives on whichever arm is set.

<div class="tree">

<!-- ============ Value ============ -->
<section class="root" id="st-value">
<div class="row" data-type="Value"><details class="node"><summary><span class="nm">Value</span> <span class="ty">struct</span> <span class="gloss">a positioned JSON tree — any value, as written</span></summary>
<div class="panel"><p>The decode pass's representation of raw JSON. Exactly the field matching <code>Kind</code> is meaningful; the others hold their zero values. Every embedded schema and shaping chip in the trees above (<code>*Value</code>) is one of these, kept as data rather than interpreted. <code>Field(name)</code> looks a member up on an object.</p></div>
</details></div>

<div class="row" data-field="Value.Kind"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Kind</span> <span class="ty">Kind</span> <span class="gloss">null | boolean | number | string | object | array</span></summary>
<div class="panel"><p>Selects which payload field below is meaningful. The six kinds are published as the <code>parser.Kind*</code> constants.</p></div>
</details></div>

<div class="row" data-field="Value.Bool"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Bool</span> <span class="ty">bool</span> <span class="gloss">KindBool: the value</span></div>

<div class="row" data-field="Value.Number"><span class="conn">├─ </span><details class="node"><summary><span class="nm">Number</span> <span class="ty">string</span> <span class="gloss">KindNumber: the literal as written</span></summary>
<div class="panel"><p>Kept as source text, not converted to a float — no precision is lost between reading a definition and re-emitting it.</p></div>
</details></div>

<div class="row" data-field="Value.Str"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Str</span> <span class="ty">string</span> <span class="gloss">KindString: the decoded contents</span></div>

<div class="row" data-field="Value.Object"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Object</span> <span class="ty">[]Member</span> <span class="gloss">KindObject: members, document order</span></div>

<div class="row" data-field="Value.Array"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Array</span> <span class="ty">[]*Value</span> <span class="gloss">KindArray: elements, document order</span></div>
</section>

<!-- ============ Member ============ -->
<section class="root" id="st-member">
<div class="row" data-type="Member"><details class="node"><summary><span class="nm">Member</span> <span class="ty">struct</span> <span class="gloss">one name/value member of an object</span></summary>
<div class="panel"><p>The decode tree uses it for object members; the AST reuses it wherever a field is an <em>ordered</em> name-to-value map — <code>assign</code> blocks, a literal <code>with</code> object. Order is document order, preserved.</p></div>
</details></div>

<div class="row" data-field="Member.Name Member.NamePos"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Name, NamePos</span> <span class="ty">string, Position</span> <span class="gloss">the member's key</span></div>
<div class="row" data-field="Member.Value"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Value</span> <span class="ty">*Value</span> <span class="gloss">the member's value; its own Pos locates it</span></div>
</section>

<!-- ============ Expr ============ -->
<section class="root" id="st-expr">
<div class="row" data-type="Expr"><details class="node"><summary><span class="nm">Expr</span> <span class="ty">struct</span> <span class="gloss">one embedded expression, uninterpreted</span></summary>
<div class="panel"><p>A string whose entire content one delimiter pair set off. The parser records the body and stops: evaluating or analyzing expressions is a consumer's concern, which is why expression-bearing chips carry opaque text rather than a parsed tree.</p><span class="spec"><b>spec:</b> Expressions</span></div>
</details></div>

<div class="row" data-field="Expr.Body"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Body</span> <span class="ty">string</span> <span class="gloss">the text between the delimiters, whitespace-trimmed</span></div>
</section>

<!-- ============ OrExpr ============ -->
<section class="root" id="st-orexpr">
<div class="row" data-type="OrExpr"><details class="node"><summary><span class="nm">OrExpr[T]</span> <span class="ty">struct</span> <span class="gloss">a literal of type T, or an expression — exactly one</span></summary>
<div class="panel"><p>The shape of every field that accepts either a literal or an expression producing one. Exactly one arm is set; the bind pass guarantees the invariant.</p><span class="spec"><b>spec:</b> Expressions</span></div>
</details></div>

<div class="row" data-field="OrExpr.Literal"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Literal</span> <span class="ty">*T</span> <span class="gloss">the literal arm</span></div>
<div class="row" data-field="OrExpr.Expr"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Expr</span> <span class="ty">*Expr</span> <span class="gloss">the expression arm</span></div>
</section>

<!-- ============ Name ============ -->
<section class="root" id="st-name">
<div class="row" data-type="Name"><details class="node"><summary><span class="nm">Name</span> <span class="ty">struct</span> <span class="gloss">a reference to a named thing, as written</span></summary>
<div class="panel"><p>Refers to a Step (<code>entrypoint</code>, <code>next</code>) or a named Flow (a call's flow target) from the reference site. The check pass proved every one resolves.</p></div>
</details></div>

<div class="row" data-field="Name.Value"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Value</span> <span class="ty">string</span> <span class="gloss">the referenced name</span></div>
</section>

<!-- ============ StringLit ============ -->
<section class="root" id="st-stringlit">
<div class="row" data-type="StringLit"><details class="node"><summary><span class="nm">StringLit</span> <span class="ty">struct</span> <span class="gloss">a literal string with its own position</span></summary>
<div class="panel"><p>Used where a list of plain strings needs a position per entry — failure-matcher patterns, attachment levels, catalog codes — so a diagnostic can point at the one entry that is wrong.</p></div>
</details></div>

<div class="row" data-field="StringLit.Value"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Value</span> <span class="ty">string</span> <span class="gloss">the string contents</span></div>
</section>

<!-- ============ RetryableField ============ -->
<section class="root" id="st-retryablefield">
<div class="row" data-type="RetryableField"><details class="node"><summary><span class="nm">RetryableField</span> <span class="ty">struct</span> <span class="gloss">bool | explicit null | expression — exactly one arm</span></summary>
<div class="panel"><p>A <code>retryable</code> envelope field. The explicit null is an assertion of "no assertion" — meaningful data, distinct from omitting the field entirely, which is why the shape has three arms rather than two.</p></div>
</details></div>

<div class="row" data-field="RetryableField.Literal"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Literal</span> <span class="ty">*bool</span> <span class="gloss">a boolean literal</span></div>
<div class="row" data-field="RetryableField.Null"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Null</span> <span class="ty">bool</span> <span class="gloss">true for an explicit JSON null</span></div>
<div class="row" data-field="RetryableField.Expr"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Expr</span> <span class="ty">*Expr</span> <span class="gloss">an expression</span></div>
</section>

<!-- ============ PreviousField ============ -->
<section class="root" id="st-previousfield">
<div class="row" data-type="PreviousField"><details class="node"><summary><span class="nm">PreviousField</span> <span class="ty">struct</span> <span class="gloss">object | explicit null (sever) | expression — exactly one arm</span></summary>
<div class="panel"><p>A <code>previous</code> envelope field, overriding the engine's failure-chain link. The explicit null severs the chain deliberately — distinct from omitting the field, which keeps the engine's default link.</p></div>
</details></div>

<div class="row" data-field="PreviousField.Literal"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Literal</span> <span class="ty">*Value</span> <span class="gloss">a literal object</span></div>
<div class="row" data-field="PreviousField.Null"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Null</span> <span class="ty">bool</span> <span class="gloss">true for an explicit JSON null</span></div>
<div class="row" data-field="PreviousField.Expr"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Expr</span> <span class="ty">*Expr</span> <span class="gloss">an expression</span></div>
</section>

<!-- ============ Position ============ -->
<section class="root" id="st-position">
<div class="row" data-type="Position"><details class="node"><summary><span class="nm">Position</span> <span class="ty">struct</span> <span class="gloss">a location in the source document</span></summary>
<div class="panel"><p>What every <code>Pos</code> and <code>NamePos</code> field carries. <code>String()</code> renders the conventional <code>file:line:column</code> form.</p></div>
</details></div>

<div class="row" data-field="Position.Filename"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Filename</span> <span class="ty">string</span> <span class="gloss">the label supplied by WithFilename; may be empty</span></div>
<div class="row" data-field="Position.Offset"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Offset</span> <span class="ty">int</span> <span class="gloss">byte offset from the start of the document</span></div>
<div class="row" data-field="Position.Line"><span class="conn">├─ </span><span class="nm-lead"></span><span class="nm">Line</span> <span class="ty">int</span> <span class="gloss">1-based line</span></div>
<div class="row" data-field="Position.Column"><span class="conn">└─ </span><span class="nm-lead"></span><span class="nm">Column</span> <span class="ty">int</span> <span class="gloss">1-based column, counting bytes within the line</span></div>
</section>

</div><!-- /.tree -->

<!-- concept definitions -->
<dialog class="concept" id="c-shaping">
  <h3>shaping</h3>
  <p>Execution moves a single value along the step graph: each step receives the value its predecessor emitted and emits a value to its successor. A shaping field holds an expression that transforms the value crossing that boundary — <code>input</code> on the way in, <code>output</code>/<code>value</code> on the way out.</p>
  <pre><code>"input": "{{ step.input.order }}"</code></pre>
  <p>The predecessor emitted <code>{"order": ..., "customer": ...}</code>; this step cares only about the order, so <code>input</code> plucks it out. An absent shaping field means <strong>passthrough</strong>: the value crosses unchanged.</p>
  <form method="dialog"><button>close</button></form>
</dialog>

<dialog class="concept" id="c-capture">
  <h3>capture</h3>
  <p>A running Flow has a bag of named variables, <code>vars</code>, readable from any later expression. The one writer into it is <code>assign</code>: a map from variable name to expression, evaluated when its step, clause, or arm completes.</p>
  <pre><code>"assign": { "requestId": "{{ provider.metadata.requestId }}" }</code></pre>
  <p>Capture matters most where it is the <em>last chance</em>: a call's arms are the only seam where the dispatch's context is alive — what they don't capture into <code>vars</code> is gone.</p>
  <form method="dialog"><button>close</button></form>
</dialog>

<dialog class="concept" id="c-result">
  <h3>Result &amp; the two arms</h3>
  <p>Everything a call runs finishes by producing a <strong>Result</strong>: an envelope with a <code>type</code> (<code>success</code>, <code>error</code>, <code>timeout</code>, ...) and either a carried <code>value</code> (success) or the failure fields (<code>code</code>, <code>message</code>, <code>retryable</code>, ...).</p>
  <p>When the Result arrives — it <em>settles</em> — exactly one of the call's two <strong>arms</strong> runs, selected by the type: <code>onSuccess</code> on a success, <code>onFailure</code> on anything else. The arms are where the author reshapes the returned value and captures from the call's context while it is still alive.</p>
  <form method="dialog"><button>close</button></form>
</dialog>

<dialog class="concept" id="c-routing">
  <h3>routing: edges &amp; terminal steps</h3>
  <p>A step's <code>next</code> names its successor on success — the ordinary edge of the graph. <code>catch</code> supplies the failure edges: clauses, each a matcher plus a <code>next</code>, first match wins.</p>
  <p>Two actions have no outgoing edge: <code>Return</code> and <code>Raise</code> are terminal — they end the Flow with a success or failure Result. Every step must either transition or terminate; the bind pass enforces it.</p>
  <form method="dialog"><button>close</button></form>
</dialog>

<dialog class="concept" id="c-terminal">
  <h3>terminal step</h3>
  <p>A step with no outgoing edge: it ends the Flow instead of transitioning. <code>Return</code> ends it with a success Result; <code>Raise</code> with a failure Result. Every step must either transition (<code>next</code>, or a matching <code>catch</code>) or terminate; the bind pass enforces it.</p>
  <form method="dialog"><button>close</button></form>
</dialog>

<dialog class="concept" id="c-dispatch">
  <h3>dispatch</h3>
  <p>To dispatch is to actually launch a call at a target — a provider or a subflow — and await its Result. A <code>Call</code> step makes one dispatch; a <code>Gather</code> makes many concurrently, then gathers the Results.</p>
  <form method="dialog"><button>close</button></form>
</dialog>

<dialog class="concept" id="c-phase">
  <h3>phase</h3>
  <p>A middleware entry wraps a dispatch, so execution crosses its boundary twice: in, then out. A <strong>phase</strong> is one of the four fixed moments where the entry acts at a crossing: <code>onEntry</code> on the way in; <code>onSuccess</code> <em>or</em> <code>onFailure</code> on the way out, selected by the rising Result; <code>onAlways</code> after either, on every exit.</p>
  <pre><code>onEntry ▸ [dispatch] ▸ onSuccess|onFailure ▸ onAlways</code></pre>
  <p>The word names both sides of the contract: a Flow's middleware entry writes a <em>phase block</em> per phase (its configuration), and a middleware provider declares a <em>phase contract</em> per phase (what it does there).</p>
  <form method="dialog"><button>close</button></form>
</dialog>

<dialog class="concept" id="c-middleware">
  <h3>middleware stack</h3>
  <p>Middleware wraps dispatches the way HTTP middleware wraps handlers: a stack of entries, each naming a middleware provider, each able to act at four phases — <code>onEntry</code> on the way in, <code>onSuccess</code>/<code>onFailure</code> as the Result ascends, <code>onAlways</code> as guaranteed cleanup.</p>
  <p>A stack declared on the Flow wraps every step; a stack on a single step wraps just that step's dispatch.</p>
  <form method="dialog"><button>close</button></form>
</dialog>

<script>
  // Concept terms open their shared definition dialog. Terms sit inside
  // <summary> elements, so stop the click from also toggling the node.
  document.querySelectorAll('.ast-explorer .term[data-c]').forEach(t => {
    t.addEventListener('click', (e) => {
      e.preventDefault();
      e.stopPropagation();
      document.getElementById('c-' + t.dataset.c).showModal();
    });
  });

  // Type names inside type chips link to where the type is documented: its
  // own section (types with a data-type header row), or the row where its
  // fields are expanded (nested helper types). Both the name set and the
  // targets derive from the data-type/data-field attributes, so new rows and
  // types link without markup changes. A name is left plain inside the
  // section that documents it.
  const target = {}; // type name -> element the link jumps to
  document.querySelectorAll('.ast-explorer .row[data-type]').forEach(row => {
    const name = row.dataset.type;
    const section = row.closest('section');
    if (!section.id) section.id = 't-' + name.toLowerCase();
    target[name] = section;
  });
  document.querySelectorAll('.ast-explorer .row[data-field]').forEach(row => {
    row.dataset.field.split(' ').forEach(token => {
      const name = token.split('.')[0];
      if (target[name]) return;
      // prefer the introducing row (the row just above, whose chip names the
      // type and carries its gloss) over the type's first field row
      const prev = row.previousElementSibling;
      const introduces = prev?.querySelector('.ty')?.textContent
        .match(new RegExp('\\b' + name + '\\b'));
      const t = introduces ? prev : row;
      if (!t.id) t.id = 't-' + name.toLowerCase();
      target[name] = t;
    });
  });
  const NAME_RE = new RegExp(
    '\\b(' + Object.keys(target).sort((a, b) => b.length - a.length).join('|') + ')\\b', 'g');
  document.querySelectorAll('.ast-explorer .ty').forEach(chip => {
    const section = chip.closest('section');
    const html = chip.textContent.replace(NAME_RE, name =>
      target[name].closest('section') === section
        ? name
        : `<a class="ty-link" href="#${target[name].id}">${name}</a>`);
    if (html !== chip.textContent) chip.innerHTML = html;
  });
  // Some chips sit inside <summary>; keep a link click from toggling the node.
  document.querySelectorAll('.ast-explorer .ty-link').forEach(a => {
    a.addEventListener('click', (e) => e.stopPropagation());
  });
</script>

</div><!-- /.ast-explorer -->
