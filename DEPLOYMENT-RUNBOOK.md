# Deployment Runbook — luxfi/oracle

> Operational guidance for running `oracled` against an O-Chain
> (`oraclevm`) deployment.

## Quick start

```sh
ORACLED_OPERATOR_ID=NodeID-... \
ORACLED_ORACLE_RPC=http://127.0.0.1:9650/ext/bc/O/rpc \
ORACLED_LISTEN=:7800 \
ORACLED_ZAP_PORT=7810 \
oracled
```

## Configuration

| Env / flag | Default | Meaning |
|---|---|---|
| `ORACLED_LISTEN` / `--listen` | `:7800` | HTTP listen address. |
| `ORACLED_ZAP_PORT` / `--zap-port` | `7810` | Intra-Lux ZAP plane port. `0` disables. |
| `ORACLED_ORACLE_RPC` / `--oracle-rpc` | `http://127.0.0.1:9650/ext/bc/O/rpc` | O-Chain JSON-RPC URL. |
| `ORACLED_OPERATOR_ID` / `--operator-id` | (required) | This operator's NodeID. |

## Signing-profile decision

Default: **strict-PQ**. Only ML-DSA-65 Observations / OracleRecords are
accepted by the O-Chain VM.

### Strict-PQ (recommended)

O-Chain genesis / config:
```json
{
  "legacyClassicalEnabled": false
}
```

All operators MUST register ML-DSA-65 keys via the appropriate
administrative path. Classical observations are refused by
`profile.Verify` before any classical signature math runs.

### Legacy classical (opt-in, migration)

```json
{
  "legacyClassicalEnabled": true
}
```

Admits Ed25519 alongside ML-DSA-65. See `LEGACY-CLASSICAL.md`.

## External-source interactions

These are NOT subject to the PQ-default rule:

- **Pyth / Chainlink price feeds**: each network's native signature
  scheme. oracled validates per the source's published spec.
- **Bitcoin / Ethereum RPC**: secp256k1 ECDSA / Schnorr per the chain.
- **Market-data APIs**: TLS-protected HTTPS, no chain-side
  cryptographic check; integrity comes from cross-source aggregation
  and bounded-deviation rejection.

## Operational checks

```sh
# Liveness probe (via O-Chain RPC)
curl -fsS http://127.0.0.1:9650/ext/bc/O/rpc \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"oracle.Health","params":{}}' | jq

# Feed list / latest value
curl -fsS http://127.0.0.1:9650/ext/bc/O/rpc \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"oracle.GetValue","params":{"feedId":"<id>"}}' | jq
```

## Disaster-recovery

Authoritative state lives on O-Chain. oracled is stateless beyond the
external-source fetch loop's in-memory caches.

Operator key rotation: register a new ML-DSA-65 key on O-Chain, wait
for confirmation, then start signing Observations with the new key.
The old key's Observations stop verifying immediately under strict-PQ
once the new key is the registered one for that NodeID.
