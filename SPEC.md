# SPEC — Lux Oracle (O-Chain + oracled)

> Standalone protocol specification for the Lux oracle. Covers the
> O-Chain VM (`oraclevm`) state machine, the oracled operator daemon's
> lifecycle, and the wire format of intra-Lux Observations / Records.
> External-source fetch formats are NOT in scope (they are dictated by
> the source).

## §1 Scope

This document specifies:

1. The O-Chain (`oraclevm`) state machine: Feeds, Observations,
   AggregatedValues, OracleRequests, OracleRecords, OracleCommits.
2. The oracled operator daemon lifecycle.
3. The signing-profile gate (`pkg/profile`) and the strict-PQ default.
4. The intra-Lux ZAP transport (`pkg/zaptransport`) used for
   pre-commit operator gossip.

Not in scope:

- The single-party FIPS 204 ML-DSA algorithm (see `luxfi/crypto/mldsa`).
- External-source data semantics (each source's spec applies).
- The UMA-derived Optimistic Oracle Solidity contracts under
  `contracts/prediction/` — those have their own AGPL-3.0 spec.

## §2 Terminology

- **Feed**: a registered oracle data feed, with operators, sources,
  policy hash, and update frequency.
- **Observation**: a single operator's measurement of a feed at a
  timestamp. Signed under the operator's registered key.
- **AggregatedValue**: the canonical output for a feed at an epoch.
  Computed by aggregation (median / TWAP / weighted median).
- **OracleRequest**: a deterministic external write/read request from
  PlatformVM. `request_id = H("LUX:OracleRequest:v1" || service_id ||
  session_id || BE32(step) || BE32(retry) || tx_id)`.
- **OracleRecord**: an executor's signed record of an external
  write/read result. Many records per request (executor committee).
- **OracleCommit**: a Merkle root over a request's records, committed
  at block boundaries.

## §3 State machine

```
                          [active]
   RegisterFeed ─────────► Feed
                            │
                            │ SubmitObservation(obs)
                            ▼
                  Observation{pending}
                            │
                            │ aggregation epoch boundary
                            ▼
                  AggregatedValue{epoch=N}
                            │
                            │ GetAttestation(feedID, epoch)
                            ▼
                  OracleAttestation (artifacts)


   (external write/read)
   RegisterRequest ──────► OracleRequest{Pending}
                            │
                            │ SubmitRecord(rec)
                            ▼
                  OracleRequest{Executing}
                            │
                            │ deadline / quorum
                            ▼
                  OracleRequest{Committed}
                            │
                            │ CommitRecords()
                            ▼
                  OracleCommit{root, recordCount}
```

Conflict key on the DAG layer: `(feedID, round)` where round =
`obs.Timestamp.UnixMilli()` for observations and `agg.Epoch` for
aggregations (vm/dag_vertex.go).

## §4 Wire format

### §4.1 Observation

```go
Observation {
    FeedID       ids.ID
    Value        []byte
    Timestamp    time.Time
    SourceMeta   [32]byte
    OperatorID   ids.NodeID
    Scheme       profile.Scheme   // 0x01 = ml-dsa-65, 0x02 = ed25519
    Signature    []byte           // ml-dsa-65: 3309 bytes; ed25519: 64 bytes
}
```

Signed message:
```
m = SHA-256( "LUX:OracleObservation:v1"
             || FeedID || Value || BE64(Timestamp.UnixNano())
             || SourceMeta || OperatorID )
ctx = "luxfi.oracle.v1"              // FIPS 204 §5.2 ctx for ml-dsa-65
                                     // prepended to m for ed25519
```

### §4.2 OracleRecord

```go
OracleRecord {
    RequestID    [32]byte
    Executor     ids.NodeID
    Timestamp    uint64
    Endpoint     string
    BodyHash     [32]byte
    ResultCode   uint32
    ExternalRef  []byte
    Scheme       profile.Scheme
    Signature    []byte
}
```

Signed message:
```
m = SHA-256( "LUX:OracleRecord:v1"
             || RequestID || Executor || BE64(Timestamp)
             || Endpoint || BodyHash || BE32(ResultCode)
             || ExternalRef )
```

### §4.3 ZAP plane

ZAP message types reserved by the oracle plane:

| Type | Value | Body |
|---|---|---|
| `MsgOracleHello` | `0x58` | handshake (TBD) |
| `MsgOracleObservation` | `0x59` | JSON `Observation` bytes at field 0 |
| `MsgOracleRecord` | `0x5A` | JSON `OracleRecord` bytes at field 0 |

Service tag: `_luxd-oracle._tcp`.

## §5 Signing-profile gate

The oracle has TWO signing surfaces:

1. **Intra-Lux operator surface** (signed Observations, OracleRecords).
   - Default scheme: `profile.SchemeMLDSA65`.
   - Domain-separation context: `profile.ContextTag = "luxfi.oracle.v1"`.
   - Classical Ed25519 opt-in only via `Config.LegacyClassicalEnabled`.

2. **External-source surface** (Pyth, Chainlink, market-data APIs,
   chain RPC).
   - Whatever the source demands. HTTP / TLS / native chain primitive.
   - Not subject to the PQ default.

The single function `profile.Verify` is the gate.
`VM.VerifyObservationSignature` and `VM.VerifyRecordSignature` each
call it exactly once; no other VM code touches `ed25519.Verify` or any
primitive directly.

## §6 Lifecycle (oracled)

```
boot → operatorID required
     → http listener :7800
     → optional zap listener :7810 (ORACLED_ZAP_PORT)
     → operator loop: fetch external data, sign Observations, submit to O-Chain
shut → zap.Stop()
```

## §7 Security goals

- **EUF-CMA Observations / Records**: an adversary without an operator's
  secret key cannot forge a signed Observation / Record that verifies
  under the operator's registered public key (ML-DSA-65 under FIPS 204).
- **No-classical-under-strict-PQ**: under default policy, classical
  schemes are refused by `profile.Verify` BEFORE any classical math.
- **Domain separation**: `ContextTag = "luxfi.oracle.v1"` is bound into
  every oracle signature; an oracle Observation cannot be replayed as a
  relay receipt (which uses `luxfi.relay.v1`).
- **Conflict freedom**: the DAG conflict key `(feedID, round)`
  guarantees at most one Observation per `(feedID, round)` ever lands
  in a block.

Out of scope: the external source's honesty (a price feed reporting
wrong data is not a cryptographic failure of the oracle).
