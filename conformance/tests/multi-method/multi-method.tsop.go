package contract

import "tsop"

type MultiMethod struct {
	tsop.SmartContract
	Owner  tsop.PubKey `tsop:"readonly"`
	Backup tsop.PubKey `tsop:"readonly"`
}

func (c *MultiMethod) computeThreshold(a tsop.Int, b tsop.Int) tsop.Int {
	return a*b + 1
}

func (c *MultiMethod) SpendWithOwner(sig tsop.Sig, amount tsop.Int) {
	threshold := c.computeThreshold(amount, 2)
	tsop.Assert(threshold > 10)
	tsop.Assert(tsop.CheckSig(sig, c.Owner))
}

func (c *MultiMethod) SpendWithBackup(sig tsop.Sig) {
	tsop.Assert(tsop.CheckSig(sig, c.Backup))
}
