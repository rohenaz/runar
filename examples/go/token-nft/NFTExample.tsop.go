package contract

import "tsop"

type SimpleNFT struct {
	tsop.StatefulSmartContract
	Owner    tsop.PubKey     // stateful
	TokenId  tsop.ByteString `tsop:"readonly"` // immutable: unique token identifier
	Metadata tsop.ByteString `tsop:"readonly"` // immutable: token metadata URI/hash
}

func (c *SimpleNFT) Transfer(sig tsop.Sig, newOwner tsop.PubKey, outputSatoshis tsop.Bigint) {
	tsop.Assert(tsop.CheckSig(sig, c.Owner))
	// AddOutput(satoshis, owner) -- single mutable prop
	c.AddOutput(outputSatoshis, newOwner)
}

func (c *SimpleNFT) Burn(sig tsop.Sig) {
	// Only owner can burn
	tsop.Assert(tsop.CheckSig(sig, c.Owner))
	// No AddOutput and no state mutation = token destroyed
}
