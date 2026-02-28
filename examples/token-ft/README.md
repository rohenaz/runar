# Fungible Token

A fungible token contract with splitting, merging, and simple transfer operations.

## What it does

Each UTXO tracks an `owner` (who can spend it) and a `balance` (how many tokens it holds). The token supports:

- **Transfer (split)** -- send a portion of tokens to a recipient, keeping the change. Creates two output UTXOs: one for the recipient and one for the sender's remaining balance.
- **Send** -- transfer the full balance to a new owner in a single output.
- **Merge** -- combine multiple UTXOs into one. Each input independently verifies the merged output; since all inputs check the same `hashOutputs`, they must agree on the result.

## Design pattern

**Stateful fungible token with multi-output verification** -- extends `StatefulSmartContract` and uses `this.addOutput()` to register transaction outputs with custom state values. The compiler collects all `addOutput` calls and auto-verifies them against the transaction's `hashOutputs` at method exit.

## TSOP features demonstrated

- `StatefulSmartContract` for automatic preimage verification
- `this.addOutput()` for multi-output state continuation (split/merge)
- Mutable properties (`owner`, `balance`) for per-UTXO state
- `readonly` property (`tokenId`) for immutable token identity
- Conservation-of-value verification via hash comparison

## Compile and use

```bash
tsop compile FungibleTokenExample.tsop.ts
```

Deploy with an initial owner, balance, and token ID. To transfer, the owner signs and specifies the recipient, amount, and satoshi value for each output.
