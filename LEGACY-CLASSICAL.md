# LEGACY-CLASSICAL — luxfi/oracle

> Disclosure of the opt-in classical (Ed25519) path on the oracle's
> intra-Lux operator surface.

## §1 Scope

Covers the `Config.LegacyClassicalEnabled` toggle (chain-side) and
`profile.Policy.LegacyClassicalEnabled` (oracled-side), which together
control whether **Ed25519** is accepted for Observations / OracleRecords
on the intra-Lux surface.

Does NOT cover:

- External-source primitives (Pyth Ed25519, Chainlink secp256k1, etc.).
  Those are NOT "legacy" — they are the native primitives of their
  respective networks.
- ML-DSA-65 itself (the default).
- The AGPL-3.0 Solidity Optimistic Oracle (which has its own
  cryptographic surface).

## §2 Why the toggle exists

To support the migration window during which some Lux operator
infrastructure still signs with Ed25519. New deployments default to
strict-PQ; the toggle is OFF.

## §3 What changes when the toggle is on

| Behaviour | `LegacyClassicalEnabled=false` (default) | `LegacyClassicalEnabled=true` |
|---|---|---|
| `SchemeMLDSA65` observations / records | accepted iff `mldsa65.Verify` accepts | same |
| `SchemeEd25519` observations / records | refused with `ErrClassicalRefused` BEFORE any classical math | accepted iff `ed25519.Verify` accepts with `ContextTag` prepended |
| Operator key registration | accepts both schemes | accepts both schemes |
| Domain-separation context | `"luxfi.oracle.v1"` | same |

## §4 Domain separation under classical

When Ed25519 is permitted, the bytes fed to `ed25519.Sign` /
`ed25519.Verify` are:

```
tagged = "luxfi.oracle.v1" || message
```

This guarantees a classical oracle Observation cannot be replayed as
a relay receipt (which uses `"luxfi.relay.v1"`). Protection is
structural domain separation, not cryptographic novelty.

## §5 Migration plan

Identical to relay's: Phase A (legacy on) → Phase B (operators register
ML-DSA-65 keys) → Phase C (toggle off via quorum-coordinated change) →
Phase D (deregister Ed25519). See `luxfi/relay/LEGACY-CLASSICAL.md` §5
for details; the oracle inherits the same discipline.

## §6 What the toggle does NOT enable

- Cross-scheme signature substitution.
- Hybrid signatures.
- External-source PQ flip (out of scope).
