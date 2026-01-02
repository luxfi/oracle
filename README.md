# Lux Oracle

**License: AGPL-3.0-only**

This repository contains the UMA-derived Optimistic Oracle and registry contracts for the Lux prediction market ecosystem.

## Why AGPL?

These contracts are derived from UMA Protocol's Optimistic Oracle which is licensed under AGPL-3.0. To comply with the copyleft requirements, this code is maintained in a separate repository and imported as a library by other Lux projects.

## Contracts

### Prediction Oracle (`contracts/prediction/`)
- `Oracle.sol` - Optimistic Oracle with assert-dispute-settle pattern
- `IOracle.sol` - Oracle interface with callbacks

### Registry (`contracts/registry/`)
- `Finder.sol` - Service locator for ecosystem contracts
- `Store.sol` - Fee management and final fees
- `IdentifierWhitelist.sol` - Supported query identifiers

## Usage

Add as a git submodule or forge dependency:

```bash
forge install luxfi/oracle
```

Then import:

```solidity
import "@luxfi/oracle/prediction/Oracle.sol";
import "@luxfi/oracle/registry/Finder.sol";
```

## Related

- **Lux Standard** (`@luxfi/contracts`) - MIT-licensed contracts that use this lib
- **UMA Protocol** - Original source of the Optimistic Oracle design
