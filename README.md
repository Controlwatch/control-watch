# ControlWatch

ControlWatch is a read-only Go CLI and library for inspecting the control
surface of Stellar regulated tokens. It records immutable snapshots, compares
semantic state, reports review-priority findings, and attaches available
on-chain evidence.

The MVP targets SEP-57/OpenZeppelin RWA tokens and also records Stellar Asset
Contract administration and classic-asset flags as context.

## Status

Active development. The sibling `fixtures` repository contains the pinned
OpenZeppelin testnet stack used to verify contract getters, events, and RPC
behavior before those details are encoded here.

## Scope and limits

ControlWatch:

- reads public Stellar RPC and Horizon data;
- never holds a secret key or submits a transaction;
- marks unreadable values as `UNMEASURABLE` instead of treating them as token
  changes; and
- reports review priority, not exploitability or intent.

ControlWatch does not provide a security audit, legal conclusion, compliance
conclusion, custody service, alerting service, or web interface. A finding says
that a measured control-surface value changed; it does not establish why the
change occurred or what its consequences are.

## Planned CLI

```text
controlwatch inspect --network <testnet|mainnet> --contract <C...>
controlwatch baseline list --network <testnet|mainnet> --contract <C...>
controlwatch compare --network <testnet|mainnet> --contract <C...>
controlwatch version
```

Usage documentation will be expanded as each command becomes available.

## Development

The project requires Go 1.23 or newer. The verified local and SDK versions are
recorded in [`docs/VERSIONS.md`](docs/VERSIONS.md).

```bash
go test ./...
go vet ./...
go build ./...
```

Live testnet checks are opt-in integration tests. Unit tests use fake clients
and recorded responses so the default suite runs without network access.

