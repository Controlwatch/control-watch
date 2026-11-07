#!/usr/bin/env bash
set -euo pipefail

# Creates the initial public backlog. Run only after `gh auth status` succeeds.
# The script is intentionally opt-in and never runs as part of CI.

repo="${GITHUB_REPOSITORY:-Controlwatch/control-watch}"
command -v gh >/dev/null 2>&1 || { echo "gh CLI is required" >&2; exit 1; }

gh label create "complexity:low" --repo "$repo" --color 0E8A16 --force
gh label create "complexity:medium" --repo "$repo" --color FBCA04 --force
gh label create "complexity:high" --repo "$repo" --color B60205 --force
gh label create "area:detectors" --repo "$repo" --color 5319E7 --force
gh label create "area:docs" --repo "$repo" --color 0075CA --force

create_issue() {
  local title="$1" label="$2" body="$3"
  if gh issue list --repo "$repo" --state all --search "in:title $title" --json title --jq '.[].title' | grep -Fxq "$title"; then
    echo "already exists: $title"
    return
  fi
  gh issue create --repo "$repo" --title "$title" --label "$label" --body "$body"
}

create_issue "Add integration coverage for live testnet fixtures" "complexity:medium" \
  "Add opt-in integration tests that consume fixtures.json from controlwatch-fixtures. Keep default CI offline and read-only."
create_issue "Publish detector examples from verified contract traces" "complexity:medium" \
  "Add sanitized examples for CW-001 through CW-012, sourced from reproducible fixture mutations. Do not include credentials or unverifiable addresses."
create_issue "Build the generated documentation site" "complexity:low" \
  "Use docs/DOCS_SITE_PROMPT.md to publish the introduction, CLI, detector, evidence, fixtures, and contributor pages."
