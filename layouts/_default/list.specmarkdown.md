{{- /*
  Single-file Markdown rendering of a section.
  - Walks .Pages.ByWeight recursively (sub-sections recurse into their own .Pages).
  - Uses .RawContent (source Markdown, no Goldmark pass).
  - Strips Hugo shortcodes and any source `# H1` lines (titles come from frontmatter).
  - Bumps body heading levels so source `##` nests under the synthesized page heading.
  - Rewrites cross-page links to in-document anchors.

  Anchor namespacing ($prefix):
  - When this file renders a single section (e.g. reference/spec.md), $prefix is
    "" and anchors are bare page/heading slugs (#flow-object, #flow-object-foo).
  - When it renders the whole site (spec.md, from the home page), $prefix is the
    top-level section name ("reference", "guide", "rationale") so anchors from
    different sections cannot collide (#reference-flow-object, #guide-...).
  Relative links (../section/...) always resolve within the current page's
  section, so they take the current section's prefix, threaded into "rewrite".
  Absolute links (/section/...) name their target section in the path, so the
  prefix comes from the matched path segment.
*/ -}}

{{- /* rewrite expects a dict {content, prefix}. prefix is "" for single-section files. */ -}}
{{- /* rewrite expects {content, prefix, whole}. prefix is "" for single-section
       files. whole is true only for the whole-site file: absolute cross-section
       links (/guide/...) are rewritten to in-document anchors only then; in a
       single-section file the other sections aren't present, so those links stay
       external. */ -}}
{{- define "rewrite" -}}
{{- $c := .content -}}
{{- $p := .prefix -}}
{{- /* Anchor prefix used for SAME-section (relative) targets, with trailing hyphen when set. */ -}}
{{- $rp := cond (eq $p "") "" (printf "%s-" $p) -}}
{{- /* Drop Hugo shortcodes. */ -}}
{{- $c = replaceRE `\{\{[<%][^}]*[>%]\}\}` "" $c -}}
{{- /* Drop any H1 lines (and the following blank line) so synthesized headings don't duplicate. */ -}}
{{- $c = replaceRE `(?m)^# .*\n\n?` "" $c -}}
{{- if .whole -}}
{{- /* --- Absolute cross-section links: /section/... -> #section-... The anchor
       prefix is the TARGET section (first path segment); any intermediate
       subsection folder is dropped, since anchors key off the page's basename.
       Most-specific (deepest) shapes first so a shallower rule can't partially
       match a deeper path. --- */ -}}
{{- /* Three-segment, anchored: /section/subsection/page/#anchor -> #section-page-anchor */ -}}
{{- $c = replaceRE `\]\(/([a-z][a-z0-9-]*)/[a-z][a-z0-9-]*/([a-z][a-z0-9-]*)/#([^)]+)\)` `](#$1-$2-$3)` $c -}}
{{- /* Three-segment: /section/subsection/page/ -> #section-page */ -}}
{{- $c = replaceRE `\]\(/([a-z][a-z0-9-]*)/[a-z][a-z0-9-]*/([a-z][a-z0-9-]*)/\)` `](#$1-$2)` $c -}}
{{- /* Two-segment, anchored: /section/page/#anchor -> #section-page-anchor */ -}}
{{- $c = replaceRE `\]\(/([a-z][a-z0-9-]*)/([a-z][a-z0-9-]*)/#([^)]+)\)` `](#$1-$2-$3)` $c -}}
{{- /* Two-segment: /section/page/ -> #section-page */ -}}
{{- $c = replaceRE `\]\(/([a-z][a-z0-9-]*)/([a-z][a-z0-9-]*)/\)` `](#$1-$2)` $c -}}
{{- /* One-segment, anchored: /section/#anchor -> #section-anchor */ -}}
{{- $c = replaceRE `\]\(/([a-z][a-z0-9-]*)/#([^)]+)\)` `](#$1-$2)` $c -}}
{{- /* One-segment: /section/ -> #section (the top-level section heading) */ -}}
{{- $c = replaceRE `\]\(/([a-z][a-z0-9-]*)/\)` `](#$1)` $c -}}
{{- end -}}
{{- /* --- Relative same-section links: prefixed with the current section ($rp). --- */ -}}
{{- /* Cross-page anchored, two-up: ../../section/subsection/#anchor -> #PFXsubsection-anchor */ -}}
{{- $c = replaceRE `\]\(\.\./\.\./([^/)#]+)/([^/)#]+)/#([^)]+)\)` (printf `](#%s$2-$3)` $rp) $c -}}
{{- /* Two-up: ../../section/subsection/ -> #PFXsubsection */ -}}
{{- $c = replaceRE `\]\(\.\./\.\./([^/)#]+)/([^/)#]+)/\)` (printf `](#%s$2)` $rp) $c -}}
{{- /* Two-up, one-segment, anchored: ../../section/#anchor -> #PFXsection-anchor */ -}}
{{- $c = replaceRE `\]\(\.\./\.\./([^/)#]+)/#([^)]+)\)` (printf `](#%s$1-$2)` $rp) $c -}}
{{- /* Two-up, one-segment: ../../section/ -> #PFXsection */ -}}
{{- $c = replaceRE `\]\(\.\./\.\./([^/)#]+)/\)` (printf `](#%s$1)` $rp) $c -}}
{{- /* One-up, two-segment, anchored: ../section/subsection/#anchor -> #PFXsubsection-anchor */ -}}
{{- $c = replaceRE `\]\(\.\./([^/)#]+)/([^/)#]+)/#([^)]+)\)` (printf `](#%s$2-$3)` $rp) $c -}}
{{- /* One-up, two-segment: ../section/subsection/ -> #PFXsubsection */ -}}
{{- $c = replaceRE `\]\(\.\./([^/)#]+)/([^/)#]+)/\)` (printf `](#%s$2)` $rp) $c -}}
{{- /* One-up, anchored: ../section/#anchor -> #PFXsection-anchor */ -}}
{{- $c = replaceRE `\]\(\.\./([^/)#]+)/#([^)]+)\)` (printf `](#%s$1-$2)` $rp) $c -}}
{{- /* One-up: ../section/ -> #PFXsection */ -}}
{{- $c = replaceRE `\]\(\.\./([^/)#]+)/\)` (printf `](#%s$1)` $rp) $c -}}
{{- /* Root-relative two-segment, anchored: section/subsection/#anchor -> #PFXsubsection-anchor */ -}}
{{- $c = replaceRE `\]\(([a-z][a-z0-9-]*)/([a-z][a-z0-9-]*)/#([^)]+)\)` (printf `](#%s$2-$3)` $rp) $c -}}
{{- /* Root-relative two-segment: section/subsection/ -> #PFXsubsection */ -}}
{{- $c = replaceRE `\]\(([a-z][a-z0-9-]*)/([a-z][a-z0-9-]*)/\)` (printf `](#%s$2)` $rp) $c -}}
{{- /* Root-relative anchored: section/#anchor -> #PFXsection-anchor */ -}}
{{- $c = replaceRE `\]\(([a-z][a-z0-9-]*)/#([^)]+)\)` (printf `](#%s$1-$2)` $rp) $c -}}
{{- /* Root-relative: section/ -> #PFXsection */ -}}
{{- $c = replaceRE `\]\(([a-z][a-z0-9-]*)/\)` (printf `](#%s$1)` $rp) $c -}}
{{- /* Trim leading and trailing whitespace/blank lines from each page body. */ -}}
{{- $c = replaceRE `^[ \t\n]+` "" $c -}}
{{- $c = replaceRE `[ \t\n]+$` "" $c -}}
{{- /* Collapse runs of blank lines to one blank line. */ -}}
{{- $c = replaceRE `\n[ \t]*\n[ \t]*\n+` "\n\n" $c -}}
{{- $c -}}
{{- end -}}

{{- define "renderPage" -}}
{{- $page := .page -}}
{{- $depth := .depth -}}
{{- $prefix := .prefix -}}
{{- $whole := .whole -}}
{{- /* Anchor prefix with trailing hyphen when set; "" for single-section files. */ -}}
{{- $ap := cond (eq $prefix "") "" (printf "%s-" $prefix) -}}
{{- $pageSlug := printf "%s%s" $ap (anchorize $page.File.ContentBaseName) -}}
{{- $hashes := "" -}}
{{- range seq $depth -}}{{- $hashes = printf "%s#" $hashes -}}{{- end -}}
{{- $raw := $page.RawContent -}}
{{- if $page.Params.singleFileSkipTOC -}}
  {{- /* Strip "In this document"/"In this section"/"Key terms" H2 blocks
         (navigation aids redundant in the flat file): from the matching heading
         to the next `## ` heading (exclusive) or EOF. */ -}}
  {{- $skip := false -}}
  {{- $kept := slice -}}
  {{- range $line := split $raw "\n" -}}
    {{- if findRE `^## (In this document|In this section|Key terms)\b` $line 1 -}}
      {{- $skip = true -}}
    {{- else if findRE `^## ` $line 1 -}}
      {{- $skip = false -}}
      {{- $kept = $kept | append $line -}}
    {{- else if not $skip -}}
      {{- $kept = $kept | append $line -}}
    {{- end -}}
  {{- end -}}
  {{- $raw = delimit $kept "\n" -}}
{{- end -}}
{{- /* Bump body heading levels by ($depth) so source `##` becomes one deeper than the page heading. */ -}}
{{- $body := replaceRE `(?m)^(#+ )` (printf "%s${1}" $hashes) $raw -}}
{{- /* Append explicit {#page-slug-heading-slug} anchors to body headings so cross-page links resolve. */ -}}
{{- $lines := slice -}}
{{- range $line := split $body "\n" -}}
  {{- $m := findRESubmatch `^(#+) +(.+?)\s*$` $line 1 -}}
  {{- if $m -}}
    {{- $headingText := index (index $m 0) 2 -}}
    {{- $hslug := anchorize $headingText -}}
    {{- $lines = $lines | append (printf "%s {#%s-%s}" (index (index $m 0) 0) $pageSlug $hslug) -}}
  {{- else -}}
    {{- $lines = $lines | append $line -}}
  {{- end -}}
{{- end -}}
{{- $body = delimit $lines "\n" -}}

{{ $hashes }} {{ $page.Title }} {#{{ $pageSlug }}}

{{ template "rewrite" (dict "content" $body "prefix" $prefix "whole" $whole) }}

{{ range $page.Pages.ByWeight -}}
{{ template "renderPage" (dict "page" . "depth" (add $depth 1) "prefix" $prefix "whole" $whole) }}
{{ end -}}
{{- end -}}
{{- /* The document title comes from frontmatter `title` (no _index.md carries a
       source H1; the rendered title is synthesized here). */ -}}
{{- if .IsHome -}}
{{- /* Whole-site file (spec.md): one document part per top-level section, with
       anchors namespaced by section name so cross-section anchors can't collide.
       The home page's own body (landing cards) is intentionally omitted.

       Sections render in .Pages.ByWeight order. The section _index.md weights
       (guide=10, reference=20, rationale=30) give the canonical reading order
       Guide -> Reference -> Rationale, matching the site navigation. */ -}}
# {{ .Title }}

{{ range .Pages.ByWeight -}}
{{- $section := . -}}
{{- $prefix := anchorize $section.File.ContentBaseName -}}
{{- /* The section _index.md body. Strip TOC blocks, then anchor its headings as
       {#prefix-slug} so absolute links into the landing page (e.g.
       /guide/#introduction) resolve. Headings keep their source level under the
       synthesized section H1. */ -}}
{{- $idxRaw := $section.RawContent -}}
{{- if $section.Params.singleFileSkipTOC -}}
  {{- $skip := false -}}
  {{- $kept := slice -}}
  {{- range $line := split $idxRaw "\n" -}}
    {{- if findRE `^## (In this document|In this section|Key terms)\b` $line 1 -}}
      {{- $skip = true -}}
    {{- else if findRE `^## ` $line 1 -}}
      {{- $skip = false -}}
      {{- $kept = $kept | append $line -}}
    {{- else if not $skip -}}
      {{- $kept = $kept | append $line -}}
    {{- end -}}
  {{- end -}}
  {{- $idxRaw = delimit $kept "\n" -}}
{{- end -}}
{{- $idxLines := slice -}}
{{- range $line := split $idxRaw "\n" -}}
  {{- $m := findRESubmatch `^(#+) +(.+?)\s*$` $line 1 -}}
  {{- if $m -}}
    {{- $idxLines = $idxLines | append (printf "%s {#%s-%s}" (index (index $m 0) 0) $prefix (anchorize (index (index $m 0) 2))) -}}
  {{- else -}}
    {{- $idxLines = $idxLines | append $line -}}
  {{- end -}}
{{- end -}}
{{- $idxRaw = delimit $idxLines "\n" -}}
{{- /* Synthesized H1 for the section, anchored at #prefix so /section/ links resolve. */ -}}
# {{ $section.Title }} {#{{ $prefix }}}

{{ template "rewrite" (dict "content" $idxRaw "prefix" $prefix "whole" true) }}

{{ range $section.Pages.ByWeight -}}
{{ template "renderPage" (dict "page" . "depth" 2 "prefix" $prefix "whole" true) }}
{{ end -}}
{{ end -}}
{{- else -}}
{{- /* Single-section file (e.g. reference/spec.md): flat, unprefixed anchors.
       Cross-section absolute links stay external (whole=false). */ -}}
{{- $rootRaw := .RawContent -}}
{{- if .Params.singleFileSkipTOC -}}
  {{- $skip := false -}}
  {{- $kept := slice -}}
  {{- range $line := split $rootRaw "\n" -}}
    {{- if findRE `^## (In this document|In this section|Key terms)\b` $line 1 -}}
      {{- $skip = true -}}
    {{- else if findRE `^## ` $line 1 -}}
      {{- $skip = false -}}
      {{- $kept = $kept | append $line -}}
    {{- else if not $skip -}}
      {{- $kept = $kept | append $line -}}
    {{- end -}}
  {{- end -}}
  {{- $rootRaw = delimit $kept "\n" -}}
{{- end -}}
# {{ .Title }}

{{ template "rewrite" (dict "content" $rootRaw "prefix" "" "whole" false) }}

{{ range .Pages.ByWeight -}}
{{ template "renderPage" (dict "page" . "depth" 2 "prefix" "" "whole" false) }}
{{ end -}}
{{- end -}}
