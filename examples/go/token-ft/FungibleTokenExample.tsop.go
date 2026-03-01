package contract

import "tsop"

type FungibleToken struct {
	tsop.StatefulSmartContract
	Owner   tsop.PubKey     // stateful: current token owner
	Balance tsop.Bigint     // stateful: token balance in this UTXO
	TokenId tsop.ByteString `tsop:"readonly"` // immutable: token identifier
}

// Split: 1 input -> 2 outputs (recipient + change)
func (c *FungibleToken) Transfer(sig tsop.Sig, to tsop.PubKey, amount tsop.Bigint, outputSatoshis tsop.Bigint) {
	tsop.Assert(tsop.CheckSig(sig, c.Owner))
	tsop.Assert(amount > 0)
	tsop.Assert(amount <= c.Balance)

	// AddOutput(satoshis, owner, balance) -- args match mutable props in order
	c.AddOutput(outputSatoshis, to, amount)
	c.AddOutput(outputSatoshis, c.Owner, c.Balance-amount)
}

// Simple send: 1 input -> 1 output, full balance
func (c *FungibleToken) Send(sig tsop.Sig, to tsop.PubKey, outputSatoshis tsop.Bigint) {
	tsop.Assert(tsop.CheckSig(sig, c.Owner))

	c.AddOutput(outputSatoshis, to, c.Balance)
}

// Merge: N inputs -> 1 output (each input calls this independently)
func (c *FungibleToken) Merge(sig tsop.Sig, totalBalance tsop.Bigint, outputSatoshis tsop.Bigint) {
	tsop.Assert(tsop.CheckSig(sig, c.Owner))
	tsop.Assert(totalBalance >= c.Balance)

	c.AddOutput(outputSatoshis, c.Owner, totalBalance)
}
