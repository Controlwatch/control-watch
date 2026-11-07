# Fixtures and testnet

The sibling [`controlwatch-fixtures`](https://github.com/Controlwatch/fixtures)
repository deploys the pinned Soroban contracts used for live verification.
It is testnet-only and keeps generated addresses in ignored `fixtures.json`.

```bash
cd ../fixtures
bash scripts/setup-testnet.sh
bash scripts/deploy-baseline.sh
cd ../control-watch
bash scripts/e2e.sh
```

Mutation scripts must start from a fresh baseline. They exercise one control at
a time and `verify.sh` checks the expected detector code.

Source of truth: the fixtures repository README and `scripts/`.
