# Verification spikes

These notes record interface details that ControlWatch must not infer. All
contract invocations use `--send no`, so the Stellar CLI simulates the call and
does not sign or submit a transaction.

## S1 — exact RWA getter names

**Date:** 2026-11-06  
**Network:** Stellar testnet  
**Token:** `CAI26Z4CPI5XQ34NEEN64ISTCHPNU733HD6AQQYQIA6NITB4E2AYYHHK`  
**Compliance:** `CDJBZS26QTNWPEZPL5NQOQZ3QELLPH47QQN6HIIT56HJMUL2AWGMBATS`

### What was checked

The pinned OpenZeppelin Stellar Contracts v0.8.0 traits and the fixture
implementations were inspected first. The relevant interfaces are composed by:

- `stellar_tokens::rwa::RWAToken`
- `stellar_contract_utils::pausable::Pausable`
- `stellar_access::access_control::AccessControl`
- `stellar_tokens::rwa::compliance::Compliance`

Each candidate getter was then simulated against the deployed baseline. The
command shape was:

```text
stellar contract invoke --id <contract> \
  --source GD3C2MXS3PT6JJSTOFH4FZGMJ7GRMN52PZQC73FP3JYOEDS4LBAKIFXB \
  --network testnet --send no -- <function> <arguments>
```

### Results

| Surface | Exact function | Arguments | Simulated result |
|---|---|---|---|
| Token | `compliance` | none | `CDJBZS26QTNWPEZPL5NQOQZ3QELLPH47QQN6HIIT56HJMUL2AWGMBATS` |
| Token | `identity_verifier` | none | `CDFDYFIFX3RCOT3YDBM6GDTXLRZNHVDXTAAUBYUBE7GTOELVKMTDQMXK` |
| Token | `paused` | none | `false` |
| Token | `get_admin` | none | `GD3C2MXS3PT6JJSTOFH4FZGMJ7GRMN52PZQC73FP3JYOEDS4LBAKIFXB` |
| Token | `get_existing_roles` | none | `["manager"]` |
| Token | `get_role_member_count` | `--role manager` | `1` |
| Token | `get_role_member` | `--role manager --index 0` | `GA52JOKWGMWMJWRQQHA5OWEBCLYDH5SGIN77D2OID4LDF7W7BE7BPUJH` |
| Compliance | `get_modules_for_hook` | `--hook Transferred` | `[]` |
| Compliance | `get_modules_for_hook` | `--hook Created` | `[]` |
| Compliance | `get_modules_for_hook` | `--hook Destroyed` | `[]` |

The CLI-generated schema help constrained the hook argument to the exact enum
variants `Transferred`, `Created`, and `Destroyed`.

### Decision

S1 passes. ControlWatch may use the exact function names above. Role membership
must be enumerated by calling `get_existing_roles`, then
`get_role_member_count(role)`, then `get_role_member(role, index)`. Compliance
modules must be queried once for each of the three exact hook variants. Empty
module vectors are valid observations, not errors.
