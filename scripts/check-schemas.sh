#!/usr/bin/env bash
# Validate the MWL JSON Schemas and the documents that instantiate them.
#
# Checks, in order:
#   1. Both schemas are valid JSON Schema 2020-12 documents.
#   2. Every valid instance validates: the spec's provider definitions and the
#      fixtures under tests/schemas/valid/.
#   3. Every fixture under tests/schemas/invalid/ is rejected.
#   4. Every conformance case under tests/conformance/ is complete and
#      compliant: the required files exist, definition.json validates against
#      the flow schema, case.json validates against the case-metadata schema,
#      the scenario directories match case.json's scenarios exactly, and in
#      every scenario expected-result.json validates against the suite's
#      Result schema, input.json parses as JSON, and arguments.json (when
#      present) is a JSON object.
#
# Requires check-jsonschema (brew install check-jsonschema).

set -euo pipefail
shopt -s nullglob

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

cd "$THIS_DIR/.."

flow_schema=static/v0.1/flow/schema.json
provider_schema=static/v0.1/provider/schema.json

status=0

echo "== Schemas against the 2020-12 metaschema"
check-jsonschema --check-metaschema "$flow_schema" "$provider_schema"

echo "== Provider definitions against $provider_schema"
provider_instances=(
  content/reference/providers/*.v1.json
  tests/schemas/valid/provider/*.json
)
check-jsonschema --schemafile "$provider_schema" "${provider_instances[@]}"

echo "== Workflow definitions against $flow_schema"
flow_instances=(tests/schemas/valid/flow/*.json)
check-jsonschema --schemafile "$flow_schema" "${flow_instances[@]}"

echo "== Invalid fixtures must be rejected"
for fixture in tests/schemas/invalid/flow/*.json; do
  if check-jsonschema --schemafile "$flow_schema" "$fixture" >/dev/null 2>&1; then
    echo "FAIL -- accepted: $fixture"
    status=1
  else
    echo "ok -- rejected: $fixture"
  fi
done
for fixture in tests/schemas/invalid/provider/*.json; do
  if check-jsonschema --schemafile "$provider_schema" "$fixture" >/dev/null 2>&1; then
    echo "FAIL -- accepted: $fixture"
    status=1
  else
    echo "ok -- rejected: $fixture"
  fi
done

echo "== Conformance cases"
result_schema=tests/conformance/result.schema.json
case_schema=tests/conformance/case.schema.json
conformance_definitions=()
conformance_expected=()
conformance_meta=()
for case_dir in tests/conformance/*/; do
  for required in README.md case.json definition.json; do
    if [ ! -f "$case_dir$required" ]; then
      echo "FAIL -- missing: $case_dir$required"
      status=1
    fi
  done
  [ -f "${case_dir}definition.json" ] &&
    conformance_definitions+=("${case_dir}definition.json")
  [ -f "${case_dir}case.json" ] &&
    conformance_meta+=("${case_dir}case.json")

  scenario_dirs=("${case_dir}scenarios"/*/)
  if [ ${#scenario_dirs[@]} -eq 0 ]; then
    echo "FAIL -- no scenarios: $case_dir"
    status=1
    continue
  fi
  for scenario_dir in "${scenario_dirs[@]}"; do
    for required in input.json expected-result.json; do
      if [ ! -f "$scenario_dir$required" ]; then
        echo "FAIL -- missing: $scenario_dir$required"
        status=1
      fi
    done
    [ -f "${scenario_dir}expected-result.json" ] &&
      conformance_expected+=("${scenario_dir}expected-result.json")
    if [ -f "${scenario_dir}input.json" ] &&
      ! python3 -m json.tool "${scenario_dir}input.json" >/dev/null 2>&1; then
      echo "FAIL -- input.json is not valid JSON: $scenario_dir"
      status=1
    fi
    if [ -f "${scenario_dir}arguments.json" ] &&
      ! python3 -c '
import json, sys
with open(sys.argv[1]) as f:
    if not isinstance(json.load(f), dict):
        raise SystemExit(1)
' "${scenario_dir}arguments.json" >/dev/null 2>&1; then
      echo "FAIL -- arguments.json is not a JSON object: $scenario_dir"
      status=1
    fi
  done

  if [ -f "${case_dir}case.json" ] &&
    ! python3 -c '
import json, os, sys
case_dir = sys.argv[1]
with open(os.path.join(case_dir, "case.json")) as f:
    declared = sorted(json.load(f).get("scenarios", {}))
present = sorted(
    entry.name
    for entry in os.scandir(os.path.join(case_dir, "scenarios"))
    if entry.is_dir()
)
if declared != present:
    raise SystemExit(1)
' "$case_dir" >/dev/null 2>&1; then
    echo "FAIL -- case.json scenarios do not match scenarios/: $case_dir"
    status=1
  fi

  # The case schema constrains the profile URIs but cannot require a specific
  # member be present; a conformance claim always includes Core.
  if [ -f "${case_dir}case.json" ] &&
    ! python3 -c '
import json, sys
core = "https://mwl.dev/v0.1/conformance/core"
with open(sys.argv[1]) as f:
    if core not in json.load(f).get("profiles", []):
        raise SystemExit(1)
' "${case_dir}case.json" >/dev/null 2>&1; then
    echo "FAIL -- case.json profiles do not include the Core URI: $case_dir"
    status=1
  fi
done
check-jsonschema --schemafile "$flow_schema" "${conformance_definitions[@]}"
check-jsonschema --schemafile "$result_schema" "${conformance_expected[@]}"
check-jsonschema --schemafile "$case_schema" "${conformance_meta[@]}"

exit "$status"
