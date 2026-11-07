# ControlWatch

[![CI](https://github.com/Controlwatch/control-watch/actions/workflows/ci.yml/badge.svg)](https://github.com/Controlwatch/control-watch/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.26.1-00ADD8?logo=go&logoColor=white)
![Network](https://img.shields.io/badge/network-testnet-7B61FF)
![Stellar](https://img.shields.io/badge/Stellar-Soroban-000000)

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

## Maintainer

| Name | Role | GitHub | Telegram |
| --- | --- | --- | --- |
| Fuhad Adesanya | Maintainer | [@fadesany](https://github.com/fadesany) | Not published |

Normal project contact is handled through GitHub issues. Sensitive reports use
the private path in [`SECURITY.md`](SECURITY.md).

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

## CLI

```text
controlwatch inspect --network <testnet|mainnet> --contract <C...> [--save] [--json <path>]
controlwatch baseline list --network <testnet|mainnet> --contract <C...>
controlwatch compare --network <testnet|mainnet> --contract <C...> [--baseline latest] [--json <path>] [--save-current]
controlwatch version
```

`compare` returns `0` for no findings, `10` when findings are present, `2` for
usage/configuration errors, `3` for operational failures, and `4` when required
comparison values could not be measured.

See [`docs/detectors.md`](docs/detectors.md) for the detector reference and
[`docs/report.schema.json`](docs/report.schema.json) for the report contract.

## Development

The project requires Go 1.26.1 or newer. The verified local and SDK versions are
recorded in [`docs/VERSIONS.md`](docs/VERSIONS.md).

```bash
go test ./...
go vet ./...
go build ./...
```

Live testnet checks are opt-in integration tests. Unit tests use fake clients
and recorded responses so the default suite runs without network access.

## Fixtures and testnet reproduction

```bash
cd ../fixtures
bash scripts/setup-testnet.sh
bash scripts/deploy-baseline.sh
cd ../control-watch
bash scripts/e2e.sh
```

The fixture scripts refuse non-testnet networks. Each mutation changes one
measured control and `fixtures/scripts/verify.sh` checks that the expected
detector is the only finding. Start every mutation from a fresh baseline.

## Credits

[![Contributors](https://contrib.rocks/image?repo=Controlwatch/control-watch)](https://github.com/Controlwatch/control-watch/graphs/contributors)

