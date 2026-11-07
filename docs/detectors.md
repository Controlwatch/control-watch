# Detector reference

ControlWatch compares semantic observation fields only. Ledger numbers,
timestamps, tool versions, and warning text never create findings. An unreadable
field is listed under `not_compared` instead.

| Code | Name | Trigger | Priority | Evidence family |
| --- | --- | --- | --- | --- |
| CW-001 | `COMPLIANCE_CONTRACT_CHANGED` | Token compliance reference changed | HIGH | `compliance_set` |
| CW-002 | `IDENTITY_VERIFIER_CHANGED` | Token identity-verifier reference changed | HIGH | `identity_verifier_set` |
| CW-003 | `MODULE_ADDED` | Module address is new | MEDIUM | `module_added` |
| CW-004 | `MODULE_REMOVED` | Module address is absent from current state | MEDIUM | `module_removed` |
| CW-005 | `WASM_HASH_CHANGED` | Tracked Wasm hash changed | HIGH; MEDIUM for modules | No universal event family |
| CW-006 | `ADMIN_CHANGED` | RWA token administrator changed | HIGH | `admin_transfer_completed`, `admin_renounced` |
| CW-007 | `ROLE_MEMBERS_CHANGED` | Member set for a configured role changed | MEDIUM | `role_granted`, `role_revoked` |
| CW-008 | `PAUSE_STATE_CHANGED` | Token pause state changed | MEDIUM | `paused`, `unpaused` |
| CW-009 | `SAC_ADMIN_CHANGED` | SAC administrator changed | HIGH | Implementation-specific; not assumed |
| CW-010 | `CLASSIFICATION_CHANGED` | Observed contract classification changed | HIGH | No event expected |
| CW-011 | `CLASSIC_FLAGS_CHANGED` | Classic issuer authorization flag changed | MEDIUM | Issuer account operation; not assumed |
| CW-012 | `MODULE_HOOKS_CHANGED` | Hook set changed for an unchanged module | LOW | `module_added`, `module_removed` |

## Comparison rules

- Findings are ordered by detector code, then subject in JSON output.
- Human output orders findings by review priority, then code, then subject.
- Role members and hook sets are treated as sorted sets.
- New and removed modules are handled separately from module hash changes.
- `UNMEASURABLE` values never become findings.

## Evidence rules

Evidence starts after the baseline ledger and is bounded by the RPC server's
retention floor. Matching events are checked by their exact segment-zero topic
symbol and their transactions are resolved before a finding is marked `FOUND`.
Event requests use pages of 100 records and stop after 1,000 pages to keep a
report bounded; exceeding that limit is an operational error.

When the change predates retained history, the finding remains visible with
`UNAVAILABLE_OUTSIDE_RETENTION` and a `changed_between` ledger interval bounded
by the relevant ledger entry's `lastModifiedLedger`. A measured difference with
no matching event in the searchable range is `NOT_EMITTED`.
