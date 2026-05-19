# Cryptographer sign-off — luxfi/oracle (Tier-A flip)

> Independent review of the oracle's PQ-default flip and ZAP-native
> transport at the current revision on `main` of
> `github.com/luxfi/oracle`.
> Reviewer: cryptographer agent (internal).

## Summary

**APPROVED WITH GATES** for ML-DSA-65-default intra-Lux operator
signing on Observations and OracleRecords, subject to the disclosures
in `LEGACY-CLASSICAL.md` and the external-source caveats in
`TRUSTED-COMPUTING-BASE.md` §3.

The oracle correctly decomplects two signing surfaces:

1. **Intra-Lux operator surface** — gated by `pkg/profile`, ML-DSA-65
   default, classical refused under strict-PQ. The gate is a single
   function (~190 LOC).

2. **External-source surface** — Pyth, Chainlink, Bitcoin / Ethereum
   RPC, market-data APIs. NOT subject to the PQ default; this is
   correct.

## What was reviewed

- `pkg/profile/profile.go` — the gate. Identical shape to relay's
  gate; `ContextTag = "luxfi.oracle.v1"`.
- `vm/vm.go` — `RegisterOperatorKey`, `VerifyObservationSignature`,
  `VerifyRecordSignature`. Each verifier calls `profile.Verify`
  exactly once.
- `pkg/zaptransport/zaptransport.go` — ZAP wrapper, opaque JSON only.
  Two message types: `MsgOracleObservation` (`0x59`) and
  `MsgOracleRecord` (`0x5A`).
- `cmd/oracled/main.go` — wires ZAP listener.
- Tests:
  - `pkg/profile/profile_test.go` (5 tests pass)
  - `vm/observation_pq_test.go` (5 tests pass — Observations + Records,
    happy / strict-refuse / legacy-accept / unregistered)
  - `vm/observation_fuzz_test.go` (2 fuzz targets, ~50k execs no
    panics)
  - `pkg/zaptransport/zaptransport_test.go` (3 e2e tests pass)

## Verified green

- [x] `GOWORK=off go build ./...` clean.
- [x] `GOWORK=off go vet ./...` clean.
- [x] `GOWORK=off go test -count=1 -race ./...` → all packages pass.
- [x] PQ default refuses Ed25519 Observations/Records.
- [x] Legacy toggle accepts Ed25519 with the correct ContextTag.
- [x] Domain separation: `ContextTag = "luxfi.oracle.v1"` is distinct
      from the relay's `luxfi.relay.v1`, so cross-protocol replay is
      structurally prevented.
- [x] Unregistered Operator/Executor: refused (no silent-accept).
- [x] JSON fuzz: ≥50k execs each, no panics.
- [x] ZAP e2e: Observation + Record broadcast round-tripped between
      two ZAP nodes.

## Gates (open, deferred)

| ID | Gate | Status |
|---|---|---|
| GATE-1 | Lean / EasyCrypt refinement of the `Permit → Verify` chain. | not blocking |
| GATE-2 | Property-based exhaustive enum check (refuse all non-MLDSA65 under `Policy{}`). | nice to have |
| GATE-3 | Aggregation-bound formal proof for canonical output under malicious-minority operators. | future |
| GATE-4 | UMA-derived contracts audit refresh under `contracts/prediction/`. | external, AGPL-3.0 inheritance |

## What is NOT covered

- ML-DSA-65 itself (FIPS 204 + CIRCL).
- External source honesty (a malicious Pyth aggregator reporting wrong
  values is not a crypto failure).
- Aggregation correctness (median / TWAP are unit-tested separately).
- AGPL-3.0 Solidity contracts.

## Conclusion

Ship the PQ-default flip. The two-surface decomposition is sound, the
gate is a single function shared verbatim with relay, and the
unregistered-operator soundness bug is closed by design (no analogue
of the relay's prior log-and-accept silent-pass ever existed here, but
the same explicit-refuse pattern is enforced).
