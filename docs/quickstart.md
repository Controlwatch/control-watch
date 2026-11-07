# Quickstart

ControlWatch reads a public Stellar contract, records a snapshot, and compares
later observations against that immutable baseline.

```bash
go build -o controlwatch ./cmd/controlwatch
./controlwatch inspect --network testnet --contract C... --save --json snapshot.json
./controlwatch baseline list --network testnet --contract C...
./controlwatch compare --network testnet --contract C... --baseline latest
```

The CLI never signs or submits a transaction. Replace `C...` with a verified
contract address. Use `--store-dir` to keep snapshots in an explicit location.

Source of truth: `cmd/controlwatch` and `pkg/cli`.
