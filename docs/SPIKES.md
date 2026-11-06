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

## S2 — exact event names and layouts

- **Date:** 2026-11-06
- **Network:** Stellar testnet
- **Token:** `CAI26Z4CPI5XQ34NEEN64ISTCHPNU733HD6AQQYQIA6NITB4E2AYYHHK`
- **Compliance:** `CDJBZS26QTNWPEZPL5NQOQZ3QELLPH47QQN6HIIT56HJMUL2AWGMBATS`

### What was checked

The event structs in OpenZeppelin Stellar Contracts v0.8.0 were compared with
the specs embedded in both deployed contracts. The deployed specs were fetched
with:

```text
stellar contract info interface --contract-id <contract> \
  --network testnet --output json-formatted
```

As a secondary check, retained events were queried from ledger 5,500,000 with
`stellar events --output json`. The RPC returned `No events` for these contract
IDs, so the spike does not claim that a retained occurrence of every event was
observed. The deployed Wasm specs themselves contain the complete event
schemas.

### Results

The layouts below use ordered topic segments. Segment zero is always the event
name as a Soroban symbol. Event data uses the spec's `map` format.

| Event | Ordered topics | Data map |
|---|---|---|
| `compliance_set` | `compliance_set`, `compliance: Address` | empty |
| `identity_verifier_set` | `identity_verifier_set`, `identity_verifier: Address` | empty |
| `module_added` | `module_added`, `hook: ComplianceHook` | `module: Address` |
| `module_removed` | `module_removed`, `hook: ComplianceHook` | `module: Address` |
| `paused` | `paused` | empty |
| `unpaused` | `unpaused` | empty |
| `admin_transfer_initiated` | `admin_transfer_initiated`, `current_admin: Address` | `new_admin: Address`, `live_until_ledger: u32` |
| `admin_transfer_completed` | `admin_transfer_completed`, `new_admin: Address` | `previous_admin: Address` |
| `admin_renounced` | `admin_renounced`, `admin: Address` | empty |
| `role_admin_changed` | `role_admin_changed`, `role: Symbol` | `previous_admin_role: Symbol`, `new_admin_role: Symbol` |
| `role_granted` | `role_granted`, `role: Symbol`, `account: Address` | `caller: Address` |
| `role_revoked` | `role_revoked`, `role: Symbol`, `account: Address` | `caller: Address` |

For `module_added` and `module_removed`, `ComplianceHook` is the union verified
in S1 with the variants `Transferred`, `Created`, and `Destroyed`.

### Decision

S2 passes. Event matching may use the exact segment-zero symbols above. Decoders
must preserve topic order and must not assume that all fields are in event data;
addresses and roles marked as topics are part of the topic list. Admin changes
are a two-event protocol, so `admin_transfer_initiated` is not evidence that the
admin actually changed; only `admin_transfer_completed` confirms completion.

## S3 — simulation with a null or unfunded source

- **Date:** 2026-11-06
- **Network:** Stellar testnet
- **Token:** `CAI26Z4CPI5XQ34NEEN64ISTCHPNU733HD6AQQYQIA6NITB4E2AYYHHK`

### What was checked

A temporary key was generated locally without `--fund`. Only its public address
was passed to the invocation, and `--send no` prevented signing or submission:

```text
stellar keys generate controlwatch-s3-unfunded
stellar keys address controlwatch-s3-unfunded
# GDFENFR7BNSY76YUW7OSTGNOJHS4VBDMJ2Z6D3VX7EXWCZHVDY43SWC7

curl https://horizon-testnet.stellar.org/accounts/GDFENFR7BNSY76YUW7OSTGNOJHS4VBDMJ2Z6D3VX7EXWCZHVDY43SWC7
# HTTP 404

stellar contract invoke \
  --id CAI26Z4CPI5XQ34NEEN64ISTCHPNU733HD6AQQYQIA6NITB4E2AYYHHK \
  --source GDFENFR7BNSY76YUW7OSTGNOJHS4VBDMJ2Z6D3VX7EXWCZHVDY43SWC7 \
  --network testnet --send no -- compliance
# "CDJBZS26QTNWPEZPL5NQOQZ3QELLPH47QQN6HIIT56HJMUL2AWGMBATS"
```

The temporary identity was removed after the test. Omitting `--source` entirely
failed before simulation because Stellar transactions require a source-account
field. A true null source therefore cannot be encoded in transaction XDR.

The conventional all-zero address
`GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF` was also checked.
Horizon returned HTTP 200 for that address on testnet, so it is not an unfunded
sentinel and ControlWatch must not describe it as one.

### Decision

S3 passes through the unfunded-source path. Read-only getter simulations may use
a syntactically valid, unfunded public account with no secret key. ControlWatch
must always encode a source account, must never sign the envelope, and must only
send it to `simulateTransaction`. The implementation must not rely on the
all-zero address being absent from a network.
