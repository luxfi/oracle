# CHANGELOG — luxfi/oracle

## Unreleased — Tier-A flip

### Added
- `pkg/profile`: signing-profile gate. ML-DSA-65 (FIPS 204) default;
  Ed25519 opt-in only via `Policy.LegacyClassicalEnabled`. Single
  `Verify` entry point.
- `pkg/zaptransport`: intra-Lux operator transport over
  `github.com/luxfi/zap`. Carries opaque JSON Observation / Record
  bytes; never invokes signature math.
- `VM.RegisterOperatorKey(nodeID, scheme, pub)` — operator key
  registry, scheme-tagged.
- `VM.VerifyObservationSignature(obs)` — single ML-DSA-65-default
  verifier for Observations.
- `VM.VerifyRecordSignature(rec)` — single ML-DSA-65-default
  verifier for OracleRecords.
- Tier-A documentation set: `OVERVIEW.md`, `SPEC.md`,
  `PROOF-CLAIMS.md`, `TRUSTED-COMPUTING-BASE.md`,
  `CRYPTOGRAPHER-SIGN-OFF.md`, `DEPLOYMENT-RUNBOOK.md`,
  `LEGACY-CLASSICAL.md`.
- `vm.Config.LegacyClassicalEnabled` — chain-level toggle.
- `oracled` flags: `--zap-port`, env `ORACLED_ZAP_PORT`.

### Changed
- `vm.Observation` carries `Scheme` (`profile.Scheme`).
- `vm.OracleRecord` carries `Scheme`.
- Observation signature now binds the domain-separation context
  `"luxfi.oracle.v1"` (FIPS 204 §5.2 ctx for ML-DSA-65; prepended
  bytes for Ed25519).

### Tests
- `pkg/profile/profile_test.go` (5 tests).
- `vm/observation_pq_test.go` (5 end-to-end PQ tests covering both
  Observation and OracleRecord verification paths).
- `vm/observation_fuzz_test.go` (2 fuzz targets, ~50k execs no panics).
- `pkg/zaptransport/zaptransport_test.go` (3 e2e tests).
