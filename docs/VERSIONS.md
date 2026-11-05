# Verified versions

Verified on 2026-11-05 before implementation.

| Component | Version | Verification |
| --- | --- | --- |
| Go | `go1.26.1 windows/amd64` | `go version` |
| Stellar Go SDK | `v0.8.0` | Latest module version returned by `go list -m -versions github.com/stellar/go-stellar-sdk` |
| Fixture OpenZeppelin contracts | `v0.8.0` | Deployed from the sibling fixture repository |

The SDK is pinned in `go.mod`. Contract-specific names are not inferred from
the SDK; they are verified against the deployed fixture before being added to
the `surface` package.
