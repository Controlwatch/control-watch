#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
FIXTURES_DIR="${FIXTURES_DIR:-${REPO_ROOT}/../fixtures}"
FIXTURES_FILE="${FIXTURES_FILE:-${FIXTURES_DIR}/fixtures.json}"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 2
  fi
}

require_command go
require_command jq

if [[ ! -f "${FIXTURES_FILE}" ]]; then
  echo "fixtures file not found: ${FIXTURES_FILE}" >&2
  echo "run fixtures/scripts/deploy-baseline.sh first" >&2
  exit 2
fi

NETWORK="$(jq -er '.network' "${FIXTURES_FILE}")"
TOKEN="$(jq -er '.token' "${FIXTURES_FILE}")"
if [[ "${NETWORK}" != "testnet" ]]; then
  echo "refusing to run: fixtures.json network must be testnet" >&2
  exit 2
fi

TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TEMP_DIR}"' EXIT
STORE_DIR="${TEMP_DIR}/snapshots"
CONTROLWATCH="${TEMP_DIR}/controlwatch"

pushd "${REPO_ROOT}" >/dev/null
go build -o "${CONTROLWATCH}" ./cmd/controlwatch

"${CONTROLWATCH}" inspect \
  --network testnet \
  --contract "${TOKEN}" \
  --store-dir "${STORE_DIR}" \
  --save \
  --json "${TEMP_DIR}/inspect.json" >/dev/null

BASELINE_ID="$("${CONTROLWATCH}" baseline list \
  --network testnet \
  --contract "${TOKEN}" \
  --store-dir "${STORE_DIR}" | jq -er '.[-1].id')"

set +e
"${CONTROLWATCH}" compare \
  --network testnet \
  --contract "${TOKEN}" \
  --baseline "${BASELINE_ID}" \
  --store-dir "${STORE_DIR}" \
  --json "${TEMP_DIR}/compare.json" >/dev/null
COMPARE_EXIT=$?
set -e
popd >/dev/null

if [[ "${COMPARE_EXIT}" -ne 0 ]]; then
  echo "baseline compare exited with ${COMPARE_EXIT}; expected zero findings" >&2
  exit 1
fi

printf 'e2e baseline passed\ncontract=%s\nbaseline=%s\n' "${TOKEN}" "${BASELINE_ID}"
