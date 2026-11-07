# Security policy

## Supported versions

ControlWatch is active development software. The current `main` branch is the
only supported version. The bundled fixtures are testnet-only and are not
production deployment guidance.

## Reporting a vulnerability

Do not open a public issue for a vulnerability. Use a private GitHub Security
Advisory for this repository, or contact the maintainer through the private
contact method configured on the repository profile. Include reproduction
steps, affected commit, environment, and the smallest useful evidence.

## In scope

- Read-only code paths that could submit or sign a transaction.
- Snapshot integrity failures, hash-verification bypasses, or mutable snapshot
  replacement.
- Incorrect comparison results caused by state decoding, scope handling, or
  evidence association.
- Secret exposure through logs, reports, fixtures, or configuration handling.
- CI or script behavior that could silently target a non-testnet network.

## Out of scope

- Vulnerabilities in Stellar protocol, public RPC, Horizon, or third-party SDKs.
- Availability issues caused only by public RPC rate limits.
- Findings that are expected consequences of a contract changing on testnet.
- Feature requests and documentation corrections.

## Response targets

| Stage | Target |
| --- | --- |
| Acknowledge report | 3 business days |
| Initial reproduction | 7 business days |
| Mitigation plan | 14 business days where applicable |
| Disclosure coordination | Agreed with the reporter |

## Disclosure

The repository is unaudited. A fix may be published after a mitigation is
available and disclosure timing has been agreed with the reporter. Do not send
private keys, seed phrases, or credentials in a report.
