# TRUSTED-COMPUTING-BASE — Lux Oracle

> What you must trust below the oracle's signing-profile gate.

## §1 Primitive TCBs

| Layer | What you trust | Why |
|---|---|---|
| `luxfi/crypto/mldsa` | FIPS 204 wrapper over Cloudflare CIRCL's ML-DSA-65 reference. KAT-validated. | Verifies actual ML-DSA-65 signature math. The oracle does NOT reimplement any FIPS 204 step. |
| `crypto/ed25519` (Go stdlib) | RFC 8032 reference. | Only used on the classical opt-in path. Under strict-PQ it is unreachable from `VM.VerifyObservationSignature` / `VM.VerifyRecordSignature`. |
| `crypto/sha256` (Go stdlib) | FIPS 180-4. | Used for canonical-bytes construction (Observation, Record) and Merkle leaf hashing. |

## §2 Codebase TCB

| Component | Trust | Mitigation |
|---|---|---|
| `pkg/profile/profile.go` | The gate. | ~190 LOC, single `Permit` function. No re-admission paths. |
| `vm/vm.go::VerifyObservationSignature` | Single caller of `profile.Verify` for Observations. | Reviewed. |
| `vm/vm.go::VerifyRecordSignature` | Single caller for Records. | Reviewed. |
| `vm/vm.go::RegisterOperatorKey` | Scheme-tagged key registry. | Refuses unknown schemes; under strict-PQ classical entries still register but verification refuses them. |
| `pkg/zaptransport` | Transport only. Opaque JSON bytes. | Does not call `profile.Verify`. |

## §3 External-source TCB (NOT part of the strict-PQ surface)

The oracle fetches from external sources:

| Surface | Primitive | Trust source |
|---|---|---|
| Bitcoin / Ethereum RPC | secp256k1 ECDSA | The chain's verifying logic. Oracled inspects state, does not vouch for it. |
| Pyth / Chainlink | Ed25519 / secp256k1 | Each network's own signature scheme. |
| Market-data APIs (CEX/DEX) | TLS-only | Confidentiality + authenticity from TLS; integrity from cross-source aggregation. |

These are NOT in the strict-PQ TCB. They are external.

## §4 Aggregation TCB

| Layer | What you trust |
|---|---|
| Median / TWAP / weighted-median implementations | Trusted via their unit tests; not subject to the strict-PQ guarantee. |
| Bounded-deviation rejection | Trusted via the configured `DeviationThreshold` policy parameter. |

## §5 Build TCB

Same as `luxfi/relay`: pinned Go toolchain, pinned `luxfi/crypto`,
`luxfi/zap`, and the OpenZeppelin / UMA Solidity dependencies.

## §6 What the TCB does NOT include

- **Source honesty**: a Chainlink network reporting a bad price is
  not a cryptographic failure of the oracle.
- **AGPL-3.0 Solidity contracts**: audited separately under
  `lib/openzeppelin-contracts/audits/`.
- **PlatformVM request determinism**: trusted via PlatformVM's own
  spec; oracled enforces `ComputeRequestID` equality on
  `RegisterRequest` but does not re-derive the upstream `service_id`
  / `session_id`.
