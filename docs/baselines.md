# Baselines and snapshots

A snapshot is immutable and identified by ledger sequence plus a short config
hash. It stores the observed contract surface, completeness, warnings, and a
content hash. A comparison always loads the requested baseline before reading
the current state.

Snapshots are stored below `<store-dir>/<network>/<contract>/`. Keep this
directory private to the operator and back it up as an audit artifact.

Source of truth: `pkg/model`, `pkg/store`, and `pkg/observe`.
