#!/usr/bin/env bash
# Build the site and run all post-build link checks.
#
# Expects to be run from the site root (where hugo.yaml lives). When run via
# the pre-commit hook, check-staged.sh handles exporting the staged index first.

set -euo pipefail

find_this() {
  THIS="${1:?'must provide script path, like "${BASH_SOURCE[0]}"'}"
  trap "echo >&2 'FATAL: could not resolve parent directory of ${THIS}'" EXIT
  [ "${THIS:0:1}" == "/" ] || THIS="$(pwd -P)/${THIS}"
  THIS_DIR="$(dirname -- "${THIS}")"
  THIS_DIR="$(cd -P -- "${THIS_DIR}" && pwd)"
  THIS="${THIS_DIR}/$(basename -- "${THIS}")"
  trap "" EXIT
}

find_this "${BASH_SOURCE[0]}"
SCRIPTS_DIR="$THIS_DIR"

echo "== Hugo build"
hugo --logLevel warn --panicOnWarning --cleanDestinationDir

echo
echo "== Spec link check: reference"
bash "$SCRIPTS_DIR/check-spec-links.sh" public/reference/spec.md

echo
echo "== Spec link check: whole-site spec"
bash "$SCRIPTS_DIR/check-spec-links.sh" --self-contained public/spec.md

echo
echo "== HTML anchor check: reference"
python3 "$SCRIPTS_DIR/check-html-anchors.py" reference

echo
echo "All checks passed."
