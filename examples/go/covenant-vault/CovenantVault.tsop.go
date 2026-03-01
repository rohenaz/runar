package contract

import "tsop"

type CovenantVault struct {
	tsop.SmartContract
	Owner     tsop.PubKey `tsop:"readonly"`
	Recipient tsop.Addr   `tsop:"readonly"`
	MinAmount tsop.Bigint `tsop:"readonly"`
}

func (c *CovenantVault) Spend(sig tsop.Sig, amount tsop.Bigint, txPreimage tsop.SigHashPreimage) {
	// Owner must authorize
	tsop.Assert(tsop.CheckSig(sig, c.Owner))
	tsop.Assert(tsop.CheckPreimage(txPreimage))

	// Enforce minimum output amount (covenant rule)
	tsop.Assert(amount >= c.MinAmount)
}
