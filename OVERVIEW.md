# Lux Oracle — Overview

`luxfi/oracle` is the decentralized oracle infrastructure for the Lux
network: it operates the **O-Chain VM (`oraclevm`)** for canonical
price-feed aggregation + attested external data, alongside the UMA-
derived Optimistic Oracle Solidity contracts under `contracts/prediction`.

This OVERVIEW is the cover sheet for the Tier-A documentation set on
the Go side. The Solidity side is licensed AGPL-3.0 and ships its own
audit trail in `lib/openzeppelin-contracts/audits/`.

## Components

| Surface | Layer | Role |
|---|---|---|
| `vm/` | O-Chain VM (oraclevm) | Consensus + state for Feeds, Observations, AggregatedValues, Oracle Requests, Records, and Merkle commitments. Source of truth. |
| `pkg/profile/` | Signing-profile gate | ONE function (`profile.Verify`) — strict-PQ default. |
| `pkg/zaptransport/` | Intra-Lux transport | `github.com/luxfi/zap` wrapper for operator-plane Observation / Record broadcast. |
| `cmd/oracled/` | Daemon | Standalone operator process. Fetches external data, signs Observations, submits to O-Chain. |
| `contracts/prediction/` | Optimistic Oracle (Solidity, AGPL-3.0) | Assert-dispute-settle pattern for prediction market resolution. Not part of the strict-PQ surface. |
| `contracts/registry/` | Finder / Store / IdentifierWhitelist | Service locator + fee management. |

## Two surfaces, one rule

Oracle talks to TWO surfaces, decomplected:

1. **Intra-Lux operator surface** — signed Observations, OracleRecords,
   OracleCommits, executor attestations.
   **PQ by default**: ML-DSA-65 (FIPS 204). Classical Ed25519 is opt-in
   only via `Config.LegacyClassicalEnabled`. Transport: ZAP.

2. **External-source surface** — Bitcoin RPC, Ethereum RPC, Pyth,
   Chainlink, market-data APIs (CEX/DEX feeds).
   **Native primitives / native transport**: HTTP, JSON, secp256k1 ECDSA
   wherever the source demands. Not subject to the PQ default. The oracle
   never PQ-flips a fetch against a public price feed.

## Tier-A documents

| File | Purpose |
|---|---|
| `OVERVIEW.md` | This file. |
| `SPEC.md` | Protocol, state machine, message format. |
| `PROOF-CLAIMS.md` | Honest scope. |
| `TRUSTED-COMPUTING-BASE.md` | Trust footprint. |
| `CRYPTOGRAPHER-SIGN-OFF.md` | Independent review. |
| `DEPLOYMENT-RUNBOOK.md` | Operator-facing guidance. |
| `LEGACY-CLASSICAL.md` | Disclosure of the opt-in classical path. |
| `CHANGELOG.md` | Substantive changes to spec / trust footprint. |

## License

Mixed: Go layer is the standard Lux license; Solidity contracts under
`contracts/prediction/` are AGPL-3.0 (UMA-derived). See `LICENSE`.
