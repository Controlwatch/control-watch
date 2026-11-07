# CLI reference

## `inspect`

Reads the selected contract surface. `--save` persists an immutable snapshot;
`--json PATH` writes the serialized snapshot to a file.

## `baseline list`

Lists stored snapshots for a network and contract. The list is ordered from
oldest to newest so the newest baseline appears last.

## `compare`

Loads a baseline, observes the current state, runs all configured detectors,
and renders a human-readable report. Pass `--json PATH` for the schema-backed
JSON report.

Exit codes are `0` for no findings, `10` for findings, `2` for usage errors,
`3` for operational failures, and `4` when required values are not measurable.

## `version`

Prints ControlWatch, Go, Stellar SDK, and Soroban SDK version information.

Source of truth: `pkg/cli`, `pkg/report`, and `docs/report.schema.json`.
