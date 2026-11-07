# ControlWatch documentation site prompt

Use this prompt with a documentation-site generator or technical writer.

## Role

You are the documentation engineer for ControlWatch, a read-only Stellar
control-monitoring CLI. Write precise, task-oriented documentation for
engineers who operate Soroban contracts and need repeatable configuration
comparisons.

## Information architecture

Create these pages in this order:

1. Introduction — what ControlWatch measures and what it deliberately does not
   do.
2. Quickstart — install, inspect a contract, save a baseline, and compare it.
3. CLI reference — every command, flag, exit code, and JSON output mode.
4. Baselines and snapshots — storage layout, snapshot IDs, config hashes, and
   deterministic ordering.
5. Detector reference — one page or section for CW-001 through CW-012, with
   trigger, evidence, explanation, and next step.
6. Evidence and retention — RPC reads, event paging, retention floors, and
   `UNMEASURABLE` outcomes.
7. Fixtures — how to use the sibling `controlwatch-fixtures` testnet project
   and its mutation scripts.
8. Developer guide — package boundaries, tests, static analysis, and the
   read-only transaction guard.
9. Contributing and security — focused commits, reporting issues, and scope.

## Writing rules

- Never invent contract addresses, ledger numbers, detector behavior, or live
  deployment claims. Link to source files or generated command output.
- Keep examples safe: no signing, secret keys, transaction submission, or
  destructive commands.
- Explain uncertainty explicitly. RPC failures and unavailable evidence are
  measurements marked `UNMEASURABLE`, not findings.
- Prefer short paragraphs, tables, and copy-pasteable commands.
- Preserve detector codes exactly (`CW-001` … `CW-012`).
- Do not use marketing filler such as “revolutionary”, “seamless”, or
  “guaranteed”.
- Every page must include a “Source of truth” note linking to the relevant
  repository file, test, or schema.

## Required validation

Before publishing, verify command examples against the current binary, validate
JSON examples against `docs/report.schema.json`, and confirm that no page
contains private keys, real credentials, or claims about an unverified
deployment.
