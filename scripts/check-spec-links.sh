#!/usr/bin/env bash
# Verifies that every in-document anchor link in spec.md points at a real anchor.
# Anchors are either explicit ({#slug} after a heading) or auto-generated from
# a heading's text using a GitHub-style slug (lowercased, spaces -> hyphens,
# non-alphanumeric stripped).

set -euo pipefail

# --self-contained: also require that no site-internal absolute link survives
# (every internal reference must be an in-document anchor). Correct for the
# whole-site spec.md, where all sections are present. A single-section file
# (e.g. reference/spec.md) legitimately links out to absent sections, so the
# check is opt-in rather than always-on.
SELF_CONTAINED=0
SPEC=""
for arg in "$@"; do
  if [[ "$arg" == "--self-contained" ]]; then
    SELF_CONTAINED=1
  else
    SPEC="$arg"
  fi
done
SPEC="${SPEC:-public/reference/spec.md}"

if [[ ! -f "$SPEC" ]]; then
  echo "error: spec file not found: $SPEC" >&2
  exit 2
fi

# Slugify: lowercase, drop everything except alnum/space/hyphen, collapse spaces to hyphens.
slugify() {
  awk '{
    s = tolower($0)
    gsub(/[^a-z0-9 -]/, "", s)
    gsub(/^ +| +$/, "", s)
    gsub(/ +/, "-", s)
    print s
  }'
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# 1. Collect explicit {#slug} anchors.
grep -oE '\{#[a-z0-9-]+\}' "$SPEC" | sed 's/^{#//; s/}$//' > "$tmp/anchors_explicit"

# 2. Collect auto-anchorable headings (strip trailing {#explicit} first, then slugify).
grep -E '^#+ ' "$SPEC" \
  | sed -E 's/^#+ +//; s/[[:space:]]*\{#[a-z0-9-]+\}[[:space:]]*$//' \
  | slugify > "$tmp/anchors_auto"

sort -u "$tmp/anchors_explicit" "$tmp/anchors_auto" > "$tmp/anchors_all"

# 3. Collect link targets.
grep -oE '\]\(#[a-z0-9-]+\)' "$SPEC" | sed 's/^](#//; s/)$//' | sort -u > "$tmp/targets"

# 4. Report broken.
broken=$(comm -23 "$tmp/targets" "$tmp/anchors_all")

# 5. Under --self-contained, collect residual site-internal absolute links.
#    Every internal reference in the whole-site spec must be an in-document
#    anchor (#...); a surviving root-relative link (e.g. /reference/foo/) means a
#    link the generator's rewrite rules failed to convert — broken in the flat
#    file, and invisible to the anchor check above. Match top-level content
#    sections only, so genuine external/static asset links are not flagged.
residual=""
if [[ "$SELF_CONTAINED" -eq 1 ]]; then
  residual=$(grep -oE '\]\(/(guide|reference|rationale)[^)]*\)' "$SPEC" | sort -u || true)
fi

n_targets=$(wc -l < "$tmp/targets")
n_anchors=$(wc -l < "$tmp/anchors_all")
n_broken=$([ -z "$broken" ] && echo 0 || echo "$broken" | wc -l)

echo "spec: $SPEC"
echo "  unique link targets: $n_targets"
echo "  unique anchors:      $n_anchors"
echo "  broken links:        $n_broken"
if [[ "$SELF_CONTAINED" -eq 1 ]]; then
  n_residual=$([ -z "$residual" ] && echo 0 || echo "$residual" | wc -l)
  echo "  unrewritten links:   $n_residual"
fi

status=0

if [[ -n "$broken" ]]; then
  echo
  echo "BROKEN ANCHOR TARGETS:"
  echo "$broken" | sed 's/^/  /'
  status=1
fi

if [[ -n "$residual" ]]; then
  echo
  echo "UNREWRITTEN SITE-INTERNAL LINKS (should be in-document anchors):"
  echo "$residual" | sed 's/^/  /'
  status=1
fi

exit "$status"
