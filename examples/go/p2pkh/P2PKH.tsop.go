package contract

import "tsop"

type P2PKH struct {
	tsop.SmartContract
	PubKeyHash tsop.Addr `tsop:"readonly"`
}

func (c *P2PKH) Unlock(sig tsop.Sig, pubKey tsop.PubKey) {
	tsop.Assert(tsop.Hash160(pubKey) == c.PubKeyHash)
	tsop.Assert(tsop.CheckSig(sig, pubKey))
}
