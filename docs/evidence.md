# Evidence and retention

Evidence is read from public RPC events and transaction metadata. Event queries
are paged in bounded batches, matched by verified topic segments, and resolved
to transaction hashes when available. The observer records the RPC retention
floor and uses ledger metadata when an explicit health value is unavailable.

Unavailable evidence is reported as `UNMEASURABLE`; it is never converted into
a finding. The event reader stops after its configured page ceiling and returns
an operational error instead of looping indefinitely.

Source of truth: `pkg/evidence` and `docs/detectors.md`.
