#!/usr/bin/env bash
# Export the git staged index to a temp directory and run a command there.
#
# Usage: check-staged.sh [--keep] [--] <cmd> [args...]
#
# Options:
#   --keep  do not delete the temp directory on exit (prints its path)

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
export SCRIPTS_DIR="$THIS_DIR"

keep=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --keep) keep=true; shift ;;
    --) shift; break ;;
    *) break ;;
  esac
done

tmp=$(mktemp -d)
cleanup() {
  if [[ "$keep" == true ]]; then
    echo "staged directory: $tmp"
  else
    rm -rf "$tmp"
  fi
}
trap cleanup EXIT

git checkout-index --all --prefix="$tmp/"
cd "$tmp"

bash -c "$@"
