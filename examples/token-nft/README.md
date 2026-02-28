# Simple NFT (Non-Fungible Token)

A non-fungible token contract with transfer and burn capabilities.

## What it does

Represents a unique, non-fungible token identified by a `tokenId` and associated `metadata`. The token has a single owner who can:

- **Transfer** -- transfer ownership to a new public key using `this.addOutput()` to specify the output.
- **Burn** -- permanently destroy the token. No `addOutput` and no state mutation means the compiler only injects the preimage check — the UTXO is consumed without a successor.

## Design pattern

**Stateful NFT with burn path** -- extends `StatefulSmartContract`. The `owner` is the only mutable property. Transfer uses `addOutput` to register the output with the new owner. Burn uses neither `addOutput` nor state mutation, so the token ceases to exist.

## TSOP features demonstrated

- `StatefulSmartContract` for automatic preimage verification
- `this.addOutput()` for explicit output registration
- `ByteString` type for arbitrary binary data (token ID, metadata)
- Immutable identity fields (`readonly tokenId`, `readonly metadata`)
- Burn pattern: a method with no output registration

## Compile and use

```bash
tsop compile NFTExample.tsop.ts
```

Deploy with an initial owner, a unique token ID, and a metadata hash. Transfer uses `addOutput` to specify the new owner and satoshi amount. To burn, the owner signs without creating a successor output.
