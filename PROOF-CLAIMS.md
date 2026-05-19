# PROOF-CLAIMS — Lux Oracle

> What this submission DOES and DOES NOT prove.

## §1 The narrow claim

> **PQ-default + classical-gate.** Under the trusted-computing base in
> `TRUSTED-COMPUTING-BASE.md`, every Observation / OracleRecord
> verification call on the default `profile.Policy` (zero value)
> refuses classical schemes BEFORE any classical signature math runs,
> and accepts ML-DSA-65 signatures iff the underlying `mldsa65.Verify`
> (FIPS 204 §3) returns accept for the domain-separated input.

This is not a security proof of ML-DSA-65 itself — that proof belongs
to FIPS 204 / module-LWE EUF-CMA reductions. We prove that the oracle
correctly *uses* the FIPS 204 primitive and correctly *refuses* the
classical path under strict-PQ.

## §2 Evidence

1. **Code surface size**: `pkg/profile/profile.go` ≈ 190 LOC. The gate
   is a single function `Permit` and a single dispatching `Verify`.
2. **Single-point-of-entry**: `VM.VerifyObservationSignature` and
   `VM.VerifyRecordSignature` each call `profile.Verify` exactly
   once. No other VM code-path calls `mldsa.*Verify*` or
   `ed25519.Verify` directly.
3. **End-to-end tests**:
   - `vm/observation_pq_test.go::TestE2E_PQObservation_DefaultMLDSA65`
   - `vm/observation_pq_test.go::TestStrictPQ_RefusesEd25519Observation`
   - `vm/observation_pq_test.go::TestLegacyEnabled_AcceptsEd25519Observation`
   - `vm/observation_pq_test.go::TestE2E_PQRecord_DefaultMLDSA65`
   - `vm/observation_pq_test.go::TestUnregisteredOperator_Refused`
4. **Fuzzing**: `vm/observation_fuzz_test.go::FuzzObservationDecode` and
   `FuzzOracleRecordDecode` exercise the JSON decode boundary; no
   panics after ~50k execs each.
5. **ZAP end-to-end**:
   `pkg/zaptransport/zaptransport_test.go::TestZAPTransport_BroadcastObservation`
   and `TestZAPTransport_BroadcastRecord` round-trip JSON payloads
   between two ZAP nodes over loopback.

## §3 What we do NOT prove

- **FIPS 204 EUF-CMA**: trusted via FIPS 204 + CIRCL.
- **External source honesty**: a malicious Pyth/Chainlink reporting
  wrong values is not detectable by oracled. The oracle's only job is
  to faithfully report what it observed. Adversarial-data resistance
  is the job of aggregation policy (median / TWAP / bounded
  deviation) at the canonical-output layer.
- **Aggregation correctness**: median / TWAP implementations live
  outside the strict-PQ surface; the proof claim here is on the
  *signing*, not on the *aggregation*.
- **AGPL-3.0 Solidity contracts under `contracts/prediction/`**: their
  audit trail is in `lib/openzeppelin-contracts/audits/` and is
  inherited from UMA.

## §4 Future tightening

- Lean / EasyCrypt refinement of the `Permit → Verify →
  mldsa65.Verify` chain.
- A property-based test asserting exhaustivity over the
  `profile.Scheme` enum.
- Formal model of the aggregation epoch boundary for bounded-
  deviation guarantees.
